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

package cm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/kubelet/qos"
	"github.com/openshift/kubernetes/pkg/types"
	utilerrors "github.com/openshift/kubernetes/pkg/util/errors"
)

const (
	podCgroupNamePrefix = "pod"
)

// podContainerManagerImpl implements podContainerManager interface.
// It is the general implementation which allows pod level container
// management if qos Cgroup is enabled.
type podContainerManagerImpl struct {
	// nodeInfo stores information about the node resource capacity
	nodeInfo *api.Node
	// qosContainersInfo hold absolute paths of the top level qos containers
	qosContainersInfo QOSContainersInfo
	// Stores the mounted cgroup subsystems
	subsystems *CgroupSubsystems
	// cgroupManager is the cgroup Manager Object responsible for managing all
	// pod cgroups.
	cgroupManager CgroupManager
}

// Make sure that podContainerManagerImpl implements the PodContainerManager interface
var _ PodContainerManager = &podContainerManagerImpl{}

// applyLimits sets pod cgroup resource limits
// It also updates the resource limits on top level qos containers.
func (m *podContainerManagerImpl) applyLimits(pod *api.Pod) error {
	// This function will house the logic for setting the resource parameters
	// on the pod container config and updating top level qos container configs
	return nil
}

// Exists checks if the pod's cgroup already exists
func (m *podContainerManagerImpl) Exists(pod *api.Pod) bool {
	podContainerName, _ := m.GetPodContainerName(pod)
	return m.cgroupManager.Exists(podContainerName)
}

// EnsureExists takes a pod as argument and makes sure that
// pod cgroup exists if qos cgroup hierarchy flag is enabled.
// If the pod level container doesen't already exist it is created.
func (m *podContainerManagerImpl) EnsureExists(pod *api.Pod) error {
	podContainerName, _ := m.GetPodContainerName(pod)
	// check if container already exist
	alreadyExists := m.Exists(pod)
	if !alreadyExists {
		// Create the pod container
		containerConfig := &CgroupConfig{
			Name:               podContainerName,
			ResourceParameters: ResourceConfigForPod(pod),
		}
		if err := m.cgroupManager.Create(containerConfig); err != nil {
			return fmt.Errorf("failed to create container for %v : %v", podContainerName, err)
		}
	}
	// Apply appropriate resource limits on the pod container
	// Top level qos containers limits are not updated
	// until we figure how to maintain the desired state in the kubelet.
	// Because maintaining the desired state is difficult without checkpointing.
	if err := m.applyLimits(pod); err != nil {
		return fmt.Errorf("failed to apply resource limits on container for %v : %v", podContainerName, err)
	}
	return nil
}

// GetPodContainerName returns the CgroupName identifer, and its literal cgroupfs form on the host.
func (m *podContainerManagerImpl) GetPodContainerName(pod *api.Pod) (CgroupName, string) {
	podQOS := qos.GetPodQOS(pod)
	// Get the parent QOS container name
	var parentContainer string
	switch podQOS {
	case qos.Guaranteed:
		parentContainer = m.qosContainersInfo.Guaranteed
	case qos.Burstable:
		parentContainer = m.qosContainersInfo.Burstable
	case qos.BestEffort:
		parentContainer = m.qosContainersInfo.BestEffort
	}
	podContainer := podCgroupNamePrefix + string(pod.UID)

	// Get the absolute path of the cgroup
	cgroupName := (CgroupName)(path.Join(parentContainer, podContainer))
	// Get the literal cgroupfs name
	cgroupfsName := m.cgroupManager.Name(cgroupName)

	return cgroupName, cgroupfsName
}

// Scan through the whole cgroup directory and kill all processes either
// attached to the pod cgroup or to a container cgroup under the pod cgroup
func (m *podContainerManagerImpl) tryKillingCgroupProcesses(podCgroup CgroupName) error {
	pidsToKill := m.cgroupManager.Pids(podCgroup)
	// No pids charged to the terminated pod cgroup return
	if len(pidsToKill) == 0 {
		return nil
	}

	var errlist []error
	// os.Kill often errors out,
	// We try killing all the pids multiple times
	for i := 0; i < 5; i++ {
		if i != 0 {
			glog.V(3).Infof("Attempt %v failed to kill all unwanted process. Retyring", i)
		}
		errlist = []error{}
		for _, pid := range pidsToKill {
			p, err := os.FindProcess(pid)
			if err != nil {
				// Process not running anymore, do nothing
				continue
			}
			glog.V(3).Infof("Attempt to kill process with pid: %v", pid)
			if err := p.Kill(); err != nil {
				glog.V(3).Infof("failed to kill process with pid: %v", pid)
				errlist = append(errlist, err)
			}
		}
		if len(errlist) == 0 {
			glog.V(3).Infof("successfully killed all unwanted processes.")
			return nil
		}
	}
	return utilerrors.NewAggregate(errlist)
}

// Destroy destroys the pod container cgroup paths
func (m *podContainerManagerImpl) Destroy(podCgroup CgroupName) error {
	// Try killing all the processes attached to the pod cgroup
	if err := m.tryKillingCgroupProcesses(podCgroup); err != nil {
		glog.V(3).Infof("failed to kill all the processes attached to the %v cgroups", podCgroup)
		return fmt.Errorf("failed to kill all the processes attached to the %v cgroups : %v", podCgroup, err)
	}

	// Now its safe to remove the pod's cgroup
	containerConfig := &CgroupConfig{
		Name:               podCgroup,
		ResourceParameters: &ResourceConfig{},
	}
	if err := m.cgroupManager.Destroy(containerConfig); err != nil {
		return fmt.Errorf("failed to delete cgroup paths for %v : %v", podCgroup, err)
	}
	return nil
}

// ReduceCPULimits reduces the CPU CFS values to the minimum amount of shares.
func (m *podContainerManagerImpl) ReduceCPULimits(podCgroup CgroupName) error {
	return m.cgroupManager.ReduceCPULimits(podCgroup)
}

// GetAllPodsFromCgroups scans through all the subsytems of pod cgroups
// Get list of pods whose cgroup still exist on the cgroup mounts
func (m *podContainerManagerImpl) GetAllPodsFromCgroups() (map[types.UID]CgroupName, error) {
	// Map for storing all the found pods on the disk
	foundPods := make(map[types.UID]CgroupName)
	qosContainersList := [3]string{m.qosContainersInfo.BestEffort, m.qosContainersInfo.Burstable, m.qosContainersInfo.Guaranteed}
	// Scan through all the subsystem mounts
	// and through each QoS cgroup directory for each subsystem mount
	// If a pod cgroup exists in even a single subsystem mount
	// we will attempt to delete it
	for _, val := range m.subsystems.MountPoints {
		for _, qosContainerName := range qosContainersList {
			// get the subsystems QoS cgroup absolute name
			qcConversion := m.cgroupManager.Name(CgroupName(qosContainerName))
			qc := path.Join(val, qcConversion)
			dirInfo, err := ioutil.ReadDir(qc)
			if err != nil {
				return nil, fmt.Errorf("failed to read the cgroup directory %v : %v", qc, err)
			}
			for i := range dirInfo {
				// note: we do a contains check because on systemd, the literal cgroupfs name will prefix the qos as well.
				if dirInfo[i].IsDir() && strings.Contains(dirInfo[i].Name(), podCgroupNamePrefix) {
					// we need to convert the name to an internal identifier
					internalName := m.cgroupManager.CgroupName(dirInfo[i].Name())
					// we then split the name on the pod prefix to determine the uid
					parts := strings.Split(string(internalName), podCgroupNamePrefix)
					// the uid is missing, so we log the unexpected cgroup not of form pod<uid>
					if len(parts) != 2 {
						location := path.Join(qc, dirInfo[i].Name())
						glog.Errorf("pod cgroup manager ignoring unexpected cgroup %v because it is not a pod", location)
						continue
					}
					podUID := parts[1]
					// because the literal cgroupfs name could encode the qos tier (on systemd), we avoid double encoding
					// by just rebuilding the fully qualified CgroupName according to our internal convention.
					cgroupName := CgroupName(path.Join(qosContainerName, podCgroupNamePrefix+podUID))
					foundPods[types.UID(podUID)] = cgroupName
				}
			}
		}
	}
	return foundPods, nil
}

// podContainerManagerNoop implements podContainerManager interface.
// It is a no-op implementation and basically does nothing
// podContainerManagerNoop is used in case the QoS cgroup Hierarchy is not
// enabled, so Exists() returns true always as the cgroupRoot
// is expected to always exist.
type podContainerManagerNoop struct {
	cgroupRoot CgroupName
}

// Make sure that podContainerManagerStub implements the PodContainerManager interface
var _ PodContainerManager = &podContainerManagerNoop{}

func (m *podContainerManagerNoop) Exists(_ *api.Pod) bool {
	return true
}

func (m *podContainerManagerNoop) EnsureExists(_ *api.Pod) error {
	return nil
}

func (m *podContainerManagerNoop) GetPodContainerName(_ *api.Pod) (CgroupName, string) {
	return m.cgroupRoot, string(m.cgroupRoot)
}

func (m *podContainerManagerNoop) GetPodContainerNameForDriver(_ *api.Pod) string {
	return ""
}

// Destroy destroys the pod container cgroup paths
func (m *podContainerManagerNoop) Destroy(_ CgroupName) error {
	return nil
}

func (m *podContainerManagerNoop) ReduceCPULimits(_ CgroupName) error {
	return nil
}

func (m *podContainerManagerNoop) GetAllPodsFromCgroups() (map[types.UID]CgroupName, error) {
	return nil, nil
}
