package provider

import (
	"context"
	"encoding/json"
	"fmt"
	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
)

const (
	paramData         = "data"
	paramStatusDetail = "status_detail"
)

func flinkConnectionDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: connectionDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique name of the Flink Connection per organization, environment scope.",
			},
			paramType: {
				Type:        schema.TypeString,
				Description: "The type of the Flink Connection.",
				Optional:    true,
			},
			paramEndpoint: {
				Type:        schema.TypeString,
				Description: "The endpoint of the Flink Connection.",
				Computed:    true,
			},
			paramData: {
				Type:        schema.TypeString,
				Description: "The auth data of the Flink Connection.",
				Computed:    true,
			},
			paramStatus: {
				Type:        schema.TypeString,
				Description: "The status of the Flink Connection.",
				Computed:    true,
			},
			paramStatusDetail: {
				Type:        schema.TypeString,
				Description: "The status details of the Flink Connection.",
				Computed:    true,
			},
			paramOrganization: optionalIdBlockSchema(),
			paramEnvironment:  optionalIdBlockSchema(),
			paramComputePool:  optionalIdBlockSchemaUpdatable(),
			paramPrincipal:    optionalIdBlockSchemaUpdatable(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Connection cluster, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
	}
}

func connectionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	connectionName := d.Get(paramDisplayName).(string)
	connectionType := d.Get(paramType).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Connection %q", connectionName), map[string]interface{}{flinkConnectionLoggingKey: connectionName})
	flinkRestClient, errClient := getFlinkClient(d, meta)
	if errClient != nil {
		return errClient
	}
	flinkConnections, err := loadFlinkConnections(ctx, flinkRestClient, connectionType)
	if err != nil {
		return diag.Errorf("error reading flink connection %q: %s", connectionName, createDescriptiveError(err))
	}

	for _, flinkConnection := range flinkConnections {
		if flinkConnection.GetName() == connectionName {
			fcJson, err := json.Marshal(flinkConnection)
			if err != nil {
				return diag.Errorf("error reading flink connection %q: error marshaling %#v to json: %s", connectionName, flinkConnection, createDescriptiveError(err))
			}
			if orgHasMultipleFlinkConnectionsWithTargetDisplayName(flinkConnections, connectionName) {
				return diag.Errorf("error reading flink connections: there are multiple flink connections with %q=%q", paramDisplayName, connectionName)
			}
			if _, err := setConnectionDataSourceAttributes(d, flinkConnection, flinkRestClient); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Connection using display name %q: %s", connectionName, fcJson))
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}
	return nil
}

func loadFlinkConnections(ctx context.Context, c *FlinkRestClient, connectionType string) ([]flinkgatewayv1.SqlV1Connection, error) {
	flinkConnections := make([]flinkgatewayv1.SqlV1Connection, 0)
	done := false
	pageToken := ""
	for !done {
		connectionPageList, resp, err := executeListFlinkConnections(ctx, c, connectionType, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading flink connections list: %s", createDescriptiveError(err, resp))
		}
		flinkConnections = append(flinkConnections, connectionPageList.GetData()...)

		nextPageUrlString := connectionPageList.GetMetadata().Next
		nextPageUrlStringNullable := flinkgatewayv1.NewNullableString(nextPageUrlString)

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString2 := *nextPageUrlStringNullable.Get()
			if nextPageUrlString2 == "" {
				done = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString2)
				if err != nil {
					return nil, fmt.Errorf("error reading Connection: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			done = true
		}
	}
	return flinkConnections, nil
}
func executeListFlinkConnections(ctx context.Context, c *FlinkRestClient, connectionType, pageToken string) (flinkgatewayv1.SqlV1ConnectionList, *http.Response, error) {
	if connectionType != "" {
		if pageToken != "" {
			return c.apiClient.ConnectionsSqlV1Api.ListSqlv1Connections(c.apiContext(ctx), c.organizationId, c.environmentId).SpecConnectionType(connectionType).PageSize(listFlinkArtifactsPageSize).PageToken(pageToken).Execute()
		} else {
			return c.apiClient.ConnectionsSqlV1Api.ListSqlv1Connections(c.apiContext(ctx), c.organizationId, c.environmentId).SpecConnectionType(connectionType).PageSize(listFlinkArtifactsPageSize).Execute()
		}
	} else {
		if pageToken != "" {
			return c.apiClient.ConnectionsSqlV1Api.ListSqlv1Connections(c.apiContext(ctx), c.organizationId, c.environmentId).PageSize(listFlinkArtifactsPageSize).PageToken(pageToken).Execute()
		} else {
			return c.apiClient.ConnectionsSqlV1Api.ListSqlv1Connections(c.apiContext(ctx), c.organizationId, c.environmentId).PageSize(listFlinkArtifactsPageSize).Execute()
		}
	}
}

func orgHasMultipleFlinkConnectionsWithTargetDisplayName(flinkConnections []flinkgatewayv1.SqlV1Connection, displayName string) bool {
	var counter = 0
	for _, flinkConnection := range flinkConnections {
		if flinkConnection.GetName() == displayName {
			counter += 1
		}
	}
	return counter > 1
}

func setConnectionDataSourceAttributes(d *schema.ResourceData, connection flinkgatewayv1.SqlV1Connection, c *FlinkRestClient) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, connection.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramType, connection.Spec.GetConnectionType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEndpoint, connection.Spec.GetEndpoint()); err != nil {
		return nil, err
	}
	if err := d.Set(paramData, connection.Spec.GetAuthData().SqlV1PlaintextProvider.GetData()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatus, connection.Status.GetPhase()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatusDetail, connection.Status.GetDetail()); err != nil {
		return nil, err
	}
	d.SetId(connection.GetName())
	return d, nil
}
