/*
Copyright 2017 The Kubernetes Authors.

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

package fake

import (
	internalversion "github.com/openshift/kubernetes/federation/client/clientset_generated/federation_internalclientset/typed/extensions/internalversion"
	restclient "github.com/openshift/kubernetes/pkg/client/restclient"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
)

type FakeExtensions struct {
	*core.Fake
}

func (c *FakeExtensions) DaemonSets(namespace string) internalversion.DaemonSetInterface {
	return &FakeDaemonSets{c, namespace}
}

func (c *FakeExtensions) Deployments(namespace string) internalversion.DeploymentInterface {
	return &FakeDeployments{c, namespace}
}

func (c *FakeExtensions) Ingresses(namespace string) internalversion.IngressInterface {
	return &FakeIngresses{c, namespace}
}

func (c *FakeExtensions) ReplicaSets(namespace string) internalversion.ReplicaSetInterface {
	return &FakeReplicaSets{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeExtensions) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
