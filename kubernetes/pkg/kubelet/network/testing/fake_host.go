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

package testing

// helper for testing plugins
// a fake host is created here that can be used by plugins for testing

import (
	"github.com/openshift/kubernetes/pkg/api"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	kubecontainer "github.com/openshift/kubernetes/pkg/kubelet/container"
	containertest "github.com/openshift/kubernetes/pkg/kubelet/container/testing"
)

type fakeNetworkHost struct {
	fakeNamespaceGetter
	kubeClient clientset.Interface
	Legacy     bool
	Runtime    *containertest.FakeRuntime
}

func NewFakeHost(kubeClient clientset.Interface) *fakeNetworkHost {
	host := &fakeNetworkHost{kubeClient: kubeClient, Legacy: true, Runtime: &containertest.FakeRuntime{}}
	return host
}

func (fnh *fakeNetworkHost) GetPodByName(name, namespace string) (*api.Pod, bool) {
	return nil, false
}

func (fnh *fakeNetworkHost) GetKubeClient() clientset.Interface {
	return nil
}

func (nh *fakeNetworkHost) GetRuntime() kubecontainer.Runtime {
	return nh.Runtime
}

func (nh *fakeNetworkHost) SupportsLegacyFeatures() bool {
	return nh.Legacy
}

type fakeNamespaceGetter struct {
	ns string
}

func (nh *fakeNamespaceGetter) GetNetNS(containerID string) (string, error) {
	return nh.ns, nil
}
