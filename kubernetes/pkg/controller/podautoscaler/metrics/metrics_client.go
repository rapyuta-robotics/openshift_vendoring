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

package metrics

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/v1"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	unversionedcore "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	"github.com/openshift/kubernetes/pkg/labels"

	heapster "github.com/openshift/k8s.io/heapster/metrics/api/v1/types"
	metrics_api "github.com/openshift/k8s.io/heapster/metrics/apis/metrics/v1alpha1"
)

// PodResourceInfo contains pod resourcemetric values as a map from pod names to
// metric values
type PodResourceInfo map[string]int64

// PodMetricsInfo contains pod resourcemetric values as a map from pod names to
// metric values
type PodMetricsInfo map[string]float64

// MetricsClient knows how to query a remote interface to retrieve container-level
// resource metrics as well as pod-level arbitrary metrics
type MetricsClient interface {
	// GetResourceMetric gets the given resource metric (and an associated oldest timestamp)
	// for all pods matching the specified selector in the given namespace
	GetResourceMetric(resource api.ResourceName, namespace string, selector labels.Selector) (PodResourceInfo, time.Time, error)

	// GetRawMetric gets the given metric (and an associated oldest timestamp)
	// for all pods matching the specified selector in the given namespace
	GetRawMetric(metricName string, namespace string, selector labels.Selector) (PodMetricsInfo, time.Time, error)
}

const (
	DefaultHeapsterNamespace = "kube-system"
	DefaultHeapsterScheme    = "http"
	DefaultHeapsterService   = "heapster"
	DefaultHeapsterPort      = "" // use the first exposed port on the service
)

var heapsterQueryStart = -5 * time.Minute

type HeapsterMetricsClient struct {
	services        unversionedcore.ServiceInterface
	podsGetter      unversionedcore.PodsGetter
	heapsterScheme  string
	heapsterService string
	heapsterPort    string
}

func NewHeapsterMetricsClient(client clientset.Interface, namespace, scheme, service, port string) MetricsClient {
	return &HeapsterMetricsClient{
		services:        client.Core().Services(namespace),
		podsGetter:      client.Core(),
		heapsterScheme:  scheme,
		heapsterService: service,
		heapsterPort:    port,
	}
}

func (h *HeapsterMetricsClient) GetResourceMetric(resource api.ResourceName, namespace string, selector labels.Selector) (PodResourceInfo, time.Time, error) {
	metricPath := fmt.Sprintf("/apis/metrics/v1alpha1/namespaces/%s/pods", namespace)
	params := map[string]string{"labelSelector": selector.String()}

	resultRaw, err := h.services.
		ProxyGet(h.heapsterScheme, h.heapsterService, h.heapsterPort, metricPath, params).
		DoRaw()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to get heapster service: %v", err)
	}

	glog.V(4).Infof("Heapster metrics result: %s", string(resultRaw))

	metrics := metrics_api.PodMetricsList{}
	err = json.Unmarshal(resultRaw, &metrics)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal heapster response: %v", err)
	}

	if len(metrics.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no metrics returned from heapster")
	}

	res := make(PodResourceInfo, len(metrics.Items))

	for _, m := range metrics.Items {
		podSum := int64(0)
		missing := len(m.Containers) == 0
		for _, c := range m.Containers {
			resValue, found := c.Usage[v1.ResourceName(resource)]
			if !found {
				missing = true
				glog.V(2).Infof("missing resource metric %v for container %s in pod %s/%s", resource, c.Name, namespace, m.Name)
				continue
			}
			podSum += resValue.MilliValue()
		}

		if !missing {
			res[m.Name] = int64(podSum)
		}
	}

	timestamp := time.Time{}
	if len(metrics.Items) > 0 {
		timestamp = metrics.Items[0].Timestamp.Time
	}

	return res, timestamp, nil
}

func (h *HeapsterMetricsClient) GetRawMetric(metricName string, namespace string, selector labels.Selector) (PodMetricsInfo, time.Time, error) {
	podList, err := h.podsGetter.Pods(namespace).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to get pod list while fetching metrics: %v", err)
	}

	if len(podList.Items) == 0 {
		return nil, time.Time{}, fmt.Errorf("no pods matched the provided selector")
	}

	podNames := make([]string, len(podList.Items))
	for i, pod := range podList.Items {
		podNames[i] = pod.Name
	}

	now := time.Now()

	startTime := now.Add(heapsterQueryStart)
	metricPath := fmt.Sprintf("/api/v1/model/namespaces/%s/pod-list/%s/metrics/%s",
		namespace,
		strings.Join(podNames, ","),
		metricName)

	resultRaw, err := h.services.
		ProxyGet(h.heapsterScheme, h.heapsterService, h.heapsterPort, metricPath, map[string]string{"start": startTime.Format(time.RFC3339)}).
		DoRaw()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to get heapster service: %v", err)
	}

	var metrics heapster.MetricResultList
	err = json.Unmarshal(resultRaw, &metrics)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to unmarshal heapster response: %v", err)
	}

	glog.V(4).Infof("Heapster metrics result: %s", string(resultRaw))

	if len(metrics.Items) != len(podNames) {
		// if we get too many metrics or two few metrics, we have no way of knowing which metric goes to which pod
		// (note that Heapster returns *empty* metric items when a pod does not exist or have that metric, so this
		// does not cover the "missing metric entry" case)
		return nil, time.Time{}, fmt.Errorf("requested metrics for %v pods, got metrics for %v", len(podNames), len(metrics.Items))
	}

	var timestamp *time.Time
	res := make(PodMetricsInfo, len(metrics.Items))
	for i, podMetrics := range metrics.Items {
		val, podTimestamp, hadMetrics := collapseTimeSamples(podMetrics, time.Minute)
		if hadMetrics {
			res[podNames[i]] = val
			if timestamp == nil || podTimestamp.Before(*timestamp) {
				timestamp = &podTimestamp
			}
		}
	}

	if timestamp == nil {
		timestamp = &time.Time{}
	}

	return res, *timestamp, nil
}

func collapseTimeSamples(metrics heapster.MetricResult, duration time.Duration) (float64, time.Time, bool) {
	floatSum := float64(0)
	intSum := int64(0)
	intSumCount := 0
	floatSumCount := 0

	var newest *heapster.MetricPoint // creation time of the newest sample for this pod
	for i, metricPoint := range metrics.Metrics {
		if newest == nil || newest.Timestamp.Before(metricPoint.Timestamp) {
			newest = &metrics.Metrics[i]
		}
	}
	if newest != nil {
		for _, metricPoint := range metrics.Metrics {
			if metricPoint.Timestamp.Add(duration).After(newest.Timestamp) {
				intSum += int64(metricPoint.Value)
				intSumCount++
				if metricPoint.FloatValue != nil {
					floatSum += *metricPoint.FloatValue
					floatSumCount++
				}
			}
		}

		if newest.FloatValue != nil {
			return floatSum / float64(floatSumCount), newest.Timestamp, true
		} else {
			return float64(intSum / int64(intSumCount)), newest.Timestamp, true
		}
	}

	return 0, time.Time{}, false
}
