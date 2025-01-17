package fake

import (
	v1 "github.com/openshift/origin/pkg/template/api/v1"
	api "github.com/openshift/kubernetes/pkg/api"
	unversioned "github.com/openshift/kubernetes/pkg/api/unversioned"
	api_v1 "github.com/openshift/kubernetes/pkg/api/v1"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
	labels "github.com/openshift/kubernetes/pkg/labels"
	watch "github.com/openshift/kubernetes/pkg/watch"
)

// FakeTemplates implements TemplateInterface
type FakeTemplates struct {
	Fake *FakeCoreV1
	ns   string
}

var templatesResource = unversioned.GroupVersionResource{Group: "", Version: "v1", Resource: "templates"}

func (c *FakeTemplates) Create(template *v1.Template) (result *v1.Template, err error) {
	obj, err := c.Fake.
		Invokes(core.NewCreateAction(templatesResource, c.ns, template), &v1.Template{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Template), err
}

func (c *FakeTemplates) Update(template *v1.Template) (result *v1.Template, err error) {
	obj, err := c.Fake.
		Invokes(core.NewUpdateAction(templatesResource, c.ns, template), &v1.Template{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Template), err
}

func (c *FakeTemplates) Delete(name string, options *api_v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewDeleteAction(templatesResource, c.ns, name), &v1.Template{})

	return err
}

func (c *FakeTemplates) DeleteCollection(options *api_v1.DeleteOptions, listOptions api_v1.ListOptions) error {
	action := core.NewDeleteCollectionAction(templatesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1.TemplateList{})
	return err
}

func (c *FakeTemplates) Get(name string) (result *v1.Template, err error) {
	obj, err := c.Fake.
		Invokes(core.NewGetAction(templatesResource, c.ns, name), &v1.Template{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Template), err
}

func (c *FakeTemplates) List(opts api_v1.ListOptions) (result *v1.TemplateList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewListAction(templatesResource, c.ns, opts), &v1.TemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.TemplateList{}
	for _, item := range obj.(*v1.TemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested templates.
func (c *FakeTemplates) Watch(opts api_v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewWatchAction(templatesResource, c.ns, opts))

}

// Patch applies the patch and returns the patched template.
func (c *FakeTemplates) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.Template, err error) {
	obj, err := c.Fake.
		Invokes(core.NewPatchSubresourceAction(templatesResource, c.ns, name, data, subresources...), &v1.Template{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Template), err
}
