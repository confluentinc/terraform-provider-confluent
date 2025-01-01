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
	"os"
	"strings"
	"time"

	ccp "github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin/v1"
	dns "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	netip "github.com/confluentinc/ccloud-sdk-go-v2/networking-ip/v1"
	pi "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
	"github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"

	apikeys "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	byok "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
	ca "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	cmk "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	connect "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	fa "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	fcpm "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	iamv1 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v1"
	iam "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	oidc "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	quotas "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	ksql "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	netap "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
	netgw "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	netpl "github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink/v1"
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	org "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	srcm "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	terraformProviderUserAgent = "terraform-provider-confluent"
)

const (
	paramApiVersion              = "api_version"
	paramCloud                   = "cloud"
	paramRegion                  = "region"
	paramOrganization            = "organization"
	paramEnvironment             = "environment"
	paramNonResourceSpecificKind = "api_key_type_non_resource_specific"
	paramId                      = "id"
	paramDisplayName             = "display_name"
	paramName                    = "name"
	paramDescription             = "description"
	paramKind                    = "kind"
	paramCsu                     = "csu"
	paramClass                   = "class"
	paramContentFormat           = "content_format"
	paramRuntimeLanguage         = "runtime_language"
	paramArtifactFile            = "artifact_file"
	paramVersions                = "versions"
)

type Client struct {
	apiKeysClient                   *apikeys.APIClient
	byokClient                      *byok.APIClient
	iamClient                       *iam.APIClient
	iamV1Client                     *iamv1.APIClient
	caClient                        *ca.APIClient
	ccpClient                       *ccp.APIClient
	cmkClient                       *cmk.APIClient
	connectClient                   *connect.APIClient
	fcpmClient                      *fcpm.APIClient
	faClient                        *fa.APIClient
	netClient                       *net.APIClient
	netAccessPointClient            *netap.APIClient
	netGatewayClient                *netgw.APIClient
	netIpClient                     *netip.APIClient
	netPLClient                     *netpl.APIClient
	netDnsClient                    *dns.APIClient
	orgClient                       *org.APIClient
	ksqlClient                      *ksql.APIClient
	flinkRestClientFactory          *FlinkRestClientFactory
	kafkaRestClientFactory          *KafkaRestClientFactory
	schemaRegistryRestClientFactory *SchemaRegistryRestClientFactory
	mdsClient                       *mds.APIClient
	oidcClient                      *oidc.APIClient
	quotasClient                    *quotas.APIClient
	srcmClient                      *srcm.APIClient
	ssoClient                       *sso.APIClient
	piClient                        *pi.APIClient
	userAgent                       string
	cloudApiKey                     string
	cloudApiSecret                  string
	kafkaClusterId                  string
	kafkaApiKey                     string
	kafkaApiSecret                  string
	kafkaRestEndpoint               string
	isKafkaClusterIdSet             bool
	isKafkaMetadataSet              bool
	schemaRegistryClusterId         string
	schemaRegistryApiKey            string
	schemaRegistryApiSecret         string
	schemaRegistryRestEndpoint      string
	isSchemaRegistryMetadataSet     bool
	flinkPrincipalId                string
	flinkOrganizationId             string
	flinkEnvironmentId              string
	flinkComputePoolId              string
	flinkApiKey                     string
	flinkApiSecret                  string
	flinkRestEndpoint               string
	isFlinkMetadataSet              bool
	isAcceptanceTestMode            bool
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

func New(version, userAgent string) func() *schema.Provider {
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
				"kafka_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("KAFKA_ID", ""),
					Description: "The Kafka Cluster ID.",
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
				"schema_registry_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("SCHEMA_REGISTRY_ID", ""),
					Description: "The Schema Registry Cluster ID.",
				},
				"schema_registry_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("SCHEMA_REGISTRY_API_KEY", ""),
					Description: "The Schema Registry Cluster API Key.",
				},
				"schema_registry_api_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("SCHEMA_REGISTRY_API_SECRET", ""),
					Description: "The Schema Registry Cluster API Secret.",
				},
				"schema_registry_rest_endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("SCHEMA_REGISTRY_REST_ENDPOINT", ""),
					Description: "The Schema Registry Cluster REST Endpoint.",
				},
				"flink_principal_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("FLINK_PRINCIPAL_ID", ""),
					Description: "The Flink Principal ID.",
				},
				"organization_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_ORGANIZATION_ID", ""),
					Description: "The Flink Organization ID.",
				},
				"environment_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_ENVIRONMENT_ID", ""),
					Description: "The Flink Environment ID.",
				},
				"flink_compute_pool_id": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("FLINK_COMPUTE_POOL_ID", ""),
					Description: "The Flink Compute Pool ID.",
				},
				"flink_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("FLINK_API_KEY", ""),
					Description: "The Flink API Key.",
				},
				"flink_api_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("FLINK_API_SECRET", ""),
					Description: "The Flink API Secret.",
				},
				"flink_rest_endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("FLINK_REST_ENDPOINT", ""),
					// Example: "https://flink.us-east-1.aws.confluent.cloud"
					Description: "The Flink REST Endpoint.",
				},
				"endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "https://api.confluent.cloud",
					Description: "The base endpoint of Confluent Cloud API.",
				},
				"max_retries": {
					Type:         schema.TypeInt,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("TF_PROVIDER_CONFLUENT_MAX_RETRIES", 4),
					ValidateFunc: validation.IntAtLeast(4),
					Description:  "Maximum number of retries of HTTP client. Defaults to 4.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"confluent_certificate_authority":              certificateAuthorityDataSource(),
				"confluent_certificate_pool":                   certificatePoolDataSource(),
				"confluent_kafka_cluster":                      kafkaDataSource(),
				"confluent_kafka_topic":                        kafkaTopicDataSource(),
				"confluent_environment":                        environmentDataSource(),
				"confluent_environments":                       environmentsDataSource(),
				"confluent_group_mapping":                      groupMappingDataSource(),
				"confluent_ksql_cluster":                       ksqlDataSource(),
				"confluent_flink_artifact":                     flinkArtifactDataSource(),
				"confluent_flink_compute_pool":                 computePoolDataSource(),
				"confluent_flink_region":                       flinkRegionDataSource(),
				"confluent_identity_pool":                      identityPoolDataSource(),
				"confluent_identity_provider":                  identityProviderDataSource(),
				"confluent_ip_addresses":                       ipAddressesDataSource(),
				"confluent_kafka_client_quota":                 kafkaClientQuotaDataSource(),
				"confluent_network":                            networkDataSource(),
				"confluent_access_point":                       accessPointDataSource(),
				"confluent_dns_record":                         dnsRecordDataSource(),
				"confluent_gateway":                            gatewayDataSource(),
				"confluent_organization":                       organizationDataSource(),
				"confluent_peering":                            peeringDataSource(),
				"confluent_transit_gateway_attachment":         transitGatewayAttachmentDataSource(),
				"confluent_private_link_access":                privateLinkAccessDataSource(),
				"confluent_private_link_attachment":            privateLinkAttachmentDataSource(),
				"confluent_private_link_attachment_connection": privateLinkAttachmentConnectionDataSource(),
				"confluent_provider_integration":               providerIntegrationDataSource(),
				"confluent_role_binding":                       roleBindingDataSource(),
				"confluent_schema":                             schemaDataSource(),
				"confluent_schemas":                            schemasDataSource(),
				"confluent_users":                              usersDataSource(),
				"confluent_service_account":                    serviceAccountDataSource(),
				"confluent_schema_registry_cluster":            schemaRegistryClusterDataSource(),
				"confluent_schema_registry_clusters":           schemaRegistryClustersDataSource(),
				"confluent_subject_mode":                       subjectModeDataSource(),
				"confluent_subject_config":                     subjectConfigDataSource(),
				"confluent_schema_registry_cluster_config":     schemaRegistryClusterConfigDataSource(),
				"confluent_schema_registry_cluster_mode":       schemaRegistryClusterModeDataSource(),
				"confluent_user":                               userDataSource(),
				"confluent_invitation":                         invitationDataSource(),
				"confluent_byok_key":                           byokDataSource(),
				"confluent_network_link_endpoint":              networkLinkEndpointDataSource(),
				"confluent_network_link_service":               networkLinkServiceDataSource(),
				"confluent_tag":                                tagDataSource(),
				"confluent_tag_binding":                        tagBindingDataSource(),
				"confluent_business_metadata":                  businessMetadataDataSource(),
				"confluent_business_metadata_binding":          businessMetadataBindingDataSource(),
				"confluent_schema_registry_kek":                schemaRegistryKekDataSource(),
				"confluent_schema_registry_dek":                schemaRegistryDekDataSource(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"confluent_api_key":                            apiKeyResource(),
				"confluent_byok_key":                           byokResource(),
				"confluent_certificate_authority":              certificateAuthorityResource(),
				"confluent_certificate_pool":                   certificatePoolResource(),
				"confluent_cluster_link":                       clusterLinkResource(),
				"confluent_kafka_cluster":                      kafkaResource(),
				"confluent_kafka_cluster_config":               kafkaConfigResource(),
				"confluent_environment":                        environmentResource(),
				"confluent_identity_pool":                      identityPoolResource(),
				"confluent_identity_provider":                  identityProviderResource(),
				"confluent_group_mapping":                      groupMappingResource(),
				"confluent_kafka_client_quota":                 kafkaClientQuotaResource(),
				"confluent_ksql_cluster":                       ksqlResource(),
				"confluent_flink_artifact":                     artifactResource(),
				"confluent_flink_compute_pool":                 computePoolResource(),
				"confluent_flink_statement":                    flinkStatementResource(),
				"confluent_connector":                          connectorResource(),
				"confluent_custom_connector_plugin":            customConnectorPluginResource(),
				"confluent_service_account":                    serviceAccountResource(),
				"confluent_kafka_topic":                        kafkaTopicResource(),
				"confluent_kafka_mirror_topic":                 kafkaMirrorTopicResource(),
				"confluent_kafka_acl":                          kafkaAclResource(),
				"confluent_network":                            networkResource(),
				"confluent_access_point":                       accessPointResource(),
				"confluent_dns_forwarder":                      dnsForwarderResource(),
				"confluent_dns_record":                         dnsRecordResource(),
				"confluent_gateway":                            gatewayResource(),
				"confluent_peering":                            peeringResource(),
				"confluent_private_link_access":                privateLinkAccessResource(),
				"confluent_private_link_attachment":            privateLinkAttachmentResource(),
				"confluent_private_link_attachment_connection": privateLinkAttachmentConnectionResource(),
				"confluent_provider_integration":               providerIntegrationResource(),
				"confluent_role_binding":                       roleBindingResource(),
				"confluent_schema":                             schemaResource(),
				"confluent_schema_exporter":                    schemaExporterResource(),
				"confluent_subject_mode":                       subjectModeResource(),
				"confluent_subject_config":                     subjectConfigResource(),
				"confluent_schema_registry_cluster_mode":       schemaRegistryClusterModeResource(),
				"confluent_schema_registry_cluster_config":     schemaRegistryClusterConfigResource(),
				"confluent_transit_gateway_attachment":         transitGatewayAttachmentResource(),
				"confluent_invitation":                         invitationResource(),
				"confluent_network_link_endpoint":              networkLinkEndpointResource(),
				"confluent_network_link_service":               networkLinkServiceResource(),
				"confluent_tf_importer":                        tfImporterResource(),
				"confluent_tag":                                tagResource(),
				"confluent_tag_binding":                        tagBindingResource(),
				"confluent_business_metadata":                  businessMetadataResource(),
				"confluent_business_metadata_binding":          businessMetadataBindingResource(),
				"confluent_schema_registry_kek":                schemaRegistryKekResource(),
				"confluent_schema_registry_dek":                schemaRegistryDekResource(),
				"confluent_catalog_entity_attributes":          catalogEntityAttributesResource(),
			},
		}

		provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
			return providerConfigure(ctx, d, provider, version, userAgent)
		}

		return provider
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
// //  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
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
		Optional:    true, // set to be optional for tableflow resource, will add ValidateFunc later
		MinItems:    1,
		MaxItems:    1,
		ForceNew:    true,
		Description: "Environment objects represent an isolated namespace for your Confluent resources for organizational purposes.",
	}
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
// //  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
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

func providerConfigure(ctx context.Context, d *schema.ResourceData, p *schema.Provider, providerVersion, additionalUserAgent string) (interface{}, diag.Diagnostics) {
	tflog.Info(ctx, "Initializing Terraform Provider for Confluent Cloud")
	endpoint := d.Get("endpoint").(string)
	cloudApiKey := d.Get("cloud_api_key").(string)
	cloudApiSecret := d.Get("cloud_api_secret").(string)
	kafkaClusterId := d.Get("kafka_id").(string)
	kafkaApiKey := d.Get("kafka_api_key").(string)
	kafkaApiSecret := d.Get("kafka_api_secret").(string)
	kafkaRestEndpoint := d.Get("kafka_rest_endpoint").(string)
	schemaRegistryClusterId := d.Get("schema_registry_id").(string)
	schemaRegistryApiKey := d.Get("schema_registry_api_key").(string)
	schemaRegistryApiSecret := d.Get("schema_registry_api_secret").(string)
	schemaRegistryRestEndpoint := d.Get("schema_registry_rest_endpoint").(string)
	flinkPrincipalId := d.Get("flink_principal_id").(string)
	flinkOrganizationId := d.Get("organization_id").(string)
	flinkEnvironmentId := d.Get("environment_id").(string)
	flinkComputePoolId := d.Get("flink_compute_pool_id").(string)
	flinkApiKey := d.Get("flink_api_key").(string)
	flinkApiSecret := d.Get("flink_api_secret").(string)
	flinkRestEndpoint := d.Get("flink_rest_endpoint").(string)
	maxRetries := d.Get("max_retries").(int)

	// 3 or 4 attributes should be set or not set at the same time
	// Option #2: (kafka_api_key, kafka_api_secret, kafka_rest_endpoint)
	// Option #3 (primary): (kafka_api_key, kafka_api_secret, kafka_rest_endpoint, kafka_id)
	allKafkaAttributesAreSet := (kafkaApiKey != "") && (kafkaApiSecret != "") && (kafkaRestEndpoint != "")
	allKafkaAttributesAreNotSet := (kafkaApiKey == "") && (kafkaApiSecret == "") && (kafkaRestEndpoint == "")
	justOneOrTwoKafkaAttributesAreSet := !(allKafkaAttributesAreSet || allKafkaAttributesAreNotSet)
	if justOneOrTwoKafkaAttributesAreSet {
		return nil, diag.Errorf("(kafka_api_key, kafka_api_secret, kafka_rest_endpoint) or (kafka_api_key, kafka_api_secret, kafka_rest_endpoint, kafka_id) attributes should be set or not set in the provider block at the same time")
	}

	// All 4 attributes should be set or not set at the same time
	allSchemaRegistryAttributesAreSet := (schemaRegistryApiKey != "") && (schemaRegistryApiSecret != "") && (schemaRegistryRestEndpoint != "") && (schemaRegistryClusterId != "")
	allSchemaRegistryAttributesAreNotSet := (schemaRegistryApiKey == "") && (schemaRegistryApiSecret == "") && (schemaRegistryRestEndpoint == "") && (schemaRegistryClusterId == "")
	justSubsetOfSchemaRegistryAttributesAreSet := !(allSchemaRegistryAttributesAreSet || allSchemaRegistryAttributesAreNotSet)
	if justSubsetOfSchemaRegistryAttributesAreSet {
		return nil, diag.Errorf("All 4 schema_registry_api_key, schema_registry_api_secret, schema_registry_rest_endpoint, schema_registry_id attributes should be set or not set in the provider block at the same time")
	}

	// All 7 attributes should be set or not set at the same time
	allFlinkAttributesAreSet := (flinkApiKey != "") && (flinkApiSecret != "") && (flinkRestEndpoint != "") && (flinkOrganizationId != "") && (flinkEnvironmentId != "") && (flinkComputePoolId != "") && (flinkPrincipalId != "")
	allFlinkAttributesAreNotSet := (flinkApiKey == "") && (flinkApiSecret == "") && (flinkRestEndpoint == "") && (flinkOrganizationId == "") && (flinkEnvironmentId == "") && (flinkComputePoolId == "") && (flinkPrincipalId == "")
	justSubsetOfFlinkAttributesAreSet := !(allFlinkAttributesAreSet || allFlinkAttributesAreNotSet)
	if justSubsetOfFlinkAttributesAreSet {
		return nil, diag.Errorf("All 7 flink_api_key, flink_api_secret, flink_rest_endpoint, organization_id, environment_id, flink_compute_pool_id, flink_principal_id attributes should be set or not set in the provider block at the same time")
	}

	userAgent := p.UserAgent(terraformProviderUserAgent, fmt.Sprintf("%s (https://confluent.cloud; support@confluent.io)", providerVersion))
	if additionalUserAgent != "" {
		userAgent = fmt.Sprintf("%s %s", additionalUserAgent, userAgent)
	}

	acceptanceTestMode := false
	if os.Getenv("TF_ACC") == "1" {
		acceptanceTestMode = true
	}
	tflog.Info(ctx, fmt.Sprintf("Provider: acceptance test mode is %t\n", acceptanceTestMode))

	apiKeysCfg := apikeys.NewConfiguration()
	byokCfg := byok.NewConfiguration()
	caCfg := ca.NewConfiguration()
	ccpCfg := ccp.NewConfiguration()
	cmkCfg := cmk.NewConfiguration()
	connectCfg := connect.NewConfiguration()
	faCfg := fa.NewConfiguration()
	fcpmCfg := fcpm.NewConfiguration()
	iamCfg := iam.NewConfiguration()
	iamV1Cfg := iamv1.NewConfiguration()
	ksqlCfg := ksql.NewConfiguration()
	mdsCfg := mds.NewConfiguration()
	netAccessPointCfg := netap.NewConfiguration()
	netGatewayCfg := netgw.NewConfiguration()
	netCfg := net.NewConfiguration()
	netIpCfg := netip.NewConfiguration()
	netPLCfg := netpl.NewConfiguration()
	netDnsCfg := dns.NewConfiguration()
	oidcCfg := oidc.NewConfiguration()
	orgCfg := org.NewConfiguration()
	piCfg := pi.NewConfiguration()
	quotasCfg := quotas.NewConfiguration()
	srcmCfg := srcm.NewConfiguration()
	ssoCfg := sso.NewConfiguration()

	apiKeysCfg.Servers[0].URL = endpoint
	byokCfg.Servers[0].URL = endpoint
	caCfg.Servers[0].URL = endpoint
	ccpCfg.Servers[0].URL = endpoint
	cmkCfg.Servers[0].URL = endpoint
	connectCfg.Servers[0].URL = endpoint
	faCfg.Servers[0].URL = endpoint
	fcpmCfg.Servers[0].URL = endpoint
	iamCfg.Servers[0].URL = endpoint
	iamV1Cfg.Servers[0].URL = endpoint
	ksqlCfg.Servers[0].URL = endpoint
	mdsCfg.Servers[0].URL = endpoint
	netCfg.Servers[0].URL = endpoint
	netIpCfg.Servers[0].URL = endpoint
	netPLCfg.Servers[0].URL = endpoint
	netAccessPointCfg.Servers[0].URL = endpoint
	netGatewayCfg.Servers[0].URL = endpoint
	netDnsCfg.Servers[0].URL = endpoint
	oidcCfg.Servers[0].URL = endpoint
	orgCfg.Servers[0].URL = endpoint
	piCfg.Servers[0].URL = endpoint
	quotasCfg.Servers[0].URL = endpoint
	srcmCfg.Servers[0].URL = endpoint
	ssoCfg.Servers[0].URL = endpoint

	apiKeysCfg.UserAgent = userAgent
	byokCfg.UserAgent = userAgent
	caCfg.UserAgent = userAgent
	ccpCfg.UserAgent = userAgent
	cmkCfg.UserAgent = userAgent
	connectCfg.UserAgent = userAgent
	faCfg.UserAgent = userAgent
	fcpmCfg.UserAgent = userAgent
	iamCfg.UserAgent = userAgent
	iamV1Cfg.UserAgent = userAgent
	ksqlCfg.UserAgent = userAgent
	mdsCfg.UserAgent = userAgent
	netCfg.UserAgent = userAgent
	netAccessPointCfg.UserAgent = userAgent
	netGatewayCfg.UserAgent = userAgent
	netIpCfg.UserAgent = userAgent
	netDnsCfg.UserAgent = userAgent
	netPLCfg.UserAgent = userAgent
	oidcCfg.UserAgent = userAgent
	orgCfg.UserAgent = userAgent
	piCfg.UserAgent = userAgent
	quotasCfg.UserAgent = userAgent
	srcmCfg.UserAgent = userAgent
	ssoCfg.UserAgent = userAgent

	var flinkRestClientFactory *FlinkRestClientFactory
	var kafkaRestClientFactory *KafkaRestClientFactory
	var schemaRegistryRestClientFactory *SchemaRegistryRestClientFactory

	flinkRestClientFactory = &FlinkRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	kafkaRestClientFactory = &KafkaRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	schemaRegistryRestClientFactory = &SchemaRegistryRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}

	apiKeysCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	byokCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	caCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ccpCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	cmkCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	connectCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	faCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	fcpmCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	iamCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	iamV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ksqlCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	mdsCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	netCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	netGatewayCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	netIpCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	netPLCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	netDnsCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	oidcCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	orgCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	piCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	quotasCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	srcmCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ssoCfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()

	client := Client{
		apiKeysClient:                   apikeys.NewAPIClient(apiKeysCfg),
		byokClient:                      byok.NewAPIClient(byokCfg),
		caClient:                        ca.NewAPIClient(caCfg),
		ccpClient:                       ccp.NewAPIClient(ccpCfg),
		cmkClient:                       cmk.NewAPIClient(cmkCfg),
		connectClient:                   connect.NewAPIClient(connectCfg),
		faClient:                        fa.NewAPIClient(faCfg),
		fcpmClient:                      fcpm.NewAPIClient(fcpmCfg),
		iamClient:                       iam.NewAPIClient(iamCfg),
		iamV1Client:                     iamv1.NewAPIClient(iamV1Cfg),
		ksqlClient:                      ksql.NewAPIClient(ksqlCfg),
		netClient:                       net.NewAPIClient(netCfg),
		netAccessPointClient:            netap.NewAPIClient(netAccessPointCfg),
		netGatewayClient:                netgw.NewAPIClient(netGatewayCfg),
		netIpClient:                     netip.NewAPIClient(netIpCfg),
		netPLClient:                     netpl.NewAPIClient(netPLCfg),
		netDnsClient:                    dns.NewAPIClient(netDnsCfg),
		oidcClient:                      oidc.NewAPIClient(oidcCfg),
		orgClient:                       org.NewAPIClient(orgCfg),
		piClient:                        pi.NewAPIClient(piCfg),
		srcmClient:                      srcm.NewAPIClient(srcmCfg),
		flinkRestClientFactory:          flinkRestClientFactory,
		kafkaRestClientFactory:          kafkaRestClientFactory,
		schemaRegistryRestClientFactory: schemaRegistryRestClientFactory,
		mdsClient:                       mds.NewAPIClient(mdsCfg),
		quotasClient:                    quotas.NewAPIClient(quotasCfg),
		ssoClient:                       sso.NewAPIClient(ssoCfg),
		userAgent:                       userAgent,
		cloudApiKey:                     cloudApiKey,
		cloudApiSecret:                  cloudApiSecret,
		kafkaClusterId:                  kafkaClusterId,
		kafkaApiKey:                     kafkaApiKey,
		kafkaApiSecret:                  kafkaApiSecret,
		kafkaRestEndpoint:               kafkaRestEndpoint,
		schemaRegistryClusterId:         schemaRegistryClusterId,
		schemaRegistryApiKey:            schemaRegistryApiKey,
		schemaRegistryApiSecret:         schemaRegistryApiSecret,
		schemaRegistryRestEndpoint:      schemaRegistryRestEndpoint,
		flinkPrincipalId:                flinkPrincipalId,
		flinkOrganizationId:             flinkOrganizationId,
		flinkEnvironmentId:              flinkEnvironmentId,
		flinkComputePoolId:              flinkComputePoolId,
		flinkApiKey:                     flinkApiKey,
		flinkApiSecret:                  flinkApiSecret,
		flinkRestEndpoint:               flinkRestEndpoint,
		// For simplicity, treat 3 (for Kafka), 4 (for SR), and 7 (for Flink) variables as a "single" one
		isKafkaMetadataSet:          allKafkaAttributesAreSet,
		isKafkaClusterIdSet:         kafkaClusterId != "",
		isSchemaRegistryMetadataSet: allSchemaRegistryAttributesAreSet,
		isFlinkMetadataSet:          allFlinkAttributesAreSet,
		isAcceptanceTestMode:        acceptanceTestMode,
	}

	return &client, nil
}

func SleepIfNotTestMode(d time.Duration, isAcceptanceTestMode bool) {
	if isAcceptanceTestMode {
		time.Sleep(500 * time.Millisecond)
		return
	}
	time.Sleep(d)
}
