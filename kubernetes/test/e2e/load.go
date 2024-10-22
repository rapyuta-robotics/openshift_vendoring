/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	"github.com/openshift/kubernetes/pkg/client/transport"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/util/intstr"
	utilnet "github.com/openshift/kubernetes/pkg/util/net"
	"github.com/openshift/kubernetes/test/e2e/framework"
	testutils "github.com/openshift/kubernetes/test/utils"

	. "github.com/openshift/github.com/onsi/ginkgo"
	. "github.com/openshift/github.com/onsi/gomega"
)

const (
	smallRCSize       = 5
	mediumRCSize      = 30
	bigRCSize         = 250
	smallRCGroupName  = "load-small-rc"
	mediumRCGroupName = "load-medium-rc"
	bigRCGroupName    = "load-big-rc"
	smallRCBatchSize  = 30
	mediumRCBatchSize = 5
	bigRCBatchSize    = 1
	// We start RCs/Services/pods/... in different namespace in this test.
	// nodeCountPerNamespace determines how many namespaces we will be using
	// depending on the number of nodes in the underlying cluster.
	nodeCountPerNamespace = 100
)

// This test suite can take a long time to run, so by default it is added to
// the ginkgo.skip list (see driver.go).
// To run this suite you must explicitly ask for it by setting the
// -t/--test flag or ginkgo.focus flag.
var _ = framework.KubeDescribe("Load capacity", func() {
	var clientset internalclientset.Interface
	var nodeCount int
	var ns string
	var configs []*testutils.RCConfig

	// Gathers metrics before teardown
	// TODO add flag that allows to skip cleanup on failure
	AfterEach(func() {
		// Verify latency metrics
		highLatencyRequests, err := framework.HighLatencyRequests(clientset)
		framework.ExpectNoError(err, "Too many instances metrics above the threshold")
		Expect(highLatencyRequests).NotTo(BeNumerically(">", 0))
	})

	// We assume a default throughput of 10 pods/second throughput.
	// We may want to revisit it in the future.
	// However, this can be overriden by LOAD_TEST_THROUGHPUT env var.
	throughput := 10
	if throughputEnv := os.Getenv("LOAD_TEST_THROUGHPUT"); throughputEnv != "" {
		if newThroughput, err := strconv.Atoi(throughputEnv); err == nil {
			throughput = newThroughput
		}
	}

	// Explicitly put here, to delete namespace at the end of the test
	// (after measuring latency metrics, etc.).
	options := framework.FrameworkOptions{
		ClientQPS:   float32(math.Max(50.0, float64(2*throughput))),
		ClientBurst: int(math.Max(100.0, float64(4*throughput))),
	}
	f := framework.NewFramework("load", options, nil)
	f.NamespaceDeletionTimeout = time.Hour

	BeforeEach(func() {
		clientset = f.ClientSet

		ns = f.Namespace.Name
		nodes := framework.GetReadySchedulableNodesOrDie(clientset)
		nodeCount = len(nodes.Items)
		Expect(nodeCount).NotTo(BeZero())

		// Terminating a namespace (deleting the remaining objects from it - which
		// generally means events) can affect the current run. Thus we wait for all
		// terminating namespace to be finally deleted before starting this test.
		err := framework.CheckTestingNSDeletedExcept(clientset, ns)
		framework.ExpectNoError(err)

		framework.ExpectNoError(framework.ResetMetrics(clientset))
	})

	type Load struct {
		podsPerNode int
		image       string
		command     []string
	}

	loadTests := []Load{
		// The container will consume 1 cpu and 512mb of memory.
		{podsPerNode: 3, image: "jess/stress", command: []string{"stress", "-c", "1", "-m", "2"}},
		{podsPerNode: 30, image: "gcr.io/google_containers/serve_hostname:v1.4"},
	}

	for _, testArg := range loadTests {
		feature := "ManualPerformance"
		if testArg.podsPerNode == 30 {
			feature = "Performance"
		}
		name := fmt.Sprintf("[Feature:%s] should be able to handle %v pods per node", feature, testArg.podsPerNode)
		itArg := testArg

		It(name, func() {
			// Create a number of namespaces.
			namespaceCount := (nodeCount + nodeCountPerNamespace - 1) / nodeCountPerNamespace
			namespaces, err := CreateNamespaces(f, namespaceCount, fmt.Sprintf("load-%v-nodepods", itArg.podsPerNode))
			framework.ExpectNoError(err)

			totalPods := itArg.podsPerNode * nodeCount
			configs = generateRCConfigs(totalPods, itArg.image, itArg.command, namespaces)
			var services []*api.Service
			// Read the environment variable to see if we want to create services
			createServices := os.Getenv("CREATE_SERVICES")
			if createServices == "true" {
				framework.Logf("Creating services")
				services := generateServicesForConfigs(configs)
				for _, service := range services {
					_, err := clientset.Core().Services(service.Namespace).Create(service)
					framework.ExpectNoError(err)
				}
				framework.Logf("%v Services created.", len(services))
			} else {
				framework.Logf("Skipping service creation")
			}

			// Simulate lifetime of RC:
			//  * create with initial size
			//  * scale RC to a random size and list all pods
			//  * scale RC to a random size and list all pods
			//  * delete it
			//
			// This will generate ~5 creations/deletions per second assuming:
			//  - X small RCs each 5 pods   [ 5 * X = totalPods / 2 ]
			//  - Y medium RCs each 30 pods [ 30 * Y = totalPods / 4 ]
			//  - Z big RCs each 250 pods   [ 250 * Z = totalPods / 4]

			// We would like to spread creating replication controllers over time
			// to make it possible to create/schedule them in the meantime.
			// Currently we assume <throughput> pods/second average throughput.
			// We may want to revisit it in the future.
			framework.Logf("Starting to create ReplicationControllers...")
			creatingTime := time.Duration(totalPods/throughput) * time.Second
			createAllRC(configs, creatingTime)
			By("============================================================================")

			// We would like to spread scaling replication controllers over time
			// to make it possible to create/schedule & delete them in the meantime.
			// Currently we assume that <throughput> pods/second average throughput.
			// The expected number of created/deleted pods is less than totalPods/3.
			scalingTime := time.Duration(totalPods/(3*throughput)) * time.Second
			framework.Logf("Starting to scale ReplicationControllers first time...")
			scaleAllRC(configs, scalingTime)
			By("============================================================================")

			framework.Logf("Starting to scale ReplicationControllers second time...")
			scaleAllRC(configs, scalingTime)
			By("============================================================================")

			// Cleanup all created replication controllers.
			// Currently we assume <throughput> pods/second average deletion throughput.
			// We may want to revisit it in the future.
			deletingTime := time.Duration(totalPods/throughput) * time.Second
			framework.Logf("Starting to delete ReplicationControllers...")
			deleteAllRC(configs, deletingTime)
			if createServices == "true" {
				framework.Logf("Starting to delete services...")
				for _, service := range services {
					err := clientset.Core().Services(ns).Delete(service.Name, nil)
					framework.ExpectNoError(err)
				}
				framework.Logf("Services deleted")
			}
		})
	}
})

func createClients(numberOfClients int) ([]*internalclientset.Clientset, error) {
	clients := make([]*internalclientset.Clientset, numberOfClients)
	for i := 0; i < numberOfClients; i++ {
		config, err := framework.LoadConfig()
		Expect(err).NotTo(HaveOccurred())
		config.QPS = 100
		config.Burst = 200
		if framework.TestContext.KubeAPIContentType != "" {
			config.ContentType = framework.TestContext.KubeAPIContentType
		}

		// For the purpose of this test, we want to force that clients
		// do not share underlying transport (which is a default behavior
		// in Kubernetes). Thus, we are explicitly creating transport for
		// each client here.
		transportConfig, err := config.TransportConfig()
		if err != nil {
			return nil, err
		}
		tlsConfig, err := transport.TLSConfigFor(transportConfig)
		if err != nil {
			return nil, err
		}
		config.Transport = utilnet.SetTransportDefaults(&http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     tlsConfig,
			MaxIdleConnsPerHost: 100,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
		})
		// Overwrite TLS-related fields from config to avoid collision with
		// Transport field.
		config.TLSClientConfig = restclient.TLSClientConfig{}

		c, err := internalclientset.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		clients[i] = c
	}
	return clients, nil
}

func computeRCCounts(total int) (int, int, int) {
	// Small RCs owns ~0.5 of total number of pods, medium and big RCs ~0.25 each.
	// For example for 3000 pods (100 nodes, 30 pods per node) there are:
	//  - 300 small RCs each 5 pods
	//  - 25 medium RCs each 30 pods
	//  - 3 big RCs each 250 pods
	bigRCCount := total / 4 / bigRCSize
	total -= bigRCCount * bigRCSize
	mediumRCCount := total / 3 / mediumRCSize
	total -= mediumRCCount * mediumRCSize
	smallRCCount := total / smallRCSize
	return smallRCCount, mediumRCCount, bigRCCount
}

func generateRCConfigs(totalPods int, image string, command []string, nss []*api.Namespace) []*testutils.RCConfig {
	configs := make([]*testutils.RCConfig, 0)

	smallRCCount, mediumRCCount, bigRCCount := computeRCCounts(totalPods)
	configs = append(configs, generateRCConfigsForGroup(nss, smallRCGroupName, smallRCSize, smallRCCount, image, command)...)
	configs = append(configs, generateRCConfigsForGroup(nss, mediumRCGroupName, mediumRCSize, mediumRCCount, image, command)...)
	configs = append(configs, generateRCConfigsForGroup(nss, bigRCGroupName, bigRCSize, bigRCCount, image, command)...)

	// Create a number of clients to better simulate real usecase
	// where not everyone is using exactly the same client.
	rcsPerClient := 20
	clients, err := createClients((len(configs) + rcsPerClient - 1) / rcsPerClient)
	framework.ExpectNoError(err)

	for i := 0; i < len(configs); i++ {
		configs[i].Client = clients[i%len(clients)]
	}

	return configs
}

func generateRCConfigsForGroup(
	nss []*api.Namespace, groupName string, size, count int, image string, command []string) []*testutils.RCConfig {
	configs := make([]*testutils.RCConfig, 0, count)
	for i := 1; i <= count; i++ {
		config := &testutils.RCConfig{
			Client:     nil, // this will be overwritten later
			Name:       groupName + "-" + strconv.Itoa(i),
			Namespace:  nss[i%len(nss)].Name,
			Timeout:    10 * time.Minute,
			Image:      image,
			Command:    command,
			Replicas:   size,
			CpuRequest: 10,       // 0.01 core
			MemRequest: 26214400, // 25MB
		}
		configs = append(configs, config)
	}
	return configs
}

func generateServicesForConfigs(configs []*testutils.RCConfig) []*api.Service {
	services := make([]*api.Service, 0, len(configs))
	for _, config := range configs {
		serviceName := config.Name + "-svc"
		labels := map[string]string{"name": config.Name}
		service := &api.Service{
			ObjectMeta: api.ObjectMeta{
				Name:      serviceName,
				Namespace: config.Namespace,
			},
			Spec: api.ServiceSpec{
				Selector: labels,
				Ports: []api.ServicePort{{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				}},
			},
		}
		services = append(services, service)
	}
	return services
}

func sleepUpTo(d time.Duration) {
	time.Sleep(time.Duration(rand.Int63n(d.Nanoseconds())))
}

func createAllRC(configs []*testutils.RCConfig, creatingTime time.Duration) {
	var wg sync.WaitGroup
	wg.Add(len(configs))
	for _, config := range configs {
		go createRC(&wg, config, creatingTime)
	}
	wg.Wait()
}

func createRC(wg *sync.WaitGroup, config *testutils.RCConfig, creatingTime time.Duration) {
	defer GinkgoRecover()
	defer wg.Done()

	sleepUpTo(creatingTime)
	framework.ExpectNoError(framework.RunRC(*config), fmt.Sprintf("creating rc %s", config.Name))
}

func scaleAllRC(configs []*testutils.RCConfig, scalingTime time.Duration) {
	var wg sync.WaitGroup
	wg.Add(len(configs))
	for _, config := range configs {
		go scaleRC(&wg, config, scalingTime)
	}
	wg.Wait()
}

// Scales RC to a random size within [0.5*size, 1.5*size] and lists all the pods afterwards.
// Scaling happens always based on original size, not the current size.
func scaleRC(wg *sync.WaitGroup, config *testutils.RCConfig, scalingTime time.Duration) {
	defer GinkgoRecover()
	defer wg.Done()

	sleepUpTo(scalingTime)
	newSize := uint(rand.Intn(config.Replicas) + config.Replicas/2)
	framework.ExpectNoError(framework.ScaleRC(config.Client, config.Namespace, config.Name, newSize, true),
		fmt.Sprintf("scaling rc %s for the first time", config.Name))
	selector := labels.SelectorFromSet(labels.Set(map[string]string{"name": config.Name}))
	options := api.ListOptions{
		LabelSelector:   selector,
		ResourceVersion: "0",
	}
	_, err := config.Client.Core().Pods(config.Namespace).List(options)
	framework.ExpectNoError(err, fmt.Sprintf("listing pods from rc %v", config.Name))
}

func deleteAllRC(configs []*testutils.RCConfig, deletingTime time.Duration) {
	var wg sync.WaitGroup
	wg.Add(len(configs))
	for _, config := range configs {
		go deleteRC(&wg, config, deletingTime)
	}
	wg.Wait()
}

func deleteRC(wg *sync.WaitGroup, config *testutils.RCConfig, deletingTime time.Duration) {
	defer GinkgoRecover()
	defer wg.Done()

	sleepUpTo(deletingTime)
	if framework.TestContext.GarbageCollectorEnabled {
		framework.ExpectNoError(framework.DeleteRCAndWaitForGC(config.Client, config.Namespace, config.Name), fmt.Sprintf("deleting rc %s", config.Name))
	} else {
		framework.ExpectNoError(framework.DeleteRCAndPods(config.Client, config.Namespace, config.Name), fmt.Sprintf("deleting rc %s", config.Name))
	}
}

func CreateNamespaces(f *framework.Framework, namespaceCount int, namePrefix string) ([]*api.Namespace, error) {
	namespaces := []*api.Namespace{}
	for i := 1; i <= namespaceCount; i++ {
		namespace, err := f.CreateNamespace(fmt.Sprintf("%v-%d", namePrefix, i), nil)
		if err != nil {
			return []*api.Namespace{}, err
		}
		namespaces = append(namespaces, namespace)
	}
	return namespaces, nil
}
