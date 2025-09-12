// Copyright 2024 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	netap "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
)

const (
	paramAwsEgressPrivateLinkEndpoint              = "aws_egress_private_link_endpoint"
	paramAzureEgressPrivateLinkEndpoint            = "azure_egress_private_link_endpoint"
	paramGcpEgressPrivateServiceConnectEndpoint    = "gcp_egress_private_service_connect_endpoint"
	paramAwsPrivateNetworkInterface                = "aws_private_network_interface"
	paramEnableHighAvailability                    = "enable_high_availability"
	paramVpcEndpointDnsName                        = "vpc_endpoint_dns_name"
	paramPrivateLinkSubresourceName                = "private_link_subresource_name"
	paramPrivateEndpointDomain                     = "private_endpoint_domain"
	paramPrivateEndpointIpAddress                  = "private_endpoint_ip_address"
	paramPrivateEndpointCustomDnsConfigDomains     = "private_endpoint_custom_dns_config_domains"
	paramPrivateServiceConnectEndpointTarget       = "private_service_connect_endpoint_target"
	paramPrivateServiceConnectEndpointConnectionId = "private_service_connect_endpoint_connection_id"
	paramPrivateServiceConnectEndpointIpAddress    = "private_service_connect_endpoint_ip_address"
	paramPrivateServiceConnectEndpointName         = "private_service_connect_endpoint_name"
	paramNetworkInterfaces                         = "network_interfaces"
	awsEgressPrivateLinkEndpoint                   = "AwsEgressPrivateLinkEndpoint"
	azureEgressPrivateLinkEndpoint                 = "AzureEgressPrivateLinkEndpoint"
	gcpEgressPrivateServiceConnectEndpoint         = "GcpEgressPrivateServiceConnectEndpoint"
	awsPrivateNetworkInterface                     = "AwsPrivateNetworkInterface"
)

var acceptedEndpointConfig = []string{paramAwsEgressPrivateLinkEndpoint, paramAzureEgressPrivateLinkEndpoint, paramGcpEgressPrivateServiceConnectEndpoint, paramAwsPrivateNetworkInterface}

func accessPointResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: accessPointCreate,
		ReadContext:   accessPointRead,
		UpdateContext: accessPointUpdate,
		DeleteContext: accessPointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: accessPointImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramGateway:                                requiredGateway(),
			paramEnvironment:                            environmentSchema(),
			paramAwsEgressPrivateLinkEndpoint:           paramAwsEgressPrivateLinkEndpointSchema(),
			paramAzureEgressPrivateLinkEndpoint:         paramAzureEgressPrivateLinkEndpointSchema(),
			paramGcpEgressPrivateServiceConnectEndpoint: paramGcpEgressPrivateServiceConnectEndpointSchema(),
			paramAwsPrivateNetworkInterface:             paramAwsPrivateNetworkInterfaceSchema(),
		},
	}
}

func paramAwsEgressPrivateLinkEndpointSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpcEndpointServiceName: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramEnableHighAvailability: {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
					ForceNew: true,
				},
				paramVpcEndpointId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramVpcEndpointDnsName: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		ExactlyOneOf: acceptedEndpointConfig,
	}
}

func paramAzureEgressPrivateLinkEndpointSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateLinkServiceResourceId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramPrivateLinkSubresourceName: {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
				},
				paramPrivateEndpointResourceId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointDomain: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointIpAddress: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramPrivateEndpointCustomDnsConfigDomains: {
					Type:     schema.TypeList,
					Computed: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
		ExactlyOneOf: acceptedEndpointConfig,
	}
}

func paramGcpEgressPrivateServiceConnectEndpointSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateServiceConnectEndpointTarget: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: `URI of the service attachment for the published service that the Private Service Connect Endpoint connects to, or "all-google-apis" for global Google APIs`,

					// Suppress the diff shown if the values are equivalent forms of "ALL_GOOGLE_APIS" and "all-google-apis"
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						const allGoogleApisNormalized = "all-google-apis"

						normalizedOld := strings.ReplaceAll(strings.ToLower(old), "_", "-")
						normalizedNew := strings.ReplaceAll(strings.ToLower(new), "_", "-")
						if normalizedOld == allGoogleApisNormalized && normalizedNew == allGoogleApisNormalized {
							return true
						}
						return false
					},
				},
				paramPrivateServiceConnectEndpointConnectionId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Connection ID of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
				paramPrivateServiceConnectEndpointIpAddress: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "IP address of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
				paramPrivateServiceConnectEndpointName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Name of the Private Service Connect Endpoint that is connected to the endpoint target.",
				},
			},
		},
		ExactlyOneOf: acceptedEndpointConfig,
	}
}

func paramAwsPrivateNetworkInterfaceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramNetworkInterfaces: {
					Type:        schema.TypeSet,
					Required:    true,
					MinItems:    6,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "List of the IDs of the Elastic Network Interfaces.",
				},
				paramAccount: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The AWS account ID associated with the ENIs you are using for the Confluent Private Network Interface.",
				},
			},
		},
		ExactlyOneOf: acceptedEndpointConfig,
	}
}

func accessPointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	gatewayId := extractStringValueFromBlock(d, paramGateway, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	isAwsEgressPrivateLinkEndpoint := len(d.Get(paramAwsEgressPrivateLinkEndpoint).([]interface{})) > 0
	isAzureEgressPrivateLinkEndpoint := len(d.Get(paramAzureEgressPrivateLinkEndpoint).([]interface{})) > 0
	isAwsPrivateNetworkInterface := len(d.Get(paramAwsPrivateNetworkInterface).([]interface{})) > 0
	isGcpEgressPrivateServiceConnectEndpoint := len(d.Get(paramGcpEgressPrivateServiceConnectEndpoint).([]interface{})) > 0

	spec := netap.NewNetworkingV1AccessPointSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	spec.SetGateway(netap.ObjectReference{Id: gatewayId})
	spec.SetEnvironment(netap.ObjectReference{Id: environmentId})

	config := netap.NetworkingV1AccessPointSpecConfigOneOf{}
	if isAwsEgressPrivateLinkEndpoint {
		enableHighAvailability := d.Get(fmt.Sprintf("%s.0.%s", paramAwsEgressPrivateLinkEndpoint, paramEnableHighAvailability)).(bool)
		config.NetworkingV1AwsEgressPrivateLinkEndpoint = &netap.NetworkingV1AwsEgressPrivateLinkEndpoint{
			Kind:                   awsEgressPrivateLinkEndpoint,
			VpcEndpointServiceName: extractStringValueFromBlock(d, paramAwsEgressPrivateLinkEndpoint, paramVpcEndpointServiceName),
			EnableHighAvailability: netap.PtrBool(enableHighAvailability),
		}
		spec.SetConfig(config)
	} else if isAzureEgressPrivateLinkEndpoint {
		config.NetworkingV1AzureEgressPrivateLinkEndpoint = &netap.NetworkingV1AzureEgressPrivateLinkEndpoint{
			Kind:                         azureEgressPrivateLinkEndpoint,
			PrivateLinkServiceResourceId: extractStringValueFromBlock(d, paramAzureEgressPrivateLinkEndpoint, paramPrivateLinkServiceResourceId),
		}

		privateLinkSubresourceName := extractStringValueFromBlock(d, paramAzureEgressPrivateLinkEndpoint, paramPrivateLinkSubresourceName)
		if privateLinkSubresourceName != "" {
			config.NetworkingV1AzureEgressPrivateLinkEndpoint.SetPrivateLinkSubresourceName(privateLinkSubresourceName)
		}
		spec.SetConfig(config)
	} else if isAwsPrivateNetworkInterface {
		networkInterfaces := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAwsPrivateNetworkInterface, paramNetworkInterfaces)).(*schema.Set).List())
		config.NetworkingV1AwsPrivateNetworkInterface = &netap.NetworkingV1AwsPrivateNetworkInterface{
			Kind:              awsPrivateNetworkInterface,
			NetworkInterfaces: &networkInterfaces,
			Account:           netap.PtrString(extractStringValueFromBlock(d, paramAwsPrivateNetworkInterface, paramAccount)),
		}
		spec.SetConfig(config)
	} else if isGcpEgressPrivateServiceConnectEndpoint {
		config.NetworkingV1GcpEgressPrivateServiceConnectEndpoint = &netap.NetworkingV1GcpEgressPrivateServiceConnectEndpoint{
			Kind:                                gcpEgressPrivateServiceConnectEndpoint,
			PrivateServiceConnectEndpointTarget: extractStringValueFromBlock(d, paramGcpEgressPrivateServiceConnectEndpoint, paramPrivateServiceConnectEndpointTarget),
		}
		spec.SetConfig(config)
	} else {
		return diag.Errorf("None of %q, %q, %q blocks was provided for confluent_access_point resource", paramAwsEgressPrivateLinkEndpoint, paramAzureEgressPrivateLinkEndpoint, paramAwsPrivateNetworkInterface)
	}

	createAccessPointRequest := netap.NetworkingV1AccessPoint{Spec: spec}
	createAccessPointRequestJson, err := json.Marshal(createAccessPointRequest)
	if err != nil {
		return diag.Errorf("error creating Access Point: error marshaling %#v to json: %s", createAccessPointRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Access Point: %s", createAccessPointRequestJson))

	req := c.netAccessPointClient.AccessPointsNetworkingV1Api.CreateNetworkingV1AccessPoint(c.netAPApiContext(ctx)).NetworkingV1AccessPoint(createAccessPointRequest)
	createdAccessPoint, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Access Point %q: %s", createdAccessPoint.GetId(), createDescriptiveError(err, resp))
	}
	d.SetId(createdAccessPoint.GetId())

	if err := waitForAccessPointToProvision(c.netAPApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Access Point %q to provision: %s", d.Id(), createDescriptiveError(err, resp))
	}

	createdAccessPointJson, err := json.Marshal(createdAccessPoint)
	if err != nil {
		return diag.Errorf("error creating Access Point %q: error marshaling %#v to json: %s", d.Id(), createdAccessPoint, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Access Point %q: %s", d.Id(), createdAccessPointJson), map[string]interface{}{accessPointKey: d.Id()})

	return accessPointRead(ctx, d, meta)
}

func executeAccessPointRead(ctx context.Context, c *Client, environmentId string, accessPointId string) (netap.NetworkingV1AccessPoint, *http.Response, error) {
	req := c.netAccessPointClient.AccessPointsNetworkingV1Api.GetNetworkingV1AccessPoint(c.netAPApiContext(ctx), accessPointId).Environment(environmentId)
	return req.Execute()
}

func accessPointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Access Point %q", d.Id()), map[string]interface{}{accessPointKey: d.Id()})

	accessPointId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readAccessPointAndSetAttributes(ctx, d, meta, environmentId, accessPointId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Access Point %q: %s", accessPointId, createDescriptiveError(err)))
	}

	return nil
}

func readAccessPointAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, accessPointId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	accessPoint, resp, err := executeAccessPointRead(c.netAPApiContext(ctx), c, environmentId, accessPointId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Access Point %q: %s", accessPointId, createDescriptiveError(err)), map[string]interface{}{accessPointKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Access Point %q in TF state because Access Point could not be found on the server", d.Id()), map[string]interface{}{accessPointKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	accessPointJson, err := json.Marshal(accessPoint)
	if err != nil {
		return nil, fmt.Errorf("error reading Access Point %q: error marshaling %#v to json: %s", accessPointId, accessPoint, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Access Point %q: %s", d.Id(), accessPointJson), map[string]interface{}{accessPointKey: d.Id()})

	if _, err := setAccessPointAttributes(d, accessPoint); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Access Point %q", accessPointId), map[string]interface{}{accessPointKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func accessPointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Access Point %q", d.Id()), map[string]interface{}{accessPointKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netAccessPointClient.AccessPointsNetworkingV1Api.DeleteNetworkingV1AccessPoint(c.netAPApiContext(ctx), d.Id()).Environment(environmentId)
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Access Point %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Access Point %q", d.Id()), map[string]interface{}{accessPointKey: d.Id()})

	return nil
}

func accessPointUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramAwsPrivateNetworkInterface) {
		return diag.Errorf("error updating Access Point %q: only %q, %q attributes can be updated for Access Point", d.Id(), paramDisplayName, paramNetworkInterfaces)
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	updateAccessPoint := netap.NewNetworkingV1AccessPointUpdate()

	updateAccessPointSpec := netap.NewNetworkingV1AccessPointSpecUpdate()
	updateAccessPointSpec.SetEnvironment(netap.ObjectReference{Id: environmentId})
	if d.HasChange(paramDisplayName) {
		updateAccessPointSpec.SetDisplayName(d.Get(paramDisplayName).(string))
	}

	if d.HasChange(paramAwsPrivateNetworkInterface) && d.HasChange(fmt.Sprintf("%s.0.%s", paramAwsPrivateNetworkInterface, paramNetworkInterfaces)) {
		networkInterfaces := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAwsPrivateNetworkInterface, paramNetworkInterfaces)).(*schema.Set).List())
		updateAccessPointSpec.SetConfig(netap.NetworkingV1AwsPrivateNetworkInterfaceAsNetworkingV1AccessPointSpecUpdateConfigOneOf(&netap.NetworkingV1AwsPrivateNetworkInterface{
			Kind:              paramAwsPrivateNetworkInterface,
			NetworkInterfaces: &networkInterfaces,
		}))
	}

	updateAccessPoint.SetSpec(*updateAccessPointSpec)
	updateAccessPointRequestJson, err := json.Marshal(updateAccessPoint)
	if err != nil {
		return diag.Errorf("error updating Access Point %q: error marshaling %#v to json: %s", d.Id(), updateAccessPointRequestJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Access Point %q: %s", d.Id(), updateAccessPointRequestJson), map[string]interface{}{accessPointKey: d.Id()})

	c := meta.(*Client)
	req := c.netAccessPointClient.AccessPointsNetworkingV1Api.UpdateNetworkingV1AccessPoint(c.netAPApiContext(ctx), d.Id()).NetworkingV1AccessPointUpdate(*updateAccessPoint)
	updatedAccessPoint, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Access Point %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedAccessPointJson, err := json.Marshal(updatedAccessPoint)
	if err != nil {
		return diag.Errorf("error updating Access Point %q: error marshaling %#v to json: %s", d.Id(), updatedAccessPoint, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Access Point %q: %s", d.Id(), updatedAccessPointJson), map[string]interface{}{accessPointKey: d.Id()})
	return accessPointRead(ctx, d, meta)
}

func accessPointImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Access Point %q", d.Id()), map[string]interface{}{accessPointKey: d.Id()})

	envIDAndAccessPointId := d.Id()
	parts := strings.Split(envIDAndAccessPointId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Access Point: invalid format: expected '<env ID>/<Access Point ID>'")
	}

	environmentId := parts[0]
	accessPointId := parts[1]
	d.SetId(accessPointId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readAccessPointAndSetAttributes(ctx, d, meta, environmentId, accessPointId); err != nil {
		return nil, fmt.Errorf("error importing Access Point %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Access Point %q", d.Id()), map[string]interface{}{accessPointKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setAccessPointAttributes(d *schema.ResourceData, accessPoint netap.NetworkingV1AccessPoint) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, accessPoint.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if accessPoint.Spec.Config.NetworkingV1AwsEgressPrivateLinkEndpoint != nil && accessPoint.Status.Config.NetworkingV1AwsEgressPrivateLinkEndpointStatus != nil {
		if err := d.Set(paramAwsEgressPrivateLinkEndpoint, []interface{}{map[string]interface{}{
			paramVpcEndpointServiceName: accessPoint.Spec.Config.NetworkingV1AwsEgressPrivateLinkEndpoint.GetVpcEndpointServiceName(),
			paramVpcEndpointId:          accessPoint.Status.Config.NetworkingV1AwsEgressPrivateLinkEndpointStatus.GetVpcEndpointId(),
			paramVpcEndpointDnsName:     accessPoint.Status.Config.NetworkingV1AwsEgressPrivateLinkEndpointStatus.GetVpcEndpointDnsName(),
			paramEnableHighAvailability: accessPoint.Spec.Config.NetworkingV1AwsEgressPrivateLinkEndpoint.GetEnableHighAvailability(),
		}}); err != nil {
			return nil, err
		}
	} else if accessPoint.Spec.Config.NetworkingV1AzureEgressPrivateLinkEndpoint != nil && accessPoint.Status.Config.NetworkingV1AzureEgressPrivateLinkEndpointStatus != nil {
		if err := d.Set(paramAzureEgressPrivateLinkEndpoint, []interface{}{map[string]interface{}{
			paramPrivateLinkServiceResourceId:          accessPoint.Spec.Config.NetworkingV1AzureEgressPrivateLinkEndpoint.GetPrivateLinkServiceResourceId(),
			paramPrivateLinkSubresourceName:            accessPoint.Spec.Config.NetworkingV1AzureEgressPrivateLinkEndpoint.GetPrivateLinkSubresourceName(),
			paramPrivateEndpointResourceId:             accessPoint.Status.Config.NetworkingV1AzureEgressPrivateLinkEndpointStatus.GetPrivateEndpointResourceId(),
			paramPrivateEndpointDomain:                 accessPoint.Status.Config.NetworkingV1AzureEgressPrivateLinkEndpointStatus.GetPrivateEndpointDomain(),
			paramPrivateEndpointIpAddress:              accessPoint.Status.Config.NetworkingV1AzureEgressPrivateLinkEndpointStatus.GetPrivateEndpointIpAddress(),
			paramPrivateEndpointCustomDnsConfigDomains: accessPoint.Status.Config.NetworkingV1AzureEgressPrivateLinkEndpointStatus.GetPrivateEndpointCustomDnsConfigDomains(),
		}}); err != nil {
			return nil, err
		}
	} else if accessPoint.Spec.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpoint != nil && accessPoint.Status.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpointStatus != nil {
		if err := d.Set(paramGcpEgressPrivateServiceConnectEndpoint, []interface{}{map[string]interface{}{
			paramPrivateServiceConnectEndpointTarget:       accessPoint.Spec.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpoint.GetPrivateServiceConnectEndpointTarget(),
			paramPrivateServiceConnectEndpointName:         accessPoint.Status.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpointStatus.GetPrivateServiceConnectEndpointName(),
			paramPrivateServiceConnectEndpointConnectionId: accessPoint.Status.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpointStatus.GetPrivateServiceConnectEndpointConnectionId(),
			paramPrivateServiceConnectEndpointIpAddress:    accessPoint.Status.Config.NetworkingV1GcpEgressPrivateServiceConnectEndpointStatus.GetPrivateServiceConnectEndpointIpAddress(),
		}}); err != nil {
			return nil, err
		}
	} else if accessPoint.Spec.Config.NetworkingV1AwsPrivateNetworkInterface != nil {
		if err := d.Set(paramAwsPrivateNetworkInterface, []interface{}{map[string]interface{}{
			paramNetworkInterfaces: accessPoint.Spec.Config.NetworkingV1AwsPrivateNetworkInterface.GetNetworkInterfaces(),
			paramAccount:           accessPoint.Spec.Config.NetworkingV1AwsPrivateNetworkInterface.GetAccount(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramGateway, paramId, accessPoint.Spec.Gateway.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, accessPoint.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(accessPoint.GetId())
	return d, nil
}
