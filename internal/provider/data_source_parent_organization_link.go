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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func parentOrganizationLinkDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: parentOrganizationLinkDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The id of the Parent Organization Link",
				Required:    true,
			},
			paramParent:       environmentDataSourceSchema(),
			paramOrganization: environmentDataSourceSchema(),
		},
	}
}

func parentOrganizationLinkDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	parentOrganizationLinkId := d.Get(paramId).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Parent Organization Link %q=%q", paramId, parentOrganizationLinkId), map[string]interface{}{parentOrganizationLinkLoggingKey: parentOrganizationLinkId})

	c := meta.(*Client)
	parentOrganizationLink, _, err := executeParentOrganizationLinkRead(c.parentApiContext(ctx), c, parentOrganizationLinkId)
	if err != nil {
		return diag.Errorf("error reading Parent Organization Link %q: %s", parentOrganizationLinkId, createDescriptiveError(err))
	}
	parentOrganizationLinkJson, err := json.Marshal(parentOrganizationLink)
	if err != nil {
		return diag.Errorf("error reading Parent Organization Link %q: error marshaling %#v to json: %s", parentOrganizationLinkId, parentOrganizationLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Parent Organization Link %q: %s", parentOrganizationLinkId, parentOrganizationLinkJson), map[string]interface{}{parentOrganizationLinkLoggingKey: parentOrganizationLinkId})

	if _, err := setParentOrganizationLinkAttributes(d, parentOrganizationLink); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}
