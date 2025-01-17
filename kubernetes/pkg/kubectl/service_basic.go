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

package kubectl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/openshift/kubernetes/pkg/api"
	"github.com/openshift/kubernetes/pkg/runtime"
	"github.com/openshift/kubernetes/pkg/util/intstr"
)

type ServiceCommonGeneratorV1 struct {
	Name      string
	TCP       []string
	Type      api.ServiceType
	ClusterIP string
	NodePort  int
}

type ServiceClusterIPGeneratorV1 struct {
	ServiceCommonGeneratorV1
}

type ServiceNodePortGeneratorV1 struct {
	ServiceCommonGeneratorV1
}

type ServiceLoadBalancerGeneratorV1 struct {
	ServiceCommonGeneratorV1
}

func (ServiceClusterIPGeneratorV1) ParamNames() []GeneratorParam {
	return []GeneratorParam{
		{"name", true},
		{"tcp", true},
		{"clusterip", false},
	}
}
func (ServiceNodePortGeneratorV1) ParamNames() []GeneratorParam {
	return []GeneratorParam{
		{"name", true},
		{"tcp", true},
		{"nodeport", true},
	}
}
func (ServiceLoadBalancerGeneratorV1) ParamNames() []GeneratorParam {
	return []GeneratorParam{
		{"name", true},
		{"tcp", true},
	}
}

func parsePorts(portString string) (int32, intstr.IntOrString, error) {
	portStringSlice := strings.Split(portString, ":")

	port, err := strconv.Atoi(portStringSlice[0])
	if err != nil {
		return 0, intstr.FromInt(0), err
	}
	if len(portStringSlice) == 1 {
		return int32(port), intstr.FromInt(int(port)), nil
	}

	var targetPort intstr.IntOrString
	if portNum, err := strconv.Atoi(portStringSlice[1]); err != nil {
		targetPort = intstr.FromString(portStringSlice[1])
	} else {
		targetPort = intstr.FromInt(portNum)
	}
	return int32(port), targetPort, nil
}

func (s ServiceCommonGeneratorV1) GenerateCommon(params map[string]interface{}) error {
	name, isString := params["name"].(string)
	if !isString {
		return fmt.Errorf("expected string, saw %v for 'name'", name)
	}
	tcpStrings, isArray := params["tcp"].([]string)
	if !isArray {
		return fmt.Errorf("expected []string, found :%v", tcpStrings)
	}
	clusterip, isString := params["clusterip"].(string)
	if !isString {
		return fmt.Errorf("expected string, saw %v for 'clusterip'", clusterip)
	}
	s.Name = name
	s.TCP = tcpStrings
	s.ClusterIP = clusterip
	return nil
}

func (s ServiceLoadBalancerGeneratorV1) Generate(params map[string]interface{}) (runtime.Object, error) {
	err := ValidateParams(s.ParamNames(), params)
	if err != nil {
		return nil, err
	}
	delegate := &ServiceCommonGeneratorV1{Type: api.ServiceTypeLoadBalancer, ClusterIP: ""}
	err = delegate.GenerateCommon(params)
	if err != nil {
		return nil, err
	}
	return delegate.StructuredGenerate()
}

func (s ServiceNodePortGeneratorV1) Generate(params map[string]interface{}) (runtime.Object, error) {
	err := ValidateParams(s.ParamNames(), params)
	if err != nil {
		return nil, err
	}
	delegate := &ServiceCommonGeneratorV1{Type: api.ServiceTypeNodePort, ClusterIP: ""}
	err = delegate.GenerateCommon(params)
	if err != nil {
		return nil, err
	}
	return delegate.StructuredGenerate()
}

func (s ServiceClusterIPGeneratorV1) Generate(params map[string]interface{}) (runtime.Object, error) {
	err := ValidateParams(s.ParamNames(), params)
	if err != nil {
		return nil, err
	}
	delegate := &ServiceCommonGeneratorV1{Type: api.ServiceTypeClusterIP, ClusterIP: ""}
	err = delegate.GenerateCommon(params)
	if err != nil {
		return nil, err
	}
	return delegate.StructuredGenerate()
}

// validate validates required fields are set to support structured generation
func (s ServiceCommonGeneratorV1) validate() error {
	if len(s.Name) == 0 {
		return fmt.Errorf("name must be specified")
	}
	if len(s.Type) == 0 {
		return fmt.Errorf("type must be specified")
	}
	if s.ClusterIP == api.ClusterIPNone && s.Type != api.ServiceTypeClusterIP {
		return fmt.Errorf("ClusterIP=None can only be used with ClusterIP service type")
	}
	if s.ClusterIP == api.ClusterIPNone && len(s.TCP) > 0 {
		return fmt.Errorf("can not map ports with clusterip=None")
	}
	if s.ClusterIP != api.ClusterIPNone && len(s.TCP) == 0 {
		return fmt.Errorf("at least one tcp port specifier must be provided")
	}
	return nil
}

func (s ServiceCommonGeneratorV1) StructuredGenerate() (runtime.Object, error) {
	err := s.validate()
	if err != nil {
		return nil, err
	}
	ports := []api.ServicePort{}
	for _, tcpString := range s.TCP {
		port, targetPort, err := parsePorts(tcpString)
		if err != nil {
			return nil, err
		}

		portName := strings.Replace(tcpString, ":", "-", -1)
		ports = append(ports, api.ServicePort{
			Name:       portName,
			Port:       port,
			TargetPort: targetPort,
			Protocol:   api.Protocol("TCP"),
			NodePort:   int32(s.NodePort),
		})
	}

	// setup default label and selector
	labels := map[string]string{}
	labels["app"] = s.Name
	selector := map[string]string{}
	selector["app"] = s.Name

	service := api.Service{
		ObjectMeta: api.ObjectMeta{
			Name:   s.Name,
			Labels: labels,
		},
		Spec: api.ServiceSpec{
			Type:     api.ServiceType(s.Type),
			Selector: selector,
			Ports:    ports,
		},
	}
	if len(s.ClusterIP) > 0 {
		service.Spec.ClusterIP = s.ClusterIP
	}
	return &service, nil
}
