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
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	connect "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
)

const (
	connectAPICreateTimeout        = 24 * time.Hour
	connectOffsetsAPIUpdateTimeout = 1 * time.Hour
	connectAPIWaitAfterCreate      = 5 * time.Second

	paramSensitiveConfig    = "config_sensitive"
	paramNonSensitiveConfig = "config_nonsensitive"
	paramOffsetsConfig      = "offsets"
	paramPartition          = "partition"
	paramOffset             = "offset"

	connectorConfigAttributeName   = "name"
	connectorConfigAttributeClass  = "connector.class"
	connectorConfigAttributeType   = "confluent.connector.type"
	connectorConfigAttributePlugin = "confluent.custom.plugin.id"
	connectorTypeManaged           = "MANAGED"
	connectorTypeCustom            = "CUSTOM"

	connectorConfigInternalAttributePrefix = "config.internal."

	twoStarsOrMorePattern = "^[*]{2,}"

	paramStatus   = "status"
	statePaused   = "PAUSED"
	stateDegraded = "DEGRADED"

	stateApplied = "APPLIED"
)

var connectorConfigFullAttributeName = fmt.Sprintf("%s.name", paramNonSensitiveConfig)
var ignoredConnectorConfigs = []string{
	"cloud.environment",
	"cloud.provider",
	"connector.crn",
	"kafka.endpoint",
	"kafka.max.partition.validation.disable",
	"kafka.region",
	"kafka.dedicated",
	"schema.registry.url",
	"valid.kafka.api.key",
	"confluent.custom.plugin.type",
	"confluent.resource.connector.tier",
}
var twoStarsOrMoreRegExp = regexp.MustCompile(twoStarsOrMorePattern)

func connectorResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: connectorCreate,
		ReadContext:   connectorRead,
		UpdateContext: connectorUpdate,
		DeleteContext: connectorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: connectorImport,
		},
		Schema: map[string]*schema.Schema{
			paramEnvironment:  environmentSchema(),
			paramKafkaCluster: requiredKafkaClusterBlockSchema(),
			paramStatus: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramNonSensitiveConfig: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "The nonsensitive configuration settings to set (e.g., `\"time.interval\" = \"DAILY\"`).",
			},
			paramSensitiveConfig: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Sensitive:   true,
				Optional:    true,
				Computed:    true,
				ForceNew:    false,
				Description: "The sensitive configuration settings to set (e.g., `\"gcs.credentials.config\" = \"**REDACTED***\"`). Should not be set for an import operation.",
			},
			paramOffsetsConfig: offsetsSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(connectAPICreateTimeout),
		},
	}
}

func offsetsSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeSet,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPartition: {
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required:    true,
					Description: "Connector partitions information.",
				},
				paramOffset: {
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required:    true,
					Description: "Connector offsets information for the partition.",
				},
			},
		},
		Optional:    true,
		Computed:    false,
		ForceNew:    false,
		Description: "Connector partitions with offsets",
	}
}

func connectorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	mergedConfig, sensitiveConfig, nonsensitiveConfig := extractConnectorConfigs(d)
	offsets := extractConnectorOffsets(d)
	displayName := d.Get(connectorConfigFullAttributeName).(string)
	if displayName == "" {
		return diag.Errorf("error creating Connector: %q attribute is missing in %q block", connectorConfigAttributeName, paramNonSensitiveConfig)
	}
	createConnectorRequest := connect.NewInlineObject()
	createConnectorRequest.SetName(displayName)
	createConnectorRequest.SetConfig(mergedConfig)
	if len(offsets) > 0 {
		createConnectorRequest.SetOffsets(offsets)
		offsetsJson, err := json.Marshal(offsets)
		if err != nil {
			return diag.Errorf("error creating Connector: error marshaling offset %#v to json: %s", offsets, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Creating new Connector with custom offsets: %s", offsetsJson))
	}

	nonsensitiveConfigJson, err := json.Marshal(nonsensitiveConfig)
	if err != nil {
		return diag.Errorf("error creating Connector: error marshaling config %#v to json: %s", nonsensitiveConfig, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Connector: %s", nonsensitiveConfigJson))
	err = validateConnectorConfig(c.connectApiContext(ctx), c, mergedConfig, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error creating Connector: %s", createDescriptiveError(err))
	}

	createdConnector, resp, err := executeConnectorCreate(c.connectApiContext(ctx), c, environmentId, clusterId, createConnectorRequest)
	if err != nil {
		descriptiveError, _ := io.ReadAll(resp.Body)
		return diag.Errorf("error creating Connector %q: %s: %s", displayName, createDescriptiveError(err), string(descriptiveError))
	}
	// There's no ID attribute in createdConnector, so we have to send another request to a different endpoint to get a connector object with ID attribute
	SleepIfNotTestMode(connectAPIWaitAfterCreate, meta.(*Client).isAcceptanceTestMode)
	createdConnectorWithId, _, err := executeConnectorRead(c.connectApiContext(ctx), c, displayName, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error creating Connector %q: error reading created Connector: %s", displayName, createDescriptiveError(err))
	}
	d.SetId(createdConnectorWithId.Id.GetId())

	if err := waitForConnectorToProvision(c.connectApiContext(ctx), c, displayName, environmentId, clusterId); err != nil {
		return diag.Errorf("error waiting for Connector %q to provision: %s", displayName, createDescriptiveError(err))
	}

	_, err = json.Marshal(createdConnector)
	if err != nil {
		return diag.Errorf("error creating Connector %q: error marshaling %#v to json: %s", d.Id(), createdConnector, createDescriptiveError(err))
	}

	// Save sensitive configs
	if err := d.Set(paramSensitiveConfig, sensitiveConfig); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Connector %q", displayName))

	// We don't save offsets in connectorRead because the backend can arbitrarily change them.
	// You can create a connector without offsets—initially, initially the backend returns "offsets": [],
	// but after some time, it may return a non-empty list of offsets.
	return connectorRead(ctx, d, meta)
}

func validateConnectorConfig(ctx context.Context, c *Client, config map[string]string, environmentId, clusterId string) error {
	// defaults to MANAGED
	connectorType := config[connectorConfigAttributeType]

	if connectorType == "" || connectorType == connectorTypeManaged {
		// connectorConfigAttributeClass is required for managed connectors
		connectorClass := config[connectorConfigAttributeClass]
		if connectorClass == "" {
			return fmt.Errorf("error validating Connector config: %q attribute is missing in %q block", connectorConfigAttributeClass, paramNonSensitiveConfig)
		}
		tflog.Debug(ctx, "Validating new Connector's config")
		validationResponse, _, err := c.connectClient.ManagedConnectorPluginsConnectV1Api.ValidateConnectv1ConnectorPlugin(c.connectApiContext(ctx), connectorClass, environmentId, clusterId).RequestBody(config).Execute()
		if err != nil {
			return fmt.Errorf("error creating Connector: error sending validation request: %s", createDescriptiveError(err))
		}
		if validationResponse.GetErrorCount() > 0 {
			return fmt.Errorf("error creating Connector %q: error validating config: %s", connectorClass, createDescriptiveError(createConfigValidationError(validationResponse)))
		}
	} else if connectorType == connectorTypeCustom {
		// connectorConfigAttributePlugin is required for custom connectors
		if _, ok := config[connectorConfigAttributePlugin]; !ok {
			return fmt.Errorf("error validating Connector config: %q attribute is missing in %q block", connectorConfigAttributePlugin, paramNonSensitiveConfig)
		}
	} else {
		return fmt.Errorf("error validating Connector config: unexpected value for %s: %s", connectorConfigAttributeType, connectorType)
	}
	return nil
}

func executeConnectorCreate(ctx context.Context, c *Client, environmentId, clusterId string, spec *connect.InlineObject) (connect.ConnectV1ConnectorWithOffsets, *http.Response, error) {
	req := c.connectClient.ConnectorsConnectV1Api.CreateConnectv1Connector(c.connectApiContext(ctx), environmentId, clusterId).InlineObject(*spec)
	return req.Execute()
}

func executeConnectorStatusCreate(ctx context.Context, c *Client, displayName, environmentId, clusterId string) (connect.InlineResponse2001, *http.Response, error) {
	req := c.connectClient.StatusConnectV1Api.ReadConnectv1ConnectorStatus(c.connectApiContext(ctx), displayName, environmentId, clusterId)
	return req.Execute()
}

func executeConnectorRead(ctx context.Context, c *Client, displayName, environmentId, clusterId string) (connect.ConnectV1ConnectorExpansion, *http.Response, error) {
	connectors, resp, err := c.connectClient.ConnectorsConnectV1Api.ListConnectv1ConnectorsWithExpansions(c.connectApiContext(ctx), environmentId, clusterId).Execute()
	if ResponseHasExpectedStatusCode(resp, http.StatusForbidden) {
		return *connect.NewConnectV1ConnectorExpansionWithDefaults(), resp, err
	}
	if err != nil {
		return *connect.NewConnectV1ConnectorExpansionWithDefaults(), resp, createDescriptiveError(err)
	}
	// Find the target connector in a list of connectors by its name
	if connector, ok := connectors[displayName]; ok {
		return connector, resp, nil
	}

	return *connect.NewConnectV1ConnectorExpansionWithDefaults(), &http.Response{StatusCode: http.StatusNotFound}, fmt.Errorf("connector %q was not found", displayName)
}

func connectorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	displayName := d.Get(connectorConfigFullAttributeName).(string)
	if displayName == "" {
		return diag.Errorf("error reading Connector: %q attribute is missing in %q block", connectorConfigAttributeName, paramNonSensitiveConfig)
	}
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Connector %q", displayName))

	if _, err := readConnectorAndSetAttributes(ctx, d, meta, displayName, environmentId, clusterId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Connector %q: %s", displayName, createDescriptiveError(err)))
	}

	return nil
}

func readConnectorAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, displayName, environmentId, clusterId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	connector, resp, err := executeConnectorRead(c.connectApiContext(ctx), c, displayName, environmentId, clusterId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Connector %q: %s", d.Id(), createDescriptiveError(err)))
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Connector %q in TF state because Connector could not be found on the server", d.Id()))
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	connectorJson, err := json.Marshal(connector)
	if err != nil {
		return nil, fmt.Errorf("error reading Connector %q: error marshaling %#v to json: %s", displayName, connector, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Connector %q: %s", displayName, connectorJson))

	if _, err := setConnectorAttributes(d, connector, environmentId, clusterId); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setConnectorAttributes(d *schema.ResourceData, connector connect.ConnectV1ConnectorExpansion, environmentId, clusterId string) (*schema.ResourceData, error) {
	// paramSensitiveConfig is set in connectorCreate()
	config := connector.Info.GetConfig()
	status := connector.Status.GetConnector()
	if err := d.Set(paramNonSensitiveConfig, extractNonsensitiveConfigs(config)); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, environmentId, d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, clusterId, d); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatus, status.GetState()); err != nil {
		return nil, err
	}
	d.SetId(connector.Id.GetId())
	return d, nil
}

func connectorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramNonSensitiveConfig, paramSensitiveConfig, paramOffsetsConfig, paramStatus) {
		return diag.Errorf("error updating Connector %q: only %q attribute, %q, %q and %q blocks can be updated for Connector", d.Id(), paramStatus, paramOffsetsConfig, paramNonSensitiveConfig, paramSensitiveConfig)
	}
	c := meta.(*Client)
	if d.HasChange(connectorConfigFullAttributeName) {
		oldValue, _ := d.GetChange(connectorConfigFullAttributeName)
		// Reset the name in TF state to avoid accidental creation during the next 'terraform plan' run
		_, _, nonsensitiveUpdatedConfig := extractConnectorConfigs(d)
		nonsensitiveUpdatedConfig[connectorConfigAttributeName] = oldValue.(string)
		_ = d.Set(paramNonSensitiveConfig, nonsensitiveUpdatedConfig)
		return diag.Errorf("error updating Connector %q: %q attribute cannot be updated", d.Id(), connectorConfigAttributeName)
	}
	displayName := d.Get(connectorConfigFullAttributeName).(string)
	if displayName == "" {
		return diag.Errorf("error updating Connector %q: %q attribute is missing in %q block", d.Id(), connectorConfigAttributeName, paramNonSensitiveConfig)
	}
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	if d.HasChange(paramStatus) {
		oldValue, newValue := d.GetChange(paramStatus)
		oldStatus := oldValue.(string)
		newStatus := newValue.(string)
		shouldPauseConnector := (oldStatus == stateRunning) && (newStatus == statePaused)
		shouldResumeConnector := (oldStatus == statePaused) && (newStatus == stateRunning)
		if shouldPauseConnector {
			tflog.Debug(ctx, fmt.Sprintf("Pausing Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})

			req := c.connectClient.LifecycleConnectV1Api.PauseConnectv1Connector(c.connectApiContext(ctx), displayName, environmentId, clusterId)
			_, err := req.Execute()
			if err != nil {
				return diag.Errorf("error updating Connector %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForConnectorToChangeStatus(c.connectApiContext(ctx), c, displayName, environmentId, clusterId, stateRunning, statePaused); err != nil {
				return diag.Errorf("error waiting for Connector %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else if shouldResumeConnector {
			tflog.Debug(ctx, fmt.Sprintf("Resuming Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})

			req := c.connectClient.LifecycleConnectV1Api.ResumeConnectv1Connector(c.connectApiContext(ctx), displayName, environmentId, clusterId)
			_, err := req.Execute()
			if err != nil {
				return diag.Errorf("error updating Connector %q: %s", d.Id(), createDescriptiveError(err))
			}
			if err := waitForConnectorToChangeStatus(c.connectApiContext(ctx), c, displayName, environmentId, clusterId, statePaused, stateRunning); err != nil {
				return diag.Errorf("error waiting for Connector %q to be updated: %s", d.Id(), createDescriptiveError(err))
			}
		} else {
			return diag.Errorf("error updating Connector %q: only %q->%q or %q->%q transitions are supported but %q->%q was attempted", d.Id(), statePaused, stateRunning, stateRunning, statePaused, oldStatus, newStatus)
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})
	}
	if d.HasChanges(paramNonSensitiveConfig, paramSensitiveConfig) {
		// Update doesn't require secret topic configuration values to be set
		updatedConfig, _, nonsensitiveUpdatedConfig := extractConnectorConfigs(d)

		debugUpdatedConfigJson, err := json.Marshal(nonsensitiveUpdatedConfig)
		if err != nil {
			return diag.Errorf("error updating Connector: error marshaling config %#v to json: %s", nonsensitiveUpdatedConfig, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Connector: %s", debugUpdatedConfigJson))

		req := c.connectClient.ConnectorsConnectV1Api.CreateOrUpdateConnectv1ConnectorConfig(c.connectApiContext(ctx), displayName, environmentId, clusterId).RequestBody(updatedConfig)
		updatedConnector, resp, err := req.Execute()

		// Delete once APIF-2634 is resolved
		if resp != nil && resp.StatusCode != http.StatusOK {
			return diag.Errorf("error updating Connector %q: %s", d.Id(), resp.Status)
		}
		if err != nil {
			return diag.Errorf("error updating Connector %q: %s", d.Id(), createDescriptiveError(err))
		}

		updatedConnectorJson, err := json.Marshal(updatedConnector)
		if err != nil {
			return diag.Errorf("error updating Connector %q: error marshaling %#v to json: %s", d.Id(), updatedConnector, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Connector %q: %s", d.Id(), updatedConnectorJson), map[string]interface{}{connectorLoggingKey: d.Id()})
	}
	if d.HasChanges(paramOffsetsConfig) {
		oldValue, newValue := d.GetChange(paramOffsetsConfig)
		tflog.Debug(ctx, fmt.Sprintf("Updating Connector %q offsets from %#v to %#v", d.Id(), oldValue, newValue))
		newOffsets := extractConnectorOffsets(d)
		var connectV1AlterOffsetRequest connect.ConnectV1AlterOffsetRequest
		if len(newOffsets) == 0 {
			tflog.Debug(ctx, fmt.Sprintf("Resetting offsets for Connector %q", d.Id()))
			connectV1AlterOffsetRequest.SetType(connect.DELETE)
		} else {
			connectV1AlterOffsetRequest.SetType(connect.PATCH)
			connectV1AlterOffsetRequest.SetOffsets(newOffsets)
			debugUpdatedOffsetsJson, err := json.Marshal(newOffsets)
			if err != nil {
				return diag.Errorf("error updating Connector %q: error marshaling offsets %#v to json: %s", d.Id(), newOffsets, createDescriptiveError(err))
			}
			tflog.Debug(ctx, fmt.Sprintf("Updating Connector %q offsets: %s", d.Id(), debugUpdatedOffsetsJson))
		}

		req := c.connectClient.OffsetsConnectV1Api.AlterConnectv1ConnectorOffsetsRequest(c.connectApiContext(ctx), displayName, environmentId, clusterId).ConnectV1AlterOffsetRequest(connectV1AlterOffsetRequest)
		updatedConnectorOffsets, resp, err := req.Execute()
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			return diag.Errorf("error updating Connector %q offsets: %s: %s", d.Id(), createDescriptiveError(err), string(body))
		}
		if err := waitForConnectorOffsetsUpdateToComplete(c.connectApiContext(ctx), c, environmentId, clusterId, displayName); err != nil {
			return diag.Errorf("error waiting for Connector %q offsets update to complete: %s", d.Id(), createDescriptiveError(err))
		}

		updatedConnectorOffsetsJson, err := json.Marshal(updatedConnectorOffsets)
		if err != nil {
			return diag.Errorf("error updating Connector %q offsets: error marshaling %#v to json: %s", d.Id(), updatedConnectorOffsets, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating Connector %q offsets : %s", d.Id(), updatedConnectorOffsetsJson), map[string]interface{}{connectorLoggingKey: d.Id()})
	}

	return connectorRead(ctx, d, meta)
}

func connectorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})
	displayName := d.Get(connectorConfigFullAttributeName).(string)
	if displayName == "" {
		return diag.Errorf("error deleting Connector %q: %q attribute is missing in %q block", d.Id(), connectorConfigAttributeName, paramNonSensitiveConfig)
	}
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	c := meta.(*Client)

	req := c.connectClient.ConnectorsConnectV1Api.DeleteConnectv1Connector(c.connectApiContext(ctx), displayName, environmentId, clusterId)
	deletionError, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Connector %q: %s", d.Id(), createDescriptiveError(err))
	}
	if deletionError.Error != nil {
		return diag.Errorf("error deleting Connector %q: %q", d.Id(), deletionError.GetError())
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})

	return nil
}

func connectorImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Connector %q", d.Id()))

	envIDAndClusterIDAndConnectorName := d.Id()
	parts := strings.Split(envIDAndClusterIDAndConnectorName, "/")

	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Connector: invalid format: expected '<env ID>/<Kafka cluster ID>/<connector name>'")
	}

	environmentId := parts[0]
	clusterId := parts[1]
	connectorName := parts[2]

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readConnectorAndSetAttributes(ctx, d, meta, connectorName, environmentId, clusterId); err != nil {
		return nil, fmt.Errorf("error importing Connector %q: %s", d.Id(), createDescriptiveError(err))
	}
	if err := d.Set(paramSensitiveConfig, make(map[string]string)); err != nil {
		return nil, createDescriptiveError(err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Connector %q", d.Id()), map[string]interface{}{connectorLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func extractNonsensitiveConfigs(configs map[string]string) map[string]string {
	nonsensitiveConfigs := make(map[string]string)

	for configurationSettingName, configurationSettingValue := range configs {
		// Skip all sensitive config settings since we don't want to store them in TF state
		isSensitiveSetting := twoStarsOrMoreRegExp.MatchString(configurationSettingValue)
		if isSensitiveSetting {
			continue
		}

		// Skip internal configs
		isInternalSetting := stringInSlice(configurationSettingName, ignoredConnectorConfigs, false) ||
			strings.HasPrefix(configurationSettingName, connectorConfigInternalAttributePrefix)
		if isInternalSetting {
			continue
		}

		nonsensitiveConfigs[configurationSettingName] = configurationSettingValue
	}

	return nonsensitiveConfigs
}

func createConfigValidationError(validationResponse connect.InlineResponse2003) error {
	var configValidationErrors strings.Builder
	idx := 1
	for _, config := range validationResponse.GetConfigs() {
		if len(config.Value.GetErrors()) > 0 {
			configValidationErrors.WriteString(fmt.Sprintf("\n%d. %q : %q", idx, config.Value.GetName(), config.Value.GetErrors()))
			idx += 1
		}
	}
	if configValidationErrors.Len() > 0 {
		return errors.New(configValidationErrors.String())
	}
	return nil
}

// extractRequiredStringValueFromMap() returns the string for the given key, or the error key doesn't exist in the configuration.
func extractRequiredStringValueFromMap(config map[string]string, key, configName string) (string, error) {
	if value, ok := config[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("%q does not exist in %q", key, configName)
}

func extractConnectorConfigs(d *schema.ResourceData) (map[string]string, map[string]string, map[string]string) {
	sensitiveConfigs := convertToStringStringMap(d.Get(paramSensitiveConfig).(map[string]interface{}))
	nonsensitiveConfigs := convertToStringStringMap(d.Get(paramNonSensitiveConfig).(map[string]interface{}))

	// Merge both configs
	config := lo.Assign(
		nonsensitiveConfigs,
		sensitiveConfigs,
	)

	return config, sensitiveConfigs, nonsensitiveConfigs
}

// extractConnectorOffsets returns an array of map with Offsets and Partitions
func extractConnectorOffsets(d *schema.ResourceData) []map[string]interface{} {
	offsets := d.Get(paramOffsetsConfig).(*schema.Set).List()
	result := make([]map[string]interface{}, 0, len(offsets))

	for _, v := range offsets {
		valueMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		partitionRaw := valueMap[paramPartition]
		offsetRaw := valueMap[paramOffset]

		partitionMap, _ := partitionRaw.(map[string]interface{})
		offsetMap, _ := offsetRaw.(map[string]interface{})

		// https://github.com/hashicorp/terraform-plugin-sdk/pull/1042
		// https://discuss.hashicorp.com/t/using-typeset-in-provider-always-adds-an-empty-element-on-update/18566/6
		if len(partitionMap) == 0 && len(offsetMap) == 0 {
			continue
		}

		result = append(result, map[string]interface{}{
			paramPartition: partitionMap,
			paramOffset:    offsetMap,
		})
	}
	return result
}

func connectorImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllConnectors,
	}
}

func loadAllConnectors(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	for _, environment := range environments {
		kafkaClusters, err := loadKafkaClusters(ctx, client, environment.GetId())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Clusters in Environment %q: %s", environment.GetId(), createDescriptiveError(err)))
			return instances, diag.FromErr(createDescriptiveError(err))
		}
		for _, kafkaCluster := range kafkaClusters {
			connectorNames, err := loadConnectorsByEnvironmentIdAndKafkaClusterId(ctx, client, environment.GetId(), kafkaCluster.GetId())
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Error reading Connectors in Environment %q and Kafka Cluster %q: %s", environment.GetId(), kafkaCluster.GetId(), createDescriptiveError(err)))
				return instances, diag.FromErr(createDescriptiveError(err))
			}
			connectorNamesJson, err := json.Marshal(connectorNames)
			if err != nil {
				return instances, diag.Errorf("error reading Connectors in Environment %q and Kafka Cluster %q: error marshaling %#v to json: %s", environment.GetId(), kafkaCluster.GetId(), connectorNames, createDescriptiveError(err))
			}
			tflog.Debug(ctx, fmt.Sprintf("Fetched Connectors in Environment %q and Kafka Cluster %q: %s", environment.GetId(), kafkaCluster.GetId(), connectorNamesJson))

			for _, connectorName := range connectorNames {
				instanceId := fmt.Sprintf("%s/%s/%s", environment.GetId(), kafkaCluster.GetId(), connectorName)
				instances[instanceId] = toValidTerraformResourceName(connectorName)
			}
		}
	}
	return instances, nil
}

func loadConnectorsByEnvironmentIdAndKafkaClusterId(ctx context.Context, c *Client, environmentId, kafkaClusterId string) ([]string, error) {
	connectors, resp, err := c.connectClient.ConnectorsConnectV1Api.ListConnectv1Connectors(c.connectApiContext(ctx), environmentId, kafkaClusterId).Execute()
	// Somehow Connect SDK returns response.StatusCode == http.StatusForbidden but err is nil.
	if ResponseHasExpectedStatusCode(resp, http.StatusForbidden) || err != nil {
		return nil, createDescriptiveError(err)
	}
	return connectors, nil
}
