package plugin

import (
	"fmt"

	"github.com/golang/glog"

	osapi "github.com/openshift/origin/pkg/sdn/api"

	kapi "github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/cache"
	utilwait "github.com/openshift/kubernetes/pkg/util/wait"
)

func (plugin *OsdnNode) SetupEgressNetworkPolicy() error {
	policies, err := plugin.osClient.EgressNetworkPolicies(kapi.NamespaceAll).List(kapi.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not get EgressNetworkPolicies: %s", err)
	}

	for _, policy := range policies.Items {
		vnid, err := plugin.policy.GetVNID(policy.Namespace)
		if err != nil {
			glog.Warningf("Could not find netid for namespace %q: %v", policy.Namespace, err)
			continue
		}
		plugin.egressPolicies[vnid] = append(plugin.egressPolicies[vnid], policy)
	}

	for vnid := range plugin.egressPolicies {
		plugin.updateEgressNetworkPolicyRules(vnid)
	}

	go utilwait.Forever(plugin.watchEgressNetworkPolicies, 0)
	return nil
}

func (plugin *OsdnNode) watchEgressNetworkPolicies() {
	RunEventQueue(plugin.osClient, EgressNetworkPolicies, func(delta cache.Delta) error {
		policy := delta.Object.(*osapi.EgressNetworkPolicy)

		vnid, err := plugin.policy.GetVNID(policy.Namespace)
		if err != nil {
			return fmt.Errorf("Could not find netid for namespace %q: %v", policy.Namespace, err)
		}

		policies := plugin.egressPolicies[vnid]
		for i, oldPolicy := range policies {
			if oldPolicy.UID == policy.UID {
				policies = append(policies[:i], policies[i+1:]...)
				break
			}
		}
		if delta.Type != cache.Deleted && len(policy.Spec.Egress) > 0 {
			policies = append(policies, *policy)
		}
		plugin.egressPolicies[vnid] = policies

		plugin.updateEgressNetworkPolicyRules(vnid)
		return nil
	})
}

func (plugin *OsdnNode) UpdateEgressNetworkPolicyVNID(namespace string, oldVnid, newVnid uint32) {
	var policy *osapi.EgressNetworkPolicy

	policies := plugin.egressPolicies[oldVnid]
	for i, oldPolicy := range policies {
		if oldPolicy.Namespace == namespace {
			policy = &oldPolicy
			plugin.egressPolicies[oldVnid] = append(policies[:i], policies[i+1:]...)
			plugin.updateEgressNetworkPolicyRules(oldVnid)
			break
		}
	}

	if policy != nil {
		plugin.egressPolicies[newVnid] = append(plugin.egressPolicies[newVnid], *policy)
		plugin.updateEgressNetworkPolicyRules(newVnid)
	}
}
