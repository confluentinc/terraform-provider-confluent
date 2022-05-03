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
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramGcp                = "gcp"
	paramVpc                = "vpc"
	paramRoutes             = "routes"
	paramCustomerRegion     = "customer_region"
	paramTenant             = "tenant"
	paramVnet               = "vnet"
	paramProject            = "project"
	paramVpcNetwork         = "vpc_network"
	paramImportCustomRoutes = "import_custom_routes"
	awsPeeringKind          = "AwsPeering"
	azurePeeringKind        = "AzurePeering"
	gcpPeeringKind          = "GcpPeering"
)

var acceptedCloudProvidersForPeering = []string{paramAws, paramAzure, paramGcp}
var paramAwsAccount = fmt.Sprintf("%s.0.%s", paramAws, paramAccount)
var paramAwsVpc = fmt.Sprintf("%s.0.%s", paramAws, paramVpc)
var paramAwsRoutes = fmt.Sprintf("%s.0.%s", paramAws, paramRoutes)
var paramAwsCustomerRegion = fmt.Sprintf("%s.0.%s", paramAws, paramCustomerRegion)
var paramAzureTenant = fmt.Sprintf("%s.0.%s", paramAzure, paramTenant)
var paramAzureVnet = fmt.Sprintf("%s.0.%s", paramAzure, paramVnet)
var paramAzureCustomerRegion = fmt.Sprintf("%s.0.%s", paramAzure, paramCustomerRegion)
var paramGcpProject = fmt.Sprintf("%s.0.%s", paramGcp, paramProject)
var paramGcpVpcNetwork = fmt.Sprintf("%s.0.%s", paramGcp, paramVpcNetwork)
var paramGcpImportCustomRoutes = fmt.Sprintf("%s.0.%s", paramGcp, paramImportCustomRoutes)

func peeringResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: peeringCreate,
		ReadContext:   peeringRead,
		UpdateContext: peeringUpdate,
		DeleteContext: peeringDelete,
		Importer: &schema.ResourceImporter{
			StateContext: peeringImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Peering.",
				Optional:    true,
				Computed:    true,
			},
			paramAws:         awsPeeringSchema(),
			paramAzure:       azurePeeringSchema(),
			paramGcp:         gcpPeeringSchema(),
			paramNetwork:     requiredNetworkSchema(),
			paramEnvironment: environmentSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(networkingAPICreateTimeout),
			Delete: schema.DefaultTimeout(networkingAPIDeleteTimeout),
		},
	}
}

func peeringCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	isAwsPeering := len(d.Get(paramAws).([]interface{})) > 0
	isAzurePeering := len(d.Get(paramAzure).([]interface{})) > 0
	isGcpPeering := len(d.Get(paramGcp).([]interface{})) > 0

	spec := net.NewNetworkingV1PeeringSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	if isAwsPeering {
		account := d.Get(paramAwsAccount).(string)
		vpc := d.Get(paramAwsVpc).(string)
		routes := convertToStringSlice(d.Get(paramAwsRoutes).([]interface{}))
		customerRegion := d.Get(paramAwsCustomerRegion).(string)
		spec.SetCloud(net.NetworkingV1PeeringSpecCloudOneOf{NetworkingV1AwsPeering: net.NewNetworkingV1AwsPeering(awsPeeringKind, account, vpc, routes, customerRegion)})
	} else if isAzurePeering {
		tenant := d.Get(paramAzureTenant).(string)
		vnet := d.Get(paramAzureVnet).(string)
		customerRegion := d.Get(paramAzureCustomerRegion).(string)
		spec.SetCloud(net.NetworkingV1PeeringSpecCloudOneOf{NetworkingV1AzurePeering: net.NewNetworkingV1AzurePeering(azurePeeringKind, tenant, vnet, customerRegion)})
	} else if isGcpPeering {
		project := d.Get(paramGcpProject).(string)
		vpcNetwork := d.Get(paramGcpVpcNetwork).(string)
		importCustomerRoutes := d.Get(paramGcpImportCustomRoutes).(bool)
		gcpPeering := net.NewNetworkingV1GcpPeering(gcpPeeringKind, project, vpcNetwork)
		gcpPeering.SetImportCustomRoutes(importCustomerRoutes)
		spec.SetCloud(net.NetworkingV1PeeringSpecCloudOneOf{NetworkingV1GcpPeering: gcpPeering})
	} else {
		return diag.Errorf("None of %q, %q, %q blocks was provided for confluent_peering resource", paramAws, paramAzure, paramGcp)
	}
	spec.SetNetwork(net.ObjectReference{Id: networkId})
	spec.SetEnvironment(net.ObjectReference{Id: environmentId})

	createPeeringRequest := net.NetworkingV1Peering{Spec: spec}
	createPeeringRequestJson, err := json.Marshal(createPeeringRequest)
	if err != nil {
		return diag.Errorf("error creating Peering: error marshaling %#v to json: %s", createPeeringRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Peering: %s", createPeeringRequestJson))

	createdPeering, _, err := executePeeringCreate(c.netApiContext(ctx), c, createPeeringRequest)
	if err != nil {
		return diag.Errorf("error creating Peering %q: %s", createdPeering.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdPeering.GetId())

	if err := waitForPeeringToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Peering %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdPeeringJson, err := json.Marshal(createdPeering)
	if err != nil {
		return diag.Errorf("error creating Peering %q: error marshaling %#v to json: %s", d.Id(), createdPeering, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Peering %q: %s", d.Id(), createdPeeringJson), map[string]interface{}{peeringLoggingKey: d.Id()})

	return peeringRead(ctx, d, meta)
}

func executePeeringCreate(ctx context.Context, c *Client, peering net.NetworkingV1Peering) (net.NetworkingV1Peering, *http.Response, error) {
	req := c.netClient.PeeringsNetworkingV1Api.CreateNetworkingV1Peering(c.netApiContext(ctx)).NetworkingV1Peering(peering)
	return req.Execute()
}

func executePeeringRead(ctx context.Context, c *Client, environmentId string, peeringId string) (net.NetworkingV1Peering, *http.Response, error) {
	req := c.netClient.PeeringsNetworkingV1Api.GetNetworkingV1Peering(c.netApiContext(ctx), peeringId).Environment(environmentId)
	return req.Execute()
}

func peeringRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})

	peeringId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readPeeringAndSetAttributes(ctx, d, meta, environmentId, peeringId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Peering %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readPeeringAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, peeringId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	peering, resp, err := executePeeringRead(c.netApiContext(ctx), c, environmentId, peeringId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Peering %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{peeringLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Peering %q in TF state because Peering could not be found on the server", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	peeringJson, err := json.Marshal(peering)
	if err != nil {
		return nil, fmt.Errorf("error reading Peering %q: error marshaling %#v to json: %s", peeringId, peering, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Peering %q: %s", d.Id(), peeringJson), map[string]interface{}{peeringLoggingKey: d.Id()})

	if _, err := setPeeringAttributes(d, peering); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setPeeringAttributes(d *schema.ResourceData, peering net.NetworkingV1Peering) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, peering.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if peering.Spec.Cloud.NetworkingV1AwsPeering != nil {
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramAccount:        peering.Spec.Cloud.NetworkingV1AwsPeering.GetAccount(),
			paramVpc:            peering.Spec.Cloud.NetworkingV1AwsPeering.GetVpc(),
			paramRoutes:         peering.Spec.Cloud.NetworkingV1AwsPeering.GetRoutes(),
			paramCustomerRegion: peering.Spec.Cloud.NetworkingV1AwsPeering.GetCustomerRegion(),
		}}); err != nil {
			return nil, err
		}
	} else if peering.Spec.Cloud.NetworkingV1AzurePeering != nil {
		if err := d.Set(paramAzure, []interface{}{map[string]interface{}{
			paramTenant:         peering.Spec.Cloud.NetworkingV1AzurePeering.GetTenant(),
			paramVnet:           peering.Spec.Cloud.NetworkingV1AzurePeering.GetVnet(),
			paramCustomerRegion: peering.Spec.Cloud.NetworkingV1AzurePeering.GetCustomerRegion(),
		}}); err != nil {
			return nil, err
		}
	} else if peering.Spec.Cloud.NetworkingV1GcpPeering != nil {
		if err := d.Set(paramGcp, []interface{}{map[string]interface{}{
			paramProject:            peering.Spec.Cloud.NetworkingV1GcpPeering.GetProject(),
			paramVpcNetwork:         peering.Spec.Cloud.NetworkingV1GcpPeering.GetVpcNetwork(),
			paramImportCustomRoutes: peering.Spec.Cloud.NetworkingV1GcpPeering.GetImportCustomRoutes(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, peering.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, peering.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(peering.GetId())
	return d, nil
}

func peeringDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netClient.PeeringsNetworkingV1Api.DeleteNetworkingV1Peering(c.netApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Peering %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForPeeringToBeDeleted(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Peering %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})

	return nil
}

func peeringUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Peering %q: only %q attribute can be updated for Peering", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	updatedDisplayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updatePeeringRequest := net.NewNetworkingV1PeeringUpdate()
	updateSpec := net.NewNetworkingV1PeeringSpecUpdate()
	updateSpec.SetDisplayName(updatedDisplayName)
	updateSpec.SetEnvironment(net.ObjectReference{Id: environmentId})
	updatePeeringRequest.SetSpec(*updateSpec)
	updatePeeringRequestJson, err := json.Marshal(updatePeeringRequest)
	if err != nil {
		return diag.Errorf("error updating Peering %q: error marshaling %#v to json: %s", d.Id(), updatePeeringRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Peering %q: %s", d.Id(), updatePeeringRequestJson), map[string]interface{}{peeringLoggingKey: d.Id()})

	req := c.netClient.PeeringsNetworkingV1Api.UpdateNetworkingV1Peering(c.netApiContext(ctx), d.Id()).NetworkingV1PeeringUpdate(*updatePeeringRequest)
	updatedPeering, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Peering %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedPeeringJson, err := json.Marshal(updatedPeering)
	if err != nil {
		return diag.Errorf("error updating Peering %q: error marshaling %#v to json: %s", d.Id(), updatedPeering, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Peering %q: %s", d.Id(), updatedPeeringJson), map[string]interface{}{peeringLoggingKey: d.Id()})
	return peeringRead(ctx, d, meta)
}

func peeringImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})

	envIDAndPeeringId := d.Id()
	parts := strings.Split(envIDAndPeeringId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Peering: invalid format: expected '<env ID>/<peer ID>'")
	}

	environmentId := parts[0]
	peeringId := parts[1]
	d.SetId(peeringId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readPeeringAndSetAttributes(ctx, d, meta, environmentId, peeringId); err != nil {
		return nil, fmt.Errorf("error importing Peering %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Peering %q", d.Id()), map[string]interface{}{peeringLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func awsPeeringSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAccount: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "AWS account for VPC to peer with the network.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\d{12}$`), "AWS Account ID is expected to consist of exactly 12 digits."),
				},
				paramVpc: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The id of the AWS VPC to peer with.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^vpc-"), "AWS VPC ID must start with 'vpc-'."),
				},
				paramRoutes: {
					Type:        schema.TypeList,
					Required:    true,
					ForceNew:    true,
					MinItems:    1,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "List of routes for the peering.",
					// TODO: ValidateFunc and ValidateDiagFunc are not yet supported on lists or sets.
					// TODO: copy validation from confluent_network.cidr attribute
				},
				paramCustomerRegion: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "Region of customer VPC.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPeering,
	}
}

func azurePeeringSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramTenant: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "Customer Azure tenant.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramVnet: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "Customer VNet to peer with.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramCustomerRegion: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "Region of customer VNet.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPeering,
	}
}

func gcpPeeringSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProject: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The name of the GCP project.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramVpcNetwork: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The name of the GCP VPC network to peer with.",
					ValidateFunc: validation.StringIsNotEmpty,
				},
				paramImportCustomRoutes: {
					Type:        schema.TypeBool,
					Optional:    true,
					ForceNew:    true,
					Default:     false,
					Description: "Enable customer route import.",
				},
			},
		},
		ExactlyOneOf: acceptedCloudProvidersForPeering,
	}
}
