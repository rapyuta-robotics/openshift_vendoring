/*
Copyright 2014 The Kubernetes Authors.

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

package cmd

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/client/restclient/fake"
	cmdtesting "github.com/openshift/kubernetes/pkg/kubectl/cmd/testing"
)

func TestCreateSecretGeneric(t *testing.T) {
	secretObject := &api.Secret{}
	secretObject.Name = "my-secret"
	f, tf, codec, ns := cmdtesting.NewAPIFactory()
	tf.Printer = &testPrinter{}
	tf.Client = &fake.RESTClient{
		NegotiatedSerializer: ns,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch p, m := req.URL.Path, req.Method; {
			case p == "/namespaces/test/secrets" && m == "POST":
				return &http.Response{StatusCode: 201, Header: defaultHeader(), Body: objBody(codec, secretObject)}, nil
			default:
				t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
				return nil, nil
			}
		}),
	}
	tf.Namespace = "test"
	buf := bytes.NewBuffer([]byte{})
	cmd := NewCmdCreateSecretGeneric(f, buf)
	cmd.Flags().Set("output", "name")
	cmd.Run(cmd, []string{secretObject.Name})
	expectedOutput := "secret/" + secretObject.Name + "\n"
	if buf.String() != expectedOutput {
		t.Errorf("expected output: %s, but got: %s", buf.String(), expectedOutput)
	}
}

func TestCreateSecretDockerRegistry(t *testing.T) {
	secretObject := &api.Secret{}
	secretObject.Name = "my-secret"
	f, tf, codec, ns := cmdtesting.NewAPIFactory()
	tf.Printer = &testPrinter{}
	tf.Client = &fake.RESTClient{
		NegotiatedSerializer: ns,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			switch p, m := req.URL.Path, req.Method; {
			case p == "/namespaces/test/secrets" && m == "POST":
				return &http.Response{StatusCode: 201, Header: defaultHeader(), Body: objBody(codec, secretObject)}, nil
			default:
				t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
				return nil, nil
			}
		}),
	}
	tf.Namespace = "test"
	buf := bytes.NewBuffer([]byte{})
	cmd := NewCmdCreateSecretDockerRegistry(f, buf)
	cmd.Flags().Set("docker-username", "test-user")
	cmd.Flags().Set("docker-password", "test-pass")
	cmd.Flags().Set("docker-email", "test-email")
	cmd.Flags().Set("output", "name")
	cmd.Run(cmd, []string{secretObject.Name})
	expectedOutput := "secret/" + secretObject.Name + "\n"
	if buf.String() != expectedOutput {
		t.Errorf("expected output: %s, but got: %s", buf.String(), expectedOutput)
	}
}
