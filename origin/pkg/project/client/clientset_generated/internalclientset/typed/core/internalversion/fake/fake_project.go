package fake

import (
	api "github.com/openshift/origin/pkg/project/api"
	pkg_api "github.com/openshift/kubernetes/pkg/api"
	unversioned "github.com/openshift/kubernetes/pkg/api/unversioned"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
	labels "github.com/openshift/kubernetes/pkg/labels"
	watch "github.com/openshift/kubernetes/pkg/watch"
)

// FakeProjects implements ProjectInterface
type FakeProjects struct {
	Fake *FakeCore
}

var projectsResource = unversioned.GroupVersionResource{Group: "", Version: "", Resource: "projects"}

func (c *FakeProjects) Create(project *api.Project) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootCreateAction(projectsResource, project), &api.Project{})
	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) Update(project *api.Project) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootUpdateAction(projectsResource, project), &api.Project{})
	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) Delete(name string, options *pkg_api.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewRootDeleteAction(projectsResource, name), &api.Project{})
	return err
}

func (c *FakeProjects) DeleteCollection(options *pkg_api.DeleteOptions, listOptions pkg_api.ListOptions) error {
	action := core.NewRootDeleteCollectionAction(projectsResource, listOptions)

	_, err := c.Fake.Invokes(action, &api.ProjectList{})
	return err
}

func (c *FakeProjects) Get(name string) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootGetAction(projectsResource, name), &api.Project{})
	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}

func (c *FakeProjects) List(opts pkg_api.ListOptions) (result *api.ProjectList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootListAction(projectsResource, opts), &api.ProjectList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &api.ProjectList{}
	for _, item := range obj.(*api.ProjectList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested projects.
func (c *FakeProjects) Watch(opts pkg_api.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewRootWatchAction(projectsResource, opts))
}

// Patch applies the patch and returns the patched project.
func (c *FakeProjects) Patch(name string, pt pkg_api.PatchType, data []byte, subresources ...string) (result *api.Project, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootPatchSubresourceAction(projectsResource, name, data, subresources...), &api.Project{})
	if obj == nil {
		return nil, err
	}
	return obj.(*api.Project), err
}
