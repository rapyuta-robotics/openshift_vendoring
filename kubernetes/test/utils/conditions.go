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

package utils

import (
	"fmt"
	"github.com/openshift/kubernetes/pkg/api"
)

type ContainerFailures struct {
	status   *api.ContainerStateTerminated
	Restarts int
}

// PodRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
func PodRunningReady(p *api.Pod) (bool, error) {
	// Check the phase is running.
	if p.Status.Phase != api.PodRunning {
		return false, fmt.Errorf("want pod '%s' on '%s' to be '%v' but was '%v'",
			p.ObjectMeta.Name, p.Spec.NodeName, api.PodRunning, p.Status.Phase)
	}
	// Check the ready condition is true.
	if !PodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, api.PodReady, api.ConditionTrue, p.Status.Conditions)
	}
	return true, nil
}

func PodRunningReadyOrSucceeded(p *api.Pod) (bool, error) {
	// Check if the phase is succeeded.
	if p.Status.Phase == api.PodSucceeded {
		return true, nil
	}
	return PodRunningReady(p)
}

// FailedContainers inspects all containers in a pod and returns failure
// information for containers that have failed or been restarted.
// A map is returned where the key is the containerID and the value is a
// struct containing the restart and failure information
func FailedContainers(pod *api.Pod) map[string]ContainerFailures {
	var state ContainerFailures
	states := make(map[string]ContainerFailures)

	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return nil
	} else {
		for _, status := range statuses {
			if status.State.Terminated != nil {
				states[status.ContainerID] = ContainerFailures{status: status.State.Terminated}
			} else if status.LastTerminationState.Terminated != nil {
				states[status.ContainerID] = ContainerFailures{status: status.LastTerminationState.Terminated}
			}
			if status.RestartCount > 0 {
				var ok bool
				if state, ok = states[status.ContainerID]; !ok {
					state = ContainerFailures{}
				}
				state.Restarts = int(status.RestartCount)
				states[status.ContainerID] = state
			}
		}
	}

	return states
}

// TerminatedContainers inspects all containers in a pod and returns a map
// of "container name: termination reason", for all currently terminated
// containers.
func TerminatedContainers(pod *api.Pod) map[string]string {
	states := make(map[string]string)
	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return states
	}
	for _, status := range statuses {
		if status.State.Terminated != nil {
			states[status.Name] = status.State.Terminated.Reason
		}
	}
	return states
}

// PodNotReady checks whether pod p's has a ready condition of status false.
func PodNotReady(p *api.Pod) (bool, error) {
	// Check the ready condition is false.
	if PodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, api.PodReady, api.ConditionFalse, p.Status.Conditions)
	}
	return true, nil
}

// podReady returns whether pod has a condition of Ready with a status of true.
// TODO: should be replaced with api.IsPodReady
func PodReady(pod *api.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == api.PodReady && cond.Status == api.ConditionTrue {
			return true
		}
	}
	return false
}
