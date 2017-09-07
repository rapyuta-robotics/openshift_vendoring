package v1

import "github.com/openshift/kubernetes/pkg/runtime"

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	RegisterDefaults(scheme)
	return nil
}
