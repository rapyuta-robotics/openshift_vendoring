/*
Copyright 2017 The Kubernetes Authors.

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

package fake

import (
	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/apimachinery/registered"
	clientset "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset"
	internalversionapps "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/apps/internalversion"
	fakeinternalversionapps "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/apps/internalversion/fake"
	internalversionauthentication "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/authentication/internalversion"
	fakeinternalversionauthentication "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/authentication/internalversion/fake"
	internalversionauthorization "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/authorization/internalversion"
	fakeinternalversionauthorization "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/authorization/internalversion/fake"
	internalversionautoscaling "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/autoscaling/internalversion"
	fakeinternalversionautoscaling "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/autoscaling/internalversion/fake"
	internalversionbatch "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/batch/internalversion"
	fakeinternalversionbatch "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/batch/internalversion/fake"
	internalversioncertificates "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/certificates/internalversion"
	fakeinternalversioncertificates "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/certificates/internalversion/fake"
	internalversioncore "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
	fakeinternalversioncore "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion/fake"
	internalversionextensions "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/extensions/internalversion"
	fakeinternalversionextensions "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/extensions/internalversion/fake"
	internalversionpolicy "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/policy/internalversion"
	fakeinternalversionpolicy "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/policy/internalversion/fake"
	internalversionrbac "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/rbac/internalversion"
	fakeinternalversionrbac "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/rbac/internalversion/fake"
	internalversionstorage "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/storage/internalversion"
	fakeinternalversionstorage "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/storage/internalversion/fake"
	"github.com/openshift/kubernetes/pkg/client/testing/core"
	"github.com/openshift/kubernetes/pkg/client/typed/discovery"
	fakediscovery "github.com/openshift/kubernetes/pkg/client/typed/discovery/fake"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/watch"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := core.NewObjectTracker(api.Scheme, api.Codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := core.Fake{}
	fakePtr.AddReactor("*", "*", core.ObjectReaction(o, registered.RESTMapper()))

	fakePtr.AddWatchReactor("*", core.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	core.Fake
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return &fakediscovery.FakeDiscovery{Fake: &c.Fake}
}

var _ clientset.Interface = &Clientset{}

// Core retrieves the CoreClient
func (c *Clientset) Core() internalversioncore.CoreInterface {
	return &fakeinternalversioncore.FakeCore{Fake: &c.Fake}
}

// Apps retrieves the AppsClient
func (c *Clientset) Apps() internalversionapps.AppsInterface {
	return &fakeinternalversionapps.FakeApps{Fake: &c.Fake}
}

// Authentication retrieves the AuthenticationClient
func (c *Clientset) Authentication() internalversionauthentication.AuthenticationInterface {
	return &fakeinternalversionauthentication.FakeAuthentication{Fake: &c.Fake}
}

// Authorization retrieves the AuthorizationClient
func (c *Clientset) Authorization() internalversionauthorization.AuthorizationInterface {
	return &fakeinternalversionauthorization.FakeAuthorization{Fake: &c.Fake}
}

// Autoscaling retrieves the AutoscalingClient
func (c *Clientset) Autoscaling() internalversionautoscaling.AutoscalingInterface {
	return &fakeinternalversionautoscaling.FakeAutoscaling{Fake: &c.Fake}
}

// Batch retrieves the BatchClient
func (c *Clientset) Batch() internalversionbatch.BatchInterface {
	return &fakeinternalversionbatch.FakeBatch{Fake: &c.Fake}
}

// Certificates retrieves the CertificatesClient
func (c *Clientset) Certificates() internalversioncertificates.CertificatesInterface {
	return &fakeinternalversioncertificates.FakeCertificates{Fake: &c.Fake}
}

// Extensions retrieves the ExtensionsClient
func (c *Clientset) Extensions() internalversionextensions.ExtensionsInterface {
	return &fakeinternalversionextensions.FakeExtensions{Fake: &c.Fake}
}

// Policy retrieves the PolicyClient
func (c *Clientset) Policy() internalversionpolicy.PolicyInterface {
	return &fakeinternalversionpolicy.FakePolicy{Fake: &c.Fake}
}

// Rbac retrieves the RbacClient
func (c *Clientset) Rbac() internalversionrbac.RbacInterface {
	return &fakeinternalversionrbac.FakeRbac{Fake: &c.Fake}
}

// Storage retrieves the StorageClient
func (c *Clientset) Storage() internalversionstorage.StorageInterface {
	return &fakeinternalversionstorage.FakeStorage{Fake: &c.Fake}
}
