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

package kuberuntime

import (
	"encoding/json"
	"strconv"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	kubecontainer "github.com/openshift/kubernetes/pkg/kubelet/container"
	"github.com/openshift/kubernetes/pkg/kubelet/types"
	"github.com/openshift/kubernetes/pkg/kubelet/util/format"
	kubetypes "github.com/openshift/kubernetes/pkg/types"
)

const (
	// TODO: move those label definitions to kubelet/types/labels.go
	// TODO: change those label names to follow kubernetes's format
	podDeletionGracePeriodLabel    = "io.kubernetes.pod.deletionGracePeriod"
	podTerminationGracePeriodLabel = "io.kubernetes.pod.terminationGracePeriod"

	containerHashLabel                   = "io.kubernetes.container.hash"
	containerRestartCountLabel           = "io.kubernetes.container.restartCount"
	containerTerminationMessagePathLabel = "io.kubernetes.container.terminationMessagePath"
	containerPreStopHandlerLabel         = "io.kubernetes.container.preStopHandler"
	containerPortsLabel                  = "io.kubernetes.container.ports"

	// kubernetesManagedLabel is used to distinguish whether a container/sandbox is managed by kubelet or not
	kubernetesManagedLabel = "io.kubernetes.managed"
)

type labeledPodSandboxInfo struct {
	// Labels from api.Pod
	Labels       map[string]string
	PodName      string
	PodNamespace string
	PodUID       kubetypes.UID
}

type annotatedPodSandboxInfo struct {
	// Annotations from api.Pod
	Annotations map[string]string
}

type labeledContainerInfo struct {
	ContainerName string
	PodName       string
	PodNamespace  string
	PodUID        kubetypes.UID
}

type annotatedContainerInfo struct {
	Hash                      uint64
	RestartCount              int
	PodDeletionGracePeriod    *int64
	PodTerminationGracePeriod *int64
	TerminationMessagePath    string
	PreStopHandler            *api.Handler
	ContainerPorts            []api.ContainerPort
}

// newPodLabels creates pod labels from api.Pod.
func newPodLabels(pod *api.Pod) map[string]string {
	labels := map[string]string{}

	// Get labels from api.Pod
	for k, v := range pod.Labels {
		labels[k] = v
	}

	labels[types.KubernetesPodNameLabel] = pod.Name
	labels[types.KubernetesPodNamespaceLabel] = pod.Namespace
	labels[types.KubernetesPodUIDLabel] = string(pod.UID)
	labels[kubernetesManagedLabel] = "true"

	return labels
}

// newPodAnnotations creates pod annotations from api.Pod.
func newPodAnnotations(pod *api.Pod) map[string]string {
	return pod.Annotations
}

// newContainerLabels creates container labels from api.Container and api.Pod.
func newContainerLabels(container *api.Container, pod *api.Pod) map[string]string {
	labels := map[string]string{}
	labels[types.KubernetesPodNameLabel] = pod.Name
	labels[types.KubernetesPodNamespaceLabel] = pod.Namespace
	labels[types.KubernetesPodUIDLabel] = string(pod.UID)
	labels[types.KubernetesContainerNameLabel] = container.Name
	labels[kubernetesManagedLabel] = "true"

	return labels
}

// newContainerAnnotations creates container annotations from api.Container and api.Pod.
func newContainerAnnotations(container *api.Container, pod *api.Pod, restartCount int) map[string]string {
	annotations := map[string]string{}
	annotations[containerHashLabel] = strconv.FormatUint(kubecontainer.HashContainer(container), 16)
	annotations[containerRestartCountLabel] = strconv.Itoa(restartCount)
	annotations[containerTerminationMessagePathLabel] = container.TerminationMessagePath

	if pod.DeletionGracePeriodSeconds != nil {
		annotations[podDeletionGracePeriodLabel] = strconv.FormatInt(*pod.DeletionGracePeriodSeconds, 10)
	}
	if pod.Spec.TerminationGracePeriodSeconds != nil {
		annotations[podTerminationGracePeriodLabel] = strconv.FormatInt(*pod.Spec.TerminationGracePeriodSeconds, 10)
	}

	if container.Lifecycle != nil && container.Lifecycle.PreStop != nil {
		// Using json enconding so that the PreStop handler object is readable after writing as a label
		rawPreStop, err := json.Marshal(container.Lifecycle.PreStop)
		if err != nil {
			glog.Errorf("Unable to marshal lifecycle PreStop handler for container %q of pod %q: %v", container.Name, format.Pod(pod), err)
		} else {
			annotations[containerPreStopHandlerLabel] = string(rawPreStop)
		}
	}

	if len(container.Ports) > 0 {
		rawContainerPorts, err := json.Marshal(container.Ports)
		if err != nil {
			glog.Errorf("Unable to marshal container ports for container %q for pod %q: %v", container.Name, format.Pod(pod), err)
		} else {
			annotations[containerPortsLabel] = string(rawContainerPorts)
		}
	}

	return annotations
}

// getPodSandboxInfoFromLabels gets labeledPodSandboxInfo from labels.
func getPodSandboxInfoFromLabels(labels map[string]string) *labeledPodSandboxInfo {
	podSandboxInfo := &labeledPodSandboxInfo{
		Labels:       make(map[string]string),
		PodName:      getStringValueFromLabel(labels, types.KubernetesPodNameLabel),
		PodNamespace: getStringValueFromLabel(labels, types.KubernetesPodNamespaceLabel),
		PodUID:       kubetypes.UID(getStringValueFromLabel(labels, types.KubernetesPodUIDLabel)),
	}

	// Remain only labels from api.Pod
	for k, v := range labels {
		if k != types.KubernetesPodNameLabel && k != types.KubernetesPodNamespaceLabel && k != types.KubernetesPodUIDLabel && k != kubernetesManagedLabel {
			podSandboxInfo.Labels[k] = v
		}
	}

	return podSandboxInfo
}

// getPodSandboxInfoFromAnnotations gets annotatedPodSandboxInfo from annotations.
func getPodSandboxInfoFromAnnotations(annotations map[string]string) *annotatedPodSandboxInfo {
	return &annotatedPodSandboxInfo{
		Annotations: annotations,
	}
}

// getContainerInfoFromLabels gets labeledContainerInfo from labels.
func getContainerInfoFromLabels(labels map[string]string) *labeledContainerInfo {
	return &labeledContainerInfo{
		PodName:       getStringValueFromLabel(labels, types.KubernetesPodNameLabel),
		PodNamespace:  getStringValueFromLabel(labels, types.KubernetesPodNamespaceLabel),
		PodUID:        kubetypes.UID(getStringValueFromLabel(labels, types.KubernetesPodUIDLabel)),
		ContainerName: getStringValueFromLabel(labels, types.KubernetesContainerNameLabel),
	}
}

// isManagedByKubelet returns true is the sandbox/container is managed by kubelet.
func isManagedByKubelet(labels map[string]string) bool {
	if _, ok := labels[kubernetesManagedLabel]; ok {
		return true
	}

	return false
}

// getContainerInfoFromAnnotations gets annotatedContainerInfo from annotations.
func getContainerInfoFromAnnotations(annotations map[string]string) *annotatedContainerInfo {
	var err error
	containerInfo := &annotatedContainerInfo{
		TerminationMessagePath: getStringValueFromLabel(annotations, containerTerminationMessagePathLabel),
	}

	if containerInfo.Hash, err = getUint64ValueFromLabel(annotations, containerHashLabel); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", containerHashLabel, annotations, err)
	}
	if containerInfo.RestartCount, err = getIntValueFromLabel(annotations, containerRestartCountLabel); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", containerRestartCountLabel, annotations, err)
	}
	if containerInfo.PodDeletionGracePeriod, err = getInt64PointerFromLabel(annotations, podDeletionGracePeriodLabel); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", podDeletionGracePeriodLabel, annotations, err)
	}
	if containerInfo.PodTerminationGracePeriod, err = getInt64PointerFromLabel(annotations, podTerminationGracePeriodLabel); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", podTerminationGracePeriodLabel, annotations, err)
	}

	preStopHandler := &api.Handler{}
	if found, err := getJSONObjectFromLabel(annotations, containerPreStopHandlerLabel, preStopHandler); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", containerPreStopHandlerLabel, annotations, err)
	} else if found {
		containerInfo.PreStopHandler = preStopHandler
	}

	containerPorts := []api.ContainerPort{}
	if found, err := getJSONObjectFromLabel(annotations, containerPortsLabel, &containerPorts); err != nil {
		glog.Errorf("Unable to get %q from annotations %q: %v", containerPortsLabel, annotations, err)
	} else if found {
		containerInfo.ContainerPorts = containerPorts
	}

	return containerInfo
}

func getStringValueFromLabel(labels map[string]string, label string) string {
	if value, found := labels[label]; found {
		return value
	}
	// Do not report error, because there should be many old containers without label now.
	glog.V(3).Infof("Container doesn't have label %s, it may be an old or invalid container", label)
	// Return empty string "" for these containers, the caller will get value by other ways.
	return ""
}

func getIntValueFromLabel(labels map[string]string, label string) (int, error) {
	if strValue, found := labels[label]; found {
		intValue, err := strconv.Atoi(strValue)
		if err != nil {
			// This really should not happen. Just set value to 0 to handle this abnormal case
			return 0, err
		}
		return intValue, nil
	}
	// Do not report error, because there should be many old containers without label now.
	glog.V(3).Infof("Container doesn't have label %s, it may be an old or invalid container", label)
	// Just set the value to 0
	return 0, nil
}

func getUint64ValueFromLabel(labels map[string]string, label string) (uint64, error) {
	if strValue, found := labels[label]; found {
		intValue, err := strconv.ParseUint(strValue, 16, 64)
		if err != nil {
			// This really should not happen. Just set value to 0 to handle this abnormal case
			return 0, err
		}
		return intValue, nil
	}
	// Do not report error, because there should be many old containers without label now.
	glog.V(3).Infof("Container doesn't have label %s, it may be an old or invalid container", label)
	// Just set the value to 0
	return 0, nil
}

func getInt64PointerFromLabel(labels map[string]string, label string) (*int64, error) {
	if strValue, found := labels[label]; found {
		int64Value, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return nil, err
		}
		return &int64Value, nil
	}
	// If the label is not found, return pointer nil.
	return nil, nil
}

// getJSONObjectFromLabel returns a bool value indicating whether an object is found.
func getJSONObjectFromLabel(labels map[string]string, label string, value interface{}) (bool, error) {
	if strValue, found := labels[label]; found {
		err := json.Unmarshal([]byte(strValue), value)
		return found, err
	}
	// If the label is not found, return not found.
	return false, nil
}
