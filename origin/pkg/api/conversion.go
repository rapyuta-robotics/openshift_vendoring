package api

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/conversion"
	"github.com/openshift/kubernetes/pkg/runtime"

	"github.com/openshift/origin/pkg/api/extension"
)

// Convert_runtime_Object_To_runtime_RawExtension ensures an object is converted to the destination version of the conversion.
func Convert_runtime_Object_To_runtime_RawExtension(in *runtime.Object, out *runtime.RawExtension, s conversion.Scope) error {
	return extension.Convert_runtime_Object_To_runtime_RawExtension(kapi.Scheme, in, out, s)
}

// Convert_runtime_RawExtension_To_runtime_Object ensures an object is converted to the destination version of the conversion.
func Convert_runtime_RawExtension_To_runtime_Object(in *runtime.RawExtension, out *runtime.Object, s conversion.Scope) error {
	return extension.Convert_runtime_RawExtension_To_runtime_Object(kapi.Scheme, in, out, s)
}
