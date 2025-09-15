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
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

const crnEnvironmentSuffix = "/environment="
const crnOrgSuffix = "/organization="

func organizationDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: organizationDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The ID of the Organization (e.g., `1111aaaa-11aa-11aa-11aa-111111aaaaaa`).",
				Computed:    true,
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Description: "The Confluent Resource Name of the Organization.",
				Computed:    true,
			},
		},
	}
}

func organizationDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading Organization")

	c := meta.(*Client)
	environments, resp, err := c.orgClient.EnvironmentsOrgV2Api.ListOrgV2Environments(c.orgApiContext(ctx)).Execute()
	if err != nil {
		return diag.Errorf("error reading Environments: %s", createDescriptiveError(err, resp))
	}

	// At least one environment is required in every organization
	// https://docs.confluent.io/cloud/current/access-management/hierarchy/cloud-environments.html#delete-an-environment
	if len(environments.GetData()) == 0 {
		return diag.Errorf("error reading Environments: no environments were found")
	}
	environment := environments.GetData()[0]
	environmentResourceName := environment.Metadata.GetResourceName()
	organizationResourceName, err := extractOrgResourceName(environmentResourceName)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}
	organizationId, err := extractOrgIdFromResourceName(organizationResourceName)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}
	if err := d.Set(paramResourceName, organizationResourceName); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	d.SetId(organizationId)

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Organization %q", organizationId))

	return nil
}

// Extracts
// crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa
// from
// crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-abc123
func extractOrgResourceName(environmentResourceName string) (string, error) {
	lastIndex := strings.LastIndex(environmentResourceName, crnEnvironmentSuffix)
	if lastIndex == -1 {
		return "", fmt.Errorf("could not find %s in %s", crnEnvironmentSuffix, environmentResourceName)
	}
	return environmentResourceName[:lastIndex], nil
}
