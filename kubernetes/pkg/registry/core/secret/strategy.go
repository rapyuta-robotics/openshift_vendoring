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

package secret

import (
	"fmt"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/errors"
	"github.com/openshift/kubernetes/pkg/api/rest"
	"github.com/openshift/kubernetes/pkg/api/validation"
	"github.com/openshift/kubernetes/pkg/fields"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/registry/generic"
	"github.com/openshift/kubernetes/pkg/runtime"
	apistorage "github.com/openshift/kubernetes/pkg/storage"
	"github.com/openshift/kubernetes/pkg/util/validation/field"
)

// strategy implements behavior for Secret objects
type strategy struct {
	runtime.ObjectTyper
	api.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Secret
// objects via the REST API.
var Strategy = strategy{api.Scheme, api.SimpleNameGenerator}

var _ = rest.RESTCreateStrategy(Strategy)

var _ = rest.RESTUpdateStrategy(Strategy)

func (strategy) NamespaceScoped() bool {
	return true
}

func (strategy) PrepareForCreate(ctx api.Context, obj runtime.Object) {
}

func (strategy) Validate(ctx api.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateSecret(obj.(*api.Secret))
}

func (strategy) Canonicalize(obj runtime.Object) {
}

func (strategy) AllowCreateOnUpdate() bool {
	return false
}

func (strategy) PrepareForUpdate(ctx api.Context, obj, old runtime.Object) {
}

func (strategy) ValidateUpdate(ctx api.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateSecretUpdate(obj.(*api.Secret), old.(*api.Secret))
}

func (strategy) AllowUnconditionalUpdate() bool {
	return true
}

func (s strategy) Export(ctx api.Context, obj runtime.Object, exact bool) error {
	t, ok := obj.(*api.Secret)
	if !ok {
		// unexpected programmer error
		return fmt.Errorf("unexpected object: %v", obj)
	}
	s.PrepareForCreate(ctx, obj)
	if exact {
		return nil
	}
	// secrets that are tied to the UID of a service account cannot be exported anyway
	if t.Type == api.SecretTypeServiceAccountToken || len(t.Annotations[api.ServiceAccountUIDKey]) > 0 {
		errs := []*field.Error{
			field.Invalid(field.NewPath("type"), t, "can not export service account secrets"),
		}
		return errors.NewInvalid(api.Kind("Secret"), t.Name, errs)
	}
	return nil
}

// Matcher returns a generic matcher for a given label and field selector.
func Matcher(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			secret, ok := obj.(*api.Secret)
			if !ok {
				return nil, nil, fmt.Errorf("not a secret")
			}
			return labels.Set(secret.Labels), SelectableFields(secret), nil
		},
	}
}

// SelectableFields returns a field set that can be used for filter selection
func SelectableFields(obj *api.Secret) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
	secretSpecificFieldsSet := fields.Set{
		"type": string(obj.Type),
	}
	return generic.MergeFieldsSets(objectMetaFieldsSet, secretSpecificFieldsSet)
}
