// Copyright 2021 Confluent Inc. All Rights Reserved.
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

	pi "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
)

const (
	paramIamRoleUrn          = "iam_role_arn"
	paramExternalId          = "external_id"
	paramCustomerRoleArn     = "customer_role_arn"
	paramUsages              = "usages"
	AwsIntegrationConfigKind = "AwsIntegrationConfig"
)

const (
	listProviderIntegrationsPageSize = 99
)

var acceptedProviderIntegrationConfig = []string{paramAws}

func providerIntegrationResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: providerIntegrationCreate,
		ReadContext:   providerIntegrationRead,
		DeleteContext: providerIntegrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: providerIntegrationImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID for provider integration.",
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The display name of provider integration.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramAws: awsProviderIntegrationConfigSchema(),
			paramUsages: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The usages of provider integration.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			paramEnvironment: environmentSchema(),
		},
	}
}

func awsProviderIntegrationConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramIamRoleUrn: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "IAM role ARN.",
				},
				paramExternalId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "External ID for the AWS role.",
				},
				paramCustomerRoleArn: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The AWS customer's IAM role ARN.",
				},
			},
		},
		Optional:     true,
		MinItems:     1,
		MaxItems:     1,
		ForceNew:     true,
		ExactlyOneOf: acceptedProviderIntegrationConfig,
		Description:  "Config objects represent AWS cloud provider specific configs.",
	}
}

func providerIntegrationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	config := pi.PimV1IntegrationConfigOneOf{}
	var cloud string
	isAwsConfigs := len(d.Get(paramAws).([]interface{})) > 0

	if isAwsConfigs {
		cloud = "aws"
		config.PimV1AwsIntegrationConfig = &pi.PimV1AwsIntegrationConfig{
			Kind:               AwsIntegrationConfigKind,
			CustomerIamRoleArn: pi.PtrString(extractStringValueFromBlock(d, paramAws, paramCustomerRoleArn)),
		}
	} else {
		return diag.Errorf("None of %q block was provided for confluent_provider_integration resource", paramAws)
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	// Create the provider integration request
	createPimRequest := pi.PimV1Integration{}
	createPimRequest.SetDisplayName(displayName)
	createPimRequest.SetProvider(cloud)
	createPimRequest.SetConfig(config)
	createPimRequest.SetEnvironment(pi.GlobalObjectReference{Id: environmentId})
	createPimRequestRequestJson, err := json.Marshal(createPimRequest)

	if err != nil {
		return diag.Errorf("error creating provider integration resource: error marshaling %#v to json: %s", createPimRequest, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating new provider integration resource: %s", createPimRequestRequestJson))
	createdPimResponse, resp, err := executeProviderIntegrationCreate(c.piApiContext(ctx), c, &createPimRequest)
	if err != nil {
		return diag.Errorf("error creating provider integration: %s", createDescriptiveError(err, resp))
	}

	d.SetId(createdPimResponse.GetId())
	createdPimResponseJson, err := json.Marshal(createdPimResponse)

	if err != nil {
		return diag.Errorf("error creating provider integration: error marshaling %#v to json: %s", createdPimResponseJson, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished creating provider integration %q: %s", d.Id(), createdPimResponseJson), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
	return providerIntegrationRead(ctx, d, meta)
}

func providerIntegrationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	pimId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration resource %q", pimId), map[string]interface{}{providerIntegrationLoggingKey: pimId})

	if _, err := readProviderIntegrationAndSetAttributes(ctx, d, meta, environmentId, pimId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading provider integration %q: %s", pimId, createDescriptiveError(err)))
	}

	return nil
}

func providerIntegrationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting provider integration %q", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	resp, err := executeProviderIntegrationDelete(ctx, c, environmentId, d.Id())
	if err != nil {
		return diag.Errorf("error deleting provider integration %q: %s", d.Id(), createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished deleting provider integration %q", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})

	return nil
}

func providerIntegrationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing provider integration %q", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})

	envIDAndComputePoolId := d.Id()
	parts := strings.Split(envIDAndComputePoolId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing provider integration: invalid format: expected '<env ID>/<provider integration ID>'")
	}

	environmentId := parts[0]
	providerIntegrationId := parts[1]
	d.SetId(providerIntegrationId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readProviderIntegrationAndSetAttributes(ctx, d, meta, environmentId, providerIntegrationId); err != nil {
		return nil, fmt.Errorf("error importing provider integration %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing provider integration %q", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readProviderIntegrationAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, id string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)
	pim, resp, err := executeProviderIntegrationRead(c.piApiContext(ctx), c, environmentId, id)

	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading provider integration %q: %s", id, createDescriptiveError(err, resp)), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing provider integration %q in TF state because provider integration could not be found on the server", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	pimJson, err := json.Marshal(pim)
	if err != nil {
		return nil, fmt.Errorf("error reading provider integration %q: error marshaling %#v to json: %s", id, pim, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration %q: %s", d.Id(), pimJson), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})

	if _, err := setProviderIntegrationAttributes(d, pim); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading provider integration %q", id), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func executeProviderIntegrationCreate(ctx context.Context, c *Client, createPimRequest *pi.PimV1Integration) (pi.PimV1Integration, *http.Response, error) {
	req := c.piClient.IntegrationsPimV1Api.CreatePimV1Integration(c.piApiContext(ctx)).PimV1Integration(*createPimRequest)
	return req.Execute()
}

func executeProviderIntegrationRead(ctx context.Context, c *Client, environmentId string, id string) (pi.PimV1Integration, *http.Response, error) {
	req := c.piClient.IntegrationsPimV1Api.GetPimV1Integration(c.piApiContext(ctx), id).Environment(environmentId)
	return req.Execute()
}

func executeProviderIntegrationDelete(ctx context.Context, c *Client, environmentId string, pimId string) (*http.Response, error) {
	req := c.piClient.IntegrationsPimV1Api.DeletePimV1Integration(c.piApiContext(ctx), pimId).Environment(environmentId)
	return req.Execute()
}

func executeListProviderIntegrations(ctx context.Context, c *Client, environmentId, pageToken string) (pi.PimV1IntegrationList, *http.Response, error) {
	if pageToken != "" {
		return c.piClient.IntegrationsPimV1Api.ListPimV1Integrations(c.piApiContext(ctx)).Environment(environmentId).PageSize(listProviderIntegrationsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.piClient.IntegrationsPimV1Api.ListPimV1Integrations(c.piApiContext(ctx)).Environment(environmentId).PageSize(listProviderIntegrationsPageSize).Execute()
	}
}

func setProviderIntegrationAttributes(d *schema.ResourceData, pim pi.PimV1Integration) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, pim.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := setProviderIntegrationConfigAttributes(d, pim.GetConfig()); err != nil {
		return nil, err
	}
	if err := d.Set(paramUsages, pim.GetUsages()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, pim.Environment.GetId(), d); err != nil {
		return nil, err
	}

	d.SetId(pim.GetId())
	return d, nil
}

func setProviderIntegrationConfigAttributes(d *schema.ResourceData, config pi.PimV1IntegrationConfigOneOf) error {
	if config.PimV1AwsIntegrationConfig != nil {
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramIamRoleUrn:      config.PimV1AwsIntegrationConfig.GetIamRoleArn(),
			paramExternalId:      config.PimV1AwsIntegrationConfig.GetExternalId(),
			paramCustomerRoleArn: config.PimV1AwsIntegrationConfig.GetCustomerIamRoleArn(),
		}}); err != nil {
			return err
		}
	}

	return nil
}
