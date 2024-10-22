/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mesosutil

import (
	"github.com/openshift/github.com/gogo/protobuf/proto"
	mesos "github.com/openshift/github.com/mesos/mesos-go/mesosproto"
)

func NewValueRange(begin, end uint64) *mesos.Value_Range {
	return &mesos.Value_Range{Begin: proto.Uint64(begin), End: proto.Uint64(end)}
}

func FilterResources(resources []*mesos.Resource, filter func(*mesos.Resource) bool) (result []*mesos.Resource) {
	for _, resource := range resources {
		if filter(resource) {
			result = append(result, resource)
		}
	}
	return result
}

func AddResourceReservation(resource *mesos.Resource, principal string, role string) *mesos.Resource {
	resource.Reservation = &mesos.Resource_ReservationInfo{Principal: proto.String(principal)}
	resource.Role = proto.String(role)
	return resource
}

func NewScalarResourceWithReservation(name string, value float64, principal string, role string) *mesos.Resource {
	return AddResourceReservation(NewScalarResource(name, value), principal, role)
}

func NewRangesResourceWithReservation(name string, ranges []*mesos.Value_Range, principal string, role string) *mesos.Resource {
	return AddResourceReservation(NewRangesResource(name, ranges), principal, role)
}

func NewSetResourceWithReservation(name string, items []string, principal string, role string) *mesos.Resource {
	return AddResourceReservation(NewSetResource(name, items), principal, role)
}

func NewVolumeResourceWithReservation(val float64, containerPath string, persistenceId string, mode *mesos.Volume_Mode, principal string, role string) *mesos.Resource {
	return AddResourceReservation(NewVolumeResource(val, containerPath, persistenceId, mode), principal, role)
}

func NewScalarResource(name string, val float64) *mesos.Resource {
	return &mesos.Resource{
		Name:   proto.String(name),
		Type:   mesos.Value_SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{Value: proto.Float64(val)},
	}
}

func NewRangesResource(name string, ranges []*mesos.Value_Range) *mesos.Resource {
	return &mesos.Resource{
		Name:   proto.String(name),
		Type:   mesos.Value_RANGES.Enum(),
		Ranges: &mesos.Value_Ranges{Range: ranges},
	}
}

func NewSetResource(name string, items []string) *mesos.Resource {
	return &mesos.Resource{
		Name: proto.String(name),
		Type: mesos.Value_SET.Enum(),
		Set:  &mesos.Value_Set{Item: items},
	}
}

func NewVolumeResource(val float64, containerPath string, persistenceId string, mode *mesos.Volume_Mode) *mesos.Resource {
	resource := NewScalarResource("disk", val)
	resource.Disk = &mesos.Resource_DiskInfo{
		Persistence: &mesos.Resource_DiskInfo_Persistence{Id: proto.String(persistenceId)},
		Volume:      &mesos.Volume{ContainerPath: proto.String(containerPath), Mode: mode},
	}
	return resource
}

func NewFrameworkID(id string) *mesos.FrameworkID {
	return &mesos.FrameworkID{Value: proto.String(id)}
}

func NewFrameworkInfo(user, name string, frameworkId *mesos.FrameworkID) *mesos.FrameworkInfo {
	return &mesos.FrameworkInfo{
		User: proto.String(user),
		Name: proto.String(name),
		Id:   frameworkId,
	}
}

func NewMasterInfo(id string, ip, port uint32) *mesos.MasterInfo {
	return &mesos.MasterInfo{
		Id:   proto.String(id),
		Ip:   proto.Uint32(ip),
		Port: proto.Uint32(port),
	}
}

func NewOfferID(id string) *mesos.OfferID {
	return &mesos.OfferID{Value: proto.String(id)}
}

func NewOffer(offerId *mesos.OfferID, frameworkId *mesos.FrameworkID, slaveId *mesos.SlaveID, hostname string) *mesos.Offer {
	return &mesos.Offer{
		Id:          offerId,
		FrameworkId: frameworkId,
		SlaveId:     slaveId,
		Hostname:    proto.String(hostname),
	}
}

func FilterOffersResources(offers []*mesos.Offer, filter func(*mesos.Resource) bool) (result []*mesos.Resource) {
	for _, offer := range offers {
		result = FilterResources(offer.Resources, filter)
	}
	return result
}

func NewSlaveID(id string) *mesos.SlaveID {
	return &mesos.SlaveID{Value: proto.String(id)}
}

func NewTaskID(id string) *mesos.TaskID {
	return &mesos.TaskID{Value: proto.String(id)}
}

func NewTaskInfo(
	name string,
	taskId *mesos.TaskID,
	slaveId *mesos.SlaveID,
	resources []*mesos.Resource,
) *mesos.TaskInfo {
	return &mesos.TaskInfo{
		Name:      proto.String(name),
		TaskId:    taskId,
		SlaveId:   slaveId,
		Resources: resources,
	}
}

func NewTaskStatus(taskId *mesos.TaskID, state mesos.TaskState) *mesos.TaskStatus {
	return &mesos.TaskStatus{
		TaskId: taskId,
		State:  mesos.TaskState(state).Enum(),
	}
}

func NewStatusUpdate(frameworkId *mesos.FrameworkID, taskStatus *mesos.TaskStatus, timestamp float64, uuid []byte) *mesos.StatusUpdate {
	return &mesos.StatusUpdate{
		FrameworkId: frameworkId,
		Status:      taskStatus,
		Timestamp:   proto.Float64(timestamp),
		Uuid:        uuid,
	}
}

func NewCommandInfo(command string) *mesos.CommandInfo {
	return &mesos.CommandInfo{Value: proto.String(command)}
}

func NewExecutorID(id string) *mesos.ExecutorID {
	return &mesos.ExecutorID{Value: proto.String(id)}
}

func NewExecutorInfo(execId *mesos.ExecutorID, command *mesos.CommandInfo) *mesos.ExecutorInfo {
	return &mesos.ExecutorInfo{
		ExecutorId: execId,
		Command:    command,
	}
}

func NewCreateOperation(volumes []*mesos.Resource) *mesos.Offer_Operation {
	return &mesos.Offer_Operation{
		Type:   mesos.Offer_Operation_CREATE.Enum(),
		Create: &mesos.Offer_Operation_Create{Volumes: volumes},
	}
}

func NewDestroyOperation(volumes []*mesos.Resource) *mesos.Offer_Operation {
	return &mesos.Offer_Operation{
		Type:    mesos.Offer_Operation_DESTROY.Enum(),
		Destroy: &mesos.Offer_Operation_Destroy{Volumes: volumes},
	}
}

func NewReserveOperation(resources []*mesos.Resource) *mesos.Offer_Operation {
	return &mesos.Offer_Operation{
		Type:    mesos.Offer_Operation_RESERVE.Enum(),
		Reserve: &mesos.Offer_Operation_Reserve{Resources: resources},
	}
}

func NewUnreserveOperation(resources []*mesos.Resource) *mesos.Offer_Operation {
	return &mesos.Offer_Operation{
		Type:      mesos.Offer_Operation_UNRESERVE.Enum(),
		Unreserve: &mesos.Offer_Operation_Unreserve{Resources: resources},
	}
}

func NewLaunchOperation(tasks []*mesos.TaskInfo) *mesos.Offer_Operation {
	return &mesos.Offer_Operation{
		Type:   mesos.Offer_Operation_LAUNCH.Enum(),
		Launch: &mesos.Offer_Operation_Launch{TaskInfos: tasks},
	}
}
