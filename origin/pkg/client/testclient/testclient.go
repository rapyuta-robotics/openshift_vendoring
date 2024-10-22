package testclient

import (
	kclientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	"github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"github.com/openshift/kubernetes/pkg/client/testing/core"
	"github.com/openshift/kubernetes/pkg/runtime"

	"github.com/openshift/origin/pkg/client"
)

// NewFixtureClients returns mocks of the OpenShift and Kubernetes clients
// with data populated from provided path.
func NewFixtureClients(objs ...runtime.Object) (client.Interface, kclientset.Interface) {
	oc := NewSimpleFake(objs...)
	kc := fake.NewSimpleClientset(objs...)
	return oc, kc
}

func NewErrorClients(err error) (client.Interface, kclientset.Interface) {
	oc := &Fake{}
	oc.PrependReactor("*", "*", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
	kc := &fake.Clientset{}
	kc.PrependReactor("*", "*", func(action core.Action) (bool, runtime.Object, error) {
		return true, nil, err
	})
	return oc, kc
}
