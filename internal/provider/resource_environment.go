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
	org "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

const (
	paramStreamGovernance = "stream_governance"
)

func environmentResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: environmentCreate,
		ReadContext:   environmentRead,
		UpdateContext: environmentUpdate,
		DeleteContext: environmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: environmentImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Description:  "A human-readable name for the Environment.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
			},
			paramStreamGovernance: streamGovernanceConfigSchema(),
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Environment.",
			},
		},
	}
}

func environmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramStreamGovernance) {
		return diag.Errorf("error updating Environment %q: only %q or %q attributes can be updated for Environment", d.Id(), paramDisplayName, paramStreamGovernance)
	}

	updateEnvironmentRequest := org.NewOrgV2Environment()
	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateEnvironmentRequest.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(getNestedStreamGovernancePackageKey()) {
		updatedPackage := extractStringValueFromBlock(d, paramStreamGovernance, paramPackage)
		updateEnvironmentRequest.SetStreamGovernanceConfig(org.OrgV2StreamGovernanceConfig{
			Package: updatedPackage,
		})
	}
	updateEnvironmentRequestJson, err := json.Marshal(updateEnvironmentRequest)
	if err != nil {
		return diag.Errorf("error updating Environment %q: error marshaling %#v to json: %s", d.Id(), updateEnvironmentRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Environment %q: %s", d.Id(), updateEnvironmentRequestJson), map[string]interface{}{environmentLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedEnvironment, resp, err := c.orgClient.EnvironmentsOrgV2Api.UpdateOrgV2Environment(c.orgApiContext(ctx), d.Id()).OrgV2Environment(*updateEnvironmentRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Environment %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedEnvironmentJson, err := json.Marshal(updatedEnvironment)
	if err != nil {
		return diag.Errorf("error updating Environment %q: error marshaling %#v to json: %s", d.Id(), updatedEnvironment, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Environment %q: %s", d.Id(), updatedEnvironmentJson), map[string]interface{}{environmentLoggingKey: d.Id()})

	return environmentRead(ctx, d, meta)
}

func environmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	createEnvironmentRequest := org.NewOrgV2Environment()
	createEnvironmentRequest.SetDisplayName(displayName)
	if d.HasChange(getNestedStreamGovernancePackageKey()) {
		createEnvironmentRequest.SetStreamGovernanceConfig(org.OrgV2StreamGovernanceConfig{
			Package: extractStringValueFromBlock(d, paramStreamGovernance, paramPackage)})
	}
	createEnvironmentRequestJson, err := json.Marshal(createEnvironmentRequest)
	if err != nil {
		return diag.Errorf("error creating Environment %q: error marshaling %#v to json: %s", displayName, createEnvironmentRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Environment: %s", createEnvironmentRequestJson))

	createdEnvironment, resp, err := executeEnvironmentCreate(c.orgApiContext(ctx), c, createEnvironmentRequest)
	if err != nil {
		return diag.Errorf("error creating Environment %q: %s", displayName, createDescriptiveError(err, resp))
	}
	d.SetId(createdEnvironment.GetId())

	createdEnvironmentJson, err := json.Marshal(createdEnvironment)
	if err != nil {
		return diag.Errorf("error creating Environment %q: error marshaling %#v to json: %s", d.Id(), createdEnvironment, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Environment %q: %s", d.Id(), createdEnvironmentJson), map[string]interface{}{environmentLoggingKey: d.Id()})

	return environmentRead(ctx, d, meta)
}

func executeEnvironmentCreate(ctx context.Context, c *Client, environment *org.OrgV2Environment) (org.OrgV2Environment, *http.Response, error) {
	req := c.orgClient.EnvironmentsOrgV2Api.CreateOrgV2Environment(c.orgApiContext(ctx)).OrgV2Environment(*environment)
	return req.Execute()
}

func environmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.orgClient.EnvironmentsOrgV2Api.DeleteOrgV2Environment(c.orgApiContext(ctx), d.Id())
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Environment %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})

	return nil
}

func executeEnvironmentRead(ctx context.Context, c *Client, environmentId string) (org.OrgV2Environment, *http.Response, error) {
	req := c.orgClient.EnvironmentsOrgV2Api.GetOrgV2Environment(c.orgApiContext(ctx), environmentId)
	return req.Execute()
}

func environmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})
	c := meta.(*Client)
	environment, resp, err := executeEnvironmentRead(c.orgApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Environment %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{environmentLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Environment %q in TF state because Environment could not be found on the server", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err, resp))
	}
	environmentJson, err := json.Marshal(environment)
	if err != nil {
		return diag.Errorf("error reading Environment %q: error marshaling %#v to json: %s", d.Id(), environment, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Environment %q: %s", d.Id(), environmentJson), map[string]interface{}{environmentLoggingKey: d.Id()})

	if _, err := setEnvironmentAttributes(d, environment); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})

	return nil
}

func environmentImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := environmentRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Environment %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Environment %q", d.Id()), map[string]interface{}{environmentLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func environmentImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllEnvironments,
	}
}

func loadAllEnvironments(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Environments: %s", createDescriptiveError(err)))
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	environmentsJson, err := json.Marshal(environments)
	if err != nil {
		return instances, diag.Errorf("error reading Environments: error marshaling %#v to json: %s", environments, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Environments: %s", environmentsJson))

	for _, environment := range environments {
		instanceId := environment.GetId()
		instances[instanceId] = toValidTerraformResourceName(environment.GetDisplayName())
	}

	return instances, nil
}

func getNestedStreamGovernancePackageKey() string {
	return fmt.Sprintf("%s.0.%s", paramStreamGovernance, paramPackage)
}

func streamGovernanceConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPackage: {
					Type:         schema.TypeString,
					Description:  "Stream Governance Package. 'ESSENTIALS' or 'ADVANCED'",
					ValidateFunc: validation.StringInSlice(acceptedBillingPackages, false),
					Required:     true,
				},
			},
		},
		Description: "Stream Governance configurations for the environment",
		Optional:    true,
		Computed:    true,
		MinItems:    1,
		MaxItems:    1,
	}
}
