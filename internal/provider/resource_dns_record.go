// Copyright 2024 Confluent Inc. All Rights Reserved.
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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	netap "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
)

const (
	paramDomain                 = "domain"
	paramPrivateLinkAccessPoint = "private_link_access_point"
	privateLinkAccessPoint      = "PrivateLinkAccessPoint"
)

var acceptedDnsRecordConfig = []string{paramPrivateLinkAccessPoint}

func dnsRecordResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: dnsRecordCreate,
		ReadContext:   dnsRecordRead,
		UpdateContext: dnsRecordUpdate,
		DeleteContext: dnsRecordDelete,
		Importer: &schema.ResourceImporter{
			StateContext: dnsRecordImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramDomain: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			paramPrivateLinkAccessPoint: privateLinkAccessPointSchema(),
			paramGateway:                requiredGateway(),
			paramEnvironment:            environmentSchema(),
		},
	}
}

func privateLinkAccessPointSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		ExactlyOneOf: acceptedDnsRecordConfig,
	}
}

func dnsRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	domain := d.Get(paramDomain).(string)
	gatewayId := extractStringValueFromBlock(d, paramGateway, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	isPrivateLinkAccessPoint := len(d.Get(paramPrivateLinkAccessPoint).([]interface{})) > 0

	spec := netap.NewNetworkingV1DnsRecordSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	spec.SetDomain(domain)
	spec.SetGateway(netap.EnvScopedObjectReference{Id: gatewayId})
	spec.SetEnvironment(netap.ObjectReference{Id: environmentId})

	config := netap.NetworkingV1DnsRecordSpecConfigOneOf{}
	if isPrivateLinkAccessPoint {
		config.NetworkingV1PrivateLinkAccessPoint = &netap.NetworkingV1PrivateLinkAccessPoint{
			Kind:       privateLinkAccessPoint,
			ResourceId: extractStringValueFromBlock(d, paramPrivateLinkAccessPoint, paramId),
		}
		spec.SetConfig(config)
	} else {
		return diag.Errorf("None of %q blocks was provided for confluent_dns_record resource", paramPrivateLinkAccessPoint)
	}

	createDnsRecordRequest := netap.NetworkingV1DnsRecord{Spec: spec}
	createDnsRecordRequestJson, err := json.Marshal(createDnsRecordRequest)
	if err != nil {
		return diag.Errorf("error creating DNS Record: error marshaling %#v to json: %s", createDnsRecordRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new DNS Record: %s", createDnsRecordRequestJson))

	req := c.netAccessPointClient.DNSRecordsNetworkingV1Api.CreateNetworkingV1DnsRecord(c.netAPApiContext(ctx)).NetworkingV1DnsRecord(createDnsRecordRequest)
	createdDnsRecord, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating DNS Record %q: %s", createdDnsRecord.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdDnsRecord.GetId())

	if err := waitForDnsRecordToProvision(c.netAPApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for DNS Record %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdDnsRecordJson, err := json.Marshal(createdDnsRecord)
	if err != nil {
		return diag.Errorf("error creating DNS Record %q: error marshaling %#v to json: %s", d.Id(), createdDnsRecord, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating DNS Record %q: %s", d.Id(), createdDnsRecordJson), map[string]interface{}{dnsRecordKey: d.Id()})

	return dnsRecordRead(ctx, d, meta)
}

func executeDnsRecordRead(ctx context.Context, c *Client, environmentId string, dnsRecordId string) (netap.NetworkingV1DnsRecord, *http.Response, error) {
	req := c.netAccessPointClient.DNSRecordsNetworkingV1Api.GetNetworkingV1DnsRecord(c.netAPApiContext(ctx), dnsRecordId).Environment(environmentId)
	return req.Execute()
}

func dnsRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading DNS Record %q", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})

	dnsRecordId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readDnsRecordAndSetAttributes(ctx, d, meta, environmentId, dnsRecordId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading DNS Record %q: %s", dnsRecordId, createDescriptiveError(err)))
	}

	return nil
}

func readDnsRecordAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, dnsRecordId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	dnsRecord, resp, err := executeDnsRecordRead(c.netAPApiContext(ctx), c, environmentId, dnsRecordId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading DNS Record %q: %s", dnsRecordId, createDescriptiveError(err)), map[string]interface{}{dnsRecordKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing DNS Record %q in TF state because DNS Record could not be found on the server", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	dnsRecordJson, err := json.Marshal(dnsRecord)
	if err != nil {
		return nil, fmt.Errorf("error reading DNS Record %q: error marshaling %#v to json: %s", dnsRecordId, dnsRecord, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched DNS Record %q: %s", d.Id(), dnsRecordJson), map[string]interface{}{dnsRecordKey: d.Id()})

	if _, err := setDnsRecordAttributes(d, dnsRecord); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading DNS Record %q", dnsRecordId), map[string]interface{}{dnsRecordKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func dnsRecordDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting DNS Record %q", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netAccessPointClient.DNSRecordsNetworkingV1Api.DeleteNetworkingV1DnsRecord(c.netAPApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting DNS Record %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForDnsRecordToBeDeleted(c.netAPApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for DNS Record %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting DNS Record %q", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})

	return nil
}

func dnsRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramPrivateLinkAccessPoint) {
		return diag.Errorf("error updating DNS Record %q: only %q, %q attribute can be updated for DNS Record", d.Id(), paramDisplayName, paramPrivateLinkAccessPoint)
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	updateDnsRecord := netap.NewNetworkingV1DnsRecordUpdate()

	updateDnsRecordSpec := netap.NewNetworkingV1DnsRecordSpecUpdate()
	updateDnsRecordSpec.SetEnvironment(netap.ObjectReference{Id: environmentId})
	if d.HasChange(paramDisplayName) {
		updateDnsRecordSpec.SetDisplayName(d.Get(paramDisplayName).(string))
	}
	if d.HasChange(paramPrivateLinkAccessPoint) {
		updateDnsRecordSpec.SetConfig(netap.NetworkingV1DnsRecordSpecUpdateConfigOneOf{
			NetworkingV1PrivateLinkAccessPoint: &netap.NetworkingV1PrivateLinkAccessPoint{
				Kind:       privateLinkAccessPoint,
				ResourceId: extractStringValueFromBlock(d, paramPrivateLinkAccessPoint, paramId),
			},
		})
	}

	updateDnsRecord.SetSpec(*updateDnsRecordSpec)
	updateDnsRecordRequestJson, err := json.Marshal(updateDnsRecord)
	if err != nil {
		return diag.Errorf("error updating DNS Record %q: error marshaling %#v to json: %s", d.Id(), updateDnsRecordRequestJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating DNS Record %q: %s", d.Id(), updateDnsRecordRequestJson), map[string]interface{}{dnsRecordKey: d.Id()})

	c := meta.(*Client)
	req := c.netAccessPointClient.DNSRecordsNetworkingV1Api.UpdateNetworkingV1DnsRecord(c.netAPApiContext(ctx), d.Id()).NetworkingV1DnsRecordUpdate(*updateDnsRecord)
	updatedDnsRecord, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating DNS Record %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedDnsRecordJson, err := json.Marshal(updatedDnsRecord)
	if err != nil {
		return diag.Errorf("error updating DNS Record %q: error marshaling %#v to json: %s", d.Id(), updatedDnsRecord, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating DNS Record %q: %s", d.Id(), updatedDnsRecordJson), map[string]interface{}{dnsRecordKey: d.Id()})
	return dnsRecordRead(ctx, d, meta)
}

func dnsRecordImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing DNS Record %q", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})

	envIDAndDnsRecordId := d.Id()
	parts := strings.Split(envIDAndDnsRecordId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing DNS Record: invalid format: expected '<env ID>/<DNS Record ID>'")
	}

	environmentId := parts[0]
	dnsRecordId := parts[1]
	d.SetId(dnsRecordId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readDnsRecordAndSetAttributes(ctx, d, meta, environmentId, dnsRecordId); err != nil {
		return nil, fmt.Errorf("error importing DNS Record %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing DNS Record %q", d.Id()), map[string]interface{}{dnsRecordKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setDnsRecordAttributes(d *schema.ResourceData, dnsRecord netap.NetworkingV1DnsRecord) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, dnsRecord.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDomain, dnsRecord.Spec.GetDomain()); err != nil {
		return nil, err
	}

	if dnsRecord.Spec.Config.NetworkingV1PrivateLinkAccessPoint != nil {
		if err := d.Set(paramPrivateLinkAccessPoint, []interface{}{map[string]interface{}{
			paramId: dnsRecord.Spec.Config.NetworkingV1PrivateLinkAccessPoint.GetResourceId(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramGateway, paramId, dnsRecord.Spec.Gateway.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, dnsRecord.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(dnsRecord.GetId())
	return d, nil
}
