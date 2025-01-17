/*
Copyright 2016 The Kubernetes Authors.

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

package testing

import (
	"errors"
	"fmt"
	"io"

	"github.com/openshift/github.com/emicklei/go-restful/swagger"
	"github.com/openshift/github.com/spf13/cobra"
	"github.com/openshift/github.com/spf13/pflag"

	fedclientset "github.com/openshift/kubernetes/federation/client/clientset_generated/federation_internalclientset"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/api/testapi"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/api/validation"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	"github.com/openshift/kubernetes/pkg/client/restclient/fake"
	"github.com/openshift/kubernetes/pkg/client/typed/discovery"
	"github.com/openshift/kubernetes/pkg/kubectl"
	cmdutil "github.com/openshift/kubernetes/pkg/kubectl/cmd/util"
	"github.com/openshift/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/runtime/serializer"
)

type InternalType struct {
	Kind       string
	APIVersion string

	Name string
}

type ExternalType struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`

	Name string `json:"name"`
}

type ExternalType2 struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`

	Name string `json:"name"`
}

func (obj *InternalType) GetObjectKind() unversioned.ObjectKind { return obj }
func (obj *InternalType) SetGroupVersionKind(gvk unversioned.GroupVersionKind) {
	obj.APIVersion, obj.Kind = gvk.ToAPIVersionAndKind()
}
func (obj *InternalType) GroupVersionKind() unversioned.GroupVersionKind {
	return unversioned.FromAPIVersionAndKind(obj.APIVersion, obj.Kind)
}
func (obj *ExternalType) GetObjectKind() unversioned.ObjectKind { return obj }
func (obj *ExternalType) SetGroupVersionKind(gvk unversioned.GroupVersionKind) {
	obj.APIVersion, obj.Kind = gvk.ToAPIVersionAndKind()
}
func (obj *ExternalType) GroupVersionKind() unversioned.GroupVersionKind {
	return unversioned.FromAPIVersionAndKind(obj.APIVersion, obj.Kind)
}
func (obj *ExternalType2) GetObjectKind() unversioned.ObjectKind { return obj }
func (obj *ExternalType2) SetGroupVersionKind(gvk unversioned.GroupVersionKind) {
	obj.APIVersion, obj.Kind = gvk.ToAPIVersionAndKind()
}
func (obj *ExternalType2) GroupVersionKind() unversioned.GroupVersionKind {
	return unversioned.FromAPIVersionAndKind(obj.APIVersion, obj.Kind)
}

func NewInternalType(kind, apiversion, name string) *InternalType {
	item := InternalType{Kind: kind,
		APIVersion: apiversion,
		Name:       name}
	return &item
}

var versionErr = errors.New("not a version")

func versionErrIfFalse(b bool) error {
	if b {
		return nil
	}
	return versionErr
}

var ValidVersion = registered.GroupOrDie(api.GroupName).GroupVersion.Version
var InternalGV = unversioned.GroupVersion{Group: "apitest", Version: runtime.APIVersionInternal}
var UnlikelyGV = unversioned.GroupVersion{Group: "apitest", Version: "unlikelyversion"}
var ValidVersionGV = unversioned.GroupVersion{Group: "apitest", Version: ValidVersion}

func newExternalScheme() (*runtime.Scheme, meta.RESTMapper, runtime.Codec) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(InternalGV.WithKind("Type"), &InternalType{})
	scheme.AddKnownTypeWithName(UnlikelyGV.WithKind("Type"), &ExternalType{})
	//This tests that kubectl will not confuse the external scheme with the internal scheme, even when they accidentally have versions of the same name.
	scheme.AddKnownTypeWithName(ValidVersionGV.WithKind("Type"), &ExternalType2{})

	codecs := serializer.NewCodecFactory(scheme)
	codec := codecs.LegacyCodec(UnlikelyGV)
	mapper := meta.NewDefaultRESTMapper([]unversioned.GroupVersion{UnlikelyGV, ValidVersionGV}, func(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
		return &meta.VersionInterfaces{
			ObjectConvertor:  scheme,
			MetadataAccessor: meta.NewAccessor(),
		}, versionErrIfFalse(version == ValidVersionGV || version == UnlikelyGV)
	})
	for _, gv := range []unversioned.GroupVersion{UnlikelyGV, ValidVersionGV} {
		for kind := range scheme.KnownTypes(gv) {
			gvk := gv.WithKind(kind)

			scope := meta.RESTScopeNamespace
			mapper.Add(gvk, scope)
		}
	}

	return scheme, mapper, codec
}

type fakeCachedDiscoveryClient struct {
	discovery.DiscoveryInterface
}

func (d *fakeCachedDiscoveryClient) Fresh() bool {
	return true
}

func (d *fakeCachedDiscoveryClient) Invalidate() {
}

type TestFactory struct {
	Mapper       meta.RESTMapper
	Typer        runtime.ObjectTyper
	Client       kubectl.RESTClient
	Describer    kubectl.Describer
	Printer      kubectl.ResourcePrinter
	Validator    validation.Schema
	Namespace    string
	ClientConfig *restclient.Config
	Err          error
}

type FakeFactory struct {
	tf    *TestFactory
	Codec runtime.Codec
}

func NewTestFactory() (cmdutil.Factory, *TestFactory, runtime.Codec, runtime.NegotiatedSerializer) {
	scheme, mapper, codec := newExternalScheme()
	t := &TestFactory{
		Validator: validation.NullSchema{},
		Mapper:    mapper,
		Typer:     scheme,
	}
	negotiatedSerializer := serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{Serializer: codec})
	return &FakeFactory{
		tf:    t,
		Codec: codec,
	}, t, codec, negotiatedSerializer
}

func (f *FakeFactory) DiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(f.tf.ClientConfig)
	if err != nil {
		return nil, err
	}
	return &fakeCachedDiscoveryClient{DiscoveryInterface: discoveryClient}, nil
}

func (f *FakeFactory) FlagSet() *pflag.FlagSet {
	return nil
}

func (f *FakeFactory) Object() (meta.RESTMapper, runtime.ObjectTyper) {
	priorityRESTMapper := meta.PriorityRESTMapper{
		Delegate: f.tf.Mapper,
		ResourcePriority: []unversioned.GroupVersionResource{
			{Group: meta.AnyGroup, Version: "v1", Resource: meta.AnyResource},
		},
		KindPriority: []unversioned.GroupVersionKind{
			{Group: meta.AnyGroup, Version: "v1", Kind: meta.AnyKind},
		},
	}
	return priorityRESTMapper, f.tf.Typer
}

func (f *FakeFactory) UnstructuredObject() (meta.RESTMapper, runtime.ObjectTyper, error) {
	groupResources := testDynamicResources()
	mapper := discovery.NewRESTMapper(groupResources, meta.InterfacesForUnstructured)
	typer := discovery.NewUnstructuredObjectTyper(groupResources)

	return cmdutil.NewShortcutExpander(mapper, nil), typer, nil
}

func (f *FakeFactory) Decoder(bool) runtime.Decoder {
	return f.Codec
}

func (f *FakeFactory) JSONEncoder() runtime.Encoder {
	return f.Codec
}

func (f *FakeFactory) RESTClient() (*restclient.RESTClient, error) {
	return nil, nil
}

func (f *FakeFactory) ClientSet() (*internalclientset.Clientset, error) {
	return nil, nil
}

func (f *FakeFactory) ClientConfig() (*restclient.Config, error) {
	return f.tf.ClientConfig, f.tf.Err
}

func (f *FakeFactory) ClientForMapping(*meta.RESTMapping) (resource.RESTClient, error) {
	return f.tf.Client, f.tf.Err
}

func (f *FakeFactory) FederationClientSetForVersion(version *unversioned.GroupVersion) (fedclientset.Interface, error) {
	return nil, nil
}
func (f *FakeFactory) FederationClientForVersion(version *unversioned.GroupVersion) (*restclient.RESTClient, error) {
	return nil, nil
}
func (f *FakeFactory) ClientSetForVersion(requiredVersion *unversioned.GroupVersion) (*internalclientset.Clientset, error) {
	return nil, nil
}
func (f *FakeFactory) ClientConfigForVersion(requiredVersion *unversioned.GroupVersion) (*restclient.Config, error) {
	return nil, nil
}

func (f *FakeFactory) UnstructuredClientForMapping(*meta.RESTMapping) (resource.RESTClient, error) {
	return nil, nil
}

func (f *FakeFactory) Describer(*meta.RESTMapping) (kubectl.Describer, error) {
	return f.tf.Describer, f.tf.Err
}

func (f *FakeFactory) Printer(mapping *meta.RESTMapping, options kubectl.PrintOptions) (kubectl.ResourcePrinter, error) {
	return f.tf.Printer, f.tf.Err
}

func (f *FakeFactory) Scaler(*meta.RESTMapping) (kubectl.Scaler, error) {
	return nil, nil
}

func (f *FakeFactory) Reaper(*meta.RESTMapping) (kubectl.Reaper, error) {
	return nil, nil
}

func (f *FakeFactory) HistoryViewer(*meta.RESTMapping) (kubectl.HistoryViewer, error) {
	return nil, nil
}

func (f *FakeFactory) Rollbacker(*meta.RESTMapping) (kubectl.Rollbacker, error) {
	return nil, nil
}

func (f *FakeFactory) StatusViewer(*meta.RESTMapping) (kubectl.StatusViewer, error) {
	return nil, nil
}

func (f *FakeFactory) MapBasedSelectorForObject(runtime.Object) (string, error) {
	return "", nil
}

func (f *FakeFactory) PortsForObject(runtime.Object) ([]string, error) {
	return nil, nil
}

func (f *FakeFactory) ProtocolsForObject(runtime.Object) (map[string]string, error) {
	return nil, nil
}

func (f *FakeFactory) LabelsForObject(runtime.Object) (map[string]string, error) {
	return nil, nil
}

func (f *FakeFactory) LogsForObject(object, options runtime.Object) (*restclient.Request, error) {
	return nil, nil
}

func (f *FakeFactory) Pauser(info *resource.Info) (bool, error) {
	return false, nil
}

func (f *FakeFactory) Resumer(info *resource.Info) (bool, error) {
	return false, nil
}

func (f *FakeFactory) ResolveImage(name string) (string, error) {
	return name, nil
}

func (f *FakeFactory) Validator(validate bool, cacheDir string) (validation.Schema, error) {
	return f.tf.Validator, f.tf.Err
}

func (f *FakeFactory) SwaggerSchema(unversioned.GroupVersionKind) (*swagger.ApiDeclaration, error) {
	return nil, nil
}

func (f *FakeFactory) DefaultNamespace() (string, bool, error) {
	return f.tf.Namespace, false, f.tf.Err
}

func (f *FakeFactory) Generators(cmdName string) map[string]kubectl.Generator {
	var generator map[string]kubectl.Generator
	switch cmdName {
	case "run":
		generator = map[string]kubectl.Generator{
			cmdutil.DeploymentV1Beta1GeneratorName: kubectl.DeploymentV1Beta1{},
		}
	}
	return generator
}

func (f *FakeFactory) CanBeExposed(unversioned.GroupKind) error {
	return nil
}

func (f *FakeFactory) CanBeAutoscaled(unversioned.GroupKind) error {
	return nil
}

func (f *FakeFactory) AttachablePodForObject(ob runtime.Object) (*api.Pod, error) {
	return nil, nil
}

func (f *FakeFactory) UpdatePodSpecForObject(obj runtime.Object, fn func(*api.PodSpec) error) (bool, error) {
	return false, nil
}

func (f *FakeFactory) EditorEnvs() []string {
	return nil
}

func (f *FakeFactory) PrintObjectSpecificMessage(obj runtime.Object, out io.Writer) {
}

func (f *FakeFactory) Command() string {
	return ""
}

func (f *FakeFactory) BindFlags(flags *pflag.FlagSet) {
}

func (f *FakeFactory) BindExternalFlags(flags *pflag.FlagSet) {
}

func (f *FakeFactory) PrintObject(cmd *cobra.Command, mapper meta.RESTMapper, obj runtime.Object, out io.Writer) error {
	return nil
}

func (f *FakeFactory) PrinterForMapping(cmd *cobra.Command, mapping *meta.RESTMapping, withNamespace bool) (kubectl.ResourcePrinter, error) {
	return f.tf.Printer, f.tf.Err
}

func (f *FakeFactory) NewBuilder() *resource.Builder {
	return nil
}

func (f *FakeFactory) DefaultResourceFilterOptions(cmd *cobra.Command, withNamespace bool) *kubectl.PrintOptions {
	return &kubectl.PrintOptions{}
}

func (f *FakeFactory) DefaultResourceFilterFunc() kubectl.Filters {
	return nil
}

func (f *FakeFactory) SuggestedPodTemplateResources() []unversioned.GroupResource {
	return []unversioned.GroupResource{}
}

type fakeMixedFactory struct {
	cmdutil.Factory
	tf        *TestFactory
	apiClient resource.RESTClient
}

func (f *fakeMixedFactory) Object() (meta.RESTMapper, runtime.ObjectTyper) {
	var multiRESTMapper meta.MultiRESTMapper
	multiRESTMapper = append(multiRESTMapper, f.tf.Mapper)
	multiRESTMapper = append(multiRESTMapper, testapi.Default.RESTMapper())
	priorityRESTMapper := meta.PriorityRESTMapper{
		Delegate: multiRESTMapper,
		ResourcePriority: []unversioned.GroupVersionResource{
			{Group: meta.AnyGroup, Version: "v1", Resource: meta.AnyResource},
		},
		KindPriority: []unversioned.GroupVersionKind{
			{Group: meta.AnyGroup, Version: "v1", Kind: meta.AnyKind},
		},
	}
	return priorityRESTMapper, runtime.MultiObjectTyper{f.tf.Typer, api.Scheme}
}

func (f *fakeMixedFactory) ClientForMapping(m *meta.RESTMapping) (resource.RESTClient, error) {
	if m.ObjectConvertor == api.Scheme {
		return f.apiClient, f.tf.Err
	}
	return f.tf.Client, f.tf.Err
}

func NewMixedFactory(apiClient resource.RESTClient) (cmdutil.Factory, *TestFactory, runtime.Codec) {
	f, t, c, _ := NewAPIFactory()
	return &fakeMixedFactory{
		Factory:   f,
		tf:        t,
		apiClient: apiClient,
	}, t, c
}

type fakeAPIFactory struct {
	cmdutil.Factory
	tf *TestFactory
}

func (f *fakeAPIFactory) Object() (meta.RESTMapper, runtime.ObjectTyper) {
	return testapi.Default.RESTMapper(), api.Scheme
}

func (f *fakeAPIFactory) UnstructuredObject() (meta.RESTMapper, runtime.ObjectTyper, error) {
	groupResources := testDynamicResources()
	mapper := discovery.NewRESTMapper(groupResources, meta.InterfacesForUnstructured)
	typer := discovery.NewUnstructuredObjectTyper(groupResources)

	return cmdutil.NewShortcutExpander(mapper, nil), typer, nil
}

func (f *fakeAPIFactory) Decoder(bool) runtime.Decoder {
	return testapi.Default.Codec()
}

func (f *fakeAPIFactory) JSONEncoder() runtime.Encoder {
	return testapi.Default.Codec()
}

func (f *fakeAPIFactory) ClientSet() (*internalclientset.Clientset, error) {
	// Swap the HTTP client out of the REST client with the fake
	// version.
	fakeClient := f.tf.Client.(*fake.RESTClient)
	clientset := internalclientset.NewForConfigOrDie(f.tf.ClientConfig)
	clientset.CoreClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthenticationClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AuthorizationClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AutoscalingClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.BatchClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.CertificatesClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.ExtensionsClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.RbacClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.StorageClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.AppsClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.PolicyClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	clientset.DiscoveryClient.RESTClient().(*restclient.RESTClient).Client = fakeClient.Client
	return clientset, f.tf.Err
}

func (f *fakeAPIFactory) RESTClient() (*restclient.RESTClient, error) {
	// Swap out the HTTP client out of the client with the fake's version.
	fakeClient := f.tf.Client.(*fake.RESTClient)
	restClient, err := restclient.RESTClientFor(f.tf.ClientConfig)
	if err != nil {
		panic(err)
	}
	restClient.Client = fakeClient.Client
	return restClient, f.tf.Err
}

func (f *fakeAPIFactory) ClientConfig() (*restclient.Config, error) {
	return f.tf.ClientConfig, f.tf.Err
}

func (f *fakeAPIFactory) ClientForMapping(*meta.RESTMapping) (resource.RESTClient, error) {
	return f.tf.Client, f.tf.Err
}

func (f *fakeAPIFactory) UnstructuredClientForMapping(*meta.RESTMapping) (resource.RESTClient, error) {
	return f.tf.Client, f.tf.Err
}

func (f *fakeAPIFactory) Describer(*meta.RESTMapping) (kubectl.Describer, error) {
	return f.tf.Describer, f.tf.Err
}

func (f *fakeAPIFactory) Printer(mapping *meta.RESTMapping, options kubectl.PrintOptions) (kubectl.ResourcePrinter, error) {
	return f.tf.Printer, f.tf.Err
}

func (f *fakeAPIFactory) LogsForObject(object, options runtime.Object) (*restclient.Request, error) {
	c, err := f.ClientSet()
	if err != nil {
		panic(err)
	}

	switch t := object.(type) {
	case *api.Pod:
		opts, ok := options.(*api.PodLogOptions)
		if !ok {
			return nil, errors.New("provided options object is not a PodLogOptions")
		}
		return c.Core().Pods(f.tf.Namespace).GetLogs(t.Name, opts), nil
	default:
		fqKinds, _, err := api.Scheme.ObjectKinds(object)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("cannot get the logs from %v", fqKinds[0])
	}
}

func (f *fakeAPIFactory) Validator(validate bool, cacheDir string) (validation.Schema, error) {
	return f.tf.Validator, f.tf.Err
}

func (f *fakeAPIFactory) DefaultNamespace() (string, bool, error) {
	return f.tf.Namespace, false, f.tf.Err
}

func (f *fakeAPIFactory) Generators(cmdName string) map[string]kubectl.Generator {
	return cmdutil.DefaultGenerators(cmdName)
}

func (f *fakeAPIFactory) PrintObject(cmd *cobra.Command, mapper meta.RESTMapper, obj runtime.Object, out io.Writer) error {
	gvks, _, err := api.Scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}

	mapping, err := mapper.RESTMapping(gvks[0].GroupKind())
	if err != nil {
		return err
	}

	printer, err := f.PrinterForMapping(cmd, mapping, false)
	if err != nil {
		return err
	}
	return printer.PrintObj(obj, out)
}

func (f *fakeAPIFactory) PrinterForMapping(cmd *cobra.Command, mapping *meta.RESTMapping, withNamespace bool) (kubectl.ResourcePrinter, error) {
	return f.tf.Printer, f.tf.Err
}

func (f *fakeAPIFactory) NewBuilder() *resource.Builder {
	mapper, typer := f.Object()

	return resource.NewBuilder(mapper, typer, resource.ClientMapperFunc(f.ClientForMapping), f.Decoder(true))
}

func (f *fakeAPIFactory) SuggestedPodTemplateResources() []unversioned.GroupResource {
	return []unversioned.GroupResource{}
}

func NewAPIFactory() (cmdutil.Factory, *TestFactory, runtime.Codec, runtime.NegotiatedSerializer) {
	t := &TestFactory{
		Validator: validation.NullSchema{},
	}
	rf := cmdutil.NewFactory(nil)
	return &fakeAPIFactory{
		Factory: rf,
		tf:      t,
	}, t, testapi.Default.Codec(), testapi.Default.NegotiatedSerializer()
}

func testDynamicResources() []*discovery.APIGroupResources {
	return []*discovery.APIGroupResources{
		{
			Group: unversioned.APIGroup{
				Versions: []unversioned.GroupVersionForDiscovery{
					{Version: "v1"},
				},
				PreferredVersion: unversioned.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]unversioned.APIResource{
				"v1": {
					{Name: "pods", Namespaced: true, Kind: "Pod"},
					{Name: "services", Namespaced: true, Kind: "Service"},
					{Name: "replicationcontrollers", Namespaced: true, Kind: "ReplicationController"},
					{Name: "componentstatuses", Namespaced: false, Kind: "ComponentStatus"},
					{Name: "nodes", Namespaced: false, Kind: "Node"},
					{Name: "type", Namespaced: false, Kind: "Type"},
				},
			},
		},
	}
}
