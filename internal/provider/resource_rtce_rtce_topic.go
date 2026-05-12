// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	rtcev1 "github.com/confluentinc/ccloud-sdk-go-v2/rtce/v1"
)

// Timeout constants for async operations
const (
	rtceV1APICreateTimeout = 1 * time.Hour
	rtceV1APIDeleteTimeout = 1 * time.Hour
)

// State constants for async provisioning
const (
	statePENDING      = "PENDING"
	statePROVISIONING = "PROVISIONING"
	stateACTIVE       = "ACTIVE"
	stateFAILED       = "FAILED"
	stateUNAVAILABLE  = "UNAVAILABLE"
)

const rtceTopicLoggingKey = "rtce_topic_id"

func rtceTopic() *schema.Resource {
	return &schema.Resource{
		CreateContext: rtceTopicCreate,
		ReadContext:   rtceTopicRead,
		UpdateContext: rtceTopicUpdate,
		DeleteContext: rtceTopicDelete,
		Importer: &schema.ResourceImporter{
			StateContext: rtceTopicImport,
		},
		Schema: map[string]*schema.Schema{
			paramCloud: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cloud provider where the RTCE topic is deployed.",
			},
			paramDescription: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "A model-readable description of the RTCE topic.",
			},
			paramEnvironment:  environmentSchema(),
			paramKafkaCluster: kafkaClusterSchema(),
			paramRegion: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cloud region where the RTCE topic is deployed.",
			},
			paramTopicName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The Kafka topic name containing the data for the RTCE topic.",
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a resource.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object this REST resource represents.",
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the resource.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(rtceV1APICreateTimeout),
			Delete: schema.DefaultTimeout(rtceV1APIDeleteTimeout),
		},
	}
}
func createRtceTopicId(environmentId string, kafkaClusterId string, topicName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentId, kafkaClusterId, topicName)
}

func rtceTopicCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	spec := rtcev1.NewRtceV1RtceTopicSpec()

	// Set required attributes
	spec.SetCloud(d.Get(paramCloud).(string))
	spec.SetDescription(d.Get(paramDescription).(string))
	environmentRef := rtcev1.NewEnvScopedObjectReferenceWithDefaults()
	environmentRef.SetId(extractStringValueFromBlock(d, paramEnvironment, paramId))
	spec.SetEnvironment(*environmentRef)
	kafkaClusterRef := rtcev1.NewEnvScopedObjectReferenceWithDefaults()
	kafkaClusterRef.SetId(extractStringValueFromBlock(d, paramKafkaCluster, paramId))
	spec.SetKafkaCluster(*kafkaClusterRef)
	spec.SetRegion(d.Get(paramRegion).(string))
	spec.SetTopicName(d.Get(paramTopicName).(string))

	// Set optional attributes

	createRtceTopicRequest := &rtcev1.RtceV1RtceTopic{Spec: spec}

	// Logging
	createRtceTopicRequestJson, err := json.Marshal(createRtceTopicRequest)
	if err != nil {
		return diag.Errorf("error creating RtceTopic: error marshaling %#v to json: %s", createRtceTopicRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new RtceTopic: %s", createRtceTopicRequestJson))

	// Make API call
	createdRtceTopic, resp, err := executeRtceTopicCreate(c.rtceV1ApiContext(ctx), c, createRtceTopicRequest)
	if err != nil {
		return diag.Errorf("error creating RtceTopic: %s", createDescriptiveError(err, resp))
	}
	createEnvironmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	createKafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	createTopicName := d.Get(paramTopicName).(string)
	d.SetId(createRtceTopicId(createEnvironmentId, createKafkaClusterId, createTopicName))

	// Wait for async provisioning to complete
	if err := waitForRtceTopicToProvision(c.rtceV1ApiContext(ctx), c, createEnvironmentId, createKafkaClusterId, createTopicName); err != nil {
		return diag.Errorf("error waiting for RtceTopic %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	// Logging
	createdRtceTopicJson, err := json.Marshal(createdRtceTopic)
	if err != nil {
		return diag.Errorf("error creating RtceTopic %q: error marshaling %#v to json: %s", d.Id(), createdRtceTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating RtceTopic %q: %s", d.Id(), createdRtceTopicJson), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	return rtceTopicRead(ctx, d, meta)
}

func executeRtceTopicCreate(ctx context.Context, c *Client, rtceTopic *rtcev1.RtceV1RtceTopic) (rtcev1.RtceV1RtceTopic, *http.Response, error) {
	req := c.rtceV1Client.RtceTopicsRtceV1Api.CreateRtceV1RtceTopic(c.rtceV1ApiContext(ctx)).RtceV1RtceTopic(*rtceTopic)
	return req.Execute()
}

func rtceTopicRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	topicName := d.Get(paramTopicName).(string)
	rtceTopic, resp, err := executeRtceTopicRead(c.rtceV1ApiContext(ctx), c, environmentId, kafkaClusterId, topicName)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading RtceTopic %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing RtceTopic %q in TF state because RtceTopic could not be found on the server", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err, resp))
	}

	rtceTopicJson, err := json.Marshal(rtceTopic)
	if err != nil {
		return diag.Errorf("error reading RtceTopic %q: error marshaling %#v to json: %s", d.Id(), rtceTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched RtceTopic %q: %s", d.Id(), rtceTopicJson), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	if _, err := setRtceTopicAttributes(d, rtceTopic); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	return nil
}

func executeRtceTopicRead(ctx context.Context, c *Client, environmentId string, kafkaClusterId string, topicName string) (rtcev1.RtceV1RtceTopic, *http.Response, error) {
	req := c.rtceV1Client.RtceTopicsRtceV1Api.GetRtceV1RtceTopic(c.rtceV1ApiContext(ctx), topicName).Environment(environmentId).SpecKafkaCluster(kafkaClusterId)
	return req.Execute()
}

func setRtceTopicAttributes(d *schema.ResourceData, rtceTopic rtcev1.RtceV1RtceTopic) (*schema.ResourceData, error) {
	spec := rtceTopic.GetSpec()
	if err := d.Set(paramCloud, spec.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, spec.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	environmentRef := spec.GetEnvironment()
	if err := d.Set(paramEnvironment, []interface{}{map[string]interface{}{
		paramId: environmentRef.GetId(),
	}}); err != nil {
		return nil, createDescriptiveError(err)
	}
	kafkaClusterRef := spec.GetKafkaCluster()
	if err := d.Set(paramKafkaCluster, []interface{}{map[string]interface{}{
		paramId: kafkaClusterRef.GetId(),
	}}); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRegion, spec.GetRegion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramTopicName, spec.GetTopicName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, rtceTopic.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, rtceTopic.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, rtceTopic.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(createRtceTopicId(environmentRef.GetId(), kafkaClusterRef.GetId(), spec.GetTopicName()))
	return d, nil
}

func rtceTopicUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDescription) {
		return diag.Errorf("error updating RtceTopic %q: only %q can be updated", d.Id(), paramDescription)
	}

	updateRtceTopicRequest := rtcev1.NewRtceV1RtceTopicUpdate()
	specUpdate := rtcev1.NewRtceV1RtceTopicSpecUpdate()
	if d.HasChange(paramDescription) {
		specUpdate.SetDescription(d.Get(paramDescription).(string))
	}

	// Environment is required in the update request body for resource identification
	updateEnvironmentRef := rtcev1.NewEnvScopedObjectReferenceWithDefaults()
	updateEnvironmentRef.SetId(extractStringValueFromBlock(d, paramEnvironment, paramId))
	specUpdate.SetEnvironment(*updateEnvironmentRef)

	// KafkaCluster is required in the update request body for resource identification
	updateKafkaClusterRef := rtcev1.NewEnvScopedObjectReferenceWithDefaults()
	updateKafkaClusterRef.SetId(extractStringValueFromBlock(d, paramKafkaCluster, paramId))
	specUpdate.SetKafkaCluster(*updateKafkaClusterRef)

	updateRtceTopicRequest.SetSpec(*specUpdate)

	updateRtceTopicRequestJson, err := json.Marshal(updateRtceTopicRequest)
	if err != nil {
		return diag.Errorf("error updating RtceTopic %q: error marshaling %#v to json: %s", d.Id(), updateRtceTopicRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating RtceTopic %q: %s", d.Id(), updateRtceTopicRequestJson), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	c := meta.(*Client)
	topicName := d.Get(paramTopicName).(string)
	updatedRtceTopic, resp, err := c.rtceV1Client.RtceTopicsRtceV1Api.UpdateRtceV1RtceTopic(c.rtceV1ApiContext(ctx), topicName).RtceV1RtceTopicUpdate(*updateRtceTopicRequest).Execute()
	if err != nil {
		return diag.Errorf("error updating RtceTopic %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedRtceTopicJson, err := json.Marshal(updatedRtceTopic)
	if err != nil {
		return diag.Errorf("error updating RtceTopic %q: error marshaling %#v to json: %s", d.Id(), updatedRtceTopic, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating RtceTopic %q: %s", d.Id(), updatedRtceTopicJson), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	return rtceTopicRead(ctx, d, meta)
}

func rtceTopicDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	topicName := d.Get(paramTopicName).(string)
	c := meta.(*Client)
	req := c.rtceV1Client.RtceTopicsRtceV1Api.DeleteRtceV1RtceTopic(c.rtceV1ApiContext(ctx), topicName).Environment(environmentId).SpecKafkaCluster(kafkaClusterId)
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting RtceTopic %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	// Wait for async deletion to complete
	if err := waitForRtceTopicToBeDeleted(c.rtceV1ApiContext(ctx), c, environmentId, kafkaClusterId, topicName); err != nil {
		return diag.Errorf("error waiting for RtceTopic %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	return nil
}

func rtceTopicImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})

	importId := d.Id()
	parts := strings.Split(importId, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing RtceTopic: invalid format: expected '<environment ID>/<Kafka Cluster ID>/<topic_name>'")
	}
	environmentId := parts[0]
	kafkaClusterId := parts[1]
	topicName := parts[len(parts)-1]
	d.SetId(createRtceTopicId(environmentId, kafkaClusterId, topicName))
	if err := d.Set(paramEnvironment, []interface{}{map[string]interface{}{
		paramId: environmentId,
	}}); err != nil {
		return nil, err
	}
	if err := d.Set(paramKafkaCluster, []interface{}{map[string]interface{}{
		paramId: kafkaClusterId,
	}}); err != nil {
		return nil, err
	}
	if err := d.Set(paramTopicName, topicName); err != nil {
		return nil, err
	}

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := rtceTopicRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing RtceTopic %q: %s", d.Id(), diagnostics[0].Summary)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished importing RtceTopic %q", d.Id()), map[string]interface{}{rtceTopicLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

// rtceTopicProvisionStatus returns a StateRefreshFunc for polling the resource status
func rtceTopicProvisionStatus(ctx context.Context, c *Client, environmentId string, kafkaClusterId string, topicName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		rtceTopic, resp, err := executeRtceTopicRead(c.rtceV1ApiContext(ctx), c, environmentId, kafkaClusterId, topicName)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading RtceTopic %q: %s", topicName, createDescriptiveError(err, resp)), map[string]interface{}{rtceTopicLoggingKey: topicName})
			return nil, stateUnknown, err
		}

		currentPhase := rtceTopic.Status.GetPhase()
		tflog.Debug(ctx, fmt.Sprintf("Waiting for RtceTopic %q provisioning status to become one of %v: current status is %q", topicName, []string{stateACTIVE}, currentPhase), map[string]interface{}{rtceTopicLoggingKey: topicName})

		// Check for pending states (still provisioning)
		switch currentPhase {
		case statePENDING:
			return rtceTopic, currentPhase, nil
		case statePROVISIONING:
			return rtceTopic, currentPhase, nil
		}

		// Check for target states (provisioning complete)
		switch currentPhase {
		case stateACTIVE:
			return rtceTopic, currentPhase, nil
		}

		// Check for failed states
		switch currentPhase {
		case stateFAILED:
			errorMsg := rtceTopic.Status.GetErrorMessage()
			if errorMsg == "" {
				statusBytes, marshalErr := json.Marshal(rtceTopic.Status)
				if marshalErr == nil && len(statusBytes) > 0 {
					errorMsg = fmt.Sprintf("no error_message provided by the API; status: %s", statusBytes)
				} else {
					errorMsg = "no error_message provided by the API"
				}
			}
			return nil, stateFAILED, fmt.Errorf("RtceTopic %q provisioning status is %q: %s", topicName, stateFAILED, errorMsg)
		case stateUNAVAILABLE:
			errorMsg := rtceTopic.Status.GetErrorMessage()
			if errorMsg == "" {
				statusBytes, marshalErr := json.Marshal(rtceTopic.Status)
				if marshalErr == nil && len(statusBytes) > 0 {
					errorMsg = fmt.Sprintf("no error_message provided by the API; status: %s", statusBytes)
				} else {
					errorMsg = "no error_message provided by the API"
				}
			}
			return nil, stateUNAVAILABLE, fmt.Errorf("RtceTopic %q provisioning status is %q: %s", topicName, stateUNAVAILABLE, errorMsg)
		}

		// Unexpected state
		return nil, stateUnexpected, fmt.Errorf("RtceTopic %q is in an unexpected state %q", topicName, currentPhase)
	}
}

// waitForRtceTopicToProvision waits for the resource to reach a ready state
func waitForRtceTopicToProvision(ctx context.Context, c *Client, environmentId string, kafkaClusterId string, topicName string) error {
	delay, pollInterval := getDelayAndPollInterval(5*time.Second, 1*time.Minute, c.isAcceptanceTestMode)
	stateConf := &resource.StateChangeConf{
		Pending:      []string{statePENDING, statePROVISIONING},
		Target:       []string{stateACTIVE},
		Refresh:      rtceTopicProvisionStatus(ctx, c, environmentId, kafkaClusterId, topicName),
		Timeout:      rtceV1APICreateTimeout,
		Delay:        delay,
		PollInterval: pollInterval,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for RtceTopic %q provisioning status to become one of %v", topicName, []string{stateACTIVE}), map[string]interface{}{rtceTopicLoggingKey: topicName})
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return err
	}
	return nil
}

// rtceTopicDeleteStatus returns a StateRefreshFunc for polling the delete status
func rtceTopicDeleteStatus(ctx context.Context, c *Client, environmentId string, kafkaClusterId string, topicName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		rtceTopic, resp, err := executeRtceTopicRead(c.rtceV1ApiContext(ctx), c, environmentId, kafkaClusterId, topicName)
		if err != nil {
			if isNonKafkaRestApiResourceNotFound(resp) {
				// Resource is deleted
				return rtceTopic, "DELETED", nil
			}
			tflog.Warn(ctx, fmt.Sprintf("Error reading RtceTopic %q: %s", topicName, createDescriptiveError(err, resp)), map[string]interface{}{rtceTopicLoggingKey: topicName})
			return nil, stateUnknown, err
		}

		// Resource still exists
		return rtceTopic, "DELETING", nil
	}
}

// waitForRtceTopicToBeDeleted waits for the resource to be fully deleted
func waitForRtceTopicToBeDeleted(ctx context.Context, c *Client, environmentId string, kafkaClusterId string, topicName string) error {
	delay, pollInterval := getDelayAndPollInterval(5*time.Second, 1*time.Minute, c.isAcceptanceTestMode)
	stateConf := &resource.StateChangeConf{
		Pending:      []string{"DELETING"},
		Target:       []string{"DELETED"},
		Refresh:      rtceTopicDeleteStatus(ctx, c, environmentId, kafkaClusterId, topicName),
		Timeout:      rtceV1APIDeleteTimeout,
		Delay:        delay,
		PollInterval: pollInterval,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for RtceTopic %q to be deleted", topicName), map[string]interface{}{rtceTopicLoggingKey: topicName})
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return err
	}
	return nil
}
func kafkaClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the kafka_cluster.",
				},
			},
		},
		Required:    true,
		MinItems:    1,
		MaxItems:    1,
		ForceNew:    true,
		Description: "The Kafka cluster containing the topic to be materialized.",
	}
}
