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
	iam "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

func serviceAccountResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: serviceAccountCreate,
		ReadContext:   serviceAccountRead,
		UpdateContext: serviceAccountUpdate,
		DeleteContext: serviceAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: serviceAccountImport,
		},
		Schema: map[string]*schema.Schema{
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Service Account.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Service Account represents.",
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "A human-readable name for the Service Account.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form description of the Service Account.",
				ValidateFunc: validation.All(
					validation.StringIsNotEmpty,
					validation.StringLenBetween(0, 128)
				),
			},
		},
	}
}

func serviceAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDescription) {
		return diag.Errorf("error updating Service Account %q: only %q attribute can be updated for Service Account", d.Id(), paramDescription)
	}

	updateServiceAccountRequest := iam.NewIamV2ServiceAccountUpdate()
	updatedDescription := d.Get(paramDescription).(string)
	updateServiceAccountRequest.SetDescription(updatedDescription)
	updateServiceAccountRequestJson, err := json.Marshal(updateServiceAccountRequest)
	if err != nil {
		return diag.Errorf("error updating Service Account %q: error marshaling %#v to json: %s", d.Id(), updateServiceAccountRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Service Account %q: %s", d.Id(), updateServiceAccountRequestJson), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedServiceAccount, _, err := c.iamClient.ServiceAccountsIamV2Api.UpdateIamV2ServiceAccount(c.iamApiContext(ctx), d.Id()).IamV2ServiceAccountUpdate(*updateServiceAccountRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Service Account %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedServiceAccountJson, err := json.Marshal(updatedServiceAccount)
	if err != nil {
		return diag.Errorf("error updating Service Account %q: error marshaling %#v to json: %s", d.Id(), updatedServiceAccount, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Service Account %q: %s", d.Id(), updatedServiceAccountJson), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	return serviceAccountRead(ctx, d, meta)
}

func serviceAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)

	createServiceAccountRequest := iam.NewIamV2ServiceAccount()
	createServiceAccountRequest.SetDisplayName(displayName)
	createServiceAccountRequest.SetDescription(description)
	createServiceAccountRequestJson, err := json.Marshal(createServiceAccountRequest)
	if err != nil {
		return diag.Errorf("error creating Service Account: error marshaling %#v to json: %s", createServiceAccountRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Service Account: %s", createServiceAccountRequestJson))

	createdServiceAccount, _, err := executeServiceAccountCreate(c.iamApiContext(ctx), c, createServiceAccountRequest)
	if err != nil {
		return diag.Errorf("error creating Service Account %q: %s", displayName, createDescriptiveError(err))
	}
	d.SetId(createdServiceAccount.GetId())

	createdServiceAccountJson, err := json.Marshal(createdServiceAccount)
	if err != nil {
		return diag.Errorf("error creating Service Account %q: error marshaling %#v to json: %s", d.Id(), createdServiceAccount, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Service Account %q: %s", d.Id(), createdServiceAccountJson), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	return serviceAccountRead(ctx, d, meta)
}

func executeServiceAccountCreate(ctx context.Context, c *Client, serviceAccount *iam.IamV2ServiceAccount) (iam.IamV2ServiceAccount, *http.Response, error) {
	req := c.iamClient.ServiceAccountsIamV2Api.CreateIamV2ServiceAccount(c.iamApiContext(ctx)).IamV2ServiceAccount(*serviceAccount)
	return req.Execute()
}

func serviceAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.iamClient.ServiceAccountsIamV2Api.DeleteIamV2ServiceAccount(c.iamApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Service Account %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	return nil
}

func executeServiceAccountRead(ctx context.Context, c *Client, serviceAccountId string) (iam.IamV2ServiceAccount, *http.Response, error) {
	req := c.iamClient.ServiceAccountsIamV2Api.GetIamV2ServiceAccount(c.iamApiContext(ctx), serviceAccountId)
	return req.Execute()
}

func serviceAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})
	c := meta.(*Client)
	serviceAccount, resp, err := executeServiceAccountRead(c.iamApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Service Account %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Service Account %q in TF state because Service Account could not be found on the server", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	serviceAccountJson, err := json.Marshal(serviceAccount)
	if err != nil {
		return diag.Errorf("error reading Service Account %q: error marshaling %#v to json: %s", d.Id(), serviceAccount, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Service Account %q: %s", d.Id(), serviceAccountJson), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	if _, err := setServiceAccountAttributes(d, serviceAccount); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})

	return nil
}

func serviceAccountImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := serviceAccountRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Service Account %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Service Account %q", d.Id()), map[string]interface{}{serviceAccountLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func serviceAccountImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllServiceAccounts,
	}
}

func loadAllServiceAccounts(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	serviceAccounts, err := loadServiceAccounts(ctx, client)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Service Accounts: %s", createDescriptiveError(err)))
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	serviceAccountsJson, err := json.Marshal(serviceAccounts)
	if err != nil {
		return instances, diag.Errorf("error reading Service Accounts: error marshaling %#v to json: %s", serviceAccounts, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Service Accounts: %s", serviceAccountsJson))

	for _, serviceAccount := range serviceAccounts {
		instanceId := serviceAccount.GetId()
		instances[instanceId] = toValidTerraformResourceName(serviceAccount.GetDisplayName())
	}

	return instances, nil
}
