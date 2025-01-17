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

package etcd

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/registry/cachesize"
	"github.com/openshift/kubernetes/pkg/registry/core/podtemplate"
	"github.com/openshift/kubernetes/pkg/registry/generic"
	"github.com/openshift/kubernetes/pkg/registry/generic/registry"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/storage"
)

type REST struct {
	*registry.Store
}

// NewREST returns a RESTStorage object that will work against pod templates.
func NewREST(opts generic.RESTOptions) *REST {
	prefix := "/" + opts.ResourcePrefix

	newListFunc := func() runtime.Object { return &api.PodTemplateList{} }
	storageInterface, dFunc := opts.Decorator(
		opts.StorageConfig,
		cachesize.GetWatchCacheSizeByResource(cachesize.PodTemplates),
		&api.PodTemplate{},
		prefix,
		podtemplate.Strategy,
		newListFunc,
		storage.NoTriggerPublisher,
	)

	store := &registry.Store{
		NewFunc:     func() runtime.Object { return &api.PodTemplate{} },
		NewListFunc: newListFunc,
		KeyRootFunc: func(ctx api.Context) string {
			return registry.NamespaceKeyRootFunc(ctx, prefix)
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return registry.NamespaceKeyFunc(ctx, prefix, name)
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*api.PodTemplate).Name, nil
		},
		PredicateFunc:           podtemplate.MatchPodTemplate,
		QualifiedResource:       api.Resource("podtemplates"),
		EnableGarbageCollection: opts.EnableGarbageCollection,
		DeleteCollectionWorkers: opts.DeleteCollectionWorkers,

		CreateStrategy: podtemplate.Strategy,
		UpdateStrategy: podtemplate.Strategy,
		DeleteStrategy: podtemplate.Strategy,
		ExportStrategy: podtemplate.Strategy,

		ReturnDeletedObject: true,

		Storage:     storageInterface,
		DestroyFunc: dFunc,
	}
	return &REST{store}
}
