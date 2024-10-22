package appliedclusterresourcequota

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	kapierrors "github.com/openshift/kubernetes/pkg/api/errors"
	"github.com/openshift/kubernetes/pkg/client/cache"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/sets"

	oapi "github.com/openshift/origin/pkg/api"
	ocache "github.com/openshift/origin/pkg/client/cache"
	quotaapi "github.com/openshift/origin/pkg/quota/api"
	"github.com/openshift/origin/pkg/quota/controller/clusterquotamapping"
	clusterresourcequotaregistry "github.com/openshift/origin/pkg/quota/registry/clusterresourcequota"
)

type AppliedClusterResourceQuotaREST struct {
	quotaMapper     clusterquotamapping.ClusterQuotaMapper
	quotaLister     *ocache.IndexerToClusterResourceQuotaLister
	namespaceLister *cache.IndexerToNamespaceLister
}

func NewREST(quotaMapper clusterquotamapping.ClusterQuotaMapper, quotaLister *ocache.IndexerToClusterResourceQuotaLister, namespaceLister *cache.IndexerToNamespaceLister) *AppliedClusterResourceQuotaREST {
	return &AppliedClusterResourceQuotaREST{
		quotaMapper:     quotaMapper,
		quotaLister:     quotaLister,
		namespaceLister: namespaceLister,
	}
}

func (r *AppliedClusterResourceQuotaREST) New() runtime.Object {
	return &quotaapi.AppliedClusterResourceQuota{}
}

func (r *AppliedClusterResourceQuotaREST) Get(ctx kapi.Context, name string) (runtime.Object, error) {
	namespace, ok := kapi.NamespaceFrom(ctx)
	if !ok {
		return nil, kapierrors.NewBadRequest("namespace is required")
	}

	quotaNames, _ := r.quotaMapper.GetClusterQuotasFor(namespace)
	quotaNamesSet := sets.NewString(quotaNames...)
	if !quotaNamesSet.Has(name) {
		return nil, kapierrors.NewNotFound(quotaapi.Resource("appliedclusterresourcequota"), name)
	}

	clusterQuota, err := r.quotaLister.Get(name)
	if err != nil {
		return nil, err
	}

	return quotaapi.ConvertClusterResourceQuotaToAppliedClusterResourceQuota(clusterQuota), nil
}

func (r *AppliedClusterResourceQuotaREST) NewList() runtime.Object {
	return &quotaapi.AppliedClusterResourceQuotaList{}
}

func (r *AppliedClusterResourceQuotaREST) List(ctx kapi.Context, options *kapi.ListOptions) (runtime.Object, error) {
	namespace, ok := kapi.NamespaceFrom(ctx)
	if !ok {
		return nil, kapierrors.NewBadRequest("namespace is required")
	}

	// TODO max resource version?  watch?
	list := &quotaapi.AppliedClusterResourceQuotaList{}
	matcher := clusterresourcequotaregistry.Matcher(oapi.ListOptionsToSelectors(options))
	quotaNames, _ := r.quotaMapper.GetClusterQuotasFor(namespace)

	for _, name := range quotaNames {
		quota, err := r.quotaLister.Get(name)
		if err != nil {
			continue
		}
		if matches, err := matcher.Matches(quota); err != nil || !matches {
			continue
		}
		list.Items = append(list.Items, *quotaapi.ConvertClusterResourceQuotaToAppliedClusterResourceQuota(quota))
	}

	return list, nil
}
