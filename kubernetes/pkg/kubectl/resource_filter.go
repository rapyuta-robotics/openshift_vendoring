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

package kubectl

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/v1"
	"github.com/openshift/kubernetes/pkg/runtime"
)

// FilterFunc is a function that knows how to filter a specific resource kind.
// It receives a generic runtime.Object which must be type-checked by the function.
// Returns a boolean value true if a resource is filtered, or false otherwise.
type FilterFunc func(runtime.Object, PrintOptions) bool

// Filters is a collection of filter funcs
type Filters []FilterFunc

func NewResourceFilter() Filters {
	return []FilterFunc{
		filterPods,
	}
}

// filterPods returns true if a pod should be skipped.
// defaults to true for terminated pods
func filterPods(obj runtime.Object, options PrintOptions) bool {
	switch p := obj.(type) {
	case *v1.Pod:
		reason := string(p.Status.Phase)
		if p.Status.Reason != "" {
			reason = p.Status.Reason
		}
		return !options.ShowAll && (reason == string(v1.PodSucceeded) || reason == string(v1.PodFailed))
	case *api.Pod:
		reason := string(p.Status.Phase)
		if p.Status.Reason != "" {
			reason = p.Status.Reason
		}
		return !options.ShowAll && (reason == string(api.PodSucceeded) || reason == string(api.PodFailed))
	}
	return false
}

// Filter loops through a collection of FilterFuncs until it finds one that can filter the given resource
func (f Filters) Filter(obj runtime.Object, opts *PrintOptions) (bool, error) {
	// check if the object is unstructured. If so, let's attempt to convert it to a type we can understand
	// before apply filter func.
	switch obj.(type) {
	case *runtime.UnstructuredList, *runtime.Unstructured, *runtime.Unknown:
		if objBytes, err := runtime.Encode(api.Codecs.LegacyCodec(), obj); err == nil {
			if decodedObj, err := runtime.Decode(api.Codecs.UniversalDecoder(), objBytes); err == nil {
				obj = decodedObj
			}
		}
	}

	for _, filter := range f {
		if ok := filter(obj, *opts); ok {
			return true, nil
		}
	}
	return false, nil
}
