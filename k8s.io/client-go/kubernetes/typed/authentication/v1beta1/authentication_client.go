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

package v1beta1

import (
	fmt "fmt"
	api "github.com/openshift/k8s.io/client-go/pkg/api"
	unversioned "github.com/openshift/k8s.io/client-go/pkg/api/unversioned"
	registered "github.com/openshift/k8s.io/client-go/pkg/apimachinery/registered"
	serializer "github.com/openshift/k8s.io/client-go/pkg/runtime/serializer"
	rest "github.com/openshift/k8s.io/client-go/rest"
)

type AuthenticationV1beta1Interface interface {
	RESTClient() rest.Interface
	TokenReviewsGetter
}

// AuthenticationV1beta1Client is used to interact with features provided by the k8s.io/kubernetes/pkg/apimachinery/registered.Group group.
type AuthenticationV1beta1Client struct {
	restClient rest.Interface
}

func (c *AuthenticationV1beta1Client) TokenReviews() TokenReviewInterface {
	return newTokenReviews(c)
}

// NewForConfig creates a new AuthenticationV1beta1Client for the given config.
func NewForConfig(c *rest.Config) (*AuthenticationV1beta1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AuthenticationV1beta1Client{client}, nil
}

// NewForConfigOrDie creates a new AuthenticationV1beta1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *AuthenticationV1beta1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new AuthenticationV1beta1Client for the given RESTClient.
func New(c rest.Interface) *AuthenticationV1beta1Client {
	return &AuthenticationV1beta1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv, err := unversioned.ParseGroupVersion("authentication.k8s.io/v1beta1")
	if err != nil {
		return err
	}
	// if authentication.k8s.io/v1beta1 is not enabled, return an error
	if !registered.IsEnabledVersion(gv) {
		return fmt.Errorf("authentication.k8s.io/v1beta1 is not enabled")
	}
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	copyGroupVersion := gv
	config.GroupVersion = &copyGroupVersion

	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AuthenticationV1beta1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
