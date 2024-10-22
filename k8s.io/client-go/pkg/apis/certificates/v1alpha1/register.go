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

package v1alpha1

import (
	"github.com/openshift/k8s.io/client-go/pkg/api/unversioned"
	"github.com/openshift/k8s.io/client-go/pkg/api/v1"
	"github.com/openshift/k8s.io/client-go/pkg/runtime"
	versionedwatch "github.com/openshift/k8s.io/client-go/pkg/watch/versioned"
)

// GroupName is the group name use in this package
const GroupName = "certificates.k8s.io"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) unversioned.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) unversioned.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addConversionFuncs)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&CertificateSigningRequest{},
		&CertificateSigningRequestList{},
		&v1.ListOptions{},
		&v1.DeleteOptions{},
		&v1.ExportOptions{},
	)

	// Add the watch version that applies
	versionedwatch.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

func (obj *CertificateSigningRequest) GetObjectKind() unversioned.ObjectKind     { return &obj.TypeMeta }
func (obj *CertificateSigningRequestList) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
