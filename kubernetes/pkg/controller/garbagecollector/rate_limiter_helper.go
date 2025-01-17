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

package garbagecollector

import (
	"fmt"
	"strings"
	"sync"

	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/client/typed/dynamic"
	"github.com/openshift/kubernetes/pkg/util/metrics"
)

// RegisteredRateLimiter records the registered RateLimters to avoid
// duplication.
type RegisteredRateLimiter struct {
	rateLimiters map[unversioned.GroupVersion]*sync.Once
}

// NewRegisteredRateLimiter returns a new RegisteredRateLimiater.
// TODO: NewRegisteredRateLimiter is not dynamic. We need to find a better way
// when GC dynamically change the resources it monitors.
func NewRegisteredRateLimiter(resources []unversioned.GroupVersionResource) *RegisteredRateLimiter {
	rateLimiters := make(map[unversioned.GroupVersion]*sync.Once)
	for _, resource := range resources {
		gv := resource.GroupVersion()
		if _, found := rateLimiters[gv]; !found {
			rateLimiters[gv] = &sync.Once{}
		}
	}
	return &RegisteredRateLimiter{rateLimiters: rateLimiters}
}

func (r *RegisteredRateLimiter) registerIfNotPresent(gv unversioned.GroupVersion, client *dynamic.Client, prefix string) {
	once, found := r.rateLimiters[gv]
	if !found {
		return
	}
	once.Do(func() {
		if rateLimiter := client.GetRateLimiter(); rateLimiter != nil {
			group := strings.Replace(gv.Group, ".", ":", -1)
			metrics.RegisterMetricAndTrackRateLimiterUsage(fmt.Sprintf("%s_%s_%s", prefix, group, gv.Version), rateLimiter)
		}
	})
}
