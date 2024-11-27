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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2-internal/iam/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
	"strings"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using IAM V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listIamV2ServiceAccounts
	listServiceAccountsPageSize = 99
	pageTokenQueryParameter     = "page_token"
)

func serviceAccountDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: serviceAccountDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the Service Account (e.g., `sa-abc123`).",
			},
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
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "A human-readable name for the Service Account.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A free-form description of the Service Account.",
			},
		},
	}
}

func serviceAccountDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.

	serviceAccountId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	if serviceAccountId != "" {
		return serviceAccountDataSourceReadUsingId(ctx, d, meta, serviceAccountId)
	} else if displayName != "" {
		return serviceAccountDataSourceReadUsingDisplayName(ctx, d, meta, displayName)
	} else {
		return diag.Errorf("error reading Service Account: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func serviceAccountDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Service Account %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	serviceAccountList, _, err := c.iamClient.ServiceAccountsIamV2Api.ListIamV2ServiceAccounts(c.iamApiContext(ctx)).DisplayName(strings.Fields(displayName)).Execute()
	if err != nil {
		return diag.Errorf("error reading Service Account %q: %s", displayName, createDescriptiveError(err))
	}
	serviceAccounts := serviceAccountList.GetData()
	if len(serviceAccounts) == 0 {
		return diag.Errorf("error reading Service Account: Service Account with %q=%q was not found", paramDisplayName, displayName)
	}
	if orgHasMultipleSAsWithTargetDisplayName(serviceAccounts, displayName) {
		return diag.Errorf("error reading Service Account: there are multiple Service Accounts with %q=%q", paramDisplayName, displayName)
	}
	serviceAccount := serviceAccounts[0]
	serviceAccountJson, err := json.Marshal(serviceAccount)
	if err != nil {
		return diag.Errorf("error reading Service Account %q: error marshaling %#v to json: %s", displayName, serviceAccount, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Service Account %q: %s", serviceAccount.GetId(), serviceAccountJson), map[string]interface{}{serviceAccountLoggingKey: serviceAccount.GetId()})

	if _, err := setServiceAccountAttributes(d, serviceAccount); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func loadServiceAccounts(ctx context.Context, c *Client) ([]v2.IamV2ServiceAccount, error) {
	serviceAccounts := make([]v2.IamV2ServiceAccount, 0)

	allServiceAccountsAreCollected := false
	pageToken := ""
	for !allServiceAccountsAreCollected {
		serviceAccountPageList, _, err := executeListServiceAccounts(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Service Accounts: %s", createDescriptiveError(err))
		}
		serviceAccounts = append(serviceAccounts, serviceAccountPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := serviceAccountPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allServiceAccountsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Service Accounts: %s", createDescriptiveError(err))
				}
			}
		} else {
			allServiceAccountsAreCollected = true
		}
	}
	return serviceAccounts, nil
}

func executeListServiceAccounts(ctx context.Context, c *Client, pageToken string) (v2.IamV2ServiceAccountList, *http.Response, error) {
	if pageToken != "" {
		return c.iamClient.ServiceAccountsIamV2Api.ListIamV2ServiceAccounts(c.iamApiContext(ctx)).PageSize(listServiceAccountsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.iamClient.ServiceAccountsIamV2Api.ListIamV2ServiceAccounts(c.iamApiContext(ctx)).PageSize(listServiceAccountsPageSize).Execute()
	}
}

func serviceAccountDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, serviceAccountId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Service Account %q=%q", paramId, serviceAccountId), map[string]interface{}{serviceAccountLoggingKey: serviceAccountId})

	c := meta.(*Client)
	serviceAccount, _, err := executeServiceAccountRead(c.iamApiContext(ctx), c, serviceAccountId)
	if err != nil {
		return diag.Errorf("error reading Service Account %q: %s", serviceAccountId, createDescriptiveError(err))
	}
	serviceAccountJson, err := json.Marshal(serviceAccount)
	if err != nil {
		return diag.Errorf("error reading Service Account %q: error marshaling %#v to json: %s", serviceAccountId, serviceAccount, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Service Account %q: %s", serviceAccountId, serviceAccountJson), map[string]interface{}{serviceAccountLoggingKey: serviceAccountId})

	if _, err := setServiceAccountAttributes(d, serviceAccount); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func setServiceAccountAttributes(d *schema.ResourceData, serviceAccount v2.IamV2ServiceAccount) (*schema.ResourceData, error) {
	if err := d.Set(paramApiVersion, serviceAccount.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, serviceAccount.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDisplayName, serviceAccount.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, serviceAccount.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(serviceAccount.GetId())
	return d, nil
}

func orgHasMultipleSAsWithTargetDisplayName(serviceAccounts []v2.IamV2ServiceAccount, displayName string) bool {
	var numberOfServiceAccountsWithTargetDisplayName = 0
	for _, serviceAccount := range serviceAccounts {
		if serviceAccount.GetDisplayName() == displayName {
			numberOfServiceAccountsWithTargetDisplayName += 1
		}
	}
	return numberOfServiceAccountsWithTargetDisplayName > 1
}
