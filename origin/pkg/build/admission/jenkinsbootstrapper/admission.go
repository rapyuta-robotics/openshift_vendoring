package jenkinsbootstrapper

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/admission"
	kapi "github.com/openshift/kubernetes/pkg/api"
	kapierrors "github.com/openshift/kubernetes/pkg/api/errors"
	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	coreclient "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	kclient "github.com/openshift/kubernetes/pkg/client/unversioned"
	"github.com/openshift/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/kubernetes/pkg/runtime"
	kutilerrors "github.com/openshift/kubernetes/pkg/util/errors"

	"github.com/openshift/origin/pkg/api/latest"
	authenticationclient "github.com/openshift/origin/pkg/auth/client"
	buildapi "github.com/openshift/origin/pkg/build/api"
	jenkinscontroller "github.com/openshift/origin/pkg/build/controller/jenkins"
	"github.com/openshift/origin/pkg/client"
	oadmission "github.com/openshift/origin/pkg/cmd/server/admission"
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/config/cmd"
)

func init() {
	admission.RegisterPlugin("openshift.io/JenkinsBootstrapper", func(c clientset.Interface, config io.Reader) (admission.Interface, error) {
		return NewJenkinsBootstrapper(c.Core()), nil
	})
}

type jenkinsBootstrapper struct {
	*admission.Handler

	privilegedRESTClientConfig restclient.Config
	serviceClient              coreclient.ServicesGetter
	openshiftClient            client.Interface

	jenkinsConfig configapi.JenkinsPipelineConfig
}

var _ = oadmission.WantsJenkinsPipelineConfig(&jenkinsBootstrapper{})
var _ = oadmission.WantsRESTClientConfig(&jenkinsBootstrapper{})
var _ = oadmission.WantsOpenshiftClient(&jenkinsBootstrapper{})

// NewJenkinsBootstrapper returns an admission plugin that will create required jenkins resources as the user if they are needed.
func NewJenkinsBootstrapper(serviceClient coreclient.ServicesGetter) admission.Interface {
	return &jenkinsBootstrapper{
		Handler:       admission.NewHandler(admission.Create),
		serviceClient: serviceClient,
	}
}

func (a *jenkinsBootstrapper) Admit(attributes admission.Attributes) error {
	if a.jenkinsConfig.AutoProvisionEnabled != nil && !*a.jenkinsConfig.AutoProvisionEnabled {
		return nil
	}
	if len(attributes.GetSubresource()) != 0 {
		return nil
	}
	if attributes.GetResource().GroupResource() != buildapi.Resource("buildconfigs") && attributes.GetResource().GroupResource() != buildapi.Resource("builds") {
		return nil
	}
	if !needsJenkinsTemplate(attributes.GetObject()) {
		return nil
	}

	namespace := attributes.GetNamespace()

	svcName := a.jenkinsConfig.ServiceName
	if len(svcName) == 0 {
		return nil
	}

	// TODO pull this from a cache.
	if _, err := a.serviceClient.Services(namespace).Get(svcName); !kapierrors.IsNotFound(err) {
		// if it isn't a "not found" error, return the error.  Either its nil and there's nothing to do or something went really wrong
		return err
	}

	glog.V(3).Infof("Adding new jenkins service %q to the project %q", svcName, namespace)
	jenkinsTemplate := jenkinscontroller.NewPipelineTemplate(namespace, a.jenkinsConfig, a.openshiftClient)
	objects, errs := jenkinsTemplate.Process()
	if len(errs) > 0 {
		return kutilerrors.NewAggregate(errs)
	}
	if !jenkinsTemplate.HasJenkinsService(objects) {
		return fmt.Errorf("template %s/%s does not contain required service %q", a.jenkinsConfig.TemplateNamespace, a.jenkinsConfig.TemplateName, a.jenkinsConfig.ServiceName)
	}

	impersonatingConfig := a.privilegedRESTClientConfig
	oldWrapTransport := impersonatingConfig.WrapTransport
	impersonatingConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return authenticationclient.NewImpersonatingRoundTripper(attributes.GetUserInfo(), oldWrapTransport(rt))
	}

	var bulkErr error

	bulk := &cmd.Bulk{
		Mapper: &resource.Mapper{
			RESTMapper:  registered.RESTMapper(),
			ObjectTyper: kapi.Scheme,
			ClientMapper: resource.ClientMapperFunc(func(mapping *meta.RESTMapping) (resource.RESTClient, error) {
				// TODO this is a nasty copy&paste from pkg/cmd/util/clientcmd/factory_object_mapping.go#ClientForMapping
				if latest.OriginKind(mapping.GroupVersionKind) {
					if err := client.SetOpenShiftDefaults(&impersonatingConfig); err != nil {
						return nil, err
					}
					impersonatingConfig.APIPath = "/apis"
					if mapping.GroupVersionKind.Group == kapi.GroupName {
						impersonatingConfig.APIPath = "/oapi"
					}
					gv := mapping.GroupVersionKind.GroupVersion()
					impersonatingConfig.GroupVersion = &gv
					return restclient.RESTClientFor(&impersonatingConfig)
				}
				// TODO and this from vendor/k8s.io/kubernetes/pkg/kubectl/cmd/util/factory_object_mapping.go#ClientForMapping
				if err := kclient.SetKubernetesDefaults(&impersonatingConfig); err != nil {
					return nil, err
				}
				gvk := mapping.GroupVersionKind
				switch gvk.Group {
				case kapi.GroupName:
					impersonatingConfig.APIPath = "/api"
				default:
					impersonatingConfig.APIPath = "/apis"
				}
				gv := gvk.GroupVersion()
				impersonatingConfig.GroupVersion = &gv
				return restclient.RESTClientFor(&impersonatingConfig)
			}),
		},
		Op: cmd.Create,
		After: func(info *resource.Info, err error) bool {
			if kapierrors.IsAlreadyExists(err) {
				return false
			}
			if err != nil {
				bulkErr = err
				return true
			}
			return false
		},
	}
	// we're intercepting the error we care about using After
	bulk.Run(objects, namespace)
	if bulkErr != nil {
		return bulkErr
	}

	glog.V(1).Infof("Jenkins Pipeline service %q created", svcName)

	return nil

}

func needsJenkinsTemplate(obj runtime.Object) bool {
	switch t := obj.(type) {
	case *buildapi.Build:
		return t.Spec.Strategy.JenkinsPipelineStrategy != nil
	case *buildapi.BuildConfig:
		return t.Spec.Strategy.JenkinsPipelineStrategy != nil
	default:
		return false
	}
}

func (a *jenkinsBootstrapper) SetJenkinsPipelineConfig(jenkinsConfig configapi.JenkinsPipelineConfig) {
	a.jenkinsConfig = jenkinsConfig
}

func (a *jenkinsBootstrapper) SetRESTClientConfig(restClientConfig restclient.Config) {
	a.privilegedRESTClientConfig = restClientConfig
}

func (a *jenkinsBootstrapper) SetOpenshiftClient(oclient client.Interface) {
	a.openshiftClient = oclient
}

func (a *jenkinsBootstrapper) Validate() error {
	if a.openshiftClient == nil {
		return fmt.Errorf("missing openshiftClient")
	}
	return nil
}
