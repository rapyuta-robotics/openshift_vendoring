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

package container

import (
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	runtimeApi "github.com/openshift/kubernetes/pkg/kubelet/api/v1alpha1/runtime"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/flowcontrol"
	"github.com/openshift/kubernetes/pkg/util/term"
	"github.com/openshift/kubernetes/pkg/volume"
)

type Version interface {
	// Compare compares two versions of the runtime. On success it returns -1
	// if the version is less than the other, 1 if it is greater than the other,
	// or 0 if they are equal.
	Compare(other string) (int, error)
	// String returns a string that represents the version.
	String() string
}

// ImageSpec is an internal representation of an image.  Currently, it wraps the
// value of a Container's Image field, but in the future it will include more detailed
// information about the different image types.
type ImageSpec struct {
	Image string
}

// ImageStats contains statistics about all the images currently available.
type ImageStats struct {
	// Total amount of storage consumed by existing images.
	TotalStorageBytes uint64
}

// Runtime interface defines the interfaces that should be implemented
// by a container runtime.
// Thread safety is required from implementations of this interface.
type Runtime interface {
	// Type returns the type of the container runtime.
	Type() string

	// Version returns the version information of the container runtime.
	Version() (Version, error)

	// APIVersion returns the cached API version information of the container
	// runtime. Implementation is expected to update this cache periodically.
	// This may be different from the runtime engine's version.
	// TODO(random-liu): We should fold this into Version()
	APIVersion() (Version, error)
	// Status returns the status of the runtime. An error is returned if the Status
	// function itself fails, nil otherwise.
	Status() (*RuntimeStatus, error)
	// GetPods returns a list of containers grouped by pods. The boolean parameter
	// specifies whether the runtime returns all containers including those already
	// exited and dead containers (used for garbage collection).
	GetPods(all bool) ([]*Pod, error)
	// GarbageCollect removes dead containers using the specified container gc policy
	// If allSourcesReady is not true, it means that kubelet doesn't have the
	// complete list of pods from all avialble sources (e.g., apiserver, http,
	// file). In this case, garbage collector should refrain itself from aggressive
	// behavior such as removing all containers of unrecognized pods (yet).
	// TODO: Revisit this method and make it cleaner.
	GarbageCollect(gcPolicy ContainerGCPolicy, allSourcesReady bool) error
	// Syncs the running pod into the desired pod.
	SyncPod(pod *api.Pod, apiPodStatus api.PodStatus, podStatus *PodStatus, pullSecrets []api.Secret, backOff *flowcontrol.Backoff) PodSyncResult
	// KillPod kills all the containers of a pod. Pod may be nil, running pod must not be.
	// TODO(random-liu): Return PodSyncResult in KillPod.
	// gracePeriodOverride if specified allows the caller to override the pod default grace period.
	// only hard kill paths are allowed to specify a gracePeriodOverride in the kubelet in order to not corrupt user data.
	// it is useful when doing SIGKILL for hard eviction scenarios, or max grace period during soft eviction scenarios.
	KillPod(pod *api.Pod, runningPod Pod, gracePeriodOverride *int64) error
	// GetPodStatus retrieves the status of the pod, including the
	// information of all containers in the pod that are visble in Runtime.
	GetPodStatus(uid types.UID, name, namespace string) (*PodStatus, error)
	// Returns the filesystem path of the pod's network namespace; if the
	// runtime does not handle namespace creation itself, or cannot return
	// the network namespace path, it should return an error.
	// TODO: Change ContainerID to a Pod ID since the namespace is shared
	// by all containers in the pod.
	GetNetNS(containerID ContainerID) (string, error)
	// Returns the container ID that represents the Pod, as passed to network
	// plugins. For example, if the runtime uses an infra container, returns
	// the infra container's ContainerID.
	// TODO: Change ContainerID to a Pod ID, see GetNetNS()
	GetPodContainerID(*Pod) (ContainerID, error)
	// TODO(vmarmol): Unify pod and containerID args.
	// GetContainerLogs returns logs of a specific container. By
	// default, it returns a snapshot of the container log. Set 'follow' to true to
	// stream the log. Set 'follow' to false and specify the number of lines (e.g.
	// "100" or "all") to tail the log.
	GetContainerLogs(pod *api.Pod, containerID ContainerID, logOptions *api.PodLogOptions, stdout, stderr io.Writer) (err error)
	// Delete a container. If the container is still running, an error is returned.
	DeleteContainer(containerID ContainerID) error
	// ImageService provides methods to image-related methods.
	ImageService
	// UpdatePodCIDR sends a new podCIDR to the runtime.
	// This method just proxies a new runtimeConfig with the updated
	// CIDR value down to the runtime shim.
	UpdatePodCIDR(podCIDR string) error
}

// DirectStreamingRuntime is the interface implemented by runtimes for which the streaming calls
// (exec/attach/port-forward) should be served directly by the Kubelet.
type DirectStreamingRuntime interface {
	// Runs the command in the container of the specified pod using nsenter.
	// Attaches the processes stdin, stdout, and stderr. Optionally uses a
	// tty.
	ExecInContainer(containerID ContainerID, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan term.Size, timeout time.Duration) error
	// Forward the specified port from the specified pod to the stream.
	PortForward(pod *Pod, port uint16, stream io.ReadWriteCloser) error
	// ContainerAttach encapsulates the attaching to containers for testability
	ContainerAttacher
}

// IndirectStreamingRuntime is the interface implemented by runtimes that handle the serving of the
// streaming calls (exec/attach/port-forward) themselves. In this case, Kubelet should redirect to
// the runtime server.
type IndirectStreamingRuntime interface {
	GetExec(id ContainerID, cmd []string, stdin, stdout, stderr, tty bool) (*url.URL, error)
	GetAttach(id ContainerID, stdin, stdout, stderr bool) (*url.URL, error)
	GetPortForward(podName, podNamespace string, podUID types.UID) (*url.URL, error)
}

type ImageService interface {
	// PullImage pulls an image from the network to local storage using the supplied
	// secrets if necessary.
	PullImage(image ImageSpec, pullSecrets []api.Secret) error
	// IsImagePresent checks whether the container image is already in the local storage.
	IsImagePresent(image ImageSpec) (bool, error)
	// Gets all images currently on the machine.
	ListImages() ([]Image, error)
	// Removes the specified image.
	RemoveImage(image ImageSpec) error
	// Returns Image statistics.
	ImageStats() (*ImageStats, error)
}

type ContainerAttacher interface {
	AttachContainer(id ContainerID, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan term.Size) (err error)
}

type ContainerCommandRunner interface {
	// RunInContainer synchronously executes the command in the container, and returns the output.
	// If the command completes with a non-0 exit code, a pkg/util/exec.ExitError will be returned.
	RunInContainer(id ContainerID, cmd []string, timeout time.Duration) ([]byte, error)
}

// Pod is a group of containers.
type Pod struct {
	// The ID of the pod, which can be used to retrieve a particular pod
	// from the pod list returned by GetPods().
	ID types.UID
	// The name and namespace of the pod, which is readable by human.
	Name      string
	Namespace string
	// List of containers that belongs to this pod. It may contain only
	// running containers, or mixed with dead ones (when GetPods(true)).
	Containers []*Container
	// List of sandboxes associated with this pod. The sandboxes are converted
	// to Container temporariliy to avoid substantial changes to other
	// components. This is only populated by kuberuntime.
	// TODO: use the runtimeApi.PodSandbox type directly.
	Sandboxes []*Container
}

// PodPair contains both runtime#Pod and api#Pod
type PodPair struct {
	// APIPod is the api.Pod
	APIPod *api.Pod
	// RunningPod is the pod defined defined in pkg/kubelet/container/runtime#Pod
	RunningPod *Pod
}

// ContainerID is a type that identifies a container.
type ContainerID struct {
	// The type of the container runtime. e.g. 'docker', 'rkt'.
	Type string
	// The identification of the container, this is comsumable by
	// the underlying container runtime. (Note that the container
	// runtime interface still takes the whole struct as input).
	ID string
}

func BuildContainerID(typ, ID string) ContainerID {
	return ContainerID{Type: typ, ID: ID}
}

// Convenience method for creating a ContainerID from an ID string.
func ParseContainerID(containerID string) ContainerID {
	var id ContainerID
	if err := id.ParseString(containerID); err != nil {
		glog.Error(err)
	}
	return id
}

func (c *ContainerID) ParseString(data string) error {
	// Trim the quotes and split the type and ID.
	parts := strings.Split(strings.Trim(data, "\""), "://")
	if len(parts) != 2 {
		return fmt.Errorf("invalid container ID: %q", data)
	}
	c.Type, c.ID = parts[0], parts[1]
	return nil
}

func (c *ContainerID) String() string {
	return fmt.Sprintf("%s://%s", c.Type, c.ID)
}

func (c *ContainerID) IsEmpty() bool {
	return *c == ContainerID{}
}

func (c *ContainerID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", c.String())), nil
}

func (c *ContainerID) UnmarshalJSON(data []byte) error {
	return c.ParseString(string(data))
}

// DockerID is an ID of docker container. It is a type to make it clear when we're working with docker container Ids
type DockerID string

func (id DockerID) ContainerID() ContainerID {
	return ContainerID{
		Type: "docker",
		ID:   string(id),
	}
}

type ContainerState string

const (
	ContainerStateCreated ContainerState = "created"
	ContainerStateRunning ContainerState = "running"
	ContainerStateExited  ContainerState = "exited"
	// This unknown encompasses all the states that we currently don't care.
	ContainerStateUnknown ContainerState = "unknown"
)

// Container provides the runtime information for a container, such as ID, hash,
// state of the container.
type Container struct {
	// The ID of the container, used by the container runtime to identify
	// a container.
	ID ContainerID
	// The name of the container, which should be the same as specified by
	// api.Container.
	Name string
	// The image name of the container, this also includes the tag of the image,
	// the expected form is "NAME:TAG".
	Image string
	// The id of the image used by the container.
	ImageID string
	// Hash of the container, used for comparison. Optional for containers
	// not managed by kubelet.
	Hash uint64
	// State is the state of the container.
	State ContainerState
}

// PodStatus represents the status of the pod and its containers.
// api.PodStatus can be derived from examining PodStatus and api.Pod.
type PodStatus struct {
	// ID of the pod.
	ID types.UID
	// Name of the pod.
	Name string
	// Namspace of the pod.
	Namespace string
	// IP of the pod.
	IP string
	// Status of containers in the pod.
	ContainerStatuses []*ContainerStatus
	// Status of the pod sandbox.
	// Only for kuberuntime now, other runtime may keep it nil.
	SandboxStatuses []*runtimeApi.PodSandboxStatus
}

// ContainerStatus represents the status of a container.
type ContainerStatus struct {
	// ID of the container.
	ID ContainerID
	// Name of the container.
	Name string
	// Status of the container.
	State ContainerState
	// Creation time of the container.
	CreatedAt time.Time
	// Start time of the container.
	StartedAt time.Time
	// Finish time of the container.
	FinishedAt time.Time
	// Exit code of the container.
	ExitCode int
	// Name of the image, this also includes the tag of the image,
	// the expected form is "NAME:TAG".
	Image string
	// ID of the image.
	ImageID string
	// Hash of the container, used for comparison.
	Hash uint64
	// Number of times that the container has been restarted.
	RestartCount int
	// A string explains why container is in such a status.
	Reason string
	// Message written by the container before exiting (stored in
	// TerminationMessagePath).
	Message string
}

// FindContainerStatusByName returns container status in the pod status with the given name.
// When there are multiple containers' statuses with the same name, the first match will be returned.
func (podStatus *PodStatus) FindContainerStatusByName(containerName string) *ContainerStatus {
	for _, containerStatus := range podStatus.ContainerStatuses {
		if containerStatus.Name == containerName {
			return containerStatus
		}
	}
	return nil
}

// Get container status of all the running containers in a pod
func (podStatus *PodStatus) GetRunningContainerStatuses() []*ContainerStatus {
	runnningContainerStatues := []*ContainerStatus{}
	for _, containerStatus := range podStatus.ContainerStatuses {
		if containerStatus.State == ContainerStateRunning {
			runnningContainerStatues = append(runnningContainerStatues, containerStatus)
		}
	}
	return runnningContainerStatues
}

// Basic information about a container image.
type Image struct {
	// ID of the image.
	ID string
	// Other names by which this image is known.
	RepoTags []string
	// Digests by which this image is known.
	RepoDigests []string
	// The size of the image in bytes.
	Size int64
}

type EnvVar struct {
	Name  string
	Value string
}

type Mount struct {
	// Name of the volume mount.
	// TODO(yifan): Remove this field, as this is not representing the unique name of the mount,
	// but the volume name only.
	Name string
	// Path of the mount within the container.
	ContainerPath string
	// Path of the mount on the host.
	HostPath string
	// Whether the mount is read-only.
	ReadOnly bool
	// Whether the mount needs SELinux relabeling
	SELinuxRelabel bool
}

type PortMapping struct {
	// Name of the port mapping
	Name string
	// Protocol of the port mapping.
	Protocol api.Protocol
	// The port number within the container.
	ContainerPort int
	// The port number on the host.
	HostPort int
	// The host IP.
	HostIP string
}

type DeviceInfo struct {
	// Path on host for mapping
	PathOnHost string
	// Path in Container to map
	PathInContainer string
	// Cgroup permissions
	Permissions string
}

// RunContainerOptions specify the options which are necessary for running containers
type RunContainerOptions struct {
	// The environment variables list.
	Envs []EnvVar
	// The mounts for the containers.
	Mounts []Mount
	// The host devices mapped into the containers.
	Devices []DeviceInfo
	// The port mappings for the containers.
	PortMappings []PortMapping
	// If the container has specified the TerminationMessagePath, then
	// this directory will be used to create and mount the log file to
	// container.TerminationMessagePath
	PodContainerDir string
	// The list of DNS servers for the container to use.
	DNS []string
	// The list of DNS search domains.
	DNSSearch []string
	// The parent cgroup to pass to Docker
	CgroupParent string
	// The type of container rootfs
	ReadOnly bool
	// hostname for pod containers
	Hostname string
	// EnableHostUserNamespace sets userns=host when users request host namespaces (pid, ipc, net),
	// are using non-namespaced capabilities (mknod, sys_time, sys_module), the pod contains a privileged container,
	// or using host path volumes.
	// This should only be enabled when the container runtime is performing user remapping AND if the
	// experimental behavior is desired.
	EnableHostUserNamespace bool
}

// VolumeInfo contains information about the volume.
type VolumeInfo struct {
	// Mounter is the volume's mounter
	Mounter volume.Mounter
	// SELinuxLabeled indicates whether this volume has had the
	// pod's SELinux label applied to it or not
	SELinuxLabeled bool
}

type VolumeMap map[string]VolumeInfo

// RuntimeConditionType is the types of required runtime conditions.
type RuntimeConditionType string

const (
	// RuntimeReady means the runtime is up and ready to accept basic containers.
	RuntimeReady RuntimeConditionType = "RuntimeReady"
	// NetworkReady means the runtime network is up and ready to accept containers which require network.
	NetworkReady RuntimeConditionType = "NetworkReady"
)

// RuntimeStatus contains the status of the runtime.
type RuntimeStatus struct {
	// Conditions is an array of current observed runtime conditions.
	Conditions []RuntimeCondition
}

// GetRuntimeCondition gets a specified runtime condition from the runtime status.
func (r *RuntimeStatus) GetRuntimeCondition(t RuntimeConditionType) *RuntimeCondition {
	for i := range r.Conditions {
		c := &r.Conditions[i]
		if c.Type == t {
			return c
		}
	}
	return nil
}

// String formats the runtime status into human readable string.
func (s *RuntimeStatus) String() string {
	var ss []string
	for _, c := range s.Conditions {
		ss = append(ss, c.String())
	}
	return fmt.Sprintf("Runtime Conditions: %s", strings.Join(ss, ", "))
}

// RuntimeCondition contains condition information for the runtime.
type RuntimeCondition struct {
	// Type of runtime condition.
	Type RuntimeConditionType
	// Status of the condition, one of true/false.
	Status bool
	// Reason is brief reason for the condition's last transition.
	Reason string
	// Message is human readable message indicating details about last transition.
	Message string
}

// String formats the runtime condition into human readable string.
func (c *RuntimeCondition) String() string {
	return fmt.Sprintf("%s=%t reason:%s message:%s", c.Type, c.Status, c.Reason, c.Message)
}

type Pods []*Pod

// FindPodByID finds and returns a pod in the pod list by UID. It will return an empty pod
// if not found.
func (p Pods) FindPodByID(podUID types.UID) Pod {
	for i := range p {
		if p[i].ID == podUID {
			return *p[i]
		}
	}
	return Pod{}
}

// FindPodByFullName finds and returns a pod in the pod list by the full name.
// It will return an empty pod if not found.
func (p Pods) FindPodByFullName(podFullName string) Pod {
	for i := range p {
		if BuildPodFullName(p[i].Name, p[i].Namespace) == podFullName {
			return *p[i]
		}
	}
	return Pod{}
}

// FindPod combines FindPodByID and FindPodByFullName, it finds and returns a pod in the
// pod list either by the full name or the pod ID. It will return an empty pod
// if not found.
func (p Pods) FindPod(podFullName string, podUID types.UID) Pod {
	if len(podFullName) > 0 {
		return p.FindPodByFullName(podFullName)
	}
	return p.FindPodByID(podUID)
}

// FindContainerByName returns a container in the pod with the given name.
// When there are multiple containers with the same name, the first match will
// be returned.
func (p *Pod) FindContainerByName(containerName string) *Container {
	for _, c := range p.Containers {
		if c.Name == containerName {
			return c
		}
	}
	return nil
}

func (p *Pod) FindContainerByID(id ContainerID) *Container {
	for _, c := range p.Containers {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (p *Pod) FindSandboxByID(id ContainerID) *Container {
	for _, c := range p.Sandboxes {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// ToAPIPod converts Pod to api.Pod. Note that if a field in api.Pod has no
// corresponding field in Pod, the field would not be populated.
func (p *Pod) ToAPIPod() *api.Pod {
	var pod api.Pod
	pod.UID = p.ID
	pod.Name = p.Name
	pod.Namespace = p.Namespace

	for _, c := range p.Containers {
		var container api.Container
		container.Name = c.Name
		container.Image = c.Image
		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}
	return &pod
}

// IsEmpty returns true if the pod is empty.
func (p *Pod) IsEmpty() bool {
	return reflect.DeepEqual(p, &Pod{})
}

// GetPodFullName returns a name that uniquely identifies a pod.
func GetPodFullName(pod *api.Pod) string {
	// Use underscore as the delimiter because it is not allowed in pod name
	// (DNS subdomain format), while allowed in the container name format.
	return pod.Name + "_" + pod.Namespace
}

// Build the pod full name from pod name and namespace.
func BuildPodFullName(name, namespace string) string {
	return name + "_" + namespace
}

// Parse the pod full name.
func ParsePodFullName(podFullName string) (string, string, error) {
	parts := strings.Split(podFullName, "_")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("failed to parse the pod full name %q", podFullName)
	}
	return parts[0], parts[1], nil
}

// Option is a functional option type for Runtime, useful for
// completely optional settings.
type Option func(Runtime)

// Sort the container statuses by creation time.
type SortContainerStatusesByCreationTime []*ContainerStatus

func (s SortContainerStatusesByCreationTime) Len() int      { return len(s) }
func (s SortContainerStatusesByCreationTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortContainerStatusesByCreationTime) Less(i, j int) bool {
	return s[i].CreatedAt.Before(s[j].CreatedAt)
}
