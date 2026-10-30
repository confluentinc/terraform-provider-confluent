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

// The Switchover API caps page_size at 100.
const listSwitchoverPageSize = 99

// confluent_switchover_pairs lists every switchover pair in an environment. It exists for discovery
// (audits, or an on-call engineer locating the pair to fail over without access to the workspace
// that created it); each element carries the same attributes as the confluent_switchover_pair data
// source.
func switchoverPairsDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: switchoverPairsDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironmentCrn: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The CRN of the environment whose switchover pairs to list.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramSwitchoverPairs: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The switchover pairs in the environment.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the switchover pair (e.g. `sw-abc123`).",
						},
						paramDisplayName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A human-readable name for the switchover pair.",
						},
						paramEnvironmentCrn: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The CRN of the environment that owns this switchover pair.",
						},
						paramMembers: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The two clusters participating in this switchover pair.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									paramName:      {Type: schema.TypeString, Computed: true},
									paramMemberCrn: {Type: schema.TypeString, Computed: true},
									paramCloud:     {Type: schema.TypeString, Computed: true},
									paramRegion:    {Type: schema.TypeString, Computed: true},
								},
							},
						},
						paramActiveMember: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the member that is currently active.",
						},
						paramFirstActive: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the member that was active when the pair was first created.",
						},
						paramFailoverType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The failover semantics most recently applied to this pair.",
						},
						paramPhase: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The lifecycle phase of the switchover pair.",
						},
					},
				},
			},
		},
	}
}

func switchoverPairsDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentCrn := d.Get(paramEnvironmentCrn).(string)
	environmentId := extractEnvironmentIdFromCrn(environmentCrn)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover pairs in environment %q", environmentId))

	pairs, err := loadSwitchoverPairs(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading switchover pairs: %s", createDescriptiveError(err))
	}

	result := make([]interface{}, len(pairs))
	for i, pair := range pairs {
		spec := pair.GetSpec()
		status := pair.GetStatus()
		pairEnvironmentCrn := spec.GetEnvironmentCrn()
		if pairEnvironmentCrn == "" {
			pairEnvironmentCrn = environmentCrn
		}
		result[i] = map[string]interface{}{
			paramId:             pair.GetId(),
			paramDisplayName:    spec.GetDisplayName(),
			paramEnvironmentCrn: pairEnvironmentCrn,
			paramMembers:        flattenSwitchoverPairMembers(spec.GetMembers()),
			paramActiveMember:   spec.GetActiveMember(),
			paramFirstActive:    spec.GetFirstActive(),
			paramFailoverType:   spec.GetFailoverType(),
			paramPhase:          status.GetPhase(),
		}
	}
	if err := d.Set(paramSwitchoverPairs, result); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	tflog.Debug(ctx, fmt.Sprintf("Finished reading %d switchover pair(s) in environment %q", len(pairs), environmentId))
	return nil
}

// loadSwitchoverPairs follows the API's page tokens until every pair has been collected.
func loadSwitchoverPairs(ctx context.Context, c *Client, environmentId string) ([]switchoverv1.SwitchoverV1SwitchoverPair, error) {
	pairs := make([]switchoverv1.SwitchoverV1SwitchoverPair, 0)
	pageToken := ""
	for {
		page, resp, err := executeListSwitchoverPairs(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading switchover pairs: %s", createDescriptiveError(err, resp))
		}
		pairs = append(pairs, page.GetData()...)

		metadata := page.GetMetadata()
		pagination := metadata.GetPagination()
		pageToken = pagination.GetNextPageToken()
		if pageToken == "" {
			return pairs, nil
		}
	}
}

func executeListSwitchoverPairs(ctx context.Context, c *Client, environmentId, pageToken string) (switchoverv1.SwitchoverV1SwitchoverPairList, *http.Response, error) {
	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.ListSwitchoverV1SwitchoverPairs(c.switchoverV1ApiContext(ctx)).Environment(environmentId).PageSize(listSwitchoverPageSize)
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}
	return req.Execute()
}
