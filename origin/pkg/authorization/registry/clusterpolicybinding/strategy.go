package clusterpolicybinding

import (
	"fmt"

	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/fields"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/runtime"
	kstorage "github.com/openshift/kubernetes/pkg/storage"
	"github.com/openshift/kubernetes/pkg/util/validation/field"

	authorizationapi "github.com/openshift/origin/pkg/authorization/api"
	"github.com/openshift/origin/pkg/authorization/api/validation"
)

// strategy implements behavior for nodes
type strategy struct {
	runtime.ObjectTyper
}

// Strategy is the default logic that applies when creating and updating ClusterPolicyBinding objects.
var Strategy = strategy{kapi.Scheme}

func (strategy) NamespaceScoped() bool {
	return false
}

// AllowCreateOnUpdate is false for policybindings.
func (strategy) AllowCreateOnUpdate() bool {
	return false
}

func (strategy) AllowUnconditionalUpdate() bool {
	return false
}

func (strategy) GenerateName(base string) string {
	return base
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (s strategy) PrepareForCreate(ctx kapi.Context, obj runtime.Object) {
	binding := obj.(*authorizationapi.ClusterPolicyBinding)

	s.scrubBindingRefs(binding)
	// force a delimited name, just in case we someday allow a reference to a global object that won't have a namespace.  We'll end up with a name like ":default".
	// ":" is not in the value space of namespaces, so no escaping is necessary
	binding.Name = authorizationapi.GetPolicyBindingName(binding.PolicyRef.Namespace)
}

// scrubBindingRefs discards pieces of the object references that we don't respect to avoid confusion.
func (s strategy) scrubBindingRefs(binding *authorizationapi.ClusterPolicyBinding) {
	binding.PolicyRef = kapi.ObjectReference{Namespace: binding.PolicyRef.Namespace, Name: authorizationapi.PolicyName}
	binding.PolicyRef.Namespace = ""

	for roleBindingKey, roleBinding := range binding.RoleBindings {
		roleBinding.RoleRef = kapi.ObjectReference{Namespace: binding.PolicyRef.Namespace, Name: roleBinding.RoleRef.Name}
		binding.RoleBindings[roleBindingKey] = roleBinding
	}
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (s strategy) PrepareForUpdate(ctx kapi.Context, obj, old runtime.Object) {
	binding := obj.(*authorizationapi.ClusterPolicyBinding)

	s.scrubBindingRefs(binding)
}

// Canonicalize normalizes the object after validation.
func (strategy) Canonicalize(obj runtime.Object) {
}

// Validate validates a new policyBinding.
func (strategy) Validate(ctx kapi.Context, obj runtime.Object) field.ErrorList {
	return validation.ValidateClusterPolicyBinding(obj.(*authorizationapi.ClusterPolicyBinding))
}

// ValidateUpdate is the default update validation for an end user.
func (strategy) ValidateUpdate(ctx kapi.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateClusterPolicyBindingUpdate(obj.(*authorizationapi.ClusterPolicyBinding), old.(*authorizationapi.ClusterPolicyBinding))
}

// Matcher returns a generic matcher for a given label and field selector.
func Matcher(label labels.Selector, field fields.Selector) kstorage.SelectionPredicate {
	return kstorage.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			policyBinding, ok := obj.(*authorizationapi.ClusterPolicyBinding)
			if !ok {
				return nil, nil, fmt.Errorf("not a policyBinding")
			}
			return labels.Set(policyBinding.ObjectMeta.Labels), authorizationapi.ClusterPolicyBindingToSelectableFields(policyBinding), nil
		},
	}
}
