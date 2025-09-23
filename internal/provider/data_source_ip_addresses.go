// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"fmt"
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking-ip/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
	"strconv"
	"time"
)

const (
	paramIpAddresses  = "ip_addresses"
	paramIpPrefix     = "ip_prefix"
	paramServices     = "services"
	paramAddressTypes = "address_types"
	paramClouds       = "clouds"
	paramRegions      = "regions"

	paramAddressType = "address_type"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing service accounts using IP Addresses API
	// https://docs.confluent.io/cloud/current/api.html#tag/IP-Addresses-(networkingv1)/operation/listNetworkingV1IpAddresses
	listIPAddressesPageSize = 99
)

func ipAddressesDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: ipAddressesDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramFilter: {
				MaxItems:    1,
				Optional:    true,
				Type:        schema.TypeList,
				Description: "Schema filters.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramClouds: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for cloud. Pass multiple times to see results matching any of the values.",
						},
						paramRegions: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for region. Pass multiple times to see results matching any of the values.",
						},
						paramServices: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for services. Pass multiple times to see results matching any of the values.",
						},
						paramAddressTypes: {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Optional:    true,
							Description: "Filter the results by exact match for address_type. Pass multiple times to see results matching any of the values.\n\n",
						},
					},
				},
			},
			paramIpAddresses: ipAddressesSchema(),
		},
	}
}

func ipAddressesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramApiVersion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "API Version defines the schema version of this representation of a IP Address.",
				},
				paramKind: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Kind defines the object this IP Address represents.",
				},
				paramIpPrefix: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The IP Address range.",
				},
				paramServices: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Computed:    true,
					Description: "The service types that will use the address.",
				},
				paramCloud: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The cloud service provider in which the address exists.",
				},
				paramRegion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The region/location where the IP Address is in use.",
				},
				paramAddressType: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Whether the address is used for egress or ingress.",
				},
			},
		},
	}
}

func ipAddressesDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "Reading IP Addresses")

	clouds := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramClouds)).([]interface{}))
	regions := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramRegions)).([]interface{}))
	services := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramServices)).([]interface{}))
	addressTypes := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramAddressTypes)).([]interface{}))

	c := meta.(*Client)
	ipAddresses, err := loadIPAddresses(c.netIPApiContext(ctx), c, clouds, regions, services, addressTypes)
	if err != nil {
		return diag.Errorf("error reading IP Addresses: %s", createDescriptiveError(err))
	}
	result := make([]map[string]interface{}, len(ipAddresses))
	for i, ipAddress := range ipAddresses {
		result[i] = map[string]interface{}{
			paramApiVersion:  ipAddress.GetApiVersion(),
			paramKind:        ipAddress.GetKind(),
			paramIpPrefix:    ipAddress.GetIpPrefix(),
			paramServices:    ipAddress.GetServices().Items,
			paramCloud:       ipAddress.GetCloud(),
			paramRegion:      ipAddress.GetRegion(),
			paramAddressType: ipAddress.GetAddressType(),
		}
	}

	if err := d.Set(paramIpAddresses, result); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func loadIPAddresses(ctx context.Context, c *Client, clouds, regions, services, addressTypes []string) ([]net.NetworkingV1IpAddress, error) {
	ipAddresses := make([]net.NetworkingV1IpAddress, 0)

	allIPAddressesAreCollected := false
	pageToken := ""
	for !allIPAddressesAreCollected {
		ipAddressesPageList, resp, err := executeListIpAddresses(c.netIPApiContext(ctx), c, clouds, regions, services, addressTypes, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading IP Addresses: %s", createDescriptiveError(err, resp))
		}
		ipAddresses = append(ipAddresses, ipAddressesPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := ipAddressesPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allIPAddressesAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading IP Addresses: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allIPAddressesAreCollected = true
		}
	}
	return ipAddresses, nil
}

func executeListIpAddresses(ctx context.Context, c *Client, clouds, regions, services, addressTypes []string, pageToken string) (net.NetworkingV1IpAddressList, *http.Response, error) {
	request := c.netIpClient.IPAddressesNetworkingV1Api.ListNetworkingV1IpAddresses(c.netIPApiContext(ctx)).PageSize(listIPAddressesPageSize)
	if len(clouds) > 0 {
		request = request.Cloud(clouds)
	}
	if len(regions) > 0 {
		request = request.Region(regions)
	}
	if len(services) > 0 {
		request = request.Services(services)
	}
	if len(addressTypes) > 0 {
		request = request.AddressType(addressTypes)
	}
	if pageToken != "" {
		request = request.PageToken(pageToken)
	}
	return request.Execute()
}
