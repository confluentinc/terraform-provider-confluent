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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing schemaRegistryRegions using SG V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listOrgV2SchemaRegistryRegions
	listSchemaRegistryRegionsPageSize = 99
)

func schemaRegistryRegionDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: schemaRegistryRegionDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The ID of the Schema Registry Region (e.g., `sgreg-123`).",
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

func schemaRegistryRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	billingPackage := d.Get(paramPackage).(string)

	return executeSchemaRegistryRegionDataSourceRead(ctx, d, meta, cloud, region, billingPackage)
}

func executeSchemaRegistryRegionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}, cloud, region, billingPackage string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Region with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage))

	c := meta.(*Client)
	schemaRegistryRegions, _, err := executeListSchemaRegistryRegions(c.srcmApiContext(ctx), c, cloud, region, billingPackage)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Region: %s", createDescriptiveError(err))
	}
	schemaRegistryRegionJson, err := json.Marshal(schemaRegistryRegions)
	if err != nil {
		return diag.Errorf("error reading Schema Registry Region: error marshaling %#v to json: %s", schemaRegistryRegions, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Regions: %s", schemaRegistryRegionJson))

	if len(schemaRegistryRegions.GetData()) == 0 {
		return diag.Errorf("error reading Schema Registry Region: there aren't any Schema Registry Regions with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage)
	}
	if len(schemaRegistryRegions.GetData()) > 1 {
		return diag.Errorf("error reading Schema Registry Region: there are multiple Schema Registry Regions with %q=%q, %q=%q, %q=%q", paramCloud, cloud, paramRegion, region, paramPackage, billingPackage)
	}
	if _, err := setSchemaRegistryRegionAttributes(d, schemaRegistryRegions.GetData()[0], billingPackage); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func executeListSchemaRegistryRegions(ctx context.Context, c *Client, cloud, region, billingPackage string) (v2.SrcmV2RegionList, *http.Response, error) {
	return c.srcmClient.RegionsSrcmV2Api.ListSrcmV2Regions(c.srcmApiContext(ctx)).PageSize(listSchemaRegistryRegionsPageSize).SpecRegionName(region).SpecCloud(cloud).SpecPackages([]string{billingPackage}).Execute()
}

func setSchemaRegistryRegionAttributes(d *schema.ResourceData, schemaRegistryRegion v2.SrcmV2Region, billingPackage string) (*schema.ResourceData, error) {
	if err := d.Set(paramCloud, schemaRegistryRegion.Spec.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRegion, schemaRegistryRegion.Spec.GetRegionName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramPackage, billingPackage); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(schemaRegistryRegion.GetId())
	return d, nil
}
