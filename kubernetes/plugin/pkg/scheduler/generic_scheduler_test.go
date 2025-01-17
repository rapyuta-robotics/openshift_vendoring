/*
Copyright 2014 The Kubernetes Authors.

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

package scheduler

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/resource"
	"github.com/openshift/kubernetes/pkg/apis/extensions"
	"github.com/openshift/kubernetes/pkg/util/sets"
	"github.com/openshift/kubernetes/pkg/util/wait"
	"github.com/openshift/kubernetes/plugin/pkg/scheduler/algorithm"
	algorithmpredicates "github.com/openshift/kubernetes/plugin/pkg/scheduler/algorithm/predicates"
	algorithmpriorities "github.com/openshift/kubernetes/plugin/pkg/scheduler/algorithm/priorities"
	priorityutil "github.com/openshift/kubernetes/plugin/pkg/scheduler/algorithm/priorities/util"
	schedulerapi "github.com/openshift/kubernetes/plugin/pkg/scheduler/api"
	"github.com/openshift/kubernetes/plugin/pkg/scheduler/schedulercache"
)

func falsePredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	return false, []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate}, nil
}

func truePredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	return true, nil, nil
}

func matchesPredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	node := nodeInfo.Node()
	if node == nil {
		return false, nil, fmt.Errorf("node not found")
	}
	if pod.Name == node.Name {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate}, nil
}

func hasNoPodsPredicate(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (bool, []algorithm.PredicateFailureReason, error) {
	if len(nodeInfo.Pods()) == 0 {
		return true, nil, nil
	}
	return false, []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate}, nil
}

func numericPriority(pod *api.Pod, nodeNameToInfo map[string]*schedulercache.NodeInfo, nodes []*api.Node) (schedulerapi.HostPriorityList, error) {
	result := []schedulerapi.HostPriority{}
	for _, node := range nodes {
		score, err := strconv.Atoi(node.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, schedulerapi.HostPriority{
			Host:  node.Name,
			Score: score,
		})
	}
	return result, nil
}

func reverseNumericPriority(pod *api.Pod, nodeNameToInfo map[string]*schedulercache.NodeInfo, nodes []*api.Node) (schedulerapi.HostPriorityList, error) {
	var maxScore float64
	minScore := math.MaxFloat64
	reverseResult := []schedulerapi.HostPriority{}
	result, err := numericPriority(pod, nodeNameToInfo, nodes)
	if err != nil {
		return nil, err
	}

	for _, hostPriority := range result {
		maxScore = math.Max(maxScore, float64(hostPriority.Score))
		minScore = math.Min(minScore, float64(hostPriority.Score))
	}
	for _, hostPriority := range result {
		reverseResult = append(reverseResult, schedulerapi.HostPriority{
			Host:  hostPriority.Host,
			Score: int(maxScore + minScore - float64(hostPriority.Score)),
		})
	}

	return reverseResult, nil
}

func makeNodeList(nodeNames []string) []*api.Node {
	result := make([]*api.Node, 0, len(nodeNames))
	for _, nodeName := range nodeNames {
		result = append(result, &api.Node{ObjectMeta: api.ObjectMeta{Name: nodeName}})
	}
	return result
}

func TestSelectHost(t *testing.T) {
	scheduler := genericScheduler{}
	tests := []struct {
		list          schedulerapi.HostPriorityList
		possibleHosts sets.String
		expectsErr    bool
	}{
		{
			list: []schedulerapi.HostPriority{
				{Host: "machine1.1", Score: 1},
				{Host: "machine2.1", Score: 2},
			},
			possibleHosts: sets.NewString("machine2.1"),
			expectsErr:    false,
		},
		// equal scores
		{
			list: []schedulerapi.HostPriority{
				{Host: "machine1.1", Score: 1},
				{Host: "machine1.2", Score: 2},
				{Host: "machine1.3", Score: 2},
				{Host: "machine2.1", Score: 2},
			},
			possibleHosts: sets.NewString("machine1.2", "machine1.3", "machine2.1"),
			expectsErr:    false,
		},
		// out of order scores
		{
			list: []schedulerapi.HostPriority{
				{Host: "machine1.1", Score: 3},
				{Host: "machine1.2", Score: 3},
				{Host: "machine2.1", Score: 2},
				{Host: "machine3.1", Score: 1},
				{Host: "machine1.3", Score: 3},
			},
			possibleHosts: sets.NewString("machine1.1", "machine1.2", "machine1.3"),
			expectsErr:    false,
		},
		// empty priorityList
		{
			list:          []schedulerapi.HostPriority{},
			possibleHosts: sets.NewString(),
			expectsErr:    true,
		},
	}

	for _, test := range tests {
		// increase the randomness
		for i := 0; i < 10; i++ {
			got, err := scheduler.selectHost(test.list)
			if test.expectsErr {
				if err == nil {
					t.Error("Unexpected non-error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !test.possibleHosts.Has(got) {
					t.Errorf("got %s is not in the possible map %v", got, test.possibleHosts)
				}
			}
		}
	}
}

func TestGenericScheduler(t *testing.T) {
	tests := []struct {
		name          string
		predicates    map[string]algorithm.FitPredicate
		prioritizers  []algorithm.PriorityConfig
		nodes         []string
		pod           *api.Pod
		pods          []*api.Pod
		expectedHosts sets.String
		expectsErr    bool
		wErr          error
	}{
		{
			predicates:   map[string]algorithm.FitPredicate{"false": falsePredicate},
			prioritizers: []algorithm.PriorityConfig{{Map: EqualPriorityMap, Weight: 1}},
			nodes:        []string{"machine1", "machine2"},
			expectsErr:   true,
			pod:          &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
			name:         "test 1",
			wErr: &FitError{
				Pod: &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
				FailedPredicates: FailedPredicateMap{
					"machine1": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
					"machine2": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
				}},
		},
		{
			predicates:    map[string]algorithm.FitPredicate{"true": truePredicate},
			prioritizers:  []algorithm.PriorityConfig{{Map: EqualPriorityMap, Weight: 1}},
			nodes:         []string{"machine1", "machine2"},
			expectedHosts: sets.NewString("machine1", "machine2"),
			name:          "test 2",
			wErr:          nil,
		},
		{
			// Fits on a machine where the pod ID matches the machine name
			predicates:    map[string]algorithm.FitPredicate{"matches": matchesPredicate},
			prioritizers:  []algorithm.PriorityConfig{{Map: EqualPriorityMap, Weight: 1}},
			nodes:         []string{"machine1", "machine2"},
			pod:           &api.Pod{ObjectMeta: api.ObjectMeta{Name: "machine2"}},
			expectedHosts: sets.NewString("machine2"),
			name:          "test 3",
			wErr:          nil,
		},
		{
			predicates:    map[string]algorithm.FitPredicate{"true": truePredicate},
			prioritizers:  []algorithm.PriorityConfig{{Function: numericPriority, Weight: 1}},
			nodes:         []string{"3", "2", "1"},
			expectedHosts: sets.NewString("3"),
			name:          "test 4",
			wErr:          nil,
		},
		{
			predicates:    map[string]algorithm.FitPredicate{"matches": matchesPredicate},
			prioritizers:  []algorithm.PriorityConfig{{Function: numericPriority, Weight: 1}},
			nodes:         []string{"3", "2", "1"},
			pod:           &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
			expectedHosts: sets.NewString("2"),
			name:          "test 5",
			wErr:          nil,
		},
		{
			predicates:    map[string]algorithm.FitPredicate{"true": truePredicate},
			prioritizers:  []algorithm.PriorityConfig{{Function: numericPriority, Weight: 1}, {Function: reverseNumericPriority, Weight: 2}},
			nodes:         []string{"3", "2", "1"},
			pod:           &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
			expectedHosts: sets.NewString("1"),
			name:          "test 6",
			wErr:          nil,
		},
		{
			predicates:   map[string]algorithm.FitPredicate{"true": truePredicate, "false": falsePredicate},
			prioritizers: []algorithm.PriorityConfig{{Function: numericPriority, Weight: 1}},
			nodes:        []string{"3", "2", "1"},
			pod:          &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
			expectsErr:   true,
			name:         "test 7",
			wErr: &FitError{
				Pod: &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
				FailedPredicates: FailedPredicateMap{
					"3": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
					"2": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
					"1": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
				},
			},
		},
		{
			predicates: map[string]algorithm.FitPredicate{
				"nopods":  hasNoPodsPredicate,
				"matches": matchesPredicate,
			},
			pods: []*api.Pod{
				{
					ObjectMeta: api.ObjectMeta{Name: "2"},
					Spec: api.PodSpec{
						NodeName: "2",
					},
					Status: api.PodStatus{
						Phase: api.PodRunning,
					},
				},
			},
			pod:          &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
			prioritizers: []algorithm.PriorityConfig{{Function: numericPriority, Weight: 1}},
			nodes:        []string{"1", "2"},
			expectsErr:   true,
			name:         "test 8",
			wErr: &FitError{
				Pod: &api.Pod{ObjectMeta: api.ObjectMeta{Name: "2"}},
				FailedPredicates: FailedPredicateMap{
					"1": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
					"2": []algorithm.PredicateFailureReason{algorithmpredicates.ErrFakePredicate},
				},
			},
		},
	}
	for _, test := range tests {
		cache := schedulercache.New(time.Duration(0), wait.NeverStop)
		for _, pod := range test.pods {
			cache.AddPod(pod)
		}
		for _, name := range test.nodes {
			cache.AddNode(&api.Node{ObjectMeta: api.ObjectMeta{Name: name}})
		}

		scheduler := NewGenericScheduler(
			cache, test.predicates, algorithm.EmptyMetadataProducer, test.prioritizers, algorithm.EmptyMetadataProducer,
			[]algorithm.SchedulerExtender{})
		machine, err := scheduler.Schedule(test.pod, algorithm.FakeNodeLister(makeNodeList(test.nodes)))

		if !reflect.DeepEqual(err, test.wErr) {
			t.Errorf("Failed : %s, Unexpected error: %v, expected: %v", test.name, err, test.wErr)
		}
		if test.expectedHosts != nil && !test.expectedHosts.Has(machine) {
			t.Errorf("Failed : %s, Expected: %s, got: %s", test.name, test.expectedHosts, machine)
		}
	}
}

func TestFindFitAllError(t *testing.T) {
	nodes := []string{"3", "2", "1"}
	predicates := map[string]algorithm.FitPredicate{"true": truePredicate, "false": falsePredicate}
	nodeNameToInfo := map[string]*schedulercache.NodeInfo{
		"3": schedulercache.NewNodeInfo(),
		"2": schedulercache.NewNodeInfo(),
		"1": schedulercache.NewNodeInfo(),
	}
	_, predicateMap, err := findNodesThatFit(&api.Pod{}, nodeNameToInfo, makeNodeList(nodes), predicates, nil, algorithm.EmptyMetadataProducer)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(predicateMap) != len(nodes) {
		t.Errorf("unexpected failed predicate map: %v", predicateMap)
	}

	for _, node := range nodes {
		failures, found := predicateMap[node]
		if !found {
			t.Errorf("failed to find node: %s in %v", node, predicateMap)
		}
		if len(failures) != 1 || failures[0] != algorithmpredicates.ErrFakePredicate {
			t.Errorf("unexpected failures: %v", failures)
		}
	}
}

func TestFindFitSomeError(t *testing.T) {
	nodes := []string{"3", "2", "1"}
	predicates := map[string]algorithm.FitPredicate{"true": truePredicate, "match": matchesPredicate}
	pod := &api.Pod{ObjectMeta: api.ObjectMeta{Name: "1"}}
	nodeNameToInfo := map[string]*schedulercache.NodeInfo{
		"3": schedulercache.NewNodeInfo(),
		"2": schedulercache.NewNodeInfo(),
		"1": schedulercache.NewNodeInfo(pod),
	}
	for name := range nodeNameToInfo {
		nodeNameToInfo[name].SetNode(&api.Node{ObjectMeta: api.ObjectMeta{Name: name}})
	}

	_, predicateMap, err := findNodesThatFit(pod, nodeNameToInfo, makeNodeList(nodes), predicates, nil, algorithm.EmptyMetadataProducer)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(predicateMap) != (len(nodes) - 1) {
		t.Errorf("unexpected failed predicate map: %v", predicateMap)
	}

	for _, node := range nodes {
		if node == pod.Name {
			continue
		}
		failures, found := predicateMap[node]
		if !found {
			t.Errorf("failed to find node: %s in %v", node, predicateMap)
		}
		if len(failures) != 1 || failures[0] != algorithmpredicates.ErrFakePredicate {
			t.Errorf("unexpected failures: %v", failures)
		}
	}
}

func makeNode(node string, milliCPU, memory int64) *api.Node {
	return &api.Node{
		ObjectMeta: api.ObjectMeta{Name: node},
		Status: api.NodeStatus{
			Capacity: api.ResourceList{
				"cpu":    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
				"memory": *resource.NewQuantity(memory, resource.BinarySI),
			},
			Allocatable: api.ResourceList{
				"cpu":    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
				"memory": *resource.NewQuantity(memory, resource.BinarySI),
			},
		},
	}
}

// The point of this test is to show that you:
// - get the same priority for a zero-request pod as for a pod with the defaults requests,
//   both when the zero-request pod is already on the machine and when the zero-request pod
//   is the one being scheduled.
// - don't get the same score no matter what we schedule.
func TestZeroRequest(t *testing.T) {
	// A pod with no resources. We expect spreading to count it as having the default resources.
	noResources := api.PodSpec{
		Containers: []api.Container{
			{},
		},
	}
	noResources1 := noResources
	noResources1.NodeName = "machine1"
	// A pod with the same resources as a 0-request pod gets by default as its resources (for spreading).
	small := api.PodSpec{
		Containers: []api.Container{
			{
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{
						"cpu": resource.MustParse(
							strconv.FormatInt(priorityutil.DefaultMilliCpuRequest, 10) + "m"),
						"memory": resource.MustParse(
							strconv.FormatInt(priorityutil.DefaultMemoryRequest, 10)),
					},
				},
			},
		},
	}
	small2 := small
	small2.NodeName = "machine2"
	// A larger pod.
	large := api.PodSpec{
		Containers: []api.Container{
			{
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{
						"cpu": resource.MustParse(
							strconv.FormatInt(priorityutil.DefaultMilliCpuRequest*3, 10) + "m"),
						"memory": resource.MustParse(
							strconv.FormatInt(priorityutil.DefaultMemoryRequest*3, 10)),
					},
				},
			},
		},
	}
	large1 := large
	large1.NodeName = "machine1"
	large2 := large
	large2.NodeName = "machine2"
	tests := []struct {
		pod   *api.Pod
		pods  []*api.Pod
		nodes []*api.Node
		test  string
	}{
		// The point of these next two tests is to show you get the same priority for a zero-request pod
		// as for a pod with the defaults requests, both when the zero-request pod is already on the machine
		// and when the zero-request pod is the one being scheduled.
		{
			pod:   &api.Pod{Spec: noResources},
			nodes: []*api.Node{makeNode("machine1", 1000, priorityutil.DefaultMemoryRequest*10), makeNode("machine2", 1000, priorityutil.DefaultMemoryRequest*10)},
			test:  "test priority of zero-request pod with machine with zero-request pod",
			pods: []*api.Pod{
				{Spec: large1}, {Spec: noResources1},
				{Spec: large2}, {Spec: small2},
			},
		},
		{
			pod:   &api.Pod{Spec: small},
			nodes: []*api.Node{makeNode("machine1", 1000, priorityutil.DefaultMemoryRequest*10), makeNode("machine2", 1000, priorityutil.DefaultMemoryRequest*10)},
			test:  "test priority of nonzero-request pod with machine with zero-request pod",
			pods: []*api.Pod{
				{Spec: large1}, {Spec: noResources1},
				{Spec: large2}, {Spec: small2},
			},
		},
		// The point of this test is to verify that we're not just getting the same score no matter what we schedule.
		{
			pod:   &api.Pod{Spec: large},
			nodes: []*api.Node{makeNode("machine1", 1000, priorityutil.DefaultMemoryRequest*10), makeNode("machine2", 1000, priorityutil.DefaultMemoryRequest*10)},
			test:  "test priority of larger pod with machine with zero-request pod",
			pods: []*api.Pod{
				{Spec: large1}, {Spec: noResources1},
				{Spec: large2}, {Spec: small2},
			},
		},
	}

	const expectedPriority int = 25
	for _, test := range tests {
		// This should match the configuration in defaultPriorities() in
		// plugin/pkg/scheduler/algorithmprovider/defaults/defaults.go if you want
		// to test what's actually in production.
		priorityConfigs := []algorithm.PriorityConfig{
			{Map: algorithmpriorities.LeastRequestedPriorityMap, Weight: 1},
			{Map: algorithmpriorities.BalancedResourceAllocationMap, Weight: 1},
			{
				Function: algorithmpriorities.NewSelectorSpreadPriority(
					algorithm.FakeServiceLister([]*api.Service{}),
					algorithm.FakeControllerLister([]*api.ReplicationController{}),
					algorithm.FakeReplicaSetLister([]*extensions.ReplicaSet{})),
				Weight: 1,
			},
		}
		nodeNameToInfo := schedulercache.CreateNodeNameToInfoMap(test.pods, test.nodes)
		list, err := PrioritizeNodes(
			test.pod, nodeNameToInfo, algorithm.EmptyMetadataProducer, priorityConfigs,
			algorithm.FakeNodeLister(test.nodes), []algorithm.SchedulerExtender{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		for _, hp := range list {
			if test.test == "test priority of larger pod with machine with zero-request pod" {
				if hp.Score == expectedPriority {
					t.Errorf("%s: expected non-%d for all priorities, got list %#v", test.test, expectedPriority, list)
				}
			} else {
				if hp.Score != expectedPriority {
					t.Errorf("%s: expected %d for all priorities, got list %#v", test.test, expectedPriority, list)
				}
			}
		}
	}
}
