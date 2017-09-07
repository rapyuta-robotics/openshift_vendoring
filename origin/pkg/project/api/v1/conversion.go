package v1

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/registry/core/namespace"
	"github.com/openshift/kubernetes/pkg/runtime"

	oapi "github.com/openshift/origin/pkg/api"
)

func addConversionFuncs(scheme *runtime.Scheme) error {
	return scheme.AddFieldLabelConversionFunc("v1", "Project",
		oapi.GetFieldLabelConversionFunc(namespace.NamespaceToSelectableFields(&kapi.Namespace{}), nil),
	)
}
