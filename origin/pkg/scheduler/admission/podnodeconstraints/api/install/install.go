package install

import (
	"github.com/golang/glog"

	"github.com/openshift/kubernetes/pkg/api/meta"
	"github.com/openshift/kubernetes/pkg/api/unversioned"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/scheduler/admission/podnodeconstraints/api"
	"github.com/openshift/origin/pkg/scheduler/admission/podnodeconstraints/api/v1"
)

const importPrefix = "github.com/openshift/origin/pkg/scheduler/admission/podnodeconstraints/api"

var accessor = meta.NewAccessor()

// availableVersions lists all known external versions for this group from most preferred to least preferred
var availableVersions = []unversioned.GroupVersion{v1.SchemeGroupVersion}

func init() {
	if err := enableVersions(availableVersions); err != nil {
		panic(err)
	}
}

// TODO: enableVersions should be centralized rather than spread in each API
// group.
// We can combine registered.RegisterVersions, registered.EnableVersions and
// registered.RegisterGroup once we have moved enableVersions there.
func enableVersions(externalVersions []unversioned.GroupVersion) error {
	addVersionsToScheme(externalVersions...)
	return nil
}

func addVersionsToScheme(externalVersions ...unversioned.GroupVersion) {
	// add the internal version to Scheme
	api.AddToScheme(configapi.Scheme)
	// add the enabled external versions to Scheme
	for _, v := range externalVersions {
		switch v {
		case v1.SchemeGroupVersion:
			v1.AddToScheme(configapi.Scheme)

		default:
			glog.Errorf("Version %s is not known, so it will not be added to the Scheme.", v)
			continue
		}
	}
}
