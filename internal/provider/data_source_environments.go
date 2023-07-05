// Copyright 2023 Confluent Inc. All Rights Reserved.
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

func environmentsDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: environmentsDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramIds: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of environments",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func environmentsDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading Environments")

	client := meta.(*Client)
	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		return diag.Errorf("error reading Environments: %s", createDescriptiveError(err))
	}

	result := make([]string, len(environments))
	for i, environment := range environments {
		result[i] = environment.GetId()
	}

	if err := d.Set(paramIds, result); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}
