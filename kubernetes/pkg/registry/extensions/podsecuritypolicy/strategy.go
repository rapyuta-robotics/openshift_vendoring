/*
Copyright 2015 The Kubernetes Authors.

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

package podsecuritypolicy

import (
	"fmt"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/rest"
	"github.com/openshift/kubernetes/pkg/apis/extensions"
	"github.com/openshift/kubernetes/pkg/apis/extensions/validation"
	"github.com/openshift/kubernetes/pkg/fields"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/registry/generic"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/storage"
	"github.com/openshift/kubernetes/pkg/util/validation/field"
)

// strategy implements behavior for PodSecurityPolicy objects
type strategy struct {
	runtime.ObjectTyper
	api.NameGenerator
}

// Strategy is the default logic that applies when creating and updating PodSecurityPolicy
// objects via the REST API.
var Strategy = strategy{api.Scheme, api.SimpleNameGenerator}

var _ = rest.RESTCreateStrategy(Strategy)

var _ = rest.RESTUpdateStrategy(Strategy)

func (strategy) NamespaceScoped() bool {
	return false
}

func (strategy) AllowCreateOnUpdate() bool {
	return false
}

func (strategy) AllowUnconditionalUpdate() bool {
	return true
}

func (strategy) PrepareForCreate(ctx api.Context, obj runtime.Object) {
}

func (strategy) PrepareForUpdate(ctx api.Context, obj, old runtime.Object) {
}

func (strategy) Canonicalize(obj runtime.Object) {
}

func (strategy) Validate(ctx api.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidatePodSecurityPolicy(obj.(*extensions.PodSecurityPolicy))
}

func (strategy) ValidateUpdate(ctx api.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidatePodSecurityPolicyUpdate(old.(*extensions.PodSecurityPolicy), obj.(*extensions.PodSecurityPolicy))
}

// Matcher returns a generic matcher for a given label and field selector.
func MatchPodSecurityPolicy(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			psp, ok := obj.(*extensions.PodSecurityPolicy)
			if !ok {
				return nil, nil, fmt.Errorf("given object is not a pod security policy.")
			}
			return labels.Set(psp.ObjectMeta.Labels), PodSecurityPolicyToSelectableFields(psp), nil
		},
	}
}

// PodSecurityPolicyToSelectableFields returns a label set that represents the object
func PodSecurityPolicyToSelectableFields(obj *extensions.PodSecurityPolicy) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, false)
}
