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

package dockertools

import (
	cadvisorapi "github.com/openshift/github.com/google/cadvisor/info/v1"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/record"
	kubecontainer "github.com/openshift/kubernetes/pkg/kubelet/container"
	"github.com/openshift/kubernetes/pkg/kubelet/network"
	proberesults "github.com/openshift/kubernetes/pkg/kubelet/prober/results"
	kubetypes "github.com/openshift/kubernetes/pkg/kubelet/types"
	"github.com/openshift/kubernetes/pkg/kubelet/util/cache"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/flowcontrol"
	"github.com/openshift/kubernetes/pkg/util/oom"
	"github.com/openshift/kubernetes/pkg/util/procfs"
)

func NewFakeDockerManager(
	client DockerInterface,
	recorder record.EventRecorder,
	livenessManager proberesults.Manager,
	containerRefManager *kubecontainer.RefManager,
	machineInfo *cadvisorapi.MachineInfo,
	podInfraContainerImage string,
	qps float32,
	burst int,
	containerLogsDir string,
	osInterface kubecontainer.OSInterface,
	networkPlugin network.NetworkPlugin,
	runtimeHelper kubecontainer.RuntimeHelper,
	httpClient kubetypes.HttpGetter, imageBackOff *flowcontrol.Backoff) *DockerManager {

	fakeOOMAdjuster := oom.NewFakeOOMAdjuster()
	fakeProcFs := procfs.NewFakeProcFS()
	fakePodGetter := &fakePodGetter{}
	dm := NewDockerManager(client, recorder, livenessManager, containerRefManager, fakePodGetter, machineInfo, podInfraContainerImage, qps,
		burst, containerLogsDir, osInterface, networkPlugin, runtimeHelper, httpClient, &NativeExecHandler{},
		fakeOOMAdjuster, fakeProcFs, false, imageBackOff, false, false, true, "/var/lib/kubelet/seccomp")
	dm.dockerPuller = &FakeDockerPuller{}

	// ttl of version cache is set to 0 so we always call version api directly in tests.
	dm.versionCache = cache.NewObjectCache(
		func() (interface{}, error) {
			return dm.getVersionInfo()
		},
		0,
	)
	return dm
}

type fakePodGetter struct {
	pods map[types.UID]*api.Pod
}

func newFakePodGetter() *fakePodGetter {
	return &fakePodGetter{make(map[types.UID]*api.Pod)}
}

func (f *fakePodGetter) GetPodByUID(uid types.UID) (*api.Pod, bool) {
	pod, found := f.pods[uid]
	return pod, found
}
