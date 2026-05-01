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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	networkinggatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
)

var acceptedGatewayTypes = []string{paramAwsEgressPrivateLinkGateway, paramAwsIngressPrivateLinkGateway, paramAwsPrivateNetworkInterfaceGateway, paramAzureEgressPrivateLinkGateway, paramAzureIngressPrivateLinkGateway, paramGcpIngressPrivateServiceConnectGateway}

func gatewayResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: gatewayCreate,
		ReadContext:   gatewayRead,
		UpdateContext: gatewayUpdate,
		DeleteContext: gatewayDelete,
		Importer: &schema.ResourceImporter{
			StateContext: gatewayImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A name for the Gateway.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramEnvironment:                            environmentSchema(),
			paramAwsEgressPrivateLinkGateway:            awsEgressPrivateLinkGatewaySchema(),
			paramAwsIngressPrivateLinkGateway:           awsIngressPrivateLinkGatewaySchema(),
			paramAwsPrivateNetworkInterfaceGateway:      awsPrivateNetworkInterfaceGatewaySchema(),
			paramAzureEgressPrivateLinkGateway:          azureEgressPrivateLinkGatewaySchema(),
			paramAzureIngressPrivateLinkGateway:         azureIngressPrivateLinkGatewaySchema(),
			paramGcpIngressPrivateServiceConnectGateway: gcpIngressPrivateServiceConnectGatewaySchema(),
		},
	}
}

func awsEgressPrivateLinkGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramPrincipalArn: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func awsIngressPrivateLinkGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "AWS region of the Ingress Private Link Gateway.",
				},
				paramVpcEndpointServiceName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the AWS VPC Endpoint Service that can be used to establish connections for all zones.",
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func azureEgressPrivateLinkGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramSubscription: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func awsPrivateNetworkInterfaceGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "AWS region of the Private Network Interface Gateway.",
				},
				paramZones: {
					Type:        schema.TypeSet,
					Required:    true,
					ForceNew:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "AWS availability zone ids of the Private Network Interface Gateway.",
				},
				paramAccount: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The AWS account ID associated with the Private Network Interface Gateway.",
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func azureIngressPrivateLinkGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Azure region of the Ingress Private Link Gateway.",
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						return strings.EqualFold(old, new)
					},
				},
				paramPrivateLinkServiceAlias: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Alias of the Confluent Cloud Private Link Service.",
				},
				paramPrivateLinkServiceResourceId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Resource ID of the Confluent Cloud Private Link Service.",
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func gcpIngressPrivateServiceConnectGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		ForceNew: true,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRegion: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "GCP region of the Ingress Private Service Connect Gateway.",
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						return strings.EqualFold(old, new)
					},
				},
				paramPrivateServiceConnectServiceAttachment: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "URI of the Private Service Connect Service Attachment in Confluent Cloud.",
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedGatewayTypes,
	}
}

func gatewayCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	createGatewayRequest := networkinggatewayv1.NewNetworkingV1Gateway()
	createGatewayRequest.Spec = networkinggatewayv1.NewNetworkingV1GatewaySpec()
	createGatewayRequest.Spec.SetEnvironment(networkinggatewayv1.ObjectReference{Id: environmentId})
	createGatewayRequest.Spec.SetDisplayName(d.Get(paramDisplayName).(string))

	isAwsEgressPrivateLink := len(d.Get(paramAwsEgressPrivateLinkGateway).([]interface{})) > 0
	isAwsIngressPrivateLink := len(d.Get(paramAwsIngressPrivateLinkGateway).([]interface{})) > 0
	isAwsPrivateNetworkInterface := len(d.Get(paramAwsPrivateNetworkInterfaceGateway).([]interface{})) > 0
	isAzureEgressPrivateLink := len(d.Get(paramAzureEgressPrivateLinkGateway).([]interface{})) > 0
	isAzureIngressPrivateLink := len(d.Get(paramAzureIngressPrivateLinkGateway).([]interface{})) > 0
	isGcpIngressPrivateServiceConnect := len(d.Get(paramGcpIngressPrivateServiceConnectGateway).([]interface{})) > 0

	if isAwsEgressPrivateLink {
		region := extractStringValueFromBlock(d, paramAwsEgressPrivateLinkGateway, paramRegion)
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1AwsEgressPrivateLinkGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1AwsEgressPrivateLinkGatewaySpec(
			awsEgressPrivateLinkGatewaySpecKind,
			region,
		)))
	} else if isAwsIngressPrivateLink {
		region := extractStringValueFromBlock(d, paramAwsIngressPrivateLinkGateway, paramRegion)
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1AwsIngressPrivateLinkGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1AwsIngressPrivateLinkGatewaySpec(
			awsIngressPrivateLinkGatewaySpecKind,
			region,
		)))
	} else if isAwsPrivateNetworkInterface {
		region := extractStringValueFromBlock(d, paramAwsPrivateNetworkInterfaceGateway, paramRegion)
		zones := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAwsPrivateNetworkInterfaceGateway, paramZones)).(*schema.Set).List())
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1AwsPrivateNetworkInterfaceGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1AwsPrivateNetworkInterfaceGatewaySpec(
			awsPrivateNetworkInterfaceGatewaySpecKind,
			region,
			zones,
		)))
	} else if isAzureEgressPrivateLink {
		region := extractStringValueFromBlock(d, paramAzureEgressPrivateLinkGateway, paramRegion)
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1AzureEgressPrivateLinkGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1AzureEgressPrivateLinkGatewaySpec(
			azureEgressPrivateLinkGatewaySpecKind,
			region,
		)))
	} else if isAzureIngressPrivateLink {
		region := extractStringValueFromBlock(d, paramAzureIngressPrivateLinkGateway, paramRegion)
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1AzureIngressPrivateLinkGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1AzureIngressPrivateLinkGatewaySpec(
			azureIngressPrivateLinkGatewaySpecKind,
			region,
		)))
	} else if isGcpIngressPrivateServiceConnect {
		region := extractStringValueFromBlock(d, paramGcpIngressPrivateServiceConnectGateway, paramRegion)
		createGatewayRequest.Spec.SetConfig(networkinggatewayv1.NetworkingV1GcpIngressPrivateServiceConnectGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(networkinggatewayv1.NewNetworkingV1GcpIngressPrivateServiceConnectGatewaySpec(
			gcpIngressPrivateServiceConnectGatewaySpecKind,
			region,
		)))
	}

	createGatewayRequestJson, err := json.Marshal(createGatewayRequest)
	if err != nil {
		return diag.Errorf("error creating Gateway: error marshaling %#v to json: %s", createGatewayRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Gateway: %s", createGatewayRequestJson))

	req := c.networkingGatewayV1Client.GatewaysNetworkingV1Api.CreateNetworkingV1Gateway(c.networkingGatewayV1ApiContext(ctx)).NetworkingV1Gateway(*createGatewayRequest)
	createdGateway, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Gateway %q: %s", createdGateway.GetId(), createDescriptiveError(err, resp))
	}
	d.SetId(createdGateway.GetId())

	if err := waitForGatewayToProvision(c.networkingGatewayV1ApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Gateway %q to provision: %s", d.Id(), createDescriptiveError(err, resp))
	}

	createdGatewayJson, err := json.Marshal(createdGateway)
	if err != nil {
		return diag.Errorf("error creating Gateway %q: error marshaling %#v to json: %s", d.Id(), createdGateway, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Gateway %q: %s", d.Id(), createdGatewayJson), map[string]interface{}{gatewayKey: d.Id()})

	return gatewayRead(ctx, d, meta)
}

func executeGatewayRead(ctx context.Context, c *Client, environmentId, gatewayId string) (networkinggatewayv1.NetworkingV1Gateway, *http.Response, error) {
	req := c.networkingGatewayV1Client.GatewaysNetworkingV1Api.GetNetworkingV1Gateway(c.networkingGatewayV1ApiContext(ctx), gatewayId).Environment(environmentId)
	return req.Execute()
}

func gatewayRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Gateway %q", d.Id()), map[string]interface{}{gatewayKey: d.Id()})

	gatewayId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readGatewayAndSetAttributes(ctx, d, meta, environmentId, gatewayId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Gateway %q: %s", gatewayId, createDescriptiveError(err)))
	}

	return nil
}

func readGatewayAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, gatewayId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	gateway, resp, err := executeGatewayRead(c.networkingGatewayV1ApiContext(ctx), c, environmentId, gatewayId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Gateway %q: %s", gatewayId, createDescriptiveError(err)), map[string]interface{}{gatewayKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Gateway %q in TF state because Gateway could not be found on the server", d.Id()), map[string]interface{}{gatewayKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	gatewayJson, err := json.Marshal(gateway)
	if err != nil {
		return nil, fmt.Errorf("error reading Gateway %q: error marshaling %#v to json: %s", gatewayId, gateway, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Gateway %q: %s", d.Id(), gatewayJson), map[string]interface{}{gatewayKey: d.Id()})

	if _, err := setGatewayAttributes(d, gateway); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Gateway %q", gatewayId), map[string]interface{}{gatewayKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setGatewayAttributes(d *schema.ResourceData, gateway networkinggatewayv1.NetworkingV1Gateway) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, gateway.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if gateway.Spec.GetConfig().NetworkingV1AwsEgressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AwsEgressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAwsEgressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:       gateway.Spec.Config.NetworkingV1AwsEgressPrivateLinkGatewaySpec.GetRegion(),
			paramPrincipalArn: gateway.Status.CloudGateway.NetworkingV1AwsEgressPrivateLinkGatewayStatus.GetPrincipalArn(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AwsIngressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AwsIngressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAwsIngressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:                 gateway.Spec.Config.NetworkingV1AwsIngressPrivateLinkGatewaySpec.GetRegion(),
			paramVpcEndpointServiceName: gateway.Status.CloudGateway.NetworkingV1AwsIngressPrivateLinkGatewayStatus.GetVpcEndpointServiceName(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AwsPeeringGatewaySpec != nil {
		if err := d.Set(paramAwsPeeringGateway, []interface{}{map[string]interface{}{
			paramRegion: gateway.Spec.Config.NetworkingV1AwsPeeringGatewaySpec.GetRegion(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec != nil {
		if err := d.Set(paramAwsPrivateNetworkInterfaceGateway, []interface{}{map[string]interface{}{
			paramRegion:  gateway.Spec.Config.NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec.GetRegion(),
			paramZones:   gateway.Spec.Config.NetworkingV1AwsPrivateNetworkInterfaceGatewaySpec.GetZones(),
			paramAccount: gateway.Status.CloudGateway.NetworkingV1AwsPrivateNetworkInterfaceGatewayStatus.GetAccount(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AzureEgressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AzureEgressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAzureEgressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:       gateway.Spec.Config.NetworkingV1AzureEgressPrivateLinkGatewaySpec.GetRegion(),
			paramSubscription: gateway.Status.CloudGateway.NetworkingV1AzureEgressPrivateLinkGatewayStatus.GetSubscription(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AzureIngressPrivateLinkGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1AzureIngressPrivateLinkGatewayStatus != nil {
		if err := d.Set(paramAzureIngressPrivateLinkGateway, []interface{}{map[string]interface{}{
			paramRegion:                       gateway.Spec.Config.NetworkingV1AzureIngressPrivateLinkGatewaySpec.GetRegion(),
			paramPrivateLinkServiceAlias:      gateway.Status.CloudGateway.NetworkingV1AzureIngressPrivateLinkGatewayStatus.GetPrivateLinkServiceAlias(),
			paramPrivateLinkServiceResourceId: gateway.Status.CloudGateway.NetworkingV1AzureIngressPrivateLinkGatewayStatus.GetPrivateLinkServiceResourceId(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1AzurePeeringGatewaySpec != nil {
		if err := d.Set(paramAzurePeeringGateway, []interface{}{map[string]interface{}{
			paramRegion: gateway.Spec.Config.NetworkingV1AzurePeeringGatewaySpec.GetRegion(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1GcpPeeringGatewaySpec != nil {
		if err := d.Set(paramGcpPeeringGateway, []interface{}{map[string]interface{}{
			paramRegion:       gateway.Spec.Config.NetworkingV1GcpPeeringGatewaySpec.GetRegion(),
			paramIAMPrincipal: gateway.Status.CloudGateway.NetworkingV1GcpPeeringGatewayStatus.GetIamPrincipal(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1GcpEgressPrivateServiceConnectGatewaySpec != nil {
		if err := d.Set(paramGcpEgressPrivateServiceConnectGateway, []interface{}{map[string]interface{}{
			paramRegion:  gateway.Spec.Config.NetworkingV1GcpEgressPrivateServiceConnectGatewaySpec.GetRegion(),
			paramProject: gateway.Status.CloudGateway.NetworkingV1GcpEgressPrivateServiceConnectGatewayStatus.GetProject(),
		}}); err != nil {
			return nil, err
		}
	} else if gateway.Spec.GetConfig().NetworkingV1GcpIngressPrivateServiceConnectGatewaySpec != nil && gateway.Status.GetCloudGateway().NetworkingV1GcpIngressPrivateServiceConnectGatewayStatus != nil {
		if err := d.Set(paramGcpIngressPrivateServiceConnectGateway, []interface{}{map[string]interface{}{
			paramRegion: gateway.Spec.Config.NetworkingV1GcpIngressPrivateServiceConnectGatewaySpec.GetRegion(),
			paramPrivateServiceConnectServiceAttachment: gateway.Status.CloudGateway.NetworkingV1GcpIngressPrivateServiceConnectGatewayStatus.GetPrivateServiceConnectServiceAttachment(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, gateway.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(gateway.GetId())
	return d, nil
}

func gatewayDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Gateway %q", d.Id()), map[string]interface{}{gatewayKey: d.Id()})
	c := meta.(*Client)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	req := c.networkingGatewayV1Client.GatewaysNetworkingV1Api.DeleteNetworkingV1Gateway(c.networkingGatewayV1ApiContext(ctx), d.Id()).Environment(environmentId)
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Gateway %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Gateway %q", d.Id()), map[string]interface{}{gatewayKey: d.Id()})

	return nil
}

func gatewayUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName) {
		return diag.Errorf("error updating Gateway %q: only %q attribute can be updated for Gateway", d.Id(), paramDisplayName)
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	updateGateway := networkinggatewayv1.NewNetworkingV1GatewayUpdate()
	updateGateway.Spec = networkinggatewayv1.NewNetworkingV1GatewaySpecUpdate()

	updateGateway.Spec.SetDisplayName(d.Get(paramDisplayName).(string))
	updateGateway.Spec.SetEnvironment(networkinggatewayv1.ObjectReference{
		Id: environmentId,
	})

	updateGatewayJson, err := json.Marshal(updateGateway)
	if err != nil {
		return diag.Errorf("error updating Gateway %q: error marshaling %#v to json: %s", d.Id(), updateGateway, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Gateway %q: %s", d.Id(), updateGatewayJson), map[string]interface{}{gatewayKey: d.Id()})

	c := meta.(*Client)
	req := c.networkingGatewayV1Client.GatewaysNetworkingV1Api.UpdateNetworkingV1Gateway(c.networkingGatewayV1ApiContext(ctx), d.Id()).NetworkingV1GatewayUpdate(*updateGateway)
	updatedGateway, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Gateway %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	UpdatedGatewayJson, err := json.Marshal(updatedGateway)
	if err != nil {
		return diag.Errorf("error updating Gateway %q: error marshaling %#v to json: %s", d.Id(), updatedGateway, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Gateway %q: %s", d.Id(), UpdatedGatewayJson), map[string]interface{}{gatewayKey: d.Id()})
	return gatewayRead(ctx, d, meta)
}

func gatewayImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Gateway %q", d.Id()), map[string]interface{}{gatewayKey: d.Id()})

	envIDAndGatewayId := d.Id()
	parts := strings.Split(envIDAndGatewayId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Gateway: invalid format: expected '<env ID>/<gateway ID>'")
	}

	environmentId := parts[0]
	gatewayId := parts[1]
	d.SetId(gatewayId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readGatewayAndSetAttributes(ctx, d, meta, environmentId, gatewayId); err != nil {
		return nil, fmt.Errorf("error importing Gateway %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Gateway %q", d.Id()), map[string]interface{}{gatewayKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
