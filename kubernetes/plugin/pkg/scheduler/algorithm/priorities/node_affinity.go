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

package priorities

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/labels"
	schedulerapi "github.com/openshift/kubernetes/plugin/pkg/scheduler/api"
	"github.com/openshift/kubernetes/plugin/pkg/scheduler/schedulercache"
)

// CalculateNodeAffinityPriority prioritizes nodes according to node affinity scheduling preferences
// indicated in PreferredDuringSchedulingIgnoredDuringExecution. Each time a node match a preferredSchedulingTerm,
// it will a get an add of preferredSchedulingTerm.Weight. Thus, the more preferredSchedulingTerms
// the node satisfies and the more the preferredSchedulingTerm that is satisfied weights, the higher
// score the node gets.
func CalculateNodeAffinityPriorityMap(pod *api.Pod, meta interface{}, nodeInfo *schedulercache.NodeInfo) (schedulerapi.HostPriority, error) {
	node := nodeInfo.Node()
	if node == nil {
		return schedulerapi.HostPriority{}, fmt.Errorf("node not found")
	}

	var affinity *api.Affinity
	if priorityMeta, ok := meta.(*priorityMetadata); ok {
		affinity = priorityMeta.affinity
	} else {
		// We couldn't parse metadata - fallback to computing it.
		var err error
		affinity, err = api.GetAffinityFromPodAnnotations(pod.Annotations)
		if err != nil {
			return schedulerapi.HostPriority{}, err
		}
	}

	var count int32
	// A nil element of PreferredDuringSchedulingIgnoredDuringExecution matches no objects.
	// An element of PreferredDuringSchedulingIgnoredDuringExecution that refers to an
	// empty PreferredSchedulingTerm matches all objects.
	if affinity != nil && affinity.NodeAffinity != nil && affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		// Match PreferredDuringSchedulingIgnoredDuringExecution term by term.
		for i := range affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			preferredSchedulingTerm := &affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]
			if preferredSchedulingTerm.Weight == 0 {
				continue
			}

			// TODO: Avoid computing it for all nodes if this becomes a performance problem.
			nodeSelector, err := api.NodeSelectorRequirementsAsSelector(preferredSchedulingTerm.Preference.MatchExpressions)
			if err != nil {
				return schedulerapi.HostPriority{}, err
			}
			if nodeSelector.Matches(labels.Set(node.Labels)) {
				count += preferredSchedulingTerm.Weight
			}
		}
	}

	return schedulerapi.HostPriority{
		Host:  node.Name,
		Score: int(count),
	}, nil
}

func CalculateNodeAffinityPriorityReduce(pod *api.Pod, meta interface{}, nodeNameToInfo map[string]*schedulercache.NodeInfo, result schedulerapi.HostPriorityList) error {
	var maxCount int
	for i := range result {
		if result[i].Score > maxCount {
			maxCount = result[i].Score
		}
	}
	maxCountFloat := float64(maxCount)

	var fScore float64
	for i := range result {
		if maxCount > 0 {
			fScore = 10 * (float64(result[i].Score) / maxCountFloat)
		} else {
			fScore = 0
		}
		if glog.V(10) {
			// We explicitly don't do glog.V(10).Infof() to avoid computing all the parameters if this is
			// not logged. There is visible performance gain from it.
			glog.Infof("%v -> %v: NodeAffinityPriority, Score: (%d)", pod.Name, result[i].Host, int(fScore))
		}
		result[i].Score = int(fScore)
	}
	return nil
}
