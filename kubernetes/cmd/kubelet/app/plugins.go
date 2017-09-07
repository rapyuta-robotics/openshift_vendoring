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

package app

// This file exists to force the desired plugin implementations to be linked.
import (
	// Credential providers
	_ "github.com/openshift/kubernetes/pkg/credentialprovider/aws"
	_ "github.com/openshift/kubernetes/pkg/credentialprovider/azure"
	_ "github.com/openshift/kubernetes/pkg/credentialprovider/gcp"
	// Network plugins
	"github.com/openshift/kubernetes/pkg/kubelet/network"
	"github.com/openshift/kubernetes/pkg/kubelet/network/cni"
	"github.com/openshift/kubernetes/pkg/kubelet/network/exec"
	"github.com/openshift/kubernetes/pkg/kubelet/network/kubenet"
	// Volume plugins
	"github.com/openshift/kubernetes/pkg/volume"
	"github.com/openshift/kubernetes/pkg/volume/aws_ebs"
	"github.com/openshift/kubernetes/pkg/volume/azure_dd"
	"github.com/openshift/kubernetes/pkg/volume/azure_file"
	"github.com/openshift/kubernetes/pkg/volume/cephfs"
	"github.com/openshift/kubernetes/pkg/volume/cinder"
	"github.com/openshift/kubernetes/pkg/volume/configmap"
	"github.com/openshift/kubernetes/pkg/volume/downwardapi"
	"github.com/openshift/kubernetes/pkg/volume/empty_dir"
	"github.com/openshift/kubernetes/pkg/volume/fc"
	"github.com/openshift/kubernetes/pkg/volume/flexvolume"
	"github.com/openshift/kubernetes/pkg/volume/flocker"
	"github.com/openshift/kubernetes/pkg/volume/gce_pd"
	"github.com/openshift/kubernetes/pkg/volume/git_repo"
	"github.com/openshift/kubernetes/pkg/volume/glusterfs"
	"github.com/openshift/kubernetes/pkg/volume/host_path"
	"github.com/openshift/kubernetes/pkg/volume/iscsi"
	"github.com/openshift/kubernetes/pkg/volume/nfs"
	"github.com/openshift/kubernetes/pkg/volume/photon_pd"
	"github.com/openshift/kubernetes/pkg/volume/quobyte"
	"github.com/openshift/kubernetes/pkg/volume/rbd"
	"github.com/openshift/kubernetes/pkg/volume/secret"
	"github.com/openshift/kubernetes/pkg/volume/vsphere_volume"
	// Cloud providers
	_ "github.com/openshift/kubernetes/pkg/cloudprovider/providers"
)

// ProbeVolumePlugins collects all volume plugins into an easy to use list.
// PluginDir specifies the directory to search for additional third party
// volume plugins.
func ProbeVolumePlugins(pluginDir string) []volume.VolumePlugin {
	allPlugins := []volume.VolumePlugin{}

	// The list of plugins to probe is decided by the kubelet binary, not
	// by dynamic linking or other "magic".  Plugins will be analyzed and
	// initialized later.
	//
	// Kubelet does not currently need to configure volume plugins.
	// If/when it does, see kube-controller-manager/app/plugins.go for example of using volume.VolumeConfig
	allPlugins = append(allPlugins, aws_ebs.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, empty_dir.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, gce_pd.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, git_repo.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, host_path.ProbeVolumePlugins(volume.VolumeConfig{})...)
	allPlugins = append(allPlugins, nfs.ProbeVolumePlugins(volume.VolumeConfig{})...)
	allPlugins = append(allPlugins, secret.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, iscsi.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, glusterfs.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, rbd.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, cinder.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, quobyte.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, cephfs.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, downwardapi.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, fc.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, flocker.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, flexvolume.ProbeVolumePlugins(pluginDir)...)
	allPlugins = append(allPlugins, azure_file.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, configmap.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, vsphere_volume.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, azure_dd.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, photon_pd.ProbeVolumePlugins()...)
	return allPlugins
}

// ProbeNetworkPlugins collects all compiled-in plugins
func ProbeNetworkPlugins(pluginDir, cniConfDir, cniBinDir string) []network.NetworkPlugin {
	allPlugins := []network.NetworkPlugin{}

	// for backwards-compat, allow pluginDir as a source of CNI config files
	if cniConfDir == "" {
		cniConfDir = pluginDir
	}
	// for each existing plugin, add to the list
	allPlugins = append(allPlugins, exec.ProbeNetworkPlugins(pluginDir)...)
	allPlugins = append(allPlugins, cni.ProbeNetworkPlugins(cniConfDir, cniBinDir)...)
	allPlugins = append(allPlugins, kubenet.NewPlugin(pluginDir))

	return allPlugins
}
