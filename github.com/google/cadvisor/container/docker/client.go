// Copyright 2015 Google Inc. All Rights Reserved.
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

// Handler for /validate content.
// Validates cadvisor dependencies - kernel, os, docker setup.

package docker

import (
	"sync"

	dclient "github.com/openshift/github.com/docker/engine-api/client"
)

var (
	dockerClient     *dclient.Client
	dockerClientErr  error
	dockerClientOnce sync.Once
)

func Client() (*dclient.Client, error) {
	dockerClientOnce.Do(func() {
		dockerClient, dockerClientErr = dclient.NewClient(*ArgDockerEndpoint, "", nil, nil)
	})
	return dockerClient, dockerClientErr
}
