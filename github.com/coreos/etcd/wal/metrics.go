// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wal

import "github.com/openshift/github.com/prometheus/client_golang/prometheus"

var (
	syncDurations = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "etcd",
		Subsystem: "disk",
		Name:      "wal_fsync_duration_seconds",
		Help:      "The latency distributions of fsync called by wal.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
	})
)

func init() {
	prometheus.MustRegister(syncDurations)
}
