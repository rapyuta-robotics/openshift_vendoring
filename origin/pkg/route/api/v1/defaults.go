package v1

import "github.com/openshift/kubernetes/pkg/runtime"

// If adding or changing route defaults, updates may be required to
// pkg/router/controller/controller.go to ensure the routes generated from
// ingress resources will match routes created via the api.

func SetDefaults_RouteSpec(obj *RouteSpec) {
	if len(obj.WildcardPolicy) == 0 {
		obj.WildcardPolicy = WildcardPolicyNone
	}
}

func SetDefaults_RouteTargetReference(obj *RouteTargetReference) {
	if len(obj.Kind) == 0 {
		obj.Kind = "Service"
	}
	if obj.Weight == nil {
		obj.Weight = new(int32)
		*obj.Weight = 100
	}
}

func SetDefaults_TLSConfig(obj *TLSConfig) {
	if len(obj.Termination) == 0 && len(obj.DestinationCACertificate) == 0 {
		obj.Termination = TLSTerminationEdge
	}
	switch obj.Termination {
	case TLSTerminationType("Reencrypt"):
		obj.Termination = TLSTerminationReencrypt
	case TLSTerminationType("Edge"):
		obj.Termination = TLSTerminationEdge
	case TLSTerminationType("Passthrough"):
		obj.Termination = TLSTerminationPassthrough
	}
}

func SetDefaults_RouteIngress(obj *RouteIngress) {
	if len(obj.WildcardPolicy) == 0 {
		obj.WildcardPolicy = WildcardPolicyNone
	}
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	RegisterDefaults(scheme)
	return scheme.AddDefaultingFuncs(
		SetDefaults_RouteSpec,
		SetDefaults_RouteTargetReference,
		SetDefaults_TLSConfig,
		SetDefaults_RouteIngress,
	)
}
