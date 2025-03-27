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
	"net/http"
	"regexp"
	"strings"

	cmk "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	kafkaClusterTypeBasic      = "Basic"
	kafkaClusterTypeStandard   = "Standard"
	kafkaClusterTypeDedicated  = "Dedicated"
	kafkaClusterTypeEnterprise = "Enterprise"
	kafkaClusterTypeFreight    = "Freight"
	paramBasicCluster          = "basic"
	paramStandardCluster       = "standard"
	paramDedicatedCluster      = "dedicated"
	paramEnterpriseCluster     = "enterprise"
	paramFreightCluster        = "freight"
	paramAvailability          = "availability"
	paramBootStrapEndpoint     = "bootstrap_endpoint"
	paramRestEndpoint          = "rest_endpoint"
	paramHttpEndpoint          = "http_endpoint"
	paramRestEndpointPrivate   = "private_rest_endpoint"
	paramCatalogEndpoint       = "catalog_endpoint"
	paramCku                   = "cku"
	paramEncryptionKey         = "encryption_key"
	paramRbacCrn               = "rbac_crn"
	paramConfluentCustomerKey  = "byok_key"

	stateInProgress = "IN_PROGRESS"
	stateDone       = "DONE"

	stateFailed        = "FAILED"
	stateUnknown       = "UNKNOWN"
	stateUnexpected    = "UNEXEPCTED"
	stateProvisioned   = "PROVISIONED"
	stateReady         = "READY"
	stateRunning       = "RUNNING"
	stateProvisioning  = "PROVISIONING"
	statePendingAccept = "PENDING_ACCEPT"

	singleZone       = "SINGLE_ZONE"
	multiZone        = "MULTI_ZONE"
	lowAvailability  = "LOW"
	highAvailability = "HIGH"
)

var acceptedAvailabilityZones = []string{singleZone, multiZone, lowAvailability, highAvailability}
var acceptedCloudProviders = []string{"AWS", "AZURE", "GCP"}
var acceptedClusterTypes = []string{paramBasicCluster, paramStandardCluster, paramDedicatedCluster, paramEnterpriseCluster, paramFreightCluster}
var paramDedicatedCku = fmt.Sprintf("%s.0.%s", paramDedicatedCluster, paramCku)
var paramDedicatedEncryptionKey = fmt.Sprintf("%s.0.%s", paramDedicatedCluster, paramEncryptionKey)
var paramDedicatedZones = fmt.Sprintf("%s.0.%s", paramDedicatedCluster, paramZones)

func kafkaResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: kafkaCreate,
		ReadContext:   kafkaRead,
		UpdateContext: kafkaUpdate,
		DeleteContext: kafkaDelete,
		Importer: &schema.ResourceImporter{
			StateContext: kafkaImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the Kafka cluster.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a Kafka cluster.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object Kafka cluster represents.",
			},
			paramAvailability: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The availability zone configuration of the Kafka cluster.",
				ValidateFunc: validation.StringInSlice(acceptedAvailabilityZones, false),
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The cloud service provider that runs the Kafka cluster.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
			},
			paramRegion: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The cloud service provider region where the Kafka cluster is running.",
			},
			paramNetwork:           optionalNetworkSchema(),
			paramBasicCluster:      basicClusterSchema(),
			paramStandardCluster:   standardClusterSchema(),
			paramDedicatedCluster:  dedicatedClusterSchema(),
			paramEnterpriseCluster: enterpriseClusterSchema(),
			paramFreightCluster:    freightClusterSchema(),
			paramBootStrapEndpoint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The bootstrap endpoint used by Kafka clients to connect to the Kafka cluster.",
			},
			paramRestEndpoint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The REST endpoint of the Kafka cluster.",
			},
			paramRbacCrn: {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The Confluent Resource Name of the Kafka cluster suitable for " +
					"confluent_role_binding's crn_pattern.",
			},
			paramEnvironment:          environmentSchema(),
			paramConfluentCustomerKey: byokSchema(),
		},
		CustomizeDiff: customdiff.Sequence(resourceKafkaCustomizeDiff),
		Timeouts: &schema.ResourceTimeout{
			// https://docs.confluent.io/cloud/current/clusters/cluster-types.html#provisioning-time
			Create: schema.DefaultTimeout(getTimeoutFor(kafkaClusterTypeDedicated)),
			// https://docs.confluent.io/cloud/current/clusters/cluster-types.html#resizing-time
			Update: schema.DefaultTimeout(getTimeoutFor(kafkaClusterTypeDedicated)),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    kafkaResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: kafkaStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func resourceKafkaCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	newClusterType := extractClusterTypeResourceDiff(diff)

	// Display an error for forbidden cluster updates during `terraform plan`:
	// More specifically, any update except Basic -> Standard is forbidden:
	// * Standard -> Basic
	// * Basic -> Dedicated
	// * Standard -> Dedicated
	// * etc.
	isForbiddenStandardBasicUpdate := newClusterType == kafkaClusterTypeBasic && diff.HasChange(paramBasicCluster) && diff.HasChange(paramStandardCluster) && !diff.HasChange(paramDedicatedCluster)
	isForbiddenDedicatedUpdate := diff.HasChange(paramDedicatedCluster) && (diff.HasChange(paramBasicCluster) || diff.HasChange(paramStandardCluster))

	if isForbiddenStandardBasicUpdate || isForbiddenDedicatedUpdate {
		return fmt.Errorf("error updating Kafka Cluster %q: clusters can only be upgraded from 'Basic' to 'Standard'", diff.Id())
	}

	return nil
}

func kafkaUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterType := extractClusterType(d)
	// Non-zero value means CKU has been set
	cku := extractCku(d)
	if d.HasChange(paramDisplayName) {
		updateClusterRequest := cmk.NewCmkV2ClusterUpdate()
		updateSpec := cmk.NewCmkV2ClusterSpecUpdate()
		updateSpec.SetDisplayName(displayName)
		updateSpec.SetEnvironment(cmk.EnvScopedObjectReference{Id: environmentId})
		updateClusterRequest.SetSpec(*updateSpec)
		updateClusterRequestJson, err := json.Marshal(updateClusterRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updateClusterRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Cluster %q: %s", d.Id(), updateClusterRequestJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

		req := c.cmkClient.ClustersCmkV2Api.UpdateCmkV2Cluster(c.cmkApiContext(ctx), d.Id()).CmkV2ClusterUpdate(*updateClusterRequest)

		updatedCluster, _, err := req.Execute()

		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err))
		}
		updatedClusterJson, err := json.Marshal(updatedCluster)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updatedCluster, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updated Kafka Cluster %q: %s", d.Id(), updatedClusterJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	}

	// Allow only Basic -> Standard upgrade
	isBasicStandardUpdate := d.HasChange(paramBasicCluster) && d.HasChange(paramStandardCluster) && !d.HasChange(paramDedicatedCluster) && clusterType == kafkaClusterTypeStandard
	// Watch out for forbidden updates / downgrades: e.g., Standard -> Basic, Basic -> Dedicated etc.
	isForbiddenStandardBasicDowngrade := d.HasChange(paramBasicCluster) && d.HasChange(paramStandardCluster) && !d.HasChange(paramDedicatedCluster) && clusterType == kafkaClusterTypeBasic
	isForbiddenDedicatedUpdate := d.HasChange(paramDedicatedCluster) && (d.HasChange(paramBasicCluster) || d.HasChange(paramStandardCluster))

	if isBasicStandardUpdate {
		updateClusterRequest := cmk.NewCmkV2ClusterUpdate()
		updateSpec := cmk.NewCmkV2ClusterSpecUpdate()
		updateSpec.SetConfig(cmk.CmkV2StandardAsCmkV2ClusterSpecUpdateConfigOneOf(cmk.NewCmkV2Standard(kafkaClusterTypeStandard)))
		updateSpec.SetEnvironment(cmk.EnvScopedObjectReference{Id: environmentId})
		updateClusterRequest.SetSpec(*updateSpec)
		updateClusterRequestJson, err := json.Marshal(updateClusterRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updateClusterRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Cluster %q: %s", d.Id(), updateClusterRequestJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

		req := c.cmkClient.ClustersCmkV2Api.UpdateCmkV2Cluster(c.cmkApiContext(ctx), d.Id()).CmkV2ClusterUpdate(*updateClusterRequest)

		updatedCluster, _, err := req.Execute()

		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err))
		}
		updatedClusterJson, err := json.Marshal(updatedCluster)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updatedCluster, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updated Kafka Cluster %q: %s", d.Id(), updatedClusterJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	} else if isForbiddenStandardBasicDowngrade || isForbiddenDedicatedUpdate {
		return diag.Errorf("error updating Kafka Cluster %q: clusters can only be upgraded from 'Basic' to 'Standard'", d.Id())
	}

	isCkuUpdate := d.HasChange(paramDedicatedCluster) && clusterType == kafkaClusterTypeDedicated && d.HasChange(paramDedicatedCku)
	if isCkuUpdate {
		availability := d.Get(paramAvailability).(string)
		err := ckuCheck(cku, availability)
		if err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}

		updateClusterRequest := cmk.NewCmkV2ClusterUpdate()
		updateSpec := cmk.NewCmkV2ClusterSpecUpdate()
		updateSpec.SetConfig(cmk.CmkV2DedicatedAsCmkV2ClusterSpecUpdateConfigOneOf(cmk.NewCmkV2Dedicated(kafkaClusterTypeDedicated, cku)))
		updateSpec.SetEnvironment(cmk.EnvScopedObjectReference{Id: environmentId})
		updateClusterRequest.SetSpec(*updateSpec)
		updateClusterRequestJson, err := json.Marshal(updateClusterRequest)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updateClusterRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Kafka Cluster %q: %s", d.Id(), updateClusterRequestJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
		req := c.cmkClient.ClustersCmkV2Api.UpdateCmkV2Cluster(c.cmkApiContext(ctx), d.Id()).CmkV2ClusterUpdate(*updateClusterRequest)

		updatedCluster, _, err := req.Execute()
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err))
		}

		if err := waitForKafkaClusterCkuUpdateToComplete(c.cmkApiContext(ctx), c, environmentId, d.Id(), cku); err != nil {
			return diag.Errorf("error waiting for Kafka Cluster %q to perform CKU update: %s", d.Id(), createDescriptiveError(err))
		}
		updatedClusterJson, err := json.Marshal(updatedCluster)
		if err != nil {
			return diag.Errorf("error updating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), updatedCluster, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updated Kafka Cluster %q: %s", d.Id(), updatedClusterJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished updating Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return kafkaRead(ctx, d, meta)
}

func executeKafkaCreate(ctx context.Context, c *Client, cluster *cmk.CmkV2Cluster) (cmk.CmkV2Cluster, *http.Response, error) {
	req := c.cmkClient.ClustersCmkV2Api.CreateCmkV2Cluster(c.cmkApiContext(ctx)).CmkV2Cluster(*cluster)
	return req.Execute()
}

func kafkaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	availability := d.Get(paramAvailability).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	clusterType := extractClusterType(d)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	byokId := extractStringValueFromBlock(d, paramConfluentCustomerKey, paramId)

	spec := cmk.NewCmkV2ClusterSpec()
	spec.SetDisplayName(displayName)
	spec.SetAvailability(availability)
	spec.SetCloud(cloud)
	spec.SetRegion(region)
	if clusterType == kafkaClusterTypeBasic {
		spec.SetConfig(cmk.CmkV2BasicAsCmkV2ClusterSpecConfigOneOf(cmk.NewCmkV2Basic(kafkaClusterTypeBasic)))
	} else if clusterType == kafkaClusterTypeStandard {
		spec.SetConfig(cmk.CmkV2StandardAsCmkV2ClusterSpecConfigOneOf(cmk.NewCmkV2Standard(kafkaClusterTypeStandard)))
	} else if clusterType == kafkaClusterTypeDedicated {
		cku := extractCku(d)
		err := ckuCheck(cku, availability)
		if err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}
		encryptionKey := extractEncryptionKey(d)

		config := cmk.NewCmkV2Dedicated(kafkaClusterTypeDedicated, cku)
		if encryptionKey != "" {
			config.SetEncryptionKey(encryptionKey)
		}

		zones := convertToStringSlice(d.Get(paramDedicatedZones).([]interface{}))
		if len(zones) > 0 {
			if availability != singleZone || networkId == "" {
				return diag.Errorf("availability is not single zone or netowrk ID not specified. Zone selection is supported only for creating single zone dedicated Kafka cluster on private network")
			}
			config.SetZones(zones)
		}

		spec.SetConfig(cmk.CmkV2DedicatedAsCmkV2ClusterSpecConfigOneOf(config))
	} else if clusterType == kafkaClusterTypeEnterprise {
		spec.SetConfig(cmk.CmkV2EnterpriseAsCmkV2ClusterSpecConfigOneOf(cmk.NewCmkV2Enterprise(kafkaClusterTypeEnterprise)))
	} else if clusterType == kafkaClusterTypeFreight {
		spec.SetConfig(cmk.CmkV2FreightAsCmkV2ClusterSpecConfigOneOf(cmk.NewCmkV2Freight(kafkaClusterTypeFreight)))
	} else {
		return diag.Errorf("error creating Kafka Cluster: unknown Kafka Cluster type was provided: %q", clusterType)
	}
	spec.SetEnvironment(cmk.EnvScopedObjectReference{Id: environmentId})
	if networkId != "" {
		spec.SetNetwork(cmk.EnvScopedObjectReference{Id: networkId})
	}
	if byokId != "" {
		spec.SetByok(cmk.GlobalObjectReference{Id: byokId})
	}
	createClusterRequest := cmk.CmkV2Cluster{Spec: spec}
	createClusterRequestJson, err := json.Marshal(createClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Cluster: error marshaling %#v to json: %s", createClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Kafka Cluster: %s", createClusterRequestJson))

	createdKafkaCluster, _, err := executeKafkaCreate(c.cmkApiContext(ctx), c, &createClusterRequest)
	if err != nil {
		return diag.Errorf("error creating Kafka Cluster %q: %s", displayName, createDescriptiveError(err))
	}
	d.SetId(createdKafkaCluster.GetId())

	if err := waitForKafkaClusterToProvision(c.cmkApiContext(ctx), c, environmentId, d.Id(), clusterType); err != nil {
		return diag.Errorf("error waiting for Kafka Cluster %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	environment, _, err := executeEnvironmentRead(c.orgApiContext(ctx), c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Environment %q: %s", environmentId, createDescriptiveError(err))
	}
	if environment.StreamGovernanceConfig != nil {
		if err := waitForAnySchemaRegistryClusterToProvision(c.srcmApiContext(ctx), c, environmentId); err != nil {
			return diag.Errorf("error waiting for Schema Registry Cluster to provision: %s", createDescriptiveError(err))
		}
	}

	createdKafkaClusterJson, err := json.Marshal(createdKafkaCluster)
	if err != nil {
		return diag.Errorf("error creating Kafka Cluster %q: error marshaling %#v to json: %s", d.Id(), createdKafkaCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Kafka Cluster %q: %s", d.Id(), createdKafkaClusterJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return kafkaRead(ctx, d, meta)
}

func extractClusterType(d *schema.ResourceData) string {
	basicConfigBlock := d.Get(paramBasicCluster).([]interface{})
	standardConfigBlock := d.Get(paramStandardCluster).([]interface{})
	dedicatedConfigBlock := d.Get(paramDedicatedCluster).([]interface{})
	enterpriseConfigBlock := d.Get(paramEnterpriseCluster).([]interface{})
	freightConfigBlock := d.Get(paramFreightCluster).([]interface{})

	if len(basicConfigBlock) == 1 {
		return kafkaClusterTypeBasic
	} else if len(standardConfigBlock) == 1 {
		return kafkaClusterTypeStandard
	} else if len(dedicatedConfigBlock) == 1 {
		return kafkaClusterTypeDedicated
	} else if len(enterpriseConfigBlock) == 1 {
		return kafkaClusterTypeEnterprise
	} else if len(freightConfigBlock) == 1 {
		return kafkaClusterTypeFreight
	}
	return ""
}

func extractClusterTypeResourceDiff(d *schema.ResourceDiff) string {
	basicConfigBlock := d.Get(paramBasicCluster).([]interface{})
	standardConfigBlock := d.Get(paramStandardCluster).([]interface{})
	dedicatedConfigBlock := d.Get(paramDedicatedCluster).([]interface{})
	enterpriseConfigBlock := d.Get(paramEnterpriseCluster).([]interface{})
	freightConfigBlock := d.Get(paramFreightCluster).([]interface{})

	if len(basicConfigBlock) == 1 {
		return kafkaClusterTypeBasic
	} else if len(standardConfigBlock) == 1 {
		return kafkaClusterTypeStandard
	} else if len(dedicatedConfigBlock) == 1 {
		return kafkaClusterTypeDedicated
	} else if len(enterpriseConfigBlock) == 1 {
		return kafkaClusterTypeEnterprise
	} else if len(freightConfigBlock) == 1 {
		return kafkaClusterTypeFreight
	}
	return ""
}

func extractCku(d *schema.ResourceData) int32 {
	// d.Get() will return 0 if the key is not present
	return int32(d.Get(paramDedicatedCku).(int))
}

func extractEncryptionKey(d *schema.ResourceData) string {
	// d.Get() will return "" if the key is not present
	return d.Get(paramDedicatedEncryptionKey).(string)
}

func kafkaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	c := meta.(*Client)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	req := c.cmkClient.ClustersCmkV2Api.DeleteCmkV2Cluster(c.cmkApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return nil
}

func kafkaImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	envIDAndClusterID := d.Id()
	parts := strings.Split(envIDAndClusterID, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Kafka Cluster: invalid format: expected '<env ID>/<Kafka cluster ID>'")
	}

	environmentId := parts[0]
	clusterId := parts[1]
	d.SetId(clusterId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readKafkaClusterAndSetAttributes(ctx, d, meta, environmentId, clusterId); err != nil {
		return nil, fmt.Errorf("error importing Kafka Cluster %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func executeKafkaRead(ctx context.Context, c *Client, environmentId string, clusterId string) (cmk.CmkV2Cluster, *http.Response, error) {
	req := c.cmkClient.ClustersCmkV2Api.GetCmkV2Cluster(c.cmkApiContext(ctx), clusterId).Environment(environmentId)
	return req.Execute()
}

func kafkaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	clusterId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readKafkaClusterAndSetAttributes(ctx, d, meta, environmentId, clusterId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readKafkaClusterAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	cluster, resp, err := executeKafkaRead(c.cmkApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Cluster %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Kafka Cluster %q in TF state because Kafka Cluster could not be found on the server", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Cluster %q: error marshaling %#v to json: %s", clusterId, cluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Cluster %q: %s", d.Id(), clusterJson), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	if _, err := setKafkaClusterAttributes(d, cluster); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Kafka Cluster %q", d.Id()), map[string]interface{}{kafkaClusterLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func basicClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		ExactlyOneOf:  acceptedClusterTypes,
		ConflictsWith: []string{paramConfluentCustomerKey},
	}
}

func standardClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		ExactlyOneOf:  acceptedClusterTypes,
		ConflictsWith: []string{paramConfluentCustomerKey},
	}
}

func dedicatedClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramCku: {
					Type:        schema.TypeInt,
					Required:    true,
					Description: "The number of Confluent Kafka Units (CKUs) for Dedicated cluster types. MULTI_ZONE dedicated clusters must have at least two CKUs.",
					// TODO: add validation for CKUs >= 2 of MULTI_ZONE dedicated clusters
					ValidateFunc: validation.IntAtLeast(1),
				},
				paramEncryptionKey: {
					Type:        schema.TypeString,
					Optional:    true,
					Computed:    true,
					Description: "The ID of the encryption key that is used to encrypt the data in the Kafka cluster.",
				},
				paramZones: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Optional:    true,
					Computed:    true,
					ForceNew:    true,
					Description: "The list of zones the cluster is in. Zone could be user specified for single-zone private network Dedicated cluster types, otherwise is auto-selected.",
				},
			},
			CustomizeDiff: resourceKafkaClusterCustomizeDiff,
		},
		ExactlyOneOf: acceptedClusterTypes,
	}
}

func resourceKafkaClusterCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	if diff.HasChange("paramZones") {
		zones := diff.Get("paramZones").([]interface{})
		if len(zones) > 1 {
			return fmt.Errorf("zone selection must contain exactly one element if specified by the user, but received %d", len(zones))
		}
	}
	return nil
}

func enterpriseClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		ExactlyOneOf: acceptedClusterTypes,
	}
}

func freightClusterSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramZones: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The list of zones the cluster is in.",
				},
			},
		},
		ExactlyOneOf:  acceptedClusterTypes,
		ConflictsWith: []string{paramConfluentCustomerKey},
	}
}

func byokSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The ID of the Confluent key that is used to encrypt the data in the Kafka cluster.",
				},
			},
		},
		ConflictsWith: []string{paramBasicCluster, paramStandardCluster, paramFreightCluster},
	}
}

func ckuCheck(cku int32, availability string) error {
	if cku < 1 && availability == singleZone {
		return fmt.Errorf("single-zone dedicated clusters must have at least 1 CKU")
	} else if cku < 2 && availability == multiZone {
		return fmt.Errorf("multi-zone dedicated clusters must have at least 2 CKUs")
	}
	return nil
}

func setKafkaClusterAttributes(d *schema.ResourceData, cluster cmk.CmkV2Cluster) (*schema.ResourceData, error) {
	if err := d.Set(paramApiVersion, cluster.GetApiVersion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramKind, cluster.GetKind()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDisplayName, cluster.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAvailability, cluster.Spec.GetAvailability()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, cluster.Spec.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, cluster.Spec.GetRegion()); err != nil {
		return nil, err
	}

	// Reset all 5 cluster types since only one of these 5 should be set
	if err := d.Set(paramBasicCluster, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramStandardCluster, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramDedicatedCluster, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramEnterpriseCluster, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramFreightCluster, []interface{}{}); err != nil {
		return nil, err
	}

	// Set a specific cluster type
	if cluster.Spec.Config.CmkV2Basic != nil {
		if err := d.Set(paramBasicCluster, []interface{}{make(map[string]string)}); err != nil {
			return nil, err
		}
	} else if cluster.Spec.Config.CmkV2Standard != nil {
		if err := d.Set(paramStandardCluster, []interface{}{make(map[string]string)}); err != nil {
			return nil, err
		}
	} else if cluster.Spec.Config.CmkV2Dedicated != nil {
		if err := d.Set(paramDedicatedCluster, []interface{}{map[string]interface{}{
			paramCku:           cluster.Status.GetCku(),
			paramEncryptionKey: cluster.Spec.Config.CmkV2Dedicated.GetEncryptionKey(),
			paramZones:         cluster.Spec.Config.CmkV2Dedicated.GetZones(),
		}}); err != nil {
			return nil, err
		}
	} else if cluster.Spec.Config.CmkV2Enterprise != nil {
		if err := d.Set(paramEnterpriseCluster, []interface{}{make(map[string]string)}); err != nil {
			return nil, err
		}
	} else if cluster.Spec.Config.CmkV2Freight != nil {
		if err := d.Set(paramFreightCluster, []interface{}{map[string]interface{}{
			paramZones: cluster.Spec.Config.CmkV2Freight.GetZones(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := d.Set(paramBootStrapEndpoint, cluster.Spec.GetKafkaBootstrapEndpoint()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRestEndpoint, cluster.Spec.GetHttpEndpoint()); err != nil {
		return nil, err
	}
	rbacCrn, err := clusterCrnToRbacClusterCrn(cluster.Metadata.GetResourceName())
	if err != nil {
		return nil, fmt.Errorf("error reading Kafka Cluster %q: could not construct %s", d.Id(), paramRbacCrn)
	}
	if err := d.Set(paramRbacCrn, rbacCrn); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, cluster.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, cluster.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramConfluentCustomerKey, paramId, cluster.Spec.Byok.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(cluster.GetId())
	return d, nil
}

func optionalNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Optional:    true,
		Computed:    true,
		Description: "Network represents a network (VPC) in Confluent Cloud. All Networks exist within Confluent-managed cloud provider accounts.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The unique identifier for the network.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(n-|nr-)"), "the network ID must start with 'n-' or 'nr-'"),
				},
			},
		},
	}
}

func optionalNetworkDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "Network represents a network (VPC) in Confluent Cloud. All Networks exist within Confluent-managed cloud provider accounts.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The unique identifier for the network.",
				},
			},
		},
	}
}

func optionalByokDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the Confluent key that is used to encrypt the data in the Kafka cluster.",
				},
			},
		},
	}
}

func kafkaClusterImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaClusters,
	}
}

func loadAllKafkaClusters(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
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
		kafkaClustersJson, err := json.Marshal(kafkaClusters)
		if err != nil {
			return instances, diag.Errorf("error reading Kafka Clusters in Environment %q: error marshaling %#v to json: %s", environment.GetId(), kafkaClusters, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Clusters in Environment %q: %s", environment.GetId(), kafkaClustersJson))

		for _, kafkaCluster := range kafkaClusters {
			instanceId := fmt.Sprintf("%s/%s", environment.GetId(), kafkaCluster.GetId())
			instances[instanceId] = toValidTerraformResourceName(kafkaCluster.Spec.GetDisplayName())
		}
	}
	return instances, nil
}
