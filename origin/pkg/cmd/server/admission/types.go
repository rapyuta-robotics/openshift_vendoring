package admission

import (
	"github.com/openshift/kubernetes/pkg/admission"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	"github.com/openshift/kubernetes/pkg/quota"

	"github.com/openshift/origin/pkg/authorization/authorizer"
	"github.com/openshift/origin/pkg/client"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/controller/shared"
	imageapi "github.com/openshift/origin/pkg/image/api"
	"github.com/openshift/origin/pkg/project/cache"
	"github.com/openshift/origin/pkg/quota/controller/clusterquotamapping"
	usercache "github.com/openshift/origin/pkg/user/cache"
)

// WantsOpenshiftClient should be implemented by admission plugins that need
// an Openshift client
type WantsOpenshiftClient interface {
	SetOpenshiftClient(client.Interface)
	admission.Validator
}

// WantsProjectCache should be implemented by admission plugins that need a
// project cache
type WantsProjectCache interface {
	SetProjectCache(*cache.ProjectCache)
	admission.Validator
}

// WantsQuotaRegistry should be implemented by admission plugins that need a quota registry
type WantsOriginQuotaRegistry interface {
	SetOriginQuotaRegistry(quota.Registry)
	admission.Validator
}

// WantsAuthorizer should be implemented by admission plugins that
// need access to the Authorizer interface
type WantsAuthorizer interface {
	SetAuthorizer(authorizer.Authorizer)
	admission.Validator
}

// WantsJenkinsPipelineConfig gives access to the JenkinsPipelineConfig.  This is a historical oddity.
// It's likely that what we really wanted was this as an admission plugin config
type WantsJenkinsPipelineConfig interface {
	SetJenkinsPipelineConfig(jenkinsConfig configapi.JenkinsPipelineConfig)
	admission.Validator
}

// WantsRESTClientConfig gives access to a RESTClientConfig.  It's useful for doing unusual things with transports.
type WantsRESTClientConfig interface {
	SetRESTClientConfig(restclient.Config)
	admission.Validator
}

// WantsInformers should be implemented by admission plugins that will select its own informer
type WantsInformers interface {
	SetInformers(shared.InformerFactory)
	admission.Validator
}

// WantsClusterQuotaMapper should be implemented by admission plugins that need to know how to map between
// cluster quota and namespaces
type WantsClusterQuotaMapper interface {
	SetClusterQuotaMapper(clusterquotamapping.ClusterQuotaMapper)
	admission.Validator
}

// WantsDefaultRegistryFunc should be implemented by admission plugins that need to know the default registry
// address.
type WantsDefaultRegistryFunc interface {
	SetDefaultRegistryFunc(imageapi.DefaultRegistryFunc)
	admission.Validator
}

// WantsGroupCache should be implemented by admission plugins that need a
// group cache.
type WantsGroupCache interface {
	SetGroupCache(*usercache.GroupCache)
	admission.Validator
}
