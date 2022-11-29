// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	v3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	paramClusterLink                 = "cluster_link"
	paramMirrorTopicName             = "mirror_topic_name"
	paramSourceKafkaTopic            = "source_kafka_topic"
	stateActive                      = "ACTIVE"
	stateStopped                     = "STOPPED"
	stateFailedOver                  = "FAILED_OVER"
	statePromoted                    = "PROMOTED"
	paramKafkaMirrorTopicCredentials = "kafka_cluster.0.credentials"
)

var acceptedKafkaMirrorTopicStates = []string{stateActive, statePaused, stateFailedOver, statePromoted}
var acceptedStoppedKafkaMirrorTopicStates = []string{stateStopped, stateFailedOver, statePromoted}

var disallowedTransitionErrorMessage = "the following list of transitions is supported:" +
	fmt.Sprintf("\n* %q -> %q", stateActive, statePaused) +
	fmt.Sprintf("\n* %q -> %q", statePaused, stateActive) +
	fmt.Sprintf("\n* %q -> %q", stateActive, statePromoted) +
	fmt.Sprintf("\n* %q -> %q", stateActive, stateFailedOver) +
	fmt.Sprintf("\n* %q -> %q", statePaused, statePromoted) +
	fmt.Sprintf("\n* %q -> %q", statePaused, stateFailedOver)

func kafkaMirrorTopicResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaMirrorTopicCreate,
		ReadContext:   kafkaMirrorTopicRead,
		UpdateContext: kafkaMirrorTopicUpdate,
		DeleteContext: kafkaMirrorTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaMirrorTopicImport,
		},
		Schema: map[string]*schema.Schema{
			paramKafkaCluster:     mirrorTopicKafkaClusterBlockSchema(),
			paramClusterLink:      clusterLinkBlockSchema(),
			paramSourceKafkaTopic: sourceKafkaTopicBlockSchema(),
			paramMirrorTopicName: {
				Type:        schema.TypeString,
				Description: "Name of the topic to be mirrored over the Kafka Mirror Topic, i.e. the source topic's name. Only required when there is a prefix configured on the link.",
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			paramStatus: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedKafkaMirrorTopicStates, false),
				// Suppress the diff for terminal statuses: FAILED_OVER, PROMOTED, and STOPPED
				// for "failovered" and "promoted" topics since API returns "STOPPED" status for both
				// but TF uses FAILED_OVER and PROMOTED to allow "failover" and "promote" actions.
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if stringInSlice(old, acceptedStoppedKafkaMirrorTopicStates, false) &&
						stringInSlice(new, acceptedStoppedKafkaMirrorTopicStates, false) {
						return true
					}
					return false
				},
			},
		},
	}
}

func kafkaMirrorTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	kafkaRestClient, err := createKafkaRestClientFromKafkaBlock(d, meta)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	linkName := extractStringValueFromBlock(d, paramClusterLink, paramLinkName)
	sourceTopicName := extractStringValueFromBlock(d, paramSourceKafkaTopic, paramTopicName)
	mirrorTopicName := d.Get(paramMirrorTopicName).(string)

	createKafkaMirrorTopicRequest, err := constructKafkaMirrorTopicRequest(sourceTopicName, mirrorTopicName)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	createKafkaMirrorTopicRequestJson, err := json.Marshal(createKafkaMirrorTopicRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: error marshaling %#v to json: %s", createKafkaMirrorTopicRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka Mirror Topic: %s", createKafkaMirrorTopicRequestJson))

	_, err = executeKafkaMirrorTopicCreate(ctx, kafkaRestClient, createKafkaMirrorTopicRequest, linkName)

	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}

	kafkaMirrorTopicId := createKafkaMirrorTopicId(kafkaRestClient.clusterId, linkName, mirrorTopicName)
	d.SetId(kafkaMirrorTopicId)

	// https://github.com/confluentinc/terraform-provider-confluent/issues/40#issuecomment-1048782379
	time.Sleep(kafkaRestAPIWaitAfterCreate)

	// Don't log created Kafka Mirror Topic since API returns an empty 201 response.
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	return kafkaMirrorTopicRead(ctx, d, meta)
}

func kafkaMirrorTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	kafkaRestClient, err := createKafkaRestClientFromKafkaBlock(d, meta)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	linkName := extractStringValueFromBlock(d, paramClusterLink, paramLinkName)
	sourceTopicName := extractStringValueFromBlock(d, paramSourceKafkaTopic, paramTopicName)
	mirrorTopicName := d.Get(paramMirrorTopicName).(string)
	if mirrorTopicName == "" {
		mirrorTopicName = sourceTopicName
	}

	_, err = readKafkaMirrorTopicAndSetAttributes(ctx, d, kafkaRestClient, linkName, mirrorTopicName)
	if err != nil {
		return diag.Errorf("error reading Kafka Mirror Topic: %s", createDescriptiveError(err))
	}

	return nil
}

func readKafkaMirrorTopicAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, linkName, mirrorTopicName string) ([]*schema.ResourceData, error) {
	kafkaMirrorTopic, resp, err := c.apiClient.ClusterLinkingV3Api.ReadKafkaMirrorTopic(c.apiContext(ctx), c.clusterId, linkName, mirrorTopicName).Execute()

	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka Mirror Topic %q in TF state because Kafka Mirror Topic could not be found on the server", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	kafkaMirrorTopicJson, err := json.Marshal(kafkaMirrorTopic)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Mirror Topic %q: error marshaling %#v to json: %s", d.Id(), kafkaMirrorTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Mirror Topic %q: %s", d.Id(), kafkaMirrorTopicJson), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	if _, err := setKafkaMirrorTopicAttributes(d, c, kafkaMirrorTopic); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setKafkaMirrorTopicAttributes(d *schema.ResourceData, c *KafkaRestClient, kafkaMirrorTopic v3.ListMirrorTopicsResponseData) (*schema.ResourceData, error) {
	if err := d.Set(paramKafkaCluster, []interface{}{map[string]interface{}{
		paramId:           c.clusterId,
		paramRestEndpoint: c.restEndpoint,
		paramCredentials: []interface{}{map[string]interface{}{
			paramKey:    c.clusterApiKey,
			paramSecret: c.clusterApiSecret,
		}},
	}}); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramClusterLink, paramLinkName, kafkaMirrorTopic.GetLinkName(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramSourceKafkaTopic, paramTopicName, kafkaMirrorTopic.GetSourceTopicName(), d); err != nil {
		return nil, err
	}

	if err := d.Set(paramMirrorTopicName, kafkaMirrorTopic.GetMirrorTopicName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramStatus, kafkaMirrorTopic.GetMirrorStatus()); err != nil {
		return nil, err
	}
	d.SetId(createKafkaMirrorTopicId(c.clusterId, kafkaMirrorTopic.GetLinkName(), kafkaMirrorTopic.GetMirrorTopicName()))
	return d, nil
}

func kafkaMirrorTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	kafkaRestClient, err := createKafkaRestClientFromKafkaBlock(d, meta)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	linkName := extractStringValueFromBlock(d, paramClusterLink, paramLinkName)
	sourceTopicName := extractStringValueFromBlock(d, paramSourceKafkaTopic, paramTopicName)
	mirrorTopicName := d.Get(paramMirrorTopicName).(string)
	if mirrorTopicName == "" {
		mirrorTopicName = sourceTopicName
	}

	_, err = kafkaRestClient.apiClient.TopicV3Api.DeleteKafkaTopic(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, mirrorTopicName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForKafkaMirrorTopicToBeDeleted(kafkaRestClient.apiContext(ctx), kafkaRestClient, linkName, mirrorTopicName); err != nil {
		return diag.Errorf("error waiting for Kafka Mirror Topic %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	return nil
}

func kafkaMirrorTopicImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

	restEndpoint, err := extractRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Kafka Mirror Topic: %s", createDescriptiveError(err))
	}

	clusterIdAndLinkNameAndMirrorTopicName := d.Id()
	parts := strings.Split(clusterIdAndLinkNameAndMirrorTopicName, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Kafka Mirror Topic: invalid format: expected '<Kafka cluster ID>/<link name>/<Kafka mirror topic name>'")
	}

	clusterId := parts[0]
	linkName := parts[1]
	mirrorTopicName := parts[2]

	kafkaRestClient := meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, false)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readKafkaMirrorTopicAndSetAttributes(ctx, d, kafkaRestClient, linkName, mirrorTopicName); err != nil {
		return nil, fmt.Errorf("error importing Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func kafkaMirrorTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramKafkaCluster, paramKafkaMirrorTopicCredentials, paramStatus) {
		return diag.Errorf("error updating Kafka Mirror Topic %q: only %q and %q attributes can be updated for Kafka Mirror Topic", d.Id(), paramKafkaMirrorTopicCredentials, paramStatus)
	}
	kafkaRestClient, err := createKafkaRestClientFromKafkaBlock(d, meta)
	if err != nil {
		return diag.Errorf("error creating Kafka Mirror Topic: %s", createDescriptiveError(err))
	}
	linkName := extractStringValueFromBlock(d, paramClusterLink, paramLinkName)
	sourceTopicName := extractStringValueFromBlock(d, paramSourceKafkaTopic, paramTopicName)
	mirrorTopicName := d.Get(paramMirrorTopicName).(string)
	if mirrorTopicName == "" {
		mirrorTopicName = sourceTopicName
	}
	if d.HasChange(paramStatus) {
		oldValue, newValue := d.GetChange(paramStatus)
		oldStatus := oldValue.(string)
		newStatus := newValue.(string)
		shouldPauseKafkaMirrorTopic := (oldStatus == stateActive) && (newStatus == statePaused)
		shouldResumeKafkaMirrorTopic := (oldStatus == statePaused) && (newStatus == stateActive)
		shouldFailoverKafkaMirrorTopic := (oldStatus == stateActive || oldStatus == statePaused) && (newStatus == stateFailedOver)
		shouldPromoteKafkaMirrorTopic := (oldStatus == stateActive || oldStatus == statePaused) && (newStatus == statePromoted)
		if shouldPauseKafkaMirrorTopic {
			tflog.Debug(ctx, fmt.Sprintf("Pausing Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

			requestData := v3.AlterMirrorsRequestData{
				MirrorTopicNames: []string{mirrorTopicName},
			}
			_, _, err := kafkaRestClient.apiClient.ClusterLinkingV3Api.UpdateKafkaMirrorTopicsPause(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, linkName).AlterMirrorsRequestData(requestData).Execute()
			if err != nil {
				return diag.Errorf("error updating Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForKafkaMirrorTopicToChangeStatus(kafkaRestClient.apiContext(ctx), kafkaRestClient, kafkaRestClient.clusterId, linkName, mirrorTopicName, stateActive, statePaused); err != nil {
				return diag.Errorf("error waiting for Kafka Mirror Topic %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else if shouldResumeKafkaMirrorTopic {
			tflog.Debug(ctx, fmt.Sprintf("Resuming Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

			requestData := v3.AlterMirrorsRequestData{
				MirrorTopicNames: []string{mirrorTopicName},
			}
			_, _, err := kafkaRestClient.apiClient.ClusterLinkingV3Api.UpdateKafkaMirrorTopicsResume(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, linkName).AlterMirrorsRequestData(requestData).Execute()
			if err != nil {
				return diag.Errorf("error updating Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForKafkaMirrorTopicToChangeStatus(kafkaRestClient.apiContext(ctx), kafkaRestClient, kafkaRestClient.clusterId, linkName, mirrorTopicName, statePaused, stateActive); err != nil {
				return diag.Errorf("error waiting for Kafka Mirror Topic %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else if shouldFailoverKafkaMirrorTopic {
			tflog.Debug(ctx, fmt.Sprintf("Running a failover of Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

			requestData := v3.AlterMirrorsRequestData{
				MirrorTopicNames: []string{mirrorTopicName},
			}
			_, _, err := kafkaRestClient.apiClient.ClusterLinkingV3Api.UpdateKafkaMirrorTopicsFailover(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, linkName).AlterMirrorsRequestData(requestData).Execute()
			if err != nil {
				return diag.Errorf("error updating Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForKafkaMirrorTopicToChangeStatus(kafkaRestClient.apiContext(ctx), kafkaRestClient, kafkaRestClient.clusterId, linkName, mirrorTopicName, oldStatus, stateStopped); err != nil {
				return diag.Errorf("error waiting for Kafka Mirror Topic %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else if shouldPromoteKafkaMirrorTopic {
			tflog.Debug(ctx, fmt.Sprintf("Running a promote of Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})

			requestData := v3.AlterMirrorsRequestData{
				MirrorTopicNames: []string{mirrorTopicName},
			}
			_, _, err := kafkaRestClient.apiClient.ClusterLinkingV3Api.UpdateKafkaMirrorTopicsPromote(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, linkName).AlterMirrorsRequestData(requestData).Execute()
			if err != nil {
				return diag.Errorf("error updating Kafka Mirror Topic %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForKafkaMirrorTopicToChangeStatus(kafkaRestClient.apiContext(ctx), kafkaRestClient, kafkaRestClient.clusterId, linkName, mirrorTopicName, oldStatus, stateStopped); err != nil {
				return diag.Errorf("error waiting for Kafka Mirror Topic %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else {
			// Reset the state in TF state
			if err := d.Set(paramStatus, oldStatus); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return diag.Errorf(fmt.Sprintf("error updating Kafka Mirror Topic %q: %s \nbut %q->%q was attempted", d.Id(), disallowedTransitionErrorMessage, oldStatus, newStatus))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Mirror Topic %q", d.Id()), map[string]interface{}{kafkaMirrorTopicLoggingKey: d.Id()})
	}

	return kafkaMirrorTopicRead(ctx, d, meta)
}

func createKafkaRestClientFromKafkaBlock(d *schema.ResourceData, meta interface{}) (*KafkaRestClient, error) {
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	kafkaClusterRestEndpoint := extractStringValueFromBlock(d, paramKafkaCluster, paramRestEndpoint)
	kafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramKafkaCluster, paramCredentials, paramKey)
	kafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramKafkaCluster, paramCredentials, paramSecret)
	// Set isMetadataSetInProviderBlock to 'false' to disable inferring rest_endpoint / Kafka API Key from 'providers' block for confluent_cluster_link resource
	return meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(kafkaClusterRestEndpoint, kafkaClusterId, kafkaClusterApiKey, kafkaClusterApiSecret, false), nil
}

func clusterLinkBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramLinkName: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The name of the Cluster Link.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func sourceKafkaTopicBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramTopicName: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The name of the Source Kafka topic.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func constructKafkaMirrorTopicRequest(sourceTopicName, mirrorTopicName string) (v3.CreateMirrorTopicRequestData, error) {
	if mirrorTopicName == "" {
		// Mirror topic name is the same as source topic name: "my-topic"
		return v3.CreateMirrorTopicRequestData{
			SourceTopicName: sourceTopicName,
			// Don't set ReplicationFactor since API returns the following on Cloud:
			// {"error_code":40002,"message":"Topic replication factor must be 3"} when passing any other value but 3
		}, nil
	} else {
		// Mirror topic name "src_my-topic" where "src_" is the prefix configured on the link overrides source topic name: "my-topic"
		return v3.CreateMirrorTopicRequestData{
			SourceTopicName: sourceTopicName,
			MirrorTopicName: &mirrorTopicName,
			// Don't set ReplicationFactor since API returns the following on Cloud:
			// {"error_code":40002,"message":"Topic replication factor must be 3"} when passing any other value but 3
		}, nil
	}
}

func executeKafkaMirrorTopicCreate(ctx context.Context, c *KafkaRestClient, requestData v3.CreateMirrorTopicRequestData, linkName string) (*http.Response, error) {
	return c.apiClient.ClusterLinkingV3Api.CreateKafkaMirrorTopic(c.apiContext(ctx), c.clusterId, linkName).CreateMirrorTopicRequestData(requestData).Execute()
}

// Similar to URL structure
// Kafka Mirror Topic ID = lkc-kjnkvg/test_link/mirror_topic_name
// Kafka Topic ID = lkc-kjnkvg/topic_name

func createKafkaMirrorTopicId(clusterId, linkName, mirrorTopicName string) string {
	return fmt.Sprintf("%s/%s/%s", clusterId, linkName, mirrorTopicName)
}

func mirrorTopicKafkaClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the referred Kafka cluster.",
				},
				paramRestEndpoint: {
					Type:         schema.TypeString,
					Optional:     true,
					ForceNew:     true,
					Description:  "The REST endpoint of the Kafka cluster (e.g., `https://pkc-00000.us-central1.gcp.confluent.cloud:443`).",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
				},
				paramCredentials: {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "The Kafka API Credentials.",
					MinItems:    1,
					MaxItems:    1,
					Sensitive:   true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							paramKey: {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "The Kafka API Key for your Confluent Cloud cluster.",
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
							paramSecret: {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "The Kafka API Secret for your Confluent Cloud cluster.",
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
		},
	}
}
