package restoptions

import (
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	"github.com/openshift/kubernetes/pkg/registry/generic"
)

type Getter interface {
	GetRESTOptions(resource unversioned.GroupResource) (generic.RESTOptions, error)
}
