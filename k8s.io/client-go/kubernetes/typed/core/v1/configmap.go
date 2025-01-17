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

package v1

import (
	api "github.com/openshift/k8s.io/client-go/pkg/api"
	v1 "github.com/openshift/k8s.io/client-go/pkg/api/v1"
	watch "github.com/openshift/k8s.io/client-go/pkg/watch"
	rest "github.com/openshift/k8s.io/client-go/rest"
)

// ConfigMapsGetter has a method to return a ConfigMapInterface.
// A group's client should implement this interface.
type ConfigMapsGetter interface {
	ConfigMaps(namespace string) ConfigMapInterface
}

// ConfigMapInterface has methods to work with ConfigMap resources.
type ConfigMapInterface interface {
	Create(*v1.ConfigMap) (*v1.ConfigMap, error)
	Update(*v1.ConfigMap) (*v1.ConfigMap, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string) (*v1.ConfigMap, error)
	List(opts v1.ListOptions) (*v1.ConfigMapList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.ConfigMap, err error)
	ConfigMapExpansion
}

// configMaps implements ConfigMapInterface
type configMaps struct {
	client rest.Interface
	ns     string
}

// newConfigMaps returns a ConfigMaps
func newConfigMaps(c *CoreV1Client, namespace string) *configMaps {
	return &configMaps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a configMap and creates it.  Returns the server's representation of the configMap, and an error, if there is any.
func (c *configMaps) Create(configMap *v1.ConfigMap) (result *v1.ConfigMap, err error) {
	result = &v1.ConfigMap{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("configmaps").
		Body(configMap).
		Do().
		Into(result)
	return
}

// Update takes the representation of a configMap and updates it. Returns the server's representation of the configMap, and an error, if there is any.
func (c *configMaps) Update(configMap *v1.ConfigMap) (result *v1.ConfigMap, err error) {
	result = &v1.ConfigMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("configmaps").
		Name(configMap.Name).
		Body(configMap).
		Do().
		Into(result)
	return
}

// Delete takes name of the configMap and deletes it. Returns an error if one occurs.
func (c *configMaps) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("configmaps").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *configMaps) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("configmaps").
		VersionedParams(&listOptions, api.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the configMap, and returns the corresponding configMap object, and an error if there is any.
func (c *configMaps) Get(name string) (result *v1.ConfigMap, err error) {
	result = &v1.ConfigMap{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("configmaps").
		Name(name).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ConfigMaps that match those selectors.
func (c *configMaps) List(opts v1.ListOptions) (result *v1.ConfigMapList, err error) {
	result = &v1.ConfigMapList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("configmaps").
		VersionedParams(&opts, api.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested configMaps.
func (c *configMaps) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.client.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("configmaps").
		VersionedParams(&opts, api.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched configMap.
func (c *configMaps) Patch(name string, pt api.PatchType, data []byte, subresources ...string) (result *v1.ConfigMap, err error) {
	result = &v1.ConfigMap{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("configmaps").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
