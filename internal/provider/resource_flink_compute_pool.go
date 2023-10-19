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
	fcpm "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
	"time"
)

const (
	paramCurrentCfu         = "current_cfu"
	paramMaxCfu             = "max_cfu"
	computePoolTypeStandard = "Standard"

	fcpmAPICreateTimeout = 1 * time.Hour
	fcpmAPIDeleteTimeout = 1 * time.Hour
)

var acceptedComputePoolTypes = []string{paramStandardCluster}

func computePoolResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: computePoolCreate,
		ReadContext:   computePoolRead,
		UpdateContext: computePoolUpdate,
		DeleteContext: computePoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: computePoolImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Flink compute pool.",
				Required:    true,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider that runs the compute pool.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
				ForceNew:     true,
				// Suppress the diff shown if the value of "cloud" attribute are equal when both compared in lower case.
				// For example, AWS == aws
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.ToLower(old) == strings.ToLower(new) {
						return true
					}
					return false
				},
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider region that hosts the Flink compute pool.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramMaxCfu: {
				Type:         schema.TypeInt,
				Description:  "Maximum number of Confluent Flink Units (CFUs) that the Flink compute pool should auto-scale to.",
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntInSlice([]int{5, 10}),
			},
			paramEnvironment: environmentSchema(),
			paramRestEndpoint: {
				Type:        schema.TypeString,
				Description: "The API endpoint of the ksqlDB cluster.",
				Computed:    true,
			},
			paramCurrentCfu: {
				Type:        schema.TypeInt,
				Description: "The number of Confluent Flink Units (CFUs) currently allocated to this Flink compute pool.",
				Computed:    true,
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Flink compute pool.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(fcpmAPICreateTimeout),
			Delete: schema.DefaultTimeout(fcpmAPIDeleteTimeout),
		},
	}
}

func computePoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	// Non-zero value means maxCfu has been set
	maxCfu := d.Get(paramMaxCfu).(int)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec := fcpm.NewFcpmV2ComputePoolSpec()
	spec.SetDisplayName(displayName)
	spec.SetCloud(cloud)
	spec.SetRegion(region)
	spec.SetMaxCfu(int32(maxCfu))
	spec.SetEnvironment(fcpm.GlobalObjectReference{Id: environmentId})

	createComputePoolRequest := fcpm.FcpmV2ComputePool{Spec: spec}
	createComputePoolRequestJson, err := json.Marshal(createComputePoolRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Compute Pool: error marshaling %#v to json: %s", createComputePoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Flink Compute Pool: %s", createComputePoolRequestJson))

	createdComputePool, _, err := executeComputePoolCreate(c.fcpmApiContext(ctx), c, createComputePoolRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Compute Pool %q: %s", createdComputePool.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdComputePool.GetId())

	if err := waitForComputePoolToProvision(c.fcpmApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Flink Compute Pool %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdComputePoolJson, err := json.Marshal(createdComputePool)
	if err != nil {
		return diag.Errorf("error creating Flink Compute Pool %q: error marshaling %#v to json: %s", d.Id(), createdComputePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Compute Pool %q: %s", d.Id(), createdComputePoolJson), map[string]interface{}{computePoolLoggingKey: d.Id()})

	return computePoolRead(ctx, d, meta)
}

func executeComputePoolCreate(ctx context.Context, c *Client, computePool fcpm.FcpmV2ComputePool) (fcpm.FcpmV2ComputePool, *http.Response, error) {
	req := c.fcpmClient.ComputePoolsFcpmV2Api.CreateFcpmV2ComputePool(c.fcpmApiContext(ctx)).FcpmV2ComputePool(computePool)
	return req.Execute()
}

func executeComputePoolRead(ctx context.Context, c *Client, environmentId string, computePoolId string) (fcpm.FcpmV2ComputePool, *http.Response, error) {
	req := c.fcpmClient.ComputePoolsFcpmV2Api.GetFcpmV2ComputePool(c.fcpmApiContext(ctx), computePoolId).Environment(environmentId)
	return req.Execute()
}

func computePoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})

	computePoolId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readComputePoolAndSetAttributes(ctx, d, meta, environmentId, computePoolId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Flink Compute Pool %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readComputePoolAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, computePoolId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	computePool, resp, err := executeComputePoolRead(c.fcpmApiContext(ctx), c, environmentId, computePoolId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Compute Pool %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{computePoolLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Compute Pool %q in TF state because Flink Compute Pool could not be found on the server", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	computePoolJson, err := json.Marshal(computePool)
	if err != nil {
		return nil, fmt.Errorf("error reading Flink Compute Pool %q: error marshaling %#v to json: %s", computePoolId, computePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Compute Pool %q: %s", d.Id(), computePoolJson), map[string]interface{}{computePoolLoggingKey: d.Id()})

	if _, err := setComputePoolAttributes(d, computePool); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setComputePoolAttributes(d *schema.ResourceData, computePool fcpm.FcpmV2ComputePool) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, computePool.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, computePool.Spec.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, computePool.Spec.GetRegion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramMaxCfu, computePool.Spec.GetMaxCfu()); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, computePool.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}

	if err := d.Set(paramRestEndpoint, computePool.Spec.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCurrentCfu, computePool.Status.GetCurrentCfu()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, computePool.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, computePool.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, computePool.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(computePool.GetId())
	return d, nil
}

func computePoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.fcpmClient.ComputePoolsFcpmV2Api.DeleteFcpmV2ComputePool(c.fcpmApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Flink Compute Pool %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})

	return nil
}

func computePoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramMaxCfu, paramDisplayName) {
		return diag.Errorf("error updating Flink Compute Pool %q: only %q, and %q attributes can be updated for Flink Compute Pool", d.Id(), paramMaxCfu, paramDisplayName)
	}

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateComputePoolRequest := fcpm.NewFcpmV2ComputePoolUpdate()
	updateSpec := fcpm.NewFcpmV2ComputePoolSpecUpdate()
	updateSpec.SetEnvironment(fcpm.GlobalObjectReference{Id: environmentId})

	if d.HasChange(paramMaxCfu) {
		updateSpec.SetMaxCfu(int32(d.Get(paramMaxCfu).(int)))
	}
	if d.HasChange(paramDisplayName) {
		updateSpec.SetDisplayName(d.Get(paramDisplayName).(string))
	}

	updateComputePoolRequest.SetSpec(*updateSpec)
	updateComputePoolRequestJson, err := json.Marshal(updateComputePoolRequest)
	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool %q: error marshaling %#v to json: %s", d.Id(), updateComputePoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Compute Pool %q: %s", d.Id(), updateComputePoolRequestJson), map[string]interface{}{computePoolLoggingKey: d.Id()})

	req := c.fcpmClient.ComputePoolsFcpmV2Api.UpdateFcpmV2ComputePool(c.fcpmApiContext(ctx), d.Id()).FcpmV2ComputePoolUpdate(*updateComputePoolRequest)
	updatedComputePool, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedComputePoolJson, err := json.Marshal(updatedComputePool)
	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool %q: error marshaling %#v to json: %s", d.Id(), updatedComputePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Flink Compute Pool %q: %s", d.Id(), updatedComputePoolJson), map[string]interface{}{computePoolLoggingKey: d.Id()})
	return computePoolRead(ctx, d, meta)
}

func computePoolImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})

	envIDAndComputePoolId := d.Id()
	parts := strings.Split(envIDAndComputePoolId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Flink Compute Pool: invalid format: expected '<env ID>/<Flink Compute Pool ID>'")
	}

	environmentId := parts[0]
	computePoolId := parts[1]
	d.SetId(computePoolId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readComputePoolAndSetAttributes(ctx, d, meta, environmentId, computePoolId); err != nil {
		return nil, fmt.Errorf("error importing Flink Compute Pool %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Compute Pool %q", d.Id()), map[string]interface{}{computePoolLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
