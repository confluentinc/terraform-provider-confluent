// Copyright 2025 Confluent Inc. All Rights Reserved.
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
)

func connectArtifactDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: connectArtifactDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Connect Artifact.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unique name of the Connect Artifact per cloud, environment scope.",
			},
			paramCloud: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cloud provider where the Connect Artifact archive is uploaded.",
			},
			paramContentFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Archive format of the Connect Artifact. Supported formats are JAR and ZIP.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the Connect Artifact.",
			},
		},
	}
}

func connectArtifactDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	artifactId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readConnectArtifactAndSetAttributes(ctx, d, meta, d.Get(paramCloud).(string), artifactId, "", environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading connect artifact %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}
