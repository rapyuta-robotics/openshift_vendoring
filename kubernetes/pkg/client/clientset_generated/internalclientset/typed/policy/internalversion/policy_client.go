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

package internalversion

import (
	api "github.com/openshift/kubernetes/pkg/api"
	registered "github.com/openshift/kubernetes/pkg/apimachinery/registered"
	restclient "github.com/openshift/kubernetes/pkg/client/restclient"
)

type PolicyInterface interface {
	RESTClient() restclient.Interface
	EvictionsGetter
	PodDisruptionBudgetsGetter
}

// PolicyClient is used to interact with features provided by the k8s.io/kubernetes/pkg/apimachinery/registered.Group group.
type PolicyClient struct {
	restClient restclient.Interface
}

func (c *PolicyClient) Evictions(namespace string) EvictionInterface {
	return newEvictions(c, namespace)
}

func (c *PolicyClient) PodDisruptionBudgets(namespace string) PodDisruptionBudgetInterface {
	return newPodDisruptionBudgets(c, namespace)
}

// NewForConfig creates a new PolicyClient for the given config.
func NewForConfig(c *restclient.Config) (*PolicyClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := restclient.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PolicyClient{client}, nil
}

// NewForConfigOrDie creates a new PolicyClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *restclient.Config) *PolicyClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PolicyClient for the given RESTClient.
func New(c restclient.Interface) *PolicyClient {
	return &PolicyClient{c}
}

func setConfigDefaults(config *restclient.Config) error {
	// if policy group is not registered, return an error
	g, err := registered.Group("policy")
	if err != nil {
		return err
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = restclient.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != g.GroupVersion.Group {
		copyGroupVersion := g.GroupVersion
		config.GroupVersion = &copyGroupVersion
	}
	config.NegotiatedSerializer = api.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}
	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *PolicyClient) RESTClient() restclient.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
