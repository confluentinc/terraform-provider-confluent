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
	"fmt"
	apikeys "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	cmk "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	connect "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	iamv1 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v1"
	iam "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	org "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

const (
	terraformProviderUserAgent = "terraform-provider-confluent"
)

const (
	paramApiVersion  = "api_version"
	paramCloud       = "cloud"
	paramRegion      = "region"
	paramEnvironment = "environment"
	paramId          = "id"
	paramDisplayName = "display_name"
	paramDescription = "description"
	paramKind        = "kind"
)

type Client struct {
	apiKeysClient          *apikeys.APIClient
	iamClient              *iam.APIClient
	iamV1Client            *iamv1.APIClient
	cmkClient              *cmk.APIClient
	connectClient          *connect.APIClient
	netClient              *net.APIClient
	orgClient              *org.APIClient
	kafkaRestClientFactory *KafkaRestClientFactory
	mdsClient              *mds.APIClient
	userAgent              string
	apiKey                 string
	apiSecret              string
}

// Customize configs for terraform-plugin-docs
func init() {
	schema.DescriptionKind = schema.StringMarkdown

	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		descriptionWithDefault := s.Description
		if s.Default != nil {
			descriptionWithDefault += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(descriptionWithDefault)
	}
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		provider := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"cloud_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_CLOUD_API_KEY", ""),
					Description: "The Confluent Cloud API Key.",
				},
				"cloud_api_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_CLOUD_API_SECRET", ""),
					Description: "The Confluent Cloud API Secret.",
				},
				"endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "https://api.confluent.cloud",
					Description: "The base endpoint of Confluent Cloud API.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"confluent_kafka_cluster_v2":       kafkaDataSource(),
				"confluent_kafka_topic_v3":         kafkaTopicDataSource(),
				"confluent_environment_v2":         environmentDataSource(),
				"confluent_network_v1":             networkDataSource(),
				"confluent_organization_v2":        organizationDataSource(),
				"confluent_peering_v1":             peeringDataSource(),
				"confluent_private_link_access_v1": privateLinkAccessDataSource(),
				"confluent_role_binding_v2":        roleBindingDataSource(),
				"confluent_service_account_v2":     serviceAccountDataSource(),
				"confluent_user_v2":                userDataSource(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"confluent_api_key_v2":             apiKeyResource(),
				"confluent_kafka_cluster_v2":       kafkaResource(),
				"confluent_environment_v2":         environmentResource(),
				"confluent_connector_v1":           connectorResource(),
				"confluent_service_account_v2":     serviceAccountResource(),
				"confluent_kafka_topic_v3":         kafkaTopicResource(),
				"confluent_kafka_acl_v3":           kafkaAclResource(),
				"confluent_network_v1":             networkResource(),
				"confluent_peering_v1":             peeringResource(),
				"confluent_private_link_access_v1": privateLinkAccessResource(),
				"confluent_role_binding_v2":        roleBindingResource(),
			},
		}

		provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			return providerConfigure(ctx, d, provider, version)
		}

		return provider
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func environmentSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the environment.",
				},
			},
		},
		Required:    true,
		MinItems:    1,
		MaxItems:    1,
		ForceNew:    true,
		Description: "Environment objects represent an isolated namespace for your Confluent resources for organizational purposes.",
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func environmentDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		Required: true,
		MaxItems: 1,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, p *schema.Provider, providerVersion string) (interface{}, diag.Diagnostics) {
	tflog.Info(ctx, "Initializing Terraform Provider for Confluent Cloud")
	endpoint := d.Get("endpoint").(string)
	apiKey := d.Get("cloud_api_key").(string)
	apiSecret := d.Get("cloud_api_secret").(string)

	userAgent := p.UserAgent(terraformProviderUserAgent, fmt.Sprintf("%s (https://confluent.cloud; support@confluent.io)", providerVersion))

	apiKeysCfg := apikeys.NewConfiguration()
	cmkCfg := cmk.NewConfiguration()
	connectCfg := connect.NewConfiguration()
	iamCfg := iam.NewConfiguration()
	iamV1Cfg := iamv1.NewConfiguration()
	mdsCfg := mds.NewConfiguration()
	netCfg := net.NewConfiguration()
	orgCfg := org.NewConfiguration()

	apiKeysCfg.Servers[0].URL = endpoint
	cmkCfg.Servers[0].URL = endpoint
	connectCfg.Servers[0].URL = endpoint
	iamCfg.Servers[0].URL = endpoint
	iamV1Cfg.Servers[0].URL = endpoint
	mdsCfg.Servers[0].URL = endpoint
	netCfg.Servers[0].URL = endpoint
	orgCfg.Servers[0].URL = endpoint

	apiKeysCfg.UserAgent = userAgent
	cmkCfg.UserAgent = userAgent
	connectCfg.UserAgent = userAgent
	iamCfg.UserAgent = userAgent
	iamV1Cfg.UserAgent = userAgent
	mdsCfg.UserAgent = userAgent
	netCfg.UserAgent = userAgent
	orgCfg.UserAgent = userAgent

	apiKeysCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	cmkCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	connectCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	iamCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	iamV1Cfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	mdsCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	netCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	orgCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()

	client := Client{
		apiKeysClient:          apikeys.NewAPIClient(apiKeysCfg),
		cmkClient:              cmk.NewAPIClient(cmkCfg),
		connectClient:          connect.NewAPIClient(connectCfg),
		iamClient:              iam.NewAPIClient(iamCfg),
		iamV1Client:            iamv1.NewAPIClient(iamV1Cfg),
		netClient:              net.NewAPIClient(netCfg),
		orgClient:              org.NewAPIClient(orgCfg),
		kafkaRestClientFactory: &KafkaRestClientFactory{userAgent: userAgent},
		mdsClient:              mds.NewAPIClient(mdsCfg),
		userAgent:              userAgent,
		apiKey:                 apiKey,
		apiSecret:              apiSecret,
	}

	return &client, nil
}
