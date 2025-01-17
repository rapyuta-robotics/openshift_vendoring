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

package testing

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/types"
)

type FakeManager struct{}

// Unused methods.
func (_ FakeManager) AddPod(_ *api.Pod)        {}
func (_ FakeManager) RemovePod(_ *api.Pod)     {}
func (_ FakeManager) CleanupPods(_ []*api.Pod) {}
func (_ FakeManager) Start()                   {}

func (_ FakeManager) UpdatePodStatus(_ types.UID, podStatus *api.PodStatus) {
	for i := range podStatus.ContainerStatuses {
		podStatus.ContainerStatuses[i].Ready = true
	}
}
