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

package user

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/apis/extensions"
	"github.com/openshift/kubernetes/pkg/util/validation/field"
)

// runAsAny implements the interface RunAsUserStrategy.
type runAsAny struct{}

var _ RunAsUserStrategy = &runAsAny{}

// NewRunAsAny provides a strategy that will return nil.
func NewRunAsAny(options *extensions.RunAsUserStrategyOptions) (RunAsUserStrategy, error) {
	return &runAsAny{}, nil
}

// Generate creates the uid based on policy rules.
func (s *runAsAny) Generate(pod *api.Pod, container *api.Container) (*int64, error) {
	return nil, nil
}

// Validate ensures that the specified values fall within the range of the strategy.
func (s *runAsAny) Validate(pod *api.Pod, container *api.Container) field.ErrorList {
	return field.ErrorList{}
}
