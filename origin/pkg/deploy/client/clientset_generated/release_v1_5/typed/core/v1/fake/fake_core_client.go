package fake

import (
	v1 "github.com/openshift/origin/pkg/deploy/client/clientset_generated/release_v1_5/typed/core/v1"
	restclient "github.com/openshift/kubernetes/pkg/client/restclient"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
)

type FakeCoreV1 struct {
	*core.Fake
}

func (c *FakeCoreV1) DeploymentConfigs(namespace string) v1.DeploymentConfigInterface {
	return &FakeDeploymentConfigs{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCoreV1) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
