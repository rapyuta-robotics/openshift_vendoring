package v1

import (
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/runtime"
)

const GroupName = ""

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ProjectRequestLimitConfig{},
	)
	return nil
}

func (obj *ProjectRequestLimitConfig) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
