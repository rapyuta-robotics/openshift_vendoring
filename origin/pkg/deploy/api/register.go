package api

import (
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/runtime"
)

const GroupName = ""
const FutureGroupName = "deploy.openshift.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) unversioned.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) unversioned.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DeploymentConfig{},
		&DeploymentConfigList{},
		&DeploymentConfigRollback{},
		&DeploymentRequest{},
		&DeploymentLog{},
		&DeploymentLogOptions{},
	)
	return nil
}

func (obj *DeploymentConfig) GetObjectKind() unversioned.ObjectKind         { return &obj.TypeMeta }
func (obj *DeploymentConfigList) GetObjectKind() unversioned.ObjectKind     { return &obj.TypeMeta }
func (obj *DeploymentConfigRollback) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
func (obj *DeploymentRequest) GetObjectKind() unversioned.ObjectKind        { return &obj.TypeMeta }
func (obj *DeploymentLog) GetObjectKind() unversioned.ObjectKind            { return &obj.TypeMeta }
func (obj *DeploymentLogOptions) GetObjectKind() unversioned.ObjectKind     { return &obj.TypeMeta }
