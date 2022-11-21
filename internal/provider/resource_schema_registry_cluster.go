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
	sg "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

func schemaRegistryClusterResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: schemaRegistryClusterCreate,
		ReadContext:   schemaRegistryClusterRead,
		UpdateContext: schemaRegistryClusterUpdate,
		DeleteContext: schemaRegistryClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schemaRegistryClusterImport,
		},
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentSchema(),
			paramRegion:      schemaRegistryRegionSchema(),
			paramPackage: {
				Type:         schema.TypeString,
				Description:  "The billing package.",
				ValidateFunc: validation.StringInSlice(acceptedBillingPackages, false),
				Required:     true,
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Schema Registry Cluster.",
				Computed:    true,
			},
			paramRestEndpoint: {
				Type:        schema.TypeString,
				Description: "The API endpoint of the Schema Registry Cluster.",
				Computed:    true,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Schema Registry Cluster.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Schema Registry Cluster represents.",
			},
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Schema Registry Cluster.",
			},
		},
	}
}

func schemaRegistryClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	billingPackage := d.Get(paramPackage).(string)
	regionId := extractStringValueFromBlock(d, paramRegion, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec := sg.NewSrcmV2ClusterSpec()
	spec.SetPackage(billingPackage)
	spec.SetRegion(sg.GlobalObjectReference{Id: regionId})
	spec.SetEnvironment(sg.GlobalObjectReference{Id: environmentId})

	createSchemaRegistryClusterRequest := sg.SrcmV2Cluster{Spec: spec}
	createSchemaRegistryClusterRequestJson, err := json.Marshal(createSchemaRegistryClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster: error marshaling %#v to json: %s", createSchemaRegistryClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Registry Cluster: %s", createSchemaRegistryClusterRequestJson))

	createdSchemaRegistryCluster, _, err := executeSchemaRegistryClusterCreate(c.srcmApiContext(ctx), c, createSchemaRegistryClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster %q: %s", createdSchemaRegistryCluster.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdSchemaRegistryCluster.GetId())

	if err := waitForSchemaRegistryClusterToProvision(c.srcmApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Schema Registry Cluster %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdSchemaRegistryClusterJson, err := json.Marshal(createdSchemaRegistryCluster)
	if err != nil {
		return diag.Errorf("error creating Schema Registry Cluster %q: error marshaling %#v to json: %s", d.Id(), createdSchemaRegistryCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Registry Cluster %q: %s", d.Id(), createdSchemaRegistryClusterJson), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	return schemaRegistryClusterRead(ctx, d, meta)
}

func executeSchemaRegistryClusterCreate(ctx context.Context, c *Client, schemaRegistryCluster sg.SrcmV2Cluster) (sg.SrcmV2Cluster, *http.Response, error) {
	req := c.srcmClient.ClustersSrcmV2Api.CreateSrcmV2Cluster(c.srcmApiContext(ctx)).SrcmV2Cluster(schemaRegistryCluster)
	return req.Execute()
}

func executeSchemaRegistryClusterRead(ctx context.Context, c *Client, environmentId string, schemaRegistryClusterId string) (sg.SrcmV2Cluster, *http.Response, error) {
	req := c.srcmClient.ClustersSrcmV2Api.GetSrcmV2Cluster(c.srcmApiContext(ctx), schemaRegistryClusterId).Environment(environmentId)
	return req.Execute()
}

func schemaRegistryClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	schemaRegistryClusterId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readSchemaRegistryClusterAndSetAttributes(ctx, d, meta, environmentId, schemaRegistryClusterId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Schema Registry Cluster %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readSchemaRegistryClusterAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, schemaRegistryClusterId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	schemaRegistryCluster, resp, err := executeSchemaRegistryClusterRead(c.srcmApiContext(ctx), c, environmentId, schemaRegistryClusterId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Registry Cluster %q in TF state because Schema Registry Cluster could not be found on the server", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	schemaRegistryClusterJson, err := json.Marshal(schemaRegistryCluster)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Registry Cluster %q: error marshaling %#v to json: %s", schemaRegistryClusterId, schemaRegistryCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Registry Cluster %q: %s", d.Id(), schemaRegistryClusterJson), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	if _, err := setSchemaRegistryClusterAttributes(d, schemaRegistryCluster); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setSchemaRegistryClusterAttributes(d *schema.ResourceData, schemaRegistryCluster sg.SrcmV2Cluster) (*schema.ResourceData, error) {
	if err := d.Set(paramPackage, schemaRegistryCluster.Spec.GetPackage()); err != nil {
		return nil, err
	}

	// Set blocks
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, schemaRegistryCluster.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramRegion, paramId, schemaRegistryCluster.Spec.Region.GetId(), d); err != nil {
		return nil, err
	}

	// Set computed attributes
	if err := d.Set(paramDisplayName, schemaRegistryCluster.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRestEndpoint, schemaRegistryCluster.Spec.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, schemaRegistryCluster.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, schemaRegistryCluster.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceName, schemaRegistryCluster.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(schemaRegistryCluster.GetId())
	return d, nil
}

func schemaRegistryClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.srcmClient.ClustersSrcmV2Api.DeleteSrcmV2Cluster(c.srcmApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Schema Registry Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	return nil
}

func schemaRegistryClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramPackage) {
		return diag.Errorf("error updating Schema Registry Cluster %q: only %q attribute can be updated for Schema Registry Cluster", d.Id(), paramPackage)
	}

	c := meta.(*Client)
	updatedPackage := d.Get(paramPackage).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateSchemaRegistryClusterRequest := sg.NewSrcmV2ClusterUpdate()
	updateSpec := sg.NewSrcmV2ClusterSpecUpdate()
	updateSpec.SetPackage(updatedPackage)
	updateSpec.SetEnvironment(sg.GlobalObjectReference{Id: environmentId})
	updateSchemaRegistryClusterRequest.SetSpec(*updateSpec)
	updateSchemaRegistryClusterRequestJson, err := json.Marshal(updateSchemaRegistryClusterRequest)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Cluster %q: error marshaling %#v to json: %s", d.Id(), updateSchemaRegistryClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Schema Registry Cluster %q: %s", d.Id(), updateSchemaRegistryClusterRequestJson), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	req := c.srcmClient.ClustersSrcmV2Api.UpdateSrcmV2Cluster(c.srcmApiContext(ctx), d.Id()).SrcmV2ClusterUpdate(*updateSchemaRegistryClusterRequest)
	updatedSchemaRegistryCluster, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Schema Registry Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedSchemaRegistryClusterJson, err := json.Marshal(updatedSchemaRegistryCluster)
	if err != nil {
		return diag.Errorf("error updating Schema Registry Cluster %q: error marshaling %#v to json: %s", d.Id(), updatedSchemaRegistryCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Registry Cluster %q: %s", d.Id(), updatedSchemaRegistryClusterJson), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})
	return schemaRegistryClusterRead(ctx, d, meta)
}

func schemaRegistryClusterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})

	envIDAndSchemaRegistryClusterId := d.Id()
	parts := strings.Split(envIDAndSchemaRegistryClusterId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Schema Registry Cluster: invalid format: expected '<env ID>/<schemaRegistryCluster ID>'")
	}

	environmentId := parts[0]
	schemaRegistryClusterId := parts[1]
	d.SetId(schemaRegistryClusterId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSchemaRegistryClusterAndSetAttributes(ctx, d, meta, environmentId, schemaRegistryClusterId); err != nil {
		return nil, fmt.Errorf("error importing Schema Registry Cluster %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema Registry Cluster %q", d.Id()), map[string]interface{}{schemaRegistryClusterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func schemaRegistryRegionSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the Schema Registry Region.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}
