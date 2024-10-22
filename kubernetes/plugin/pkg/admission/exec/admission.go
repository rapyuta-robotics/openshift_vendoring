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

package exec

import (
	"fmt"
	"io"

	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"

	"github.com/openshift/kubernetes/pkg/admission"
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/api/errors"
	"github.com/openshift/kubernetes/pkg/api/rest"
)

func init() {
	admission.RegisterPlugin("DenyEscalatingExec", func(client clientset.Interface, config io.Reader) (admission.Interface, error) {
		return NewDenyEscalatingExec(client), nil
	})

	// This is for legacy support of the DenyExecOnPrivileged admission controller.  Most
	// of the time DenyEscalatingExec should be preferred.
	admission.RegisterPlugin("DenyExecOnPrivileged", func(client clientset.Interface, config io.Reader) (admission.Interface, error) {
		return NewDenyExecOnPrivileged(client), nil
	})
}

// denyExec is an implementation of admission.Interface which says no to a pod/exec on
// a pod using host based configurations.
type denyExec struct {
	*admission.Handler
	client clientset.Interface

	// these flags control which items will be checked to deny exec/attach
	hostIPC    bool
	hostPID    bool
	privileged bool
}

// NewDenyEscalatingExec creates a new admission controller that denies an exec operation on a pod
// using host based configurations.
func NewDenyEscalatingExec(client clientset.Interface) admission.Interface {
	return &denyExec{
		Handler:    admission.NewHandler(admission.Connect),
		client:     client,
		hostIPC:    true,
		hostPID:    true,
		privileged: true,
	}
}

// NewDenyExecOnPrivileged creates a new admission controller that is only checking the privileged
// option.  This is for legacy support of the DenyExecOnPrivileged admission controller.  Most
// of the time NewDenyEscalatingExec should be preferred.
func NewDenyExecOnPrivileged(client clientset.Interface) admission.Interface {
	return &denyExec{
		Handler:    admission.NewHandler(admission.Connect),
		client:     client,
		hostIPC:    false,
		hostPID:    false,
		privileged: true,
	}
}

func (d *denyExec) Admit(a admission.Attributes) (err error) {
	connectRequest, ok := a.GetObject().(*rest.ConnectRequest)
	if !ok {
		return errors.NewBadRequest("a connect request was received, but could not convert the request object.")
	}
	// Only handle exec or attach requests on pods
	if connectRequest.ResourcePath != "pods/exec" && connectRequest.ResourcePath != "pods/attach" {
		return nil
	}
	pod, err := d.client.Core().Pods(a.GetNamespace()).Get(connectRequest.Name)
	if err != nil {
		return admission.NewForbidden(a, err)
	}

	if d.hostPID && pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.HostPID {
		return admission.NewForbidden(a, fmt.Errorf("cannot exec into or attach to a container using host pid"))
	}

	if d.hostIPC && pod.Spec.SecurityContext != nil && pod.Spec.SecurityContext.HostIPC {
		return admission.NewForbidden(a, fmt.Errorf("cannot exec into or attach to a container using host ipc"))
	}

	if d.privileged && isPrivileged(pod) {
		return admission.NewForbidden(a, fmt.Errorf("cannot exec into or attach to a privileged container"))
	}

	return nil
}

// isPrivileged will return true a pod has any privileged containers
func isPrivileged(pod *api.Pod) bool {
	for _, c := range pod.Spec.InitContainers {
		if c.SecurityContext == nil {
			continue
		}
		if *c.SecurityContext.Privileged {
			return true
		}
	}
	for _, c := range pod.Spec.Containers {
		if c.SecurityContext == nil {
			continue
		}
		if *c.SecurityContext.Privileged {
			return true
		}
	}
	return false
}
