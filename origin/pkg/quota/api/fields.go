package api

import "github.com/openshift/kubernetes/pkg/fields"

func ClusterResourceQuotaToSelectableFields(quota *ClusterResourceQuota) fields.Set {
	return fields.Set{
		"metadata.name": quota.Name,
	}
}
