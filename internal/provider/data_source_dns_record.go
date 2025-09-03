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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dnsRecordDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dnsRecordDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the DNS Record, for example, `dnsrec-abc123`.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramDomain: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramGateway:                gatewayDataSourceSchema(),
			paramPrivateLinkAccessPoint: privateLinkAccessPointDataSourceSchema(),
		},
	}
}

func gatewayDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func privateLinkAccessPointDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func dnsRecordDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dnsRecordId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading DNS Record %q=%q", paramId, dnsRecordId), map[string]interface{}{dnsRecordKey: dnsRecordId})

	c := meta.(*Client)
	request := c.netAccessPointClient.DNSRecordsNetworkingV1Api.GetNetworkingV1DnsRecord(c.netAPApiContext(ctx), dnsRecordId).Environment(environmentId)
	dnsRecord, resp, err := c.netAccessPointClient.DNSRecordsNetworkingV1Api.GetNetworkingV1DnsRecordExecute(request)
	if err != nil {
		return diag.Errorf("error reading DNS Record %q: %s", dnsRecordId, createDescriptiveError(err, resp))
	}
	dnsRecordJson, err := json.Marshal(dnsRecord)
	if err != nil {
		return diag.Errorf("error reading DNS Record %q: error marshaling %#v to json: %s", dnsRecordId, dnsRecord, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched DNS Record %q: %s", dnsRecordId, dnsRecordJson), map[string]interface{}{dnsRecordKey: dnsRecordId})

	if _, err := setDnsRecordAttributes(d, dnsRecord); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
