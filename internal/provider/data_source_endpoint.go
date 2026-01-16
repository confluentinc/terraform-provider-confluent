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
	"strconv"
	"time"

	end "github.com/confluentinc/ccloud-sdk-go-v2-internal/endpoint/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramService          = "service"
	paramAccessPoint      = "access_point"
	paramIsPrivate        = "is_private"
	paramEndpointType     = "endpoint_type"
	paramEndpointResource = "resource"

	listEndpointsPageSize = 100
)

func endpointDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: endpointDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramFilter: {
				MaxItems:    1,
				MinItems:    1,
				Required:    true,
				Type:        schema.TypeList,
				Description: "Endpoint filters.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramEnvironment: environmentDataSourceSchema(),
						paramService: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Confluent Cloud service. Accepted values are: `KAFKA`, `SCHEMA_REGISTRY`, `FLINK`.",
						},
						paramEndpointResource: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The resource associated with the endpoint. The resource can be one of Kafka Cluster ID (example: `lkc-12345`), or Schema Registry Cluster ID (example: `lsrc-12345`). May be omitted if not associated with a resource.",
						},
						paramCloud: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The cloud service provider. Accepted values are: `AWS`, `GCP`, `AZURE`.",
						},
						paramRegion: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The cloud service provider region in which the resource is located.",
						},
						paramIsPrivate: {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Whether the endpoint is private (true) or public (false).",
						},
					},
				},
			},
			paramEndpoints: endpointsSchema(),
		},
	}
}

func endpointsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the Endpoint.",
				},
				paramApiVersion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "API Version defines the schema version of this representation of an Endpoint.",
				},
				paramKind: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Kind defines the object this Endpoint represents.",
				},
				paramCloud: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The cloud service provider.",
				},
				paramRegion: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The cloud service provider region in which the resource is located.",
				},
				paramService: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The Confluent Cloud service.",
				},
				paramIsPrivate: {
					Type:        schema.TypeBool,
					Computed:    true,
					Description: "Whether the endpoint is private (true) or public (false).",
				},
				paramConnectionType: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The network connection type.",
				},
				paramEndpoint: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The endpoint URL or address.",
				},
				paramEndpointType: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The endpoint type for the service.",
				},
				paramEnvironment: environmentDataSourceSchema(),
				paramEndpointResource: {
					Type: schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							paramId: {
								Type:     schema.TypeString,
								Computed: true,
							},
							paramKind: {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Computed:    true,
					Description: "The resource associated with the endpoint.",
				},
				paramGateway: gatewayDataSourceSchema(),
				paramAccessPoint: {
					Type: schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							paramId: {
								Type:     schema.TypeString,
								Computed: true,
							},
						},
					},
					Computed:    true,
					Description: "The access point to which this belongs.",
				},
			},
		},
		Description: "List of endpoints matching the filter criteria.",
	}
}

func endpointDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	environmentId := extractStringValueFromBlock(d, fmt.Sprintf("%s.0.%s", paramFilter, paramEnvironment), paramId)
	if environmentId == "" {
		return diag.Errorf("error reading endpoints: environment ID is required in filter")
	}

	service := d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramService)).(string)
	resource := d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramEndpointResource)).(string)
	cloud := d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramCloud)).(string)
	region := d.Get(fmt.Sprintf("%s.0.%s", paramFilter, paramRegion)).(string)
	var isPrivate *bool
	if v, ok := d.GetOk(fmt.Sprintf("%s.0.%s", paramFilter, paramIsPrivate)); ok {
		val := v.(bool)
		isPrivate = &val
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading Endpoints with filters: environment=%q, service=%q", environmentId, service))

	c := meta.(*Client)
	endpoints, err := loadEndpoints(c.endApiContext(ctx), c, environmentId, service, resource, cloud, region, isPrivate)
	if err != nil {
		return diag.Errorf("error reading endpoints: %s", createDescriptiveError(err))
	}

	result := make([]map[string]interface{}, len(endpoints))
	for i, endpoint := range endpoints {
		endpointData := map[string]interface{}{
			paramId:             endpoint.GetId(),
			paramApiVersion:     endpoint.GetApiVersion(),
			paramKind:           endpoint.GetKind(),
			paramCloud:          endpoint.GetCloud(),
			paramRegion:         endpoint.GetRegion(),
			paramService:        endpoint.GetService(),
			paramIsPrivate:      endpoint.GetIsPrivate(),
			paramConnectionType: endpoint.GetConnectionType(),
			paramEndpoint:       endpoint.GetEndpoint(),
		}

		// Set endpoint_type
		if endpoint.HasEndpointType() {
			endpointData[paramEndpointType] = endpoint.GetEndpointType()
		}

		// Set environment
		if endpoint.HasEnvironment() {
			env := endpoint.GetEnvironment()
			endpointData[paramEnvironment] = []map[string]interface{}{
				{
					paramId: env.GetId(),
				},
			}
		}

		// Set resource
		if endpoint.HasResource() {
			resource := endpoint.GetResource()
			resourceData := map[string]interface{}{
				paramId: resource.GetId(),
			}
			if resource.HasKind() {
				resourceData[paramKind] = resource.GetKind()
			}
			endpointData[paramEndpointResource] = []map[string]interface{}{resourceData}
		}

		// Set gateway
		if endpoint.HasGateway() {
			gateway := endpoint.GetGateway()
			endpointData[paramGateway] = []map[string]interface{}{
				{
					paramId: gateway.GetId(),
				},
			}
		}

		// Set access_point
		if endpoint.HasAccessPoint() {
			accessPoint := endpoint.GetAccessPoint()
			endpointData[paramAccessPoint] = []map[string]interface{}{
				{
					paramId: accessPoint.GetId(),
				},
			}
		}

		result[i] = endpointData
	}

	if err := d.Set(paramEndpoints, result); err != nil {
		return diag.FromErr(err)
	}

	endpointsJson, err := json.Marshal(endpoints)
	if err != nil {
		return diag.Errorf("error reading endpoints: error marshaling %#v to json: %s", endpoints, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched %d endpoints: %s", len(endpoints), endpointsJson))

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return nil
}

func loadEndpoints(ctx context.Context, c *Client, environmentId, service, resource, cloud, region string, isPrivate *bool) ([]end.EndpointV1Endpoint, error) {
	endpoints := make([]end.EndpointV1Endpoint, 0)

	allEndpointsAreCollected := false
	pageToken := ""
	for !allEndpointsAreCollected {
		endpointsPageList, resp, err := executeListEndpoints(ctx, c, environmentId, service, resource, cloud, region, isPrivate, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading endpoints: %s", createDescriptiveError(err, resp))
		}
		endpoints = append(endpoints, endpointsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := endpointsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allEndpointsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading endpoints: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allEndpointsAreCollected = true
		}
	}
	return endpoints, nil
}

func executeListEndpoints(ctx context.Context, c *Client, environmentId, service, resource, cloud, region string, isPrivate *bool, pageToken string) (end.EndpointV1EndpointList, *http.Response, error) {
	request := c.endClient.EndpointsEndpointV1Api.ListEndpointV1Endpoints(ctx).Environment(environmentId).Service(service).PageSize(listEndpointsPageSize)

	if resource != "" {
		request = request.Resource(resource)
	}
	if cloud != "" {
		request = request.Cloud(cloud)
	}
	if region != "" {
		request = request.Region(region)
	}
	if isPrivate != nil {
		request = request.IsPrivate(*isPrivate)
	}
	if pageToken != "" {
		request = request.PageToken(pageToken)
	}

	return request.Execute()
}
