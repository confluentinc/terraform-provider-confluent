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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func kafkaClientQuotaDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: kafkaClientQuotaDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Kafka Client Quota (e.g., `rb-abc123`).",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the Kafka Client Quota.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the Kafka Client Quota.",
			},
			paramKafkaCluster: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Computed: true,
			},
			paramEnvironment: {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramId: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Computed: true,
			},
			paramPrincipals: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "A list of service accounts. Special name \"default\" can be used to represent the default quota for all users and service accounts.",
			},
			paramThroughput: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Block for representing a Kafka Quota.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramIngressByteRate: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ingress throughput limit in bytes per second.",
						},
						paramEgressByteRate: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The egress throughput limit in bytes per second.",
						},
					},
				},
			},
		},
	}
}

func kafkaClientQuotaDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	kafkaClientQuotaId := d.Get(paramId).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Client Quota %q", kafkaClientQuotaId), map[string]interface{}{kafkaClientQuotaLoggingKey: kafkaClientQuotaId})
	c := meta.(*Client)
	kafkaClientQuota, _, err := executeKafkaClientQuotaRead(c.mdsApiContext(ctx), c, kafkaClientQuotaId)
	if err != nil {
		return diag.Errorf("error reading Kafka Client Quota %q: %s", kafkaClientQuotaId, createDescriptiveError(err))
	}
	kafkaClientQuotaJson, err := json.Marshal(kafkaClientQuota)
	if err != nil {
		return diag.Errorf("error reading Kafka Client Quota %q: error marshaling %#v to json: %s", kafkaClientQuotaId, kafkaClientQuota, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Client Quota %q: %s", kafkaClientQuotaId, kafkaClientQuotaJson), map[string]interface{}{kafkaClientQuotaLoggingKey: kafkaClientQuotaId})

	if _, err := setKafkaClientQuotaAttributes(d, kafkaClientQuota); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Client Quota %q", kafkaClientQuotaId), map[string]interface{}{kafkaClientQuotaLoggingKey: kafkaClientQuotaId})

	return nil
}
