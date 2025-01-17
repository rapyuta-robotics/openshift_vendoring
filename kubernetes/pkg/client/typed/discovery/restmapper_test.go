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

package discovery

import (
	"reflect"
	"testing"

	"github.com/openshift/kubernetes/pkg/api/errors"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	"github.com/openshift/kubernetes/pkg/client/restclient/fake"
	"github.com/openshift/kubernetes/pkg/version"

	"github.com/openshift/github.com/emicklei/go-restful/swagger"
	"github.com/openshift/github.com/stretchr/testify/assert"
)

func TestRESTMapper(t *testing.T) {
	resources := []*APIGroupResources{
		{
			Group: unversioned.APIGroup{
				Versions: []unversioned.GroupVersionForDiscovery{
					{Version: "v1"},
					{Version: "v2"},
				},
				PreferredVersion: unversioned.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]unversioned.APIResource{
				"v1": {
					{Name: "pods", Namespaced: true, Kind: "Pod"},
				},
				"v2": {
					{Name: "pods", Namespaced: true, Kind: "Pod"},
				},
			},
		},
		{
			Group: unversioned.APIGroup{
				Name: "extensions",
				Versions: []unversioned.GroupVersionForDiscovery{
					{Version: "v1beta"},
				},
				PreferredVersion: unversioned.GroupVersionForDiscovery{Version: "v1beta"},
			},
			VersionedResources: map[string][]unversioned.APIResource{
				"v1beta": {
					{Name: "jobs", Namespaced: true, Kind: "Job"},
				},
			},
		},
	}

	restMapper := NewRESTMapper(resources, nil)

	kindTCs := []struct {
		input unversioned.GroupVersionResource
		want  unversioned.GroupVersionKind
	}{
		{
			input: unversioned.GroupVersionResource{
				Version:  "v1",
				Resource: "pods",
			},
			want: unversioned.GroupVersionKind{
				Version: "v1",
				Kind:    "Pod",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Version:  "v2",
				Resource: "pods",
			},
			want: unversioned.GroupVersionKind{
				Version: "v2",
				Kind:    "Pod",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Resource: "pods",
			},
			want: unversioned.GroupVersionKind{
				Version: "v1",
				Kind:    "Pod",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Resource: "jobs",
			},
			want: unversioned.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta",
				Kind:    "Job",
			},
		},
	}

	for _, tc := range kindTCs {
		got, err := restMapper.KindFor(tc.input)
		if err != nil {
			t.Errorf("KindFor(%#v) unexpected error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("KindFor(%#v) = %#v, want %#v", tc.input, got, tc.want)
		}
	}

	resourceTCs := []struct {
		input unversioned.GroupVersionResource
		want  unversioned.GroupVersionResource
	}{
		{
			input: unversioned.GroupVersionResource{
				Version:  "v1",
				Resource: "pods",
			},
			want: unversioned.GroupVersionResource{
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Version:  "v2",
				Resource: "pods",
			},
			want: unversioned.GroupVersionResource{
				Version:  "v2",
				Resource: "pods",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Resource: "pods",
			},
			want: unversioned.GroupVersionResource{
				Version:  "v1",
				Resource: "pods",
			},
		},
		{
			input: unversioned.GroupVersionResource{
				Resource: "jobs",
			},
			want: unversioned.GroupVersionResource{
				Group:    "extensions",
				Version:  "v1beta",
				Resource: "jobs",
			},
		},
	}

	for _, tc := range resourceTCs {
		got, err := restMapper.ResourceFor(tc.input)
		if err != nil {
			t.Errorf("ResourceFor(%#v) unexpected error: %v", tc.input, err)
			continue
		}

		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ResourceFor(%#v) = %#v, want %#v", tc.input, got, tc.want)
		}
	}
}

func TestDeferredDiscoveryRESTMapper_CacheMiss(t *testing.T) {
	assert := assert.New(t)

	cdc := fakeCachedDiscoveryInterface{fresh: false}
	m := NewDeferredDiscoveryRESTMapper(&cdc, registered.InterfacesFor)
	assert.False(cdc.fresh, "should NOT be fresh after instantiation")
	assert.Zero(cdc.invalidateCalls, "should not have called Invalidate()")

	gvk, err := m.KindFor(unversioned.GroupVersionResource{
		Group:    "a",
		Version:  "v1",
		Resource: "foo",
	})
	assert.NoError(err)
	assert.True(cdc.fresh, "should be fresh after a cache-miss")
	assert.Equal(cdc.invalidateCalls, 1, "should have called Invalidate() once")
	assert.Equal(gvk.Kind, "Foo")

	gvk, err = m.KindFor(unversioned.GroupVersionResource{
		Group:    "a",
		Version:  "v1",
		Resource: "foo",
	})
	assert.NoError(err)
	assert.Equal(cdc.invalidateCalls, 1, "should NOT have called Invalidate() again")

	gvk, err = m.KindFor(unversioned.GroupVersionResource{
		Group:    "a",
		Version:  "v1",
		Resource: "bar",
	})
	assert.Error(err)
	assert.Equal(cdc.invalidateCalls, 1, "should NOT have called Invalidate() again after another cache-miss, but with fresh==true")

	cdc.fresh = false
	gvk, err = m.KindFor(unversioned.GroupVersionResource{
		Group:    "a",
		Version:  "v1",
		Resource: "bar",
	})
	assert.Error(err)
	assert.Equal(cdc.invalidateCalls, 2, "should HAVE called Invalidate() again after another cache-miss, but with fresh==false")
}

type fakeCachedDiscoveryInterface struct {
	invalidateCalls int
	fresh           bool
	enabledA        bool
}

var _ CachedDiscoveryInterface = &fakeCachedDiscoveryInterface{}

func (c *fakeCachedDiscoveryInterface) Fresh() bool {
	return c.fresh
}

func (c *fakeCachedDiscoveryInterface) Invalidate() {
	c.invalidateCalls = c.invalidateCalls + 1
	c.fresh = true
	c.enabledA = true
}

func (c *fakeCachedDiscoveryInterface) RESTClient() restclient.Interface {
	return &fake.RESTClient{}
}

func (c *fakeCachedDiscoveryInterface) ServerGroups() (*unversioned.APIGroupList, error) {
	if c.enabledA {
		return &unversioned.APIGroupList{
			Groups: []unversioned.APIGroup{
				{
					Name: "a",
					Versions: []unversioned.GroupVersionForDiscovery{
						{
							GroupVersion: "a/v1",
							Version:      "v1",
						},
					},
					PreferredVersion: unversioned.GroupVersionForDiscovery{
						GroupVersion: "a/v1",
						Version:      "v1",
					},
				},
			},
		}, nil
	}
	return &unversioned.APIGroupList{}, nil
}

func (c *fakeCachedDiscoveryInterface) ServerResourcesForGroupVersion(groupVersion string) (*unversioned.APIResourceList, error) {
	if c.enabledA && groupVersion == "a/v1" {
		return &unversioned.APIResourceList{
			GroupVersion: "a/v1",
			APIResources: []unversioned.APIResource{
				{
					Name:       "foo",
					Kind:       "Foo",
					Namespaced: false,
				},
			},
		}, nil
	}

	return nil, errors.NewNotFound(unversioned.GroupResource{}, "")
}

func (c *fakeCachedDiscoveryInterface) ServerResources() (map[string]*unversioned.APIResourceList, error) {
	if c.enabledA {
		av1, _ := c.ServerResourcesForGroupVersion("a/v1")
		return map[string]*unversioned.APIResourceList{
			"a/v1": av1,
		}, nil
	}
	return map[string]*unversioned.APIResourceList{}, nil
}

func (c *fakeCachedDiscoveryInterface) ServerPreferredResources() ([]unversioned.GroupVersionResource, error) {
	if c.enabledA {
		return []unversioned.GroupVersionResource{
			{
				Group:    "a",
				Version:  "v1",
				Resource: "foo",
			},
		}, nil
	}
	return []unversioned.GroupVersionResource{}, nil
}

func (c *fakeCachedDiscoveryInterface) ServerPreferredNamespacedResources() ([]unversioned.GroupVersionResource, error) {
	return []unversioned.GroupVersionResource{}, nil
}

func (c *fakeCachedDiscoveryInterface) ServerVersion() (*version.Info, error) {
	return &version.Info{}, nil
}

func (c *fakeCachedDiscoveryInterface) SwaggerSchema(version unversioned.GroupVersion) (*swagger.ApiDeclaration, error) {
	return &swagger.ApiDeclaration{}, nil
}
