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
	sg "github.com/confluentinc/ccloud-sdk-go-v2/stream-governance/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

const (
	paramPackage             = "package"
	billingPackageEssentials = "ESSENTIALS"
	billingPackageAdvanced   = "ADVANCED"
)

var acceptedBillingPackages = []string{billingPackageEssentials, billingPackageAdvanced}

func streamGovernanceClusterResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: streamGovernanceClusterCreate,
		ReadContext:   streamGovernanceClusterRead,
		UpdateContext: streamGovernanceClusterUpdate,
		DeleteContext: streamGovernanceClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: streamGovernanceClusterImport,
		},
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentSchema(),
			paramRegion:      streamGovernanceRegionSchema(),
			paramPackage: {
				Type:         schema.TypeString,
				Description:  "The billing package.",
				ValidateFunc: validation.StringInSlice(acceptedBillingPackages, false),
				Required:     true,
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Stream Governance Cluster.",
				Computed:    true,
			},
			paramHttpEndpoint: {
				Type:        schema.TypeString,
				Description: "The API endpoint of the Stream Governance Cluster.",
				Computed:    true,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Stream Governance Cluster.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Stream Governance Cluster represents.",
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Stream Governance Cluster.",
			},
		},
	}
}

func streamGovernanceClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	billingPackage := d.Get(paramPackage).(string)
	regionId := extractStringValueFromBlock(d, paramRegion, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec := sg.NewStreamGovernanceV2ClusterSpec()
	spec.SetPackage(billingPackage)
	spec.SetRegion(sg.GlobalObjectReference{Id: regionId})
	spec.SetEnvironment(sg.GlobalObjectReference{Id: environmentId})

	createStreamGovernanceClusterRequest := sg.StreamGovernanceV2Cluster{Spec: spec}
	createStreamGovernanceClusterRequestJson, err := json.Marshal(createStreamGovernanceClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Stream Governance Cluster: error marshaling %#v to json: %s", createStreamGovernanceClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Stream Governance Cluster: %s", createStreamGovernanceClusterRequestJson))

	createdStreamGovernanceCluster, _, err := executeStreamGovernanceClusterCreate(c.sgApiContext(ctx), c, createStreamGovernanceClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Stream Governance Cluster %q: %s", createdStreamGovernanceCluster.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdStreamGovernanceCluster.GetId())

	if err := waitForStreamGovernanceClusterToProvision(c.sgApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Stream Governance Cluster %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdStreamGovernanceClusterJson, err := json.Marshal(createdStreamGovernanceCluster)
	if err != nil {
		return diag.Errorf("error creating Stream Governance Cluster %q: error marshaling %#v to json: %s", d.Id(), createdStreamGovernanceCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Stream Governance Cluster %q: %s", d.Id(), createdStreamGovernanceClusterJson), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	return streamGovernanceClusterRead(ctx, d, meta)
}

func executeStreamGovernanceClusterCreate(ctx context.Context, c *Client, streamGovernanceCluster sg.StreamGovernanceV2Cluster) (sg.StreamGovernanceV2Cluster, *http.Response, error) {
	req := c.sgClient.ClustersStreamGovernanceV2Api.CreateStreamGovernanceV2Cluster(c.sgApiContext(ctx)).StreamGovernanceV2Cluster(streamGovernanceCluster)
	return req.Execute()
}

func executeStreamGovernanceClusterRead(ctx context.Context, c *Client, environmentId string, streamGovernanceClusterId string) (sg.StreamGovernanceV2Cluster, *http.Response, error) {
	req := c.sgClient.ClustersStreamGovernanceV2Api.GetStreamGovernanceV2Cluster(c.sgApiContext(ctx), streamGovernanceClusterId).Environment(environmentId)
	return req.Execute()
}

func streamGovernanceClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	streamGovernanceClusterId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readStreamGovernanceClusterAndSetAttributes(ctx, d, meta, environmentId, streamGovernanceClusterId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Stream Governance Cluster %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readStreamGovernanceClusterAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, streamGovernanceClusterId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	streamGovernanceCluster, resp, err := executeStreamGovernanceClusterRead(c.sgApiContext(ctx), c, environmentId, streamGovernanceClusterId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Stream Governance Cluster %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Stream Governance Cluster %q in TF state because Stream Governance Cluster could not be found on the server", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	streamGovernanceClusterJson, err := json.Marshal(streamGovernanceCluster)
	if err != nil {
		return nil, fmt.Errorf("error reading Stream Governance Cluster %q: error marshaling %#v to json: %s", streamGovernanceClusterId, streamGovernanceCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Stream Governance Cluster %q: %s", d.Id(), streamGovernanceClusterJson), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	if _, err := setStreamGovernanceClusterAttributes(d, streamGovernanceCluster); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setStreamGovernanceClusterAttributes(d *schema.ResourceData, streamGovernanceCluster sg.StreamGovernanceV2Cluster) (*schema.ResourceData, error) {
	if err := d.Set(paramPackage, streamGovernanceCluster.Spec.GetPackage()); err != nil {
		return nil, err
	}

	// Set blocks
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, streamGovernanceCluster.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramRegion, paramId, streamGovernanceCluster.Spec.Region.GetId(), d); err != nil {
		return nil, err
	}

	// Set computed attributes
	if err := d.Set(paramDisplayName, streamGovernanceCluster.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramHttpEndpoint, streamGovernanceCluster.Spec.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, streamGovernanceCluster.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, streamGovernanceCluster.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, streamGovernanceCluster.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(streamGovernanceCluster.GetId())
	return d, nil
}

func streamGovernanceClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.sgClient.ClustersStreamGovernanceV2Api.DeleteStreamGovernanceV2Cluster(c.sgApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Stream Governance Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	return nil
}

func streamGovernanceClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramPackage) {
		return diag.Errorf("error updating Stream Governance Cluster %q: only %q attribute can be updated for Stream Governance Cluster", d.Id(), paramPackage)
	}

	c := meta.(*Client)
	updatedPackage := d.Get(paramPackage).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateStreamGovernanceClusterRequest := sg.NewStreamGovernanceV2ClusterUpdate()
	updateSpec := sg.NewStreamGovernanceV2ClusterSpecUpdate()
	updateSpec.SetPackage(updatedPackage)
	updateSpec.SetEnvironment(sg.GlobalObjectReference{Id: environmentId})
	updateStreamGovernanceClusterRequest.SetSpec(*updateSpec)
	updateStreamGovernanceClusterRequestJson, err := json.Marshal(updateStreamGovernanceClusterRequest)
	if err != nil {
		return diag.Errorf("error updating Stream Governance Cluster %q: error marshaling %#v to json: %s", d.Id(), updateStreamGovernanceClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Stream Governance Cluster %q: %s", d.Id(), updateStreamGovernanceClusterRequestJson), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	req := c.sgClient.ClustersStreamGovernanceV2Api.UpdateStreamGovernanceV2Cluster(c.sgApiContext(ctx), d.Id()).StreamGovernanceV2ClusterUpdate(*updateStreamGovernanceClusterRequest)
	updatedStreamGovernanceCluster, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Stream Governance Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedStreamGovernanceClusterJson, err := json.Marshal(updatedStreamGovernanceCluster)
	if err != nil {
		return diag.Errorf("error updating Stream Governance Cluster %q: error marshaling %#v to json: %s", d.Id(), updatedStreamGovernanceCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Stream Governance Cluster %q: %s", d.Id(), updatedStreamGovernanceClusterJson), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})
	return streamGovernanceClusterRead(ctx, d, meta)
}

func streamGovernanceClusterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})

	envIDAndStreamGovernanceClusterId := d.Id()
	parts := strings.Split(envIDAndStreamGovernanceClusterId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Stream Governance Cluster: invalid format: expected '<env ID>/<streamGovernanceCluster ID>'")
	}

	environmentId := parts[0]
	streamGovernanceClusterId := parts[1]
	d.SetId(streamGovernanceClusterId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readStreamGovernanceClusterAndSetAttributes(ctx, d, meta, environmentId, streamGovernanceClusterId); err != nil {
		return nil, fmt.Errorf("error importing Stream Governance Cluster %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Stream Governance Cluster %q", d.Id()), map[string]interface{}{streamGovernanceClusterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func streamGovernanceRegionSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the Stream Governance Region.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}
