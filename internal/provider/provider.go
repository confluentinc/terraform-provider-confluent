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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	apikeysv2 "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	byokv1 "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
	camv1 "github.com/confluentinc/ccloud-sdk-go-v2/cam/v1"
	ccpmv1 "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
	certificateauthorityv2 "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	cmkv2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	connectcustompluginv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin/v1"
	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	datacatalogv1 "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	endpointv1 "github.com/confluentinc/ccloud-sdk-go-v2/endpoint/v1"
	flinkartifactv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	flinkv2 "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	iamipfilteringv2 "github.com/confluentinc/ccloud-sdk-go-v2/iam-ip-filtering/v2"
	iamv1 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v1"
	iamv2 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	identityproviderv2 "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	kafkaquotasv1 "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	ksqlv2 "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	mdsv2 "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	networkingaccesspointv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
	networkingdnsforwarderv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	networkinggatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	networkingipv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-ip/v1"
	networkingprivatelinkv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink/v1"
	networkingv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	orgv2 "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	providerintegrationv1 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
	providerintegrationv2 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v2"
	srcmv3 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	ssov2 "github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"
	stsv1 "github.com/confluentinc/ccloud-sdk-go-v2/sts/v1"
)

type Client struct {
	apiKeysV2Client                 *apikeysv2.APIClient
	byokV1Client                    *byokv1.APIClient
	iamV2Client                     *iamv2.APIClient
	iamIpFilteringV2Client          *iamipfilteringv2.APIClient
	iamV1Client                     *iamv1.APIClient
	certificateAuthorityV2Client    *certificateauthorityv2.APIClient
	connectCustomPluginV1Client     *connectcustompluginv1.APIClient
	ccpmV1Client                    *ccpmv1.APIClient
	camV1Client                     *camv1.APIClient
	cmkV2Client                     *cmkv2.APIClient
	connectV1Client                 *connectv1.APIClient
	dataCatalogV1Client             *datacatalogv1.APIClient
	catalogRestClientFactory        *CatalogRestClientFactory
	flinkV2Client                   *flinkv2.APIClient
	flinkArtifactV1Client           *flinkartifactv1.APIClient
	networkingV1Client              *networkingv1.APIClient
	networkingAccessPointV1Client   *networkingaccesspointv1.APIClient
	networkingGatewayV1Client       *networkinggatewayv1.APIClient
	networkingIpV1Client            *networkingipv1.APIClient
	networkingPrivatelinkV1Client   *networkingprivatelinkv1.APIClient
	networkingDnsforwarderV1Client  *networkingdnsforwarderv1.APIClient
	orgV2Client                     *orgv2.APIClient
	ksqlV2Client                    *ksqlv2.APIClient
	flinkRestClientFactory          *FlinkRestClientFactory
	kafkaRestClientFactory          *KafkaRestClientFactory
	schemaRegistryRestClientFactory *SchemaRegistryRestClientFactory
	tableflowRestClientFactory      *TableflowRestClientFactory
	mdsV2Client                     *mdsv2.APIClient
	identityProviderV2Client        *identityproviderv2.APIClient
	kafkaQuotasV1Client             *kafkaquotasv1.APIClient
	srcmV3Client                    *srcmv3.APIClient
	ssoV2Client                     *ssov2.APIClient
	stsV1Client                     *stsv1.APIClient
	providerIntegrationV1Client     *providerintegrationv1.APIClient
	providerIntegrationV2Client     *providerintegrationv2.APIClient
	endpointV1Client                *endpointv1.APIClient
	userAgent                       string
	catalogRestEndpoint             string
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
	isCatalogRegistryMetadataSet    bool
	isSchemaRegistryMetadataSet     bool
	flinkPrincipalId                string
	flinkOrganizationId             string
	flinkEnvironmentId              string
	flinkComputePoolId              string
	flinkApiKey                     string
	flinkApiSecret                  string
	flinkRestEndpoint               string
	oauthToken                      *OAuthToken
	stsToken                        *STSToken
	isFlinkMetadataSet              bool
	tableflowApiKey                 string
	tableflowApiSecret              string
	isTableflowMetadataSet          bool
	isAcceptanceTestMode            bool
	isLiveProductionTestMode        bool
	isOAuthEnabled                  bool
	// cli-tfgen:tf-client-fields
}

type ResourceMetadataSetFlags struct {
	isCatalogMetadataSet        bool
	isFlinkMetadataSet          bool
	isKafkaMetadataSet          bool
	isSchemaRegistryMetadataSet bool
	isTableflowMetadataSet      bool
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
				"catalog_rest_endpoint": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CATALOG_REST_ENDPOINT", ""),
					Description: "The Stream Catalog REST Endpoint.",
				},
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
				"tableflow_api_key": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("TABLEFLOW_API_KEY", ""),
					Description: "The Tableflow API Key.",
				},
				"tableflow_api_secret": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc("TABLEFLOW_API_SECRET", ""),
					Description: "The Tableflow API Secret.",
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
				"oauth": providerOAuthSchema(),
			},
			DataSourcesMap: map[string]*schema.Resource{
				"confluent_catalog_integration":                catalogIntegrationDataSource(),
				"confluent_certificate_authority":              certificateAuthorityDataSource(),
				"confluent_certificate_pool":                   certificatePoolDataSource(),
				"confluent_cluster_link":                       clusterLinkDataSource(),
				"confluent_connect_artifact":                   connectArtifactDataSource(),
				"confluent_ip_filter":                          ipFilterDataSource(),
				"confluent_ip_group":                           ipGroupDataSource(),
				"confluent_kafka_cluster":                      kafkaDataSource(),
				"confluent_kafka_clusters":                     kafkaClustersDataSource(),
				"confluent_kafka_topic":                        kafkaTopicDataSource(),
				"confluent_environment":                        environmentDataSource(),
				"confluent_environments":                       environmentsDataSource(),
				"confluent_group_mapping":                      groupMappingDataSource(),
				"confluent_ksql_cluster":                       ksqlDataSource(),
				"confluent_flink_artifact":                     flinkArtifactDataSource(),
				"confluent_flink_compute_pool":                 computePoolDataSource(),
				"confluent_flink_connection":                   flinkConnectionDataSource(),
				"confluent_flink_region":                       flinkRegionDataSource(),
				"confluent_identity_pool":                      identityPoolDataSource(),
				"confluent_identity_provider":                  identityProviderDataSource(),
				"confluent_ip_addresses":                       ipAddressesDataSource(),
				"confluent_kafka_client_quota":                 kafkaClientQuotaDataSource(),
				"confluent_network":                            networkDataSource(),
				"confluent_access_point":                       accessPointDataSource(),
				"confluent_endpoint":                           endpointDataSource(),
				"confluent_dns_record":                         dnsRecordDataSource(),
				"confluent_gateway":                            gatewayDataSource(),
				"confluent_gateways":                           gatewaysDataSource(),
				"confluent_organization":                       organizationDataSource(),
				"confluent_peering":                            peeringDataSource(),
				"confluent_transit_gateway_attachment":         transitGatewayAttachmentDataSource(),
				"confluent_private_link_access":                privateLinkAccessDataSource(),
				"confluent_private_link_attachment":            privateLinkAttachmentDataSource(),
				"confluent_private_link_attachment_connection": privateLinkAttachmentConnectionDataSource(),
				"confluent_provider_integration":               providerIntegrationDataSource(),
				"confluent_provider_integration_setup":         providerIntegrationSetupDataSource(),
				"confluent_provider_integration_authorization": providerIntegrationAuthorizationDataSource(),
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
				"confluent_tableflow_topic":                    tableflowTopicDataSource(),
				"confluent_tag":                                tagDataSource(),
				"confluent_tag_binding":                        tagBindingDataSource(),
				"confluent_business_metadata":                  businessMetadataDataSource(),
				"confluent_business_metadata_binding":          businessMetadataBindingDataSource(),
				"confluent_schema_registry_kek":                schemaRegistryKekDataSource(),
				"confluent_schema_registry_dek":                schemaRegistryDekDataSource(),
				// cli-tfgen:tf-datasources
			},
			ResourcesMap: map[string]*schema.Resource{
				"confluent_catalog_integration":                catalogIntegrationResource(),
				"confluent_api_key":                            apiKeyResource(),
				"confluent_byok_key":                           byokResource(),
				"confluent_certificate_authority":              certificateAuthorityResource(),
				"confluent_certificate_pool":                   certificatePoolResource(),
				"confluent_cluster_link":                       clusterLinkResource(),
				"confluent_connect_artifact":                   connectArtifactResource(),
				"confluent_ip_group":                           ipGroupResource(),
				"confluent_ip_filter":                          ipFilterResource(),
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
				"confluent_flink_connection":                   flinkConnectionResource(),
				"confluent_flink_statement":                    flinkStatementResource(),
				"confluent_connector":                          connectorResource(),
				"confluent_custom_connector_plugin":            customConnectorPluginResource(),
				"confluent_custom_connector_plugin_version":    customConnectorPluginVersionResource(),
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
				"confluent_plugin":                             pluginResource(),
				"confluent_private_link_access":                privateLinkAccessResource(),
				"confluent_private_link_attachment":            privateLinkAttachmentResource(),
				"confluent_private_link_attachment_connection": privateLinkAttachmentConnectionResource(),
				"confluent_provider_integration":               providerIntegrationResource(),
				"confluent_provider_integration_setup":         providerIntegrationSetupResource(),
				"confluent_provider_integration_authorization": providerIntegrationAuthorizationResource(),
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
				"confluent_tableflow_topic":                    tableflowTopicResource(),
				"confluent_tag":                                tagResource(),
				"confluent_tag_binding":                        tagBindingResource(),
				"confluent_business_metadata":                  businessMetadataResource(),
				"confluent_business_metadata_binding":          businessMetadataBindingResource(),
				"confluent_schema_registry_kek":                schemaRegistryKekResource(),
				"confluent_schema_registry_dek":                schemaRegistryDekResource(),
				"confluent_catalog_entity_attributes":          catalogEntityAttributesResource(),
				// cli-tfgen:tf-resources
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
		Required:    true,
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
	catalogRestEndpoint := d.Get("catalog_rest_endpoint").(string)
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
	tableflowApiKey := d.Get("tableflow_api_key").(string)
	tableflowApiSecret := d.Get("tableflow_api_secret").(string)
	maxRetries := d.Get("max_retries").(int)

	userAgent := p.UserAgent(terraformProviderUserAgent, fmt.Sprintf("%s (https://confluent.cloud; support@confluent.io)", providerVersion))
	if additionalUserAgent != "" {
		userAgent = fmt.Sprintf("%s %s", additionalUserAgent, userAgent)
	}

	acceptanceTestMode := false
	if os.Getenv("TF_ACC") == "1" {
		acceptanceTestMode = true
	}
	tflog.Info(ctx, fmt.Sprintf("Provider: acceptance test mode is %t\n", acceptanceTestMode))

	liveProductionTestMode := false
	if os.Getenv("TF_ACC_PROD") == "1" {
		liveProductionTestMode = true
	}
	tflog.Info(ctx, fmt.Sprintf("Provider: live production test mode is %t\n", liveProductionTestMode))

	apiKeysV2Cfg := apikeysv2.NewConfiguration()
	byokV1Cfg := byokv1.NewConfiguration()
	certificateAuthorityV2Cfg := certificateauthorityv2.NewConfiguration()
	camV1Cfg := camv1.NewConfiguration()
	dataCatalogV1Cfg := datacatalogv1.NewConfiguration()
	connectCustomPluginV1Cfg := connectcustompluginv1.NewConfiguration()
	ccpmV1Cfg := ccpmv1.NewConfiguration()
	cmkV2Cfg := cmkv2.NewConfiguration()
	connectV1Cfg := connectv1.NewConfiguration()
	flinkArtifactV1Cfg := flinkartifactv1.NewConfiguration()
	flinkV2Cfg := flinkv2.NewConfiguration()
	iamV2Cfg := iamv2.NewConfiguration()
	iamIpFilteringV2Cfg := iamipfilteringv2.NewConfiguration()
	iamV1Cfg := iamv1.NewConfiguration()
	ksqlV2Cfg := ksqlv2.NewConfiguration()
	mdsV2Cfg := mdsv2.NewConfiguration()
	networkingAccessPointV1Cfg := networkingaccesspointv1.NewConfiguration()
	networkingGatewayV1Cfg := networkinggatewayv1.NewConfiguration()
	networkingV1Cfg := networkingv1.NewConfiguration()
	networkingIpV1Cfg := networkingipv1.NewConfiguration()
	networkingPrivatelinkV1Cfg := networkingprivatelinkv1.NewConfiguration()
	networkingDnsforwarderV1Cfg := networkingdnsforwarderv1.NewConfiguration()
	endpointV1Cfg := endpointv1.NewConfiguration()
	identityProviderV2Cfg := identityproviderv2.NewConfiguration()
	orgV2Cfg := orgv2.NewConfiguration()
	providerIntegrationV1Cfg := providerintegrationv1.NewConfiguration()
	providerIntegrationV2Cfg := providerintegrationv2.NewConfiguration()
	kafkaQuotasV1Cfg := kafkaquotasv1.NewConfiguration()
	srcmV3Cfg := srcmv3.NewConfiguration()
	ssoV2Cfg := ssov2.NewConfiguration()
	stsV1Cfg := stsv1.NewConfiguration()
	// cli-tfgen:tf-client-cfg

	apiKeysV2Cfg.Servers[0].URL = endpoint
	byokV1Cfg.Servers[0].URL = endpoint
	certificateAuthorityV2Cfg.Servers[0].URL = endpoint
	connectCustomPluginV1Cfg.Servers[0].URL = endpoint
	ccpmV1Cfg.Servers[0].URL = endpoint
	camV1Cfg.Servers[0].URL = endpoint
	cmkV2Cfg.Servers[0].URL = endpoint
	connectV1Cfg.Servers[0].URL = endpoint
	flinkArtifactV1Cfg.Servers[0].URL = endpoint
	flinkV2Cfg.Servers[0].URL = endpoint
	iamV2Cfg.Servers[0].URL = endpoint
	iamIpFilteringV2Cfg.Servers[0].URL = endpoint
	iamV1Cfg.Servers[0].URL = endpoint
	ksqlV2Cfg.Servers[0].URL = endpoint
	mdsV2Cfg.Servers[0].URL = endpoint
	networkingV1Cfg.Servers[0].URL = endpoint
	networkingIpV1Cfg.Servers[0].URL = endpoint
	networkingPrivatelinkV1Cfg.Servers[0].URL = endpoint
	networkingAccessPointV1Cfg.Servers[0].URL = endpoint
	networkingGatewayV1Cfg.Servers[0].URL = endpoint
	networkingDnsforwarderV1Cfg.Servers[0].URL = endpoint
	endpointV1Cfg.Servers[0].URL = endpoint
	identityProviderV2Cfg.Servers[0].URL = endpoint
	orgV2Cfg.Servers[0].URL = endpoint
	providerIntegrationV1Cfg.Servers[0].URL = endpoint
	providerIntegrationV2Cfg.Servers[0].URL = endpoint
	kafkaQuotasV1Cfg.Servers[0].URL = endpoint
	srcmV3Cfg.Servers[0].URL = endpoint
	ssoV2Cfg.Servers[0].URL = endpoint
	stsV1Cfg.Servers[0].URL = endpoint
	// cli-tfgen:tf-client-endpoint

	apiKeysV2Cfg.UserAgent = userAgent
	byokV1Cfg.UserAgent = userAgent
	certificateAuthorityV2Cfg.UserAgent = userAgent
	dataCatalogV1Cfg.UserAgent = userAgent
	connectCustomPluginV1Cfg.UserAgent = userAgent
	ccpmV1Cfg.UserAgent = userAgent
	camV1Cfg.UserAgent = userAgent
	cmkV2Cfg.UserAgent = userAgent
	connectV1Cfg.UserAgent = userAgent
	flinkArtifactV1Cfg.UserAgent = userAgent
	flinkV2Cfg.UserAgent = userAgent
	iamV2Cfg.UserAgent = userAgent
	iamIpFilteringV2Cfg.UserAgent = userAgent
	iamV1Cfg.UserAgent = userAgent
	ksqlV2Cfg.UserAgent = userAgent
	mdsV2Cfg.UserAgent = userAgent
	networkingV1Cfg.UserAgent = userAgent
	networkingAccessPointV1Cfg.UserAgent = userAgent
	networkingGatewayV1Cfg.UserAgent = userAgent
	networkingIpV1Cfg.UserAgent = userAgent
	networkingDnsforwarderV1Cfg.UserAgent = userAgent
	endpointV1Cfg.UserAgent = userAgent
	networkingPrivatelinkV1Cfg.UserAgent = userAgent
	identityProviderV2Cfg.UserAgent = userAgent
	orgV2Cfg.UserAgent = userAgent
	providerIntegrationV1Cfg.UserAgent = userAgent
	providerIntegrationV2Cfg.UserAgent = userAgent
	kafkaQuotasV1Cfg.UserAgent = userAgent
	srcmV3Cfg.UserAgent = userAgent
	ssoV2Cfg.UserAgent = userAgent
	stsV1Cfg.UserAgent = userAgent
	// cli-tfgen:tf-client-useragent

	var catalogRestClientFactory *CatalogRestClientFactory
	var flinkRestClientFactory *FlinkRestClientFactory
	var kafkaRestClientFactory *KafkaRestClientFactory
	var schemaRegistryRestClientFactory *SchemaRegistryRestClientFactory
	var tableflowRestClientFactory *TableflowRestClientFactory

	catalogRestClientFactory = &CatalogRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	flinkRestClientFactory = &FlinkRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	kafkaRestClientFactory = &KafkaRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	schemaRegistryRestClientFactory = &SchemaRegistryRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries}
	tableflowRestClientFactory = &TableflowRestClientFactory{ctx: ctx, userAgent: userAgent, maxRetries: &maxRetries, endpoint: endpoint}

	apiKeysV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	byokV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	certificateAuthorityV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	dataCatalogV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	connectCustomPluginV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ccpmV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	camV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	cmkV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	connectV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	flinkArtifactV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	flinkV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	iamV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	iamIpFilteringV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	iamV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ksqlV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	mdsV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	networkingV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	networkingGatewayV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	networkingIpV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	networkingPrivatelinkV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	networkingDnsforwarderV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	endpointV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	identityProviderV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	orgV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	providerIntegrationV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	providerIntegrationV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	kafkaQuotasV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	srcmV3Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	ssoV2Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	stsV1Cfg.HTTPClient = NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()
	// cli-tfgen:tf-client-httpclient

	secureTokenServiceClient := stsv1.NewAPIClient(stsV1Cfg)

	var externalOAuthToken *OAuthToken
	var stsOAuthToken *STSToken
	var err diag.Diagnostics
	var oauthEnabled bool
	var resourceMetadataFlags ResourceMetadataSetFlags
	if _, ok := d.GetOk(paramOAuthBlockName); ok {
		oauthEnabled = true
		externalOAuthToken, stsOAuthToken, err = initializeOAuthConfigs(ctx, d, secureTokenServiceClient)
		if err != nil {
			return nil, err
		}
		if err = validateOAuthAndProviderAPIKeysCoexist(
			cloudApiKey, cloudApiSecret,
			kafkaApiKey, kafkaApiSecret,
			schemaRegistryApiKey, schemaRegistryApiSecret,
			flinkApiKey, flinkApiSecret,
			tableflowApiKey, tableflowApiSecret); err != nil {
			return nil, err
		}
		// Verify that the resources specific attributes should NOT be partially set when OAuth is enabled
		if resourceMetadataFlags, err = validateAllOrNoneAttributesSetForResourcesWithOAuth(
			kafkaClusterId, kafkaRestEndpoint,
			schemaRegistryClusterId, schemaRegistryRestEndpoint, catalogRestEndpoint,
			flinkOrganizationId, flinkEnvironmentId, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId); err != nil {
			return nil, err
		}
	} else {
		resourceMetadataFlags, err = validateAllOrNoneAttributesSetForResources(
			kafkaApiKey, kafkaApiSecret, kafkaClusterId, kafkaRestEndpoint,
			schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryClusterId, schemaRegistryRestEndpoint, catalogRestEndpoint,
			flinkApiKey, flinkApiSecret, flinkOrganizationId, flinkEnvironmentId, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId,
			tableflowApiKey, tableflowApiSecret)
		if err != nil {
			return nil, err
		}
	}

	client := Client{
		apiKeysV2Client:                 apikeysv2.NewAPIClient(apiKeysV2Cfg),
		byokV1Client:                    byokv1.NewAPIClient(byokV1Cfg),
		dataCatalogV1Client:             datacatalogv1.NewAPIClient(dataCatalogV1Cfg),
		certificateAuthorityV2Client:    certificateauthorityv2.NewAPIClient(certificateAuthorityV2Cfg),
		connectCustomPluginV1Client:     connectcustompluginv1.NewAPIClient(connectCustomPluginV1Cfg),
		ccpmV1Client:                    ccpmv1.NewAPIClient(ccpmV1Cfg),
		camV1Client:                     camv1.NewAPIClient(camV1Cfg),
		cmkV2Client:                     cmkv2.NewAPIClient(cmkV2Cfg),
		connectV1Client:                 connectv1.NewAPIClient(connectV1Cfg),
		flinkArtifactV1Client:           flinkartifactv1.NewAPIClient(flinkArtifactV1Cfg),
		flinkV2Client:                   flinkv2.NewAPIClient(flinkV2Cfg),
		iamV2Client:                     iamv2.NewAPIClient(iamV2Cfg),
		iamIpFilteringV2Client:          iamipfilteringv2.NewAPIClient(iamIpFilteringV2Cfg),
		iamV1Client:                     iamv1.NewAPIClient(iamV1Cfg),
		ksqlV2Client:                    ksqlv2.NewAPIClient(ksqlV2Cfg),
		networkingV1Client:              networkingv1.NewAPIClient(networkingV1Cfg),
		networkingAccessPointV1Client:   networkingaccesspointv1.NewAPIClient(networkingAccessPointV1Cfg),
		networkingGatewayV1Client:       networkinggatewayv1.NewAPIClient(networkingGatewayV1Cfg),
		networkingIpV1Client:            networkingipv1.NewAPIClient(networkingIpV1Cfg),
		networkingPrivatelinkV1Client:   networkingprivatelinkv1.NewAPIClient(networkingPrivatelinkV1Cfg),
		networkingDnsforwarderV1Client:  networkingdnsforwarderv1.NewAPIClient(networkingDnsforwarderV1Cfg),
		endpointV1Client:                endpointv1.NewAPIClient(endpointV1Cfg),
		identityProviderV2Client:        identityproviderv2.NewAPIClient(identityProviderV2Cfg),
		orgV2Client:                     orgv2.NewAPIClient(orgV2Cfg),
		providerIntegrationV1Client:     providerintegrationv1.NewAPIClient(providerIntegrationV1Cfg),
		providerIntegrationV2Client:     providerintegrationv2.NewAPIClient(providerIntegrationV2Cfg),
		srcmV3Client:                    srcmv3.NewAPIClient(srcmV3Cfg),
		catalogRestClientFactory:        catalogRestClientFactory,
		flinkRestClientFactory:          flinkRestClientFactory,
		kafkaRestClientFactory:          kafkaRestClientFactory,
		schemaRegistryRestClientFactory: schemaRegistryRestClientFactory,
		tableflowRestClientFactory:      tableflowRestClientFactory,
		mdsV2Client:                     mdsv2.NewAPIClient(mdsV2Cfg),
		kafkaQuotasV1Client:             kafkaquotasv1.NewAPIClient(kafkaQuotasV1Cfg),
		ssoV2Client:                     ssov2.NewAPIClient(ssoV2Cfg),
		stsV1Client:                     secureTokenServiceClient,
		// cli-tfgen:tf-client-literal
		userAgent:                  userAgent,
		catalogRestEndpoint:        catalogRestEndpoint,
		cloudApiKey:                cloudApiKey,
		cloudApiSecret:             cloudApiSecret,
		kafkaClusterId:             kafkaClusterId,
		kafkaApiKey:                kafkaApiKey,
		kafkaApiSecret:             kafkaApiSecret,
		kafkaRestEndpoint:          kafkaRestEndpoint,
		schemaRegistryClusterId:    schemaRegistryClusterId,
		schemaRegistryApiKey:       schemaRegistryApiKey,
		schemaRegistryApiSecret:    schemaRegistryApiSecret,
		schemaRegistryRestEndpoint: schemaRegistryRestEndpoint,
		flinkPrincipalId:           flinkPrincipalId,
		flinkOrganizationId:        flinkOrganizationId,
		flinkEnvironmentId:         flinkEnvironmentId,
		flinkComputePoolId:         flinkComputePoolId,
		flinkApiKey:                flinkApiKey,
		flinkApiSecret:             flinkApiSecret,
		flinkRestEndpoint:          flinkRestEndpoint,
		tableflowApiKey:            tableflowApiKey,
		tableflowApiSecret:         tableflowApiSecret,
		oauthToken:                 externalOAuthToken,
		stsToken:                   stsOAuthToken,

		// For simplicity, treat 3 (for Kafka), 4 (for SR), 4 (for catalog), 7 (for Flink), and 2 (for Tableflow) variables as a "single" one
		isKafkaMetadataSet:           resourceMetadataFlags.isKafkaMetadataSet,
		isKafkaClusterIdSet:          kafkaClusterId != "",
		isSchemaRegistryMetadataSet:  resourceMetadataFlags.isSchemaRegistryMetadataSet,
		isCatalogRegistryMetadataSet: resourceMetadataFlags.isCatalogMetadataSet,
		isFlinkMetadataSet:           resourceMetadataFlags.isFlinkMetadataSet,
		isTableflowMetadataSet:       resourceMetadataFlags.isTableflowMetadataSet,
		isAcceptanceTestMode:         acceptanceTestMode,
		isLiveProductionTestMode:     liveProductionTestMode,
		isOAuthEnabled:               oauthEnabled,
	}

	return &client, nil
}

func initializeOAuthConfigs(ctx context.Context, d *schema.ResourceData, stsV1Client *stsv1.APIClient) (*OAuthToken, *STSToken, diag.Diagnostics) {
	tflog.Info(ctx, "Initializing OAuth settings for Confluent Cloud")
	providedToken := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthExternalAccessToken)
	identityPoolId := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthIdentityPoolId)
	maxRetries := d.Get("max_retries").(int)
	var oauthToken *OAuthToken
	var err error

	// Use this single retryable client to fetch external token
	retryableClient := NewRetryableClientFactory(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()

	if providedToken == "" {
		// External OAuth token initialization through fetching token from external IDP
		clientId := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthExternalClientId)
		clientSecret := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthExternalClientSecret)
		externalTokenURL := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthExternalTokenURL)
		scope := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthExternalTokenScope)
		oauthToken, err = fetchExternalOAuthToken(ctx, externalTokenURL, clientId, clientSecret, scope, identityPoolId, nil, retryableClient)
		if err != nil {
			return nil, nil, diag.FromErr(err)
		}
	} else {
		// External OAuth token initialization through fetching token from provided token
		tflog.Warn(ctx, "Initializing OAuth setting from provided static token, please make sure this external token from Identity Provider is valid and not expired")
		tflog.Warn(ctx, "This token and the associated STS token won't be refreshed automatically by the provider, please make sure to update it manually in case of expiration")
		oauthToken = &OAuthToken{
			AccessToken:      providedToken,
			IdentityPoolId:   identityPoolId,
			ClientId:         "",
			ClientSecret:     "",
			TokenUrl:         "",
			ExpiresInSeconds: "",
			Scope:            "",
			TokenType:        "",
			ValidUntil:       time.Now().Add(100 * 365 * 24 * time.Hour),
			HTTPClient:       retryableClient,
		}
	}

	// STS token exchanged from external OAuth token
	// STS token will be fetched using the STS special client
	expiredInSeconds := extractStringValueFromBlock(d, paramOAuthBlockName, paramOAuthSTSTokenExpiredInSeconds)
	stsToken, err := fetchSTSOAuthToken(ctx, oauthToken.AccessToken, identityPoolId, expiredInSeconds, nil, stsV1Client)
	if err != nil {
		return nil, nil, diag.FromErr(err)
	}

	return oauthToken, stsToken, nil
}

func providerOAuthSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramOAuthExternalTokenURL: {
					Type:     schema.TypeString,
					Optional: true,
					// A user should provide a value for either "oauth_external_token_url" or "oauth_external_access_token" attribute, not both
					ExactlyOneOf: []string{"oauth.0.oauth_external_token_url", "oauth.0.oauth_external_access_token"},
					Description:  "OAuth token URL to fetch access token from external Identity Provider.",
				},
				paramOAuthExternalClientId: {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "OAuth token application client id from external Identity Provider.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramOAuthExternalClientSecret: {
					Type:         schema.TypeString,
					Optional:     true,
					Sensitive:    true,
					Description:  "OAuth token application client secret from external Identity Provider.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramOAuthExternalAccessToken: {
					Type:      schema.TypeString,
					Optional:  true,
					Sensitive: true,
					// A user should provide a value for either "oauth_external_token_url" or "oauth_external_access_token" attribute, not both
					ExactlyOneOf: []string{"oauth.0.oauth_external_token_url", "oauth.0.oauth_external_access_token"},
					Description:  "OAuth existing static access token already fetched from external Identity Provider.",
				},
				paramOAuthExternalTokenScope: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "OAuth client application scope, this is a required field when using Microsoft Azure Entra ID as the identity provider.",
				},
				paramOAuthIdentityPoolId: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "OAuth identity pool id used for processing external token and exchange STS token, registered with Confluent Cloud.",
				},
				paramOAuthSTSTokenExpiredInSeconds: {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "OAuth STS access token expired in second from Confluent Cloud",
				},
			},
		},
		Description: "OAuth config settings",
		Optional:    true,
		MinItems:    1,
		MaxItems:    1,
	}
}

func validateOAuthAndProviderAPIKeysCoexist(
	cloudApiKey, cloudApiSecret,
	kafkaApiKey, kafkaApiSecret,
	schemaRegistryApiKey, schemaRegistryApiSecret,
	flinkApiKey, flinkApiSecret,
	tableflowApiKey, tableflowApiSecret string) diag.Diagnostics {
	if cloudApiKey != "" || cloudApiSecret != "" {
		return diag.Errorf("(cloud_api_key, cloud_api_secret) attributes should not be set in the provider block when oauth block is present")
	}
	if kafkaApiKey != "" || kafkaApiSecret != "" {
		return diag.Errorf("(kafka_api_key, kafka_api_secret) attributes should not be set in the provider block when oauth block is present")
	}
	if schemaRegistryApiKey != "" || schemaRegistryApiSecret != "" {
		return diag.Errorf("(schema_registry_api_key, schema_registry_api_secret) attributes should not be set in the provider block when oauth block is present")
	}
	if flinkApiKey != "" || flinkApiSecret != "" {
		return diag.Errorf("(flink_api_key, flink_api_secret) attributes should not be set in the provider block when oauth block is present")
	}
	if tableflowApiKey != "" || tableflowApiSecret != "" {
		return diag.Errorf("(tableflow_api_key, tableflow_api_secret) attributes should not be set in the provider block when oauth block is present")
	}
	return nil
}

func SleepIfNotTestMode(d time.Duration, isAcceptanceTestMode bool, isLiveProductionTestMode bool) {
	// In live production test mode, use full delay since we're testing against real infrastructure
	if isAcceptanceTestMode && !isLiveProductionTestMode {
		time.Sleep(500 * time.Millisecond)
		return
	}
	time.Sleep(d)
}
