package network

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator 0.17.0.0
// Changes may cause incorrect behavior and will be lost if the code is
// regenerated.

import (
	"github.com/openshift/github.com/Azure/go-autorest/autorest"
	"github.com/openshift/github.com/Azure/go-autorest/autorest/azure"
	"github.com/openshift/github.com/Azure/go-autorest/autorest/validation"
	"net/http"
)

// InterfacesClient is the the Microsoft Azure Network management API provides
// a RESTful set of web services that interact with Microsoft Azure Networks
// service to manage your network resources. The API has entities that
// capture the relationship between an end user and the Microsoft Azure
// Networks service.
type InterfacesClient struct {
	ManagementClient
}

// NewInterfacesClient creates an instance of the InterfacesClient client.
func NewInterfacesClient(subscriptionID string) InterfacesClient {
	return NewInterfacesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewInterfacesClientWithBaseURI creates an instance of the InterfacesClient
// client.
func NewInterfacesClientWithBaseURI(baseURI string, subscriptionID string) InterfacesClient {
	return InterfacesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate creates or updates a network interface. This method may poll
// for completion. Polling can be canceled by passing the cancel channel
// argument. The channel will be used to cancel polling and any outstanding
// HTTP requests.
//
// resourceGroupName is the name of the resource group. networkInterfaceName
// is the name of the network interface. parameters is parameters supplied to
// the create or update network interface operation.
func (client InterfacesClient) CreateOrUpdate(resourceGroupName string, networkInterfaceName string, parameters Interface, cancel <-chan struct{}) (result autorest.Response, err error) {
	if err := validation.Validate([]validation.Validation{
		{TargetValue: parameters,
			Constraints: []validation.Constraint{{Target: "parameters.InterfacePropertiesFormat", Name: validation.Null, Rule: false,
				Chain: []validation.Constraint{{Target: "parameters.InterfacePropertiesFormat.NetworkSecurityGroup", Name: validation.Null, Rule: false,
					Chain: []validation.Constraint{{Target: "parameters.InterfacePropertiesFormat.NetworkSecurityGroup.SecurityGroupPropertiesFormat", Name: validation.Null, Rule: false,
						Chain: []validation.Constraint{{Target: "parameters.InterfacePropertiesFormat.NetworkSecurityGroup.SecurityGroupPropertiesFormat.NetworkInterfaces", Name: validation.ReadOnly, Rule: true, Chain: nil},
							{Target: "parameters.InterfacePropertiesFormat.NetworkSecurityGroup.SecurityGroupPropertiesFormat.Subnets", Name: validation.ReadOnly, Rule: true, Chain: nil},
						}},
					}},
				}}}}}); err != nil {
		return result, validation.NewErrorWithValidationError(err, "network.InterfacesClient", "CreateOrUpdate")
	}

	req, err := client.CreateOrUpdatePreparer(resourceGroupName, networkInterfaceName, parameters, cancel)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "CreateOrUpdate", nil, "Failure preparing request")
	}

	resp, err := client.CreateOrUpdateSender(req)
	if err != nil {
		result.Response = resp
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "CreateOrUpdate", resp, "Failure sending request")
	}

	result, err = client.CreateOrUpdateResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "CreateOrUpdate", resp, "Failure responding to request")
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client InterfacesClient) CreateOrUpdatePreparer(resourceGroupName string, networkInterfaceName string, parameters Interface, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName": autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":    autorest.Encode("path", resourceGroupName),
		"subscriptionId":       autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsJSON(),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces/{networkInterfaceName}", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) CreateOrUpdateSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// CreateOrUpdateResponder handles the response to the CreateOrUpdate request. The method always
// closes the http.Response Body.
func (client InterfacesClient) CreateOrUpdateResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated, http.StatusOK),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Delete deletes the specified network interface. This method may poll for
// completion. Polling can be canceled by passing the cancel channel
// argument. The channel will be used to cancel polling and any outstanding
// HTTP requests.
//
// resourceGroupName is the name of the resource group. networkInterfaceName
// is the name of the network interface.
func (client InterfacesClient) Delete(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (result autorest.Response, err error) {
	req, err := client.DeletePreparer(resourceGroupName, networkInterfaceName, cancel)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "Delete", nil, "Failure preparing request")
	}

	resp, err := client.DeleteSender(req)
	if err != nil {
		result.Response = resp
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "Delete", resp, "Failure sending request")
	}

	result, err = client.DeleteResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "Delete", resp, "Failure responding to request")
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client InterfacesClient) DeletePreparer(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName": autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":    autorest.Encode("path", resourceGroupName),
		"subscriptionId":       autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces/{networkInterfaceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) DeleteSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// DeleteResponder handles the response to the Delete request. The method always
// closes the http.Response Body.
func (client InterfacesClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusNoContent, http.StatusAccepted, http.StatusOK),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets information about the specified network interface.
//
// resourceGroupName is the name of the resource group. networkInterfaceName
// is the name of the network interface. expand is expands referenced
// resources.
func (client InterfacesClient) Get(resourceGroupName string, networkInterfaceName string, expand string) (result Interface, err error) {
	req, err := client.GetPreparer(resourceGroupName, networkInterfaceName, expand)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "Get", nil, "Failure preparing request")
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "Get", resp, "Failure sending request")
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client InterfacesClient) GetPreparer(resourceGroupName string, networkInterfaceName string, expand string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName": autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":    autorest.Encode("path", resourceGroupName),
		"subscriptionId":       autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}
	if len(expand) > 0 {
		queryParameters["$expand"] = autorest.Encode("query", expand)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces/{networkInterfaceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client InterfacesClient) GetResponder(resp *http.Response) (result Interface, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// GetEffectiveRouteTable gets all route tables applied to a network
// interface. This method may poll for completion. Polling can be canceled by
// passing the cancel channel argument. The channel will be used to cancel
// polling and any outstanding HTTP requests.
//
// resourceGroupName is the name of the resource group. networkInterfaceName
// is the name of the network interface.
func (client InterfacesClient) GetEffectiveRouteTable(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (result autorest.Response, err error) {
	req, err := client.GetEffectiveRouteTablePreparer(resourceGroupName, networkInterfaceName, cancel)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "GetEffectiveRouteTable", nil, "Failure preparing request")
	}

	resp, err := client.GetEffectiveRouteTableSender(req)
	if err != nil {
		result.Response = resp
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "GetEffectiveRouteTable", resp, "Failure sending request")
	}

	result, err = client.GetEffectiveRouteTableResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "GetEffectiveRouteTable", resp, "Failure responding to request")
	}

	return
}

// GetEffectiveRouteTablePreparer prepares the GetEffectiveRouteTable request.
func (client InterfacesClient) GetEffectiveRouteTablePreparer(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName": autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":    autorest.Encode("path", resourceGroupName),
		"subscriptionId":       autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces/{networkInterfaceName}/effectiveRouteTable", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// GetEffectiveRouteTableSender sends the GetEffectiveRouteTable request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) GetEffectiveRouteTableSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// GetEffectiveRouteTableResponder handles the response to the GetEffectiveRouteTable request. The method always
// closes the http.Response Body.
func (client InterfacesClient) GetEffectiveRouteTableResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// GetVirtualMachineScaleSetNetworkInterface get the specified network
// interface in a virtual machine scale set.
//
// resourceGroupName is the name of the resource group.
// virtualMachineScaleSetName is the name of the virtual machine scale set.
// virtualmachineIndex is the virtual machine index. networkInterfaceName is
// the name of the network interface. expand is expands referenced resources.
func (client InterfacesClient) GetVirtualMachineScaleSetNetworkInterface(resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string, networkInterfaceName string, expand string) (result Interface, err error) {
	req, err := client.GetVirtualMachineScaleSetNetworkInterfacePreparer(resourceGroupName, virtualMachineScaleSetName, virtualmachineIndex, networkInterfaceName, expand)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "GetVirtualMachineScaleSetNetworkInterface", nil, "Failure preparing request")
	}

	resp, err := client.GetVirtualMachineScaleSetNetworkInterfaceSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "GetVirtualMachineScaleSetNetworkInterface", resp, "Failure sending request")
	}

	result, err = client.GetVirtualMachineScaleSetNetworkInterfaceResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "GetVirtualMachineScaleSetNetworkInterface", resp, "Failure responding to request")
	}

	return
}

// GetVirtualMachineScaleSetNetworkInterfacePreparer prepares the GetVirtualMachineScaleSetNetworkInterface request.
func (client InterfacesClient) GetVirtualMachineScaleSetNetworkInterfacePreparer(resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string, networkInterfaceName string, expand string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName":       autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":          autorest.Encode("path", resourceGroupName),
		"subscriptionId":             autorest.Encode("path", client.SubscriptionID),
		"virtualmachineIndex":        autorest.Encode("path", virtualmachineIndex),
		"virtualMachineScaleSetName": autorest.Encode("path", virtualMachineScaleSetName),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}
	if len(expand) > 0 {
		queryParameters["$expand"] = autorest.Encode("query", expand)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/microsoft.Compute/virtualMachineScaleSets/{virtualMachineScaleSetName}/virtualMachines/{virtualmachineIndex}/networkInterfaces/{networkInterfaceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// GetVirtualMachineScaleSetNetworkInterfaceSender sends the GetVirtualMachineScaleSetNetworkInterface request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) GetVirtualMachineScaleSetNetworkInterfaceSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// GetVirtualMachineScaleSetNetworkInterfaceResponder handles the response to the GetVirtualMachineScaleSetNetworkInterface request. The method always
// closes the http.Response Body.
func (client InterfacesClient) GetVirtualMachineScaleSetNetworkInterfaceResponder(resp *http.Response) (result Interface, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// List gets all network interfaces in a resource group.
//
// resourceGroupName is the name of the resource group.
func (client InterfacesClient) List(resourceGroupName string) (result InterfaceListResult, err error) {
	req, err := client.ListPreparer(resourceGroupName)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "List", nil, "Failure preparing request")
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "List", resp, "Failure sending request")
	}

	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "List", resp, "Failure responding to request")
	}

	return
}

// ListPreparer prepares the List request.
func (client InterfacesClient) ListPreparer(resourceGroupName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) ListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client InterfacesClient) ListResponder(resp *http.Response) (result InterfaceListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListNextResults retrieves the next set of results, if any.
func (client InterfacesClient) ListNextResults(lastResults InterfaceListResult) (result InterfaceListResult, err error) {
	req, err := lastResults.InterfaceListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "List", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "List", resp, "Failure sending next results request")
	}

	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "List", resp, "Failure responding to next results request")
	}

	return
}

// ListAll gets all network interfaces in a subscription.
func (client InterfacesClient) ListAll() (result InterfaceListResult, err error) {
	req, err := client.ListAllPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", nil, "Failure preparing request")
	}

	resp, err := client.ListAllSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", resp, "Failure sending request")
	}

	result, err = client.ListAllResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", resp, "Failure responding to request")
	}

	return
}

// ListAllPreparer prepares the ListAll request.
func (client InterfacesClient) ListAllPreparer() (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"subscriptionId": autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/providers/Microsoft.Network/networkInterfaces", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// ListAllSender sends the ListAll request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) ListAllSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// ListAllResponder handles the response to the ListAll request. The method always
// closes the http.Response Body.
func (client InterfacesClient) ListAllResponder(resp *http.Response) (result InterfaceListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListAllNextResults retrieves the next set of results, if any.
func (client InterfacesClient) ListAllNextResults(lastResults InterfaceListResult) (result InterfaceListResult, err error) {
	req, err := lastResults.InterfaceListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}

	resp, err := client.ListAllSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", resp, "Failure sending next results request")
	}

	result, err = client.ListAllResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListAll", resp, "Failure responding to next results request")
	}

	return
}

// ListEffectiveNetworkSecurityGroups gets all network security groups applied
// to a network interface. This method may poll for completion. Polling can
// be canceled by passing the cancel channel argument. The channel will be
// used to cancel polling and any outstanding HTTP requests.
//
// resourceGroupName is the name of the resource group. networkInterfaceName
// is the name of the network interface.
func (client InterfacesClient) ListEffectiveNetworkSecurityGroups(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (result autorest.Response, err error) {
	req, err := client.ListEffectiveNetworkSecurityGroupsPreparer(resourceGroupName, networkInterfaceName, cancel)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListEffectiveNetworkSecurityGroups", nil, "Failure preparing request")
	}

	resp, err := client.ListEffectiveNetworkSecurityGroupsSender(req)
	if err != nil {
		result.Response = resp
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListEffectiveNetworkSecurityGroups", resp, "Failure sending request")
	}

	result, err = client.ListEffectiveNetworkSecurityGroupsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListEffectiveNetworkSecurityGroups", resp, "Failure responding to request")
	}

	return
}

// ListEffectiveNetworkSecurityGroupsPreparer prepares the ListEffectiveNetworkSecurityGroups request.
func (client InterfacesClient) ListEffectiveNetworkSecurityGroupsPreparer(resourceGroupName string, networkInterfaceName string, cancel <-chan struct{}) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkInterfaceName": autorest.Encode("path", networkInterfaceName),
		"resourceGroupName":    autorest.Encode("path", resourceGroupName),
		"subscriptionId":       autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkInterfaces/{networkInterfaceName}/effectiveNetworkSecurityGroups", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{Cancel: cancel})
}

// ListEffectiveNetworkSecurityGroupsSender sends the ListEffectiveNetworkSecurityGroups request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) ListEffectiveNetworkSecurityGroupsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client,
		req,
		azure.DoPollForAsynchronous(client.PollingDelay))
}

// ListEffectiveNetworkSecurityGroupsResponder handles the response to the ListEffectiveNetworkSecurityGroups request. The method always
// closes the http.Response Body.
func (client InterfacesClient) ListEffectiveNetworkSecurityGroupsResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// ListVirtualMachineScaleSetNetworkInterfaces gets all network interfaces in
// a virtual machine scale set.
//
// resourceGroupName is the name of the resource group.
// virtualMachineScaleSetName is the name of the virtual machine scale set.
func (client InterfacesClient) ListVirtualMachineScaleSetNetworkInterfaces(resourceGroupName string, virtualMachineScaleSetName string) (result InterfaceListResult, err error) {
	req, err := client.ListVirtualMachineScaleSetNetworkInterfacesPreparer(resourceGroupName, virtualMachineScaleSetName)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", nil, "Failure preparing request")
	}

	resp, err := client.ListVirtualMachineScaleSetNetworkInterfacesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", resp, "Failure sending request")
	}

	result, err = client.ListVirtualMachineScaleSetNetworkInterfacesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", resp, "Failure responding to request")
	}

	return
}

// ListVirtualMachineScaleSetNetworkInterfacesPreparer prepares the ListVirtualMachineScaleSetNetworkInterfaces request.
func (client InterfacesClient) ListVirtualMachineScaleSetNetworkInterfacesPreparer(resourceGroupName string, virtualMachineScaleSetName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName":          autorest.Encode("path", resourceGroupName),
		"subscriptionId":             autorest.Encode("path", client.SubscriptionID),
		"virtualMachineScaleSetName": autorest.Encode("path", virtualMachineScaleSetName),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/microsoft.Compute/virtualMachineScaleSets/{virtualMachineScaleSetName}/networkInterfaces", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// ListVirtualMachineScaleSetNetworkInterfacesSender sends the ListVirtualMachineScaleSetNetworkInterfaces request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) ListVirtualMachineScaleSetNetworkInterfacesSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// ListVirtualMachineScaleSetNetworkInterfacesResponder handles the response to the ListVirtualMachineScaleSetNetworkInterfaces request. The method always
// closes the http.Response Body.
func (client InterfacesClient) ListVirtualMachineScaleSetNetworkInterfacesResponder(resp *http.Response) (result InterfaceListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListVirtualMachineScaleSetNetworkInterfacesNextResults retrieves the next set of results, if any.
func (client InterfacesClient) ListVirtualMachineScaleSetNetworkInterfacesNextResults(lastResults InterfaceListResult) (result InterfaceListResult, err error) {
	req, err := lastResults.InterfaceListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}

	resp, err := client.ListVirtualMachineScaleSetNetworkInterfacesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", resp, "Failure sending next results request")
	}

	result, err = client.ListVirtualMachineScaleSetNetworkInterfacesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetNetworkInterfaces", resp, "Failure responding to next results request")
	}

	return
}

// ListVirtualMachineScaleSetVMNetworkInterfaces gets information about all
// network interfaces in a virtual machine in a virtual machine scale set.
//
// resourceGroupName is the name of the resource group.
// virtualMachineScaleSetName is the name of the virtual machine scale set.
// virtualmachineIndex is the virtual machine index.
func (client InterfacesClient) ListVirtualMachineScaleSetVMNetworkInterfaces(resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string) (result InterfaceListResult, err error) {
	req, err := client.ListVirtualMachineScaleSetVMNetworkInterfacesPreparer(resourceGroupName, virtualMachineScaleSetName, virtualmachineIndex)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", nil, "Failure preparing request")
	}

	resp, err := client.ListVirtualMachineScaleSetVMNetworkInterfacesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", resp, "Failure sending request")
	}

	result, err = client.ListVirtualMachineScaleSetVMNetworkInterfacesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", resp, "Failure responding to request")
	}

	return
}

// ListVirtualMachineScaleSetVMNetworkInterfacesPreparer prepares the ListVirtualMachineScaleSetVMNetworkInterfaces request.
func (client InterfacesClient) ListVirtualMachineScaleSetVMNetworkInterfacesPreparer(resourceGroupName string, virtualMachineScaleSetName string, virtualmachineIndex string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName":          autorest.Encode("path", resourceGroupName),
		"subscriptionId":             autorest.Encode("path", client.SubscriptionID),
		"virtualmachineIndex":        autorest.Encode("path", virtualmachineIndex),
		"virtualMachineScaleSetName": autorest.Encode("path", virtualMachineScaleSetName),
	}

	queryParameters := map[string]interface{}{
		"api-version": client.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/microsoft.Compute/virtualMachineScaleSets/{virtualMachineScaleSetName}/virtualMachines/{virtualmachineIndex}/networkInterfaces", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare(&http.Request{})
}

// ListVirtualMachineScaleSetVMNetworkInterfacesSender sends the ListVirtualMachineScaleSetVMNetworkInterfaces request. The method will close the
// http.Response Body if it receives an error.
func (client InterfacesClient) ListVirtualMachineScaleSetVMNetworkInterfacesSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req)
}

// ListVirtualMachineScaleSetVMNetworkInterfacesResponder handles the response to the ListVirtualMachineScaleSetVMNetworkInterfaces request. The method always
// closes the http.Response Body.
func (client InterfacesClient) ListVirtualMachineScaleSetVMNetworkInterfacesResponder(resp *http.Response) (result InterfaceListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListVirtualMachineScaleSetVMNetworkInterfacesNextResults retrieves the next set of results, if any.
func (client InterfacesClient) ListVirtualMachineScaleSetVMNetworkInterfacesNextResults(lastResults InterfaceListResult) (result InterfaceListResult, err error) {
	req, err := lastResults.InterfaceListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}

	resp, err := client.ListVirtualMachineScaleSetVMNetworkInterfacesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", resp, "Failure sending next results request")
	}

	result, err = client.ListVirtualMachineScaleSetVMNetworkInterfacesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InterfacesClient", "ListVirtualMachineScaleSetVMNetworkInterfaces", resp, "Failure responding to next results request")
	}

	return
}
