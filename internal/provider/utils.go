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
	quotas "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	ksql "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	org "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	sg "github.com/confluentinc/ccloud-sdk-go-v2/stream-governance/v2"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
)

const (
	crnKafkaSuffix                     = "/kafka="
	kafkaAclLoggingKey                 = "kafka_acl_id"
	kafkaClusterLoggingKey             = "kafka_cluster_id"
	kafkaClusterConfigLoggingKey       = "kafka_cluster_config_id"
	streamGovernanceClusterLoggingKey  = "stream_governance_cluster_id"
	kafkaTopicLoggingKey               = "kafka_topic_id"
	serviceAccountLoggingKey           = "service_account_id"
	userLoggingKey                     = "user_id"
	environmentLoggingKey              = "environment_id"
	roleBindingLoggingKey              = "role_binding_id"
	apiKeyLoggingKey                   = "api_key_id"
	networkLoggingKey                  = "network_key_id"
	connectorLoggingKey                = "connector_key_id"
	privateLinkAccessLoggingKey        = "private_link_access_id"
	peeringLoggingKey                  = "peering_id"
	transitGatewayAttachmentLoggingKey = "transit_gateway_attachment_id"
	ksqlClusterLoggingKey              = "ksql_cluster_id"
	identityProviderLoggingKey         = "identity_provider_id"
	identityPoolLoggingKey             = "identity_pool_id"
	clusterLinkLoggingKey              = "cluster_link_id"
	kafkaMirrorTopicLoggingKey         = "kafka_mirror_topic_id"
	kafkaClientQuotaLoggingKey         = "kafka_client_quota_id"
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

func (c *Client) sgApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), sg.ContextBasicAuth, sg.BasicAuth{
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
	apiClient                    *kafkarestv3.APIClient
	clusterId                    string
	clusterApiKey                string
	clusterApiSecret             string
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

// Creates retryable HTTP client that performs automatic retries with exponential backoff for 429
// and 5** (except 501) errors. Otherwise, the response is returned and left to the caller to interpret.
func createRetryableHttpClientWithExponentialBackoff() *http.Client {
	retryClient := retryablehttp.NewClient()

	// Implicitly using default retry configuration
	// under the assumption is it's OK to spend retrying a single HTTP call around 15 seconds in total: 1 + 2 + 4 + 8
	// An exponential backoff equation: https://github.com/hashicorp/go-retryablehttp/blob/master/client.go#L493
	// retryWaitMax = math.Pow(2, float64(attemptNum)) * float64(retryWaitMin)
	// defaultRetryWaitMin = 1 * time.Second
	// defaultRetryWaitMax = 30 * time.Second
	// defaultRetryMax     = 4

	return retryClient.StandardClient()
}

type KafkaRestClientFactory struct {
	userAgent string
}

type GenericOpenAPIError interface {
	Model() interface{}
}

func (f KafkaRestClientFactory) CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool) *KafkaRestClient {
	config := kafkarestv3.NewConfiguration()
	config.Servers[0].URL = restEndpoint
	config.UserAgent = f.userAgent
	config.HTTPClient = createRetryableHttpClientWithExponentialBackoff()
	return &KafkaRestClient{
		apiClient:                    kafkarestv3.NewAPIClient(config),
		clusterId:                    clusterId,
		clusterApiKey:                clusterApiKey,
		clusterApiSecret:             clusterApiSecret,
		restEndpoint:                 restEndpoint,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
	}
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
	// There's input validation that principal attribute must start with "User:sa-" or "User:u-"
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
	}
	return "", fmt.Errorf("the principal must start with 'User:sa-' or 'User:u-'")
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

	for clusterSetting, _ := range clusterSettingsMap {
		if !stringInSlice(clusterSetting, editableClusterSettings, false) {
			return diag.Errorf("error creating / updating Cluster Config: %q cluster setting is read-only and cannot be updated. "+
				"Read %s for more details.", clusterSetting, docsClusterConfigUrl)
		}
	}
	return nil
}
