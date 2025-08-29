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
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func roleBindingDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: roleBindingDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Role Binding (e.g., `rb-abc123`).",
			},
			paramPrincipal: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The principal User to bind the role to.",
			},
			paramRoleName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the role to bind to the principal.",
			},
			paramCrnPattern: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A CRN that specifies the scope and resource patterns necessary for the role to bind.",
			},
		},
	}
}

func roleBindingDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	roleBindingId := d.Get(paramId).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Role Binding %q", roleBindingId), map[string]interface{}{roleBindingLoggingKey: roleBindingId})
	c := meta.(*Client)
	roleBinding, resp, err := executeRoleBindingRead(c.mdsApiContext(ctx), c, roleBindingId)
	if err != nil {
		return diag.Errorf("error reading Role Binding %q: %s", roleBindingId, createDescriptiveError(err, resp))
	}
	roleBindingJson, err := json.Marshal(roleBinding)
	if err != nil {
		return diag.Errorf("error reading Role Binding %q: error marshaling %#v to json: %s", roleBindingId, roleBinding, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Role Binding %q: %s", roleBindingId, roleBindingJson), map[string]interface{}{roleBindingLoggingKey: roleBindingId})

	if _, err := setRoleBindingDataSourceAttributes(d, roleBinding); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Role Binding %q", roleBindingId), map[string]interface{}{roleBindingLoggingKey: roleBindingId})

	return nil
}

func setRoleBindingDataSourceAttributes(d *schema.ResourceData, roleBinding mds.IamV2RoleBinding) (*schema.ResourceData, error) {
	if err := d.Set(paramPrincipal, roleBinding.GetPrincipal()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRoleName, roleBinding.GetRoleName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCrnPattern, roleBinding.GetCrnPattern()); err != nil {
		return nil, createDescriptiveError(err)
	}

	d.SetId(roleBinding.GetId())
	return d, nil
}
