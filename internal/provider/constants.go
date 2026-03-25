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

import "time"

const (
	terraformProviderUserAgent = "terraform-provider-confluent"
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

const (
	acceptanceTestModePollInterval   = 1 * time.Second
	acceptanceTestModeWaitTime       = 1 * time.Second
	avroFormat                               = "AVRO"
	awsEgressPrivateLinkEndpoint                   = "AwsEgressPrivateLinkEndpoint"
	awsEgressPrivateLinkGatewaySpecKind       = "AwsEgressPrivateLinkGatewaySpec"
	awsGlueSpecKind   = "AwsGlue"
	awsIngressPrivateLinkEndpoint                  = "AwsIngressPrivateLinkEndpoint"
	awsIngressPrivateLinkGatewaySpecKind      = "AwsIngressPrivateLinkGatewaySpec"
	AwsIntegrationConfigKind = "AwsIntegrationConfig"
	awsPeeringKind          = "AwsPeering"
	awsPrivateLinkAccessKind   = "AwsPrivateLinkAccess"
	awsPrivateNetworkInterface                     = "AwsPrivateNetworkInterface"
	awsPrivateNetworkInterfaceGatewaySpecKind = "AwsPrivateNetworkInterfaceGatewaySpec"
	awsTransitGatewayAttachmentKind = "AwsTransitGatewayAttachment"
	azureEgressPrivateLinkEndpoint                 = "AzureEgressPrivateLinkEndpoint"
	azureEgressPrivateLinkGatewaySpecKind     = "AzureEgressPrivateLinkGatewaySpec"
	azurePeeringKind        = "AzurePeering"
	azurePrivateLinkAccessKind = "AzurePrivateLinkAccess"
	azureSpecKind          = "AzureDataLakeStorageGen2"
	basicAuthCredentialsSourceConfig      = "basic.auth.credentials.source"
	basicAuthUserInfoConfig               = "basic.auth.user.info"
	bearerAuthClientId          = "bearer.auth.client.id"
	bearerAuthClientSecret      = "bearer.auth.client.secret"
	bearerAuthCredentialsSource = "bearer.auth.credentials.source"
	bearerAuthIdentityPoolId    = "bearer.auth.identity.pool.id"
	bearerAuthIssuerEndpointUrl = "bearer.auth.issuer.endpoint.url"
	bearerAuthLogicalCluster    = "bearer.auth.logical.cluster"
	bearerAuthScope             = "bearer.auth.scope"
	billingPackageAdvanced   = "ADVANCED"
	billingPackageEssentials = "ESSENTIALS"
	bootstrapServersConfigKey = "bootstrap.servers"
	byobAwsSpecKind        = "ByobAws"
	cloudKindInLowercase     = "cloud"
	clusterKind              = "Cluster"
	cmkApiVersion       = "cmk/v2"
	compatibilityLevelBackward           = "BACKWARD"
	compatibilityLevelBackwardTransitive = "BACKWARD_TRANSITIVE"
	compatibilityLevelForward            = "FORWARD"
	compatibilityLevelForwardTransitive  = "FORWARD_TRANSITIVE"
	compatibilityLevelFull               = "FULL"
	compatibilityLevelFullTransitive     = "FULL_TRANSITIVE"
	compatibilityLevelNone               = "NONE"
	configOAuthBearer = "OAUTHBEARER"
	configOperationDelete       = "DELETE"
	connectAPICreateTimeout        = 24 * time.Hour
	connectAPIWaitAfterCreate      = 5 * time.Second
	connectionModeConfigKey  = "connection.mode"
	connectionModeInbound    = "INBOUND"
	connectionModeOutbound   = "OUTBOUND"
	connectionTypePeering        = "PEERING"
	connectionTypePrivateLink    = "PRIVATELINK"
	connectionTypeTransitGateway = "TRANSITGATEWAY"
	connectOffsetsAPIUpdateTimeout = 1 * time.Hour
	connectorConfigAttributeClass  = "connector.class"
	connectorConfigAttributeName   = "name"
	connectorConfigAttributePlugin = "confluent.custom.plugin.id"
	connectorConfigAttributeType   = "confluent.connector.type"
	connectorConfigInternalAttributePrefix = "config.internal."
	connectorTypeCustom            = "CUSTOM"
	connectorTypeManaged           = "MANAGED"
	allGoogleApisNormalized = "all-google-apis"
	importer = "TFImporter"
	crnEnvironmentSuffix = "/environment="
	crnOrgSuffix = "/organization="
	dataCatalogAPIWaitAfterCreate = 30 * time.Second
	dataCatalogExporterTimeout    = 10 * time.Minute
	dataCatalogTimeout            = time.Minute
	defaultOutputPath       = "./imported_confluent_infrastructure"
	defaultTfStateFile      = "terraform.tfstate"
	defaultVariablesTfFile  = "variables.tf"
	docsClusterConfigUrl = "https://docs.confluent.io/cloud/current/clusters/broker-config.html#change-cluster-settings-for-dedicated-clusters"
	docsClusterLinkConfigUrl = "https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/cluster-links-cc.html#configuring-cluster-link-behavior"
	docsUrl                     = "https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic"
	dynamicClusterLinkConfig = "DYNAMIC_CLUSTER_LINK_CONFIG"
	dynamicTopicConfig          = "DYNAMIC_TOPIC_CONFIG"
	errorHandlingLogMode     = "LOG"
	errorHandlingSkipMode    = "SKIP"
	errorHandlingSuspendMode = "SUSPEND"
	externalTokenExpirationBuffer = 3 * time.Minute
	fcpmAPICreateTimeout = 1 * time.Hour
	fcpmAPIDeleteTimeout = 1 * time.Hour
	fcpmApiVersion      = "fcpm/v2"
	fieldEntityType  = "sr_field"
	flinkCarryOverOffsetsProperty = "sql.tables.initial-offset-from"
	forwardViaGcp       = "ForwardViaGcpDnsZones"
	forwardViaIp        = "ForwardViaIp"
	gcpEgressPrivateServiceConnectEndpoint         = "GcpEgressPrivateServiceConnectEndpoint"
	gcpPeeringKind          = "GcpPeering"
	gcpPrivateLinkAccessKind   = "GcpPrivateServiceConnectAccess"
	govCloudNotAvailableErrorMessage = "this service is not available in confluent cloud for government"
	highAvailability = "HIGH"
	iamApiVersion       = "iam/v2"
	importDestinationKafkaBootstrapEndpointEnvVar = "IMPORT_DESTINATION_KAFKA_BOOTSTRAP_ENDPOINT"
	importDestinationKafkaRestEndpointEnvVar      = "IMPORT_DESTINATION_KAFKA_REST_ENDPOINT"
	importerCreateTimeout = 8 * time.Hour
	importLocalKafkaBootstrapEndpointEnvVar  = "IMPORT_LOCAL_KAFKA_BOOTSTRAP_ENDPOINT"
	importLocalKafkaRestEndpointEnvVar       = "IMPORT_LOCAL_KAFKA_REST_ENDPOINT"
	importRemoteKafkaBootstrapEndpointEnvVar = "IMPORT_REMOTE_KAFKA_BOOTSTRAP_ENDPOINT"
	importRemoteKafkaRestEndpointEnvVar      = "IMPORT_REMOTE_KAFKA_REST_ENDPOINT"
	importSourceKafkaBootstrapEndpointEnvVar      = "IMPORT_SOURCE_KAFKA_BOOTSTRAP_ENDPOINT"
	importSourceKafkaRestEndpointEnvVar           = "IMPORT_SOURCE_KAFKA_REST_ENDPOINT"
	jsonFormat                               = "JSON"
	kafkaClusterTypeBasic            = "Basic"
	kafkaClusterTypeDedicated        = "Dedicated"
	kafkaClusterTypeEnterprise       = "Enterprise"
	kafkaClusterTypeFreight          = "Freight"
	kafkaClusterTypeStandard         = "Standard"
	kafkaQuotasAPIWaitAfterCreate = 30 * time.Second
	kafkaQuotasAPIWaitAfterUpdate = 15 * time.Second
	kafkaRestAPIWaitAfterCreate = 10 * time.Second
	kindAws   = "AwsKey"
	kindAzure = "AzureKey"
	kindGcp   = "GcpKey"
	ksqlCreateTimeout             = 12 * time.Hour
	ksqldbcmApiVersion  = "ksqldbcm/v2"
	ksqlDbKind               = "ksqlDB"
	latestSchemaVersionAndPlaceholderForSchemaIdentifier = "latest"
	linkModeBidirectional    = "BIDIRECTIONAL"
	linkModeConfigKey        = "link.mode"
	linkModeDestination      = "DESTINATION"
	linkModeSource           = "SOURCE"
	listComputePoolsPageSize = 99
	listEndpointsPageSize = 100
	listEnvironmentsPageSize = 99
	listFlinkArtifactsPageSize = 99
	listFlinkRegionsPageSize = 99
	listGatewaysPageSize = 99
	listGroupMappingsPageSize = 99
	listIdentityPoolsPageSize = 99
	listIdentityProvidersPageSize = 99
	listIPAddressesPageSize = 99
	listKafkaClustersPageSize = 99
	listKsqlClustersPageSize = 99
	listNetworkLinkServicesPageSize = 99
	listNetworksPageSize = 99
	listPeeringsPageSize = 99
	listPrivateLinkAccessesPageSize = 99
	listProviderIntegrationsPageSize = 99
	listSchemaRegistryClustersPageSize = 99
	listServiceAccountsPageSize = 99
	listTransitGatewayAttachmentsPageSize = 99
	listUsersPageSize = 100
	localSaslJaasConfigConfigKey                  = "local.sasl.jaas.config"
	localSaslLoginCallbackHandlerClassConfigKey   = "local.sasl.login.callback.handler.class"
	localSaslMechanismConfigKey                   = "local.sasl.mechanism"
	localSaslOAuthBearerTokenEndpointUrlConfigKey = "local.sasl.oauthbearer.token.endpoint.url"
	localSecurityProtocolConfigKey                = "local.security.protocol"
	lowAvailability  = "LOW"
	managedStorageSpecKind = "Managed"
	modeImport           = "IMPORT"
	modeReadOnly         = "READONLY"
	modeReadOnlyOverride = "READONLY_OVERRIDE"
	modeReadWrite        = "READWRITE"
	multiZone        = "MULTI_ZONE"
	networkingAPICreateTimeout = 2 * time.Hour
	networkingAPIDeleteTimeout = 5 * time.Hour
	pageTokenQueryParameter     = "page_token"
	presignedUrlLocation           = "PRESIGNED_URL_LOCATION"
	principalPrefix = "User:"
	privateLinkAccessPoint      = "PrivateLinkAccessPoint"
	protobufFormat                           = "PROTOBUF"
	providerAzure = "AZURE"
	providerGcp   = "GCP"
	qualifiedName = "qualifiedName"
	rbacWaitAfterCreateToSync = 90 * time.Second
	recordEntityType = "sr_record"
	regionKind               = "Region"
	remoteLinkConnectionMode = "remote.link.connection.mode"
	resumeFlinkStatementErrorFormat = "error resuming Flink Statement: %s"
	saslJaasConfigConfigKey                  = "sasl.jaas.config"
	saslLoginCallbackHandlerClassConfigKey   = "sasl.login.callback.handler.class"
	saslMechanismConfigKey                   = "sasl.mechanism"
	saslOAuthBearerTokenEndpointUrlConfigKey = "sasl.oauthbearer.token.endpoint.url"
	schemaEntityType = "sr_schema"
	schemaExporterAPICreateTimeout = 12 * time.Hour
	schemaNotCompatibleErrorMessage = `Compatibility check on the schema has failed against one or more versions in the subject, depending on how the compatibility is set.
See https://docs.confluent.io/platform/current/schema-registry/avro.html#sr-compatibility-types for details.
For example, if compatibility on the subject is set to BACKWARD, FORWARD, or FULL, the compatibility check is against the latest version.
If compatibility is set to one of the TRANSITIVE types, the check is against all previous versions.`
	schemaRegistryAPIWaitAfterCreateOrDelete = 10 * time.Second
	schemaRegistryKind       = "SchemaRegistry"
	schemaRegistryUrlConfig               = "schema.registry.url"
	securityProtocolConfigKey = "security.protocol"
	serviceAccountKind       = "ServiceAccount"
	singleZone       = "SINGLE_ZONE"
	snowflakeSpecKind = "Snowflake"
	srcmV2ApiVersion    = "srcm/v2"
	srcmV3ApiVersion    = "srcm/v3"
	stateActive                      = "ACTIVE"
	stateApplied = "APPLIED"
	stateCompleted = "COMPLETED"
	stateCreated                     = "CREATED"
	stateDegraded = "DEGRADED"
	stateDeProvisioning = "DEPROVISIONING"
	stateDone       = "DONE"
	stateExpired                     = "EXPIRED"
	stateFailed        = "FAILED"
	stateFailedOver                  = "FAILED_OVER"
	stateFailing   = "FAILING"
	stateInactive       = "INACTIVE"
	stateInProgress = "IN_PROGRESS"
	statementsAPICreateTimeout = 6 * time.Hour
	statePaused   = "PAUSED"
	statePending   = "PENDING"
	statePendingAccept = "PENDING_ACCEPT"
	statePendingStopped              = "PENDING_STOPPED"
	stateProcessing           = "PROCESSING"
	statePromoted                    = "PROMOTED"
	stateProvisioned   = "PROVISIONED"
	stateProvisioning  = "PROVISIONING"
	stateReady         = "READY"
	stateRunning       = "RUNNING"
	stateStopped                     = "STOPPED"
	stateStopping  = "STOPPING"
	stateUnexpected    = "UNEXPECTED"
	stateUnknown       = "UNKNOWN"
	stateUp                          = "UP"
	stateWaitingForConnections                  = "WAITING_FOR_CONNECTIONS"
	stateWaitingForProcessing = "WAITING_FOR_PROCESSING"
	statusAccepted     = "INVITE_STATUS_ACCEPTED"
	stopFlinkStatementErrorFormat   = "error stopping Flink Statement: %s"
	stsTokenExpirationBuffer      = 1 * time.Minute
	tableflowApiVersion = "tableflow/v1"
	tableflowKind            = "Tableflow"
	tableflowKindInLowercase = "tableflow"
	tfConfigurationFileName = "main.tf"
	tfLockFileName          = ".terraform.lock.hcl"
	tfStateFileName         = "terraform.tfstate"
	twoStarsOrMorePattern = "^[*]{2,}"
	unitySpecKind     = "Unity"
	userKind                 = "User"
)

const (
	Cloud ImporterMode = iota
	Kafka
	SchemaRegistry
)
