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
	"time"
)

// https://docs.confluent.io/cloud/current/clusters/broker-config.html#change-cluster-settings-for-dedicated-clusters
var editableClusterSettings = []string{
	"auto.create.topics.enable",
	"ssl.cipher.suites",
	"num.partitions",
	"log.cleaner.max.compaction.lag.ms",
	"log.retention.ms",
}

const docsClusterConfigUrl = "https://docs.confluent.io/cloud/current/clusters/broker-config.html#change-cluster-settings-for-dedicated-clusters"

func kafkaConfigResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaConfigCreate,
		ReadContext:   kafkaConfigRead,
		UpdateContext: kafkaConfigUpdate,
		DeleteContext: kafkaConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaConfigImport,
		},
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: optionalKafkaClusterBlockSchema(),
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:         true,
				Description:      "The custom cluster settings to set (e.g., `\"num.partitions\" = \"8\"`).",
				ValidateDiagFunc: clusterSettingsKeysValidate,
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
	}
}

func kafkaConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Kafka Config: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
	configs := extractClusterConfigs(d.Get(paramConfigs).(map[string]interface{}))

	createConfigRequest := kafkarestv3.AlterConfigBatchRequestData{
		Data: configs,
	}
	createConfigRequestJson, err := json.Marshal(createConfigRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Config: error marshaling %#v to json: %s", createConfigRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka Config: %s", createConfigRequestJson))

	_, err = executeKafkaConfigCreate(ctx, kafkaRestClient, createConfigRequest)

	if err != nil {
		return diag.Errorf("error creating Kafka Config: %s", createDescriptiveError(err))
	}

	kafkaConfigId := createKafkaConfigId(kafkaRestClient.clusterId)
	d.SetId(kafkaConfigId)

	// https://github.com/confluentinc/terraform-provider-confluent/issues/40#issuecomment-1048782379
	time.Sleep(kafkaRestAPIWaitAfterCreate)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return kafkaConfigRead(ctx, d, meta)
}

func executeKafkaConfigCreate(ctx context.Context, c *KafkaRestClient, requestData kafkarestv3.AlterConfigBatchRequestData) (*http.Response, error) {
	return c.apiClient.ConfigsV3Api.UpdateKafkaClusterConfigs(c.apiContext(ctx), c.clusterId).AlterConfigBatchRequestData(requestData).Execute()
}

func kafkaConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: KCFUN-389
	// Call Reset Cluster Config method once KCFUN-389 is done. Currently kafkaConfigDelete() does nothing.
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return nil
}

func kafkaConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Config: %s", createDescriptiveError(err))
	}
	clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Kafka Config: %s", createDescriptiveError(err))
	}
	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)

	_, err = readConfigAndSetAttributes(ctx, d, kafkaRestClient)
	if err != nil {
		return diag.Errorf("error reading Kafka Config: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return nil
}

func createKafkaConfigId(clusterId string) string {
	return fmt.Sprintf("%s", clusterId)
}

func kafkaConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Config: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Config: %s", createDescriptiveError(err))
	}

	clusterId := d.Id()

	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readConfigAndSetAttributes(ctx, d, kafkaRestClient); err != nil {
		return nil, fmt.Errorf("error importing Kafka Config %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient) ([]*schema.ResourceData, error) {
	kafkaConfig, resp, err := c.apiClient.ConfigsV3Api.ListKafkaClusterConfigs(c.apiContext(ctx), c.clusterId).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Config %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka Config %q in TF state because Kafka Config could not be found on the server", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	kafkaConfigJson, err := json.Marshal(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Config %q: error marshaling %#v to json: %s", d.Id(), kafkaConfig, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Config %q: %s", d.Id(), kafkaConfigJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	if err := d.Set(paramConfigs, convertKafkaConfigToMap(kafkaConfig)); err != nil {
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

	d.SetId(createKafkaConfigId(c.clusterId))

	return []*schema.ResourceData{d}, nil
}

func kafkaConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramConfigs) {
		return diag.Errorf("error updating Kafka Config %q: only %q and %q blocks can be updated for Kafka Config", d.Id(), paramCredentials, paramConfigs)
	}
	if d.HasChange(paramConfigs) {
		// TF Provider allows the following operations for editable cluster settings under 'config' block:
		// 1. Adding new key value pair, for example, "retention.ms" = "600000"
		// 2. Update a value for existing key value pair, for example, "retention.ms" = "600000" -> "retention.ms" = "600001"
		// You might find the list of editable cluster settings and their limits at
		// https://docs.confluent.io/cloud/current/clusters/broker-config.html#change-cluster-settings-for-dedicated-clusters
		//https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster_config

		// Extract 'old' and 'new' (include changes in TF configuration) cluster settings
		// * 'old' cluster settings -- all cluster settings from TF configuration _before_ changes / updates (currently set on Confluent Cloud)
		// * 'new' cluster settings -- all cluster settings from TF configuration _after_ changes
		oldClusterSettingsMap, newClusterSettingsMap := extractOldAndNewSettings(d)

		// Verify that no cluster settings were removed (reset to its default value) in TF configuration which is an unsupported operation at the moment
		for oldSettingName := range oldClusterSettingsMap {
			if _, ok := newClusterSettingsMap[oldSettingName]; !ok {
				return diag.Errorf("error updating Kafka Config %q: reset to cluster setting's default value operation (in other words, removing cluster settings from %q block) "+
					"is not supported at the moment. "+
					"Instead, find its default value at %s and set its current value to the default value.", d.Id(), paramConfigs, docsUrl)
			}
		}

		// Construct a request for Kafka REST API
		_, newSettingsMapAny := d.GetChange(paramConfigs)
		updateConfigRequest := kafkarestv3.AlterConfigBatchRequestData{
			Data: extractClusterConfigs(newSettingsMapAny.(map[string]interface{})),
		}
		restEndpoint, err := extractRestEndpoint(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Config: %s", createDescriptiveError(err))
		}
		clusterId, err := extractKafkaClusterId(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Config: %s", createDescriptiveError(err))
		}
		clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, false)
		if err != nil {
			return diag.Errorf("error updating Kafka Config: %s", createDescriptiveError(err))
		}
		kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isKafkaMetadataSet, meta.(*Client).isKafkaClusterIdSet)
		updateConfigRequestJson, err := json.Marshal(updateConfigRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Config: error marshaling %#v to json: %s", updateConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Config %q: %s", d.Id(), updateConfigRequestJson), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})

		// Send a request to Kafka REST API
		_, err = executeKafkaConfigUpdate(ctx, kafkaRestClient, updateConfigRequest)
		if err != nil {
			// For example, Kafka REST API will return Bad Request if new cluster setting value exceeds the max limit:
			// 400 Bad Request: Config property 'delete.retention.ms' with value '63113904003' exceeded max limit of 60566400000.
			return diag.Errorf("error updating Kafka Config: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Config %q", d.Id()), map[string]interface{}{kafkaClusterConfigLoggingKey: d.Id()})
	}
	return kafkaConfigRead(ctx, d, meta)
}

func executeKafkaConfigUpdate(ctx context.Context, c *KafkaRestClient, requestData kafkarestv3.AlterConfigBatchRequestData) (*http.Response, error) {
	return c.apiClient.ConfigsV3Api.UpdateKafkaClusterConfigs(c.apiContext(ctx), c.clusterId).AlterConfigBatchRequestData(requestData).Execute()
}

func convertKafkaConfigToMap(clusterConfigList kafkarestv3.ClusterConfigDataList) map[string]string {
	config := make(map[string]string)
	for _, remoteConfig := range clusterConfigList.Data {
		if remoteConfig.Value.IsSet() {
			config[remoteConfig.Name] = *remoteConfig.Value.Get()
		}
	}
	return config
}

func extractClusterConfigs(configs map[string]interface{}) []kafkarestv3.AlterConfigBatchRequestDataData {
	configResult := make([]kafkarestv3.AlterConfigBatchRequestDataData, len(configs))

	i := 0
	for name, value := range configs {
		v := value.(string)
		configResult[i] = kafkarestv3.AlterConfigBatchRequestDataData{
			Name:  name,
			Value: *kafkarestv3.NewNullableString(&v),
		}
		i += 1
	}

	return configResult
}
