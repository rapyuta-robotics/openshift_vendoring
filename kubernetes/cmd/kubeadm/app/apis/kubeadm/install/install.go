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

package install

import (
	"github.com/openshift/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	"github.com/openshift/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	"github.com/openshift/kubernetes/pkg/apimachinery/announced"
)

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  kubeadm.GroupName,
			VersionPreferenceOrder:     []string{v1alpha1.SchemeGroupVersion.Version},
			ImportPrefix:               "github.com/openshift/kubernetes/cmd/kubeadm/app/apis/kubeadm",
			AddInternalObjectsToScheme: kubeadm.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1alpha1.SchemeGroupVersion.Version: v1alpha1.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}
