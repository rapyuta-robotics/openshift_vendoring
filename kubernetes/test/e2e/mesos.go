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

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/util/wait"
	"github.com/openshift/kubernetes/test/e2e/framework"

	. "github.com/openshift/github.com/onsi/ginkgo"
	. "github.com/openshift/github.com/onsi/gomega"
)

var _ = framework.KubeDescribe("Mesos", func() {
	f := framework.NewDefaultFramework("pods")
	var c clientset.Interface
	var ns string

	BeforeEach(func() {
		framework.SkipUnlessProviderIs("mesos/docker")
		c = f.ClientSet
		ns = f.Namespace.Name
	})

	It("applies slave attributes as labels", func() {
		nodeClient := f.ClientSet.Core().Nodes()

		rackA := labels.SelectorFromSet(map[string]string{"k8s.mesosphere.io/attribute-rack": "1"})
		options := api.ListOptions{LabelSelector: rackA}
		nodes, err := nodeClient.List(options)
		if err != nil {
			framework.Failf("Failed to query for node: %v", err)
		}
		Expect(len(nodes.Items)).To(Equal(1))

		var addr string
		for _, a := range nodes.Items[0].Status.Addresses {
			if a.Type == api.NodeInternalIP {
				addr = a.Address
			}
		}
		Expect(len(addr)).NotTo(Equal(""))
	})

	It("starts static pods on every node in the mesos cluster", func() {
		client := f.ClientSet
		framework.ExpectNoError(framework.AllNodesReady(client, wait.ForeverTestTimeout), "all nodes ready")

		nodelist := framework.GetReadySchedulableNodesOrDie(client)
		const ns = "static-pods"
		numpods := int32(len(nodelist.Items))
		framework.ExpectNoError(framework.WaitForPodsRunningReady(client, ns, numpods, wait.ForeverTestTimeout, map[string]string{}),
			fmt.Sprintf("number of static pods in namespace %s is %d", ns, numpods))
	})

	It("schedules pods annotated with roles on correct slaves", func() {
		// launch a pod to find a node which can launch a pod. We intentionally do
		// not just take the node list and choose the first of them. Depending on the
		// cluster and the scheduler it might be that a "normal" pod cannot be
		// scheduled onto it.
		By("Trying to launch a pod with a label to get a node which can launch it.")
		podName := "with-label"
		_, err := c.Core().Pods(ns).Create(&api.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind: "Pod",
			},
			ObjectMeta: api.ObjectMeta{
				Name: podName,
				Annotations: map[string]string{
					"k8s.mesosphere.io/roles": "public",
				},
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  podName,
						Image: framework.GetPauseImageName(f.ClientSet),
					},
				},
			},
		})
		framework.ExpectNoError(err)

		framework.ExpectNoError(framework.WaitForPodNameRunningInNamespace(c, podName, ns))
		pod, err := c.Core().Pods(ns).Get(podName)
		framework.ExpectNoError(err)

		nodeClient := f.ClientSet.Core().Nodes()

		// schedule onto node with rack=2 being assigned to the "public" role
		rack2 := labels.SelectorFromSet(map[string]string{
			"k8s.mesosphere.io/attribute-rack": "2",
		})
		options := api.ListOptions{LabelSelector: rack2}
		nodes, err := nodeClient.List(options)
		framework.ExpectNoError(err)

		Expect(nodes.Items[0].Name).To(Equal(pod.Spec.NodeName))
	})
})
