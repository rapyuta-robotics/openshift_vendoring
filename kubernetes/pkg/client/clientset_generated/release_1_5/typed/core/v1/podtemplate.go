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

package v1

import (
	api "github.com/openshift/kubernetes/pkg/api"
	v1 "github.com/openshift/kubernetes/pkg/api/v1"
	restclient "github.com/openshift/kubernetes/pkg/client/restclient"
	watch "github.com/openshift/kubernetes/pkg/watch"
)

// PodTemplatesGetter has a method to return a PodTemplateInterface.
// A group's client should implement this interface.
type PodTemplatesGetter interface {
	PodTemplates(namespace string) PodTemplateInterface
}

// PodTemplateInterface has methods to work with PodTemplate resources.
type PodTemplateInterface interface {
	Create(*v1.PodTemplate) (*v1.PodTemplate, error)
	Update(*v1.PodTemplate) (*v1.PodTemplate, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string) (*v1.PodTemplate, error)
	List(opts v1.ListOptions) (*v1.PodTemplateList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.PodTemplate, err error)
	PodTemplateExpansion
}

// podTemplates implements PodTemplateInterface
type podTemplates struct {
	client restclient.Interface
	ns     string
}

// newPodTemplates returns a PodTemplates
func newPodTemplates(c *CoreV1Client, namespace string) *podTemplates {
	return &podTemplates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a podTemplate and creates it.  Returns the server's representation of the podTemplate, and an error, if there is any.
func (c *podTemplates) Create(podTemplate *v1.PodTemplate) (result *v1.PodTemplate, err error) {
	result = &v1.PodTemplate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("podtemplates").
		Body(podTemplate).
		Do().
		Into(result)
	return
}

// Update takes the representation of a podTemplate and updates it. Returns the server's representation of the podTemplate, and an error, if there is any.
func (c *podTemplates) Update(podTemplate *v1.PodTemplate) (result *v1.PodTemplate, err error) {
	result = &v1.PodTemplate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("podtemplates").
		Name(podTemplate.Name).
		Body(podTemplate).
		Do().
		Into(result)
	return
}

// Delete takes name of the podTemplate and deletes it. Returns an error if one occurs.
func (c *podTemplates) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("podtemplates").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *podTemplates) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("podtemplates").
		VersionedParams(&listOptions, api.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the podTemplate, and returns the corresponding podTemplate object, and an error if there is any.
func (c *podTemplates) Get(name string) (result *v1.PodTemplate, err error) {
	result = &v1.PodTemplate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("podtemplates").
		Name(name).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PodTemplates that match those selectors.
func (c *podTemplates) List(opts v1.ListOptions) (result *v1.PodTemplateList, err error) {
	result = &v1.PodTemplateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("podtemplates").
		VersionedParams(&opts, api.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested podTemplates.
func (c *podTemplates) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("podtemplates").
		VersionedParams(&opts, api.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched podTemplate.
func (c *podTemplates) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.PodTemplate, err error) {
	result = &v1.PodTemplate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("podtemplates").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
