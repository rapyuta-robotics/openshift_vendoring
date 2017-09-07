package secrets

import (
	"reflect"

	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/util/sets"
)

type KnownSecretType struct {
	Type             kapi.SecretType
	RequiredContents sets.String
}

func (ks KnownSecretType) Matches(secretContent map[string][]byte) bool {
	if secretContent == nil {
		return false
	}
	secretKeys := sets.StringKeySet(secretContent)
	return reflect.DeepEqual(ks.RequiredContents.List(), secretKeys.List())
}

var (
	KnownSecretTypes = []KnownSecretType{
		{kapi.SecretTypeDockercfg, sets.NewString(kapi.DockerConfigKey)},
		{kapi.SecretTypeDockerConfigJson, sets.NewString(kapi.DockerConfigJsonKey)},
	}
)
