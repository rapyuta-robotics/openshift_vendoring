/*
Copyright 2016 The Kubernetes Authors.

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

package v1beta1

import (
	api "github.com/openshift/k8s.io/client-go/pkg/api"
	v1 "github.com/openshift/k8s.io/client-go/pkg/api/v1"
	v1beta1 "github.com/openshift/k8s.io/client-go/pkg/apis/extensions/v1beta1"
	watch "github.com/openshift/k8s.io/client-go/pkg/watch"
	rest "github.com/openshift/k8s.io/client-go/rest"
)

// PodSecurityPoliciesGetter has a method to return a PodSecurityPolicyInterface.
// A group's client should implement this interface.
type PodSecurityPoliciesGetter interface {
	PodSecurityPolicies() PodSecurityPolicyInterface
}

// PodSecurityPolicyInterface has methods to work with PodSecurityPolicy resources.
type PodSecurityPolicyInterface interface {
	Create(*v1beta1.PodSecurityPolicy) (*v1beta1.PodSecurityPolicy, error)
	Update(*v1beta1.PodSecurityPolicy) (*v1beta1.PodSecurityPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string) (*v1beta1.PodSecurityPolicy, error)
	List(opts v1.ListOptions) (*v1beta1.PodSecurityPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1beta1.PodSecurityPolicy, err error)
	PodSecurityPolicyExpansion
}

// podSecurityPolicies implements PodSecurityPolicyInterface
type podSecurityPolicies struct {
	client rest.Interface
}

// newPodSecurityPolicies returns a PodSecurityPolicies
func newPodSecurityPolicies(c *ExtensionsV1beta1Client) *podSecurityPolicies {
	return &podSecurityPolicies{
		client: c.RESTClient(),
	}
}

// Create takes the representation of a podSecurityPolicy and creates it.  Returns the server's representation of the podSecurityPolicy, and an error, if there is any.
func (c *podSecurityPolicies) Create(podSecurityPolicy *v1beta1.PodSecurityPolicy) (result *v1beta1.PodSecurityPolicy, err error) {
	result = &v1beta1.PodSecurityPolicy{}
	err = c.client.Post().
		Resource("podsecuritypolicies").
		Body(podSecurityPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a podSecurityPolicy and updates it. Returns the server's representation of the podSecurityPolicy, and an error, if there is any.
func (c *podSecurityPolicies) Update(podSecurityPolicy *v1beta1.PodSecurityPolicy) (result *v1beta1.PodSecurityPolicy, err error) {
	result = &v1beta1.PodSecurityPolicy{}
	err = c.client.Put().
		Resource("podsecuritypolicies").
		Name(podSecurityPolicy.Name).
		Body(podSecurityPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the podSecurityPolicy and deletes it. Returns an error if one occurs.
func (c *podSecurityPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("podsecuritypolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *podSecurityPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("podsecuritypolicies").
		VersionedParams(&listOptions, api.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the podSecurityPolicy, and returns the corresponding podSecurityPolicy object, and an error if there is any.
func (c *podSecurityPolicies) Get(name string) (result *v1beta1.PodSecurityPolicy, err error) {
	result = &v1beta1.PodSecurityPolicy{}
	err = c.client.Get().
		Resource("podsecuritypolicies").
		Name(name).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PodSecurityPolicies that match those selectors.
func (c *podSecurityPolicies) List(opts v1.ListOptions) (result *v1beta1.PodSecurityPolicyList, err error) {
	result = &v1beta1.PodSecurityPolicyList{}
	err = c.client.Get().
		Resource("podsecuritypolicies").
		VersionedParams(&opts, api.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested podSecurityPolicies.
func (c *podSecurityPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Resource("podsecuritypolicies").
		VersionedParams(&opts, api.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched podSecurityPolicy.
func (c *podSecurityPolicies) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1beta1.PodSecurityPolicy, err error) {
	result = &v1beta1.PodSecurityPolicy{}
	err = c.client.Patch(pt).
		Resource("podsecuritypolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
