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
	"os"
	"path/filepath"
	"testing"

	"github.com/openshift/github.com/stretchr/testify/assert"
	"github.com/openshift/kubernetes/pkg/api"
	runtimeApi "github.com/openshift/kubernetes/pkg/kubelet/api/v1alpha1/runtime"
	containertest "github.com/openshift/kubernetes/pkg/kubelet/container/testing"
)

// TestCreatePodSandbox tests creating sandbox and its corresponding pod log directory.
func TestCreatePodSandbox(t *testing.T) {
	fakeRuntime, _, m, err := createTestRuntimeManager()
	pod := &api.Pod{
		ObjectMeta: api.ObjectMeta{
			UID:       "12345678",
			Name:      "bar",
			Namespace: "new",
		},
		Spec: api.PodSpec{
			Containers: []api.Container{
				{
					Name:            "foo",
					Image:           "busybox",
					ImagePullPolicy: api.PullIfNotPresent,
				},
			},
		},
	}

	fakeOS := m.osInterface.(*containertest.FakeOS)
	fakeOS.MkdirAllFn = func(path string, perm os.FileMode) error {
		// Check pod logs root directory is created.
		assert.Equal(t, filepath.Join(podLogsRootDirectory, "12345678"), path)
		assert.Equal(t, os.FileMode(0755), perm)
		return nil
	}
	id, _, err := m.createPodSandbox(pod, 1)
	assert.NoError(t, err)
	fakeRuntime.AssertCalls([]string{"RunPodSandbox"})
	sandboxes, err := fakeRuntime.ListPodSandbox(&runtimeApi.PodSandboxFilter{Id: &id})
	assert.NoError(t, err)
	assert.Equal(t, len(sandboxes), 1)
	// TODO Check pod sandbox configuration
}
