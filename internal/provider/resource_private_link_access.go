// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramNetwork               = "network"
	paramAccount               = "account"
	paramSubscription          = "subscription"
	paramAzure                 = "azure"
	paramAws                   = "aws"
	awsPrivateLinkAccessKind   = "AwsPrivateLinkAccess"
	azurePrivateLinkAccessKind = "AzurePrivateLinkAccess"
	gcpPrivateLinkAccessKind   = "GcpPrivateServiceConnectAccess"
)

var acceptedCloudProvidersForPrivateLinkAccess = []string{paramAws, paramAzure, paramGcp}

func privateLinkAccessResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: privateLinkAccessCreate,
		ReadContext:   privateLinkAccessRead,
		UpdateContext: privateLinkAccessUpdate,
		DeleteContext: privateLinkAccessDelete,
		Importer: &schema.ResourceImporter{
			StateContext: privateLinkAccessImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the PrivateLink access.",
				Optional:    true,
				Computed:    true,
			},
			paramAws:         awsSchema(),
			paramAzure:       azureSchema(),
			paramGcp:         gcpSchema(),
			paramNetwork:     requiredNetworkSchema(),
			paramEnvironment: environmentSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(networkingAPICreateTimeout),
			Delete: schema.DefaultTimeout(networkingAPIDeleteTimeout),
		},
	}
}

func awsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAccount: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "AWS Account ID to allow for PrivateLink access. Find here (https://console.aws.amazon.com/billing/home?#/account) under My Account in your AWS Management Console.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\d{12}$`), "AWS Account ID is expected to consist of exactly 12 digits."),
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPrivateLinkAccess,
	}
}

func privateLinkAccessCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	// Non-empty value means AWS account attribute has been set
	awsAccount := extractStringValueFromBlock(d, paramAws, paramAccount)
	// Non-empty value means Azure subscription attribute has been set
	azureSubscription := extractStringValueFromBlock(d, paramAzure, paramSubscription)
	// Non-empty value means GCP project attribute has been set
	gcpProject := extractStringValueFromBlock(d, paramGcp, paramProject)

	spec := net.NewNetworkingV1PrivateLinkAccessSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	if awsAccount != "" {
		spec.SetCloud(net.NetworkingV1PrivateLinkAccessSpecCloudOneOf{NetworkingV1AwsPrivateLinkAccess: net.NewNetworkingV1AwsPrivateLinkAccess(awsPrivateLinkAccessKind, awsAccount)})
	} else if azureSubscription != "" {
		spec.SetCloud(net.NetworkingV1PrivateLinkAccessSpecCloudOneOf{NetworkingV1AzurePrivateLinkAccess: net.NewNetworkingV1AzurePrivateLinkAccess(azurePrivateLinkAccessKind, azureSubscription)})
	} else if gcpProject != "" {
		spec.SetCloud(net.NetworkingV1PrivateLinkAccessSpecCloudOneOf{NetworkingV1GcpPrivateServiceConnectAccess: net.NewNetworkingV1GcpPrivateServiceConnectAccess(gcpPrivateLinkAccessKind, gcpProject)})
	}
	spec.SetNetwork(net.ObjectReference{Id: networkId})
	spec.SetEnvironment(net.ObjectReference{Id: environmentId})

	createPrivateLinkAccessRequest := net.NetworkingV1PrivateLinkAccess{Spec: spec}
	createPrivateLinkAccessRequestJson, err := json.Marshal(createPrivateLinkAccessRequest)
	if err != nil {
		return diag.Errorf("error creating Private Link Access: error marshaling %#v to json: %s", createPrivateLinkAccessRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Private Link Access: %s", createPrivateLinkAccessRequestJson))

	createdPrivateLinkAccess, _, err := executePrivateLinkAccessCreate(c.netApiContext(ctx), c, createPrivateLinkAccessRequest)
	if err != nil {
		return diag.Errorf("error creating Private Link Access %q: %s", createdPrivateLinkAccess.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdPrivateLinkAccess.GetId())

	if err := waitForPrivateLinkAccessToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Private Link Access %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdPrivateLinkAccessJson, err := json.Marshal(createdPrivateLinkAccess)
	if err != nil {
		return diag.Errorf("error creating Private Link Access %q: error marshaling %#v to json: %s", d.Id(), createdPrivateLinkAccess, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Private Link Access %q: %s", d.Id(), createdPrivateLinkAccessJson), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	return privateLinkAccessRead(ctx, d, meta)
}

func executePrivateLinkAccessCreate(ctx context.Context, c *Client, privateLinkAccess net.NetworkingV1PrivateLinkAccess) (net.NetworkingV1PrivateLinkAccess, *http.Response, error) {
	req := c.netClient.PrivateLinkAccessesNetworkingV1Api.CreateNetworkingV1PrivateLinkAccess(c.netApiContext(ctx)).NetworkingV1PrivateLinkAccess(privateLinkAccess)
	return req.Execute()
}

func executePrivateLinkAccessRead(ctx context.Context, c *Client, environmentId string, privateLinkAccessId string) (net.NetworkingV1PrivateLinkAccess, *http.Response, error) {
	req := c.netClient.PrivateLinkAccessesNetworkingV1Api.GetNetworkingV1PrivateLinkAccess(c.netApiContext(ctx), privateLinkAccessId).Environment(environmentId)
	return req.Execute()
}

func privateLinkAccessRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	privateLinkAccessId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readPrivateLinkAccessAndSetAttributes(ctx, d, meta, environmentId, privateLinkAccessId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Private Link Access %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readPrivateLinkAccessAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, privateLinkAccessId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	privateLinkAccess, resp, err := executePrivateLinkAccessRead(c.netApiContext(ctx), c, environmentId, privateLinkAccessId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Private Link Access %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Private Link Access %q in TF state because Private Link Access could not be found on the server", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	privateLinkAccessJson, err := json.Marshal(privateLinkAccess)
	if err != nil {
		return nil, fmt.Errorf("error reading Private Link Access %q: error marshaling %#v to json: %s", privateLinkAccessId, privateLinkAccess, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Access %q: %s", d.Id(), privateLinkAccessJson), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	if _, err := setPrivateLinkAccessAttributes(d, privateLinkAccess); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setPrivateLinkAccessAttributes(d *schema.ResourceData, privateLinkAccess net.NetworkingV1PrivateLinkAccess) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, privateLinkAccess.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if privateLinkAccess.Spec.Cloud.NetworkingV1AwsPrivateLinkAccess != nil {
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramAccount: privateLinkAccess.Spec.Cloud.NetworkingV1AwsPrivateLinkAccess.GetAccount(),
		}}); err != nil {
			return nil, err
		}
	} else if privateLinkAccess.Spec.Cloud.NetworkingV1AzurePrivateLinkAccess != nil {
		if err := d.Set(paramAzure, []interface{}{map[string]interface{}{
			paramSubscription: privateLinkAccess.Spec.Cloud.NetworkingV1AzurePrivateLinkAccess.GetSubscription(),
		}}); err != nil {
			return nil, err
		}
	} else if privateLinkAccess.Spec.Cloud.NetworkingV1GcpPrivateServiceConnectAccess != nil {
		if err := d.Set(paramGcp, []interface{}{map[string]interface{}{
			paramProject: privateLinkAccess.Spec.Cloud.NetworkingV1GcpPrivateServiceConnectAccess.GetProject(),
		}}); err != nil {
			return nil, err
		}
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, privateLinkAccess.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, privateLinkAccess.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(privateLinkAccess.GetId())
	return d, nil
}

func privateLinkAccessDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netClient.PrivateLinkAccessesNetworkingV1Api.DeleteNetworkingV1PrivateLinkAccess(c.netApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Private Link Access %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForPrivateLinkAccessToBeDeleted(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Private Link Access %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	return nil
}

func privateLinkAccessUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Private Link Access %q: only %q attribute can be updated for Private Link Access", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	updatedDisplayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updatePrivateLinkAccessRequest := net.NewNetworkingV1PrivateLinkAccessUpdate()
	updateSpec := net.NewNetworkingV1PrivateLinkAccessSpecUpdate()
	updateSpec.SetDisplayName(updatedDisplayName)
	updateSpec.SetEnvironment(net.ObjectReference{Id: environmentId})
	updatePrivateLinkAccessRequest.SetSpec(*updateSpec)
	updatePrivateLinkAccessRequestJson, err := json.Marshal(updatePrivateLinkAccessRequest)
	if err != nil {
		return diag.Errorf("error updating Private Link Access %q: error marshaling %#v to json: %s", d.Id(), updatePrivateLinkAccessRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Private Link Access %q: %s", d.Id(), updatePrivateLinkAccessRequestJson), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	req := c.netClient.PrivateLinkAccessesNetworkingV1Api.UpdateNetworkingV1PrivateLinkAccess(c.netApiContext(ctx), d.Id()).NetworkingV1PrivateLinkAccessUpdate(*updatePrivateLinkAccessRequest)
	updatedPrivateLinkAccess, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Private Link Access %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedPrivateLinkAccessJson, err := json.Marshal(updatedPrivateLinkAccess)
	if err != nil {
		return diag.Errorf("error updating Private Link Access %q: error marshaling %#v to json: %s", d.Id(), updatedPrivateLinkAccess, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Private Link Access %q: %s", d.Id(), updatedPrivateLinkAccessJson), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})
	return privateLinkAccessRead(ctx, d, meta)
}

func privateLinkAccessImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})

	envIDAndPrivateLinkAccessId := d.Id()
	parts := strings.Split(envIDAndPrivateLinkAccessId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Private Link Access: invalid format: expected '<env ID>/<pla ID>'")
	}

	environmentId := parts[0]
	privateLinkAccessId := parts[1]
	d.SetId(privateLinkAccessId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readPrivateLinkAccessAndSetAttributes(ctx, d, meta, environmentId, privateLinkAccessId); err != nil {
		return nil, fmt.Errorf("error importing Private Link Access %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Private Link Access %q", d.Id()), map[string]interface{}{privateLinkAccessLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func azureSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Optional: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramSubscription: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Azure subscription to allow for PrivateLink access.",
					// TODO: add ValidateFunc
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPrivateLinkAccess,
	}
}

func gcpSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Optional: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProject: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The GCP project ID to allow for Private Service Connect access.",
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPrivateLinkAccess,
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func requiredNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Required:    true,
		ForceNew:    true,
		Description: "Network represents a network (VPC) in Confluent Cloud. All Networks exist within Confluent-managed cloud provider accounts.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The unique identifier for the network.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(n-|nr-)"), "the network ID must start with 'n-' or 'nr-'"),
				},
			},
		},
	}
}
