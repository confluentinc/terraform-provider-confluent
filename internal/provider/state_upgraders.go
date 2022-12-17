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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

func kafkaResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Kafka cluster.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Kafka cluster.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Kafka cluster represents.",
			},
			paramAvailability: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The availability zone configuration of the Kafka cluster.",
				ValidateFunc: validation.StringInSlice(acceptedAvailabilityZones, false),
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The cloud service provider that runs the Kafka cluster.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
			},
			paramRegion: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cloud service provider region where the Kafka cluster is running.",
			},
			paramNetwork:          optionalNetworkSchema(),
			paramBasicCluster:     basicClusterSchema(),
			paramStandardCluster:  standardClusterSchema(),
			paramDedicatedCluster: dedicatedClusterSchema(),
			paramBootStrapEndpoint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bootstrap endpoint used by Kafka clients to connect to the Kafka cluster.",
			},
			paramHttpEndpoint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The REST endpoint of the Kafka cluster.",
			},
			paramRbacCrn: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The Confluent Resource Name of the Kafka cluster suitable for " +
					"confluent_role_binding's crn_pattern.",
			},
			paramEnvironment: environmentSchema(),
		},
	}
}

// Modifies the attribute(s) appropriately for the migration.
func kafkaStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	// 1. When upgrading from 0.10.0, rename "http_endpoint" to "rest_endpoint" by copying the value and deleting "http_endpoint" attribute
	if httpEndpoint, found := rawState[paramHttpEndpoint]; found {
		httpEndpointString := httpEndpoint.(string)
		rawState[paramRestEndpoint] = httpEndpointString
		delete(rawState, paramHttpEndpoint)
	}
	// 2. When upgrading from 0.11.0 no changes are necessary: "rest_endpoint" exists already

	return rawState, nil
}

func kafkaAclResourceV1() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: kafkaClusterBlockSchema(),
			paramResourceType: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The type of the resource.",
				ValidateFunc: validation.StringInSlice(acceptedResourceTypes, false),
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The resource name for the ACL.",
			},
			paramPatternType: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The pattern type for the ACL.",
				ValidateFunc: validation.StringInSlice(acceptedPatternTypes, false),
			},
			paramPrincipal: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The principal for the ACL.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^User:(\\*)$|^User:(sa|u|pool)-"), "the principal must start with 'User:sa-' or 'User:u-' or 'User:pool-' or 'User:*'."),
			},
			paramHost: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The host for the ACL.",
			},
			paramOperation: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The operation type for the ACL.",
				ValidateFunc: validation.StringInSlice(acceptedOperations, false),
			},
			paramPermission: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The permission for the ACL.",
				ValidateFunc: validation.StringInSlice(acceptedPermissions, false),
			},
			paramHttpEndpoint: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
	}
}

func kafkaTopicResourceV1() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: kafkaClusterBlockSchema(),
			paramTopicName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the topic, for example, `orders-1`.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9\\._\-]+$`), "The topic name can be up to 249 characters in length, and can include the following characters: a-z, A-Z, 0-9, . (dot), _ (underscore), and - (dash)."),
			},
			paramPartitionsCount: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      6,
				ForceNew:     true,
				Description:  "The number of partitions to create in the topic.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			paramHttpEndpoint: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Computed:    true,
				Description: "The custom topic settings to set (e.g., `\"cleanup.policy\" = \"compact\"`).",
			},
			paramCredentials: credentialsSchema(),
		},
	}
}
