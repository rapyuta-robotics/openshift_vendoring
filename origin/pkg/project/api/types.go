package api

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
)

// ProjectList is a list of Project objects.
type ProjectList struct {
	unversioned.TypeMeta
	unversioned.ListMeta
	Items []Project
}

const (
	// These are internal finalizer values to Origin
	FinalizerOrigin kapi.FinalizerName = "openshift.io/origin"
)

// ProjectSpec describes the attributes on a Project
type ProjectSpec struct {
	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []kapi.FinalizerName
}

// ProjectStatus is information about the current status of a Project
type ProjectStatus struct {
	Phase kapi.NamespacePhase
}

// +genclient=true
// +nonNamespaced=true

// Project is a logical top-level container for a set of origin resources
type Project struct {
	unversioned.TypeMeta
	kapi.ObjectMeta

	Spec   ProjectSpec
	Status ProjectStatus
}

type ProjectRequest struct {
	unversioned.TypeMeta
	kapi.ObjectMeta
	DisplayName string
	Description string
}

// These constants represent annotations keys affixed to projects
const (
	// ProjectNodeSelector is an annotation that holds the node selector;
	// the node selector annotation determines which nodes will have pods from this project scheduled to them
	ProjectNodeSelector = "openshift.io/node-selector"
	// ProjectRequester is the username that requested a given project.  Its not guaranteed to be present,
	// but it is set by the default project template.
	ProjectRequester = "openshift.io/requester"
)
