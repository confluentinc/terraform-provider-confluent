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
	"encoding/json"
	"fmt"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const (
	paramResourceName = "resource_name"
	paramResourceType = "resource_type"
	paramPatternType  = "pattern_type"
	paramPrincipal    = "principal"
	paramHost         = "host"
	paramOperation    = "operation"
	paramPermission   = "permission"

	principalPrefix = "User:"
)

var acceptedResourceTypes = []string{"UNKNOWN", "ANY", "TOPIC", "GROUP", "CLUSTER", "TRANSACTIONAL_ID", "DELEGATION_TOKEN"}
var acceptedPatternTypes = []string{"UNKNOWN", "ANY", "MATCH", "LITERAL", "PREFIXED"}
var acceptedOperations = []string{"UNKNOWN", "ANY", "ALL", "READ", "WRITE", "CREATE", "DELETE", "ALTER", "DESCRIBE", "CLUSTER_ACTION", "DESCRIBE_CONFIGS", "ALTER_CONFIGS", "IDEMPOTENT_WRITE"}
var acceptedPermissions = []string{"UNKNOWN", "ANY", "DENY", "ALLOW"}

func extractAcl(d *schema.ResourceData) (Acl, error) {
	resourceType, err := stringToAclResourceType(d.Get(paramResourceType).(string))
	if err != nil {
		return Acl{}, err
	}
	return Acl{
		ResourceType: resourceType,
		ResourceName: d.Get(paramResourceName).(string),
		PatternType:  d.Get(paramPatternType).(string),
		Principal:    d.Get(paramPrincipal).(string),
		Host:         d.Get(paramHost).(string),
		Operation:    d.Get(paramOperation).(string),
		Permission:   d.Get(paramPermission).(string),
	}, nil
}

func kafkaAclResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaAclCreate,
		ReadContext:   kafkaAclRead,
		UpdateContext: kafkaAclUpdate,
		DeleteContext: kafkaAclDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaAclImport,
		},
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: optionalKafkaClusterBlockSchema(),
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
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The principal for the ACL.",
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
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    kafkaClusterBlockV0().CoreConfigSchema().ImpliedType(),
				Upgrade: kafkaClusterBlockStateUpgradeV0,
				Version: 0,
			},
			{
				Type:    kafkaAclResourceV1().CoreConfigSchema().ImpliedType(),
				Upgrade: kafkaStateUpgradeV0,
				Version: 1,
			},
		},
	}
}

func kafkaAclCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka ACLs: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	acl, err := extractAcl(d)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	createAclRequest := kafkarestv3.CreateAclRequestData{
		ResourceType: acl.ResourceType,
		ResourceName: acl.ResourceName,
		PatternType:  acl.PatternType,
		Principal:    acl.Principal,
		Host:         acl.Host,
		Operation:    acl.Operation,
		Permission:   acl.Permission,
	}
	createAclRequestJson, err := json.Marshal(createAclRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka ACLs: error marshaling %#v to json: %s", createAclRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka ACLs: %s", createAclRequestJson))

	_, err = executeKafkaAclCreate(ctx, kafkaRestClient, createAclRequest)

	if err != nil {
		return diag.Errorf("error creating Kafka ACLs: %s", createDescriptiveError(err))
	}
	kafkaAclId := createKafkaAclId(kafkaRestClient.clusterId, acl)
	d.SetId(kafkaAclId)

	// https://github.com/confluentinc/terraform-provider-confluentcloud/issues/40#issuecomment-1048782379
	SleepIfNotTestMode(kafkaRestAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	return kafkaAclRead(ctx, d, meta)
}

func executeKafkaAclCreate(ctx context.Context, c *KafkaRestClient, requestData kafkarestv3.CreateAclRequestData) (*http.Response, error) {
	return c.apiClient.ACLV3Api.CreateKafkaAcls(c.apiContext(ctx), c.clusterId).CreateAclRequestData(requestData).Execute()
}

func kafkaAclDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)

	acl, err := extractAcl(d)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	_, _, err = executeKafkaAclDelete(kafkaRestClient.apiContext(ctx), kafkaRestClient, acl)

	if err != nil {
		return diag.Errorf("error deleting Kafka ACLs %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	return nil
}

func executeKafkaAclDelete(ctx context.Context, c *KafkaRestClient, acl Acl) (kafkarestv3.InlineResponse200, *http.Response, error) {
	return c.apiClient.ACLV3Api.DeleteKafkaAcls(c.apiContext(ctx), c.clusterId).ResourceType(acl.ResourceType).ResourceName(acl.ResourceName).PatternType(acl.PatternType).Principal(acl.Principal).Host(acl.Host).Operation(acl.Operation).Permission(acl.Permission).Execute()
}

func executeKafkaAclRead(ctx context.Context, c *KafkaRestClient, acl Acl) (kafkarestv3.AclDataList, *http.Response, error) {
	return c.apiClient.ACLV3Api.GetKafkaAcls(c.apiContext(ctx), c.clusterId).ResourceType(acl.ResourceType).ResourceName(acl.ResourceName).PatternType(acl.PatternType).Principal(acl.Principal).Host(acl.Host).Operation(acl.Operation).Permission(acl.Permission).Execute()
}

func kafkaAclRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka ACLs: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka ACLs: %s", createDescriptiveError(err))
	}
	client := meta.(*Client)
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	acl, err := extractAcl(d)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	_, err = readAclAndSetAttributes(ctx, d, client, kafkaRestClient, acl)
	if err != nil {
		return diag.Errorf("error reading Kafka ACLs: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	return nil
}

func createKafkaAclId(clusterId string, acl Acl) string {
	return fmt.Sprintf("%s/%s", clusterId, strings.Join([]string{
		string(acl.ResourceType),
		acl.ResourceName,
		string(acl.PatternType),
		acl.Principal,
		acl.Host,
		string(acl.Operation),
		string(acl.Permission),
	}, "#"))
}

func readAclAndSetAttributes(ctx context.Context, d *schema.ResourceData, client *Client, c *KafkaRestClient, acl Acl) ([]*schema.ResourceData, error) {
	remoteAcls, resp, err := executeKafkaAclRead(ctx, c, acl)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka ACLs %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka ACLs %q in TF state because Kafka ACLs could not be found on the server", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	if len(remoteAcls.Data) == 0 {
		// Essentially len(data) = 0 means 404, so we should duplicate the code from the previous if statement
		if !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka ACLs %q in TF state because Kafka ACLs could not be found on the server", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}
		return nil, fmt.Errorf("error reading Kafka ACLs %q: no Kafka ACLs were matched", d.Id())
	} else if len(remoteAcls.Data) > 1 {
		// TODO: use remoteAcls.Data
		return nil, fmt.Errorf("error reading Kafka ACLs %q: multiple Kafka ACLs were matched", d.Id())
	}
	matchedAcl := remoteAcls.Data[0]
	matchedAclJson, err := json.Marshal(matchedAcl)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka ACLs: error marshaling %#v to json: %s", matchedAcl, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka ACLs %q: %s", d.Id(), matchedAclJson), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	if err := d.Set(paramResourceType, matchedAcl.ResourceType); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, matchedAcl.ResourceName); err != nil {
		return nil, err
	}
	if err := d.Set(paramPatternType, matchedAcl.PatternType); err != nil {
		return nil, err
	}
	// Use principal with resource ID
	if err := d.Set(paramPrincipal, acl.Principal); err != nil {
		return nil, err
	}
	if err := d.Set(paramHost, matchedAcl.Host); err != nil {
		return nil, err
	}
	if err := d.Set(paramOperation, matchedAcl.Operation); err != nil {
		return nil, err
	}
	if err := d.Set(paramPermission, matchedAcl.Permission); err != nil {
		return nil, err
	}
	if !c.isClusterIdSetInProviderBlock {
		if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, c.clusterId, d); err != nil {
			return nil, err
		}
	}
	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
	}
	d.SetId(createKafkaAclId(c.clusterId, acl))

	return []*schema.ResourceData{d}, nil
}

func kafkaAclImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Topic: %s", createDescriptiveError(err))
	}

	clusterIdAndSerializedAcl := d.Id()

	parts := strings.Split(clusterIdAndSerializedAcl, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Kafka ACLs: invalid format: expected '<Kafka cluster ID>/<resource type>#<resource name>#<pattern type>#<principal>#<host>#<operation>#<permission>'")
	}

	clusterId := parts[0]
	serializedAcl := parts[1]

	acl, err := deserializeAcl(serializedAcl)
	if err != nil {
		return nil, err
	}

	client := meta.(*Client)
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readAclAndSetAttributes(ctx, d, client, kafkaRestClient, acl); err != nil {
		return nil, fmt.Errorf("error importing Kafka ACLs %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka ACLs %q", d.Id()), map[string]interface{}{kafkaAclLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func deserializeAcl(serializedAcl string) (Acl, error) {
	parts := strings.Split(serializedAcl, "#")
	if len(parts) != 7 {
		return Acl{}, fmt.Errorf("invalid format for kafka ACL import: expected '<Kafka cluster ID>/<resource type>#<resource name>#<pattern type>#<principal>#<host>#<operation>#<permission>'")
	}

	resourceType, err := stringToAclResourceType(parts[0])
	if err != nil {
		return Acl{}, err
	}

	return Acl{
		ResourceType: resourceType,
		ResourceName: parts[1],
		PatternType:  parts[2],
		Principal:    parts[3],
		Host:         parts[4],
		Operation:    parts[5],
		Permission:   parts[6],
	}, nil
}

func kafkaAclUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials) {
		return diag.Errorf("error updating Kafka ACLs %q: only %q block can be updated for Kafka ACLs", d.Id(), paramCredentials)
	}
	return kafkaAclRead(ctx, d, meta)
}

func kafkaAclImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaAcls,
	}
}

func loadAllKafkaAcls(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	kafkaRestClient := client.kafkaRestClientFactory.CreateKafkaRestClient(client.kafkaRestEndpoint, client.kafkaClusterId, client.kafkaApiKey, client.kafkaApiSecret, true, true)

	acls, _, err := kafkaRestClient.apiClient.ACLV3Api.GetKafkaAcls(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()

	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka ACLs for Kafka Cluster %q: %s", kafkaRestClient.clusterId, createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err))
	}
	kafkaAclsJson, err := json.Marshal(acls)
	if err != nil {
		return nil, diag.Errorf("error reading Kafka ACLs for Kafka Cluster %q: error marshaling %#v to json: %s", kafkaRestClient.clusterId, acls, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka ACLs for Kafka Cluster %q: %s", kafkaRestClient.clusterId, kafkaAclsJson))

	// APIF-2038: Kafka REST API only accepts integer ID at the moment
	serviceAccounts, _, err := client.iamV1Client.ServiceAccountsV1Api.ListV1ServiceAccounts(client.iamV1ApiContext(ctx)).Execute()
	users, _, err := client.iamV1Client.UsersV1Api.ListV1Users(client.iamV1ApiContext(ctx)).Execute()

	principalIdMap := make(map[int32]string)

	for _, principal := range serviceAccounts.GetUsers() {
		principalIdMap[principal.GetId()] = principal.GetResourceId()
	}
	for _, principal := range users.GetUsers() {
		principalIdMap[principal.GetId()] = principal.GetResourceId()
	}

	for _, aclData := range acls.GetData() {
		principalWithResourceId, err := principalWithIntegerIdToPrincipalWithResourceId(principalIdMap, aclData.GetPrincipal())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("%s", createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
			continue
		}
		acl := Acl{
			ResourceType: aclData.GetResourceType(),
			ResourceName: aclData.GetResourceName(),
			PatternType:  aclData.GetPatternType(),
			Principal:    principalWithResourceId,
			Host:         aclData.GetHost(),
			Operation:    aclData.GetOperation(),
			Permission:   aclData.GetPermission(),
		}
		instanceId := createKafkaAclId(client.kafkaClusterId, acl)
		instances[instanceId] = toValidTerraformResourceName(createAclInstanceName(acl))
	}

	return instances, nil
}

func createAclInstanceName(acl Acl) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", acl.Permission, acl.Operation, acl.PatternType, acl.ResourceName, acl.ResourceType)
}

// APIF-2043: TEMPORARY METHOD
// Converts principal with an integer ID (User:6789) to principal with a resourceID (User:sa-01234)
func principalWithIntegerIdToPrincipalWithResourceId(principalIdMap map[int32]string, principalWithIntegerId string) (string, error) {
	// There's input validation that principal attribute must start with "User:sa-" or "User:u-" or "User:pool-" or "User:group-" or "User:*"

	if principalWithIntegerId == "User:*" || strings.HasPrefix(principalWithIntegerId, "User:sa-") || strings.HasPrefix(principalWithIntegerId, "User:u-") || strings.HasPrefix(principalWithIntegerId, "User:pool-") || strings.HasPrefix(principalWithIntegerId, "User:group-") {
		return principalWithIntegerId, nil
	}

	// User:12345 -> sa-12345
	intIdStr := principalWithIntegerId[5:]
	intId, err := strconv.Atoi(intIdStr)
	if err != nil {
		return "", fmt.Errorf("failed to convert int ID %s to int", intIdStr)
	}
	int32Id := int32(intId)

	if principalResourceId, ok := principalIdMap[int32Id]; ok {
		return fmt.Sprintf("User:%s", principalResourceId), nil
	}

	return "", fmt.Errorf("the matching resource ID for a principal with int ID=%d is nil", intId)
}
