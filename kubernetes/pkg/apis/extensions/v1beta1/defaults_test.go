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

package v1beta1_test

import (
	"reflect"
	"testing"

	"github.com/openshift/kubernetes/pkg/api"
	_ "github.com/openshift/kubernetes/pkg/api/install"
	"github.com/openshift/kubernetes/pkg/api/resource"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/api/v1"
	_ "github.com/openshift/kubernetes/pkg/apis/extensions/install"
	. "github.com/openshift/kubernetes/pkg/apis/extensions/v1beta1"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/intstr"
)

func TestSetDefaultDaemonSet(t *testing.T) {
	defaultLabels := map[string]string{"foo": "bar"}
	period := int64(v1.DefaultTerminationGracePeriodSeconds)
	defaultTemplate := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
		},
		ObjectMeta: v1.ObjectMeta{
			Labels: defaultLabels,
		},
	}
	templateNoLabel := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
		},
	}
	tests := []struct {
		original *DaemonSet
		expected *DaemonSet
	}{
		{ // Labels change/defaulting test.
			original: &DaemonSet{
				Spec: DaemonSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &DaemonSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: defaultLabels,
				},
				Spec: DaemonSetSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: defaultLabels,
					},
					Template: defaultTemplate,
				},
			},
		},
		{ // Labels change/defaulting test.
			original: &DaemonSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: DaemonSetSpec{
					Template: defaultTemplate,
				},
			},
			expected: &DaemonSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: DaemonSetSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: defaultLabels,
					},
					Template: defaultTemplate,
				},
			},
		},
		{ // Update strategy.
			original: &DaemonSet{},
			expected: &DaemonSet{
				Spec: DaemonSetSpec{
					Template: templateNoLabel,
				},
			},
		},
		{ // Update strategy.
			original: &DaemonSet{
				Spec: DaemonSetSpec{},
			},
			expected: &DaemonSet{
				Spec: DaemonSetSpec{
					Template: templateNoLabel,
				},
			},
		},
		{ // Custom unique label key.
			original: &DaemonSet{
				Spec: DaemonSetSpec{},
			},
			expected: &DaemonSet{
				Spec: DaemonSetSpec{
					Template: templateNoLabel,
				},
			},
		},
	}

	for i, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*DaemonSet)
		if !ok {
			t.Errorf("(%d) unexpected object: %v", i, got)
			t.FailNow()
		}
		if !reflect.DeepEqual(got.Spec, expected.Spec) {
			t.Errorf("(%d) got different than expected\ngot:\n\t%+v\nexpected:\n\t%+v", i, got.Spec, expected.Spec)
		}
	}
}

func TestSetDefaultDeployment(t *testing.T) {
	defaultIntOrString := intstr.FromInt(1)
	differentIntOrString := intstr.FromInt(5)
	period := int64(v1.DefaultTerminationGracePeriodSeconds)
	defaultTemplate := v1.PodTemplateSpec{
		Spec: v1.PodSpec{
			DNSPolicy:                     v1.DNSClusterFirst,
			RestartPolicy:                 v1.RestartPolicyAlways,
			SecurityContext:               &v1.PodSecurityContext{},
			TerminationGracePeriodSeconds: &period,
		},
	}
	tests := []struct {
		original *Deployment
		expected *Deployment
	}{
		{
			original: &Deployment{},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(1),
					Strategy: DeploymentStrategy{
						Type: RollingUpdateDeploymentStrategyType,
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge:       &defaultIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					Template: defaultTemplate,
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge: &differentIntOrString,
						},
					},
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						Type: RollingUpdateDeploymentStrategyType,
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge:       &differentIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					Template: defaultTemplate,
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(3),
					Strategy: DeploymentStrategy{
						Type:          RollingUpdateDeploymentStrategyType,
						RollingUpdate: nil,
					},
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(3),
					Strategy: DeploymentStrategy{
						Type: RollingUpdateDeploymentStrategyType,
						RollingUpdate: &RollingUpdateDeployment{
							MaxSurge:       &defaultIntOrString,
							MaxUnavailable: &defaultIntOrString,
						},
					},
					Template: defaultTemplate,
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					Template: defaultTemplate,
				},
			},
		},
		{
			original: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					ProgressDeadlineSeconds: newInt32(30),
				},
			},
			expected: &Deployment{
				Spec: DeploymentSpec{
					Replicas: newInt32(5),
					Strategy: DeploymentStrategy{
						Type: RecreateDeploymentStrategyType,
					},
					Template:                defaultTemplate,
					ProgressDeadlineSeconds: newInt32(30),
				},
			},
		},
	}

	for _, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		got, ok := obj2.(*Deployment)
		if !ok {
			t.Errorf("unexpected object: %v", got)
			t.FailNow()
		}
		if !reflect.DeepEqual(got.Spec, expected.Spec) {
			t.Errorf("object mismatch!\nexpected:\n\t%+v\ngot:\n\t%+v", got.Spec, expected.Spec)
		}
	}
}

func TestSetDefaultJob(t *testing.T) {
	defaultLabels := map[string]string{"default": "default"}
	tests := map[string]struct {
		original     *Job
		expected     *Job
		expectLabels bool
	}{
		"both unspecified -> sets both to 1": {
			original: &Job{
				Spec: JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Completions: newInt32(1),
					Parallelism: newInt32(1),
				},
			},
			expectLabels: true,
		},
		"both unspecified -> sets both to 1 and no default labels": {
			original: &Job{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"mylabel": "myvalue"},
				},
				Spec: JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Completions: newInt32(1),
					Parallelism: newInt32(1),
				},
			},
		},
		"WQ: Parallelism explicitly 0 and completions unset -> no change": {
			original: &Job{
				Spec: JobSpec{
					Parallelism: newInt32(0),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Parallelism: newInt32(0),
				},
			},
			expectLabels: true,
		},
		"WQ: Parallelism explicitly 2 and completions unset -> no change": {
			original: &Job{
				Spec: JobSpec{
					Parallelism: newInt32(2),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Parallelism: newInt32(2),
				},
			},
			expectLabels: true,
		},
		"Completions explicitly 2 and parallelism unset -> parallelism is defaulted": {
			original: &Job{
				Spec: JobSpec{
					Completions: newInt32(2),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Completions: newInt32(2),
					Parallelism: newInt32(1),
				},
			},
			expectLabels: true,
		},
		"Both set -> no change": {
			original: &Job{
				Spec: JobSpec{
					Completions: newInt32(10),
					Parallelism: newInt32(11),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Completions: newInt32(10),
					Parallelism: newInt32(11),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expectLabels: true,
		},
		"Both set, flipped -> no change": {
			original: &Job{
				Spec: JobSpec{
					Completions: newInt32(11),
					Parallelism: newInt32(10),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{Labels: defaultLabels},
					},
				},
			},
			expected: &Job{
				Spec: JobSpec{
					Completions: newInt32(11),
					Parallelism: newInt32(10),
				},
			},
			expectLabels: true,
		},
	}

	for name, test := range tests {
		original := test.original
		expected := test.expected
		obj2 := roundTrip(t, runtime.Object(original))
		actual, ok := obj2.(*Job)
		if !ok {
			t.Errorf("%s: unexpected object: %v", name, actual)
			t.FailNow()
		}
		if (actual.Spec.Completions == nil) != (expected.Spec.Completions == nil) {
			t.Errorf("%s: got different *completions than expected: %v %v", name, actual.Spec.Completions, expected.Spec.Completions)
		}
		if actual.Spec.Completions != nil && expected.Spec.Completions != nil {
			if *actual.Spec.Completions != *expected.Spec.Completions {
				t.Errorf("%s: got different completions than expected: %d %d", name, *actual.Spec.Completions, *expected.Spec.Completions)
			}
		}
		if (actual.Spec.Parallelism == nil) != (expected.Spec.Parallelism == nil) {
			t.Errorf("%s: got different *Parallelism than expected: %v %v", name, actual.Spec.Parallelism, expected.Spec.Parallelism)
		}
		if actual.Spec.Parallelism != nil && expected.Spec.Parallelism != nil {
			if *actual.Spec.Parallelism != *expected.Spec.Parallelism {
				t.Errorf("%s: got different parallelism than expected: %d %d", name, *actual.Spec.Parallelism, *expected.Spec.Parallelism)
			}
		}
		if test.expectLabels != reflect.DeepEqual(actual.Labels, actual.Spec.Template.Labels) {
			if test.expectLabels {
				t.Errorf("%s: expected: %v, got: %v", name, actual.Spec.Template.Labels, actual.Labels)
			} else {
				t.Errorf("%s: unexpected equality: %v", name, actual.Labels)
			}
		}
	}
}

func TestSetDefaultJobSelector(t *testing.T) {
	tests := []struct {
		original         *Job
		expectedSelector *unversioned.LabelSelector
	}{
		// selector set explicitly, nil autoSelector
		{
			original: &Job{
				Spec: JobSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: map[string]string{"job": "selector"},
					},
				},
			},
			expectedSelector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"job": "selector"},
			},
		},
		// selector set explicitly, autoSelector=true
		{
			original: &Job{
				Spec: JobSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: map[string]string{"job": "selector"},
					},
					AutoSelector: newBool(true),
				},
			},
			expectedSelector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"job": "selector"},
			},
		},
		// selector set explicitly, autoSelector=false
		{
			original: &Job{
				Spec: JobSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: map[string]string{"job": "selector"},
					},
					AutoSelector: newBool(false),
				},
			},
			expectedSelector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"job": "selector"},
			},
		},
		// selector from template labels
		{
			original: &Job{
				Spec: JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{"job": "selector"},
						},
					},
				},
			},
			expectedSelector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"job": "selector"},
			},
		},
		// selector from template labels, autoSelector=false
		{
			original: &Job{
				Spec: JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{"job": "selector"},
						},
					},
					AutoSelector: newBool(false),
				},
			},
			expectedSelector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"job": "selector"},
			},
		},
		// selector not copied from template labels, autoSelector=true
		{
			original: &Job{
				Spec: JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{"job": "selector"},
						},
					},
					AutoSelector: newBool(true),
				},
			},
			expectedSelector: nil,
		},
	}

	for i, testcase := range tests {
		obj2 := roundTrip(t, runtime.Object(testcase.original))
		got, ok := obj2.(*Job)
		if !ok {
			t.Errorf("%d: unexpected object: %v", i, got)
			t.FailNow()
		}
		if !reflect.DeepEqual(got.Spec.Selector, testcase.expectedSelector) {
			t.Errorf("%d: got different selectors %#v %#v", i, got.Spec.Selector, testcase.expectedSelector)
		}
	}
}

func TestSetDefaultReplicaSet(t *testing.T) {
	tests := []struct {
		rs             *ReplicaSet
		expectLabels   bool
		expectSelector bool
	}{
		{
			rs: &ReplicaSet{
				Spec: ReplicaSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabels:   true,
			expectSelector: true,
		},
		{
			rs: &ReplicaSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: ReplicaSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabels:   false,
			expectSelector: true,
		},
		{
			rs: &ReplicaSet{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"bar": "foo",
					},
				},
				Spec: ReplicaSetSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: map[string]string{
							"some": "other",
						},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabels:   false,
			expectSelector: false,
		},
		{
			rs: &ReplicaSet{
				Spec: ReplicaSetSpec{
					Selector: &unversioned.LabelSelector{
						MatchLabels: map[string]string{
							"some": "other",
						},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectLabels:   true,
			expectSelector: false,
		},
	}

	for _, test := range tests {
		rs := test.rs
		obj2 := roundTrip(t, runtime.Object(rs))
		rs2, ok := obj2.(*ReplicaSet)
		if !ok {
			t.Errorf("unexpected object: %v", rs2)
			t.FailNow()
		}
		if test.expectSelector != reflect.DeepEqual(rs2.Spec.Selector.MatchLabels, rs2.Spec.Template.Labels) {
			if test.expectSelector {
				t.Errorf("expected: %v, got: %v", rs2.Spec.Template.Labels, rs2.Spec.Selector)
			} else {
				t.Errorf("unexpected equality: %v", rs.Spec.Selector)
			}
		}
		if test.expectLabels != reflect.DeepEqual(rs2.Labels, rs2.Spec.Template.Labels) {
			if test.expectLabels {
				t.Errorf("expected: %v, got: %v", rs2.Spec.Template.Labels, rs2.Labels)
			} else {
				t.Errorf("unexpected equality: %v", rs.Labels)
			}
		}
	}
}

func TestSetDefaultReplicaSetReplicas(t *testing.T) {
	tests := []struct {
		rs             ReplicaSet
		expectReplicas int32
	}{
		{
			rs: ReplicaSet{
				Spec: ReplicaSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 1,
		},
		{
			rs: ReplicaSet{
				Spec: ReplicaSetSpec{
					Replicas: newInt32(0),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 0,
		},
		{
			rs: ReplicaSet{
				Spec: ReplicaSetSpec{
					Replicas: newInt32(3),
					Template: v1.PodTemplateSpec{
						ObjectMeta: v1.ObjectMeta{
							Labels: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			expectReplicas: 3,
		},
	}

	for _, test := range tests {
		rs := &test.rs
		obj2 := roundTrip(t, runtime.Object(rs))
		rs2, ok := obj2.(*ReplicaSet)
		if !ok {
			t.Errorf("unexpected object: %v", rs2)
			t.FailNow()
		}
		if rs2.Spec.Replicas == nil {
			t.Errorf("unexpected nil Replicas")
		} else if test.expectReplicas != *rs2.Spec.Replicas {
			t.Errorf("expected: %d replicas, got: %d", test.expectReplicas, *rs2.Spec.Replicas)
		}
	}
}

func TestDefaultRequestIsNotSetForReplicaSet(t *testing.T) {
	s := v1.PodSpec{}
	s.Containers = []v1.Container{
		{
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("100m"),
				},
			},
		},
	}
	rs := &ReplicaSet{
		Spec: ReplicaSetSpec{
			Replicas: newInt32(3),
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: s,
			},
		},
	}
	output := roundTrip(t, runtime.Object(rs))
	rs2 := output.(*ReplicaSet)
	defaultRequest := rs2.Spec.Template.Spec.Containers[0].Resources.Requests
	requestValue := defaultRequest[v1.ResourceCPU]
	if requestValue.String() != "0" {
		t.Errorf("Expected 0 request value, got: %s", requestValue.String())
	}
}

func TestSetDefaultHorizontalPodAutoscalerMinReplicas(t *testing.T) {
	tests := []struct {
		hpa            HorizontalPodAutoscaler
		expectReplicas int32
	}{
		{
			hpa:            HorizontalPodAutoscaler{},
			expectReplicas: 1,
		},
		{
			hpa: HorizontalPodAutoscaler{
				Spec: HorizontalPodAutoscalerSpec{
					MinReplicas: newInt32(3),
				},
			},
			expectReplicas: 3,
		},
	}

	for _, test := range tests {
		hpa := &test.hpa
		obj2 := roundTrip(t, runtime.Object(hpa))
		hpa2, ok := obj2.(*HorizontalPodAutoscaler)
		if !ok {
			t.Errorf("unexpected object: %v", hpa2)
			t.FailNow()
		}
		if hpa2.Spec.MinReplicas == nil {
			t.Errorf("unexpected nil MinReplicas")
		} else if test.expectReplicas != *hpa2.Spec.MinReplicas {
			t.Errorf("expected: %d MinReplicas, got: %d", test.expectReplicas, *hpa2.Spec.MinReplicas)
		}
	}
}

func TestSetDefaultHorizontalPodAutoscalerCpuUtilization(t *testing.T) {
	tests := []struct {
		hpa               HorizontalPodAutoscaler
		expectUtilization int32
	}{
		{
			hpa:               HorizontalPodAutoscaler{},
			expectUtilization: 80,
		},
		{
			hpa: HorizontalPodAutoscaler{
				Spec: HorizontalPodAutoscalerSpec{
					CPUUtilization: &CPUTargetUtilization{
						TargetPercentage: int32(50),
					},
				},
			},
			expectUtilization: 50,
		},
	}

	for _, test := range tests {
		hpa := &test.hpa
		obj2 := roundTrip(t, runtime.Object(hpa))
		hpa2, ok := obj2.(*HorizontalPodAutoscaler)
		if !ok {
			t.Errorf("unexpected object: %v", hpa2)
			t.FailNow()
		}
		if hpa2.Spec.CPUUtilization == nil {
			t.Errorf("unexpected nil CPUUtilization")
		} else if test.expectUtilization != hpa2.Spec.CPUUtilization.TargetPercentage {
			t.Errorf("expected: %d CPUUtilization, got: %d", test.expectUtilization, hpa2.Spec.CPUUtilization.TargetPercentage)
		}
	}
}

func roundTrip(t *testing.T, obj runtime.Object) runtime.Object {
	data, err := runtime.Encode(api.Codecs.LegacyCodec(SchemeGroupVersion), obj)
	if err != nil {
		t.Errorf("%v\n %#v", err, obj)
		return nil
	}
	obj2, err := runtime.Decode(api.Codecs.UniversalDecoder(), data)
	if err != nil {
		t.Errorf("%v\nData: %s\nSource: %#v", err, string(data), obj)
		return nil
	}
	obj3 := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(runtime.Object)
	err = api.Scheme.Convert(obj2, obj3, nil)
	if err != nil {
		t.Errorf("%v\nSource: %#v", err, obj2)
		return nil
	}
	return obj3
}

func newInt32(val int32) *int32 {
	p := new(int32)
	*p = val
	return p
}

func newString(val string) *string {
	p := new(string)
	*p = val
	return p
}

func newBool(val bool) *bool {
	b := new(bool)
	*b = val
	return b
}
