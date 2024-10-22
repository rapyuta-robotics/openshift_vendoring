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

package master

// These imports are the API groups the API server will support.
import (
	"fmt"

	_ "github.com/openshift/kubernetes/pkg/api/install"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	_ "github.com/openshift/kubernetes/pkg/apis/apps/install"
	_ "github.com/openshift/kubernetes/pkg/apis/authentication/install"
	_ "github.com/openshift/kubernetes/pkg/apis/authorization/install"
	_ "github.com/openshift/kubernetes/pkg/apis/autoscaling/install"
	_ "github.com/openshift/kubernetes/pkg/apis/batch/install"
	_ "github.com/openshift/kubernetes/pkg/apis/certificates/install"
	_ "github.com/openshift/kubernetes/pkg/apis/componentconfig/install"
	_ "github.com/openshift/kubernetes/pkg/apis/extensions/install"
	_ "github.com/openshift/kubernetes/pkg/apis/imagepolicy/install"
	_ "github.com/openshift/kubernetes/pkg/apis/policy/install"
	_ "github.com/openshift/kubernetes/pkg/apis/rbac/install"
	_ "github.com/openshift/kubernetes/pkg/apis/storage/install"
)

func init() {
	if missingVersions := registered.ValidateEnvRequestedVersions(); len(missingVersions) != 0 {
		panic(fmt.Sprintf("KUBE_API_VERSIONS contains versions that are not installed: %q.", missingVersions))
	}
}
