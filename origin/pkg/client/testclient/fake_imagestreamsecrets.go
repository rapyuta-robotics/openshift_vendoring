package testclient

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/testing/core"

	"github.com/openshift/origin/pkg/client"
	imageapi "github.com/openshift/origin/pkg/image/api"
)

// FakeImageStreamSecrets implements ImageStreamSecretInterface. Meant to be
// embedded into a struct to get a default implementation. This makes faking
// out just the methods you want to test easier.
type FakeImageStreamSecrets struct {
	Fake      *Fake
	Namespace string
}

var _ client.ImageStreamSecretInterface = &FakeImageStreamSecrets{}

func (c *FakeImageStreamSecrets) Secrets(name string, options kapi.ListOptions) (*kapi.SecretList, error) {
	obj, err := c.Fake.Invokes(core.NewGetAction(imageapi.SchemeGroupVersion.WithResource("imagestreams/secrets"), c.Namespace, name), &kapi.SecretList{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kapi.SecretList), err
}
