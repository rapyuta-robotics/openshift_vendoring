package fake

import (
	v1 "github.com/openshift/origin/pkg/oauth/api/v1"
	api "github.com/openshift/kubernetes/pkg/api"
	unversioned "github.com/openshift/kubernetes/pkg/api/unversioned"
	api_v1 "github.com/openshift/kubernetes/pkg/api/v1"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
	labels "github.com/openshift/kubernetes/pkg/labels"
	watch "github.com/openshift/kubernetes/pkg/watch"
)

// FakeOAuthClients implements OAuthClientInterface
type FakeOAuthClients struct {
	Fake *FakeCoreV1
	ns   string
}

var oauthclientsResource = unversioned.GroupVersionResource{Group: "", Version: "v1", Resource: "oauthclients"}

func (c *FakeOAuthClients) Create(oAuthClient *v1.OAuthClient) (result *v1.OAuthClient, err error) {
	obj, err := c.Fake.
		Invokes(core.NewCreateAction(oauthclientsResource, c.ns, oAuthClient), &v1.OAuthClient{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.OAuthClient), err
}

func (c *FakeOAuthClients) Update(oAuthClient *v1.OAuthClient) (result *v1.OAuthClient, err error) {
	obj, err := c.Fake.
		Invokes(core.NewUpdateAction(oauthclientsResource, c.ns, oAuthClient), &v1.OAuthClient{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.OAuthClient), err
}

func (c *FakeOAuthClients) Delete(name string, options *api_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewDeleteAction(oauthclientsResource, c.ns, name), &v1.OAuthClient{})

	return err
}

func (c *FakeOAuthClients) DeleteCollection(options *api_v1.DeleteOptions, listOptions api_v1.ListOptions) error {
	action := core.NewDeleteCollectionAction(oauthclientsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1.OAuthClientList{})
	return err
}

func (c *FakeOAuthClients) Get(name string) (result *v1.OAuthClient, err error) {
	obj, err := c.Fake.
		Invokes(core.NewGetAction(oauthclientsResource, c.ns, name), &v1.OAuthClient{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.OAuthClient), err
}

func (c *FakeOAuthClients) List(opts api_v1.ListOptions) (result *v1.OAuthClientList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewListAction(oauthclientsResource, c.ns, opts), &v1.OAuthClientList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.OAuthClientList{}
	for _, item := range obj.(*v1.OAuthClientList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested oAuthClients.
func (c *FakeOAuthClients) Watch(opts api_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewWatchAction(oauthclientsResource, c.ns, opts))

}

// Patch applies the patch and returns the patched oAuthClient.
func (c *FakeOAuthClients) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.OAuthClient, err error) {
	obj, err := c.Fake.
		Invokes(core.NewPatchSubresourceAction(oauthclientsResource, c.ns, name, data, subresources...), &v1.OAuthClient{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.OAuthClient), err
}
