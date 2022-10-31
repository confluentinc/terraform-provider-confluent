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
	"time"
)

const (
	paramConnectionTypes                         = "connection_types"
	paramCidr                                    = "cidr"
	paramZones                                   = "zones"
	paramPrivateLinkEndpointService              = "private_link_endpoint_service"
	paramPrivateLinkServiceAliases               = "private_link_service_aliases"
	paramPrivateServiceConnectServiceAttachments = "private_service_connect_service_attachments"
	paramDnsDomain                               = "dns_domain"
	paramZonalSubdomains                         = "zonal_subdomains"

	connectionTypePrivateLink    = "PRIVATELINK"
	connectionTypeTransitGateway = "TRANSITGATEWAY"
	connectionTypePeering        = "PEERING"

	networkingAPICreateTimeout = 2 * time.Hour
	networkingAPIDeleteTimeout = 5 * time.Hour
)

var acceptedConnectionTypes = []string{connectionTypePeering, connectionTypeTransitGateway, connectionTypePrivateLink}

func networkResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: networkCreate,
		ReadContext:   networkRead,
		UpdateContext: networkUpdate,
		DeleteContext: networkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: networkImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Network.",
				Optional:    true,
				Computed:    true,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider in which the network exists.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
				ForceNew:     true,
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider region where the network exists.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramConnectionTypes: connectionTypesSchema(),
			paramCidr: {
				Type:         schema.TypeString,
				Description:  "The IPv4 CIDR block to used for this network. Must be /16. Required for VPC peering and AWS TransitGateway.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+/16$`), "The IPv4 CIDR block must be /16."),
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
			},
			paramZones:       zonesSchema(),
			paramEnvironment: environmentSchema(),
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the Network.",
			},
			paramDnsDomain: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The root DNS domain for the network if applicable. Present on networks that support PrivateLink.",
			},
			paramZonalSubdomains: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Computed:    true,
				Description: "The DNS subdomain for each zone. Present on networks that support PrivateLink. Keys are zones and values are DNS domains.",
			},
			paramAws:   awsNetworkSchema(),
			paramAzure: azureNetworkSchema(),
			paramGcp:   gcpNetworkSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(networkingAPICreateTimeout),
			Delete: schema.DefaultTimeout(networkingAPIDeleteTimeout),
		},
	}
}

func awsNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpc: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The Confluent Cloud VPC ID.",
				},
				paramAccount: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The AWS account ID associated with the Confluent Cloud VPC.",
				},
				paramPrivateLinkEndpointService: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The endpoint service of the Confluent Cloud VPC (used for PrivateLink) if available.",
				},
			},
		},
	}
}

func azureNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateLinkServiceAliases: {
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed: true,
				},
			},
		},
	}
}

func gcpNetworkSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProject: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The GCP project.",
				},
				paramVpcNetwork: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The GCP VPC network name.",
				},
				paramPrivateServiceConnectServiceAttachments: {
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The mapping of zones to Private Service Connect service attachments if available. Keys are zones and values are [GCP Private Service Connect service attachment](https://cloud.google.com/vpc/docs/configure-private-service-connect-producer#api_7).",
				},
			},
		},
	}
}

func networkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)

	connectionTypes := convertToStringSlice(d.Get(paramConnectionTypes).([]interface{}))
	err := verifyListValues(connectionTypes, acceptedConnectionTypes, false)
	if err != nil {
		return diag.Errorf("input validation error reading Network's %q: %s", paramConnectionTypes, createDescriptiveError(err))
	}

	cidr, err := readCidr(d, cloud, connectionTypes)
	if err != nil {
		return diag.Errorf("input validation error reading Network's %q: %s", paramCidr, createDescriptiveError(err))
	}

	zones, err := readZones(d, cloud, connectionTypes)
	if err != nil {
		return diag.Errorf("input validation error reading Network's %q: %s", paramZones, createDescriptiveError(err))
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec := net.NewNetworkingV1NetworkSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	spec.SetCloud(cloud)
	spec.SetRegion(region)
	spec.SetConnectionTypes(net.NetworkingV1ConnectionTypes{
		Items: connectionTypes,
	})
	if cidr != "" {
		spec.SetCidr(cidr)
	}
	if len(zones) > 0 {
		spec.SetZones(zones)
	}
	spec.SetEnvironment(net.ObjectReference{Id: environmentId})

	createNetworkRequest := net.NetworkingV1Network{Spec: spec}
	createNetworkRequestJson, err := json.Marshal(createNetworkRequest)
	if err != nil {
		return diag.Errorf("error creating Network: error marshaling %#v to json: %s", createNetworkRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Network: %s", createNetworkRequestJson))

	createdNetwork, _, err := executeNetworkCreate(c.netApiContext(ctx), c, createNetworkRequest)
	if err != nil {
		return diag.Errorf("error creating Network %q: %s", createdNetwork.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdNetwork.GetId())

	if err := waitForNetworkToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Network %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdNetworkJson, err := json.Marshal(createdNetwork)
	if err != nil {
		return diag.Errorf("error creating Network %q: error marshaling %#v to json: %s", d.Id(), createdNetwork, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Network %q: %s", d.Id(), createdNetworkJson), map[string]interface{}{networkLoggingKey: d.Id()})

	return networkRead(ctx, d, meta)
}

func executeNetworkCreate(ctx context.Context, c *Client, network net.NetworkingV1Network) (net.NetworkingV1Network, *http.Response, error) {
	req := c.netClient.NetworksNetworkingV1Api.CreateNetworkingV1Network(c.netApiContext(ctx)).NetworkingV1Network(network)
	return req.Execute()
}

func executeNetworkRead(ctx context.Context, c *Client, environmentId string, networkId string) (net.NetworkingV1Network, *http.Response, error) {
	req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(ctx), networkId).Environment(environmentId)
	return req.Execute()
}

func networkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})

	networkId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readNetworkAndSetAttributes(ctx, d, meta, environmentId, networkId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Network %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readNetworkAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, networkId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	network, resp, err := executeNetworkRead(c.netApiContext(ctx), c, environmentId, networkId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Network %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{networkLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Network %q in TF state because Network could not be found on the server", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	networkJson, err := json.Marshal(network)
	if err != nil {
		return nil, fmt.Errorf("error reading Network %q: error marshaling %#v to json: %s", networkId, network, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Network %q: %s", d.Id(), networkJson), map[string]interface{}{networkLoggingKey: d.Id()})

	if _, err := setNetworkAttributes(d, network); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setNetworkAttributes(d *schema.ResourceData, network net.NetworkingV1Network) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, network.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, network.Spec.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, network.Spec.GetRegion()); err != nil {
		return nil, err
	}

	if err := d.Set(paramConnectionTypes, network.Spec.ConnectionTypes.Items); err != nil {
		return nil, err
	}
	if err := d.Set(paramCidr, network.Spec.GetCidr()); err != nil {
		return nil, err
	}
	if err := d.Set(paramZones, network.Spec.GetZones()); err != nil {
		return nil, err
	}

	if err := d.Set(paramDnsDomain, network.Status.GetDnsDomain()); err != nil {
		return nil, err
	}
	if err := d.Set(paramZonalSubdomains, network.Status.GetZonalSubdomains()); err != nil {
		return nil, err
	}

	// Set optional computed blocks
	if strings.EqualFold(paramAws, network.Spec.GetCloud()) {
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramVpc:                        network.Status.Cloud.NetworkingV1AwsNetwork.GetVpc(),
			paramAccount:                    network.Status.Cloud.NetworkingV1AwsNetwork.GetAccount(),
			paramPrivateLinkEndpointService: network.Status.Cloud.NetworkingV1AwsNetwork.GetPrivateLinkEndpointService()}}); err != nil {
			return nil, err
		}
	} else if strings.EqualFold(paramAzure, network.Spec.GetCloud()) {
		if err := d.Set(paramAzure, []interface{}{map[string]interface{}{
			paramPrivateLinkServiceAliases: network.Status.Cloud.NetworkingV1AzureNetwork.GetPrivateLinkServiceAliases()}}); err != nil {
			return nil, err
		}
	} else if strings.EqualFold(paramGcp, network.Spec.GetCloud()) {
		if err := d.Set(paramGcp, []interface{}{map[string]interface{}{
			paramProject:    network.Status.Cloud.NetworkingV1GcpNetwork.GetProject(),
			paramVpcNetwork: network.Status.Cloud.NetworkingV1GcpNetwork.GetVpcNetwork(),
			paramPrivateServiceConnectServiceAttachments: network.Status.Cloud.NetworkingV1GcpNetwork.GetPrivateServiceConnectServiceAttachments()}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, network.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, network.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(network.GetId())
	return d, nil
}

func networkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netClient.NetworksNetworkingV1Api.DeleteNetworkingV1Network(c.netApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Network %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})

	return nil
}

func networkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Network %q: only %q attribute can be updated for Network", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	updatedDisplayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateNetworkRequest := net.NewNetworkingV1NetworkUpdate()
	updateSpec := net.NewNetworkingV1NetworkSpecUpdate()
	updateSpec.SetDisplayName(updatedDisplayName)
	updateSpec.SetEnvironment(net.ObjectReference{Id: environmentId})
	updateNetworkRequest.SetSpec(*updateSpec)
	updateNetworkRequestJson, err := json.Marshal(updateNetworkRequest)
	if err != nil {
		return diag.Errorf("error updating Network %q: error marshaling %#v to json: %s", d.Id(), updateNetworkRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Network %q: %s", d.Id(), updateNetworkRequestJson), map[string]interface{}{networkLoggingKey: d.Id()})

	req := c.netClient.NetworksNetworkingV1Api.UpdateNetworkingV1Network(c.netApiContext(ctx), d.Id()).NetworkingV1NetworkUpdate(*updateNetworkRequest)
	updatedNetwork, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Network %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedNetworkJson, err := json.Marshal(updatedNetwork)
	if err != nil {
		return diag.Errorf("error updating Network %q: error marshaling %#v to json: %s", d.Id(), updatedNetwork, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Network %q: %s", d.Id(), updatedNetworkJson), map[string]interface{}{networkLoggingKey: d.Id()})
	return networkRead(ctx, d, meta)
}

func networkImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})

	envIDAndNetworkId := d.Id()
	parts := strings.Split(envIDAndNetworkId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Network: invalid format: expected '<env ID>/<network ID>'")
	}

	environmentId := parts[0]
	networkId := parts[1]
	d.SetId(networkId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readNetworkAndSetAttributes(ctx, d, meta, environmentId, networkId); err != nil {
		return nil, fmt.Errorf("error importing Network %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Network %q", d.Id()), map[string]interface{}{networkLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func connectionTypesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MinItems: 1,
		Elem:     &schema.Schema{Type: schema.TypeString},
		ForceNew: true,
		// TODO: ValidateFunc and ValidateDiagFunc are not yet supported on lists or sets.
		// ValidateFunc: validation.StringInSlice(acceptedAvailabilityZones, false)
	}
}

func zonesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		MinItems: 3,
		MaxItems: 3,
		Elem:     &schema.Schema{Type: schema.TypeString},
		ForceNew: true,
		Description: "The 3 availability zones for this network. They can optionally be specified for only AWS" +
			" networks used with PrivateLink. Otherwise, they are automatically chosen by Confluent Cloud.",
	}
}

// cidr is required for VPC peering and AWS TransitGateway.
// TODO: update error messages
func validateCidr(cidr, cloud string, connectionTypes []string) error {
	isAwsTransitGateway := cloud == strings.ToUpper(paramAws) && stringInSlice(connectionTypeTransitGateway, connectionTypes, false)
	isVpcPeering := stringInSlice(connectionTypePeering, connectionTypes, false)
	isCidrRequired := isAwsTransitGateway || isVpcPeering
	if cidr == "" && isCidrRequired {
		return fmt.Errorf("cidr is required for VPC peering and AWS TransitGateway")
	}
	isPrivateLink := stringInSlice(connectionTypePrivateLink, connectionTypes, false)
	if cidr != "" && isPrivateLink {
		return fmt.Errorf("cidr is not allowed for PRIVATE_LINK networks")
	}
	return nil
}

// zones can only be specified for AWS networks used with PrivateLink or for GCP networks used with Private Service Connect
func validateZones(zones []string, cloud string, connectionTypes []string) error {
	isAwsPrivateLinkOrGcpPrivateServiceConnect := (cloud == strings.ToUpper(paramAws) || cloud == strings.ToUpper(paramGcp)) && stringInSlice(connectionTypePrivateLink, connectionTypes, false)
	if len(zones) > 0 && !isAwsPrivateLinkOrGcpPrivateServiceConnect {
		return fmt.Errorf("zones can only be specified for AWS networks used with PrivateLink or for GCP networks used with Private Service Connect")
	}
	return nil
}

func readCidr(d *schema.ResourceData, cloud string, connectionTypes []string) (string, error) {
	cidr := d.Get(paramCidr).(string)
	if err := validateCidr(cidr, cloud, connectionTypes); err != nil {
		return "", err
	}
	return cidr, nil
}

func readZones(d *schema.ResourceData, cloud string, connectionTypes []string) ([]string, error) {
	zones := convertToStringSlice(d.Get(paramZones).([]interface{}))
	if err := validateZones(zones, cloud, connectionTypes); err != nil {
		return nil, err
	}
	return zones, nil
}
