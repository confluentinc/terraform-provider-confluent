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
	oidc "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
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
	oidcClient             *oidc.APIClient
	userAgent              string
	cloudApiKey            string
	cloudApiSecret         string
	kafkaApiKey            string
	kafkaApiSecret         string
	kafkaRestEndpoint      string
	isKafkaMetadataSet     bool
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
				"kafka_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("KAFKA_API_KEY", ""),
					Description: "The Kafka Cluster API Key.",
				},
				"kafka_api_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("KAFKA_API_SECRET", ""),
					Description: "The Kafka Cluster API Secret.",
				},
				"kafka_rest_endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("KAFKA_REST_ENDPOINT", ""),
					Description: "The Kafka Cluster REST Endpoint.",
				},
				"endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "https://api.confluent.cloud",
					Description: "The base endpoint of Confluent Cloud API.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"confluent_kafka_cluster":       kafkaDataSource(),
				"confluent_kafka_topic":         kafkaTopicDataSource(),
				"confluent_environment":         environmentDataSource(),
				"confluent_identity_pool":       identityPoolDataSource(),
				"confluent_identity_provider":   identityProviderDataSource(),
				"confluent_network":             networkDataSource(),
				"confluent_organization":        organizationDataSource(),
				"confluent_peering":             peeringDataSource(),
				"confluent_private_link_access": privateLinkAccessDataSource(),
				"confluent_role_binding":        roleBindingDataSource(),
				"confluent_service_account":     serviceAccountDataSource(),
				"confluent_user":                userDataSource(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"confluent_api_key":             apiKeyResource(),
				"confluent_kafka_cluster":       kafkaResource(),
				"confluent_environment":         environmentResource(),
				"confluent_identity_pool":       identityPoolResource(),
				"confluent_identity_provider":   identityProviderResource(),
				"confluent_connector":           connectorResource(),
				"confluent_service_account":     serviceAccountResource(),
				"confluent_kafka_topic":         kafkaTopicResource(),
				"confluent_kafka_acl":           kafkaAclResource(),
				"confluent_network":             networkResource(),
				"confluent_peering":             peeringResource(),
				"confluent_private_link_access": privateLinkAccessResource(),
				"confluent_role_binding":        roleBindingResource(),
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
	cloudApiKey := d.Get("cloud_api_key").(string)
	cloudApiSecret := d.Get("cloud_api_secret").(string)
	kafkaApiKey := d.Get("kafka_api_key").(string)
	kafkaApiSecret := d.Get("kafka_api_secret").(string)
	kafkaRestEndpoint := d.Get("kafka_rest_endpoint").(string)

	// All 3 attributes should be set or not set at the same time
	allKafkaAttributesAreSet := (kafkaApiKey != "") && (kafkaApiSecret != "") && (kafkaRestEndpoint != "")
	allKafkaAttributesAreNotSet := (kafkaApiKey == "") && (kafkaApiSecret == "") && (kafkaRestEndpoint == "")
	justOneOrTwoKafkaAttributesAreSet := !(allKafkaAttributesAreSet || allKafkaAttributesAreNotSet)
	if justOneOrTwoKafkaAttributesAreSet {
		return nil, diag.Errorf("All 3 kafka_api_key, kafka_api_secret, kafka_rest_endpoint attributes should be set or not set in the provider block at the same time")
	}

	userAgent := p.UserAgent(terraformProviderUserAgent, fmt.Sprintf("%s (https://confluent.cloud; support@confluent.io)", providerVersion))

	apiKeysCfg := apikeys.NewConfiguration()
	cmkCfg := cmk.NewConfiguration()
	connectCfg := connect.NewConfiguration()
	iamCfg := iam.NewConfiguration()
	iamV1Cfg := iamv1.NewConfiguration()
	mdsCfg := mds.NewConfiguration()
	netCfg := net.NewConfiguration()
	oidcCfg := oidc.NewConfiguration()
	orgCfg := org.NewConfiguration()

	apiKeysCfg.Servers[0].URL = endpoint
	cmkCfg.Servers[0].URL = endpoint
	connectCfg.Servers[0].URL = endpoint
	iamCfg.Servers[0].URL = endpoint
	iamV1Cfg.Servers[0].URL = endpoint
	mdsCfg.Servers[0].URL = endpoint
	netCfg.Servers[0].URL = endpoint
	oidcCfg.Servers[0].URL = endpoint
	orgCfg.Servers[0].URL = endpoint

	apiKeysCfg.UserAgent = userAgent
	cmkCfg.UserAgent = userAgent
	connectCfg.UserAgent = userAgent
	iamCfg.UserAgent = userAgent
	iamV1Cfg.UserAgent = userAgent
	mdsCfg.UserAgent = userAgent
	netCfg.UserAgent = userAgent
	oidcCfg.UserAgent = userAgent
	orgCfg.UserAgent = userAgent

	apiKeysCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	cmkCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	// TODO: Uncomment once APIF-2660 is completed
	// connectCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	iamCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	iamV1Cfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	mdsCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	netCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	oidcCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	orgCfg.HTTPClient = createRetryableHttpClientWithExponentialBackoff()

	// TODO: Delete once APIF-2660 is completed
	tempConnectClient := createRetryableHttpClientWithExponentialBackoff()
	tempConnectClient.Transport = &ItsActuallyJsonRoundTripper{tempConnectClient.Transport}
	connectCfg.HTTPClient = tempConnectClient

	client := Client{
		apiKeysClient:          apikeys.NewAPIClient(apiKeysCfg),
		cmkClient:              cmk.NewAPIClient(cmkCfg),
		connectClient:          connect.NewAPIClient(connectCfg),
		iamClient:              iam.NewAPIClient(iamCfg),
		iamV1Client:            iamv1.NewAPIClient(iamV1Cfg),
		netClient:              net.NewAPIClient(netCfg),
		oidcClient:             oidc.NewAPIClient(oidcCfg),
		orgClient:              org.NewAPIClient(orgCfg),
		kafkaRestClientFactory: &KafkaRestClientFactory{userAgent: userAgent},
		mdsClient:              mds.NewAPIClient(mdsCfg),
		userAgent:              userAgent,
		cloudApiKey:            cloudApiKey,
		cloudApiSecret:         cloudApiSecret,
		kafkaApiKey:            kafkaApiKey,
		kafkaApiSecret:         kafkaApiSecret,
		kafkaRestEndpoint:      kafkaRestEndpoint,
		// For simplicity, treat all 3 variables as a "single" one
		isKafkaMetadataSet: allKafkaAttributesAreSet,
	}

	return &client, nil
}
