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

package fake

import (
	unversioned "github.com/openshift/kubernetes/pkg/api/unversioned"
	policy "github.com/openshift/kubernetes/pkg/apis/policy"
	core "github.com/openshift/kubernetes/pkg/client/testing/core"
)

func (c *FakeEvictions) Evict(eviction *policy.Eviction) error {
	action := core.GetActionImpl{}
	action.Verb = "post"
	action.Namespace = c.ns
	action.Resource = unversioned.GroupVersionResource{Group: "", Version: "", Resource: "pods"}
	action.Subresource = "eviction"
	_, err := c.Fake.Invokes(action, eviction)
	return err
}
