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
	parent "github.com/confluentinc/ccloud-sdk-go-v2-internal/parent/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const paramInvitationRestrictionsEnabled = "invitation_restrictions_enabled"

func parentDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: parentDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The id of the Parent",
				Required:    true,
			},
			paramInvitationRestrictionsEnabled: {
				Type:        schema.TypeBool,
				Description: "Controls whether the Parent has domain-based invitation restrictions enabled.",
				Computed:    true,
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Description: "The Confluent Resource Name of the Parent.",
				Computed:    true,
			},
		},
	}
}

func parentDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	parentId := d.Get(paramId).(string)
	if parentId == "" {
		return diag.Errorf("error reading Parent: Parent id is missing")
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Parent %q=%q", paramId, parentId), map[string]interface{}{byokKeyLoggingKey: parentId})

	c := meta.(*Client)
	key, _, err := executeParentRead(c.byokApiContext(ctx), c, parentId)
	if err != nil {
		return diag.Errorf("error reading Parent %q: %s", parentId, createDescriptiveError(err))
	}
	keyJson, err := json.Marshal(key)
	if err != nil {
		return diag.Errorf("error reading Parent %q: error marshaling %#v to json: %s", parentId, key, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Parent %q: %s", parentId, keyJson), map[string]interface{}{byokKeyLoggingKey: parentId})

	if _, err := setParentAttributes(d, key); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func executeParentRead(ctx context.Context, c *Client, id string) (parent.IamV2Parent, *http.Response, error) {
	req := c.parentClient.ParentsIamV2Api.GetIamV2Parent(c.parentApiContext(ctx), id)
	return req.Execute()
}

func setParentAttributes(d *schema.ResourceData, parentOrganization parent.IamV2Parent) (*schema.ResourceData, error) {
	if err := d.Set(paramInvitationRestrictionsEnabled, parentOrganization.GetInvitationRestrictionsEnabled()); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, parentOrganization.Metadata.GetResourceName()); err != nil {
		return nil, err
	}
	d.SetId(parentOrganization.GetId())
	return d, nil
}
