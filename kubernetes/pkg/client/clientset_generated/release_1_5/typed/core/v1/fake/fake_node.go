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
	api "github.com/openshift/kubernetes/pkg/api"
	unversioned "github.com/openshift/kubernetes/pkg/api/unversioned"
	v1 "github.com/openshift/kubernetes/pkg/api/v1"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
	labels "github.com/openshift/kubernetes/pkg/labels"
	watch "github.com/openshift/kubernetes/pkg/watch"
)

// FakeNodes implements NodeInterface
type FakeNodes struct {
	Fake *FakeCoreV1
}

var nodesResource = unversioned.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

func (c *FakeNodes) Create(node *v1.Node) (result *v1.Node, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootCreateAction(nodesResource, node), &v1.Node{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Node), err
}

func (c *FakeNodes) Update(node *v1.Node) (result *v1.Node, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootUpdateAction(nodesResource, node), &v1.Node{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Node), err
}

func (c *FakeNodes) UpdateStatus(node *v1.Node) (*v1.Node, error) {
	obj, err := c.Fake.
		Invokes(core.NewRootUpdateSubresourceAction(nodesResource, "status", node), &v1.Node{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Node), err
}

func (c *FakeNodes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(core.NewRootDeleteAction(nodesResource, name), &v1.Node{})
	return err
}

func (c *FakeNodes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := core.NewRootDeleteCollectionAction(nodesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1.NodeList{})
	return err
}

func (c *FakeNodes) Get(name string) (result *v1.Node, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootGetAction(nodesResource, name), &v1.Node{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Node), err
}

func (c *FakeNodes) List(opts v1.ListOptions) (result *v1.NodeList, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootListAction(nodesResource, opts), &v1.NodeList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := core.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.NodeList{}
	for _, item := range obj.(*v1.NodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested nodes.
func (c *FakeNodes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(core.NewRootWatchAction(nodesResource, opts))
}

// Patch applies the patch and returns the patched node.
func (c *FakeNodes) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.Node, err error) {
	obj, err := c.Fake.
		Invokes(core.NewRootPatchSubresourceAction(nodesResource, name, data, subresources...), &v1.Node{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.Node), err
}
