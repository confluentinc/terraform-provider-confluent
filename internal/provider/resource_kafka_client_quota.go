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
	quotas "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"time"
)

const (
	paramThroughput      = "throughput"
	paramIngressByteRate = "ingress_byte_rate"
	paramEgressByteRate  = "egress_byte_rate"
	paramPrincipals      = "principals"

	kafkaQuotasAPIWaitAfterCreate = 30 * time.Second
)

var attributeIngressByteRate = fmt.Sprintf("%s.0.%s", paramThroughput, paramIngressByteRate)
var attributeEgressByteRate = fmt.Sprintf("%s.0.%s", paramThroughput, paramEgressByteRate)

func kafkaClientQuotaResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaClientQuotaCreate,
		ReadContext:   kafkaClientQuotaRead,
		UpdateContext: kafkaClientQuotaUpdate,
		DeleteContext: kafkaClientQuotaDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaClientQuotaImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Kafka Client Quota.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "A description of the Kafka Client Quota.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramKafkaCluster: kafkaClusterBlockSchema(),
			paramEnvironment:  environmentSchema(),
			paramPrincipals: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				MinItems:    1,
				Required:    true,
				Description: "A list of service accounts. Special name \"default\" can be used to represent the default quota for all users and service accounts.",
			},
			paramThroughput: throughputSchema(),
		},
	}
}

func kafkaClientQuotaUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramPrincipals, paramThroughput) {
		return diag.Errorf("error updating Kafka Client Quota %q: only %q, %q, %q, %q attributes can be updated for Kafka Client Quota", d.Id(), paramDisplayName, paramDescription, paramPrincipals, paramThroughput)
	}

	updateKafkaClientQuotaRequest := quotas.NewKafkaQuotasV1ClientQuotaUpdate()
	updateSpec := quotas.NewKafkaQuotasV1ClientQuotaSpecUpdate()

	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateSpec.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updateSpec.SetDescription(updatedDescription)
	}
	if d.HasChange(paramPrincipals) {
		updatedPrincipals := convertSetToStringList(d, paramPrincipals)
		updateSpec.SetPrincipals(convertToGlobalObjectReferences(updatedPrincipals))
	}
	if d.HasChange(paramThroughput) {
		updatedIngressByteRate := d.Get(attributeIngressByteRate).(string)
		updatedEgressByteRate := d.Get(attributeEgressByteRate).(string)
		updatedThroughput := quotas.NewKafkaQuotasV1Throughput(updatedIngressByteRate, updatedEgressByteRate)
		updateSpec.SetThroughput(*updatedThroughput)
	}

	updateKafkaClientQuotaRequest.SetSpec(*updateSpec)
	updateKafkaClientQuotaRequestJson, err := json.Marshal(updateKafkaClientQuotaRequest)
	if err != nil {
		return diag.Errorf("error updating Kafka Client Quota %q: error marshaling %#v to json: %s", d.Id(), updateKafkaClientQuotaRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Client Quota %q: %s", d.Id(), updateKafkaClientQuotaRequestJson), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedClientQuota, _, err := c.quotasClient.ClientQuotasKafkaQuotasV1Api.UpdateKafkaQuotasV1ClientQuota(c.quotasApiContext(ctx), d.Id()).KafkaQuotasV1ClientQuotaUpdate(*updateKafkaClientQuotaRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Kafka Client Quota %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedClientQuotaJson, err := json.Marshal(updatedClientQuota)
	if err != nil {
		return diag.Errorf("error updating Kafka Client Quota %q: error marshaling %#v to json: %s", d.Id(), updatedClientQuota, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Client Quota %q: %s", d.Id(), updatedClientQuotaJson), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	return kafkaClientQuotaRead(ctx, d, meta)
}

func kafkaClientQuotaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	ingressByteRate := extractStringValueFromBlock(d, paramThroughput, paramIngressByteRate)
	egressByteRate := extractStringValueFromBlock(d, paramThroughput, paramEgressByteRate)

	globalObjectReferencePrincipals := convertToGlobalObjectReferences(convertSetToStringList(d, paramPrincipals))

	spec := quotas.NewKafkaQuotasV1ClientQuotaSpec()
	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetCluster(quotas.EnvScopedObjectReference{Id: kafkaClusterId})
	spec.SetEnvironment(quotas.GlobalObjectReference{Id: environmentId})
	spec.SetPrincipals(globalObjectReferencePrincipals)
	spec.SetThroughput(quotas.KafkaQuotasV1Throughput{IngressByteRate: ingressByteRate, EgressByteRate: egressByteRate})

	createKafkaClientQuotaRequest := quotas.KafkaQuotasV1ClientQuota{Spec: spec}
	createKafkaClientQuotaRequestJson, err := json.Marshal(createKafkaClientQuotaRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Client Quota: error marshaling %#v to json: %s", createKafkaClientQuotaRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka Client Quota: %s", createKafkaClientQuotaRequestJson))

	createdKafkaClientQuota, _, err := executeKafkaClientQuotaCreate(c.quotasApiContext(ctx), c, createKafkaClientQuotaRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Client Quota: %s", createDescriptiveError(err))
	}
	d.SetId(createdKafkaClientQuota.GetId())

	time.Sleep(kafkaQuotasAPIWaitAfterCreate)

	createdClientQuotaJson, err := json.Marshal(createdKafkaClientQuota)
	if err != nil {
		return diag.Errorf("error creating Kafka Client Quota: %q: error marshaling %#v to json: %s", createdKafkaClientQuota.GetId(), createdKafkaClientQuota, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka Client Quota %q: %s", d.Id(), createdClientQuotaJson), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	return kafkaClientQuotaRead(ctx, d, meta)
}

func executeKafkaClientQuotaCreate(ctx context.Context, c *Client, kafkaClientQuota quotas.KafkaQuotasV1ClientQuota) (quotas.KafkaQuotasV1ClientQuota, *http.Response, error) {
	req := c.quotasClient.ClientQuotasKafkaQuotasV1Api.CreateKafkaQuotasV1ClientQuota(c.quotasApiContext(ctx)).KafkaQuotasV1ClientQuota(kafkaClientQuota)
	return req.Execute()
}

func kafkaClientQuotaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.quotasClient.ClientQuotasKafkaQuotasV1Api.DeleteKafkaQuotasV1ClientQuota(c.quotasApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Kafka Client Quota %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	return nil
}

func kafkaClientQuotaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	kafkaClientQuotaId := d.Id()

	if _, err := readKafkaClientQuotaAndSetAttributes(ctx, d, meta, kafkaClientQuotaId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Kafka Client Quota %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readKafkaClientQuotaAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, kafkaClientQuotaId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	kafkaClientQuota, resp, err := executeKafkaClientQuotaRead(c.quotasApiContext(ctx), c, kafkaClientQuotaId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Client Quota %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka Client Quota %q in TF state because Kafka Client Quota could not be found on the server", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, createDescriptiveError(err)
	}
	kafkaClientQuotaJson, err := json.Marshal(kafkaClientQuota)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Client Quota %q: error marshaling %#v to json: %s", d.Id(), kafkaClientQuota, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Client Quota %q: %s", d.Id(), kafkaClientQuotaJson), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	if _, err := setKafkaClientQuotaAttributes(d, kafkaClientQuota); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func executeKafkaClientQuotaRead(ctx context.Context, c *Client, kafkaClientQuotaId string) (quotas.KafkaQuotasV1ClientQuota, *http.Response, error) {
	req := c.quotasClient.ClientQuotasKafkaQuotasV1Api.GetKafkaQuotasV1ClientQuota(c.quotasApiContext(ctx), kafkaClientQuotaId)
	return req.Execute()
}

func setKafkaClientQuotaAttributes(d *schema.ResourceData, kafkaClientQuota quotas.KafkaQuotasV1ClientQuota) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, kafkaClientQuota.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, kafkaClientQuota.Spec.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}

	// Principals
	if err := d.Set(paramPrincipals, convertToListOfIds(kafkaClientQuota.Spec.GetPrincipals())); err != nil {
		return nil, createDescriptiveError(err)
	}

	// Throughput
	if err := d.Set(paramThroughput, []interface{}{map[string]interface{}{
		paramIngressByteRate: kafkaClientQuota.Spec.Throughput.GetIngressByteRate(),
		paramEgressByteRate:  kafkaClientQuota.Spec.Throughput.GetEgressByteRate()}}); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, kafkaClientQuota.Spec.Cluster.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, kafkaClientQuota.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(kafkaClientQuota.GetId())
	return d, nil
}

func kafkaClientQuotaImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := kafkaClientQuotaRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Kafka Client Quota %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka Client Quota %q", d.Id()), map[string]interface{}{kafkaClientQuotaLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func throughputSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Required:    true,
		Description: "Block for representing a Kafka Quota.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramIngressByteRate: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The ingress throughput limit in bytes per second.",
				},
				paramEgressByteRate: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The egress throughput limit in bytes per second.",
				},
			},
		},
	}
}

func convertToGlobalObjectReferences(ids []string) []quotas.GlobalObjectReference {
	globalObjectReferences := make([]quotas.GlobalObjectReference, len(ids))
	for i, id := range ids {
		globalObjectReferences[i] = quotas.GlobalObjectReference{Id: id}
	}
	return globalObjectReferences
}

func convertToListOfIds(globalObjectReferences []quotas.GlobalObjectReference) []string {
	ids := make([]string, len(globalObjectReferences))
	for i, globalObjectReference := range globalObjectReferences {
		ids[i] = globalObjectReference.GetId()
	}
	return ids
}

func convertSetToStringList(d *schema.ResourceData, attributeName string) []string {
	setValues := d.Get(attributeName).(*schema.Set).List()
	stringSetValues := make([]string, len(setValues))
	for i, _ := range setValues {
		stringSetValues[i] = setValues[i].(string)
	}
	return stringSetValues
}
