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
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	v3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
)

func clusterLinkDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: clusterLinkDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The composite ID of the Cluster Link data-source, in the format <Kafka cluster ID>/<Cluster Link name>.",
				Computed:    true,
			},
			paramKafkaCluster: optionalKafkaClusterBlockDataSourceSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramClusterLinkId: {
				Type:        schema.TypeString,
				Description: "The actual Cluster Link ID assigned from Confluent Cloud that uniquely represents a link between two Kafka clusters.",
				Computed:    true,
			},
			paramLinkName: {
				Type:        schema.TypeString,
				Description: "The name of the Cluster Link.",
				Required:    true,
			},
			paramLinkState: {
				Type:        schema.TypeString,
				Description: "The current state of the Cluster Link.",
				Computed:    true,
			},
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The custom cluster link settings retrieved (e.g., `\"acl.sync.ms\" = \"5100\"`).",
			},
		},
	}
}

func clusterLinkDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if err := dataSourceCredentialBlockValidationWithOAuth(d, meta.(*Client).isOAuthEnabled); err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, false, false, meta.(*Client).oauthToken)

	linkName := d.Get(paramLinkName).(string)
	err = readDataSourceClusterLinkAndSetAttributes(ctx, d, kafkaRestClient, linkName)
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}

	// Set the compositeClusterLinkId to match the behavior of the `confluent_cluster_link` resource
	compositeClusterLinkId := createClusterLinkCompositeId(clusterId, linkName)
	d.SetId(compositeClusterLinkId)
	tflog.Debug(ctx, fmt.Sprintf("Finished reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return nil
}

func readDataSourceClusterLinkAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, linkName string) error {
	clusterLink, _, err := c.apiClient.ClusterLinkingV3Api.GetKafkaLink(c.apiContext(ctx), c.clusterId, linkName).Execute()
	if err != nil {
		return fmt.Errorf("error reading Cluster Link %s: %s", linkName, createDescriptiveError(err))
	}

	clusterLinkJson, err := json.Marshal(clusterLink)
	if err != nil {
		return fmt.Errorf("error reading Cluster Link %q: error marshaling %#v to json: %s", d.Id(), clusterLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Cluster Link %q: %s", d.Id(), clusterLinkJson), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	if err := setDataSourceClusterLinkAttributes(ctx, d, c, clusterLink); err != nil {
		return createDescriptiveError(err)
	}
	return nil
}

func setDataSourceClusterLinkAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, clusterLink v3.ListLinksResponseData) error {
	if err := d.Set(paramClusterLinkId, clusterLink.GetClusterLinkId()); err != nil {
		return err
	}
	if err := d.Set(paramLinkState, clusterLink.GetLinkState()); err != nil {
		return err
	}
	configs, err := loadClusterLinkConfigs(ctx, d, c, clusterLink.GetLinkName())
	if err != nil {
		return err
	}
	if err := d.Set(paramConfigs, configs); err != nil {
		return err
	}
	return nil
}
