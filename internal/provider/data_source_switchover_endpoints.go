// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	switchoverv1 "github.com/confluentinc/ccloud-sdk-go-v2/switchover/v1"
)

// confluent_switchover_endpoints lists the switchover endpoints in an environment, optionally only
// those bound to one pair. Each element carries the same attributes as the
// confluent_switchover_endpoint data source.
func switchoverEndpointsDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: switchoverEndpointsDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironmentCrn: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The CRN of the environment whose switchover endpoints to list.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramSwitchoverPairId: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Only list endpoints bound to this switchover pair (e.g. `sw-abc123`).",
			},
			paramSwitchoverEndpoints: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The matching switchover endpoints.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the switchover endpoint (e.g. `se-abc123`).",
						},
						paramDisplayName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A human-readable name for the switchover endpoint.",
						},
						paramParentResourceCrn: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The CRN of the switchover pair this endpoint is bound to.",
						},
						paramTarget: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the endpoint that is currently active.",
						},
						paramEndpoints: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The endpoint definitions, one per side.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									paramName:           {Type: schema.TypeString, Computed: true},
									paramHostname:       {Type: schema.TypeString, Computed: true},
									paramCloud:          {Type: schema.TypeString, Computed: true},
									paramRegion:         {Type: schema.TypeString, Computed: true},
									paramConnectionType: {Type: schema.TypeString, Computed: true},
									paramEndpointFilter: {
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												paramType:           {Type: schema.TypeString, Computed: true},
												paramNetworkCrn:     {Type: schema.TypeString, Computed: true},
												paramAccessPointCrn: {Type: schema.TypeString, Computed: true},
											},
										},
									},
								},
							},
						},
						paramPhase: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The lifecycle phase of the switchover endpoint.",
						},
					},
				},
			},
		},
	}
}

func switchoverEndpointsDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentId := extractEnvironmentIdFromCrn(d.Get(paramEnvironmentCrn).(string))
	switchoverPairId := d.Get(paramSwitchoverPairId).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover endpoints in environment %q (pair filter %q)", environmentId, switchoverPairId))

	endpoints, err := loadSwitchoverEndpoints(ctx, c, environmentId, switchoverPairId)
	if err != nil {
		return diag.Errorf("error reading switchover endpoints: %s", createDescriptiveError(err))
	}

	result := make([]interface{}, len(endpoints))
	for i, endpoint := range endpoints {
		spec := endpoint.GetSpec()
		status := endpoint.GetStatus()
		result[i] = map[string]interface{}{
			paramId:                endpoint.GetId(),
			paramDisplayName:       spec.GetDisplayName(),
			paramParentResourceCrn: spec.GetParentResourceCrn(),
			paramTarget:            spec.GetTarget(),
			paramEndpoints:         flattenSwitchoverEndpoints(spec.GetEndpoints()),
			paramPhase:             status.GetPhase(),
		}
	}
	if err := d.Set(paramSwitchoverEndpoints, result); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	tflog.Debug(ctx, fmt.Sprintf("Finished reading %d switchover endpoint(s) in environment %q", len(endpoints), environmentId))
	return nil
}

// loadSwitchoverEndpoints follows the API's page tokens until every endpoint has been collected.
func loadSwitchoverEndpoints(ctx context.Context, c *Client, environmentId, switchoverPairId string) ([]switchoverv1.SwitchoverV1SwitchoverEndpoint, error) {
	endpoints := make([]switchoverv1.SwitchoverV1SwitchoverEndpoint, 0)
	pageToken := ""
	for {
		page, resp, err := executeListSwitchoverEndpoints(ctx, c, environmentId, switchoverPairId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading switchover endpoints: %s", createDescriptiveError(err, resp))
		}
		endpoints = append(endpoints, page.GetData()...)

		metadata := page.GetMetadata()
		pagination := metadata.GetPagination()
		pageToken = pagination.GetNextPageToken()
		if pageToken == "" {
			return endpoints, nil
		}
	}
}

func executeListSwitchoverEndpoints(ctx context.Context, c *Client, environmentId, switchoverPairId, pageToken string) (switchoverv1.SwitchoverV1SwitchoverEndpointList, *http.Response, error) {
	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.ListSwitchoverV1SwitchoverEndpoints(c.switchoverV1ApiContext(ctx)).Environment(environmentId).PageSize(listSwitchoverPageSize)
	if switchoverPairId != "" {
		req = req.SwitchoverPair(switchoverPairId)
	}
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	return req.Execute()
}
