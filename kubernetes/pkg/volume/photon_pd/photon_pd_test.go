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

package photon_pd

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/mount"
	utiltesting "github.com/openshift/kubernetes/pkg/util/testing"
	"github.com/openshift/kubernetes/pkg/volume"
	volumetest "github.com/openshift/kubernetes/pkg/volume/testing"
)

func TestCanSupport(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("photonpdTest")
	if err != nil {
		t.Fatalf("can't make a temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))

	plug, err := plugMgr.FindPluginByName("kubernetes.io/photon-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	if plug.GetPluginName() != "kubernetes.io/photon-pd" {
		t.Errorf("Wrong name: %s", plug.GetPluginName())
	}
	if !plug.CanSupport(&volume.Spec{Volume: &api.Volume{VolumeSource: api.VolumeSource{PhotonPersistentDisk: &api.PhotonPersistentDiskVolumeSource{}}}}) {
		t.Errorf("Expected true")
	}
	if !plug.CanSupport(&volume.Spec{PersistentVolume: &api.PersistentVolume{Spec: api.PersistentVolumeSpec{PersistentVolumeSource: api.PersistentVolumeSource{PhotonPersistentDisk: &api.PhotonPersistentDiskVolumeSource{}}}}}) {
		t.Errorf("Expected true")
	}
}

func TestGetAccessModes(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("photonpdTest")
	if err != nil {
		t.Fatalf("can't make a temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))

	plug, err := plugMgr.FindPersistentPluginByName("kubernetes.io/photon-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}

	if !contains(plug.GetAccessModes(), api.ReadWriteOnce) {
		t.Errorf("Expected to support AccessModeTypes:  %s", api.ReadWriteOnce)
	}
	if contains(plug.GetAccessModes(), api.ReadOnlyMany) {
		t.Errorf("Expected not to support AccessModeTypes:  %s", api.ReadOnlyMany)
	}
}

func contains(modes []api.PersistentVolumeAccessMode, mode api.PersistentVolumeAccessMode) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}

type fakePDManager struct {
}

func (fake *fakePDManager) CreateVolume(c *photonPersistentDiskProvisioner) (pdID string, volumeSizeGB int, err error) {
	return "test-photon-pd-id", 10, nil
}

func (fake *fakePDManager) DeleteVolume(cd *photonPersistentDiskDeleter) error {
	if cd.pdID != "test-photon-pd-id" {
		return fmt.Errorf("Deleter got unexpected volume name: %s", cd.pdID)
	}
	return nil
}

func TestPlugin(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("photonpdTest")
	if err != nil {
		t.Fatalf("can't make a temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))

	plug, err := plugMgr.FindPluginByName("kubernetes.io/photon-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	spec := &api.Volume{
		Name: "vol1",
		VolumeSource: api.VolumeSource{
			PhotonPersistentDisk: &api.PhotonPersistentDiskVolumeSource{
				PdID:   "pdid",
				FSType: "ext4",
			},
		},
	}
	fakeManager := &fakePDManager{}
	fakeMounter := &mount.FakeMounter{}
	mounter, err := plug.(*photonPersistentDiskPlugin).newMounterInternal(volume.NewSpecFromVolume(spec), types.UID("poduid"), fakeManager, fakeMounter)
	if err != nil {
		t.Errorf("Failed to make a new Mounter: %v", err)
	}
	if mounter == nil {
		t.Errorf("Got a nil Mounter")
	}

	volPath := path.Join(tmpDir, "pods/poduid/volumes/kubernetes.io~photon-pd/vol1")
	path := mounter.GetPath()
	if path != volPath {
		t.Errorf("Got unexpected path: %s", path)
	}

	if err := mounter.SetUp(nil); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("SetUp() failed, volume path not created: %s", path)
		} else {
			t.Errorf("SetUp() failed: %v", err)
		}
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("SetUp() failed, volume path not created: %s", path)
		} else {
			t.Errorf("SetUp() failed: %v", err)
		}
	}

	fakeManager = &fakePDManager{}
	unmounter, err := plug.(*photonPersistentDiskPlugin).newUnmounterInternal("vol1", types.UID("poduid"), fakeManager, fakeMounter)
	if err != nil {
		t.Errorf("Failed to make a new Unmounter: %v", err)
	}
	if unmounter == nil {
		t.Errorf("Got a nil Unmounter")
	}

	if err := unmounter.TearDown(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Errorf("TearDown() failed, volume path still exists: %s", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("SetUp() failed: %v", err)
	}

	// Test Provisioner
	options := volume.VolumeOptions{
		PVC: volumetest.CreateTestPVC("10Gi", []api.PersistentVolumeAccessMode{api.ReadWriteOnce}),
		PersistentVolumeReclaimPolicy: api.PersistentVolumeReclaimDelete,
	}
	provisioner, err := plug.(*photonPersistentDiskPlugin).newProvisionerInternal(options, &fakePDManager{})
	persistentSpec, err := provisioner.Provision()
	if err != nil {
		t.Errorf("Provision() failed: %v", err)
	}

	if persistentSpec.Spec.PersistentVolumeSource.PhotonPersistentDisk.PdID != "test-photon-pd-id" {
		t.Errorf("Provision() returned unexpected persistent disk ID: %s", persistentSpec.Spec.PersistentVolumeSource.PhotonPersistentDisk.PdID)
	}
	cap := persistentSpec.Spec.Capacity[api.ResourceStorage]
	size := cap.Value()
	if size != 10*1024*1024*1024 {
		t.Errorf("Provision() returned unexpected volume size: %v", size)
	}

	// Test Deleter
	volSpec := &volume.Spec{
		PersistentVolume: persistentSpec,
	}
	deleter, err := plug.(*photonPersistentDiskPlugin).newDeleterInternal(volSpec, &fakePDManager{})
	err = deleter.Delete()
	if err != nil {
		t.Errorf("Deleter() failed: %v", err)
	}
}

func TestMounterAndUnmounterTypeAssert(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("photonpdTest")
	if err != nil {
		t.Fatalf("can't make a temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))

	plug, err := plugMgr.FindPluginByName("kubernetes.io/photon-pd")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	spec := &api.Volume{
		Name: "vol1",
		VolumeSource: api.VolumeSource{
			PhotonPersistentDisk: &api.PhotonPersistentDiskVolumeSource{
				PdID:   "pdid",
				FSType: "ext4",
			},
		},
	}

	mounter, err := plug.(*photonPersistentDiskPlugin).newMounterInternal(volume.NewSpecFromVolume(spec), types.UID("poduid"), &fakePDManager{}, &mount.FakeMounter{})
	if _, ok := mounter.(volume.Unmounter); ok {
		t.Errorf("Volume Mounter can be type-assert to Unmounter")
	}

	unmounter, err := plug.(*photonPersistentDiskPlugin).newUnmounterInternal("vol1", types.UID("poduid"), &fakePDManager{}, &mount.FakeMounter{})
	if _, ok := unmounter.(volume.Mounter); ok {
		t.Errorf("Volume Unmounter can be type-assert to Mounter")
	}
}
