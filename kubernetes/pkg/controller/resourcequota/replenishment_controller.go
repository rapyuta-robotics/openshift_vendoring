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

package resourcequota

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/client/cache"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	"github.com/openshift/kubernetes/pkg/controller"
	"github.com/openshift/kubernetes/pkg/controller/informers"
	"github.com/openshift/kubernetes/pkg/quota/evaluator/core"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/metrics"
	utilruntime "github.com/openshift/kubernetes/pkg/util/runtime"
	"github.com/openshift/kubernetes/pkg/watch"
)

// ReplenishmentFunc is a function that is invoked when controller sees a change
// that may require a quota to be replenished (i.e. object deletion, or object moved to terminal state)
type ReplenishmentFunc func(groupKind unversioned.GroupKind, namespace string, object runtime.Object)

// ReplenishmentControllerOptions is an options struct that tells a factory
// how to configure a controller that can inform the quota system it should
// replenish quota
type ReplenishmentControllerOptions struct {
	// The kind monitored for replenishment
	GroupKind unversioned.GroupKind
	// The period that should be used to re-sync the monitored resource
	ResyncPeriod controller.ResyncPeriodFunc
	// The function to invoke when a change is observed that should trigger
	// replenishment
	ReplenishmentFunc ReplenishmentFunc
}

// PodReplenishmentUpdateFunc will replenish if the old pod was quota tracked but the new is not
func PodReplenishmentUpdateFunc(options *ReplenishmentControllerOptions) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		oldPod := oldObj.(*api.Pod)
		newPod := newObj.(*api.Pod)
		if core.QuotaPod(oldPod) && !core.QuotaPod(newPod) {
			options.ReplenishmentFunc(options.GroupKind, newPod.Namespace, oldPod)
		}
	}
}

// ObjectReplenenishmentDeleteFunc will replenish on every delete
func ObjectReplenishmentDeleteFunc(options *ReplenishmentControllerOptions) func(obj interface{}) {
	return func(obj interface{}) {
		metaObject, err := meta.Accessor(obj)
		if err != nil {
			tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
			if !ok {
				glog.Errorf("replenishment controller could not get object from tombstone %+v, could take up to %v before quota is replenished", obj, options.ResyncPeriod())
				utilruntime.HandleError(err)
				return
			}
			metaObject, err = meta.Accessor(tombstone.Obj)
			if err != nil {
				glog.Errorf("replenishment controller tombstone contained object that is not a meta %+v, could take up to %v before quota is replenished", tombstone.Obj, options.ResyncPeriod())
				utilruntime.HandleError(err)
				return
			}
		}
		options.ReplenishmentFunc(options.GroupKind, metaObject.GetNamespace(), nil)
	}
}

// ReplenishmentControllerFactory knows how to build replenishment controllers
type ReplenishmentControllerFactory interface {
	// NewController returns a controller configured with the specified options.
	// This method is NOT thread-safe.
	NewController(options *ReplenishmentControllerOptions) (cache.ControllerInterface, error)
}

// replenishmentControllerFactory implements ReplenishmentControllerFactory
type replenishmentControllerFactory struct {
	kubeClient            clientset.Interface
	sharedInformerFactory informers.SharedInformerFactory
}

// NewReplenishmentControllerFactory returns a factory that knows how to build controllers
// to replenish resources when updated or deleted
func NewReplenishmentControllerFactory(f informers.SharedInformerFactory, kubeClient clientset.Interface) ReplenishmentControllerFactory {
	return &replenishmentControllerFactory{
		kubeClient:            kubeClient,
		sharedInformerFactory: f,
	}
}

// NewReplenishmentControllerFactoryFromClient returns a factory that knows how to build controllers to replenish resources
// when updated or deleted using the specified client.
func NewReplenishmentControllerFactoryFromClient(kubeClient clientset.Interface) ReplenishmentControllerFactory {
	return NewReplenishmentControllerFactory(nil, kubeClient)
}

// controllerFor returns a replenishment controller for the specified group resource.
func controllerFor(
	groupResource unversioned.GroupResource,
	f informers.SharedInformerFactory,
	handlerFuncs cache.ResourceEventHandlerFuncs) (cache.ControllerInterface, error) {
	genericInformer, err := f.ForResource(groupResource)
	if err != nil {
		return nil, err
	}
	informer := genericInformer.Informer()
	informer.AddEventHandler(handlerFuncs)
	return informer.GetController(), nil
}

func (r *replenishmentControllerFactory) NewController(options *ReplenishmentControllerOptions) (result cache.ControllerInterface, err error) {
	if r.kubeClient != nil && r.kubeClient.Core().RESTClient().GetRateLimiter() != nil {
		metrics.RegisterMetricAndTrackRateLimiterUsage("replenishment_controller", r.kubeClient.Core().RESTClient().GetRateLimiter())
	}

	switch options.GroupKind {
	case api.Kind("Pod"):
		if r.sharedInformerFactory != nil {
			result, err = controllerFor(api.Resource("pods"), r.sharedInformerFactory, cache.ResourceEventHandlerFuncs{
				UpdateFunc: PodReplenishmentUpdateFunc(options),
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			})
			break
		}
		result = informers.NewPodInformer(r.kubeClient, options.ResyncPeriod())
	case api.Kind("Service"):
		// TODO move to informer when defined
		_, result = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options api.ListOptions) (runtime.Object, error) {
					return r.kubeClient.Core().Services(api.NamespaceAll).List(options)
				},
				WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
					return r.kubeClient.Core().Services(api.NamespaceAll).Watch(options)
				},
			},
			&api.Service{},
			options.ResyncPeriod(),
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: ServiceReplenishmentUpdateFunc(options),
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			},
		)
	case api.Kind("ReplicationController"):
		// TODO move to informer when defined
		_, result = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options api.ListOptions) (runtime.Object, error) {
					return r.kubeClient.Core().ReplicationControllers(api.NamespaceAll).List(options)
				},
				WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
					return r.kubeClient.Core().ReplicationControllers(api.NamespaceAll).Watch(options)
				},
			},
			&api.ReplicationController{},
			options.ResyncPeriod(),
			cache.ResourceEventHandlerFuncs{
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			},
		)
	case api.Kind("PersistentVolumeClaim"):
		if r.sharedInformerFactory != nil {
			result, err = controllerFor(api.Resource("persistentvolumeclaims"), r.sharedInformerFactory, cache.ResourceEventHandlerFuncs{
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			})
			break
		}
		// TODO (derekwaynecarr) remove me when we can require a sharedInformerFactory in all code paths...
		_, result = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options api.ListOptions) (runtime.Object, error) {
					return r.kubeClient.Core().PersistentVolumeClaims(api.NamespaceAll).List(options)
				},
				WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
					return r.kubeClient.Core().PersistentVolumeClaims(api.NamespaceAll).Watch(options)
				},
			},
			&api.PersistentVolumeClaim{},
			options.ResyncPeriod(),
			cache.ResourceEventHandlerFuncs{
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			},
		)
	case api.Kind("Secret"):
		// TODO move to informer when defined
		_, result = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options api.ListOptions) (runtime.Object, error) {
					return r.kubeClient.Core().Secrets(api.NamespaceAll).List(options)
				},
				WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
					return r.kubeClient.Core().Secrets(api.NamespaceAll).Watch(options)
				},
			},
			&api.Secret{},
			options.ResyncPeriod(),
			cache.ResourceEventHandlerFuncs{
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			},
		)
	case api.Kind("ConfigMap"):
		// TODO move to informer when defined
		_, result = cache.NewInformer(
			&cache.ListWatch{
				ListFunc: func(options api.ListOptions) (runtime.Object, error) {
					return r.kubeClient.Core().ConfigMaps(api.NamespaceAll).List(options)
				},
				WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
					return r.kubeClient.Core().ConfigMaps(api.NamespaceAll).Watch(options)
				},
			},
			&api.ConfigMap{},
			options.ResyncPeriod(),
			cache.ResourceEventHandlerFuncs{
				DeleteFunc: ObjectReplenishmentDeleteFunc(options),
			},
		)
	default:
		return nil, NewUnhandledGroupKindError(options.GroupKind)
	}
	return result, err
}

// ServiceReplenishmentUpdateFunc will replenish if the service was quota tracked has changed service type
func ServiceReplenishmentUpdateFunc(options *ReplenishmentControllerOptions) func(oldObj, newObj interface{}) {
	return func(oldObj, newObj interface{}) {
		oldService := oldObj.(*api.Service)
		newService := newObj.(*api.Service)
		if core.GetQuotaServiceType(oldService) != core.GetQuotaServiceType(newService) {
			options.ReplenishmentFunc(options.GroupKind, newService.Namespace, nil)
		}
	}
}

type unhandledKindErr struct {
	kind unversioned.GroupKind
}

func (e unhandledKindErr) Error() string {
	return fmt.Sprintf("no replenishment controller available for %s", e.kind)
}

func NewUnhandledGroupKindError(kind unversioned.GroupKind) error {
	return unhandledKindErr{kind: kind}
}

func IsUnhandledGroupKindError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(unhandledKindErr)
	return ok
}

// UnionReplenishmentControllerFactory iterates through its constituent factories ignoring, UnhandledGroupKindErrors
// returning the first success or failure it hits.  If there are no hits either way, it return an UnhandledGroupKind error
type UnionReplenishmentControllerFactory []ReplenishmentControllerFactory

func (f UnionReplenishmentControllerFactory) NewController(options *ReplenishmentControllerOptions) (cache.ControllerInterface, error) {
	for _, factory := range f {
		controller, err := factory.NewController(options)
		if !IsUnhandledGroupKindError(err) {
			return controller, err
		}
	}

	return nil, NewUnhandledGroupKindError(options.GroupKind)
}
