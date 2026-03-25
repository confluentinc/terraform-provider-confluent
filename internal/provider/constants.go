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

const (
	terraformProviderUserAgent = "terraform-provider-confluent"
)

const (
	byokKeyLoggingKey                         = "byok_key_id"
	connectArtifactLoggingKey                 = "connect_artifact_id"
	certificateAuthorityKey                   = "certificate_authority_id"
	certificatePoolKey                        = "certificate_pool_id"
	crnKafkaSuffix                            = "/kafka="
	kafkaAclLoggingKey                        = "kafka_acl_id"
	kafkaClusterLoggingKey                    = "kafka_cluster_id"
	kafkaClusterConfigLoggingKey              = "kafka_cluster_config_id"
	schemaRegistryClusterLoggingKey           = "schema_registry_cluster_id"
	kafkaTopicLoggingKey                      = "kafka_topic_id"
	serviceAccountLoggingKey                  = "service_account_id"
	userLoggingKey                            = "user_id"
	environmentLoggingKey                     = "environment_id"
	tfImporterLoggingKey                      = "tf_importer_environment_id"
	roleBindingLoggingKey                     = "role_binding_id"
	apiKeyLoggingKey                          = "api_key_id"
	computePoolLoggingKey                     = "compute_pool_id"
	flinkArtifactLoggingKey                   = "flink_artifact_id"
	flinkConnectionLoggingKey                 = "flink_connection_id"
	flinkStatementLoggingKey                  = "flink_statement_key_id"
	networkLoggingKey                         = "network_key_id"
	customConnectorPluginLoggingKey           = "custom_connector_plugin_key_id"
	customConnectorPluginVersionLoggingKey    = "custom_connector_plugin_version_key_id"
	pluginLoggingKey                          = "plugin_key_id"
	connectorLoggingKey                       = "connector_key_id"
	groupMappingLoggingKey                    = "group_mapping_id"
	privateLinkAccessLoggingKey               = "private_link_access_id"
	privateLinkAttachmentLoggingKey           = "private_link_attachment_id"
	privateLinkAttachmentConnectionLoggingKey = "private_link_attachment_connection_id"
	networkLinkEndpointLoggingKey             = "network_link_endpoint_id"
	networkLinkServiceLoggingKey              = "network_link_service_id"
	peeringLoggingKey                         = "peering_id"
	dnsForwarderKey                           = "dns_forwarder_id"
	dnsRecordKey                              = "dns_record_id"
	accessPointKey                            = "access_point_id"
	gatewayKey                                = "gateway_id"
	transitGatewayAttachmentLoggingKey        = "transit_gateway_attachment_id"
	ksqlClusterLoggingKey                     = "ksql_cluster_id"
	identityProviderLoggingKey                = "identity_provider_id"
	identityPoolLoggingKey                    = "identity_pool_id"
	ipGroupLoggingKey                         = "ip_group_id"
	ipFilterLoggingKey                        = "ip_filter_id"
	clusterLinkLoggingKey                     = "cluster_link_id"
	kafkaMirrorTopicLoggingKey                = "kafka_mirror_topic_id"
	kafkaClientQuotaLoggingKey                = "kafka_client_quota_id"
	schemaLoggingKey                          = "schema_id"
	schemaExporterLoggingKey                  = "schema_exporter_id"
	tagLoggingKey                             = "tag_id"
	tagBindingLoggingKey                      = "tag_binding_id"
	businessMetadataLoggingKey                = "business_metadata_id"
	businessMetadataBindingLoggingKey         = "business_metadata_binding_id"
	subjectModeLoggingKey                     = "subject_mode_id"
	subjectConfigLoggingKey                   = "subject_config_id"
	schemaRegistryClusterModeLoggingKey       = "schema_registry_cluster_mode_id"
	schemaRegistryClusterConfigLoggingKey     = "schema_registry_cluster_config_id"
	invitationLoggingKey                      = "invitation_id"
	tfCustomConnectorPluginTestUrl            = "TF_TEST_URL"
	flinkOrganizationIdTest                   = "1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	flinkEnvironmentIdTest                    = "env-abc123"
	schemaRegistryKekKey                      = "kek_id"
	schemaRegistryDekKey                      = "dek_id"
	entityAttributesLoggingKey                = "entity_attributes_id"
	providerIntegrationLoggingKey             = "provider_integration_id"
	tableflowTopicKey                         = "tableflow_topic_id"
	catalogIntegrationKey                     = "catalog_integration_id"

	deprecationMessageMajorRelease3 = "The %q %s has been deprecated and will be removed in the next major version of the provider (3.0.0). " +
		"Refer to the Upgrade Guide at https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-3-upgrade for more details. " +
		"The guide will be published once version 3.0.0 is released."
)
