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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	paramKafkaCluster           = "kafka_cluster"
	paramTopicName              = "topic_name"
	paramCredentials            = "credentials"
	paramPartitionsCount        = "partitions_count"
	paramKey                    = "key"
	paramSecret                 = "secret"
	paramConfigs                = "config"
	kafkaRestAPIWaitAfterCreate = 10 * time.Second
	docsUrl                     = "https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic"
	dynamicTopicConfig          = "DYNAMIC_TOPIC_CONFIG"
)

// https://docs.confluent.io/cloud/current/client-apps/topics/manage.html#ak-topic-configurations-for-all-ccloud-cluster-types
// https://docs.confluent.io/cloud/current/sr/broker-side-schema-validation.html#sv-configuration-options-on-a-topic
var editableTopicSettings = []string{"cleanup.policy", "delete.retention.ms", "max.message.bytes", "max.compaction.lag.ms",
	"message.timestamp.difference.max.ms", "message.timestamp.before.max.ms", "message.timestamp.after.max.ms",
	"message.timestamp.type", "min.compaction.lag.ms", "min.insync.replicas",
	"retention.bytes", "retention.ms", "segment.bytes", "segment.ms", "confluent.key.schema.validation", "confluent.value.schema.validation",
	"confluent.key.subject.name.strategy", "confluent.value.subject.name.strategy", "confluent.topic.type"}

func extractConfigs(configs map[string]interface{}) []kafkarestv3.CreateTopicRequestDataConfigs {
	configResult := make([]kafkarestv3.CreateTopicRequestDataConfigs, len(configs))

	i := 0
	for name, value := range configs {
		v := value.(string)
		configResult[i] = kafkarestv3.CreateTopicRequestDataConfigs{
			Name:  name,
			Value: *kafkarestv3.NewNullableString(&v),
		}
		i += 1
	}

	return configResult
}

func extractClusterApiKeyAndApiSecretFromCredentialsBlock(d *schema.ResourceData) (string, string) {
	clusterApiKey := extractStringValueFromBlock(d, paramCredentials, paramKey)
	clusterApiSecret := extractStringValueFromBlock(d, paramCredentials, paramSecret)
	return clusterApiKey, clusterApiSecret
}

func kafkaTopicResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaTopicCreate,
		ReadContext:   kafkaTopicRead,
		UpdateContext: kafkaTopicUpdate,
		DeleteContext: kafkaTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaTopicImport,
		},
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: optionalKafkaClusterBlockSchema(),
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
				Description:  "The number of partitions to create in the topic.",
				ValidateFunc: validation.IntAtLeast(1),
			},
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
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
		SchemaVersion: 2,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    kafkaClusterBlockV0().CoreConfigSchema().ImpliedType(),
				Upgrade: kafkaClusterBlockStateUpgradeV0,
				Version: 0,
			},
			{
				Type:    kafkaTopicResourceV1().CoreConfigSchema().ImpliedType(),
				Upgrade: kafkaStateUpgradeV0,
				Version: 1,
			},
		},
		CustomizeDiff: customdiff.All(
			customdiff.ForceNewIfChange(paramPartitionsCount, func(ctx context.Context, old, new, meta interface{}) bool {
				// "partition" can only increase in-place, so we must create a new resource
				// if it is decreased.
				return new.(int) < old.(int)
			}),
		),
	}
}

func extractKafkaClusterId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isKafkaClusterIdSet {
		return client.kafkaClusterId, nil
	}
	if isImportOperation {
		clusterId := getEnv("IMPORT_KAFKA_ID", "")
		if clusterId != "" {
			return clusterId, nil
		} else {
			return "", fmt.Errorf("one of provider.kafka_id (defaults to KAFKA_ID environment variable) or IMPORT_KAFKA_ID environment variable must be set")
		}
	}
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	if clusterId != "" {
		return clusterId, nil
	}
	return "", fmt.Errorf("one of provider.kafka_id (defaults to KAFKA_ID environment variable) or resource.kafka_cluster.id must be set")
}

func extractRestEndpoint(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isKafkaMetadataSet {
		return client.kafkaRestEndpoint, nil
	}
	if isImportOperation {
		restEndpoint := getEnv("IMPORT_KAFKA_REST_ENDPOINT", "")
		if restEndpoint != "" {
			return restEndpoint, nil
		} else {
			return "", fmt.Errorf("one of provider.kafka_rest_endpoint (defaults to KAFKA_REST_ENDPOINT environment variable) or IMPORT_KAFKA_REST_ENDPOINT environment variable must be set")
		}
	}
	restEndpoint := d.Get(paramRestEndpoint).(string)
	if restEndpoint != "" {
		return restEndpoint, nil
	}
	return "", fmt.Errorf("one of provider.kafka_rest_endpoint (defaults to KAFKA_REST_ENDPOINT environment variable) or resource.rest_endpoint must be set")
}

func extractClusterApiKeyAndApiSecret(client *Client, d *schema.ResourceData, isImportOperation bool) (string, string, error) {
	if client.isKafkaMetadataSet {
		return client.kafkaApiKey, client.kafkaApiSecret, nil
	}
	if isImportOperation {
		clusterApiKey := getEnv("IMPORT_KAFKA_API_KEY", "")
		clusterApiSecret := getEnv("IMPORT_KAFKA_API_SECRET", "")
		if clusterApiKey != "" && clusterApiSecret != "" {
			return clusterApiKey, clusterApiSecret, nil
		} else {
			return "", "", fmt.Errorf("one of (provider.kafka_api_key, provider.kafka_api_secret), (KAFKA_API_KEY, KAFKA_API_SECRET environment variables) or (IMPORT_KAFKA_API_KEY, IMPORT_KAFKA_API_SECRET environment variables) must be set")
		}
	}
	clusterApiKey, clusterApiSecret := extractClusterApiKeyAndApiSecretFromCredentialsBlock(d)
	if clusterApiKey != "" {
		return clusterApiKey, clusterApiSecret, nil
	}
	return "", "", fmt.Errorf("one of (provider.kafka_api_key, provider.kafka_api_secret), (KAFKA_API_KEY, KAFKA_API_SECRET environment variables) or (resource.credentials.key, resource.credentials.secret) must be set")
}

func kafkaTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Topic: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	topicName := d.Get(paramTopicName).(string)
	partitionsCountInt32 := int32(d.Get(paramPartitionsCount).(int))
	configs := extractConfigs(d.Get(paramConfigs).(map[string]interface{}))

	createTopicRequest := kafkarestv3.CreateTopicRequestData{
		TopicName:       topicName,
		PartitionsCount: &partitionsCountInt32,
		Configs:         &configs,
	}
	createTopicRequestJson, err := json.Marshal(createTopicRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Topic: error marshaling %#v to json: %s", createTopicRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka Topic: %s", createTopicRequestJson))

	createdKafkaTopic, _, err := executeKafkaTopicCreate(ctx, kafkaRestClient, createTopicRequest)

	if err != nil {
		return diag.Errorf("error creating Kafka Topic: %s", createDescriptiveError(err))
	}

	kafkaTopicId := createKafkaTopicId(kafkaRestClient.clusterId, topicName)
	d.SetId(kafkaTopicId)

	// https://github.com/confluentinc/terraform-provider-confluentcloud/issues/40#issuecomment-1048782379
	SleepIfNotTestMode(kafkaRestAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)

	createdKafkaTopicJson, err := json.Marshal(createdKafkaTopic)
	if err != nil {
		return diag.Errorf("error creating Kafka Topic: error marshaling %#v to json: %s", createdKafkaTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka Topic %q: %s", d.Id(), createdKafkaTopicJson), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	return kafkaTopicRead(ctx, d, meta)
}

func executeKafkaTopicCreate(ctx context.Context, c *KafkaRestClient, requestData kafkarestv3.CreateTopicRequestData) (kafkarestv3.TopicData, *http.Response, error) {
	return c.apiClient.TopicV3Api.CreateKafkaTopic(c.apiContext(ctx), c.clusterId).CreateTopicRequestData(requestData).Execute()
}

func kafkaTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Kafka Topic: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	topicName := d.Get(paramTopicName).(string)

	_, err = kafkaRestClient.apiClient.TopicV3Api.DeleteKafkaTopic(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, topicName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Kafka Topic %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForKafkaTopicToBeDeleted(kafkaRestClient.apiContext(ctx), kafkaRestClient, topicName, meta.(*Client).isAcceptanceTestMode); err != nil {
		return diag.Errorf("error waiting for Kafka Topic %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	return nil
}

func kafkaTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	topicName := d.Get(paramTopicName).(string)

	_, err = readTopicAndSetAttributes(ctx, d, kafkaRestClient, topicName)
	if err != nil {
		return diag.Errorf("error reading Kafka Topic: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	return nil
}

func createKafkaTopicId(clusterId, topicName string) string {
	return fmt.Sprintf("%s/%s", clusterId, topicName)
}

func credentialsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "The Cluster API Credentials.",
		MinItems:    1,
		MaxItems:    1,
		Sensitive:   true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramKey: {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "The Cluster API Key for your Confluent Cloud cluster.",
					Sensitive:    true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramSecret: {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "The Cluster API Secret for your Confluent Cloud cluster.",
					Sensitive:    true,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
	}
}

func requiredKafkaClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The Kafka cluster ID (e.g., `lkc-12345`).",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^lkc-"), "the Kafka cluster ID must be of the form 'lkc-'"),
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func optionalKafkaClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The Kafka cluster ID (e.g., `lkc-12345`).",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^lkc-"), "the Kafka cluster ID must be of the form 'lkc-'"),
				},
			},
		},
		Optional: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func kafkaClusterIdSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ForceNew:     true,
		Description:  "The Kafka cluster ID (e.g., `lkc-12345`).",
		ValidateFunc: validation.StringMatch(regexp.MustCompile("^lkc-"), "the Kafka cluster ID must be of the form 'lkc-'"),
	}
}

func kafkaTopicImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Topic: %s", createDescriptiveError(err))
	}

	clusterIDAndTopicName := d.Id()
	parts := strings.Split(clusterIDAndTopicName, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Kafka Topic: invalid format: expected '<Kafka cluster ID>/<topic name>'")
	}

	clusterId := parts[0]
	topicName := parts[1]

	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readTopicAndSetAttributes(ctx, d, kafkaRestClient, topicName); err != nil {
		return nil, fmt.Errorf("error importing Kafka Topic %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka Topic %q", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readTopicAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, topicName string) ([]*schema.ResourceData, error) {
	kafkaTopic, resp, err := c.apiClient.TopicV3Api.GetKafkaTopic(c.apiContext(ctx), c.clusterId, topicName).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Topic %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka Topic %q in TF state because Kafka Topic could not be found on the server", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	kafkaTopicJson, err := json.Marshal(kafkaTopic)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Topic %q: error marshaling %#v to json: %s", d.Id(), kafkaTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Topic %q: %s", d.Id(), kafkaTopicJson), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

	if err := d.Set(paramTopicName, kafkaTopic.TopicName); err != nil {
		return nil, err
	}
	if err := d.Set(paramPartitionsCount, kafkaTopic.PartitionsCount); err != nil {
		return nil, err
	}

	configs, err := loadTopicConfigs(ctx, d, c, topicName)
	if err != nil {
		return nil, err
	}
	if err := d.Set(paramConfigs, configs); err != nil {
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

	d.SetId(createKafkaTopicId(c.clusterId, topicName))

	return []*schema.ResourceData{d}, nil
}

func kafkaTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramConfigs, paramPartitionsCount) {
		return diag.Errorf("error updating Kafka Topic %q: only %q, %q and %q blocks can be updated for Kafka Topic", d.Id(), paramCredentials, paramConfigs, paramPartitionsCount)
	}
	if d.HasChange(paramPartitionsCount) {
		oldPartitionsCount, newPartitionsCount := d.GetChange(paramPartitionsCount)
		oldPartitionsCountInt32 := int32(oldPartitionsCount.(int))
		newPartitionsCountInt32 := int32(newPartitionsCount.(int))
		// Construct a request for Kafka REST API
		updateTopicRequest := kafkarestv3.UpdatePartitionCountRequestData{
			PartitionsCount: newPartitionsCountInt32,
		}
		restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
		topicName := d.Get(paramTopicName).(string)
		updateTopicRequestJson, err := json.Marshal(updateTopicRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: error marshaling %#v to json: %s", updateTopicRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Topic %q: %s", d.Id(), updateTopicRequestJson), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

		// Send a request to Kafka REST API
		_, _, err = executeKafkaTopicPartitionsCountUpdate(ctx, kafkaRestClient, topicName, updateTopicRequest)
		if err != nil {
			// For example, Kafka REST API will return Bad Request if new partitions count is not bigger than the current one:
			// 400 Bad Request: Topic currently has 6 partitions, which is higher than the requested 2.

			// At this point new partitions count is saved to TF file,
			// so we need to revert it to the old value to avoid TF drift.
			if err := d.Set(paramPartitionsCount, oldPartitionsCountInt32); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return diag.FromErr(createDescriptiveError(err))
		}
		// Give some time to Kafka REST API to apply an update of partitions count
		SleepIfNotTestMode(kafkaRestAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Topic %q: topic settings update has been completed", d.Id()), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})
	}
	if d.HasChange(paramConfigs) {
		// TF Provider allows the following operations for editable topic settings under 'config' block:
		// 1. Adding new key value pair, for example, "retention.ms" = "600000"
		// 2. Update a value for existing key value pair, for example, "retention.ms" = "600000" -> "retention.ms" = "600001"
		// You might find the list of editable topic settings and their limits at
		// https://docs.confluent.io/cloud/current/client-apps/topics/manage.html#ak-topic-configurations-for-all-ccloud-cluster-types

		// Extract 'old' and 'new' (include changes in TF configuration) topic settings
		// * 'old' topic settings -- all topic settings from TF configuration _before_ changes / updates (currently set on Confluent Cloud)
		// * 'new' topic settings -- all topic settings from TF configuration _after_ changes
		oldTopicSettingsMap, newTopicSettingsMap := extractOldAndNewSettings(d)

		// Verify that no topic settings were removed (reset to its default value) in TF configuration which is an unsupported operation at the moment
		for oldTopicSettingName := range oldTopicSettingsMap {
			if _, ok := newTopicSettingsMap[oldTopicSettingName]; !ok {
				return diag.Errorf("error updating Kafka Topic %q: reset to topic setting's default value operation (in other words, removing topic settings from 'configs' block) "+
					"is not supported at the moment. "+
					"Instead, find its default value at %s and set its current value to the default value.", d.Id(), docsUrl)
			}
		}

		// Store only topic settings that were updated in TF configuration.
		// Will be used for creating a request to Kafka REST API.
		var topicSettingsUpdateBatch []kafkarestv3.AlterConfigBatchRequestDataData

		// Verify that topics that were changed in TF configuration settings are indeed editable
		for topicSettingName, newTopicSettingValue := range newTopicSettingsMap {
			oldTopicSettingValue, ok := oldTopicSettingsMap[topicSettingName]
			isTopicSettingValueUpdated := !(ok && oldTopicSettingValue == newTopicSettingValue)
			if isTopicSettingValueUpdated {
				// operation #1 (ok = False) or operation #2 (ok = True, oldTopicSettingValue != newTopicSettingValue)
				isTopicSettingEditable := stringInSlice(topicSettingName, editableTopicSettings, false)
				if isTopicSettingEditable {
					topicSettingsUpdateBatch = append(topicSettingsUpdateBatch, kafkarestv3.AlterConfigBatchRequestDataData{
						Name:  topicSettingName,
						Value: *kafkarestv3.NewNullableString(ptr(newTopicSettingValue)),
					})
				} else {
					return diag.Errorf("error updating Kafka Topic %q: %q topic setting is read-only and cannot be updated. "+
						"Read %s for more details.", d.Id(), topicSettingName, docsUrl)
				}
			}
		}

		// Construct a request for Kafka REST API
		updateTopicRequest := kafkarestv3.AlterConfigBatchRequestData{
			Data: topicSettingsUpdateBatch,
		}
		restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: %s", createDescriptiveError(err))
		}
		kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
		topicName := d.Get(paramTopicName).(string)
		updateTopicRequestJson, err := json.Marshal(updateTopicRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: error marshaling %#v to json: %s", updateTopicRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Topic %q: %s", d.Id(), updateTopicRequestJson), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})

		// Send a request to Kafka REST API
		_, err = executeKafkaTopicUpdate(ctx, kafkaRestClient, topicName, updateTopicRequest)
		if err != nil {
			// For example, Kafka REST API will return Bad Request if new topic setting value exceeds the max limit:
			// 400 Bad Request: Config property 'delete.retention.ms' with value '63113904003' exceeded max limit of 60566400000.
			return diag.FromErr(createDescriptiveError(err))
		}
		// Give some time to Kafka REST API to apply an update of topic settings
		SleepIfNotTestMode(kafkaRestAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)

		// Check that topic configs update was successfully executed
		// In other words, remote topic setting values returned by Kafka REST API match topic setting values from updated TF configuration
		actualTopicSettings, err := loadTopicConfigs(ctx, d, kafkaRestClient, topicName)
		if err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}

		var updatedTopicSettings, outdatedTopicSettings []string
		for _, v := range topicSettingsUpdateBatch {
			if !v.Value.IsSet() {
				// It will never happen because of the way we construct topicSettingsUpdateBatch
				continue
			}
			topicSettingName := v.Name
			expectedValue := *v.Value.Get()
			actualValue, ok := actualTopicSettings[topicSettingName]
			if ok && actualValue != expectedValue {
				outdatedTopicSettings = append(outdatedTopicSettings, topicSettingName)
			} else {
				updatedTopicSettings = append(updatedTopicSettings, topicSettingName)
			}
		}
		if len(outdatedTopicSettings) > 0 {
			diag.Errorf("error updating Kafka Topic %q: topic settings update failed for %#v. "+
				"Double check that these topic settings are indeed editable and provided target values do not exceed min/max allowed values by reading %s", d.Id(), outdatedTopicSettings, docsUrl)
		}
		updatedTopicSettingsJson, err := json.Marshal(updatedTopicSettings)
		if err != nil {
			return diag.Errorf("error updating Kafka Topic: error marshaling %#v to json: %s", updatedTopicSettings, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Topic %q: topic settings update has been completed for %s", d.Id(), updatedTopicSettingsJson), map[string]interface{}{kafkaTopicLoggingKey: d.Id()})
	}
	return nil
}

func executeKafkaTopicUpdate(ctx context.Context, c *KafkaRestClient, topicName string, requestData kafkarestv3.AlterConfigBatchRequestData) (*http.Response, error) {
	return c.apiClient.ConfigsV3Api.UpdateKafkaTopicConfigBatch(c.apiContext(ctx), c.clusterId, topicName).AlterConfigBatchRequestData(requestData).Execute()
}

func executeKafkaTopicPartitionsCountUpdate(ctx context.Context, c *KafkaRestClient, topicName string, requestData kafkarestv3.UpdatePartitionCountRequestData) (kafkarestv3.TopicData, *http.Response, error) {
	return c.apiClient.TopicV3Api.UpdatePartitionCountKafkaTopic(c.apiContext(ctx), c.clusterId, topicName).UpdatePartitionCountRequestData(requestData).Execute()
}

func setKafkaCredentials(kafkaApiKey, kafkaApiSecret string, d *schema.ResourceData) error {
	return d.Set(paramCredentials, []interface{}{map[string]interface{}{
		paramKey:    kafkaApiKey,
		paramSecret: kafkaApiSecret,
	}})
}

func loadTopicConfigs(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, topicName string) (map[string]string, error) {
	topicConfigList, _, err := c.apiClient.ConfigsV3Api.ListKafkaTopicConfigs(c.apiContext(ctx), c.clusterId, topicName).Execute()
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Topic %q: could not load configs %s", topicName, createDescriptiveError(err))
	}

	config := make(map[string]string)
	for _, remoteConfig := range topicConfigList.Data {
		// Extract configs that were set via terraform vs set by default
		if remoteConfig.Source == dynamicTopicConfig && remoteConfig.Value.IsSet() {
			config[remoteConfig.Name] = *remoteConfig.Value.Get()
		}
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Topic: error marshaling %#v to json: %s", config, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Topic %q Settings: %s", d.Id(), configJson), map[string]interface{}{"kafka_acl_id": d.Id()})

	return config, nil
}

func extractOldAndNewSettings(d *schema.ResourceData) (map[string]string, map[string]string) {
	oldConfigs, newConfigs := d.GetChange(paramConfigs)
	return convertToStringStringMap(oldConfigs.(map[string]interface{})), convertToStringStringMap(newConfigs.(map[string]interface{}))
}

// TODO: we might want to load all the resources instead
func kafkaTopicImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaTopics,
	}
}

func loadAllKafkaTopics(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	kafkaRestClient := client.kafkaRestClientFactory.CreateKafkaRestClient(client.kafkaRestEndpoint, client.kafkaClusterId, client.kafkaApiKey, client.kafkaApiSecret, true, true)

	topics, _, err := kafkaRestClient.apiClient.TopicV3Api.ListKafkaTopics(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Topics for Kafka Cluster %q: %s", kafkaRestClient.clusterId, createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err))
	}
	topicsJson, err := json.Marshal(topics)
	if err != nil {
		return nil, diag.Errorf("error reading Kafka Topics for Kafka Cluster %q: error marshaling %#v to json: %s", kafkaRestClient.clusterId, topics, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Topics for Kafka Cluster %q: %s", kafkaRestClient.clusterId, topicsJson), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})

	for _, topic := range topics.GetData() {
		if shouldFilterOutTopic(topic.GetTopicName()) {
			continue
		}
		instanceId := createKafkaTopicId(kafkaRestClient.clusterId, topic.GetTopicName())
		instances[instanceId] = toValidTerraformResourceName(topic.GetTopicName())
	}

	return instances, nil
}

var additionalInternalTopics = []string{"_schemas", "__consumer_offsets", "confluent-audit-log-events"}
var additionalInternalKsqlTopicPattern = regexp.MustCompile(`pksqlc-[a-zA-Z0-9]*-processing-log`)
var additionalInternalConnectTopicPattern = regexp.MustCompile(`dlq-lcc-[a-zA-Z0-9]*`)

func shouldFilterOutTopic(topicName string) bool {
	if stringInSlice(topicName, additionalInternalTopics, false) {
		return true
	}
	if additionalInternalKsqlTopicPattern.MatchString(topicName) || additionalInternalConnectTopicPattern.MatchString(topicName) {
		return true
	}
	return false
}
