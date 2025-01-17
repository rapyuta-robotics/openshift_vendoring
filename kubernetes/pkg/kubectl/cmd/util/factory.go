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

package util

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/openshift/github.com/emicklei/go-restful/swagger"
	"github.com/openshift/github.com/spf13/cobra"
	"github.com/openshift/github.com/spf13/pflag"

	fedclientset "github.com/openshift/kubernetes/federation/client/clientset_generated/federation_internalclientset"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/api/validation"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	coreclient "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	"github.com/openshift/kubernetes/pkg/client/restclient"
	"github.com/openshift/kubernetes/pkg/client/typed/discovery"
	"github.com/openshift/kubernetes/pkg/client/unversioned/clientcmd"
	"github.com/openshift/kubernetes/pkg/kubectl"
	"github.com/openshift/kubernetes/pkg/kubectl/resource"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/registry/extensions/thirdpartyresourcedata"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/runtime/serializer/json"
	"github.com/openshift/kubernetes/pkg/watch"
)

const (
	FlagMatchBinaryVersion = "match-server-version"
)

// Factory provides abstractions that allow the Kubectl command to be extended across multiple types
// of resources and different API sets.
// The rings are here for a reason.  In order for composers to be able to provide alternative factory implementations
// they need to provide low level pieces of *certain* functions so that when the factory calls back into itself
// it uses the custom version of the function.  Rather than try to enumerate everything that someone would want to override
// we split the factory into rings, where each ring can depend on methods  an earlier ring, but cannot depend
// upon peer methods in its own ring.
// TODO: make the functions interfaces
// TODO: pass the various interfaces on the factory directly into the command constructors (so the
// commands are decoupled from the factory).
type Factory interface {
	ClientAccessFactory
	ObjectMappingFactory
	BuilderFactory
}

type DiscoveryClientFactory interface {
	// Returns a discovery client
	DiscoveryClient() (discovery.CachedDiscoveryInterface, error)
}

// ClientAccessFactory holds the first level of factory methods.
// Generally provides discovery, negotiation, and no-dep calls.
// TODO The polymorphic calls probably deserve their own interface.
type ClientAccessFactory interface {
	DiscoveryClientFactory

	// ClientSet gives you back an internal, generated clientset
	ClientSet() (*internalclientset.Clientset, error)
	// Returns a RESTClient for accessing Kubernetes resources or an error.
	RESTClient() (*restclient.RESTClient, error)
	// Returns a client.Config for accessing the Kubernetes server.
	ClientConfig() (*restclient.Config, error)

	// TODO this should probably be removed and collapsed into whatever we want to use long term
	// probably returning a restclient for a version and leaving contruction up to someone else
	FederationClientSetForVersion(version *unversioned.GroupVersion) (fedclientset.Interface, error)
	// TODO remove this should be rolled into restclient with the right version
	FederationClientForVersion(version *unversioned.GroupVersion) (*restclient.RESTClient, error)
	// TODO remove.  This should be rolled into `ClientSet`
	ClientSetForVersion(requiredVersion *unversioned.GroupVersion) (*internalclientset.Clientset, error)
	// TODO remove.  This should be rolled into `ClientConfig`
	ClientConfigForVersion(requiredVersion *unversioned.GroupVersion) (*restclient.Config, error)

	// Returns interfaces for decoding objects - if toInternal is set, decoded objects will be converted
	// into their internal form (if possible). Eventually the internal form will be removed as an option,
	// and only versioned objects will be returned.
	Decoder(toInternal bool) runtime.Decoder
	// Returns an encoder capable of encoding a provided object into JSON in the default desired version.
	JSONEncoder() runtime.Encoder

	// UpdatePodSpecForObject will call the provided function on the pod spec this object supports,
	// return false if no pod spec is supported, or return an error.
	UpdatePodSpecForObject(obj runtime.Object, fn func(*api.PodSpec) error) (bool, error)

	// MapBasedSelectorForObject returns the map-based selector associated with the provided object. If a
	// new set-based selector is provided, an error is returned if the selector cannot be converted to a
	// map-based selector
	MapBasedSelectorForObject(object runtime.Object) (string, error)
	// PortsForObject returns the ports associated with the provided object
	PortsForObject(object runtime.Object) ([]string, error)
	// ProtocolsForObject returns the <port, protocol> mapping associated with the provided object
	ProtocolsForObject(object runtime.Object) (map[string]string, error)
	// LabelsForObject returns the labels associated with the provided object
	LabelsForObject(object runtime.Object) (map[string]string, error)

	// Returns internal flagset
	FlagSet() *pflag.FlagSet
	// Command will stringify and return all environment arguments ie. a command run by a client
	// using the factory.
	Command() string
	// BindFlags adds any flags that are common to all kubectl sub commands.
	BindFlags(flags *pflag.FlagSet)
	// BindExternalFlags adds any flags defined by external projects (not part of pflags)
	BindExternalFlags(flags *pflag.FlagSet)

	DefaultResourceFilterOptions(cmd *cobra.Command, withNamespace bool) *kubectl.PrintOptions
	// DefaultResourceFilterFunc returns a collection of FilterFuncs suitable for filtering specific resource types.
	DefaultResourceFilterFunc() kubectl.Filters

	// SuggestedPodTemplateResources returns a list of resource types that declare a pod template
	SuggestedPodTemplateResources() []unversioned.GroupResource

	// Returns a Printer for formatting objects of the given type or an error.
	Printer(mapping *meta.RESTMapping, options kubectl.PrintOptions) (kubectl.ResourcePrinter, error)
	// Pauser marks the object in the info as paused ie. it will not be reconciled by its controller.
	Pauser(info *resource.Info) (bool, error)
	// Resumer resumes a paused object inside the info ie. it will be reconciled by its controller.
	Resumer(info *resource.Info) (bool, error)

	// ResolveImage resolves the image names. For kubernetes this function is just
	// passthrough but it allows to perform more sophisticated image name resolving for
	// third-party vendors.
	ResolveImage(imageName string) (string, error)

	// Returns the default namespace to use in cases where no
	// other namespace is specified and whether the namespace was
	// overridden.
	DefaultNamespace() (string, bool, error)
	// Generators returns the generators for the provided command
	Generators(cmdName string) map[string]kubectl.Generator
	// Check whether the kind of resources could be exposed
	CanBeExposed(kind unversioned.GroupKind) error
	// Check whether the kind of resources could be autoscaled
	CanBeAutoscaled(kind unversioned.GroupKind) error

	// EditorEnvs returns a group of environment variables that the edit command
	// can range over in order to determine if the user has specified an editor
	// of their choice.
	EditorEnvs() []string

	// PrintObjectSpecificMessage prints object-specific messages on the provided writer
	PrintObjectSpecificMessage(obj runtime.Object, out io.Writer)
}

// ObjectMappingFactory holds the second level of factory methods.  These functions depend upon ClientAccessFactory methods.
// Generally they provide object typing and functions that build requests based on the negotiated clients.
type ObjectMappingFactory interface {
	// Returns interfaces for dealing with arbitrary runtime.Objects.
	Object() (meta.RESTMapper, runtime.ObjectTyper)
	// Returns interfaces for dealing with arbitrary
	// runtime.Unstructured. This performs API calls to discover types.
	UnstructuredObject() (meta.RESTMapper, runtime.ObjectTyper, error)
	// Returns a RESTClient for working with the specified RESTMapping or an error. This is intended
	// for working with arbitrary resources and is not guaranteed to point to a Kubernetes APIServer.
	ClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error)
	// Returns a RESTClient for working with Unstructured objects.
	UnstructuredClientForMapping(mapping *meta.RESTMapping) (resource.RESTClient, error)
	// Returns a Describer for displaying the specified RESTMapping type or an error.
	Describer(mapping *meta.RESTMapping) (kubectl.Describer, error)

	// LogsForObject returns a request for the logs associated with the provided object
	LogsForObject(object, options runtime.Object) (*restclient.Request, error)
	// Returns a Scaler for changing the size of the specified RESTMapping type or an error
	Scaler(mapping *meta.RESTMapping) (kubectl.Scaler, error)
	// Returns a Reaper for gracefully shutting down resources.
	Reaper(mapping *meta.RESTMapping) (kubectl.Reaper, error)
	// Returns a HistoryViewer for viewing change history
	HistoryViewer(mapping *meta.RESTMapping) (kubectl.HistoryViewer, error)
	// Returns a Rollbacker for changing the rollback version of the specified RESTMapping type or an error
	Rollbacker(mapping *meta.RESTMapping) (kubectl.Rollbacker, error)
	// Returns a StatusViewer for printing rollout status.
	StatusViewer(mapping *meta.RESTMapping) (kubectl.StatusViewer, error)

	// AttachablePodForObject returns the pod to which to attach given an object.
	AttachablePodForObject(object runtime.Object) (*api.Pod, error)

	// PrinterForMapping returns a printer suitable for displaying the provided resource type.
	// Requires that printer flags have been added to cmd (see AddPrinterFlags).
	PrinterForMapping(cmd *cobra.Command, mapping *meta.RESTMapping, withNamespace bool) (kubectl.ResourcePrinter, error)

	// Returns a schema that can validate objects stored on disk.
	Validator(validate bool, cacheDir string) (validation.Schema, error)
	// SwaggerSchema returns the schema declaration for the provided group version kind.
	SwaggerSchema(unversioned.GroupVersionKind) (*swagger.ApiDeclaration, error)
}

// BuilderFactory holds the second level of factory methods.  These functions depend upon ObjectMappingFactory and ClientAccessFactory methods.
// Generally they depend upon client mapper functions
type BuilderFactory interface {
	// PrintObject prints an api object given command line flags to modify the output format
	PrintObject(cmd *cobra.Command, mapper meta.RESTMapper, obj runtime.Object, out io.Writer) error
	// One stop shopping for a Builder
	NewBuilder() *resource.Builder
}

func getGroupVersionKinds(gvks []unversioned.GroupVersionKind, group string) []unversioned.GroupVersionKind {
	result := []unversioned.GroupVersionKind{}
	for ix := range gvks {
		if gvks[ix].Group == group {
			result = append(result, gvks[ix])
		}
	}
	return result
}

func makeInterfacesFor(versionList []unversioned.GroupVersion) func(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
	accessor := meta.NewAccessor()
	return func(version unversioned.GroupVersion) (*meta.VersionInterfaces, error) {
		for ix := range versionList {
			if versionList[ix].String() == version.String() {
				return &meta.VersionInterfaces{
					ObjectConvertor:  thirdpartyresourcedata.NewThirdPartyObjectConverter(api.Scheme),
					MetadataAccessor: accessor,
				}, nil
			}
		}
		return nil, fmt.Errorf("unsupported storage version: %s (valid: %v)", version, versionList)
	}
}

type factory struct {
	ClientAccessFactory
	ObjectMappingFactory
	BuilderFactory
}

// NewFactory creates a factory with the default Kubernetes resources defined
// if optionalClientConfig is nil, then flags will be bound to a new clientcmd.ClientConfig.
// if optionalClientConfig is not nil, then this factory will make use of it.
func NewFactory(optionalClientConfig clientcmd.ClientConfig) Factory {
	clientAccessFactory := NewClientAccessFactory(optionalClientConfig)
	objectMappingFactory := NewObjectMappingFactory(clientAccessFactory)
	builderFactory := NewBuilderFactory(clientAccessFactory, objectMappingFactory)

	return &factory{
		ClientAccessFactory:  clientAccessFactory,
		ObjectMappingFactory: objectMappingFactory,
		BuilderFactory:       builderFactory,
	}
}

// GetFirstPod returns a pod matching the namespace and label selector
// and the number of all pods that match the label selector.
func GetFirstPod(client coreclient.PodsGetter, namespace string, selector labels.Selector, timeout time.Duration, sortBy func([]*api.Pod) sort.Interface) (*api.Pod, int, error) {
	options := api.ListOptions{LabelSelector: selector}

	podList, err := client.Pods(namespace).List(options)
	if err != nil {
		return nil, 0, err
	}
	pods := []*api.Pod{}
	for i := range podList.Items {
		pod := podList.Items[i]
		pods = append(pods, &pod)
	}
	if len(pods) > 0 {
		sort.Sort(sortBy(pods))
		return pods[0], len(podList.Items), nil
	}

	// Watch until we observe a pod
	options.ResourceVersion = podList.ResourceVersion
	w, err := client.Pods(namespace).Watch(options)
	if err != nil {
		return nil, 0, err
	}
	defer w.Stop()

	condition := func(event watch.Event) (bool, error) {
		return event.Type == watch.Added || event.Type == watch.Modified, nil
	}
	event, err := watch.Until(timeout, w, condition)
	if err != nil {
		return nil, 0, err
	}
	pod, ok := event.Object.(*api.Pod)
	if !ok {
		return nil, 0, fmt.Errorf("%#v is not a pod event", event)
	}
	return pod, 1, nil
}

func makePortsString(ports []api.ServicePort, useNodePort bool) string {
	pieces := make([]string, len(ports))
	for ix := range ports {
		var port int32
		if useNodePort {
			port = ports[ix].NodePort
		} else {
			port = ports[ix].Port
		}
		pieces[ix] = fmt.Sprintf("%s:%d", strings.ToLower(string(ports[ix].Protocol)), port)
	}
	return strings.Join(pieces, ",")
}

func getPorts(spec api.PodSpec) []string {
	result := []string{}
	for _, container := range spec.Containers {
		for _, port := range container.Ports {
			result = append(result, strconv.Itoa(int(port.ContainerPort)))
		}
	}
	return result
}

func getProtocols(spec api.PodSpec) map[string]string {
	result := make(map[string]string)
	for _, container := range spec.Containers {
		for _, port := range container.Ports {
			result[strconv.Itoa(int(port.ContainerPort))] = string(port.Protocol)
		}
	}
	return result
}

// Extracts the ports exposed by a service from the given service spec.
func getServicePorts(spec api.ServiceSpec) []string {
	result := []string{}
	for _, servicePort := range spec.Ports {
		result = append(result, strconv.Itoa(int(servicePort.Port)))
	}
	return result
}

// Extracts the protocols exposed by a service from the given service spec.
func getServiceProtocols(spec api.ServiceSpec) map[string]string {
	result := make(map[string]string)
	for _, servicePort := range spec.Ports {
		result[strconv.Itoa(int(servicePort.Port))] = string(servicePort.Protocol)
	}
	return result
}

type clientSwaggerSchema struct {
	c        restclient.Interface
	cacheDir string
}

const schemaFileName = "schema.json"

type schemaClient interface {
	Get() *restclient.Request
}

func recursiveSplit(dir string) []string {
	parent, file := path.Split(dir)
	if len(parent) == 0 {
		return []string{file}
	}
	return append(recursiveSplit(parent[:len(parent)-1]), file)
}

func substituteUserHome(dir string) (string, error) {
	if len(dir) == 0 || dir[0] != '~' {
		return dir, nil
	}
	parts := recursiveSplit(dir)
	if len(parts[0]) == 1 {
		parts[0] = os.Getenv("HOME")
	} else {
		usr, err := user.Lookup(parts[0][1:])
		if err != nil {
			return "", err
		}
		parts[0] = usr.HomeDir
	}
	return path.Join(parts...), nil
}

func writeSchemaFile(schemaData []byte, cacheDir, cacheFile, prefix, groupVersion string) error {
	if err := os.MkdirAll(path.Join(cacheDir, prefix, groupVersion), 0755); err != nil {
		return err
	}
	tmpFile, err := ioutil.TempFile(cacheDir, "schema")
	if err != nil {
		// If we can't write, keep going.
		if os.IsPermission(err) {
			return nil
		}
		return err
	}
	if _, err := io.Copy(tmpFile, bytes.NewBuffer(schemaData)); err != nil {
		return err
	}
	if err := os.Link(tmpFile.Name(), cacheFile); err != nil {
		// If we can't write due to file existing, or permission problems, keep going.
		if os.IsExist(err) || os.IsPermission(err) {
			return nil
		}
		return err
	}
	return nil
}

func getSchemaAndValidate(c schemaClient, data []byte, prefix, groupVersion, cacheDir string, delegate validation.Schema) (err error) {
	var schemaData []byte
	var firstSeen bool
	fullDir, err := substituteUserHome(cacheDir)
	if err != nil {
		return err
	}
	cacheFile := path.Join(fullDir, prefix, groupVersion, schemaFileName)

	if len(cacheDir) != 0 {
		if schemaData, err = ioutil.ReadFile(cacheFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if schemaData == nil {
		firstSeen = true
		schemaData, err = downloadSchemaAndStore(c, cacheDir, fullDir, cacheFile, prefix, groupVersion)
		if err != nil {
			return err
		}
	}
	schema, err := validation.NewSwaggerSchemaFromBytes(schemaData, delegate)
	if err != nil {
		return err
	}
	err = schema.ValidateBytes(data)
	if _, ok := err.(validation.TypeNotFoundError); ok && !firstSeen {
		// As a temporary hack, kubectl would re-get the schema if validation
		// fails for type not found reason.
		// TODO: runtime-config settings needs to make into the file's name
		schemaData, err = downloadSchemaAndStore(c, cacheDir, fullDir, cacheFile, prefix, groupVersion)
		if err != nil {
			return err
		}
		schema, err := validation.NewSwaggerSchemaFromBytes(schemaData, delegate)
		if err != nil {
			return err
		}
		return schema.ValidateBytes(data)
	}

	return err
}

// Download swagger schema from apiserver and store it to file.
func downloadSchemaAndStore(c schemaClient, cacheDir, fullDir, cacheFile, prefix, groupVersion string) (schemaData []byte, err error) {
	schemaData, err = c.Get().
		AbsPath("/swaggerapi", prefix, groupVersion).
		Do().
		Raw()
	if err != nil {
		return
	}
	if len(cacheDir) != 0 {
		if err = writeSchemaFile(schemaData, fullDir, cacheFile, prefix, groupVersion); err != nil {
			return
		}
	}
	return
}

func (c *clientSwaggerSchema) ValidateBytes(data []byte) error {
	gvk, err := json.DefaultMetaFactory.Interpret(data)
	if err != nil {
		return err
	}
	if ok := registered.IsEnabledVersion(gvk.GroupVersion()); !ok {
		// if we don't have this in our scheme, just skip validation because its an object we don't recognize
		return nil
	}

	switch gvk.Group {
	case api.GroupName:
		return getSchemaAndValidate(c.c, data, "api", gvk.GroupVersion().String(), c.cacheDir, c)

	default:
		return getSchemaAndValidate(c.c, data, "apis/", gvk.GroupVersion().String(), c.cacheDir, c)
	}
}
