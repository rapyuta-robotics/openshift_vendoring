package fake

import (
	internalversion "github.com/openshift/origin/pkg/image/client/clientset_generated/internalclientset/typed/core/internalversion"
	restclient "github.com/openshift/kubernetes/pkg/client/restclient"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
)

type FakeCore struct {
	*core.Fake
}

func (c *FakeCore) Images(namespace string) internalversion.ImageInterface {
	return &FakeImages{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCore) RESTClient() restclient.Interface {
	var ret *restclient.RESTClient
	return ret
}
