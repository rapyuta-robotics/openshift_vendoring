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

package meta

import (
	"fmt"
	"reflect"

	"github.com/openshift/k8s.io/client-go/pkg/api/meta/metatypes"
	"github.com/openshift/k8s.io/client-go/pkg/api/unversioned"
	"github.com/openshift/k8s.io/client-go/pkg/conversion"
	"github.com/openshift/k8s.io/client-go/pkg/runtime"
	"github.com/openshift/k8s.io/client-go/pkg/types"

	"github.com/golang/glog"
)

// errNotList is returned when an object implements the Object style interfaces but not the List style
// interfaces.
var errNotList = fmt.Errorf("object does not implement the List interfaces")

// ListAccessor returns a List interface for the provided object or an error if the object does
// not provide List.
// IMPORTANT: Objects are a superset of lists, so all Objects return List metadata. Do not use this
// check to determine whether an object *is* a List.
// TODO: return bool instead of error
func ListAccessor(obj interface{}) (List, error) {
	switch t := obj.(type) {
	case List:
		return t, nil
	case unversioned.List:
		return t, nil
	case ListMetaAccessor:
		if m := t.GetListMeta(); m != nil {
			return m, nil
		}
		return nil, errNotList
	case unversioned.ListMetaAccessor:
		if m := t.GetListMeta(); m != nil {
			return m, nil
		}
		return nil, errNotList
	case Object:
		return t, nil
	case ObjectMetaAccessor:
		if m := t.GetObjectMeta(); m != nil {
			return m, nil
		}
		return nil, errNotList
	default:
		return nil, errNotList
	}
}

// errNotObject is returned when an object implements the List style interfaces but not the Object style
// interfaces.
var errNotObject = fmt.Errorf("object does not implement the Object interfaces")

// Accessor takes an arbitrary object pointer and returns meta.Interface.
// obj must be a pointer to an API type. An error is returned if the minimum
// required fields are missing. Fields that are not required return the default
// value and are a no-op if set.
// TODO: return bool instead of error
func Accessor(obj interface{}) (Object, error) {
	switch t := obj.(type) {
	case Object:
		return t, nil
	case ObjectMetaAccessor:
		if m := t.GetObjectMeta(); m != nil {
			return m, nil
		}
		return nil, errNotObject
	case List, unversioned.List, ListMetaAccessor, unversioned.ListMetaAccessor:
		return nil, errNotObject
	default:
		return nil, errNotObject
	}
}

// TypeAccessor returns an interface that allows retrieving and modifying the APIVersion
// and Kind of an in-memory internal object.
// TODO: this interface is used to test code that does not have ObjectMeta or ListMeta
// in round tripping (objects which can use apiVersion/kind, but do not fit the Kube
// api conventions).
func TypeAccessor(obj interface{}) (Type, error) {
	if typed, ok := obj.(runtime.Object); ok {
		return objectAccessor{typed}, nil
	}
	v, err := conversion.EnforcePtr(obj)
	if err != nil {
		return nil, err
	}
	t := v.Type()
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, but got %v: %v (%#v)", v.Kind(), t, v.Interface())
	}

	typeMeta := v.FieldByName("TypeMeta")
	if !typeMeta.IsValid() {
		return nil, fmt.Errorf("struct %v lacks embedded TypeMeta type", t)
	}
	a := &genericAccessor{}
	if err := extractFromTypeMeta(typeMeta, a); err != nil {
		return nil, fmt.Errorf("unable to find type fields on %#v: %v", typeMeta, err)
	}
	return a, nil
}

type objectAccessor struct {
	runtime.Object
}

func (obj objectAccessor) GetKind() string {
	return obj.GetObjectKind().GroupVersionKind().Kind
}

func (obj objectAccessor) SetKind(kind string) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	gvk.Kind = kind
	obj.GetObjectKind().SetGroupVersionKind(gvk)
}

func (obj objectAccessor) GetAPIVersion() string {
	return obj.GetObjectKind().GroupVersionKind().GroupVersion().String()
}

func (obj objectAccessor) SetAPIVersion(version string) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	gv, err := unversioned.ParseGroupVersion(version)
	if err != nil {
		gv = unversioned.GroupVersion{Version: version}
	}
	gvk.Group, gvk.Version = gv.Group, gv.Version
	obj.GetObjectKind().SetGroupVersionKind(gvk)
}

// NewAccessor returns a MetadataAccessor that can retrieve
// or manipulate resource version on objects derived from core API
// metadata concepts.
func NewAccessor() MetadataAccessor {
	return resourceAccessor{}
}

// resourceAccessor implements ResourceVersioner and SelfLinker.
type resourceAccessor struct{}

func (resourceAccessor) Kind(obj runtime.Object) (string, error) {
	return objectAccessor{obj}.GetKind(), nil
}

func (resourceAccessor) SetKind(obj runtime.Object, kind string) error {
	objectAccessor{obj}.SetKind(kind)
	return nil
}

func (resourceAccessor) APIVersion(obj runtime.Object) (string, error) {
	return objectAccessor{obj}.GetAPIVersion(), nil
}

func (resourceAccessor) SetAPIVersion(obj runtime.Object, version string) error {
	objectAccessor{obj}.SetAPIVersion(version)
	return nil
}

func (resourceAccessor) Namespace(obj runtime.Object) (string, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetNamespace(), nil
}

func (resourceAccessor) SetNamespace(obj runtime.Object, namespace string) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetNamespace(namespace)
	return nil
}

func (resourceAccessor) Name(obj runtime.Object) (string, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetName(), nil
}

func (resourceAccessor) SetName(obj runtime.Object, name string) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetName(name)
	return nil
}

func (resourceAccessor) GenerateName(obj runtime.Object) (string, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetGenerateName(), nil
}

func (resourceAccessor) SetGenerateName(obj runtime.Object, name string) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetGenerateName(name)
	return nil
}

func (resourceAccessor) UID(obj runtime.Object) (types.UID, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetUID(), nil
}

func (resourceAccessor) SetUID(obj runtime.Object, uid types.UID) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetUID(uid)
	return nil
}

func (resourceAccessor) SelfLink(obj runtime.Object) (string, error) {
	accessor, err := ListAccessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetSelfLink(), nil
}

func (resourceAccessor) SetSelfLink(obj runtime.Object, selfLink string) error {
	accessor, err := ListAccessor(obj)
	if err != nil {
		return err
	}
	accessor.SetSelfLink(selfLink)
	return nil
}

func (resourceAccessor) Labels(obj runtime.Object) (map[string]string, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return nil, err
	}
	return accessor.GetLabels(), nil
}

func (resourceAccessor) SetLabels(obj runtime.Object, labels map[string]string) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetLabels(labels)
	return nil
}

func (resourceAccessor) Annotations(obj runtime.Object) (map[string]string, error) {
	accessor, err := Accessor(obj)
	if err != nil {
		return nil, err
	}
	return accessor.GetAnnotations(), nil
}

func (resourceAccessor) SetAnnotations(obj runtime.Object, annotations map[string]string) error {
	accessor, err := Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetAnnotations(annotations)
	return nil
}

func (resourceAccessor) ResourceVersion(obj runtime.Object) (string, error) {
	accessor, err := ListAccessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetResourceVersion(), nil
}

func (resourceAccessor) SetResourceVersion(obj runtime.Object, version string) error {
	accessor, err := ListAccessor(obj)
	if err != nil {
		return err
	}
	accessor.SetResourceVersion(version)
	return nil
}

// extractFromOwnerReference extracts v to o. v is the OwnerReferences field of an object.
func extractFromOwnerReference(v reflect.Value, o *metatypes.OwnerReference) error {
	if err := runtime.Field(v, "APIVersion", &o.APIVersion); err != nil {
		return err
	}
	if err := runtime.Field(v, "Kind", &o.Kind); err != nil {
		return err
	}
	if err := runtime.Field(v, "Name", &o.Name); err != nil {
		return err
	}
	if err := runtime.Field(v, "UID", &o.UID); err != nil {
		return err
	}
	var controllerPtr *bool
	if err := runtime.Field(v, "Controller", &controllerPtr); err != nil {
		return err
	}
	if controllerPtr != nil {
		controller := *controllerPtr
		o.Controller = &controller
	}
	return nil
}

// setOwnerReference sets v to o. v is the OwnerReferences field of an object.
func setOwnerReference(v reflect.Value, o *metatypes.OwnerReference) error {
	if err := runtime.SetField(o.APIVersion, v, "APIVersion"); err != nil {
		return err
	}
	if err := runtime.SetField(o.Kind, v, "Kind"); err != nil {
		return err
	}
	if err := runtime.SetField(o.Name, v, "Name"); err != nil {
		return err
	}
	if err := runtime.SetField(o.UID, v, "UID"); err != nil {
		return err
	}
	if o.Controller != nil {
		controller := *(o.Controller)
		if err := runtime.SetField(&controller, v, "Controller"); err != nil {
			return err
		}
	}
	return nil
}

// genericAccessor contains pointers to strings that can modify an arbitrary
// struct and implements the Accessor interface.
type genericAccessor struct {
	namespace         *string
	name              *string
	generateName      *string
	uid               *types.UID
	apiVersion        *string
	kind              *string
	resourceVersion   *string
	selfLink          *string
	creationTimestamp *unversioned.Time
	deletionTimestamp **unversioned.Time
	labels            *map[string]string
	annotations       *map[string]string
	ownerReferences   reflect.Value
	finalizers        *[]string
}

func (a genericAccessor) GetNamespace() string {
	if a.namespace == nil {
		return ""
	}
	return *a.namespace
}

func (a genericAccessor) SetNamespace(namespace string) {
	if a.namespace == nil {
		return
	}
	*a.namespace = namespace
}

func (a genericAccessor) GetName() string {
	if a.name == nil {
		return ""
	}
	return *a.name
}

func (a genericAccessor) SetName(name string) {
	if a.name == nil {
		return
	}
	*a.name = name
}

func (a genericAccessor) GetGenerateName() string {
	if a.generateName == nil {
		return ""
	}
	return *a.generateName
}

func (a genericAccessor) SetGenerateName(generateName string) {
	if a.generateName == nil {
		return
	}
	*a.generateName = generateName
}

func (a genericAccessor) GetUID() types.UID {
	if a.uid == nil {
		return ""
	}
	return *a.uid
}

func (a genericAccessor) SetUID(uid types.UID) {
	if a.uid == nil {
		return
	}
	*a.uid = uid
}

func (a genericAccessor) GetAPIVersion() string {
	return *a.apiVersion
}

func (a genericAccessor) SetAPIVersion(version string) {
	*a.apiVersion = version
}

func (a genericAccessor) GetKind() string {
	return *a.kind
}

func (a genericAccessor) SetKind(kind string) {
	*a.kind = kind
}

func (a genericAccessor) GetResourceVersion() string {
	return *a.resourceVersion
}

func (a genericAccessor) SetResourceVersion(version string) {
	*a.resourceVersion = version
}

func (a genericAccessor) GetSelfLink() string {
	return *a.selfLink
}

func (a genericAccessor) SetSelfLink(selfLink string) {
	*a.selfLink = selfLink
}

func (a genericAccessor) GetCreationTimestamp() unversioned.Time {
	return *a.creationTimestamp
}

func (a genericAccessor) SetCreationTimestamp(timestamp unversioned.Time) {
	*a.creationTimestamp = timestamp
}

func (a genericAccessor) GetDeletionTimestamp() *unversioned.Time {
	return *a.deletionTimestamp
}

func (a genericAccessor) SetDeletionTimestamp(timestamp *unversioned.Time) {
	*a.deletionTimestamp = timestamp
}

func (a genericAccessor) GetLabels() map[string]string {
	if a.labels == nil {
		return nil
	}
	return *a.labels
}

func (a genericAccessor) SetLabels(labels map[string]string) {
	*a.labels = labels
}

func (a genericAccessor) GetAnnotations() map[string]string {
	if a.annotations == nil {
		return nil
	}
	return *a.annotations
}

func (a genericAccessor) SetAnnotations(annotations map[string]string) {
	if a.annotations == nil {
		emptyAnnotations := make(map[string]string)
		a.annotations = &emptyAnnotations
	}
	*a.annotations = annotations
}

func (a genericAccessor) GetFinalizers() []string {
	if a.finalizers == nil {
		return nil
	}
	return *a.finalizers
}

func (a genericAccessor) SetFinalizers(finalizers []string) {
	*a.finalizers = finalizers
}

func (a genericAccessor) GetOwnerReferences() []metatypes.OwnerReference {
	var ret []metatypes.OwnerReference
	s := a.ownerReferences
	if s.Kind() != reflect.Ptr || s.Elem().Kind() != reflect.Slice {
		glog.Errorf("expect %v to be a pointer to slice", s)
		return ret
	}
	s = s.Elem()
	// Set the capacity to one element greater to avoid copy if the caller later append an element.
	ret = make([]metatypes.OwnerReference, s.Len(), s.Len()+1)
	for i := 0; i < s.Len(); i++ {
		if err := extractFromOwnerReference(s.Index(i), &ret[i]); err != nil {
			glog.Errorf("extractFromOwnerReference failed: %v", err)
			return ret
		}
	}
	return ret
}

func (a genericAccessor) SetOwnerReferences(references []metatypes.OwnerReference) {
	s := a.ownerReferences
	if s.Kind() != reflect.Ptr || s.Elem().Kind() != reflect.Slice {
		glog.Errorf("expect %v to be a pointer to slice", s)
	}
	s = s.Elem()
	newReferences := reflect.MakeSlice(s.Type(), len(references), len(references))
	for i := 0; i < len(references); i++ {
		if err := setOwnerReference(newReferences.Index(i), &references[i]); err != nil {
			glog.Errorf("setOwnerReference failed: %v", err)
			return
		}
	}
	s.Set(newReferences)
}

// extractFromTypeMeta extracts pointers to version and kind fields from an object
func extractFromTypeMeta(v reflect.Value, a *genericAccessor) error {
	if err := runtime.FieldPtr(v, "APIVersion", &a.apiVersion); err != nil {
		return err
	}
	if err := runtime.FieldPtr(v, "Kind", &a.kind); err != nil {
		return err
	}
	return nil
}
