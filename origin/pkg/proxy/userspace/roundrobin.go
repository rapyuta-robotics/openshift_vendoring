package userspace

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/proxy"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/slice"
)

var (
	ErrMissingServiceEntry = errors.New("missing service entry")
	ErrMissingEndpoints    = errors.New("missing endpoints")
)

type affinityState struct {
	clientIP string
	//clientProtocol  api.Protocol //not yet used
	//sessionCookie   string       //not yet used
	endpoint string
	lastUsed time.Time
}

type affinityPolicy struct {
	affinityType api.ServiceAffinity
	affinityMap  map[string]*affinityState // map client IP -> affinity info
	ttlMinutes   int
}

// LoadBalancerRR is a round-robin load balancer.
type LoadBalancerRR struct {
	lock     sync.RWMutex
	services map[proxy.ServicePortName]*balancerState
}

// Ensure this implements LoadBalancer.
var _ LoadBalancer = &LoadBalancerRR{}

type balancerState struct {
	endpoints []string // a list of "ip:port" style strings
	index     int      // current index into endpoints
	affinity  affinityPolicy
}

func newAffinityPolicy(affinityType api.ServiceAffinity, ttlMinutes int) *affinityPolicy {
	return &affinityPolicy{
		affinityType: affinityType,
		affinityMap:  make(map[string]*affinityState),
		ttlMinutes:   ttlMinutes,
	}
}

// NewLoadBalancerRR returns a new LoadBalancerRR.
func NewLoadBalancerRR() *LoadBalancerRR {
	return &LoadBalancerRR{
		services: map[proxy.ServicePortName]*balancerState{},
	}
}

func (lb *LoadBalancerRR) NewService(svcPort proxy.ServicePortName, affinityType api.ServiceAffinity, ttlMinutes int) error {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.newServiceInternal(svcPort, affinityType, ttlMinutes)
	return nil
}

// This assumes that lb.lock is already held.
func (lb *LoadBalancerRR) newServiceInternal(svcPort proxy.ServicePortName, affinityType api.ServiceAffinity, ttlMinutes int) *balancerState {
	if ttlMinutes == 0 {
		ttlMinutes = 180 //default to 3 hours if not specified.  Should 0 be unlimited instead????
	}

	if _, exists := lb.services[svcPort]; !exists {
		lb.services[svcPort] = &balancerState{affinity: *newAffinityPolicy(affinityType, ttlMinutes)}
		glog.V(4).Infof("LoadBalancerRR service %q did not exist, created", svcPort)
	} else if affinityType != "" {
		lb.services[svcPort].affinity.affinityType = affinityType
	}
	return lb.services[svcPort]
}

// return true if this service is using some form of session affinity.
func isSessionAffinity(affinity *affinityPolicy) bool {
	// Should never be empty string, but checking for it to be safe.
	if affinity.affinityType == "" || affinity.affinityType == api.ServiceAffinityNone {
		return false
	}
	return true
}

// ServiceHasEndpoints checks whether a service entry has endpoints.
func (lb *LoadBalancerRR) ServiceHasEndpoints(svcPort proxy.ServicePortName) bool {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	state, exists := lb.services[svcPort]
	return exists && state != nil && len(state.endpoints) > 0
}

// NextEndpoint returns a service endpoint.
// The service endpoint is chosen using the round-robin algorithm.
func (lb *LoadBalancerRR) NextEndpoint(svcPort proxy.ServicePortName, srcAddr net.Addr, sessionAffinityReset bool) (string, error) {
	// Coarse locking is simple.  We can get more fine-grained if/when we
	// can prove it matters.
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists || state == nil {
		return "", ErrMissingServiceEntry
	}
	if len(state.endpoints) == 0 {
		return "", ErrMissingEndpoints
	}
	glog.V(4).Infof("NextEndpoint for service %q, srcAddr=%v: endpoints: %+v", svcPort, srcAddr, state.endpoints)

	sessionAffinityEnabled := isSessionAffinity(&state.affinity)

	var ipaddr string
	if sessionAffinityEnabled {
		// Caution: don't shadow ipaddr
		var err error
		ipaddr, _, err = net.SplitHostPort(srcAddr.String())
		if err != nil {
			return "", fmt.Errorf("malformed source address %q: %v", srcAddr.String(), err)
		}
		if !sessionAffinityReset {
			sessionAffinity, hasSessionAffinity := state.affinity.affinityMap[ipaddr]
			if hasSessionAffinity && int(time.Now().Sub(sessionAffinity.lastUsed).Minutes()) < state.affinity.ttlMinutes {
				// Affinity wins.
				endpoint := sessionAffinity.endpoint
				sessionAffinity.lastUsed = time.Now()
				glog.V(4).Infof("NextEndpoint for service %q from IP %s with sessionAffinity %+v: %s", svcPort, ipaddr, sessionAffinity, endpoint)
				return endpoint, nil
			}
		}
	}
	// Take the next endpoint.
	endpoint := state.endpoints[state.index]
	state.index = (state.index + 1) % len(state.endpoints)

	if sessionAffinityEnabled {
		var affinity *affinityState
		affinity = state.affinity.affinityMap[ipaddr]
		if affinity == nil {
			affinity = new(affinityState) //&affinityState{ipaddr, "TCP", "", endpoint, time.Now()}
			state.affinity.affinityMap[ipaddr] = affinity
		}
		affinity.lastUsed = time.Now()
		affinity.endpoint = endpoint
		affinity.clientIP = ipaddr
		glog.V(4).Infof("Updated affinity key %s: %+v", ipaddr, state.affinity.affinityMap[ipaddr])
	}

	return endpoint, nil
}

type hostPortPair struct {
	host string
	port int
}

func isValidEndpoint(hpp *hostPortPair) bool {
	return hpp.host != "" && hpp.port > 0
}

func flattenValidEndpoints(endpoints []hostPortPair) []string {
	// Convert Endpoint objects into strings for easier use later.  Ignore
	// the protocol field - we'll get that from the Service objects.
	var result []string
	for i := range endpoints {
		hpp := &endpoints[i]
		if isValidEndpoint(hpp) {
			result = append(result, net.JoinHostPort(hpp.host, strconv.Itoa(hpp.port)))
		}
	}
	return result
}

// Remove any session affinity records associated to a particular endpoint (for example when a pod goes down).
func removeSessionAffinityByEndpoint(state *balancerState, svcPort proxy.ServicePortName, endpoint string) {
	for _, affinity := range state.affinity.affinityMap {
		if affinity.endpoint == endpoint {
			glog.V(4).Infof("Removing client: %s from affinityMap for service %q", affinity.endpoint, svcPort)
			delete(state.affinity.affinityMap, affinity.clientIP)
		}
	}
}

// Loop through the valid endpoints and then the endpoints associated with the Load Balancer.
// Then remove any session affinity records that are not in both lists.
// This assumes the lb.lock is held.
func (lb *LoadBalancerRR) updateAffinityMap(svcPort proxy.ServicePortName, newEndpoints []string) {
	allEndpoints := map[string]int{}
	for _, newEndpoint := range newEndpoints {
		allEndpoints[newEndpoint] = 1
	}
	state, exists := lb.services[svcPort]
	if !exists {
		return
	}
	for _, existingEndpoint := range state.endpoints {
		allEndpoints[existingEndpoint] = allEndpoints[existingEndpoint] + 1
	}
	for mKey, mVal := range allEndpoints {
		if mVal == 1 {
			glog.V(2).Infof("Delete endpoint %s for service %q", mKey, svcPort)
			removeSessionAffinityByEndpoint(state, svcPort, mKey)
		}
	}
}

// OnEndpointsUpdate manages the registered service endpoints.
// Registered endpoints are updated if found in the update set or
// unregistered if missing from the update set.
func (lb *LoadBalancerRR) OnEndpointsUpdate(allEndpoints []api.Endpoints) {
	registeredEndpoints := make(map[proxy.ServicePortName]bool)
	lb.lock.Lock()
	defer lb.lock.Unlock()

	// Update endpoints for services.
	for i := range allEndpoints {
		svcEndpoints := &allEndpoints[i]

		// We need to build a map of portname -> all ip:ports for that
		// portname.  Explode Endpoints.Subsets[*] into this structure.
		portsToEndpoints := map[string][]hostPortPair{}
		for i := range svcEndpoints.Subsets {
			ss := &svcEndpoints.Subsets[i]
			for i := range ss.Ports {
				port := &ss.Ports[i]
				for i := range ss.Addresses {
					addr := &ss.Addresses[i]
					portsToEndpoints[port.Name] = append(portsToEndpoints[port.Name], hostPortPair{addr.IP, int(port.Port)})
					// Ignore the protocol field - we'll get that from the Service objects.
				}
			}
		}

		for portname := range portsToEndpoints {
			svcPort := proxy.ServicePortName{NamespacedName: types.NamespacedName{Namespace: svcEndpoints.Namespace, Name: svcEndpoints.Name}, Port: portname}
			state, exists := lb.services[svcPort]
			curEndpoints := []string{}
			if state != nil {
				curEndpoints = state.endpoints
			}
			newEndpoints := flattenValidEndpoints(portsToEndpoints[portname])

			if !exists || state == nil || len(curEndpoints) != len(newEndpoints) || !slicesEquiv(slice.CopyStrings(curEndpoints), newEndpoints) {
				glog.V(1).Infof("LoadBalancerRR: Setting endpoints for %s to %+v", svcPort, newEndpoints)
				lb.updateAffinityMap(svcPort, newEndpoints)
				// OnEndpointsUpdate can be called without NewService being called externally.
				// To be safe we will call it here.  A new service will only be created
				// if one does not already exist.  The affinity will be updated
				// later, once NewService is called.
				state = lb.newServiceInternal(svcPort, api.ServiceAffinity(""), 0)
				state.endpoints = slice.ShuffleStrings(newEndpoints)

				// Reset the round-robin index.
				state.index = 0
			}
			registeredEndpoints[svcPort] = true
		}
	}
	// Remove endpoints missing from the update.
	for k := range lb.services {
		if _, exists := registeredEndpoints[k]; !exists {
			glog.V(2).Infof("LoadBalancerRR: Removing endpoints for %s", k)
			delete(lb.services, k)
		}
	}
}

// Tests whether two slices are equivalent.  This sorts both slices in-place.
func slicesEquiv(lhs, rhs []string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	if reflect.DeepEqual(slice.SortStrings(lhs), slice.SortStrings(rhs)) {
		return true
	}
	return false
}

func (lb *LoadBalancerRR) CleanupStaleStickySessions(svcPort proxy.ServicePortName) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	state, exists := lb.services[svcPort]
	if !exists {
		return
	}
	for ip, affinity := range state.affinity.affinityMap {
		if int(time.Now().Sub(affinity.lastUsed).Minutes()) >= state.affinity.ttlMinutes {
			glog.V(4).Infof("Removing client %s from affinityMap for service %q", affinity.clientIP, svcPort)
			delete(state.affinity.affinityMap, ip)
		}
	}
}
