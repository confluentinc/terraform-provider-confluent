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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing flinkRegions using SG V2 API
	// https://docs.confluent.io/cloud/current/api.html#tag/Regions-(fcpmv2)/operation/listFcpmV2Regions
	listFlinkRegionsPageSize = 99
)

func flinkRegionDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: flinkRegionDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The ID of the Flink Region (e.g., `aws.us-east-1`).",
				Computed:    true,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				// Suppress the diff shown if the value of "cloud" attribute are equal when both compared in lower case.
				// For example, AWS == aws
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.ToLower(old) == strings.ToLower(new) {
						return true
					}
					return false
				},
			},
			paramRegion: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramRestEndpoint: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func flinkRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)

	return executeFlinkRegionDataSourceRead(ctx, d, meta, cloud, region)
}

func executeFlinkRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}, cloud, region string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Region with %q=%q, %q=%q", paramCloud, cloud, paramRegion, region))

	c := meta.(*Client)
	flinkRegions, _, err := executeListFlinkRegions(c.fcpmApiContext(ctx), c, cloud, region)
	if err != nil {
		return diag.Errorf("error reading Flink Region: %s", createDescriptiveError(err))
	}
	flinkRegionJson, err := json.Marshal(flinkRegions)
	if err != nil {
		return diag.Errorf("error reading Flink Region: error marshaling %#v to json: %s", flinkRegions, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Regions: %s", flinkRegionJson))

	if len(flinkRegions.GetData()) == 0 {
		return diag.Errorf("error reading Flink Region: there aren't any Flink Regions with %q=%q, %q=%q", paramCloud, cloud, paramRegion, region)
	}
	if len(flinkRegions.GetData()) > 1 {
		return diag.Errorf("error reading Flink Region: there are multiple Flink Regions with %q=%q, %q=%q", paramCloud, cloud, paramRegion, region)
	}
	if _, err := setFlinkRegionAttributes(d, flinkRegions.GetData()[0]); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func executeListFlinkRegions(ctx context.Context, c *Client, cloud, region string) (v2.FcpmV2RegionList, *http.Response, error) {
	return c.fcpmClient.RegionsFcpmV2Api.ListFcpmV2Regions(c.fcpmApiContext(ctx)).PageSize(listFlinkRegionsPageSize).RegionName(region).Cloud(cloud).Execute()
}

func setFlinkRegionAttributes(d *schema.ResourceData, flinkRegion v2.FcpmV2Region) (*schema.ResourceData, error) {
	if err := d.Set(paramCloud, flinkRegion.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRegion, flinkRegion.GetRegionName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, flinkRegion.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, flinkRegion.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRestEndpoint, flinkRegion.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(flinkRegion.GetId())
	return d, nil
}
