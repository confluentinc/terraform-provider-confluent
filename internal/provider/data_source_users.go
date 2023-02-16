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
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const paramUsers = "users"

func usersDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: usersDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramUsers: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of users in Confluent",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the User (e.g., `u-abc123`).",
						},
						paramApiVersion: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "API Version defines the schema version of this representation of a User.",
						},
						paramKind: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Kind defines the object User represents.",
						},
						paramEmail: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The email address of the User.",
						},
						paramFullName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The full name of the User.",
						},
					},
				},
			},
		},
	}
}

func usersDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading all users")

	client := meta.(*Client)
	users, err := loadUsers(ctx, client)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	arr := make([]map[string]interface{}, len(users))
	for i, user := range users {
		arr[i] = map[string]interface{}{
			paramId:         user.Id,
			paramApiVersion: user.ApiVersion,
			paramKind:       user.Kind,
			paramEmail:      user.Email,
			paramFullName:   user.FullName,
		}
	}
	if err := d.Set(paramUsers, arr); err != nil {
		return diag.FromErr(err)
	}

	// force this data to be refreshed in every Terraform apply by setting a
	// unique id
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}
