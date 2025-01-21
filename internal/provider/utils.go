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
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	apikeys "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	byok "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
	ca "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	cmk "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	ccp "github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin/v1"
	connect "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	dc "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	fa "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	fgb "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	fcpm "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	iamv1 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v1"
	iam "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	oidc "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	quotas "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	ksql "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	netap "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
	dns "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	netgw "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	netip "github.com/confluentinc/ccloud-sdk-go-v2/networking-ip/v1"
	netpl "github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink/v1"
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	org "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	pi "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
	schemaregistry "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	srcm "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	"github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"
	"github.com/dghubble/sling"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	byokKeyLoggingKey                         = "byok_key_id"
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
	flinkStatementLoggingKey                  = "flink_statement_key_id"
	networkLoggingKey                         = "network_key_id"
	customConnectorPluginLoggingKey           = "custom_connector_plugin_key_id"
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
	invitationloggingKey                      = "invitation_id"
	tfCustomConnectorPluginTestUrl            = "TF_TEST_URL"
	flinkOrganizationIdTest                   = "1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	flinkEnvironmentIdTest                    = "env-abc123"
	schemaRegistryKekKey                      = "kek_id"
	schemaRegistryDekKey                      = "dek_id"
	entityAttributesLoggingKey                = "entity_attributes_id"
	providerIntegrationLoggingKey             = "provider_integration_id"

	deprecationMessageMajorRelease3 = "The %q %s has been deprecated and will be removed in the next major version of the provider (3.0.0). " +
		"Refer to the Upgrade Guide at https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-3-upgrade for more details. " +
		"The guide will be published once version 3.0.0 is released."
)

func (c *Client) apiKeysApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), apikeys.ContextBasicAuth, apikeys.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) byokApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, byok.ContextBasicAuth, byok.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func kafkaRestApiContextWithClusterApiKey(ctx context.Context, kafkaApiKey string, kafkaApiSecret string) context.Context {
	if kafkaApiKey != "" && kafkaApiSecret != "" {
		return context.WithValue(context.Background(), kafkarestv3.ContextBasicAuth, kafkarestv3.BasicAuth{
			UserName: kafkaApiKey,
			Password: kafkaApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Kafka API Key")
	return ctx
}

func (c *Client) ccpApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), ccp.ContextBasicAuth, ccp.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) cmkApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), cmk.ContextBasicAuth, cmk.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) iamApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), iam.ContextBasicAuth, iam.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) caApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), ca.ContextBasicAuth, ca.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) ssoApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), sso.ContextBasicAuth, sso.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) piApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), pi.ContextBasicAuth, pi.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) iamV1ApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), iamv1.ContextBasicAuth, iamv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) mdsApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), mds.ContextBasicAuth, mds.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), net.ContextBasicAuth, net.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) faApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), fa.ContextBasicAuth, fa.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) fcpmApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), fcpm.ContextBasicAuth, fcpm.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netAPApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), netap.ContextBasicAuth, netap.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netGWApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), netgw.ContextBasicAuth, netgw.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netIPApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), netip.ContextBasicAuth, netip.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netPLApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), netpl.ContextBasicAuth, netpl.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) netDnsApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), dns.ContextBasicAuth, dns.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) srcmApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), srcm.ContextBasicAuth, srcm.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) connectApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), connect.ContextBasicAuth, connect.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) orgApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), org.ContextBasicAuth, org.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) ksqlApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, ksql.ContextBasicAuth, ksql.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) oidcApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), oidc.ContextBasicAuth, oidc.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) quotasApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), quotas.ContextBasicAuth, quotas.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func orgApiContext(ctx context.Context, cloudApiKey, cloudApiSecret string) context.Context {
	if cloudApiKey != "" && cloudApiSecret != "" {
		return context.WithValue(context.Background(), org.ContextBasicAuth, org.BasicAuth{
			UserName: cloudApiKey,
			Password: cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Cloud API Key or Cloud API Secret is empty")
	return ctx
}

func getTimeoutFor(clusterType string) time.Duration {
	if clusterType == kafkaClusterTypeDedicated {
		return 72 * time.Hour
	} else {
		return 1 * time.Hour
	}
}

func stringToAclResourceType(aclResourceType string) (kafkarestv3.AclResourceType, error) {
	switch aclResourceType {
	case "UNKNOWN":
		return kafkarestv3.UNKNOWN, nil
	case "ANY":
		return kafkarestv3.ANY, nil
	case "TOPIC":
		return kafkarestv3.TOPIC, nil
	case "GROUP":
		return kafkarestv3.GROUP, nil
	case "CLUSTER":
		return kafkarestv3.CLUSTER, nil
	case "TRANSACTIONAL_ID":
		return kafkarestv3.TRANSACTIONAL_ID, nil
	case "DELEGATION_TOKEN":
		return kafkarestv3.DELEGATION_TOKEN, nil
	}
	return "", fmt.Errorf("unknown ACL resource type was found: %q", aclResourceType)
}

type Acl struct {
	ResourceType kafkarestv3.AclResourceType
	ResourceName string
	PatternType  string
	Principal    string
	Host         string
	Operation    string
	Permission   string
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type KafkaRestClient struct {
	apiClient                     *kafkarestv3.APIClient
	clusterId                     string
	clusterApiKey                 string
	clusterApiSecret              string
	restEndpoint                  string
	isMetadataSetInProviderBlock  bool
	isClusterIdSetInProviderBlock bool
}

type SchemaRegistryRestClient struct {
	apiClient                    *schemaregistry.APIClient
	dataCatalogApiClient         *dc.APIClient
	clusterId                    string
	clusterApiKey                string
	clusterApiSecret             string
	restEndpoint                 string
	isMetadataSetInProviderBlock bool
}

type FlinkRestClient struct {
	apiClient                    *fgb.APIClient
	organizationId               string
	environmentId                string
	computePoolId                string
	principalId                  string
	flinkApiKey                  string
	flinkApiSecret               string
	restEndpoint                 string
	isMetadataSetInProviderBlock bool
}

func (c *KafkaRestClient) apiContext(ctx context.Context) context.Context {
	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(context.Background(), kafkarestv3.ContextBasicAuth, kafkarestv3.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Kafka API Key for Kafka Cluster %q", c.clusterId), map[string]interface{}{kafkaClusterLoggingKey: c.clusterId})
	return ctx
}

func (c *SchemaRegistryRestClient) apiContext(ctx context.Context) context.Context {
	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(context.Background(), schemaregistry.ContextBasicAuth, schemaregistry.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Schema Registry API Key for Stream Governance Cluster %q", c.clusterId))
	return ctx
}

func (c *SchemaRegistryRestClient) dataCatalogApiContext(ctx context.Context) context.Context {
	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(context.Background(), dc.ContextBasicAuth, dc.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Schema Registry API Key for Stream Governance Cluster %q", c.clusterId))
	return ctx
}

func (c *FlinkRestClient) apiContext(ctx context.Context) context.Context {
	if c.flinkApiKey != "" && c.flinkApiSecret != "" {
		return context.WithValue(context.Background(), fgb.ContextBasicAuth, fgb.BasicAuth{
			UserName: c.flinkApiKey,
			Password: c.flinkApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Flink API Key for Flink %q", c.restEndpoint))
	return ctx
}

type GenericOpenAPIError interface {
	Model() interface{}
}

func setStringAttributeInListBlockOfSizeOne(blockName, attributeName, attributeValue string, d *schema.ResourceData) error {
	return d.Set(blockName, []interface{}{map[string]interface{}{
		attributeName: attributeValue,
	}})
}

// createDescriptiveError will convert GenericOpenAPIError error into an error with a more descriptive error message.
// diag.FromErr(createDescriptiveError(err)) should be used instead of diag.FromErr(err) in this project
// since GenericOpenAPIError.Error() returns just HTTP status code and its generic name (i.e., "400 Bad Request")
func createDescriptiveError(err error) error {
	if err == nil {
		return nil
	}
	// At this point it's just status code and its generic name
	errorMessage := err.Error()
	// Add error.detail to the final error message
	if genericOpenAPIError, ok := err.(GenericOpenAPIError); ok {
		failure := genericOpenAPIError.Model()
		reflectedFailure := reflect.ValueOf(&failure).Elem().Elem()
		reflectedFailureValue := reflect.Indirect(reflectedFailure)
		if reflectedFailureValue.IsValid() {
			errs := reflectedFailureValue.FieldByName("Errors")
			kafkaRestOrConnectErr := reflectedFailureValue.FieldByName("Message")
			if errs.Kind() == reflect.Slice && errs.Len() > 0 {
				nest := errs.Index(0)
				detailPtr := nest.FieldByName("Detail")
				if detailPtr.IsValid() {
					errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(detailPtr))
				}
			} else if kafkaRestOrConnectErr.IsValid() && kafkaRestOrConnectErr.Kind() == reflect.Struct {
				detailPtr := kafkaRestOrConnectErr.FieldByName("value")
				if detailPtr.IsValid() {
					errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(detailPtr))
				}
			} else if kafkaRestOrConnectErr.IsValid() && kafkaRestOrConnectErr.Kind() == reflect.Pointer {
				errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(kafkaRestOrConnectErr))
			}
		}
	}
	return fmt.Errorf(errorMessage)
}

// Reports whether the response has http.StatusForbidden status due to an invalid Cloud API Key vs other reasons
// which is useful to distinguish from scenarios where http.StatusForbidden represents http.StatusNotFound for
// security purposes.
func ResponseHasStatusForbiddenDueToInvalidAPIKey(response *http.Response) bool {
	if ResponseHasExpectedStatusCode(response, http.StatusForbidden) {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return false
		}
		bodyString := string(bodyBytes)
		// Search for a specific error message that indicates the invalid Cloud API Key has been used
		return strings.Contains(bodyString, "invalid API key")
	}
	return false
}

func ResponseHasExpectedStatusCode(response *http.Response, expectedStatusCode int) bool {
	return response != nil && response.StatusCode == expectedStatusCode
}

func isNonKafkaRestApiResourceNotFound(response *http.Response) bool {
	return ResponseHasExpectedStatusCode(response, http.StatusNotFound) ||
		(ResponseHasExpectedStatusCode(response, http.StatusForbidden) && !ResponseHasStatusForbiddenDueToInvalidAPIKey(response))
}

// APIF-2043: TEMPORARY METHOD
// Converts principal with a resourceID (User:sa-01234) to principal with an integer ID (User:6789)
func principalWithResourceIdToPrincipalWithIntegerId(c *Client, principalWithResourceId string) (string, error) {
	// There's input validation that principal attribute must start with "User:sa-" or "User:u-" or "User:pool-" r "User:group-" or "User:*"

	if principalWithResourceId == "User:*" {
		return principalWithResourceId, nil
	}

	// User:sa-abc123 -> sa-abc123
	resourceId := principalWithResourceId[5:]
	if strings.HasPrefix(principalWithResourceId, "User:sa-") {
		integerId, err := saResourceIdToSaIntegerId(c, resourceId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%d", principalPrefix, integerId), nil
	} else if strings.HasPrefix(principalWithResourceId, "User:u-") {
		integerId, err := userResourceIdToUserIntegerId(c, resourceId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%d", principalPrefix, integerId), nil
	} else if strings.HasPrefix(principalWithResourceId, "User:pool-") || strings.HasPrefix(principalWithResourceId, "User:group-") {
		return principalWithResourceId, nil
	}
	return "", fmt.Errorf("the principal must start with 'User:sa-' or 'User:u-' or 'User:pool-' or 'User:group-' or 'User:*'")
}

// APIF-2043: TEMPORARY METHOD
// Converts service account's resourceID (sa-abc123) to its integer ID (67890)
func saResourceIdToSaIntegerId(c *Client, saResourceId string) (int, error) {
	list, _, err := c.iamV1Client.ServiceAccountsV1Api.ListV1ServiceAccounts(c.iamV1ApiContext(context.Background())).Execute()
	if err != nil {
		return 0, err
	}
	for _, sa := range list.GetUsers() {
		if sa.GetResourceId() == saResourceId {
			if sa.HasId() {
				return int(sa.GetId()), nil
			} else {
				return 0, fmt.Errorf("the matching integer ID for a service account with resource ID=%s is nil", saResourceId)
			}
		}
	}
	return 0, fmt.Errorf("the service account with resource ID=%s was not found", saResourceId)
}

// APIF-2043: TEMPORARY METHOD
// Converts user's resourceID (u-abc123) to its integer ID (67890)
func userResourceIdToUserIntegerId(c *Client, userResourceId string) (int, error) {
	list, _, err := c.iamV1Client.UsersV1Api.ListV1Users(c.iamV1ApiContext(context.Background())).Execute()
	if err != nil {
		return 0, err
	}
	for _, user := range list.GetUsers() {
		if user.GetResourceId() == userResourceId {
			if user.HasId() {
				return int(user.GetId()), nil
			} else {
				return 0, fmt.Errorf("the matching integer ID for a user with resource ID=%s is nil", userResourceId)
			}
		}
	}
	return 0, fmt.Errorf("the user with resource ID=%s was not found", userResourceId)
}

func clusterCrnToRbacClusterCrn(clusterCrn string) (string, error) {
	// Converts
	// crn://confluent.cloud/organization=./environment=./cloud-cluster=lkc-198rjz/kafka=lkc-198rjz
	// to
	// crn://confluent.cloud/organization=./environment=./cloud-cluster=lkc-198rjz
	lastIndex := strings.LastIndex(clusterCrn, crnKafkaSuffix)
	if lastIndex == -1 {
		return "", fmt.Errorf("could not find %s in %s", crnKafkaSuffix, clusterCrn)
	}
	return clusterCrn[:lastIndex], nil
}

func convertToStringStringMap(data map[string]interface{}) map[string]string {
	stringMap := make(map[string]string)

	for key, value := range data {
		stringMap[key] = value.(string)
	}

	return stringMap
}

func convertToStringStringListMap(data []interface{}) map[string][]string {
	stringListMap := make(map[string][]string)
	for _, item := range data {
		kv := item.(map[string]interface{})
		key := kv[paramKey].(string)
		value := convertToStringSlice(kv[paramValue].(*schema.Set).List())
		stringListMap[key] = value
	}
	return stringListMap
}

func ptr(s string) *string {
	return &s
}

func kafkaClusterBlockV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: kafkaClusterIdSchema(),
		},
	}
}
func kafkaClusterBlockStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	kafkaClusterIdString := rawState[paramKafkaCluster].(string)
	rawState[paramKafkaCluster] = []interface{}{map[string]interface{}{
		paramId: kafkaClusterIdString,
	}}
	return rawState, nil
}

// Extracts "foo" from "https://api.confluent.cloud/iam/v2/service-accounts?page_token=foo"
func extractPageToken(nextPageUrlString string) (string, error) {
	nextPageUrl, err := url.Parse(nextPageUrlString)
	if err != nil {
		return "", fmt.Errorf("could not parse %q into URL, %s", nextPageUrlString, createDescriptiveError(err))
	}
	pageToken := nextPageUrl.Query().Get(pageTokenQueryParameter)
	if pageToken == "" {
		return "", fmt.Errorf("could not parse the value for %q query parameter from %q", pageTokenQueryParameter, nextPageUrlString)
	}
	return pageToken, nil
}

func verifyListValues(values, acceptedValues []string, ignoreCase bool) error {
	for _, actualValue := range values {
		found := stringInSlice(actualValue, acceptedValues, ignoreCase)
		if !found {
			return fmt.Errorf("expected %s to be one of %v, got %s", actualValue, acceptedValues, actualValue)
		}
	}
	return nil
}

func stringInSlice(target string, slice []string, ignoreCase bool) bool {
	for _, v := range slice {
		if v == target || (ignoreCase && strings.EqualFold(v, target)) {
			return true
		}
	}
	return false
}

func convertToStringSlice(items []interface{}) []string {
	stringItems := make([]string, len(items))
	for i, item := range items {
		stringItems[i] = fmt.Sprint(item)
	}
	return stringItems
}

func clusterSettingsKeysValidate(v interface{}, path cty.Path) diag.Diagnostics {
	clusterSettingsMap := v.(map[string]interface{})

	if len(clusterSettingsMap) == 0 {
		return diag.Errorf("error creating / updating Cluster Config: %q block should not be empty", paramConfigs)
	}

	for clusterSetting := range clusterSettingsMap {
		if !stringInSlice(clusterSetting, editableClusterSettings, false) {
			return diag.Errorf("error creating / updating Cluster Config: %q cluster setting is read-only and cannot be updated. "+
				"Read %s for more details.", clusterSetting, docsClusterConfigUrl)
		}
	}
	return nil
}

func clusterLinkSettingsKeysValidate(v interface{}, path cty.Path) diag.Diagnostics {
	clusterSettingsMap := v.(map[string]interface{})

	for clusterSetting := range clusterSettingsMap {
		if !stringInSlice(clusterSetting, editableClusterLinkSettings, false) {
			return diag.Errorf("error creating / updating Cluster Link: %q cluster link setting is read-only and cannot be updated. "+
				"Read %s for more details.", clusterSetting, docsClusterLinkConfigUrl)
		}
	}
	return nil
}

// https://github.com/confluentinc/cli/blob/main/internal/connect/utils.go#L88C1-L125C2
func uploadFile(url, filePath string, formFields map[string]any) error {
	// TODO: figure out a way to mock this function and delete this hack
	if url == tfCustomConnectorPluginTestUrl {
		return nil
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	for key, value := range formFields {
		if strValue, ok := value.(string); ok {
			_ = writer.WriteField(key, strValue)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 20 * time.Minute,
	}
	_, err = sling.New().Client(client).Base(url).Set("Content-Type", writer.FormDataContentType()).Post("").Body(&buffer).ReceiveSuccess(nil)
	if err != nil {
		return err
	}

	return nil
}

func extractCloudAndRegionName(resourceId string) (string, string, error) {
	parts := strings.Split(resourceId, ".")
	cloud := ""
	regionName := ""
	if len(parts) == 3 {
		// old version of API Key Mgmt API
		cloud = parts[1]
		regionName = parts[2]
	} else if len(parts) == 2 {
		// new version of API Key Mgmt API
		cloud = parts[0]
		regionName = parts[1]
	} else {
		return "", "", fmt.Errorf("error extracting cloud and region name: invalid format: expected " +
			"'<cloud>.<region name>' or '<environment>.<cloud>.<region name>'")
	}

	return cloud, regionName, nil
}

func extractOrgIdFromResourceName(resourceName string) (string, error) {
	// Match any string of non-slash characters after organization= until the next slash or the end of the string.
	re := regexp.MustCompile(`/organization=([^/]+)(/|$)`)
	match := re.FindStringSubmatch(resourceName)
	if len(match) > 1 {
		return match[1], nil
	} else {
		return "", fmt.Errorf("could not find organization ID in %v: %s", paramResourceName, resourceName)
	}
}

func generateFlinkStatementName() string {
	clientName := "tf"
	date := time.Now().Format("2006-01-02")
	localTime := time.Now().Format("150405")
	id := uuid.New().String()
	return fmt.Sprintf("%s-%s-%s-%s", clientName, date, localTime, id)
}

func parseStatementName(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid ID format: expected '<Environment ID>/Compute Pool ID>/<Statement name>'")
	}
	return parts[2], nil
}

func canUpdateSchemaEntityType(oldEntityName, newEntityName string) bool {
	oldParts := strings.Split(oldEntityName, ":")
	newParts := strings.Split(newEntityName, ":")
	if 3 != len(oldParts) || 3 != len(newParts) {
		return false
	}
	if oldParts[0] != newParts[0] || oldParts[1] != newParts[1] {
		return false
	}
	return isNewSchemaIdGreaterThanOld(oldParts, newParts)
}

func canUpdateFieldOrRecordEntityType(oldEntityName, newEntityName string) bool {
	oldParts := strings.Split(oldEntityName, ":")
	newParts := strings.Split(newEntityName, ":")
	if 4 != len(oldParts) || 4 != len(newParts) {
		return false
	}
	if oldParts[0] != newParts[0] || oldParts[1] != newParts[1] || oldParts[3] != newParts[3] {
		return false
	}
	return isNewSchemaIdGreaterThanOld(oldParts, newParts)
}

func isNewSchemaIdGreaterThanOld(oldParts, newParts []string) bool {
	oldSchemaId, err := strconv.Atoi(oldParts[2])
	if err != nil {
		return false
	}
	newSchemaId, err := strconv.Atoi(newParts[2])
	if err != nil {
		return false
	}
	// Tags are propagated for new versions
	return newSchemaId > oldSchemaId
}

func canUpdateEntityName(entityType, oldEntityName, newEntityName string) bool {
	switch entityType {
	case schemaEntityType:
		return canUpdateSchemaEntityType(oldEntityName, newEntityName)
	case fieldEntityType, recordEntityType:
		return canUpdateFieldOrRecordEntityType(oldEntityName, newEntityName)
	default:
		return false
	}
}

func canUpdateEntityNameBusinessMetadata(entityType, oldEntityName, newEntityName string) bool {
	if oldEntityName == newEntityName {
		return true
	}
	switch entityType {
	case schemaEntityType:
		return canUpdateSchemaEntityType(oldEntityName, newEntityName)
	default:
		return false
	}
}
