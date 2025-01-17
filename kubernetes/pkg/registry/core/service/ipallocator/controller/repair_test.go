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

package controller

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/registry/core/service/ipallocator"
	"github.com/openshift/kubernetes/pkg/registry/registrytest"
)

type mockRangeRegistry struct {
	getCalled bool
	item      *api.RangeAllocation
	err       error

	updateCalled bool
	updated      *api.RangeAllocation
	updateErr    error
}

func (r *mockRangeRegistry) Get() (*api.RangeAllocation, error) {
	r.getCalled = true
	return r.item, r.err
}

func (r *mockRangeRegistry) CreateOrUpdate(alloc *api.RangeAllocation) error {
	r.updateCalled = true
	r.updated = alloc
	return r.updateErr
}

func TestRepair(t *testing.T) {
	registry := registrytest.NewServiceRegistry()
	ipregistry := &mockRangeRegistry{
		item: &api.RangeAllocation{Range: "192.168.1.0/24"},
	}
	_, cidr, _ := net.ParseCIDR(ipregistry.item.Range)
	r := NewRepair(0, registry, cidr, ipregistry)

	if err := r.RunOnce(); err != nil {
		t.Fatal(err)
	}
	if !ipregistry.updateCalled || ipregistry.updated == nil || ipregistry.updated.Range != cidr.String() || ipregistry.updated != ipregistry.item {
		t.Errorf("unexpected ipregistry: %#v", ipregistry)
	}

	ipregistry = &mockRangeRegistry{
		item:      &api.RangeAllocation{Range: "192.168.1.0/24"},
		updateErr: fmt.Errorf("test error"),
	}
	r = NewRepair(0, registry, cidr, ipregistry)
	if err := r.RunOnce(); !strings.Contains(err.Error(), ": test error") {
		t.Fatal(err)
	}
}

func TestRepairLeak(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	previous := ipallocator.NewCIDRRange(cidr)
	previous.Allocate(net.ParseIP("192.168.1.10"))

	var dst api.RangeAllocation
	err := previous.Snapshot(&dst)
	if err != nil {
		t.Fatal(err)
	}

	registry := registrytest.NewServiceRegistry()
	ipregistry := &mockRangeRegistry{
		item: &api.RangeAllocation{
			ObjectMeta: api.ObjectMeta{
				ResourceVersion: "1",
			},
			Range: dst.Range,
			Data:  dst.Data,
		},
	}
	r := NewRepair(0, registry, cidr, ipregistry)
	// Run through the "leak detection holdoff" loops.
	for i := 0; i < (numRepairsBeforeLeakCleanup - 1); i++ {
		if err := r.RunOnce(); err != nil {
			t.Fatal(err)
		}
		after, err := ipallocator.NewFromSnapshot(ipregistry.updated)
		if err != nil {
			t.Fatal(err)
		}
		if !after.Has(net.ParseIP("192.168.1.10")) {
			t.Errorf("expected ipallocator to still have leaked IP")
		}
	}
	// Run one more time to actually remove the leak.
	if err := r.RunOnce(); err != nil {
		t.Fatal(err)
	}
	after, err := ipallocator.NewFromSnapshot(ipregistry.updated)
	if err != nil {
		t.Fatal(err)
	}
	if after.Has(net.ParseIP("192.168.1.10")) {
		t.Errorf("expected ipallocator to not have leaked IP")
	}
}

func TestRepairWithExisting(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	previous := ipallocator.NewCIDRRange(cidr)

	var dst api.RangeAllocation
	err := previous.Snapshot(&dst)
	if err != nil {
		t.Fatal(err)
	}

	registry := registrytest.NewServiceRegistry()
	registry.List = api.ServiceList{
		Items: []api.Service{
			{
				Spec: api.ServiceSpec{ClusterIP: "192.168.1.1"},
			},
			{
				Spec: api.ServiceSpec{ClusterIP: "192.168.1.100"},
			},
			{ // outside CIDR, will be dropped
				Spec: api.ServiceSpec{ClusterIP: "192.168.0.1"},
			},
			{ // empty, ignored
				Spec: api.ServiceSpec{ClusterIP: ""},
			},
			{ // duplicate, dropped
				Spec: api.ServiceSpec{ClusterIP: "192.168.1.1"},
			},
			{ // headless
				Spec: api.ServiceSpec{ClusterIP: "None"},
			},
		},
	}

	ipregistry := &mockRangeRegistry{
		item: &api.RangeAllocation{
			ObjectMeta: api.ObjectMeta{
				ResourceVersion: "1",
			},
			Range: dst.Range,
			Data:  dst.Data,
		},
	}
	r := NewRepair(0, registry, cidr, ipregistry)
	if err := r.RunOnce(); err != nil {
		t.Fatal(err)
	}
	after, err := ipallocator.NewFromSnapshot(ipregistry.updated)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Has(net.ParseIP("192.168.1.1")) || !after.Has(net.ParseIP("192.168.1.100")) {
		t.Errorf("unexpected ipallocator state: %#v", after)
	}
	if free := after.Free(); free != 252 {
		t.Errorf("unexpected ipallocator state: %d free", free)
	}
}
