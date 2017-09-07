package router

import (
	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/util/sets"
	"github.com/openshift/kubernetes/pkg/watch"

	routeapi "github.com/openshift/origin/pkg/route/api"
)

// Plugin is the interface the router controller dispatches watch events
// for the Routes and Endpoints resources to.
type Plugin interface {
	HandleRoute(watch.EventType, *routeapi.Route) error
	HandleEndpoints(watch.EventType, *kapi.Endpoints) error
	// If sent, filter the list of accepted routes and endpoints to this set
	HandleNamespaces(namespaces sets.String) error
	HandleNode(watch.EventType, *kapi.Node) error
	Commit() error
}
