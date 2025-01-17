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

package generic

import (
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/quota"
)

// Ensure it implements the required interface
var _ quota.Registry = &GenericRegistry{}

// GenericRegistry implements Registry
type GenericRegistry struct {
	// internal evaluators by group kind
	InternalEvaluators map[unversioned.GroupKind]quota.Evaluator
}

// Evaluators returns the map of evaluators by groupKind
func (r *GenericRegistry) Evaluators() map[unversioned.GroupKind]quota.Evaluator {
	return r.InternalEvaluators
}
