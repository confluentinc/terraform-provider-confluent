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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
	"strings"

	dns "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
)

const (
	paramGateway      = "gateway"
	paramForwardViaIp = "forward_via_ip"
	paramDomains      = "domains"
	paramDnsServerIps = "dns_server_ips"
	forwardViaIp      = "ForwardViaIp"
)

var acceptedDnsForwarderConfig = []string{paramForwardViaIp}

func dnsForwarderResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: dnsForwarderCreate,
		ReadContext:   dnsForwarderRead,
		UpdateContext: dnsForwarderUpdate,
		DeleteContext: dnsForwarderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: dnsForwarderImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramDomains: {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			paramForwardViaIp: forwardViaIpSchema(),
			paramGateway:      requiredGateway(),
			paramEnvironment:  environmentSchema(),
		},
	}
}

func forwardViaIpSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramDnsServerIps: {
					Type:     schema.TypeSet,
					Computed: true,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
		ExactlyOneOf: acceptedDnsForwarderConfig,
	}
}

func requiredGateway() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the gateway.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func dnsForwarderCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	gatewayId := extractStringValueFromBlock(d, paramGateway, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	domains := convertToStringSlice(d.Get(paramDomains).(*schema.Set).List())

	isForwardViaIp := len(d.Get(paramForwardViaIp).([]interface{})) > 0

	spec := dns.NewNetworkingV1DnsForwarderSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	if len(domains) > 0 {
		spec.SetDomains(domains)
	}

	config := dns.NetworkingV1DnsForwarderSpecConfigOneOf{}
	if isForwardViaIp {
		dnsServerIps := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramForwardViaIp, paramDnsServerIps)).(*schema.Set).List())
		config.NetworkingV1ForwardViaIp = &dns.NetworkingV1ForwardViaIp{DnsServerIps: dnsServerIps, Kind: forwardViaIp}
		spec.SetConfig(config)
	} else {
		return diag.Errorf("None of %q blocks was provided for confluent_dns_forwarder resource", paramDnsServerIps)
	}
	spec.SetGateway(dns.ObjectReference{Id: gatewayId})
	spec.SetEnvironment(dns.ObjectReference{Id: environmentId})

	createDnsForwarderRequest := dns.NetworkingV1DnsForwarder{Spec: spec}
	createDnsForwarderRequestJson, err := json.Marshal(createDnsForwarderRequest)
	if err != nil {
		return diag.Errorf("error creating DNS Forwarder: error marshaling %#v to json: %s", createDnsForwarderRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new DnsForwarder: %s", createDnsForwarderRequestJson))

	req := c.netDnsClient.DNSForwardersNetworkingV1Api.CreateNetworkingV1DnsForwarder(c.netDnsApiContext(ctx)).NetworkingV1DnsForwarder(createDnsForwarderRequest)
	createdDnsForwarder, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating DNS Forwarder %q: %s", createdDnsForwarder.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdDnsForwarder.GetId())

	if err := waitForDnsForwarderToProvision(c.netDnsApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for DNS Forwarder %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdDnsForwarderJson, err := json.Marshal(createdDnsForwarder)
	if err != nil {
		return diag.Errorf("error creating DNS Forwarder %q: error marshaling %#v to json: %s", d.Id(), createdDnsForwarder, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating DNS Forwarder %q: %s", d.Id(), createdDnsForwarderJson), map[string]interface{}{dnsForwarderKey: d.Id()})

	return dnsForwarderRead(ctx, d, meta)
}

func executeDnsForwarderRead(ctx context.Context, c *Client, environmentId string, dnsForwarderId string) (dns.NetworkingV1DnsForwarder, *http.Response, error) {
	req := c.netDnsClient.DNSForwardersNetworkingV1Api.GetNetworkingV1DnsForwarder(c.netDnsApiContext(ctx), dnsForwarderId).Environment(environmentId)
	return req.Execute()
}

func dnsForwarderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})

	dnsForwarderId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readDnsForwarderAndSetAttributes(ctx, d, meta, environmentId, dnsForwarderId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading DNS Forwarder %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readDnsForwarderAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, dnsForwarderId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	req := c.netDnsClient.DNSForwardersNetworkingV1Api.GetNetworkingV1DnsForwarder(c.netDnsApiContext(ctx), dnsForwarderId).Environment(environmentId)
	dnsForwarder, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading DNS Forwarder %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{dnsForwarderKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing DNS Forwarder %q in TF state because DNS Forwarder could not be found on the server", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	dnsForwarderJson, err := json.Marshal(dnsForwarder)
	if err != nil {
		return nil, fmt.Errorf("error reading DNS Forwarder %q: error marshaling %#v to json: %s", dnsForwarderId, dnsForwarder, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched DNS Forwarder %q: %s", d.Id(), dnsForwarderJson), map[string]interface{}{dnsForwarderKey: d.Id()})

	if _, err := setDnsForwarderAttributes(d, dnsForwarder); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func dnsForwarderDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netDnsClient.DNSForwardersNetworkingV1Api.DeleteNetworkingV1DnsForwarder(c.netDnsApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting DNS Forwarder %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})

	return nil
}

func dnsForwarderUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating DNS Forwarder %q: only %q attribute can be updated for DNS Forwarder", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	updatedDisplayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateDnsForwarderRequest := dns.NewNetworkingV1DnsForwarderUpdate()
	updateSpec := dns.NewNetworkingV1DnsForwarderSpecUpdate()
	updateSpec.SetDisplayName(updatedDisplayName)
	updateSpec.SetEnvironment(dns.ObjectReference{Id: environmentId})
	updateDnsForwarderRequest.SetSpec(*updateSpec)
	updateDnsForwarderRequestJson, err := json.Marshal(updateDnsForwarderRequest)
	if err != nil {
		return diag.Errorf("error updating DNS Forwarder %q: error marshaling %#v to json: %s", d.Id(), updateDnsForwarderRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating DNS Forwarder %q: %s", d.Id(), updateDnsForwarderRequestJson), map[string]interface{}{dnsForwarderKey: d.Id()})

	req := c.netDnsClient.DNSForwardersNetworkingV1Api.UpdateNetworkingV1DnsForwarder(c.netDnsApiContext(ctx), d.Id()).NetworkingV1DnsForwarderUpdate(*updateDnsForwarderRequest)
	updatedDnsForwarder, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating DNS Forwarder %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedDnsForwarderJson, err := json.Marshal(updatedDnsForwarder)
	if err != nil {
		return diag.Errorf("error updating DNS Forwarder %q: error marshaling %#v to json: %s", d.Id(), updatedDnsForwarder, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating DNS Forwarder %q: %s", d.Id(), updatedDnsForwarderJson), map[string]interface{}{dnsForwarderKey: d.Id()})
	return dnsForwarderRead(ctx, d, meta)
}

func dnsForwarderImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})

	envIDAndDnsForwarderId := d.Id()
	parts := strings.Split(envIDAndDnsForwarderId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing DNS Forwarder: invalid format: expected '<env ID>/<DNS Forwarder ID>'")
	}

	environmentId := parts[0]
	dnsForwarderId := parts[1]
	d.SetId(dnsForwarderId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readDnsForwarderAndSetAttributes(ctx, d, meta, environmentId, dnsForwarderId); err != nil {
		return nil, fmt.Errorf("error importing DNS Forwarder %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing DNS Forwarder %q", d.Id()), map[string]interface{}{dnsForwarderKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setDnsForwarderAttributes(d *schema.ResourceData, dnsForwarder dns.NetworkingV1DnsForwarder) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, dnsForwarder.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDomains, dnsForwarder.Spec.GetDomains()); err != nil {
		return nil, err
	}

	if dnsForwarder.Spec.Config.NetworkingV1ForwardViaIp != nil {
		if err := d.Set(paramForwardViaIp, []interface{}{map[string]interface{}{
			paramDnsServerIps: dnsForwarder.Spec.Config.NetworkingV1ForwardViaIp.GetDnsServerIps(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramGateway, paramId, dnsForwarder.Spec.Gateway.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, dnsForwarder.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(dnsForwarder.GetId())
	return d, nil
}
