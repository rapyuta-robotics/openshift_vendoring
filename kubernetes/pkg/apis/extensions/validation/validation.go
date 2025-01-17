/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validation

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/unversioned"
	unversionedvalidation "github.com/openshift/kubernetes/pkg/api/unversioned/validation"
	apivalidation "github.com/openshift/kubernetes/pkg/api/validation"
	"github.com/openshift/kubernetes/pkg/apis/extensions"
	"github.com/openshift/kubernetes/pkg/labels"
	"github.com/openshift/kubernetes/pkg/security/apparmor"
	"github.com/openshift/kubernetes/pkg/security/podsecuritypolicy/seccomp"
	psputil "github.com/openshift/kubernetes/pkg/security/podsecuritypolicy/util"
	"github.com/openshift/kubernetes/pkg/util/intstr"
	"github.com/openshift/kubernetes/pkg/util/sets"
	"github.com/openshift/kubernetes/pkg/util/validation"
	"github.com/openshift/kubernetes/pkg/util/validation/field"
)

func ValidateThirdPartyResourceUpdate(update, old *extensions.ThirdPartyResource) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateThirdPartyResource(update)...)
	return allErrs
}

func ValidateThirdPartyResourceName(name string, prefix bool) []string {
	// Make sure it's a valid DNS subdomain
	if msgs := apivalidation.NameIsDNSSubdomain(name, prefix); len(msgs) != 0 {
		return msgs
	}

	// Make sure it's at least three segments (kind + two-segment group name)
	if !prefix {
		parts := strings.Split(name, ".")
		if len(parts) < 3 {
			return []string{"must be at least three segments long: <kind>.<domain>.<tld>"}
		}
	}

	return nil
}

func ValidateThirdPartyResource(obj *extensions.ThirdPartyResource) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&obj.ObjectMeta, false, ValidateThirdPartyResourceName, field.NewPath("metadata"))...)

	versions := sets.String{}
	if len(obj.Versions) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("versions"), "must specify at least one version"))
	}
	for ix := range obj.Versions {
		version := &obj.Versions[ix]
		if len(version.Name) == 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("versions").Index(ix).Child("name"), version, "must not be empty"))
		} else {
			for _, msg := range validation.IsDNS1123Label(version.Name) {
				allErrs = append(allErrs, field.Invalid(field.NewPath("versions").Index(ix).Child("name"), version, msg))
			}
		}
		if versions.Has(version.Name) {
			allErrs = append(allErrs, field.Duplicate(field.NewPath("versions").Index(ix).Child("name"), version))
		}
		versions.Insert(version.Name)
	}
	return allErrs
}

// ValidateDaemonSet tests if required fields in the DaemonSet are set.
func ValidateDaemonSet(ds *extensions.DaemonSet) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMeta(&ds.ObjectMeta, true, ValidateDaemonSetName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDaemonSetSpec(&ds.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateDaemonSetUpdate tests if required fields in the DaemonSet are set.
func ValidateDaemonSetUpdate(ds, oldDS *extensions.DaemonSet) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&ds.ObjectMeta, &oldDS.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDaemonSetSpec(&ds.Spec, field.NewPath("spec"))...)
	return allErrs
}

// validateDaemonSetStatus validates a DaemonSetStatus
func validateDaemonSetStatus(status *extensions.DaemonSetStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.CurrentNumberScheduled), fldPath.Child("currentNumberScheduled"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.NumberMisscheduled), fldPath.Child("numberMisscheduled"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.DesiredNumberScheduled), fldPath.Child("desiredNumberScheduled"))...)
	return allErrs
}

// ValidateDaemonSetStatus validates tests if required fields in the DaemonSet Status section
func ValidateDaemonSetStatusUpdate(ds, oldDS *extensions.DaemonSet) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&ds.ObjectMeta, &oldDS.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, validateDaemonSetStatus(&ds.Status, field.NewPath("status"))...)
	return allErrs
}

// ValidateDaemonSetSpec tests if required fields in the DaemonSetSpec are set.
func ValidateDaemonSetSpec(spec *extensions.DaemonSetSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(spec.Selector, fldPath.Child("selector"))...)

	selector, err := unversioned.LabelSelectorAsSelector(spec.Selector)
	if err == nil && !selector.Matches(labels.Set(spec.Template.Labels)) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("template", "metadata", "labels"), spec.Template.Labels, "`selector` does not match template `labels`"))
	}
	if spec.Selector != nil && len(spec.Selector.MatchLabels)+len(spec.Selector.MatchExpressions) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("selector"), spec.Selector, "empty selector is not valid for daemonset."))
	}

	allErrs = append(allErrs, apivalidation.ValidatePodTemplateSpec(&spec.Template, fldPath.Child("template"))...)
	// Daemons typically run on more than one node, so mark Read-Write persistent disks as invalid.
	allErrs = append(allErrs, apivalidation.ValidateReadOnlyPersistentDisks(spec.Template.Spec.Volumes, fldPath.Child("template", "spec", "volumes"))...)
	// RestartPolicy has already been first-order validated as per ValidatePodTemplateSpec().
	if spec.Template.Spec.RestartPolicy != api.RestartPolicyAlways {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("template", "spec", "restartPolicy"), spec.Template.Spec.RestartPolicy, []string{string(api.RestartPolicyAlways)}))
	}
	return allErrs
}

// ValidateDaemonSetName can be used to check whether the given daemon set name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateDaemonSetName = apivalidation.NameIsDNSSubdomain

// Validates that the given name can be used as a deployment name.
var ValidateDeploymentName = apivalidation.NameIsDNSSubdomain

func ValidatePositiveIntOrPercent(intOrPercent intstr.IntOrString, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch intOrPercent.Type {
	case intstr.String:
		for _, msg := range validation.IsValidPercent(intOrPercent.StrVal) {
			allErrs = append(allErrs, field.Invalid(fldPath, intOrPercent, msg))
		}
	case intstr.Int:
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(intOrPercent.IntValue()), fldPath)...)
	default:
		allErrs = append(allErrs, field.Invalid(fldPath, intOrPercent, "must be an integer or percentage (e.g '5%%')"))
	}
	return allErrs
}

func getPercentValue(intOrStringValue intstr.IntOrString) (int, bool) {
	if intOrStringValue.Type != intstr.String {
		return 0, false
	}
	if len(validation.IsValidPercent(intOrStringValue.StrVal)) != 0 {
		return 0, false
	}
	value, _ := strconv.Atoi(intOrStringValue.StrVal[:len(intOrStringValue.StrVal)-1])
	return value, true
}

func getIntOrPercentValue(intOrStringValue intstr.IntOrString) int {
	value, isPercent := getPercentValue(intOrStringValue)
	if isPercent {
		return value
	}
	return intOrStringValue.IntValue()
}

func IsNotMoreThan100Percent(intOrStringValue intstr.IntOrString, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	value, isPercent := getPercentValue(intOrStringValue)
	if !isPercent || value <= 100 {
		return nil
	}
	allErrs = append(allErrs, field.Invalid(fldPath, intOrStringValue, "must not be greater than 100%"))
	return allErrs
}

func ValidateRollingUpdateDeployment(rollingUpdate *extensions.RollingUpdateDeployment, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidatePositiveIntOrPercent(rollingUpdate.MaxUnavailable, fldPath.Child("maxUnavailable"))...)
	allErrs = append(allErrs, ValidatePositiveIntOrPercent(rollingUpdate.MaxSurge, fldPath.Child("maxSurge"))...)
	if getIntOrPercentValue(rollingUpdate.MaxUnavailable) == 0 && getIntOrPercentValue(rollingUpdate.MaxSurge) == 0 {
		// Both MaxSurge and MaxUnavailable cannot be zero.
		allErrs = append(allErrs, field.Invalid(fldPath.Child("maxUnavailable"), rollingUpdate.MaxUnavailable, "may not be 0 when `maxSurge` is 0"))
	}
	// Validate that MaxUnavailable is not more than 100%.
	allErrs = append(allErrs, IsNotMoreThan100Percent(rollingUpdate.MaxUnavailable, fldPath.Child("maxUnavailable"))...)
	return allErrs
}

func ValidateDeploymentStrategy(strategy *extensions.DeploymentStrategy, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	switch strategy.Type {
	case extensions.RecreateDeploymentStrategyType:
		if strategy.RollingUpdate != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("rollingUpdate"), "may not be specified when strategy `type` is '"+string(extensions.RecreateDeploymentStrategyType+"'")))
		}
	case extensions.RollingUpdateDeploymentStrategyType:
		// This should never happen since it's set and checked in defaults.go
		if strategy.RollingUpdate == nil {
			allErrs = append(allErrs, field.Required(fldPath.Child("rollingUpdate"), "this should be defaulted and never be nil"))
		} else {
			allErrs = append(allErrs, ValidateRollingUpdateDeployment(strategy.RollingUpdate, fldPath.Child("rollingUpdate"))...)
		}
	default:
		validValues := []string{string(extensions.RecreateDeploymentStrategyType), string(extensions.RollingUpdateDeploymentStrategyType)}
		allErrs = append(allErrs, field.NotSupported(fldPath, strategy, validValues))
	}
	return allErrs
}

func ValidateRollback(rollback *extensions.RollbackConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	v := rollback.Revision
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(v), fldPath.Child("version"))...)
	return allErrs
}

// Validates given deployment spec.
func ValidateDeploymentSpec(spec *extensions.DeploymentSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(spec.Replicas), fldPath.Child("replicas"))...)

	if spec.Selector == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("selector"), ""))
	} else {
		allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(spec.Selector, fldPath.Child("selector"))...)
		if len(spec.Selector.MatchLabels)+len(spec.Selector.MatchExpressions) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("selector"), spec.Selector, "empty selector is not valid for deployment."))
		}
	}

	selector, err := unversioned.LabelSelectorAsSelector(spec.Selector)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("selector"), spec.Selector, "invalid label selector."))
	} else {
		allErrs = append(allErrs, ValidatePodTemplateSpecForReplicaSet(&spec.Template, selector, spec.Replicas, fldPath.Child("template"))...)
	}

	allErrs = append(allErrs, ValidateDeploymentStrategy(&spec.Strategy, fldPath.Child("strategy"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(spec.MinReadySeconds), fldPath.Child("minReadySeconds"))...)
	if spec.RevisionHistoryLimit != nil {
		// zero is a valid RevisionHistoryLimit
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(*spec.RevisionHistoryLimit), fldPath.Child("revisionHistoryLimit"))...)
	}
	if spec.RollbackTo != nil {
		allErrs = append(allErrs, ValidateRollback(spec.RollbackTo, fldPath.Child("rollback"))...)
	}
	if spec.ProgressDeadlineSeconds != nil {
		allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(*spec.ProgressDeadlineSeconds), fldPath.Child("progressDeadlineSeconds"))...)
		if *spec.ProgressDeadlineSeconds <= spec.MinReadySeconds {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("progressDeadlineSeconds"), spec.ProgressDeadlineSeconds, "must be greater than minReadySeconds."))
		}
	}
	return allErrs
}

// Validates given deployment status.
func ValidateDeploymentStatus(status *extensions.DeploymentStatus, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(status.ObservedGeneration, fldPath.Child("observedGeneration"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.Replicas), fldPath.Child("replicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.UpdatedReplicas), fldPath.Child("updatedReplicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.AvailableReplicas), fldPath.Child("availableReplicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(status.UnavailableReplicas), fldPath.Child("unavailableReplicas"))...)
	return allErrs
}

func ValidateDeploymentUpdate(update, old *extensions.Deployment) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDeploymentSpec(&update.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateDeploymentStatusUpdate(update, old *extensions.Deployment) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDeploymentStatus(&update.Status, field.NewPath("status"))...)
	return allErrs
}

func ValidateDeployment(obj *extensions.Deployment) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMeta(&obj.ObjectMeta, true, ValidateDeploymentName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDeploymentSpec(&obj.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidateDeploymentRollback(obj *extensions.DeploymentRollback) field.ErrorList {
	allErrs := apivalidation.ValidateAnnotations(obj.UpdatedAnnotations, field.NewPath("updatedAnnotations"))
	if len(obj.Name) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("name"), "name is required"))
	}
	allErrs = append(allErrs, ValidateRollback(&obj.RollbackTo, field.NewPath("rollback"))...)
	return allErrs
}

func ValidateThirdPartyResourceDataUpdate(update, old *extensions.ThirdPartyResourceData) field.ErrorList {
	return apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
}

func ValidateThirdPartyResourceData(obj *extensions.ThirdPartyResourceData) field.ErrorList {
	return apivalidation.ValidateObjectMeta(&obj.ObjectMeta, true, apivalidation.NameIsDNSLabel, field.NewPath("metadata"))
}

// ValidateIngress tests if required fields in the Ingress are set.
func ValidateIngress(ingress *extensions.Ingress) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMeta(&ingress.ObjectMeta, true, ValidateIngressName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateIngressSpec(&ingress.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateIngressName validates that the given name can be used as an Ingress name.
var ValidateIngressName = apivalidation.NameIsDNSSubdomain

func validateIngressTLS(spec *extensions.IngressSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: Perform a more thorough validation of spec.TLS.Hosts that takes
	// the wildcard spec from RFC 6125 into account.
	for _, itls := range spec.TLS {
		for i, host := range itls.Hosts {
			if strings.Contains(host, "*") {
				for _, msg := range validation.IsWildcardDNS1123Subdomain(host) {
					allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("hosts"), host, msg))
				}
				continue
			}
			for _, msg := range validation.IsDNS1123Subdomain(host) {
				allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("hosts"), host, msg))
			}
		}
	}

	return allErrs
}

// ValidateIngressSpec tests if required fields in the IngressSpec are set.
func ValidateIngressSpec(spec *extensions.IngressSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO: Is a default backend mandatory?
	if spec.Backend != nil {
		allErrs = append(allErrs, validateIngressBackend(spec.Backend, fldPath.Child("backend"))...)
	} else if len(spec.Rules) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, spec.Rules, "either `backend` or `rules` must be specified"))
	}
	if len(spec.Rules) > 0 {
		allErrs = append(allErrs, validateIngressRules(spec.Rules, fldPath.Child("rules"))...)
	}
	if len(spec.TLS) > 0 {
		allErrs = append(allErrs, validateIngressTLS(spec, fldPath.Child("tls"))...)
	}
	return allErrs
}

// ValidateIngressUpdate tests if required fields in the Ingress are set.
func ValidateIngressUpdate(ingress, oldIngress *extensions.Ingress) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&ingress.ObjectMeta, &oldIngress.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateIngressSpec(&ingress.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateIngressStatusUpdate tests if required fields in the Ingress are set when updating status.
func ValidateIngressStatusUpdate(ingress, oldIngress *extensions.Ingress) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&ingress.ObjectMeta, &oldIngress.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, apivalidation.ValidateLoadBalancerStatus(&ingress.Status.LoadBalancer, field.NewPath("status", "loadBalancer"))...)
	return allErrs
}

func validateIngressRules(ingressRules []extensions.IngressRule, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(ingressRules) == 0 {
		return append(allErrs, field.Required(fldPath, ""))
	}
	for i, ih := range ingressRules {
		if len(ih.Host) > 0 {
			if isIP := (net.ParseIP(ih.Host) != nil); isIP {
				allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("host"), ih.Host, "must be a DNS name, not an IP address"))
			}
			// TODO: Ports and ips are allowed in the host part of a url
			// according to RFC 3986, consider allowing them.
			if strings.Contains(ih.Host, "*") {
				for _, msg := range validation.IsWildcardDNS1123Subdomain(ih.Host) {
					allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("host"), ih.Host, msg))
				}
				continue
			}
			for _, msg := range validation.IsDNS1123Subdomain(ih.Host) {
				allErrs = append(allErrs, field.Invalid(fldPath.Index(i).Child("host"), ih.Host, msg))
			}
		}
		allErrs = append(allErrs, validateIngressRuleValue(&ih.IngressRuleValue, fldPath.Index(0))...)
	}
	return allErrs
}

func validateIngressRuleValue(ingressRule *extensions.IngressRuleValue, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if ingressRule.HTTP != nil {
		allErrs = append(allErrs, validateHTTPIngressRuleValue(ingressRule.HTTP, fldPath.Child("http"))...)
	}
	return allErrs
}

func validateHTTPIngressRuleValue(httpIngressRuleValue *extensions.HTTPIngressRuleValue, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(httpIngressRuleValue.Paths) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("paths"), ""))
	}
	for i, rule := range httpIngressRuleValue.Paths {
		if len(rule.Path) > 0 {
			if !strings.HasPrefix(rule.Path, "/") {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("paths").Index(i).Child("path"), rule.Path, "must be an absolute path"))
			}
			// TODO: More draconian path regex validation.
			// Path must be a valid regex. This is the basic requirement.
			// In addition to this any characters not allowed in a path per
			// RFC 3986 section-3.3 cannot appear as a literal in the regex.
			// Consider the example: http://host/valid?#bar, everything after
			// the last '/' is a valid regex that matches valid#bar, which
			// isn't a valid path, because the path terminates at the first ?
			// or #. A more sophisticated form of validation would detect that
			// the user is confusing url regexes with path regexes.
			_, err := regexp.CompilePOSIX(rule.Path)
			if err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("paths").Index(i).Child("path"), rule.Path, "must be a valid regex"))
			}
		}
		allErrs = append(allErrs, validateIngressBackend(&rule.Backend, fldPath.Child("backend"))...)
	}
	return allErrs
}

// validateIngressBackend tests if a given backend is valid.
func validateIngressBackend(backend *extensions.IngressBackend, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// All backends must reference a single local service by name, and a single service port by name or number.
	if len(backend.ServiceName) == 0 {
		return append(allErrs, field.Required(fldPath.Child("serviceName"), ""))
	} else {
		for _, msg := range apivalidation.ValidateServiceName(backend.ServiceName, false) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("serviceName"), backend.ServiceName, msg))
		}
	}
	allErrs = append(allErrs, apivalidation.ValidatePortNumOrName(backend.ServicePort, fldPath.Child("servicePort"))...)
	return allErrs
}

func ValidateScale(scale *extensions.Scale) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&scale.ObjectMeta, true, apivalidation.NameIsDNSSubdomain, field.NewPath("metadata"))...)

	if scale.Spec.Replicas < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "replicas"), scale.Spec.Replicas, "must be greater than or equal to 0"))
	}

	return allErrs
}

// ValidateReplicaSetName can be used to check whether the given ReplicaSet
// name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateReplicaSetName = apivalidation.NameIsDNSSubdomain

// ValidateReplicaSet tests if required fields in the ReplicaSet are set.
func ValidateReplicaSet(rs *extensions.ReplicaSet) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMeta(&rs.ObjectMeta, true, ValidateReplicaSetName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateReplicaSetSpec(&rs.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateReplicaSetUpdate tests if required fields in the ReplicaSet are set.
func ValidateReplicaSetUpdate(rs, oldRs *extensions.ReplicaSet) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&rs.ObjectMeta, &oldRs.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidateReplicaSetSpec(&rs.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateReplicaSetStatusUpdate tests if required fields in the ReplicaSet are set.
func ValidateReplicaSetStatusUpdate(rs, oldRs *extensions.ReplicaSet) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&rs.ObjectMeta, &oldRs.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(rs.Status.Replicas), field.NewPath("status", "replicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(rs.Status.FullyLabeledReplicas), field.NewPath("status", "fullyLabeledReplicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(rs.Status.ReadyReplicas), field.NewPath("status", "readyReplicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(rs.Status.AvailableReplicas), field.NewPath("status", "availableReplicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(rs.Status.ObservedGeneration), field.NewPath("status", "observedGeneration"))...)
	return allErrs
}

// ValidateReplicaSetSpec tests if required fields in the ReplicaSet spec are set.
func ValidateReplicaSetSpec(spec *extensions.ReplicaSetSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(spec.Replicas), fldPath.Child("replicas"))...)
	allErrs = append(allErrs, apivalidation.ValidateNonnegativeField(int64(spec.MinReadySeconds), fldPath.Child("minReadySeconds"))...)

	if spec.Selector == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("selector"), ""))
	} else {
		allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(spec.Selector, fldPath.Child("selector"))...)
		if len(spec.Selector.MatchLabels)+len(spec.Selector.MatchExpressions) == 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("selector"), spec.Selector, "empty selector is not valid for deployment."))
		}
	}

	selector, err := unversioned.LabelSelectorAsSelector(spec.Selector)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("selector"), spec.Selector, "invalid label selector."))
	} else {
		allErrs = append(allErrs, ValidatePodTemplateSpecForReplicaSet(&spec.Template, selector, spec.Replicas, fldPath.Child("template"))...)
	}
	return allErrs
}

// Validates the given template and ensures that it is in accordance with the desired selector and replicas.
func ValidatePodTemplateSpecForReplicaSet(template *api.PodTemplateSpec, selector labels.Selector, replicas int32, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if template == nil {
		allErrs = append(allErrs, field.Required(fldPath, ""))
	} else {
		if !selector.Empty() {
			// Verify that the ReplicaSet selector matches the labels in template.
			labels := labels.Set(template.Labels)
			if !selector.Matches(labels) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("metadata", "labels"), template.Labels, "`selector` does not match template `labels`"))
			}
		}
		allErrs = append(allErrs, apivalidation.ValidatePodTemplateSpec(template, fldPath)...)
		if replicas > 1 {
			allErrs = append(allErrs, apivalidation.ValidateReadOnlyPersistentDisks(template.Spec.Volumes, fldPath.Child("spec", "volumes"))...)
		}
		// RestartPolicy has already been first-order validated as per ValidatePodTemplateSpec().
		if template.Spec.RestartPolicy != api.RestartPolicyAlways {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("spec", "restartPolicy"), template.Spec.RestartPolicy, []string{string(api.RestartPolicyAlways)}))
		}
	}
	return allErrs
}

// ValidatePodSecurityPolicyName can be used to check whether the given
// pod security policy name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidatePodSecurityPolicyName = apivalidation.NameIsDNSSubdomain

func ValidatePodSecurityPolicy(psp *extensions.PodSecurityPolicy) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMeta(&psp.ObjectMeta, false, ValidatePodSecurityPolicyName, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidatePodSecurityPolicySpecificAnnotations(psp.Annotations, field.NewPath("metadata").Child("annotations"))...)
	allErrs = append(allErrs, ValidatePodSecurityPolicySpec(&psp.Spec, field.NewPath("spec"))...)
	return allErrs
}

func ValidatePodSecurityPolicySpec(spec *extensions.PodSecurityPolicySpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validatePSPRunAsUser(fldPath.Child("runAsUser"), &spec.RunAsUser)...)
	allErrs = append(allErrs, validatePSPSELinux(fldPath.Child("seLinux"), &spec.SELinux)...)
	allErrs = append(allErrs, validatePSPSupplementalGroup(fldPath.Child("supplementalGroups"), &spec.SupplementalGroups)...)
	allErrs = append(allErrs, validatePSPFSGroup(fldPath.Child("fsGroup"), &spec.FSGroup)...)
	allErrs = append(allErrs, validatePodSecurityPolicyVolumes(fldPath, spec.Volumes)...)
	allErrs = append(allErrs, validatePSPCapsAgainstDrops(spec.RequiredDropCapabilities, spec.DefaultAddCapabilities, field.NewPath("defaultAddCapabilities"))...)
	allErrs = append(allErrs, validatePSPCapsAgainstDrops(spec.RequiredDropCapabilities, spec.AllowedCapabilities, field.NewPath("allowedCapabilities"))...)

	return allErrs
}

func ValidatePodSecurityPolicySpecificAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if p := annotations[apparmor.DefaultProfileAnnotationKey]; p != "" {
		if err := apparmor.ValidateProfileFormat(p); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Key(apparmor.DefaultProfileAnnotationKey), p, err.Error()))
		}
	}
	if allowed := annotations[apparmor.AllowedProfilesAnnotationKey]; allowed != "" {
		for _, p := range strings.Split(allowed, ",") {
			if err := apparmor.ValidateProfileFormat(p); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Key(apparmor.AllowedProfilesAnnotationKey), allowed, err.Error()))
			}
		}
	}

	sysctlAnnotation := annotations[extensions.SysctlsPodSecurityPolicyAnnotationKey]
	sysctlFldPath := fldPath.Key(extensions.SysctlsPodSecurityPolicyAnnotationKey)
	sysctls, err := extensions.SysctlsFromPodSecurityPolicyAnnotation(sysctlAnnotation)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(sysctlFldPath, sysctlAnnotation, err.Error()))
	} else {
		allErrs = append(allErrs, validatePodSecurityPolicySysctls(sysctlFldPath, sysctls)...)
	}

	if p := annotations[seccomp.DefaultProfileAnnotationKey]; p != "" {
		allErrs = append(allErrs, apivalidation.ValidateSeccompProfile(p, fldPath.Key(seccomp.DefaultProfileAnnotationKey))...)
	}
	if allowed := annotations[seccomp.AllowedProfilesAnnotationKey]; allowed != "" {
		for _, p := range strings.Split(allowed, ",") {
			allErrs = append(allErrs, apivalidation.ValidateSeccompProfile(p, fldPath.Key(seccomp.AllowedProfilesAnnotationKey))...)
		}
	}
	return allErrs
}

// validatePSPSELinux validates the SELinux fields of PodSecurityPolicy.
func validatePSPSELinux(fldPath *field.Path, seLinux *extensions.SELinuxStrategyOptions) field.ErrorList {
	allErrs := field.ErrorList{}

	// ensure the selinux strategy has a valid rule
	supportedSELinuxRules := sets.NewString(string(extensions.SELinuxStrategyMustRunAs),
		string(extensions.SELinuxStrategyRunAsAny))
	if !supportedSELinuxRules.Has(string(seLinux.Rule)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("rule"), seLinux.Rule, supportedSELinuxRules.List()))
	}

	return allErrs
}

// validatePSPRunAsUser validates the RunAsUser fields of PodSecurityPolicy.
func validatePSPRunAsUser(fldPath *field.Path, runAsUser *extensions.RunAsUserStrategyOptions) field.ErrorList {
	allErrs := field.ErrorList{}

	// ensure the user strategy has a valid rule
	supportedRunAsUserRules := sets.NewString(string(extensions.RunAsUserStrategyMustRunAs),
		string(extensions.RunAsUserStrategyMustRunAsNonRoot),
		string(extensions.RunAsUserStrategyRunAsAny))
	if !supportedRunAsUserRules.Has(string(runAsUser.Rule)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("rule"), runAsUser.Rule, supportedRunAsUserRules.List()))
	}

	// validate range settings
	for idx, rng := range runAsUser.Ranges {
		allErrs = append(allErrs, validateIDRanges(fldPath.Child("ranges").Index(idx), rng)...)
	}

	return allErrs
}

// validatePSPFSGroup validates the FSGroupStrategyOptions fields of the PodSecurityPolicy.
func validatePSPFSGroup(fldPath *field.Path, groupOptions *extensions.FSGroupStrategyOptions) field.ErrorList {
	allErrs := field.ErrorList{}

	supportedRules := sets.NewString(
		string(extensions.FSGroupStrategyMustRunAs),
		string(extensions.FSGroupStrategyRunAsAny),
	)
	if !supportedRules.Has(string(groupOptions.Rule)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("rule"), groupOptions.Rule, supportedRules.List()))
	}

	for idx, rng := range groupOptions.Ranges {
		allErrs = append(allErrs, validateIDRanges(fldPath.Child("ranges").Index(idx), rng)...)
	}
	return allErrs
}

// validatePSPSupplementalGroup validates the SupplementalGroupsStrategyOptions fields of the PodSecurityPolicy.
func validatePSPSupplementalGroup(fldPath *field.Path, groupOptions *extensions.SupplementalGroupsStrategyOptions) field.ErrorList {
	allErrs := field.ErrorList{}

	supportedRules := sets.NewString(
		string(extensions.SupplementalGroupsStrategyRunAsAny),
		string(extensions.SupplementalGroupsStrategyMustRunAs),
	)
	if !supportedRules.Has(string(groupOptions.Rule)) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("rule"), groupOptions.Rule, supportedRules.List()))
	}

	for idx, rng := range groupOptions.Ranges {
		allErrs = append(allErrs, validateIDRanges(fldPath.Child("ranges").Index(idx), rng)...)
	}
	return allErrs
}

// validatePodSecurityPolicyVolumes validates the volume fields of PodSecurityPolicy.
func validatePodSecurityPolicyVolumes(fldPath *field.Path, volumes []extensions.FSType) field.ErrorList {
	allErrs := field.ErrorList{}
	allowed := psputil.GetAllFSTypesAsSet()
	// add in the * value since that is a pseudo type that is not included by default
	allowed.Insert(string(extensions.All))
	for _, v := range volumes {
		if !allowed.Has(string(v)) {
			allErrs = append(allErrs, field.NotSupported(fldPath.Child("volumes"), v, allowed.List()))
		}
	}

	return allErrs
}

const sysctlPatternSegmentFmt string = "([a-z0-9][-_a-z0-9]*)?[a-z0-9*]"
const SysctlPatternFmt string = "(" + apivalidation.SysctlSegmentFmt + "\\.)*" + sysctlPatternSegmentFmt

var sysctlPatternRegexp = regexp.MustCompile("^" + SysctlPatternFmt + "$")

func IsValidSysctlPattern(name string) bool {
	if len(name) > apivalidation.SysctlMaxLength {
		return false
	}
	return sysctlPatternRegexp.MatchString(name)
}

// validatePodSecurityPolicySysctls validates the sysctls fields of PodSecurityPolicy.
func validatePodSecurityPolicySysctls(fldPath *field.Path, sysctls []string) field.ErrorList {
	allErrs := field.ErrorList{}
	for i, s := range sysctls {
		if !IsValidSysctlPattern(string(s)) {
			allErrs = append(
				allErrs,
				field.Invalid(fldPath.Index(i), sysctls[i], fmt.Sprintf("must have at most %d characters and match regex %s",
					apivalidation.SysctlMaxLength,
					SysctlPatternFmt,
				)),
			)
		}
	}

	return allErrs
}

// validateIDRanges ensures the range is valid.
func validateIDRanges(fldPath *field.Path, rng extensions.IDRange) field.ErrorList {
	allErrs := field.ErrorList{}

	// if 0 <= Min <= Max then we do not need to validate max.  It is always greater than or
	// equal to 0 and Min.
	if rng.Min < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("min"), rng.Min, "min cannot be negative"))
	}
	if rng.Max < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("max"), rng.Max, "max cannot be negative"))
	}
	if rng.Min > rng.Max {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("min"), rng.Min, "min cannot be greater than max"))
	}

	return allErrs
}

// validatePSPCapsAgainstDrops ensures an allowed cap is not listed in the required drops.
func validatePSPCapsAgainstDrops(requiredDrops []api.Capability, capsToCheck []api.Capability, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if requiredDrops == nil {
		return allErrs
	}
	for _, cap := range capsToCheck {
		if hasCap(cap, requiredDrops) {
			allErrs = append(allErrs, field.Invalid(fldPath, cap,
				fmt.Sprintf("capability is listed in %s and requiredDropCapabilities", fldPath.String())))
		}
	}
	return allErrs
}

// hasCap checks for needle in haystack.
func hasCap(needle api.Capability, haystack []api.Capability) bool {
	for _, c := range haystack {
		if needle == c {
			return true
		}
	}
	return false
}

// ValidatePodSecurityPolicyUpdate validates a PSP for updates.
func ValidatePodSecurityPolicyUpdate(old *extensions.PodSecurityPolicy, new *extensions.PodSecurityPolicy) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&new.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))...)
	allErrs = append(allErrs, ValidatePodSecurityPolicySpecificAnnotations(new.Annotations, field.NewPath("metadata").Child("annotations"))...)
	allErrs = append(allErrs, ValidatePodSecurityPolicySpec(&new.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateNetworkPolicyName can be used to check whether the given networkpolicy
// name is valid.
func ValidateNetworkPolicyName(name string, prefix bool) []string {
	return apivalidation.NameIsDNSSubdomain(name, prefix)
}

// ValidateNetworkPolicySpec tests if required fields in the networkpolicy spec are set.
func ValidateNetworkPolicySpec(spec *extensions.NetworkPolicySpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(&spec.PodSelector, fldPath.Child("podSelector"))...)

	// Validate ingress rules.
	for _, i := range spec.Ingress {
		// TODO: Update From to be a pointer to slice as soon as auto-generation supports it.
		for _, f := range i.From {
			numFroms := 0
			if f.PodSelector != nil {
				numFroms++
				allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(f.PodSelector, fldPath.Child("podSelector"))...)
			}
			if f.NamespaceSelector != nil {
				if numFroms > 0 {
					allErrs = append(allErrs, field.Forbidden(fldPath, "may not specify more than 1 from type"))
				} else {
					numFroms++
					allErrs = append(allErrs, unversionedvalidation.ValidateLabelSelector(f.NamespaceSelector, fldPath.Child("namespaces"))...)
				}
			}

			if numFroms == 0 {
				// At least one of PodSelector and NamespaceSelector must be defined.
				allErrs = append(allErrs, field.Required(fldPath, "must specify a from type"))
			}
		}
	}
	return allErrs
}

// ValidateNetworkPolicy validates a networkpolicy.
func ValidateNetworkPolicy(np *extensions.NetworkPolicy) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMeta(&np.ObjectMeta, true, ValidateNetworkPolicyName, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateNetworkPolicySpec(&np.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateNetworkPolicyUpdate tests if an update to a NetworkPolicy is valid.
func ValidateNetworkPolicyUpdate(update, old *extensions.NetworkPolicy) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateObjectMetaUpdate(&update.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))...)
	if !reflect.DeepEqual(update.Spec, old.Spec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "updates to networkpolicy spec are forbidden."))
	}
	return allErrs
}
