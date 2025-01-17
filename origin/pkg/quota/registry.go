package quota

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	kquota "github.com/openshift/kubernetes/pkg/quota"
	"github.com/openshift/kubernetes/pkg/quota/install"

	osclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/controller/shared"
	imageapi "github.com/openshift/origin/pkg/image/api"
	"github.com/openshift/origin/pkg/quota/image"
)

// NewOriginQuotaRegistry returns a registry object that knows how to evaluate quota usage of OpenShift
// resources.
func NewOriginQuotaRegistry(isInformer shared.ImageStreamInformer, osClient osclient.Interface) kquota.Registry {
	return image.NewImageQuotaRegistry(isInformer, osClient)
}

// NewAllResourceQuotaRegistry returns a registry object that knows how to evaluate all resources
func NewAllResourceQuotaRegistry(informerFactory shared.InformerFactory, osClient osclient.Interface, kubeClientSet clientset.Interface) kquota.Registry {
	return kquota.UnionRegistry{install.NewRegistry(kubeClientSet, informerFactory.KubernetesInformers()), NewOriginQuotaRegistry(informerFactory.ImageStreams(), osClient)}
}

// AllEvaluatedGroupKinds is the list of all group kinds that we evaluate for quotas in openshift and kube
var AllEvaluatedGroupKinds = []unversioned.GroupKind{
	kapi.Kind("Pod"),
	kapi.Kind("Service"),
	kapi.Kind("ReplicationController"),
	kapi.Kind("PersistentVolumeClaim"),
	kapi.Kind("Secret"),
	kapi.Kind("ConfigMap"),

	imageapi.Kind("ImageStream"),
}
