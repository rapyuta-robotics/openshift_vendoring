/*
Copyright 2016 The Kubernetes Authors.

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

package deployment

import (
	"fmt"
	"reflect"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/apis/extensions"
	"github.com/openshift/kubernetes/pkg/controller/deployment/util"
)

// hasFailed determines if a deployment has failed or not by estimating its progress.
// Progress for a deployment is considered when a new replica set is created or adopted,
// and when new pods scale up or old pods scale down. Progress is not estimated for paused
// deployments or when users don't really care about it ie. progressDeadlineSeconds is not
// specified.
func (dc *DeploymentController) hasFailed(d *extensions.Deployment) (bool, error) {
	if d.Spec.ProgressDeadlineSeconds == nil || d.Spec.RollbackTo != nil || d.Spec.Paused {
		return false, nil
	}

	newRS, oldRSs, err := dc.getAllReplicaSetsAndSyncRevision(d, false)
	if err != nil {
		return false, err
	}

	// There is a template change so we don't need to check for any progress right now.
	if newRS == nil {
		return false, nil
	}

	// Look at the status of the deployment - if there is already a NewRSAvailableReason
	// then we don't need to estimate any progress. This is needed in order to avoid
	// estimating progress for scaling events after a rollout has finished.
	cond := util.GetDeploymentCondition(d.Status, extensions.DeploymentProgressing)
	if cond != nil && cond.Reason == util.NewRSAvailableReason {
		return false, nil
	}

	// TODO: Look for permanent failures here.
	// See https://github.com/kubernetes/kubernetes/issues/18568

	allRSs := append(oldRSs, newRS)
	newStatus := dc.calculateStatus(allRSs, newRS, d)

	// If the deployment is complete or it is progressing, there is no need to check if it
	// has timed out.
	if util.DeploymentComplete(d, &newStatus) || util.DeploymentProgressing(d, &newStatus) {
		return false, nil
	}

	// Check if the deployment has timed out.
	return util.DeploymentTimedOut(d, &newStatus), nil
}

// syncRolloutStatus updates the status of a deployment during a rollout. There are
// cases this helper will run that cannot be prevented from the scaling detection,
// for example a resync of the deployment after it was scaled up. In those cases,
// we shouldn't try to estimate any progress.
func (dc *DeploymentController) syncRolloutStatus(allRSs []*extensions.ReplicaSet, newRS *extensions.ReplicaSet, d *extensions.Deployment) error {
	newStatus := dc.calculateStatus(allRSs, newRS, d)

	// If there is no progressDeadlineSeconds set, remove any Progressing condition.
	if d.Spec.ProgressDeadlineSeconds == nil {
		util.RemoveDeploymentCondition(&newStatus, extensions.DeploymentProgressing)
	}

	// If there is only one replica set that is active then that means we are not running
	// a new rollout and this is a resync where we don't need to estimate any progress.
	// In such a case, we should simply not estimate any progress for this deployment.
	currentCond := util.GetDeploymentCondition(d.Status, extensions.DeploymentProgressing)
	isResyncEvent := newStatus.Replicas == newStatus.UpdatedReplicas && currentCond != nil && currentCond.Reason == util.NewRSAvailableReason
	// Check for progress only if there is a progress deadline set and the latest rollout
	// hasn't completed yet. We also need to ensure the new replica set exists, otherwise
	// we cannot estimate any progress.
	if d.Spec.ProgressDeadlineSeconds != nil && !isResyncEvent && newRS != nil {
		switch {
		case util.DeploymentComplete(d, &newStatus):
			// Update the deployment conditions with a message for the new replica set that
			// was successfully deployed. If the condition already exists, we ignore this update.
			msg := fmt.Sprintf("Replica set %q has successfully progressed.", newRS.Name)
			condition := util.NewDeploymentCondition(extensions.DeploymentProgressing, api.ConditionTrue, util.NewRSAvailableReason, msg)
			util.SetDeploymentCondition(&newStatus, *condition)

		case util.DeploymentProgressing(d, &newStatus):
			// If there is any progress made, continue by not checking if the deployment failed. This
			// behavior emulates the rolling updater progressDeadline check.
			msg := fmt.Sprintf("Replica set %q is progressing.", newRS.Name)
			condition := util.NewDeploymentCondition(extensions.DeploymentProgressing, api.ConditionTrue, util.ReplicaSetUpdatedReason, msg)
			// Update the current Progressing condition or add a new one if it doesn't exist.
			// If a Progressing condition with status=true already exists, we should update
			// everything but lastTransitionTime. SetDeploymentCondition already does that but
			// it also is not updating conditions when the reason of the new condition is the
			// same as the old. The Progressing condition is a special case because we want to
			// update with the same reason and change just lastUpdateTime iff we notice any
			// progress. That's why we handle it here.
			if currentCond != nil {
				if currentCond.Status == api.ConditionTrue {
					condition.LastTransitionTime = currentCond.LastTransitionTime
				}
				util.RemoveDeploymentCondition(&newStatus, extensions.DeploymentProgressing)
			}
			util.SetDeploymentCondition(&newStatus, *condition)

		case util.DeploymentTimedOut(d, &newStatus):
			// Update the deployment with a timeout condition. If the condition already exists,
			// we ignore this update.
			msg := fmt.Sprintf("Replica set %q has timed out progressing.", newRS.Name)
			condition := util.NewDeploymentCondition(extensions.DeploymentProgressing, api.ConditionFalse, util.TimedOutReason, msg)
			util.SetDeploymentCondition(&newStatus, *condition)
		}
	}

	// Move failure conditions of all replica sets in deployment conditions. For now,
	// only one failure condition is returned from getReplicaFailures.
	if replicaFailureCond := dc.getReplicaFailures(allRSs, newRS); len(replicaFailureCond) > 0 {
		// There will be only one ReplicaFailure condition on the replica set.
		util.SetDeploymentCondition(&newStatus, replicaFailureCond[0])
	} else {
		util.RemoveDeploymentCondition(&newStatus, extensions.DeploymentReplicaFailure)
	}

	// Do not update if there is nothing new to add.
	if reflect.DeepEqual(d.Status, newStatus) {
		// TODO: If there is no sign of progress at this point then there is a high chance that the
		// deployment is stuck. We should resync this deployment at some point[1] in the future[2] and
		// check if it has timed out. We definitely need this, otherwise we depend on the controller
		// resync interval. See https://github.com/kubernetes/kubernetes/issues/34458.
		//
		// [1] time.Now() + progressDeadlineSeconds - lastUpdateTime (of the Progressing condition).
		// [2] Use dc.queue.AddAfter
		return nil
	}

	newDeployment := d
	newDeployment.Status = newStatus
	_, err := dc.client.Extensions().Deployments(newDeployment.Namespace).UpdateStatus(newDeployment)
	return err
}

// getReplicaFailures will convert replica failure conditions from replica sets
// to deployment conditions.
func (dc *DeploymentController) getReplicaFailures(allRSs []*extensions.ReplicaSet, newRS *extensions.ReplicaSet) []extensions.DeploymentCondition {
	var conditions []extensions.DeploymentCondition
	if newRS != nil {
		for _, c := range newRS.Status.Conditions {
			if c.Type != extensions.ReplicaSetReplicaFailure {
				continue
			}
			conditions = append(conditions, util.ReplicaSetToDeploymentCondition(c))
		}
	}

	// Return failures for the new replica set over failures from old replica sets.
	if len(conditions) > 0 {
		return conditions
	}

	for i := range allRSs {
		rs := allRSs[i]
		if rs == nil {
			continue
		}

		for _, c := range rs.Status.Conditions {
			if c.Type != extensions.ReplicaSetReplicaFailure {
				continue
			}
			conditions = append(conditions, util.ReplicaSetToDeploymentCondition(c))
		}
	}
	return conditions
}
