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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/stream-governance/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing streamGovernanceRegions using SG V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listOrgV2StreamGovernanceRegions
	listStreamGovernanceRegionsPageSize = 99
)

func streamGovernanceRegionDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: streamGovernanceRegionDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The ID of the Stream Governance Region (e.g., `sgreg-123`).",
				Computed:    true,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
			},
			paramRegion: {
				Type:     schema.TypeString,
				Required: true,
			},
			paramPackage: {
				Type:         schema.TypeString,
				Description:  "The billing package.",
				ValidateFunc: validation.StringInSlice(acceptedBillingPackages, false),
				Required:     true,
			},
		},
	}
}

func streamGovernanceRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	billingPackage := d.Get(paramPackage).(string)

	return executeStreamGovernanceRegionDataSourceRead(ctx, d, meta, cloud, region, billingPackage)
}

func executeStreamGovernanceRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}, cloud, region, billingPackage string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Stream Governance Region with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage))

	c := meta.(*Client)
	streamGovernanceRegions, _, err := executeListStreamGovernanceRegions(c.sgApiContext(ctx), c, cloud, region, billingPackage)
	if err != nil {
		return diag.Errorf("error reading Stream Governance Region: %s", createDescriptiveError(err))
	}
	streamGovernanceRegionJson, err := json.Marshal(streamGovernanceRegions)
	if err != nil {
		return diag.Errorf("error reading Stream Governance Region: error marshaling %#v to json: %s", streamGovernanceRegions, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Stream Governance Regions: %s", streamGovernanceRegionJson))

	if len(streamGovernanceRegions.GetData()) == 0 {
		return diag.Errorf("error reading Stream Governance Region: there aren't any Stream Governance Regions with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage)
	}
	if len(streamGovernanceRegions.GetData()) > 1 {
		return diag.Errorf("error reading Stream Governance Region: there are multiple Stream Governance Regions with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage)
	}
	if _, err := setStreamGovernanceRegionAttributes(d, streamGovernanceRegions.GetData()[0], billingPackage); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func executeListStreamGovernanceRegions(ctx context.Context, c *Client, cloud, region, billingPackage string) (v2.StreamGovernanceV2RegionList, *http.Response, error) {
	return c.sgClient.RegionsStreamGovernanceV2Api.ListStreamGovernanceV2Regions(c.sgApiContext(ctx)).PageSize(listStreamGovernanceRegionsPageSize).SpecRegionName(region).SpecCloud(cloud).SpecPackages([]string{billingPackage}).Execute()
}

func setStreamGovernanceRegionAttributes(d *schema.ResourceData, streamGovernanceRegion v2.StreamGovernanceV2Region, billingPackage string) (*schema.ResourceData, error) {
	if err := d.Set(paramCloud, streamGovernanceRegion.Spec.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRegion, streamGovernanceRegion.Spec.GetRegionName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramPackage, billingPackage); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(streamGovernanceRegion.GetId())
	return d, nil
}
