// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"strings"

	tableflow "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramEnableCompaction      = "enable_compaction"
	paramEnablePartitioning    = "enable_partitioning"
	paramSuspended             = "suspended"
	paramRetentionMs           = "retention_ms"
	paramByobAws               = "byob_aws"
	paramManagedStorage        = "managed_storage"
	paramBucketName            = "bucket_name"
	paramBucketRegion          = "bucket_region"
	paramProviderIntegrationId = "provider_integration_id"
	paramTableFormats          = "table_formats"
	paramTablePath             = "table_path"
	paramRecordFailureStrategy = "record_failure_strategy"
	paramErrorHandling         = "error_handling"
	paramLogTarget             = "log_target"
	paramWriteMode             = "write_mode"

	byobAwsSpecKind        = "ByobAws"
	managedStorageSpecKind = "Managed"

	errorHandlingSuspendMode = "SUSPEND"
	errorHandlingSkipMode    = "SKIP"
	errorHandlingLogMode     = "LOG"
)

var acceptedBucketTypes = []string{paramByobAws, paramManagedStorage}
var acceptedErrorHandlingModes = []string{errorHandlingSuspendMode, errorHandlingSkipMode, errorHandlingLogMode}

func tableflowTopicResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: tableflowTopicCreate,
		ReadContext:   tableflowTopicRead,
		UpdateContext: tableflowTopicUpdate,
		DeleteContext: tableflowTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: tableflowTopicImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the Kafka topic for which Tableflow is enabled.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramEnableCompaction: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "This flag determines whether to enable compaction for the Tableflow enabled topic.",
			},
			paramEnablePartitioning: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "This flag determines whether to enable partitioning for the Tableflow enabled topic.",
			},
			paramSuspended: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the Tableflow should be suspended.",
			},
			paramRetentionMs: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "604800000",
				Description: "The max age of snapshots (Iceberg) or versions (Delta) (snapshot/version expiration) to keep on the table in milliseconds for the Tableflow enabled topic.",
			},
			paramTableFormats: {
				Type:        schema.TypeSet,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The supported table formats for the Tableflow-enabled topic.",
			},
			paramTablePath: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The current storage path where the data and metadata is stored for this table.",
			},
			paramRecordFailureStrategy: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Deprecated:  "This attribute is deprecated and will be removed in a future release.",
				Description: "The strategy to handle record failures in the Tableflow enabled topic during materialization.",
			},
			paramWriteMode: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Indicates the write mode of the tableflow topic.",
			},
			paramErrorHandling:  errorHandlingSchema(),
			paramKafkaCluster:   requiredKafkaClusterBlockSchema(),
			paramEnvironment:    environmentSchema(),
			paramCredentials:    credentialsSchema(),
			paramByobAws:        byobAwsSchema(),
			paramManagedStorage: managedStorageSchema(),
		},
		CustomizeDiff: customdiff.Sequence(resourceCredentialBlockValidationWithOAuth),
	}
}

func byobAwsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		ForceNew:    true,
		Optional:    true,
		Description: "The Tableflow storage configuration for BYOB enabled topic in AWS.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramBucketName: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramBucketRegion: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramProviderIntegrationId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedBucketTypes,
	}
}

func managedStorageSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "The storage configuration for Confluent managed tableflow enabled topic.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		MaxItems:     0,
		ExactlyOneOf: acceptedBucketTypes,
	}
}

func errorHandlingSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramMode: {
					Type:         schema.TypeString,
					Optional:     true,
					Computed:     true,
					Description:  "The error handling mode where the bad records are logged to a dead-letter queue (DLQ) topic and the materialization continues with the next record.",
					ValidateFunc: validation.StringInSlice(acceptedErrorHandlingModes, true),
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						if strings.ToLower(old) == strings.ToLower(new) {
							return true
						}
						return false
					},
				},
				paramLogTarget: {
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: `The topic to which the bad records will be logged. Creates the topic if it doesn't already exist. The default topic is "error_log".`,
				},
			},
		},
		MaxItems: 1,
	}
}

func tableflowTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	isByobAws := len(d.Get(paramByobAws).([]interface{})) > 0
	isManaged := len(d.Get(paramManagedStorage).([]interface{})) > 0

	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	tableflowTopicSpec := tableflow.NewTableflowV1TableflowTopicSpec()
	tableflowTopicSpec.SetDisplayName(displayName)
	tableflowTopicSpec.SetEnvironment(tableflow.GlobalObjectReference{Id: environmentId})
	tableflowTopicSpec.SetKafkaCluster(tableflow.EnvScopedObjectReference{Id: clusterId})
	if tableFormats := convertToStringSlice(d.Get(paramTableFormats).(*schema.Set).List()); len(tableFormats) > 0 {
		tableflowTopicSpec.SetTableFormats(tableFormats)
	}

	tableflowTopicSpec.Config = tableflow.NewTableflowV1TableFlowTopicConfigsSpec()
	if retentionMs := d.Get(paramRetentionMs).(string); retentionMs != "" {
		tableflowTopicSpec.Config.SetRetentionMs(retentionMs)
	}
	if recordFailureStrategy := d.Get(paramRecordFailureStrategy).(string); recordFailureStrategy != "" {
		tableflowTopicSpec.Config.SetRecordFailureStrategy(recordFailureStrategy)
	}

	if len(d.Get(paramErrorHandling).([]interface{})) > 0 {
		mode := extractStringValueFromBlock(d, paramErrorHandling, paramMode)
		target := extractStringValueFromBlock(d, paramErrorHandling, paramLogTarget)

		if strings.ToUpper(mode) == errorHandlingSuspendMode {
			tableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingSuspend: &tableflow.TableflowV1ErrorHandlingSuspend{
					Mode: errorHandlingSuspendMode,
				},
			})
		} else if strings.ToUpper(mode) == errorHandlingSkipMode {
			tableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingSkip: &tableflow.TableflowV1ErrorHandlingSkip{
					Mode: errorHandlingSkipMode,
				},
			})
		} else if strings.ToUpper(mode) == errorHandlingLogMode {
			tableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingLog: &tableflow.TableflowV1ErrorHandlingLog{
					Mode:   errorHandlingLogMode,
					Target: tableflow.PtrString(target),
				},
			})
		}
	}

	if isByobAws {
		tableflowTopicSpec.SetStorage(tableflow.TableflowV1TableflowTopicSpecStorageOneOf{
			TableflowV1ByobAwsSpec: &tableflow.TableflowV1ByobAwsSpec{
				Kind:                  byobAwsSpecKind,
				BucketName:            extractStringValueFromBlock(d, paramByobAws, paramBucketName),
				BucketRegion:          tableflow.PtrString(extractStringValueFromBlock(d, paramByobAws, paramBucketRegion)),
				ProviderIntegrationId: extractStringValueFromBlock(d, paramByobAws, paramProviderIntegrationId),
			},
		})
	} else if isManaged {
		tableflowTopicSpec.SetStorage(tableflow.TableflowV1TableflowTopicSpecStorageOneOf{
			TableflowV1ManagedStorageSpec: tableflow.NewTableflowV1ManagedStorageSpec(managedStorageSpecKind),
		})
	}

	createTableflowTopicRequest := tableflow.NewTableflowV1TableflowTopic()
	createTableflowTopicRequest.SetSpec(*tableflowTopicSpec)

	createTableflowTopicRequestJson, err := json.Marshal(createTableflowTopicRequest)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: error marshaling %#v to json: %s", createTableflowTopicRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Tableflow Topic: %s", createTableflowTopicRequestJson))

	req := tableflowRestClient.apiClient.TableflowTopicsTableflowV1Api.CreateTableflowV1TableflowTopic(tableflowRestClient.apiContext(ctx)).TableflowV1TableflowTopic(*createTableflowTopicRequest)
	createdTableflowTopic, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err, resp))
	}

	d.SetId(displayName)

	createdTableflowTopicJson, err := json.Marshal(createdTableflowTopic)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic %q: error marshaling %#v to json: %s", d.Id(), createdTableflowTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Tableflow Topic %q: %s", d.Id(), createdTableflowTopicJson), map[string]interface{}{tableflowTopicKey: d.Id()})

	return tableflowTopicRead(ctx, d, meta)
}

func executeTableflowTopicRead(ctx context.Context, c *TableflowRestClient, environmentId, clusterId, displayName string) (tableflow.TableflowV1TableflowTopic, *http.Response, error) {
	return c.apiClient.TableflowTopicsTableflowV1Api.GetTableflowV1TableflowTopic(c.apiContext(ctx), displayName).Environment(environmentId).SpecKafkaCluster(clusterId).Execute()
}

func tableflowTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Tableflow Topic %q", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	tableflowTopicId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	if _, err := readTableflowTopicAndSetAttributes(ctx, d, tableflowRestClient, environmentId, clusterId, tableflowTopicId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Tableflow Topic %q: %s", tableflowTopicId, createDescriptiveError(err)))
	}

	return nil
}

func readTableflowTopicAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *TableflowRestClient, environmentId, clusterId, tableflowTopicId string) ([]*schema.ResourceData, error) {
	tableflowTopic, resp, err := executeTableflowTopicRead(c.apiContext(ctx), c, environmentId, clusterId, tableflowTopicId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Tableflow Topic %q: %s", tableflowTopicId, createDescriptiveError(err, resp)), map[string]interface{}{tableflowTopicKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Tableflow Topic %q in TF state because Tableflow Topic could not be found on the server", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	tableflowTopicJson, err := json.Marshal(tableflowTopic)
	if err != nil {
		return nil, fmt.Errorf("error reading Tableflow Topic %q: error marshaling %#v to json: %s", tableflowTopicId, tableflowTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Tableflow Topic %q: %s", d.Id(), tableflowTopicJson), map[string]interface{}{tableflowTopicKey: d.Id()})

	if _, err := setTableflowTopicAttributes(d, c, tableflowTopic); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Tableflow Topic %q", tableflowTopicId), map[string]interface{}{tableflowTopicKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setTableflowTopicAttributes(d *schema.ResourceData, c *TableflowRestClient, tableflowTopic tableflow.TableflowV1TableflowTopic) (*schema.ResourceData, error) {
	storageType, err := getStorageType(tableflowTopic)
	if err != nil {
		return nil, err
	}

	if err := d.Set(paramDisplayName, tableflowTopic.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEnableCompaction, tableflowTopic.GetSpec().Config.GetEnableCompaction()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEnablePartitioning, tableflowTopic.GetSpec().Config.GetEnablePartitioning()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSuspended, tableflowTopic.Spec.GetSuspended()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRetentionMs, tableflowTopic.GetSpec().Config.GetRetentionMs()); err != nil {
		return nil, err
	}
	if err := d.Set(paramTableFormats, tableflowTopic.Spec.GetTableFormats()); err != nil {
		return nil, err
	}

	if storageType == byobAwsSpecKind {
		if err := d.Set(paramTablePath, tableflowTopic.GetSpec().Storage.TableflowV1ByobAwsSpec.GetTablePath()); err != nil {
			return nil, err
		}
	} else if storageType == managedStorageSpecKind {
		if err := d.Set(paramTablePath, tableflowTopic.GetSpec().Storage.TableflowV1ManagedStorageSpec.GetTablePath()); err != nil {
			return nil, err
		}
	}

	if err := d.Set(paramRecordFailureStrategy, tableflowTopic.GetSpec().Config.GetRecordFailureStrategy()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, tableflowTopic.GetSpec().Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, tableflowTopic.GetSpec().KafkaCluster.GetId(), d); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramWriteMode, tableflowTopic.Status.GetWriteMode()); err != nil {
		return nil, err
	}

	if tableflowTopic.Spec.GetStorage().TableflowV1ByobAwsSpec != nil {
		if err := d.Set(paramByobAws, []interface{}{map[string]interface{}{
			paramBucketName:            tableflowTopic.Spec.GetStorage().TableflowV1ByobAwsSpec.GetBucketName(),
			paramBucketRegion:          tableflowTopic.Spec.GetStorage().TableflowV1ByobAwsSpec.GetBucketRegion(),
			paramProviderIntegrationId: tableflowTopic.Spec.GetStorage().TableflowV1ByobAwsSpec.GetProviderIntegrationId(),
		}}); err != nil {
			return nil, err
		}
	} else if tableflowTopic.Spec.GetStorage().TableflowV1ManagedStorageSpec != nil {
		if err := d.Set(paramManagedStorage, []interface{}{make(map[string]string)}); err != nil {
			return nil, err
		}
	}

	// Since target is not returned for Suspend and Skip modes, we need to extract it from the error_handling block to prevent drift in case the user has set it
	target := extractStringValueFromBlock(d, paramErrorHandling, paramLogTarget)
	if tableflowTopic.GetSpec().Config.GetErrorHandling().TableflowV1ErrorHandlingSuspend != nil {
		if err := d.Set(paramErrorHandling, []interface{}{map[string]interface{}{
			paramMode:      errorHandlingSuspendMode,
			paramLogTarget: target,
		}}); err != nil {
			return nil, err
		}
	} else if tableflowTopic.GetSpec().Config.GetErrorHandling().TableflowV1ErrorHandlingSkip != nil {
		if err := d.Set(paramErrorHandling, []interface{}{map[string]interface{}{
			paramMode:      errorHandlingSkipMode,
			paramLogTarget: target,
		}}); err != nil {
			return nil, err
		}
	} else if tableflowTopic.GetSpec().Config.GetErrorHandling().TableflowV1ErrorHandlingLog != nil {
		if err := d.Set(paramErrorHandling, []interface{}{map[string]interface{}{
			paramMode:      errorHandlingLogMode,
			paramLogTarget: tableflowTopic.GetSpec().Config.GetErrorHandling().TableflowV1ErrorHandlingLog.GetTarget(),
		}}); err != nil {
			return nil, err
		}
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.tableflowApiKey, c.tableflowApiSecret, d, false); err != nil {
			return nil, err
		}
	}

	d.SetId(tableflowTopic.Spec.GetDisplayName())
	return d, nil
}

func getStorageType(tableflowTopic tableflow.TableflowV1TableflowTopic) (string, error) {
	config := tableflowTopic.GetSpec().Storage

	if config.TableflowV1ByobAwsSpec != nil {
		return byobAwsSpecKind, nil
	}

	if config.TableflowV1ManagedStorageSpec != nil {
		return managedStorageSpecKind, nil
	}

	return "", fmt.Errorf("error reading storage type for Tableflow Topic %q", tableflowTopic.Spec.GetDisplayName())
}

func tableflowTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Tableflow Topic %q", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	req := tableflowRestClient.apiClient.TableflowTopicsTableflowV1Api.DeleteTableflowV1TableflowTopic(tableflowRestClient.apiContext(ctx), d.Id()).Environment(environmentId).SpecKafkaCluster(clusterId)
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Tableflow Topic %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Tableflow Topic %q", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})

	return nil
}

func tableflowTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramRetentionMs, paramTableFormats, paramRecordFailureStrategy, paramErrorHandling) {
		return diag.Errorf("error updating Tableflow Topic %q: only %q, %q, %q, %q, %q, %q attributes can be updated for Tableflow Topic", d.Id(), paramRetentionMs, paramTableFormats, paramRecordFailureStrategy, paramErrorHandling, paramMode, paramLogTarget)
	}

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	updateTableflowTopicSpec := tableflow.NewTableflowV1TableflowTopicSpecUpdate()
	updateTableflowTopicSpec.Config = tableflow.NewTableflowV1TableFlowTopicConfigsSpec()
	updateTableflowTopicSpec.SetEnvironment(tableflow.GlobalObjectReference{Id: environmentId})
	updateTableflowTopicSpec.SetKafkaCluster(tableflow.EnvScopedObjectReference{Id: clusterId})
	if d.HasChange(paramRetentionMs) {
		updateTableflowTopicSpec.Config.SetRetentionMs(d.Get(paramRetentionMs).(string))
	}
	if d.HasChange(paramTableFormats) {
		updateTableflowTopicSpec.SetTableFormats(convertToStringSlice(d.Get(paramTableFormats).(*schema.Set).List()))
	}
	if d.HasChange(paramRecordFailureStrategy) {
		updateTableflowTopicSpec.Config.SetRecordFailureStrategy(d.Get(paramRecordFailureStrategy).(string))
	}
	if d.HasChange(paramErrorHandling) {
		mode := extractStringValueFromBlock(d, paramErrorHandling, paramMode)
		target := extractStringValueFromBlock(d, paramErrorHandling, paramLogTarget)

		if strings.ToUpper(mode) == errorHandlingSuspendMode {
			updateTableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingSuspend: &tableflow.TableflowV1ErrorHandlingSuspend{
					Mode: errorHandlingSuspendMode,
				},
			})
		} else if strings.ToUpper(mode) == errorHandlingSkipMode {
			updateTableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingSkip: &tableflow.TableflowV1ErrorHandlingSkip{
					Mode: errorHandlingSkipMode,
				},
			})
		} else if strings.ToUpper(mode) == errorHandlingLogMode {
			updateTableflowTopicSpec.Config.SetErrorHandling(tableflow.TableflowV1TableFlowTopicConfigsSpecErrorHandlingOneOf{
				TableflowV1ErrorHandlingLog: &tableflow.TableflowV1ErrorHandlingLog{
					Mode:   errorHandlingLogMode,
					Target: tableflow.PtrString(target),
				},
			})
		}
	}

	updateTableflowTopic := tableflow.NewTableflowV1TableflowTopicUpdate()
	updateTableflowTopic.SetSpec(*updateTableflowTopicSpec)

	updateTableflowTopicJson, err := json.Marshal(updateTableflowTopic)
	if err != nil {
		return diag.Errorf("error updating Tableflow Topic %q: error marshaling %#v to json: %s", d.Id(), updateTableflowTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Tableflow Topic %q: %s", d.Id(), updateTableflowTopicJson), map[string]interface{}{tableflowTopicKey: d.Id()})

	req := tableflowRestClient.apiClient.TableflowTopicsTableflowV1Api.UpdateTableflowV1TableflowTopic(tableflowRestClient.apiContext(ctx), d.Id()).TableflowV1TableflowTopicUpdate(*updateTableflowTopic)
	updatedTableflowTopic, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error updating Tableflow Topic %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	UpdatedTableflowTopicJson, err := json.Marshal(updatedTableflowTopic)
	if err != nil {
		return diag.Errorf("error updating Tableflow Topic %q: error marshaling %#v to json: %s", d.Id(), updatedTableflowTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Tableflow Topic %q: %s", d.Id(), UpdatedTableflowTopicJson), map[string]interface{}{tableflowTopicKey: d.Id()})
	return tableflowTopicRead(ctx, d, meta)
}

func tableflowTopicImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Tableflow Topic %q", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, true)
	if err != nil {
		return nil, fmt.Errorf("error creating Tableflow Topic: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet, c.oauthToken, c.stsToken)

	envIDAndClusterIDAndTopicName := d.Id()
	parts := strings.Split(envIDAndClusterIDAndTopicName, "/")

	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Tableflow Topic: invalid format: expected '<env ID>/<Kafka cluster ID>/<topic name>'")
	}

	environmentId := parts[0]
	clusterId := parts[1]
	topicName := parts[2]
	d.SetId(topicName)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readTableflowTopicAndSetAttributes(ctx, d, tableflowRestClient, environmentId, clusterId, d.Id()); err != nil {
		return nil, fmt.Errorf("error importing Tableflow Topic %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Tableflow Topic %q", d.Id()), map[string]interface{}{tableflowTopicKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func extractTableflowApiKeyAndApiSecret(client *Client, d *schema.ResourceData, isImportOperation bool) (string, string, error) {
	if client.isTableflowMetadataSet {
		return client.tableflowApiKey, client.tableflowApiSecret, nil
	}
	if isImportOperation {
		tableflowApiKey := getEnv("IMPORT_TABLEFLOW_API_KEY", "")
		tableflowApiSecret := getEnv("IMPORT_TABLEFLOW_API_SECRET", "")
		if tableflowApiKey != "" && tableflowApiSecret != "" {
			return tableflowApiKey, tableflowApiSecret, nil
		} else {
			return "", "", fmt.Errorf("one of (provider.tableflow_api_key, provider.tableflow_api_secret), (TABLEFLOW_API_KEY, TABLEFLOW_API_SECRET environment variables) or (IMPORT_TABLEFLOW_API_KEY, IMPORT_TABLEFLOW_API_SECRET environment variables) must be set")
		}
	}
	tableflowApiKey, tableflowApiSecret := extractClusterApiKeyAndApiSecretFromCredentialsBlock(d)
	if tableflowApiKey != "" {
		return tableflowApiKey, tableflowApiSecret, nil
	}
	return "", "", fmt.Errorf("one of (provider.tableflow_api_key, provider.tableflow_api_secret), (TABLEFLOW_API_KEY, TABLEFLOW_API_SECRET environment variables) or (resource.credentials.key, resource.credentials.secret) must be set")
}
