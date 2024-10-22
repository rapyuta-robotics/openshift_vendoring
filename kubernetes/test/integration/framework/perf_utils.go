/*
Copyright 2016 The Kubernetes Authors.

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

package framework

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/resource"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	e2eframework "github.com/openshift/kubernetes/test/e2e/framework"
	testutils "github.com/openshift/kubernetes/test/utils"

	"github.com/golang/glog"
)

const (
	retries = 5
)

type IntegrationTestNodePreparer struct {
	client          clientset.Interface
	countToStrategy []testutils.CountToStrategy
	nodeNamePrefix  string
}

func NewIntegrationTestNodePreparer(client clientset.Interface, countToStrategy []testutils.CountToStrategy, nodeNamePrefix string) testutils.TestNodePreparer {
	return &IntegrationTestNodePreparer{
		client:          client,
		countToStrategy: countToStrategy,
		nodeNamePrefix:  nodeNamePrefix,
	}
}

func (p *IntegrationTestNodePreparer) PrepareNodes() error {
	numNodes := 0
	for _, v := range p.countToStrategy {
		numNodes += v.Count
	}

	glog.Infof("Making %d nodes", numNodes)
	baseNode := &api.Node{
		ObjectMeta: api.ObjectMeta{
			GenerateName: p.nodeNamePrefix,
		},
		Spec: api.NodeSpec{
			// TODO: investigate why this is needed.
			ExternalID: "foo",
		},
		Status: api.NodeStatus{
			Capacity: api.ResourceList{
				api.ResourcePods:   *resource.NewQuantity(110, resource.DecimalSI),
				api.ResourceCPU:    resource.MustParse("4"),
				api.ResourceMemory: resource.MustParse("32Gi"),
			},
			Phase: api.NodeRunning,
			Conditions: []api.NodeCondition{
				{Type: api.NodeReady, Status: api.ConditionTrue},
			},
		},
	}
	for i := 0; i < numNodes; i++ {
		if _, err := p.client.Core().Nodes().Create(baseNode); err != nil {
			glog.Fatalf("Error creating node: %v", err)
		}
	}

	nodes := e2eframework.GetReadySchedulableNodesOrDie(p.client)
	index := 0
	sum := 0
	for _, v := range p.countToStrategy {
		sum += v.Count
		for ; index < sum; index++ {
			if err := testutils.DoPrepareNode(p.client, &nodes.Items[index], v.Strategy); err != nil {
				glog.Errorf("Aborting node preparation: %v", err)
				return err
			}
		}
	}
	return nil
}

func (p *IntegrationTestNodePreparer) CleanupNodes() error {
	nodes := e2eframework.GetReadySchedulableNodesOrDie(p.client)
	for i := range nodes.Items {
		if err := p.client.Core().Nodes().Delete(nodes.Items[i].Name, &api.DeleteOptions{}); err != nil {
			glog.Errorf("Error while deleting Node: %v", err)
		}
	}
	return nil
}
