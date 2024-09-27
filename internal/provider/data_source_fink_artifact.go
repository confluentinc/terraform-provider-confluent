package provider

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

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"

	fa "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
)

const (
	listFlinkArtifactsPageSize = 99
)

func flinkArtifactDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: flinkArtifactDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID for flink artifact.",
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The display name of flink artifact.",
			},
			paramClass: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The class for flink artifact",
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Description:  "The public cloud flink artifact name",
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
				Type:         schema.TypeString,
				Description:  "The cloud service provider region that hosts the Flink artifact.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The class for flink artifact",
			},
		},
	}
}

func flinkArtifactDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	faId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	name := d.Get(paramDisplayName).(string)

	if faId != "" {
		return flinkArtifactDataSourceReadUsingId(ctx, d, meta, faId)
	} else if name != "" {
		return flinkArtifactDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, name)
	} else {
		return diag.Errorf("error reading flink artifact: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func flinkArtifactDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, artifactId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading flink artifact data source using Id %q", artifactId), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	c := meta.(*Client)
	fam, _, err := executeArtifactRead(c.faApiContext(ctx), c, d.Get(paramRegion).(string), d.Get(paramCloud).(string), artifactId)

	if err != nil {
		return diag.Errorf("error reading flink artifact data source using Id %q: %s", artifactId, createDescriptiveError(err))
	}
	famJson, err := json.Marshal(fam)
	if err != nil {
		return diag.Errorf("error reading flink artifact %q: error marshaling %#v to json: %s", artifactId, fam, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched flink artifact %q: %s", artifactId, famJson), map[string]interface{}{flinkArtifactLoggingKey: artifactId})

	if _, err := setArtifactAttributes(d, fam); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading flink artifact %q", artifactId), map[string]interface{}{flinkArtifactLoggingKey: artifactId})
	return nil
}

func flinkArtifactDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading flink artifact data source using display name %q", displayName))
	c := meta.(*Client)
	flinkArtifacts, err := loadFlinkArtifacts(ctx, c, environmentId, d.Get(paramCloud).(string), d.Get(paramRegion).(string))

	if err != nil {
		return diag.Errorf("error reading flink artifact data source using display name %q: %s", displayName, createDescriptiveError(err))
	}

	for _, flinkArtifact := range flinkArtifacts {
		if flinkArtifact.GetDisplayName() == displayName {
			famJson, err := json.Marshal(flinkArtifact)
			if err != nil {
				return diag.Errorf("error reading flink artifact using display name %q: error marshaling %#v to json: %s", displayName, flinkArtifact, createDescriptiveError(err))
			}
			if orgHasMultipleProviderIntegrationsWithTargetDisplayName(flinkArtifacts, displayName) {
				return diag.Errorf("error reading flink artifacts: there are multiple flink artifacts with %q=%q", paramDisplayName, displayName)
			}
			if _, err := setArtifactAttributes(d, flinkArtifact); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched flink artifact using display name %q: %s", displayName, famJson))
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return nil
}

func loadFlinkArtifacts(ctx context.Context, c *Client, environmentId, cloud, region string) ([]fa.ArtifactV1FlinkArtifact, error) {
	flinkArtifacts := make([]fa.ArtifactV1FlinkArtifact, 0)
	done := false
	pageToken := ""
	for !done {
		artifactPageList, _, err := executeListFlinkArtifacts(ctx, c, environmentId, cloud, region, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading flink artifacts list: %s", createDescriptiveError(err))
		}
		flinkArtifacts = append(flinkArtifacts, artifactPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := artifactPageList.GetMetadata().Next
		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				done = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading flink artifacts list: %s", createDescriptiveError(err))
				}
			}
		} else {
			done = true
		}
	}

	return flinkArtifacts, nil
}

func executeListFlinkArtifacts(ctx context.Context, c *Client, environmentId, cloud, region, pageToken string) (fa.ArtifactV1FlinkArtifactList, *http.Response, error) {
	if pageToken != "" {
		return c.faClient.FlinkArtifactsArtifactV1Api.ListArtifactV1FlinkArtifacts(c.faApiContext(ctx)).Environment(environmentId).Cloud(cloud).Region(region).PageSize(listFlinkArtifactsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.faClient.FlinkArtifactsArtifactV1Api.ListArtifactV1FlinkArtifacts(c.faApiContext(ctx)).Environment(environmentId).Cloud(cloud).Region(region).PageSize(listFlinkArtifactsPageSize).Execute()
	}
}

func orgHasMultipleProviderIntegrationsWithTargetDisplayName(flinkArtifacts []fa.ArtifactV1FlinkArtifact, displayName string) bool {
	var counter = 0
	for _, flinkArtifact := range flinkArtifacts {
		if flinkArtifact.GetDisplayName() == displayName {
			counter += 1
		}
	}
	return counter > 1
}
