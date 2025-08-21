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
	fa "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
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
				Description:  "The ID of the Flink Artifact.",
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The unique name of the Flink Artifact per cloud, region, environment scope.",
			},
			paramClass: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Java class or alias for the Flink Artifact as provided by developer.",
				Deprecated:  fmt.Sprintf(deprecationMessageMajorRelease3, "class", "attribute"),
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Description:  "Cloud provider where the Flink Artifact archive is uploaded.",
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The Cloud provider region the Flink Artifact archive is uploaded.",
				ValidateFunc: validation.StringLenBetween(1, 60),
				Required:     true,
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Archive format of the Flink Artifact (JAR or ZIP).",
			},
			paramRuntimeLanguage: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Runtime language of the Flink Artifact as Python or JAVA. The default runtime language is JAVA.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the Flink Artifact.",
			},
			paramDocumentationLink: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Documentation link of the Flink Artifact.",
			},
			paramVersions: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of versions for this Flink Artifact.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramVersion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The version of this Flink Artifact.",
						},
					},
				},
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The schema version of this representation of a resource.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The object this REST resource represents.",
			},
		},
	}
}

func flinkArtifactDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	faId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	name := d.Get(paramDisplayName).(string)

	if faId != "" {
		return flinkArtifactDataSourceReadUsingId(ctx, d, meta, faId, environmentId)
	} else if name != "" {
		return flinkArtifactDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, name)
	} else {
		return diag.Errorf("error reading flink artifact: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func flinkArtifactDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, artifactId, envId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Artifact data source using Id %q", artifactId), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	c := meta.(*Client)
	fam, resp, err := executeArtifactRead(c.faApiContext(ctx), c, d.Get(paramRegion).(string), d.Get(paramCloud).(string), artifactId, envId)

	if err != nil {
		return diag.Errorf("error reading flink artifact data source using Id %q: %s", artifactId, createDescriptiveError(err, resp))
	}
	famJson, err := json.Marshal(fam)
	if err != nil {
		return diag.Errorf("error reading flink artifact %q: error marshaling %#v to json: %s", artifactId, fam, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Artifact %q: %s", artifactId, famJson), map[string]interface{}{flinkArtifactLoggingKey: artifactId})

	if _, err := setArtifactAttributes(d, fam, ""); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Artifact %q", artifactId), map[string]interface{}{flinkArtifactLoggingKey: artifactId})
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
			if orgHasMultipleFlinkArtifactsWithTargetDisplayName(flinkArtifacts, displayName) {
				return diag.Errorf("error reading flink artifacts: there are multiple flink artifacts with %q=%q", paramDisplayName, displayName)
			}
			if _, err := setArtifactAttributes(d, flinkArtifact, ""); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Artifact using display name %q: %s", displayName, famJson))
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
		artifactPageList, resp, err := executeListFlinkArtifacts(ctx, c, environmentId, cloud, region, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading flink artifacts list: %s", createDescriptiveError(err, resp))
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
					return nil, fmt.Errorf("error reading flink artifacts list: %s", createDescriptiveError(err, resp))
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

func orgHasMultipleFlinkArtifactsWithTargetDisplayName(flinkArtifacts []fa.ArtifactV1FlinkArtifact, displayName string) bool {
	var counter = 0
	for _, flinkArtifact := range flinkArtifacts {
		if flinkArtifact.GetDisplayName() == displayName {
			counter += 1
		}
	}
	return counter > 1
}
