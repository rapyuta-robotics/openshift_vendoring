package ipfailover

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
)

type IPFailoverConfiguratorPlugin interface {
	Generate() (*kapi.List, error)
}
