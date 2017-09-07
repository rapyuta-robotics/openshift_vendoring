package testclient

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/client/testing/core"

	quotaapi "github.com/openshift/origin/pkg/quota/api"
)

type FakeAppliedClusterResourceQuotas struct {
	Fake      *Fake
	Namespace string
}

var appliedClusterResourceQuotasResource = unversioned.GroupVersionResource{Group: "", Version: "", Resource: "appliedclusterresourcequotas"}

func (c *FakeAppliedClusterResourceQuotas) Get(name string) (*quotaapi.AppliedClusterResourceQuota, error) {
	obj, err := c.Fake.Invokes(core.NewGetAction(appliedClusterResourceQuotasResource, c.Namespace, name), &quotaapi.AppliedClusterResourceQuota{})
	if obj == nil {
		return nil, err
	}

	return obj.(*quotaapi.AppliedClusterResourceQuota), err
}

func (c *FakeAppliedClusterResourceQuotas) List(opts kapi.ListOptions) (*quotaapi.AppliedClusterResourceQuotaList, error) {
	obj, err := c.Fake.Invokes(core.NewListAction(appliedClusterResourceQuotasResource, c.Namespace, opts), &quotaapi.AppliedClusterResourceQuotaList{})
	if obj == nil {
		return nil, err
	}

	return obj.(*quotaapi.AppliedClusterResourceQuotaList), err
}
