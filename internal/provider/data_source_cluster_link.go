// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	v3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramDestinationClusterId = "destination_cluster_id"
	paramRemoteClusterId      = "remote_cluster_id"
	paramSourceClusterId      = "source_cluster_id"
)

func clusterLinkDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: clusterLinkDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The Terraform identifier of the Cluster Link data-source, in the format <Kafka cluster ID>/<Cluster Link name>.",
				Required:    true,
			},
			paramKafkaCluster: clusterLinkDataSourceKafkaClusterBlockSchema(),
			paramClusterLinkId: {
				Type:        schema.TypeString,
				Description: "The actual Cluster Link ID assigned by Confluent Cloud that uniquely represents a link between two Kafka clusters.",
				Computed:    true,
			},
			paramLinkMode: {
				Type:        schema.TypeString,
				Description: "The mode of the Cluster Link.",
				Computed:    true,
			},
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The custom cluster link settings to set (e.g., `\"acl.sync.ms\" = \"5100\"`).",
			},
		},
	}
}

func clusterLinkDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	parts := strings.SplitN(d.Get(paramId).(string), "/", 2)
	if len(parts) != 2 {
		// handle unexpected format, e.g., log an error or return
		fmt.Println("Unexpected format for link_name")
		return diag.Errorf("Unexpected format of link_name: %s", paramLinkName)
	}

	clusterId := parts[0]
	linkName := parts[1]
	apiKey := d.Get(paramKey).(string)
	apiSecret := d.Get(paramSecret).(string)
	endpoint := d.Get(paramRestEndpoint).(string)

	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(endpoint, clusterId, apiKey, apiSecret, false, false)

	_, err := readDataSourceClusterLinkAndSetAttributes(ctx, d, kafkaRestClient, linkName, "", "")
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}

	return nil
}

func readDataSourceClusterLinkAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, linkName, linkMode, connectionMode string) ([]*schema.ResourceData, error) {
	clusterLink, resp, err := c.apiClient.ClusterLinkingV3Api.GetKafkaLink(c.apiContext(ctx), c.clusterId, linkName).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Cluster Link %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Cluster Link %q in TF state because Cluster Link could not be found on the server", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	clusterLinkJson, err := json.Marshal(clusterLink)
	if err != nil {
		return nil, fmt.Errorf("error reading Cluster Link %q: error marshaling %#v to json: %s", d.Id(), clusterLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Cluster Link %q: %s", d.Id(), clusterLinkJson), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	if _, err := setDataSourceClusterLinkAttributes(ctx, d, c, clusterLink, linkMode, connectionMode); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setDataSourceClusterLinkAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, clusterLink v3.ListLinksResponseData,
	linkMode, connectionMode string) (*schema.ResourceData, error) {
	if err := d.Set(paramLinkName, clusterLink.GetLinkName()); err != nil {
		return nil, err
	}
	/*
		if err := d.Set(paramLinkMode, linkMode); err != nil {
			return nil, err
		}
		if err := d.Set(paramConnectionMode, connectionMode); err != nil {
			return nil, err
		}
	*/
	if err := d.Set(paramClusterLinkId, clusterLink.GetClusterLinkId()); err != nil {
		return nil, err
	}

	configs, err := loadClusterLinkConfigs(ctx, d, c, clusterLink.GetLinkName())
	if err != nil {
		return nil, err
	}
	if err := d.Set(paramConfigs, configs); err != nil {
		return nil, err
	}

	d.SetId(createClusterLinkId(c.clusterId, clusterLink.LinkName))
	return d, nil
}

func clusterLinkDataSourceKafkaClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The unique identifier for the referred Kafka cluster.",
				},
				paramRestEndpoint: {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
				},
				paramCredentials: {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "The Kafka API Credentials.",
					MinItems:    1,
					MaxItems:    1,
					Sensitive:   true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							paramKey: {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "The Kafka API Key for your Confluent Cloud cluster.",
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
							paramSecret: {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "The Kafka API Secret for your Confluent Cloud cluster.",
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
		},
	}
}
