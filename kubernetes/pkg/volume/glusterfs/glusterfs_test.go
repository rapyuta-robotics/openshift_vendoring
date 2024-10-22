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

package glusterfs

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	gapi "github.com/openshift/github.com/heketi/heketi/pkg/glusterfs/api"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"github.com/openshift/kubernetes/pkg/client/testing/core"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/types"
	"github.com/openshift/kubernetes/pkg/util/exec"
	"github.com/openshift/kubernetes/pkg/util/mount"
	utiltesting "github.com/openshift/kubernetes/pkg/util/testing"
	"github.com/openshift/kubernetes/pkg/volume"
	volumetest "github.com/openshift/kubernetes/pkg/volume/testing"
)

func TestCanSupport(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("glusterfs_test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))
	plug, err := plugMgr.FindPluginByName("kubernetes.io/glusterfs")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	if plug.GetPluginName() != "kubernetes.io/glusterfs" {
		t.Errorf("Wrong name: %s", plug.GetPluginName())
	}
	if plug.CanSupport(&volume.Spec{PersistentVolume: &api.PersistentVolume{Spec: api.PersistentVolumeSpec{PersistentVolumeSource: api.PersistentVolumeSource{}}}}) {
		t.Errorf("Expected false")
	}
	if plug.CanSupport(&volume.Spec{Volume: &api.Volume{VolumeSource: api.VolumeSource{}}}) {
		t.Errorf("Expected false")
	}
}

func TestGetAccessModes(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("glusterfs_test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))

	plug, err := plugMgr.FindPersistentPluginByName("kubernetes.io/glusterfs")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	if !contains(plug.GetAccessModes(), api.ReadWriteOnce) || !contains(plug.GetAccessModes(), api.ReadOnlyMany) || !contains(plug.GetAccessModes(), api.ReadWriteMany) {
		t.Errorf("Expected three AccessModeTypes:  %s, %s, and %s", api.ReadWriteOnce, api.ReadOnlyMany, api.ReadWriteMany)
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

func doTestPlugin(t *testing.T, spec *volume.Spec) {
	tmpDir, err := utiltesting.MkTmpdir("glusterfs_test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, nil, nil))
	plug, err := plugMgr.FindPluginByName("kubernetes.io/glusterfs")
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	ep := &api.Endpoints{ObjectMeta: api.ObjectMeta{Name: "foo"}, Subsets: []api.EndpointSubset{{
		Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}}}}}
	var fcmd exec.FakeCmd
	fcmd = exec.FakeCmd{
		CombinedOutputScript: []exec.FakeCombinedOutputAction{
			// mount
			func() ([]byte, error) {
				return []byte{}, nil
			},
		},
	}
	fake := exec.FakeExec{
		CommandScript: []exec.FakeCommandAction{
			func(cmd string, args ...string) exec.Cmd { return exec.InitFakeCmd(&fcmd, cmd, args...) },
		},
	}
	pod := &api.Pod{ObjectMeta: api.ObjectMeta{UID: types.UID("poduid")}}
	mounter, err := plug.(*glusterfsPlugin).newMounterInternal(spec, ep, pod, &mount.FakeMounter{}, &fake)
	volumePath := mounter.GetPath()
	if err != nil {
		t.Errorf("Failed to make a new Mounter: %v", err)
	}
	if mounter == nil {
		t.Error("Got a nil Mounter")
	}
	path := mounter.GetPath()
	expectedPath := fmt.Sprintf("%s/pods/poduid/volumes/kubernetes.io~glusterfs/vol1", tmpDir)
	if path != expectedPath {
		t.Errorf("Unexpected path, expected %q, got: %q", expectedPath, path)
	}
	if err := mounter.SetUp(nil); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(volumePath); err != nil {
		if os.IsNotExist(err) {
			t.Errorf("SetUp() failed, volume path not created: %s", volumePath)
		} else {
			t.Errorf("SetUp() failed: %v", err)
		}
	}
	unmounter, err := plug.(*glusterfsPlugin).newUnmounterInternal("vol1", types.UID("poduid"), &mount.FakeMounter{})
	if err != nil {
		t.Errorf("Failed to make a new Unmounter: %v", err)
	}
	if unmounter == nil {
		t.Error("Got a nil Unmounter")
	}
	if err := unmounter.TearDown(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(volumePath); err == nil {
		t.Errorf("TearDown() failed, volume path still exists: %s", volumePath)
	} else if !os.IsNotExist(err) {
		t.Errorf("SetUp() failed: %v", err)
	}
}

func TestPluginVolume(t *testing.T) {
	vol := &api.Volume{
		Name:         "vol1",
		VolumeSource: api.VolumeSource{Glusterfs: &api.GlusterfsVolumeSource{EndpointsName: "ep", Path: "vol", ReadOnly: false}},
	}
	doTestPlugin(t, volume.NewSpecFromVolume(vol))
}

func TestPluginPersistentVolume(t *testing.T) {
	vol := &api.PersistentVolume{
		ObjectMeta: api.ObjectMeta{
			Name: "vol1",
		},
		Spec: api.PersistentVolumeSpec{
			PersistentVolumeSource: api.PersistentVolumeSource{
				Glusterfs: &api.GlusterfsVolumeSource{EndpointsName: "ep", Path: "vol", ReadOnly: false},
			},
		},
	}

	doTestPlugin(t, volume.NewSpecFromPersistentVolume(vol, false))
}

func TestPersistentClaimReadOnlyFlag(t *testing.T) {
	tmpDir, err := utiltesting.MkTmpdir("glusterfs_test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pv := &api.PersistentVolume{
		ObjectMeta: api.ObjectMeta{
			Name: "pvA",
		},
		Spec: api.PersistentVolumeSpec{
			PersistentVolumeSource: api.PersistentVolumeSource{
				Glusterfs: &api.GlusterfsVolumeSource{EndpointsName: "ep", Path: "vol", ReadOnly: false},
			},
			ClaimRef: &api.ObjectReference{
				Name: "claimA",
			},
		},
	}

	claim := &api.PersistentVolumeClaim{
		ObjectMeta: api.ObjectMeta{
			Name:      "claimA",
			Namespace: "nsA",
		},
		Spec: api.PersistentVolumeClaimSpec{
			VolumeName: "pvA",
		},
		Status: api.PersistentVolumeClaimStatus{
			Phase: api.ClaimBound,
		},
	}

	ep := &api.Endpoints{
		ObjectMeta: api.ObjectMeta{
			Namespace: "nsA",
			Name:      "ep",
		},
		Subsets: []api.EndpointSubset{{
			Addresses: []api.EndpointAddress{{IP: "127.0.0.1"}},
			Ports:     []api.EndpointPort{{Name: "foo", Port: 80, Protocol: api.ProtocolTCP}},
		}},
	}

	client := fake.NewSimpleClientset(pv, claim, ep)

	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volumetest.NewFakeVolumeHost(tmpDir, client, nil))
	plug, _ := plugMgr.FindPluginByName(glusterfsPluginName)

	// readOnly bool is supplied by persistent-claim volume source when its mounter creates other volumes
	spec := volume.NewSpecFromPersistentVolume(pv, true)
	pod := &api.Pod{ObjectMeta: api.ObjectMeta{Namespace: "nsA", UID: types.UID("poduid")}}
	mounter, _ := plug.NewMounter(spec, pod, volume.VolumeOptions{})

	if !mounter.GetAttributes().ReadOnly {
		t.Errorf("Expected true for mounter.IsReadOnly")
	}
}

func TestParseClassParameters(t *testing.T) {
	secret := api.Secret{
		Type: "kubernetes.io/glusterfs",
		Data: map[string][]byte{
			"data": []byte("mypassword"),
		},
	}
	tests := []struct {
		name         string
		parameters   map[string]string
		secret       *api.Secret
		expectError  bool
		expectConfig *provisioningConfig
	}{
		{
			"password",
			map[string]string{
				"resturl":     "https://localhost:8080",
				"restuser":    "admin",
				"restuserkey": "password",
			},
			nil,   // secret
			false, // expect error
			&provisioningConfig{
				url:         "https://localhost:8080",
				user:        "admin",
				userKey:     "password",
				secretValue: "password",
				gidMin:      2000,
				gidMax:      2147483647,
				volumeType:  gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},
		{
			"secret",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restuser":        "admin",
				"secretname":      "mysecret",
				"secretnamespace": "default",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:             "https://localhost:8080",
				user:            "admin",
				secretName:      "mysecret",
				secretNamespace: "default",
				secretValue:     "mypassword",
				gidMin:          2000,
				gidMax:          2147483647,
				volumeType:      gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},
		{
			"no authentication",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     2000,
				gidMax:     2147483647,
				volumeType: gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},
		{
			"missing secret",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"secretname":      "mysecret",
				"secretnamespace": "default",
			},
			nil,  // secret
			true, // expect error
			nil,
		},
		{
			"secret with no namespace",
			map[string]string{
				"resturl":    "https://localhost:8080",
				"secretname": "mysecret",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"missing url",
			map[string]string{
				"restuser":    "admin",
				"restuserkey": "password",
			},
			nil,  // secret
			true, // expect error
			nil,
		},
		{
			"unknown parameter",
			map[string]string{
				"unknown":     "yes",
				"resturl":     "https://localhost:8080",
				"restuser":    "admin",
				"restuserkey": "password",
			},
			nil,  // secret
			true, // expect error
			nil,
		},
		{
			"invalid gidMin #1",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "0",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMin #2",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "1999",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMin #3",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "1999",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMax #1",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMax":          "0",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMax #2",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMax":          "1999",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMax #3",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMax":          "1999",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid gidMin:gidMax",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "5001",
				"gidMax":          "5000",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"valid gidMin",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "4000",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     4000,
				gidMax:     2147483647,
				volumeType: gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},
		{
			"valid gidMax",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMax":          "5000",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     2000,
				gidMax:     5000,
				volumeType: gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},
		{
			"valid gidMin:gidMax",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "4000",
				"gidMax":          "5000",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     4000,
				gidMax:     5000,
				volumeType: gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 3}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},

		{
			"valid volumetype: replicate",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "4000",
				"gidMax":          "5000",
				"volumetype":      "replicate:4",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     4000,
				gidMax:     5000,
				volumeType: gapi.VolumeDurabilityInfo{Type: "replicate", Replicate: gapi.ReplicaDurability{Replica: 4}, Disperse: gapi.DisperseDurability{Data: 0, Redundancy: 0}},
			},
		},

		{
			"valid volumetype: disperse",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"gidMin":          "4000",
				"gidMax":          "5000",
				"volumetype":      "disperse:4:2",
			},
			&secret,
			false, // expect error
			&provisioningConfig{
				url:        "https://localhost:8080",
				gidMin:     4000,
				gidMax:     5000,
				volumeType: gapi.VolumeDurabilityInfo{Type: "disperse", Replicate: gapi.ReplicaDurability{Replica: 0}, Disperse: gapi.DisperseDurability{Data: 4, Redundancy: 2}},
			},
		},
		{
			"invalid volumetype (disperse) parameter",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"volumetype":      "disperse:4:asd",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid volumetype (replicate) parameter",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"volumetype":      "replicate:asd",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid volumetype: unknown volumetype",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"volumetype":      "dispersereplicate:4:2",
			},
			&secret,
			true, // expect error
			nil,
		},
		{
			"invalid volumetype : negative value",
			map[string]string{
				"resturl":         "https://localhost:8080",
				"restauthenabled": "false",
				"volumetype":      "replicate:-1000",
			},
			&secret,
			true, // expect error
			nil,
		},
	}

	for _, test := range tests {

		client := &fake.Clientset{}
		client.AddReactor("get", "secrets", func(action core.Action) (handled bool, ret runtime.Object, err error) {
			if test.secret != nil {
				return true, test.secret, nil
			}
			return true, nil, fmt.Errorf("Test %s did not set a secret", test.name)
		})

		cfg, err := parseClassParameters(test.parameters, client)

		if err != nil && !test.expectError {
			t.Errorf("Test %s got unexpected error %v", test.name, err)
		}
		if err == nil && test.expectError {
			t.Errorf("test %s expected error and got none", test.name)
		}
		if test.expectConfig != nil {
			if !reflect.DeepEqual(cfg, test.expectConfig) {
				t.Errorf("Test %s returned unexpected data, expected: %+v, got: %+v", test.name, test.expectConfig, cfg)
			}
		}
	}
}
