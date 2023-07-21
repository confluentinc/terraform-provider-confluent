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
	paramLinkName                = "link_name"
	paramSourceKafkaCluster      = "source_kafka_cluster"
	paramDestinationKafkaCluster = "destination_kafka_cluster"
	paramLocalKafkaCluster       = "local_kafka_cluster"
	paramRemoteKafkaCluster      = "remote_kafka_cluster"
	paramLinkMode                = "link_mode"
	paramConnectionMode          = "connection_mode"

	bootstrapServersConfigKey = "bootstrap.servers"
	securityProtocolConfigKey = "security.protocol"

	saslMechanismConfigKey  = "sasl.mechanism"
	saslJaasConfigConfigKey = "sasl.jaas.config"

	localSecurityProtocolConfigKey = "local.security.protocol"
	localSaslMechanismConfigKey    = "local.sasl.mechanism"
	localSaslJaasConfigConfigKey   = "local.sasl.jaas.config"
	connectionModeConfigKey        = "connection.mode"
	linkModeConfigKey              = "link.mode"
	remoteLinkConnectionMode       = "remote.link.connection.mode"
	linkModeDestination            = "DESTINATION"
	linkModeSource                 = "SOURCE"
	linkModeBidirectional          = "BIDIRECTIONAL"
	connectionModeInbound          = "INBOUND"
	connectionModeOutbound         = "OUTBOUND"

	importSourceKafkaRestEndpointEnvVar           = "IMPORT_SOURCE_KAFKA_REST_ENDPOINT"
	importSourceKafkaBootstrapEndpointEnvVar      = "IMPORT_SOURCE_KAFKA_BOOTSTRAP_ENDPOINT"
	importDestinationKafkaRestEndpointEnvVar      = "IMPORT_DESTINATION_KAFKA_REST_ENDPOINT"
	importDestinationKafkaBootstrapEndpointEnvVar = "IMPORT_DESTINATION_KAFKA_BOOTSTRAP_ENDPOINT"

	importLocalKafkaRestEndpointEnvVar       = "IMPORT_LOCAL_KAFKA_REST_ENDPOINT"
	importLocalKafkaBootstrapEndpointEnvVar  = "IMPORT_LOCAL_KAFKA_BOOTSTRAP_ENDPOINT"
	importRemoteKafkaRestEndpointEnvVar      = "IMPORT_REMOTE_KAFKA_REST_ENDPOINT"
	importRemoteKafkaBootstrapEndpointEnvVar = "IMPORT_REMOTE_KAFKA_BOOTSTRAP_ENDPOINT"

	paramSourceKafkaCredentials      = "source_kafka_cluster.0.credentials"
	paramDestinationKafkaCredentials = "destination_kafka_cluster.0.credentials"
	paramLocalKafkaCredentials       = "local_kafka_cluster.0.credentials"
	paramRemoteKafkaCredentials      = "remote_kafka_cluster.0.credentials"

	docsClusterLinkConfigUrl = "https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/cluster-links-cc.html#configuring-cluster-link-behavior"
	dynamicClusterLinkConfig = "DYNAMIC_CLUSTER_LINK_CONFIG"
)

var sourceKafkaCredentialsBlockKey = fmt.Sprintf("%s.0.%s.#", paramSourceKafkaCluster, paramCredentials)
var destinationKafkaCredentialsBlockKey = fmt.Sprintf("%s.0.%s.#", paramDestinationKafkaCluster, paramCredentials)
var localKafkaCredentialsBlockKey = fmt.Sprintf("%s.0.%s.#", paramLocalKafkaCluster, paramCredentials)
var remoteKafkaCredentialsBlockKey = fmt.Sprintf("%s.0.%s.#", paramRemoteKafkaCluster, paramCredentials)

// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/cluster-links-cc.html#configuring-cluster-link-behavior
var editableClusterLinkSettings = []string{
	"acl.filters",
	"acl.sync.enable",
	"acl.sync.ms",
	"auto.create.mirror.topics.enable",
	"auto.create.mirror.topics.filters",
	"cluster.link.prefix",
	"consumer.group.prefix.enable",
	"consumer.offset.group.filters",
	"consumer.offset.sync.enable",
	"consumer.offset.sync.ms",
	"topic.config.sync.ms",
}

func clusterLinkResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: clusterLinkCreate,
		ReadContext:   clusterLinkRead,
		UpdateContext: clusterLinkUpdate,
		DeleteContext: clusterLinkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: clusterLinkImport,
		},
		Schema: map[string]*schema.Schema{
			paramLinkName: {
				Type:        schema.TypeString,
				Description: "The name of the Cluster Link.",
				Required:    true,
				ForceNew:    true,
			},
			paramSourceKafkaCluster:      clusterLinkKafkaClusterBlockSchema(paramSourceKafkaCluster),
			paramDestinationKafkaCluster: clusterLinkKafkaClusterBlockSchema(paramDestinationKafkaCluster),
			paramLocalKafkaCluster:       clusterLinkKafkaClusterBlockSchema(paramLocalKafkaCluster),
			paramRemoteKafkaCluster:      clusterLinkKafkaClusterBlockSchema(paramRemoteKafkaCluster),
			paramLinkMode: {
				Type:         schema.TypeString,
				Description:  "The mode of the Cluster Link.",
				Optional:     true,
				Default:      linkModeDestination,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{linkModeDestination, linkModeSource, linkModeBidirectional}, false),
			},
			paramConnectionMode: {
				Type:         schema.TypeString,
				Description:  "The connection mode of the Cluster Link.",
				Optional:     true,
				Default:      connectionModeOutbound,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{connectionModeInbound, connectionModeOutbound}, false),
			},
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:         true,
				Computed:         true,
				Description:      "The custom cluster link settings to set (e.g., `\"acl.sync.ms\" = \"5100\"`).",
				ValidateDiagFunc: clusterLinkSettingsKeysValidate,
			},
		},
	}
}

func clusterLinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: update validateClusterLinkInput to accept BIDIRECTIONAL link mode
	err := validateClusterLinkInput(d)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}
	kafkaRestClient, err := createKafkaRestClientForClusterLink(d, meta)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}
	linkName := d.Get(paramLinkName).(string)

	createClusterLinkRequest, err := constructClusterLinkRequest(d)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}
	createClusterLinkRequestJson, err := json.Marshal(createClusterLinkRequest)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: error marshaling %#v to json: %s", createClusterLinkRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Cluster Link: %s", createClusterLinkRequestJson))

	_, err = executeClusterLinkCreate(ctx, kafkaRestClient, createClusterLinkRequest, linkName)

	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}

	clusterLinkId := createClusterLinkId(kafkaRestClient.clusterId, linkName)
	d.SetId(clusterLinkId)

	// https://github.com/confluentinc/terraform-provider-confluent/issues/40#issuecomment-1048782379
	time.Sleep(kafkaRestAPIWaitAfterCreate)

	// Don't log created cluster link since API returns an empty 201 response.
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return clusterLinkRead(ctx, d, meta)
}

func executeClusterLinkCreate(ctx context.Context, c *KafkaRestClient, requestData v3.CreateLinkRequestData, linkName string) (*http.Response, error) {
	return c.apiClient.ClusterLinkingV3Api.CreateKafkaLink(c.apiContext(ctx), c.clusterId).CreateLinkRequestData(requestData).LinkName(linkName).Execute()
}

type ClusterLinkMetadata struct {
	sourceClusterId              string
	destinationClusterId         string
	localClusterId               string
	remoteClusterId              string
	sourceRestEndpoint           string
	destinationRestEndpoint      string
	localRestEndpoint            string
	remoteRestEndpoint           string
	sourceBootstrapEndpoint      string
	destinationBootstrapEndpoint string
	localBootstrapEndpoint       string
	remoteBootstrapEndpoint      string
	sourceClusterApiKey          string
	sourceClusterApiSecret       string
	destinationClusterApiKey     string
	destinationClusterApiSecret  string
	localClusterApiKey           string
	localClusterApiSecret        string
	remoteClusterApiKey          string
	remoteClusterApiSecret       string
}

func clusterLinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	kafkaRestClient, err := createKafkaRestClientForClusterLink(d, meta)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}
	linkName := d.Get(paramLinkName).(string)
	linkMode := d.Get(paramLinkMode).(string)
	connectionMode := d.Get(paramConnectionMode).(string)

	clusterLinkMetadata := &ClusterLinkMetadata{
		sourceClusterId:      extractStringValueFromBlock(d, paramSourceKafkaCluster, paramId),
		destinationClusterId: extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramId),
		localClusterId:       extractStringValueFromBlock(d, paramLocalKafkaCluster, paramId),
		remoteClusterId:      extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramId),

		sourceRestEndpoint:      extractStringValueFromBlock(d, paramSourceKafkaCluster, paramRestEndpoint),
		destinationRestEndpoint: extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramRestEndpoint),
		localRestEndpoint:       extractStringValueFromBlock(d, paramLocalKafkaCluster, paramRestEndpoint),
		remoteRestEndpoint:      extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramRestEndpoint),

		sourceBootstrapEndpoint:      extractStringValueFromBlock(d, paramSourceKafkaCluster, paramBootStrapEndpoint),
		destinationBootstrapEndpoint: extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramBootStrapEndpoint),
		localBootstrapEndpoint:       extractStringValueFromBlock(d, paramLocalKafkaCluster, paramBootStrapEndpoint),
		remoteBootstrapEndpoint:      extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramBootStrapEndpoint),

		sourceClusterApiKey:         extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramKey),
		sourceClusterApiSecret:      extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramSecret),
		destinationClusterApiKey:    extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramKey),
		destinationClusterApiSecret: extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramSecret),

		localClusterApiKey:     extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramKey),
		localClusterApiSecret:  extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramSecret),
		remoteClusterApiKey:    extractStringValueFromNestedBlock(d, paramRemoteKafkaCluster, paramCredentials, paramKey),
		remoteClusterApiSecret: extractStringValueFromNestedBlock(d, paramRemoteKafkaCluster, paramCredentials, paramSecret),
	}

	_, err = readClusterLinkAndSetAttributes(ctx, d, kafkaRestClient, linkName, linkMode, connectionMode, clusterLinkMetadata)
	if err != nil {
		return diag.Errorf("error reading Cluster Link: %s", createDescriptiveError(err))
	}

	return nil
}

func readClusterLinkAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, linkName, linkMode, connectionMode string, clusterLinkMetadata *ClusterLinkMetadata) ([]*schema.ResourceData, error) {
	clusterLink, resp, err := c.apiClient.ClusterLinkingV3Api.GetKafkaLink(c.apiContext(ctx), c.clusterId, linkName).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Cluster Link %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

		isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Cluster Link %q in TF state because Cluster Link could not be found on the server", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	clusterLinkJson, err := json.Marshal(clusterLink)
	if err != nil {
		return nil, fmt.Errorf("error reading Cluster Link %q: error marshaling %#v to json: %s", d.Id(), clusterLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Cluster Link %q: %s", d.Id(), clusterLinkJson), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	if _, err := setClusterLinkAttributes(ctx, d, c, clusterLink, linkMode, connectionMode, clusterLinkMetadata); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setClusterLinkAttributes(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, clusterLink v3.ListLinksResponseData,
	linkMode, connectionMode string, clusterLinkMetadata *ClusterLinkMetadata) (*schema.ResourceData, error) {
	if err := d.Set(paramLinkName, clusterLink.GetLinkName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramLinkMode, linkMode); err != nil {
		return nil, err
	}
	if err := d.Set(paramConnectionMode, connectionMode); err != nil {
		return nil, err
	}

	if linkMode == linkModeBidirectional {
		if err := d.Set(paramLocalKafkaCluster, []interface{}{map[string]interface{}{
			paramId:                clusterLinkMetadata.localClusterId,
			paramRestEndpoint:      clusterLinkMetadata.localRestEndpoint,
			paramBootStrapEndpoint: clusterLinkMetadata.localBootstrapEndpoint,
			paramCredentials: []interface{}{map[string]interface{}{
				paramKey:    clusterLinkMetadata.localClusterApiKey,
				paramSecret: clusterLinkMetadata.localClusterApiSecret,
			}},
		}}); err != nil {
			return nil, err
		}
		if connectionMode == connectionModeInbound {
			if err := d.Set(paramRemoteKafkaCluster, []interface{}{map[string]interface{}{
				paramId:                clusterLinkMetadata.remoteClusterId,
				paramRestEndpoint:      clusterLinkMetadata.remoteRestEndpoint,
				paramBootStrapEndpoint: clusterLinkMetadata.remoteBootstrapEndpoint,
			}}); err != nil {
				return nil, err
			}
		} else {
			if err := d.Set(paramRemoteKafkaCluster, []interface{}{map[string]interface{}{
				paramId:                clusterLinkMetadata.remoteClusterId,
				paramRestEndpoint:      clusterLinkMetadata.remoteRestEndpoint,
				paramBootStrapEndpoint: clusterLinkMetadata.remoteBootstrapEndpoint,
				paramCredentials: []interface{}{map[string]interface{}{
					paramKey:    clusterLinkMetadata.remoteClusterApiKey,
					paramSecret: clusterLinkMetadata.remoteClusterApiSecret,
				}},
			}}); err != nil {
				return nil, err
			}
		}
	} else {
		if err := d.Set(paramDestinationKafkaCluster, []interface{}{map[string]interface{}{
			paramId:                clusterLinkMetadata.destinationClusterId,
			paramRestEndpoint:      clusterLinkMetadata.destinationRestEndpoint,
			paramBootStrapEndpoint: clusterLinkMetadata.destinationBootstrapEndpoint,
			paramCredentials: []interface{}{map[string]interface{}{
				paramKey:    clusterLinkMetadata.destinationClusterApiKey,
				paramSecret: clusterLinkMetadata.destinationClusterApiSecret,
			}},
		}}); err != nil {
			return nil, err
		}
		if linkMode == linkModeDestination && connectionMode == connectionModeInbound {
			if err := d.Set(paramSourceKafkaCluster, []interface{}{map[string]interface{}{
				paramId:                clusterLinkMetadata.sourceClusterId,
				paramRestEndpoint:      clusterLinkMetadata.sourceRestEndpoint,
				paramBootStrapEndpoint: clusterLinkMetadata.sourceBootstrapEndpoint,
			}}); err != nil {
				return nil, err
			}
		} else {
			if err := d.Set(paramSourceKafkaCluster, []interface{}{map[string]interface{}{
				paramId:                clusterLinkMetadata.sourceClusterId,
				paramRestEndpoint:      clusterLinkMetadata.sourceRestEndpoint,
				paramBootStrapEndpoint: clusterLinkMetadata.sourceBootstrapEndpoint,
				paramCredentials: []interface{}{map[string]interface{}{
					paramKey:    clusterLinkMetadata.sourceClusterApiKey,
					paramSecret: clusterLinkMetadata.sourceClusterApiSecret,
				}},
			}}); err != nil {
				return nil, err
			}
		}
	}

	configs, err := loadClusterLinkConfigs(ctx, d, c, clusterLink.GetLinkName())
	if err != nil {
		return nil, err
	}
	if err := d.Set(paramConfigs, configs); err != nil {
		return nil, err
	}

	d.SetId(createClusterLinkId(c.clusterId, clusterLink.LinkName))
	return d, nil
}

func clusterLinkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramSourceKafkaCluster, paramSourceKafkaCredentials, paramDestinationKafkaCluster, paramDestinationKafkaCredentials, paramLocalKafkaCluster, paramLocalKafkaCredentials, paramRemoteKafkaCluster, paramRemoteKafkaCredentials, paramConfigs) {
		return diag.Errorf("error updating Cluster Link %q: only %q, %q, %q, %q and %q attributes can be updated for Cluster Link", d.Id(), paramSourceKafkaCredentials, paramDestinationKafkaCredentials, paramLocalKafkaCredentials, paramRemoteKafkaCredentials, paramConfigs)
	}
	if d.HasChange(paramConfigs) {
		// TF Provider allows the following operations for editable cluster link settings under 'config' block:
		// 1. Adding new key value pair, for example, "retention.ms" = "600000"
		// 2. Update a value for existing key value pair, for example, "retention.ms" = "600000" -> "retention.ms" = "600001"
		// You might find the list of editable cluster link settings and their limits at
		// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/cluster-links-cc.html#configuring-cluster-link-behavior

		// Extract 'old' and 'new' (include changes in TF configuration) cluster link settings
		// * 'old' cluster link settings -- all cluster link settings from TF configuration _before_ changes / updates (currently set on Confluent Cloud)
		// * 'new' cluster link settings -- all cluster link settings from TF configuration _after_ changes
		oldClusterSettingsMap, newClusterSettingsMap := extractOldAndNewSettings(d)

		// Verify that no cluster link settings were removed (reset to its default value) in TF configuration which is an unsupported operation at the moment
		for oldSettingName := range oldClusterSettingsMap {
			if _, ok := newClusterSettingsMap[oldSettingName]; !ok {
				return diag.Errorf("error updating Cluster Link %q: reset to cluster link setting's default value operation (in other words, removing cluster link settings from %q block) "+
					"is not supported at the moment. "+
					"Instead, find its default value at %s and set its current value to the default value.", d.Id(), paramConfigs, docsClusterLinkConfigUrl)
			}
		}

		// Construct a request for Kafka REST API
		_, newSettingsMapAny := d.GetChange(paramConfigs)
		updateConfigRequest := v3.AlterConfigBatchRequestData{
			Data: extractClusterLinkConfigsAlterConfigBatchRequestData(newSettingsMapAny.(map[string]interface{})),
		}
		kafkaRestClient, err := createKafkaRestClientForClusterLink(d, meta)
		if err != nil {
			return diag.Errorf("error updating Cluster Link: %s", createDescriptiveError(err))
		}
		linkName := d.Get(paramLinkName).(string)
		updateConfigRequestJson, err := json.Marshal(updateConfigRequest)
		if err != nil {
			return diag.Errorf("error updating Cluster Link: error marshaling %#v to json: %s", updateConfigRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Cluster Link %q: %s", d.Id(), updateConfigRequestJson), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

		// Send a request to Kafka REST API
		_, err = executeClusterLinkConfigUpdate(ctx, kafkaRestClient, linkName, updateConfigRequest)
		if err != nil {
			// For example, Kafka REST API will return Bad Request if new cluster link setting value exceeds the max limit:
			// 400 Bad Request: Config property 'delete.retention.ms' with value '63113904003' exceeded max limit of 60566400000.
			return diag.Errorf("error updating Cluster Link Config: %s", createDescriptiveError(err))
		}
		time.Sleep(kafkaRestAPIWaitAfterCreate)
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})
	}
	return clusterLinkRead(ctx, d, meta)
}

func clusterLinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	kafkaRestClient, err := createKafkaRestClientForClusterLink(d, meta)
	if err != nil {
		return diag.Errorf("error creating Cluster Link: %s", createDescriptiveError(err))
	}
	linkName := d.Get(paramLinkName).(string)

	_, err = kafkaRestClient.apiClient.ClusterLinkingV3Api.DeleteKafkaLink(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId, linkName).Execute()

	if err != nil {
		return diag.Errorf("error deleting Cluster Link %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForClusterLinkToBeDeleted(kafkaRestClient.apiContext(ctx), kafkaRestClient, linkName); err != nil {
		return diag.Errorf("error waiting for Cluster Link %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return nil
}

func clusterLinkImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	sourceRestEndpoint, sourceBootstrapEndpoint := extractKafkaClusterRestAndBootstrapEndpoints(importSourceKafkaRestEndpointEnvVar, importSourceKafkaBootstrapEndpointEnvVar)

	destinationRestEndpoint, destinationBootstrapEndpoint := extractKafkaClusterRestAndBootstrapEndpoints(importDestinationKafkaRestEndpointEnvVar, importDestinationKafkaBootstrapEndpointEnvVar)

	localRestEndpoint, localBootstrapEndpoint := extractKafkaClusterRestAndBootstrapEndpoints(importLocalKafkaRestEndpointEnvVar, importLocalKafkaBootstrapEndpointEnvVar)

	remoteRestEndpoint, remoteBootstrapEndpoint := extractKafkaClusterRestAndBootstrapEndpoints(importRemoteKafkaRestEndpointEnvVar, importRemoteKafkaBootstrapEndpointEnvVar)

	sourceClusterApiKey, sourceClusterApiSecret := extractSourceClusterApiKeyAndApiSecret()

	destinationClusterApiKey, destinationClusterApiSecret := extractDestinationClusterApiKeyAndApiSecret()

	localClusterApiKey, localClusterApiSecret := extractLocalClusterApiKeyAndApiSecret()

	remoteClusterApiKey, remoteClusterApiSecret := extractRemoteClusterApiKeyAndApiSecret()

	linkNameAndLinkModeAndConnectionModeAndSourceClusterIdAndDestinationClusterId := d.Id()
	parts := strings.Split(linkNameAndLinkModeAndConnectionModeAndSourceClusterIdAndDestinationClusterId, "/")

	if len(parts) != 5 {
		return nil, fmt.Errorf("error importing Cluster Link: invalid format: expected '<cluster link name>/<cluster link mode>/<cluster connection mode>/<Source (or Local) Kafka cluster ID>/<Destination (or Remote) Kafka cluster ID>'")
	}

	linkName := parts[0]
	linkMode := parts[1]
	connectionMode := parts[2]
	sourceClusterId := parts[3]
	destinationClusterId := parts[4]
	localClusterId := ""
	remoteClusterId := ""
	if linkMode == linkModeBidirectional {
		sourceClusterId = ""
		destinationClusterId = ""
		localClusterId = parts[3]
		remoteClusterId = parts[4]
	}

	clusterLinkMetadata := &ClusterLinkMetadata{
		sourceClusterId:      sourceClusterId,
		destinationClusterId: destinationClusterId,
		localClusterId:       localClusterId,
		remoteClusterId:      remoteClusterId,

		sourceRestEndpoint:      sourceRestEndpoint,
		destinationRestEndpoint: destinationRestEndpoint,
		localRestEndpoint:       localRestEndpoint,
		remoteRestEndpoint:      remoteRestEndpoint,

		sourceBootstrapEndpoint:      sourceBootstrapEndpoint,
		destinationBootstrapEndpoint: destinationBootstrapEndpoint,
		localBootstrapEndpoint:       localBootstrapEndpoint,
		remoteBootstrapEndpoint:      remoteBootstrapEndpoint,

		sourceClusterApiKey:         sourceClusterApiKey,
		sourceClusterApiSecret:      sourceClusterApiSecret,
		destinationClusterApiKey:    destinationClusterApiKey,
		destinationClusterApiSecret: destinationClusterApiSecret,

		localClusterApiKey:     localClusterApiKey,
		localClusterApiSecret:  localClusterApiSecret,
		remoteClusterApiKey:    remoteClusterApiKey,
		remoteClusterApiSecret: remoteClusterApiSecret,
	}

	var kafkaRestClient *KafkaRestClient
	if linkMode == linkModeBidirectional {
		kafkaRestClient = meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(localRestEndpoint, localClusterId, localClusterApiKey, localClusterApiSecret, false, false)
	} else if linkMode == linkModeDestination {
		kafkaRestClient = meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(destinationRestEndpoint, destinationClusterId, destinationClusterApiKey, destinationClusterApiSecret, false, false)
	} else {
		kafkaRestClient = meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(sourceRestEndpoint, sourceClusterId, sourceClusterApiKey, sourceClusterApiSecret, false, false)
	}

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readClusterLinkAndSetAttributes(ctx, d, kafkaRestClient, linkName, linkMode, connectionMode, clusterLinkMetadata); err != nil {
		return nil, fmt.Errorf("error importing Cluster Link %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Cluster Link %q", d.Id()), map[string]interface{}{clusterLinkLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func convertToConfigData(configs map[string]interface{}) []v3.ConfigData {
	configResult := make([]v3.ConfigData, len(configs))

	i := 0
	for name, value := range configs {
		v := value.(string)
		configResult[i] = v3.ConfigData{
			Name:  name,
			Value: *v3.NewNullableString(&v),
		}
		i += 1
	}
	return configResult
}

func constructCloudConfigForBidirectionalOutboundMode(localApiKey, localKafkaApiSecret,
	remoteKafkaBootstrapEndpoint, remoteKafkaApiKey, remoteKafkaApiSecret string) map[string]interface{} {
	config := make(map[string]interface{})
	config[connectionModeConfigKey] = connectionModeOutbound
	config[remoteLinkConnectionMode] = connectionModeInbound
	config[linkModeConfigKey] = linkModeBidirectional
	config[localSecurityProtocolConfigKey] = "SASL_SSL"
	config[localSaslMechanismConfigKey] = "PLAIN"
	config[localSaslJaasConfigConfigKey] = fmt.Sprintf("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";", localApiKey, localKafkaApiSecret)

	// TODO: use constructCloudConfigForDestinationOutboundMode and merge 2 configs
	config[bootstrapServersConfigKey] = remoteKafkaBootstrapEndpoint
	config[securityProtocolConfigKey] = "SASL_SSL"
	config[saslMechanismConfigKey] = "PLAIN"
	config[saslJaasConfigConfigKey] = fmt.Sprintf("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";", remoteKafkaApiKey, remoteKafkaApiSecret)

	return config
}

func constructCloudConfigForBidirectionalInboundMode(bootstrapEndpoint string) map[string]interface{} {
	config := make(map[string]interface{})
	config[connectionModeConfigKey] = connectionModeInbound
	config[linkModeConfigKey] = linkModeBidirectional
	config[bootstrapServersConfigKey] = bootstrapEndpoint
	return config
}

func constructCloudConfigForDestinationOutboundMode(bootstrapEndpoint, kafkaApiKey, kafkaApiSecret string) map[string]interface{} {
	config := make(map[string]interface{})
	config[connectionModeConfigKey] = connectionModeOutbound
	config[linkModeConfigKey] = linkModeDestination
	config[bootstrapServersConfigKey] = bootstrapEndpoint
	config[securityProtocolConfigKey] = "SASL_SSL"
	config[saslMechanismConfigKey] = "PLAIN"
	config[saslJaasConfigConfigKey] = fmt.Sprintf("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";", kafkaApiKey, kafkaApiSecret)
	return config
}

func constructCloudConfigForDestinationInboundMode(bootstrapEndpoint string) map[string]interface{} {
	config := make(map[string]interface{})
	config[connectionModeConfigKey] = connectionModeInbound
	config[linkModeConfigKey] = linkModeDestination
	config[bootstrapServersConfigKey] = bootstrapEndpoint
	return config
}

func constructCloudConfigForSourceOutboundMode(sourceKafkaApiKey, sourceKafkaApiSecret,
	destinationKafkaBootstrapEndpoint, destinationKafkaApiKey, destinationKafkaApiSecret string) map[string]interface{} {
	config := make(map[string]interface{})
	config[connectionModeConfigKey] = connectionModeOutbound
	config[linkModeConfigKey] = linkModeSource
	config[localSecurityProtocolConfigKey] = "SASL_SSL"
	config[localSaslMechanismConfigKey] = "PLAIN"
	config[localSaslJaasConfigConfigKey] = fmt.Sprintf("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";", sourceKafkaApiKey, sourceKafkaApiSecret)

	// TODO: use constructCloudConfigForDestinationOutboundMode and merge 2 configs
	config[bootstrapServersConfigKey] = destinationKafkaBootstrapEndpoint
	config[securityProtocolConfigKey] = "SASL_SSL"
	config[saslMechanismConfigKey] = "PLAIN"
	config[saslJaasConfigConfigKey] = fmt.Sprintf("org.apache.kafka.common.security.plain.PlainLoginModule required username=\"%s\" password=\"%s\";", destinationKafkaApiKey, destinationKafkaApiSecret)

	return config
}

func createClusterLinkId(clusterId, linkName string) string {
	return fmt.Sprintf("%s/%s", clusterId, linkName)
}

func createKafkaRestClientForClusterLink(d *schema.ResourceData, meta interface{}) (*KafkaRestClient, error) {
	linkMode := d.Get(paramLinkMode).(string)
	if linkMode == linkModeBidirectional {
		localKafkaClusterId := extractStringValueFromBlock(d, paramLocalKafkaCluster, paramId)
		localKafkaClusterRestEndpoint := extractStringValueFromBlock(d, paramLocalKafkaCluster, paramRestEndpoint)
		localKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramKey)
		localKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramSecret)
		// Set isMetadataSetInProviderBlock to 'false' to disable inferring rest_endpoint / Kafka API Key from 'providers' block for confluent_cluster_link resource
		return meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(localKafkaClusterRestEndpoint, localKafkaClusterId, localKafkaClusterApiKey, localKafkaClusterApiSecret, false, false), nil
	} else if linkMode == linkModeDestination {
		destinationKafkaClusterId := extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramId)
		destinationKafkaClusterRestEndpoint := extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramRestEndpoint)
		destinationKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramKey)
		destinationKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramSecret)
		// Set isMetadataSetInProviderBlock to 'false' to disable inferring rest_endpoint / Kafka API Key from 'providers' block for confluent_cluster_link resource
		return meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(destinationKafkaClusterRestEndpoint, destinationKafkaClusterId, destinationKafkaClusterApiKey, destinationKafkaClusterApiSecret, false, false), nil
	} else {
		// linkMode = linkModeSource
		sourceKafkaClusterId := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramId)
		sourceKafkaClusterRestEndpoint := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramRestEndpoint)
		sourceKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramKey)
		sourceKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramSecret)
		// Set isMetadataSetInProviderBlock to 'false' to disable inferring rest_endpoint / Kafka API Key from 'providers' block for confluent_cluster_link resource
		return meta.(*Client).kafkaRestClientFactory.CreateKafkaRestClient(sourceKafkaClusterRestEndpoint, sourceKafkaClusterId, sourceKafkaClusterApiKey, sourceKafkaClusterApiSecret, false, false), nil
	}
}

func constructClusterLinkRequest(d *schema.ResourceData) (v3.CreateLinkRequestData, error) {
	linkMode := d.Get(paramLinkMode).(string)
	connectionMode := d.Get(paramConnectionMode).(string)
	clusterLinkSettings := extractClusterLinkConfigsConfigData(d.Get(paramConfigs).(map[string]interface{}))

	if linkMode == linkModeBidirectional {
		if connectionMode == connectionModeOutbound {
			localKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramKey)
			localKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramLocalKafkaCluster, paramCredentials, paramSecret)
			remoteKafkaClusterId := extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramId)
			remoteKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramRemoteKafkaCluster, paramCredentials, paramKey)
			remoteKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramRemoteKafkaCluster, paramCredentials, paramSecret)
			remoteKafkaClusterBootstrapEndpoint := extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramBootStrapEndpoint)
			configs := convertToConfigData(constructCloudConfigForBidirectionalOutboundMode(localKafkaClusterApiKey, localKafkaClusterApiSecret, remoteKafkaClusterBootstrapEndpoint, remoteKafkaClusterApiKey, remoteKafkaClusterApiSecret))

			// Add top level cluster link configs
			configs = append(configs, clusterLinkSettings...)

			return v3.CreateLinkRequestData{
				RemoteClusterId: &remoteKafkaClusterId,
				Configs:         &configs,
			}, nil
		} else {
			// connectionMode == connectionModeInbound
			remoteKafkaClusterId := extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramId)
			remoteKafkaClusterBootstrapEndpoint := extractStringValueFromBlock(d, paramRemoteKafkaCluster, paramBootStrapEndpoint)
			configs := convertToConfigData(constructCloudConfigForBidirectionalInboundMode(remoteKafkaClusterBootstrapEndpoint))

			// Add top level cluster link configs
			configs = append(configs, clusterLinkSettings...)

			return v3.CreateLinkRequestData{
				RemoteClusterId: &remoteKafkaClusterId,
				Configs:         &configs,
			}, nil
		}
	} else if linkMode == linkModeDestination {
		if connectionMode == connectionModeOutbound {
			sourceKafkaClusterId := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramId)
			sourceKafkaClusterBootstrapEndpoint := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramBootStrapEndpoint)
			sourceKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramKey)
			sourceKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramSecret)
			configs := convertToConfigData(constructCloudConfigForDestinationOutboundMode(sourceKafkaClusterBootstrapEndpoint, sourceKafkaClusterApiKey, sourceKafkaClusterApiSecret))

			// Add top level cluster link configs
			configs = append(configs, clusterLinkSettings...)

			return v3.CreateLinkRequestData{
				SourceClusterId: &sourceKafkaClusterId,
				Configs:         &configs,
			}, nil
		} else {
			// connectionMode == connectionModeInbound
			sourceKafkaClusterId := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramId)
			sourceKafkaClusterBootstrapEndpoint := extractStringValueFromBlock(d, paramSourceKafkaCluster, paramBootStrapEndpoint)
			configs := convertToConfigData(constructCloudConfigForDestinationInboundMode(sourceKafkaClusterBootstrapEndpoint))

			// Add top level cluster link configs
			configs = append(configs, clusterLinkSettings...)

			return v3.CreateLinkRequestData{
				SourceClusterId: &sourceKafkaClusterId,
				Configs:         &configs,
			}, nil
		}
	} else {
		// linkMode = linkModeSource
		sourceKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramKey)
		sourceKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramSourceKafkaCluster, paramCredentials, paramSecret)
		destinationKafkaClusterId := extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramId)
		destinationKafkaClusterBootstrapEndpoint := extractStringValueFromBlock(d, paramDestinationKafkaCluster, paramBootStrapEndpoint)
		destinationKafkaClusterApiKey := extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramKey)
		destinationKafkaClusterApiSecret := extractStringValueFromNestedBlock(d, paramDestinationKafkaCluster, paramCredentials, paramSecret)
		configs := convertToConfigData(constructCloudConfigForSourceOutboundMode(sourceKafkaClusterApiKey, sourceKafkaClusterApiSecret, destinationKafkaClusterBootstrapEndpoint, destinationKafkaClusterApiKey, destinationKafkaClusterApiSecret))

		// Add top level cluster link configs
		configs = append(configs, clusterLinkSettings...)

		return v3.CreateLinkRequestData{
			DestinationClusterId: &destinationKafkaClusterId,
			Configs:              &configs,
		}, nil
	}
}

func clusterLinkKafkaClusterBlockSchema(blockName string) *schema.Schema {
	oneOfEndpointsKeys := []string{
		fmt.Sprintf("%s.0.%s", blockName, paramRestEndpoint),
		fmt.Sprintf("%s.0.%s", blockName, paramBootStrapEndpoint),
	}

	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Optional: true,
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
					// A user should provide a value for either "paramRestEndpoint" or "paramBootStrapEndpoint" attribute
					ExactlyOneOf: oneOfEndpointsKeys,
				},
				paramBootStrapEndpoint: {
					Type:        schema.TypeString,
					Optional:    true,
					ForceNew:    true,
					Description: "The bootstrap endpoint used by Kafka clients to connect to the Kafka cluster. (e.g., `SASL_SSL://pkc-00000.us-central1.gcp.confluent.cloud:9092` or pkc-00000.us-central1.gcp.confluent.cloud:9092`).",
					// A user should provide a value for either "paramRestEndpoint" or "paramBootStrapEndpoint" attribute
					ExactlyOneOf: oneOfEndpointsKeys,
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

func extractKafkaClusterRestAndBootstrapEndpoints(restEndpointEnvVar, bootstrapEndpointEnvVar string) (string, string) {
	restEndpoint := getEnv(restEndpointEnvVar, "")
	bootstrapEndpoint := getEnv(bootstrapEndpointEnvVar, "")

	return restEndpoint, bootstrapEndpoint
}

func extractSourceClusterApiKeyAndApiSecret() (string, string) {
	clusterApiKey := getEnv("IMPORT_SOURCE_KAFKA_API_KEY", "")
	clusterApiSecret := getEnv("IMPORT_SOURCE_KAFKA_API_SECRET", "")
	return clusterApiKey, clusterApiSecret
}

func extractDestinationClusterApiKeyAndApiSecret() (string, string) {
	clusterApiKey := getEnv("IMPORT_DESTINATION_KAFKA_API_KEY", "")
	clusterApiSecret := getEnv("IMPORT_DESTINATION_KAFKA_API_SECRET", "")
	return clusterApiKey, clusterApiSecret
}

func extractLocalClusterApiKeyAndApiSecret() (string, string) {
	clusterApiKey := getEnv("IMPORT_LOCAL_KAFKA_API_KEY", "")
	clusterApiSecret := getEnv("IMPORT_LOCAL_KAFKA_API_SECRET", "")
	return clusterApiKey, clusterApiSecret
}

func extractRemoteClusterApiKeyAndApiSecret() (string, string) {
	clusterApiKey := getEnv("IMPORT_REMOTE_KAFKA_API_KEY", "")
	clusterApiSecret := getEnv("IMPORT_REMOTE_KAFKA_API_SECRET", "")
	return clusterApiKey, clusterApiSecret
}

func validateClusterLinkInputByLinkModeAndConnectionMode(d *schema.ResourceData, linkMode, connectionMode string) error {
	if linkMode == linkModeBidirectional {
		if d.Get(localKafkaCredentialsBlockKey).(int) == 0 {
			return fmt.Errorf("%q must be specified for %q", paramCredentials, paramLocalKafkaCluster)
		}
		// Expect
		// * bootstrap_endpoint to be specified for a remote cluster
		// * rest_endpoint to be specified for a local cluster
		if d.Get(fmt.Sprintf("%s.0.%s", paramRemoteKafkaCluster, paramBootStrapEndpoint)).(string) == "" {
			return fmt.Errorf("%q must be specified for %q", paramBootStrapEndpoint, paramRemoteKafkaCluster)
		}
		if d.Get(fmt.Sprintf("%s.0.%s", paramLocalKafkaCluster, paramRestEndpoint)).(string) == "" {
			return fmt.Errorf("%q must be specified for %q", paramRestEndpoint, paramLocalKafkaCluster)
		}
		return nil
	}

	if d.Get(destinationKafkaCredentialsBlockKey).(int) == 0 {
		return fmt.Errorf("%q must be specified for %q", paramCredentials, paramDestinationKafkaCluster)
	}
	if linkMode == linkModeDestination {
		// Expect
		// * bootstrap_endpoint to be specified for a source cluster
		// * rest_endpoint to be specified for a destination cluster
		if d.Get(fmt.Sprintf("%s.0.%s", paramSourceKafkaCluster, paramBootStrapEndpoint)).(string) == "" {
			return fmt.Errorf("%q must be specified for %q", paramBootStrapEndpoint, paramSourceKafkaCluster)
		}
		if d.Get(fmt.Sprintf("%s.0.%s", paramDestinationKafkaCluster, paramRestEndpoint)).(string) == "" {
			return fmt.Errorf("%q must be specified for %q", paramRestEndpoint, paramDestinationKafkaCluster)
		}
		if connectionMode == connectionModeOutbound {
			if d.Get(sourceKafkaCredentialsBlockKey).(int) == 0 {
				return fmt.Errorf("%q must be specified for %q", paramCredentials, paramSourceKafkaCluster)
			}
		} else {
			if d.Get(sourceKafkaCredentialsBlockKey).(int) != 0 {
				return fmt.Errorf("%q must not be specified for %q", paramCredentials, paramSourceKafkaCluster)
			}
		}
	} else if linkMode == linkModeSource {
		if connectionMode == connectionModeOutbound {
			// Expect
			// * rest_endpoint to be specified for a source cluster
			// * bootstrap_endpoint to be specified for a destination cluster
			if d.Get(fmt.Sprintf("%s.0.%s", paramSourceKafkaCluster, paramRestEndpoint)).(string) == "" {
				return fmt.Errorf("%q must be specified for %q", paramRestEndpoint, paramSourceKafkaCluster)
			}
			if d.Get(fmt.Sprintf("%s.0.%s", paramDestinationKafkaCluster, paramBootStrapEndpoint)).(string) == "" {
				return fmt.Errorf("%q must be specified for %q", paramBootStrapEndpoint, paramDestinationKafkaCluster)
			}
			if d.Get(sourceKafkaCredentialsBlockKey).(int) == 0 {
				return fmt.Errorf("%q must be specified for %q", paramCredentials, paramSourceKafkaCluster)
			}
		} else {
			return fmt.Errorf("source initiated cluster link can't have %q=%q", connectionMode, linkModeDestination)
		}
	}
	return nil
}

func validateClusterLinkInput(d *schema.ResourceData) error {
	linkMode := d.Get(paramLinkMode).(string)
	connectionMode := d.Get(paramConnectionMode).(string)
	return validateClusterLinkInputByLinkModeAndConnectionMode(d, linkMode, connectionMode)
}

func loadClusterLinkConfigs(ctx context.Context, d *schema.ResourceData, c *KafkaRestClient, linkName string) (map[string]string, error) {
	clusterLinkConfig, _, err := c.apiClient.ClusterLinkingV3Api.ListKafkaLinkConfigs(c.apiContext(ctx), c.clusterId, linkName).Execute()
	if err != nil {
		return nil, fmt.Errorf("error reading Cluster Link %q: could not load configs %s", linkName, createDescriptiveError(err))
	}

	config := make(map[string]string)
	for _, remoteConfig := range clusterLinkConfig.GetData() {
		// Extract configs that were set via overriden vs set by default
		if stringInSlice(remoteConfig.GetName(), editableClusterLinkSettings, false) && remoteConfig.Source == dynamicClusterLinkConfig {
			config[remoteConfig.GetName()] = remoteConfig.GetValue()
		}
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error reading Cluster Link: error marshaling %#v to json: %s", config, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Cluster Link %q Settings: %s", d.Id(), configJson), map[string]interface{}{clusterLinkLoggingKey: d.Id()})

	return config, nil
}

func extractClusterLinkConfigsAlterConfigBatchRequestData(configs map[string]interface{}) []v3.AlterConfigBatchRequestDataData {
	configResult := make([]v3.AlterConfigBatchRequestDataData, len(configs))

	i := 0
	for name, value := range configs {
		v := value.(string)
		configResult[i] = v3.AlterConfigBatchRequestDataData{
			Name:  name,
			Value: *v3.NewNullableString(&v),
		}
		i += 1
	}

	return configResult
}

func extractClusterLinkConfigsConfigData(configs map[string]interface{}) []v3.ConfigData {
	configResult := make([]v3.ConfigData, len(configs))

	i := 0
	for name, value := range configs {
		v := value.(string)
		configResult[i] = v3.ConfigData{
			Name:  name,
			Value: *v3.NewNullableString(&v),
		}
		i += 1
	}

	return configResult
}

func executeClusterLinkConfigUpdate(ctx context.Context, c *KafkaRestClient, linkName string, requestData v3.AlterConfigBatchRequestData) (*http.Response, error) {
	return c.apiClient.ClusterLinkingV3Api.UpdateKafkaLinkConfigBatch(c.apiContext(ctx), c.clusterId, linkName).AlterConfigBatchRequestData(requestData).Execute()
}
