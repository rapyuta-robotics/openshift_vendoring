package userspace

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/proxy"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/exec"
	ipttest "github.com/openshift/kubernetes/pkg/util/iptables/testing"
	"github.com/openshift/kubernetes/pkg/util/runtime"
)

const (
	udpIdleTimeoutForTest = 250 * time.Millisecond
)

func joinHostPort(host string, port int) string {
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}

func waitForClosedPortTCP(p *Proxier, proxyPort int) error {
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("tcp", joinHostPort("", proxyPort))
		if err != nil {
			return nil
		}
		conn.Close()
		time.Sleep(1 * time.Millisecond)
	}
	return fmt.Errorf("port %d still open", proxyPort)
}

func waitForClosedPortUDP(p *Proxier, proxyPort int) error {
	for i := 0; i < 50; i++ {
		conn, err := net.Dial("udp", joinHostPort("", proxyPort))
		if err != nil {
			return nil
		}
		conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		// To detect a closed UDP port write, then read.
		_, err = conn.Write([]byte("x"))
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Timeout() {
				return nil
			}
		}
		var buf [4]byte
		_, err = conn.Read(buf[0:])
		if err != nil {
			if e, ok := err.(net.Error); ok && !e.Timeout() {
				return nil
			}
		}
		conn.Close()
		time.Sleep(1 * time.Millisecond)
	}
	return fmt.Errorf("port %d still open", proxyPort)
}

var tcpServerPort int32
var udpServerPort int32

func TestMain(m *testing.M) {
	// Don't handle panics
	runtime.ReallyCrash = true

	// TCP setup.
	tcp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.URL.Path[1:]))
	}))
	defer tcp.Close()

	u, err := url.Parse(tcp.URL)
	if err != nil {
		panic(fmt.Sprintf("failed to parse: %v", err))
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		panic(fmt.Sprintf("failed to parse: %v", err))
	}
	tcpServerPortValue, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to atoi(%s): %v", port, err))
	}
	tcpServerPort = int32(tcpServerPortValue)

	// UDP setup.
	udp, err := newUDPEchoServer()
	if err != nil {
		panic(fmt.Sprintf("failed to make a UDP server: %v", err))
	}
	_, port, err = net.SplitHostPort(udp.LocalAddr().String())
	if err != nil {
		panic(fmt.Sprintf("failed to parse: %v", err))
	}
	udpServerPortValue, err := strconv.Atoi(port)
	if err != nil {
		panic(fmt.Sprintf("failed to atoi(%s): %v", port, err))
	}
	udpServerPort = int32(udpServerPortValue)
	go udp.Loop()

	ret := m.Run()
	// it should be safe to call Close() multiple times.
	tcp.Close()
	os.Exit(ret)
}

func testEchoTCP(t *testing.T, address string, port int) {
	path := "aaaaa"
	res, err := http.Get("http://" + address + ":" + fmt.Sprintf("%d", port) + "/" + path)
	if err != nil {
		t.Fatalf("error connecting to server: %v", err)
	}
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("error reading data: %v %v", err, string(data))
	}
	if string(data) != path {
		t.Errorf("expected: %s, got %s", path, string(data))
	}
}

func testEchoUDP(t *testing.T, address string, port int) {
	data := "abc123"

	conn, err := net.Dial("udp", joinHostPort(address, port))
	if err != nil {
		t.Fatalf("error connecting to server: %v", err)
	}
	if _, err = conn.Write([]byte(data)); err != nil {
		t.Fatalf("error sending to server: %v", err)
	}
	var resp [1024]byte
	n, err := conn.Read(resp[0:])
	if err != nil {
		t.Errorf("error receiving data: %v", err)
	}
	if string(resp[0:n]) != data {
		t.Errorf("expected: %s, got %s", data, string(resp[0:n]))
	}
}

func waitForNumProxyLoops(t *testing.T, p *Proxier, want int32) {
	var got int32
	for i := 0; i < 600; i++ {
		got = atomic.LoadInt32(&p.numProxyLoops)
		if got == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("expected %d ProxyLoops running, got %d", want, got)
}

func waitForNumProxyClients(t *testing.T, s *ServiceInfo, want int, timeout time.Duration) {
	var got int
	now := time.Now()
	deadline := now.Add(timeout)
	for time.Now().Before(deadline) {
		s.ActiveClients.Lock()
		got = len(s.ActiveClients.Clients)
		s.ActiveClients.Unlock()
		if got == want {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Errorf("expected %d ProxyClients live, got %d", want, got)
}

func TestTCPProxy(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func TestUDPProxy(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoUDP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func TestUDPProxyTimeout(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	waitForNumProxyLoops(t, p, 1)
	testEchoUDP(t, "127.0.0.1", svcInfo.proxyPort)
	// When connecting to a UDP service endpoint, there shoule be a Conn for proxy.
	waitForNumProxyClients(t, svcInfo, 1, time.Second)
	// If conn has no activity for serviceInfo.timeout since last Read/Write, it shoule be closed because of timeout.
	waitForNumProxyClients(t, svcInfo, 0, 2*time.Second)
}

func TestMultiPortProxy(t *testing.T) {
	lb := NewLoadBalancerRR()
	serviceP := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo-p"}, Port: "p"}
	serviceQ := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo-q"}, Port: "q"}
	lb.OnEndpointsUpdate([]api.Endpoints{{
		ObjectMeta: api.ObjectMeta{Name: serviceP.Name, Namespace: serviceP.Namespace},
		Subsets: []api.EndpointSubset{{
			Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
			Ports:     []api.EndpointPort{{Name: "p", Protocol: "TCP", Port: tcpServerPort}},
		}},
	}, {
		ObjectMeta: api.ObjectMeta{Name: serviceQ.Name, Namespace: serviceQ.Namespace},
		Subsets: []api.EndpointSubset{{
			Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
			Ports:     []api.EndpointPort{{Name: "q", Protocol: "UDP", Port: udpServerPort}},
		}},
	}})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRefP := api.ObjectReference{Name: serviceP.Name, Namespace: serviceP.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfoP, err := p.addServiceOnPort(serviceP, serviceRefP, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoTCP(t, "127.0.0.1", svcInfoP.proxyPort)
	waitForNumProxyLoops(t, p, 1)

	serviceRefQ := api.ObjectReference{Name: serviceQ.Name, Namespace: serviceQ.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfoQ, err := p.addServiceOnPort(serviceQ, serviceRefQ, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoUDP(t, "127.0.0.1", svcInfoQ.proxyPort)
	waitForNumProxyLoops(t, p, 2)
}

func TestMultiPortOnServiceUpdate(t *testing.T) {
	lb := NewLoadBalancerRR()
	serviceP := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	serviceQ := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "q"}
	serviceX := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "x"}

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: serviceP.Name, Namespace: serviceP.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     80,
			Protocol: "TCP",
		}, {
			Name:     "q",
			Port:     81,
			Protocol: "UDP",
		}}},
	}})
	waitForNumProxyLoops(t, p, 2)
	svcInfo, exists := p.getServiceInfo(serviceP)
	if !exists {
		t.Fatalf("can't find serviceInfo for %s", serviceP)
	}
	if svcInfo.portal.ip.String() != "1.2.3.4" || svcInfo.portal.port != 80 || svcInfo.protocol != "TCP" {
		t.Errorf("unexpected serviceInfo for %s: %#v", serviceP, svcInfo)
	}

	svcInfo, exists = p.getServiceInfo(serviceQ)
	if !exists {
		t.Fatalf("can't find serviceInfo for %s", serviceQ)
	}
	if svcInfo.portal.ip.String() != "1.2.3.4" || svcInfo.portal.port != 81 || svcInfo.protocol != "UDP" {
		t.Errorf("unexpected serviceInfo for %s: %#v", serviceQ, svcInfo)
	}

	svcInfo, exists = p.getServiceInfo(serviceX)
	if exists {
		t.Fatalf("found unwanted serviceInfo for %s: %#v", serviceX, svcInfo)
	}
}

// Helper: Stops the proxy for the named service.
func stopProxyByName(proxier *Proxier, service proxy.ServicePortName) error {
	info, found := proxier.getServiceInfo(service)
	if !found {
		return fmt.Errorf("unknown service: %s", service)
	}
	return proxier.stopProxy(service, info)
}

func TestTCPProxyStop(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Namespace: service.Namespace, Name: service.Name},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	if !svcInfo.IsAlive() {
		t.Fatalf("wrong value for IsAlive(): expected true")
	}
	conn, err := net.Dial("tcp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	stopProxyByName(p, service)
	if svcInfo.IsAlive() {
		t.Fatalf("wrong value for IsAlive(): expected false")
	}
	// Wait for the port to really close.
	if err = waitForClosedPortTCP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)
}

func TestUDPProxyStop(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Namespace: service.Namespace, Name: service.Name},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	conn, err := net.Dial("udp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	stopProxyByName(p, service)
	// Wait for the port to really close.
	if err = waitForClosedPortUDP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)
}

func TestTCPProxyUpdateDelete(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Namespace: service.Namespace, Name: service.Name},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	conn, err := net.Dial("tcp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{})
	if err = waitForClosedPortTCP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)
}

func TestUDPProxyUpdateDelete(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Namespace: service.Namespace, Name: service.Name},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	conn, err := net.Dial("udp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{})
	if err = waitForClosedPortUDP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)
}

func TestTCPProxyUpdateDeleteUpdate(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	conn, err := net.Dial("tcp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{})
	if err = waitForClosedPortTCP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     int32(svcInfo.proxyPort),
			Protocol: "TCP",
		}}},
	}})
	svcInfo, exists := p.getServiceInfo(service)
	if !exists {
		t.Fatalf("can't find serviceInfo for %s", service)
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func TestUDPProxyUpdateDeleteUpdate(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	conn, err := net.Dial("udp", joinHostPort("", svcInfo.proxyPort))
	if err != nil {
		t.Fatalf("error connecting to proxy: %v", err)
	}
	conn.Close()
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{})
	if err = waitForClosedPortUDP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	waitForNumProxyLoops(t, p, 0)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     int32(svcInfo.proxyPort),
			Protocol: "UDP",
		}}},
	}})
	svcInfo, exists := p.getServiceInfo(service)
	if !exists {
		t.Fatalf("can't find serviceInfo")
	}
	testEchoUDP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func TestTCPProxyUpdatePort(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     99,
			Protocol: "TCP",
		}}},
	}})
	// Wait for the socket to actually get free.
	if err = waitForClosedPortTCP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	svcInfo, exists := p.getServiceInfo(service)
	if !exists {
		t.Fatalf("can't find serviceInfo")
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	// This is a bit async, but this should be sufficient.
	time.Sleep(500 * time.Millisecond)
	waitForNumProxyLoops(t, p, 1)
}

func TestUDPProxyUpdatePort(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: udpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "UDP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     99,
			Protocol: "UDP",
		}}},
	}})
	// Wait for the socket to actually get free.
	if err = waitForClosedPortUDP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	svcInfo, exists := p.getServiceInfo(service)
	if !exists {
		t.Fatalf("can't find serviceInfo")
	}
	testEchoUDP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func TestProxyUpdatePublicIPs(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{
			Ports: []api.ServicePort{{
				Name:     "p",
				Port:     int32(svcInfo.portal.port),
				Protocol: "TCP",
			}},
			ClusterIP:   svcInfo.portal.ip.String(),
			ExternalIPs: []string{"4.3.2.1"},
		},
	}})
	// Wait for the socket to actually get free.
	if err = waitForClosedPortTCP(p, svcInfo.proxyPort); err != nil {
		t.Fatalf(err.Error())
	}
	svcInfo, exists := p.getServiceInfo(service)
	if !exists {
		t.Fatalf("can't find serviceInfo")
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	// This is a bit async, but this should be sufficient.
	time.Sleep(500 * time.Millisecond)
	waitForNumProxyLoops(t, p, 1)
}

func TestProxyUpdatePortal(t *testing.T) {
	lb := NewLoadBalancerRR()
	service := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: "testnamespace", Name: "echo"}, Port: "p"}
	lb.OnEndpointsUpdate([]api.Endpoints{
		{
			ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
			Subsets: []api.EndpointSubset{{
				Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
				Ports:     []api.EndpointPort{{Name: "p", Port: tcpServerPort}},
			}},
		},
	})

	fexec := makeFakeExec()

	p, err := createProxier(lb, net.ParseIP("0.0.0.0"), ipttest.NewFake(), fexec, net.ParseIP("127.0.0.1"), nil, time.Minute, udpIdleTimeoutForTest, NewProxySocket)
	if err != nil {
		t.Fatal(err)
	}
	waitForNumProxyLoops(t, p, 0)

	serviceRef := api.ObjectReference{Name: service.Name, Namespace: service.Namespace, Kind: "Service", APIVersion: "v1"}
	svcInfo, err := p.addServiceOnPort(service, serviceRef, "TCP", 0, time.Second)
	if err != nil {
		t.Fatalf("error adding new service: %#v", err)
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     int32(svcInfo.proxyPort),
			Protocol: "TCP",
		}}},
	}})
	_, exists := p.getServiceInfo(service)
	if exists {
		t.Fatalf("service with empty ClusterIP should not be included in the proxy")
	}

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "None", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     int32(svcInfo.proxyPort),
			Protocol: "TCP",
		}}},
	}})
	_, exists = p.getServiceInfo(service)
	if exists {
		t.Fatalf("service with 'None' as ClusterIP should not be included in the proxy")
	}

	p.OnServiceUpdate([]api.Service{{
		ObjectMeta: api.ObjectMeta{Name: service.Name, Namespace: service.Namespace},
		Spec: api.ServiceSpec{ClusterIP: "1.2.3.4", Ports: []api.ServicePort{{
			Name:     "p",
			Port:     int32(svcInfo.proxyPort),
			Protocol: "TCP",
		}}},
	}})
	svcInfo, exists = p.getServiceInfo(service)
	if !exists {
		t.Fatalf("service with ClusterIP set not found in the proxy")
	}
	testEchoTCP(t, "127.0.0.1", svcInfo.proxyPort)
	waitForNumProxyLoops(t, p, 1)
}

func makeFakeExec() *exec.FakeExec {
	fcmd := exec.FakeCmd{
		CombinedOutputScript: []exec.FakeCombinedOutputAction{
			func() ([]byte, error) { return []byte("1 flow entries have been deleted"), nil },
		},
	}
	return &exec.FakeExec{
		CommandScript: []exec.FakeCommandAction{
			func(cmd string, args ...string) exec.Cmd { return exec.InitFakeCmd(&fcmd, cmd, args...) },
		},
		LookPathFunc: func(cmd string) (string, error) { return cmd, nil },
	}
}

// TODO(justinsb): Add test for nodePort conflict detection, once we have nodePort wired in
