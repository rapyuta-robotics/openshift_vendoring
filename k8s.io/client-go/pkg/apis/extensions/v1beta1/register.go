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

package v1beta1

import (
	"github.com/openshift/k8s.io/client-go/pkg/api/unversioned"
	"github.com/openshift/k8s.io/client-go/pkg/api/v1"
	"github.com/openshift/k8s.io/client-go/pkg/runtime"
	versionedwatch "github.com/openshift/k8s.io/client-go/pkg/watch/versioned"
)

// GroupName is the group name use in this package
const GroupName = "extensions"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1beta1"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, addDefaultingFuncs, addConversionFuncs)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Deployment{},
		&DeploymentList{},
		&DeploymentRollback{},
		&HorizontalPodAutoscaler{},
		&HorizontalPodAutoscalerList{},
		&Job{},
		&JobList{},
		&ReplicationControllerDummy{},
		&Scale{},
		&ThirdPartyResource{},
		&ThirdPartyResourceList{},
		&DaemonSetList{},
		&DaemonSet{},
		&ThirdPartyResourceData{},
		&ThirdPartyResourceDataList{},
		&Ingress{},
		&IngressList{},
		&v1.ListOptions{},
		&v1.DeleteOptions{},
		&v1.ExportOptions{},
		&ReplicaSet{},
		&ReplicaSetList{},
		&PodSecurityPolicy{},
		&PodSecurityPolicyList{},
		&NetworkPolicy{},
		&NetworkPolicyList{},
	)
	// Add the watch version that applies
	versionedwatch.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
