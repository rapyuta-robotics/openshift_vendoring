/*
Copyright 2014 The Kubernetes Authors.

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

package securitycontext

import (
	"fmt"
	"strconv"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/kubelet/leaky"

	dockercontainer "github.com/openshift/github.com/docker/engine-api/types/container"
)

// NewSimpleSecurityContextProvider creates a new SimpleSecurityContextProvider.
func NewSimpleSecurityContextProvider(securityOptSeparator rune) SecurityContextProvider {
	return SimpleSecurityContextProvider{securityOptSeparator}
}

// SimpleSecurityContextProvider is the default implementation of a SecurityContextProvider.
type SimpleSecurityContextProvider struct {
	securityOptSeparator rune
}

// ModifyContainerConfig is called before the Docker createContainer call.
// The security context provider can make changes to the Config with which
// the container is created.
func (p SimpleSecurityContextProvider) ModifyContainerConfig(pod *api.Pod, container *api.Container, config *dockercontainer.Config) {
	effectiveSC := DetermineEffectiveSecurityContext(pod, container)
	if effectiveSC == nil {
		return
	}
	if effectiveSC.RunAsUser != nil {
		config.User = strconv.Itoa(int(*effectiveSC.RunAsUser))
	}
}

// ModifyHostConfig is called before the Docker runContainer call. The
// security context provider can make changes to the HostConfig, affecting
// security options, whether the container is privileged, volume binds, etc.
func (p SimpleSecurityContextProvider) ModifyHostConfig(pod *api.Pod, container *api.Container, hostConfig *dockercontainer.HostConfig, supplementalGids []int64) {
	// Apply supplemental groups
	if container.Name != leaky.PodInfraContainerName {
		// TODO: We skip application of supplemental groups to the
		// infra container to work around a runc issue which
		// requires containers to have the '/etc/group'. For
		// more information see:
		// https://github.com/opencontainers/runc/pull/313
		// This can be removed once the fix makes it into the
		// required version of docker.
		if pod.Spec.SecurityContext != nil {
			for _, group := range pod.Spec.SecurityContext.SupplementalGroups {
				hostConfig.GroupAdd = append(hostConfig.GroupAdd, strconv.Itoa(int(group)))
			}
			if pod.Spec.SecurityContext.FSGroup != nil {
				hostConfig.GroupAdd = append(hostConfig.GroupAdd, strconv.Itoa(int(*pod.Spec.SecurityContext.FSGroup)))
			}
		}

		for _, group := range supplementalGids {
			hostConfig.GroupAdd = append(hostConfig.GroupAdd, strconv.Itoa(int(group)))
		}
	}

	// Apply effective security context for container
	effectiveSC := DetermineEffectiveSecurityContext(pod, container)
	if effectiveSC == nil {
		return
	}

	if effectiveSC.Privileged != nil {
		hostConfig.Privileged = *effectiveSC.Privileged
	}

	if effectiveSC.Capabilities != nil {
		add, drop := MakeCapabilities(effectiveSC.Capabilities.Add, effectiveSC.Capabilities.Drop)
		hostConfig.CapAdd = add
		hostConfig.CapDrop = drop
	}

	if effectiveSC.SELinuxOptions != nil {
		hostConfig.SecurityOpt = ModifySecurityOptions(hostConfig.SecurityOpt, effectiveSC.SELinuxOptions, p.securityOptSeparator)
	}
}

// ModifySecurityOptions adds SELinux options to config using the given
// separator.
func ModifySecurityOptions(config []string, selinuxOpts *api.SELinuxOptions, separator rune) []string {
	// Note, strictly speaking, we are actually mutating the values of these
	// keys, rather than formatting name and value into a string.  Docker re-
	// uses the same option name multiple times (it's just 'label') with
	// different values which are themselves key-value pairs.  For example,
	// the SELinux type is represented by the security opt:
	//
	//   label<separator>type:<selinux_type>
	//
	// In Docker API versions before 1.23, the separator was the `:` rune; in
	// API version 1.23 it changed to the `=` rune.
	config = modifySecurityOption(config, DockerLabelUser(separator), selinuxOpts.User)
	config = modifySecurityOption(config, DockerLabelRole(separator), selinuxOpts.Role)
	config = modifySecurityOption(config, DockerLabelType(separator), selinuxOpts.Type)
	config = modifySecurityOption(config, DockerLabelLevel(separator), selinuxOpts.Level)

	return config
}

// modifySecurityOption adds the security option of name to the config array
// with value in the form of name:value.
func modifySecurityOption(config []string, name, value string) []string {
	if len(value) > 0 {
		config = append(config, fmt.Sprintf("%s:%s", name, value))
	}

	return config
}

// MakeCapabilities creates string slices from Capability slices
func MakeCapabilities(capAdd []api.Capability, capDrop []api.Capability) ([]string, []string) {
	var (
		addCaps  []string
		dropCaps []string
	)
	for _, cap := range capAdd {
		addCaps = append(addCaps, string(cap))
	}
	for _, cap := range capDrop {
		dropCaps = append(dropCaps, string(cap))
	}
	return addCaps, dropCaps
}

func DetermineEffectiveSecurityContext(pod *api.Pod, container *api.Container) *api.SecurityContext {
	effectiveSc := securityContextFromPodSecurityContext(pod)
	containerSc := container.SecurityContext

	if effectiveSc == nil && containerSc == nil {
		return nil
	}
	if effectiveSc != nil && containerSc == nil {
		return effectiveSc
	}
	if effectiveSc == nil && containerSc != nil {
		return containerSc
	}

	if containerSc.SELinuxOptions != nil {
		effectiveSc.SELinuxOptions = new(api.SELinuxOptions)
		*effectiveSc.SELinuxOptions = *containerSc.SELinuxOptions
	}

	if containerSc.Capabilities != nil {
		effectiveSc.Capabilities = new(api.Capabilities)
		*effectiveSc.Capabilities = *containerSc.Capabilities
	}

	if containerSc.Privileged != nil {
		effectiveSc.Privileged = new(bool)
		*effectiveSc.Privileged = *containerSc.Privileged
	}

	if containerSc.RunAsUser != nil {
		effectiveSc.RunAsUser = new(int64)
		*effectiveSc.RunAsUser = *containerSc.RunAsUser
	}

	if containerSc.RunAsNonRoot != nil {
		effectiveSc.RunAsNonRoot = new(bool)
		*effectiveSc.RunAsNonRoot = *containerSc.RunAsNonRoot
	}

	if containerSc.ReadOnlyRootFilesystem != nil {
		effectiveSc.ReadOnlyRootFilesystem = new(bool)
		*effectiveSc.ReadOnlyRootFilesystem = *containerSc.ReadOnlyRootFilesystem
	}

	return effectiveSc
}

func securityContextFromPodSecurityContext(pod *api.Pod) *api.SecurityContext {
	if pod.Spec.SecurityContext == nil {
		return nil
	}

	synthesized := &api.SecurityContext{}

	if pod.Spec.SecurityContext.SELinuxOptions != nil {
		synthesized.SELinuxOptions = &api.SELinuxOptions{}
		*synthesized.SELinuxOptions = *pod.Spec.SecurityContext.SELinuxOptions
	}
	if pod.Spec.SecurityContext.RunAsUser != nil {
		synthesized.RunAsUser = new(int64)
		*synthesized.RunAsUser = *pod.Spec.SecurityContext.RunAsUser
	}

	if pod.Spec.SecurityContext.RunAsNonRoot != nil {
		synthesized.RunAsNonRoot = new(bool)
		*synthesized.RunAsNonRoot = *pod.Spec.SecurityContext.RunAsNonRoot
	}

	return synthesized
}