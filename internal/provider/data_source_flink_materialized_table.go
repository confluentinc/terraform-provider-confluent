package provider

import (
	"context"
	"encoding/json"
	"fmt"
	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
)

func flinkMaterializedTableDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: materializedTableDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique name of the materialized table.",
			},
			paramKafkaCluster: {
				Type:        schema.TypeString,
				Description: "The Kafka Cluster Id hosting the Materialized Table's topic.",
				Computed:    true,
			},
			paramQuery: {
				Type:        schema.TypeString,
				Description: "he query section of the latest Materialized Table.",
				Computed:    true,
			},
			paramWatermarkColumnName: {
				Type:        schema.TypeString,
				Description: "The name of the watermark columns.",
				Computed:    true,
			},
			paramWatermarkExpression: {
				Type:        schema.TypeString,
				Description: "The watermark expression.",
				Computed:    true,
			},
			paramDistributedByColumnNames: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The names of the columns the table is distributed by.",
				Computed:    true,
			},
			paramDistributedByBuckets: {
				Type:        schema.TypeInt,
				Description: "The number of the buckets the table is distributed by.",
				Computed:    true,
			},

			paramStopped: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			paramColumns:     columnsSchemaDataSource(),
			paramConstraints: constraintsSchemaDataSource(),

			paramOrganization: optionalIdBlockSchema(),
			paramEnvironment:  optionalIdBlockSchema(),
			paramComputePool:  optionalIdBlockSchemaUpdatable(),
			paramPrincipal:    optionalIdBlockSchemaUpdatable(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Materialized Table, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
	}
}

func columnsSchemaDataSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramColumnComputed: columnComputedSchemaDataSource(),
				paramColumnPhysical: columnPhysicalSchemaDataSource(),
				paramColumnMetadata: columnMetadataSchemaDataSource(),
			},
		},
	}
}

func columnComputedSchemaDataSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramComputedName: {
					Type:        schema.TypeString,
					Description: "Name of the computed column.",
					Optional:    true,
					Computed:    true,
				},
				paramComputedType: {
					Type:        schema.TypeString,
					Description: "Type of the computed column.",
					Optional:    true,
					Computed:    true,
				},
				paramComputedComment: {
					Type:        schema.TypeString,
					Description: "Comment for the computed column.",
					Optional:    true,
					Computed:    true,
				},
				paramComputedKind: {
					Type:        schema.TypeString,
					Description: "Kind of the computed column.",
					Optional:    true,
					Computed:    true,
				},
				paramComputedExpression: {
					Type:        schema.TypeString,
					Description: "Expression of the computed column.",
					Optional:    true,
					Computed:    true,
				},
				paramComputedVirtual: {
					Type:        schema.TypeBool,
					Description: "Whether computed column is virtual.",
					Optional:    true,
					Computed:    true,
				},
			},
		},
	}
}

func columnPhysicalSchemaDataSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPhysicalName: {
					Type:        schema.TypeString,
					Description: "Name of the physical column.",
					Optional:    true,
					Computed:    true,
				},
				paramPhysicalType: {
					Type:        schema.TypeString,
					Description: "Type of the physical column.",
					Optional:    true,
					Computed:    true,
				},
				paramPhysicalComment: {
					Type:        schema.TypeString,
					Description: "Comment for the physical column.",
					Optional:    true,
					Computed:    true,
				},
				paramPhysicalKind: {
					Type:        schema.TypeString,
					Description: "Kind of the physical column.",
					Optional:    true,
					Computed:    true,
				},
			},
		},
	}
}

func columnMetadataSchemaDataSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramMetadataName: {
					Type:        schema.TypeString,
					Description: "Name of the metadata column.",
					Optional:    true,
					Computed:    true,
				},
				paramMetadataType: {
					Type:        schema.TypeString,
					Description: "Type of the metadata column.",
					Optional:    true,
					Computed:    true,
				},
				paramMetadataComment: {
					Type:        schema.TypeString,
					Description: "Comment for the metadata column.",
					Optional:    true,
					Computed:    true,
				},
				paramMetadataKind: {
					Type:        schema.TypeString,
					Description: "Kind of the metadata column.",
					Optional:    true,
					Computed:    true,
				},
				paramMetadataKey: {
					Type:        schema.TypeString,
					Description: "Metadata key of the metadata column.",
					Optional:    true,
					Computed:    true,
				},
				paramMetadataVirtual: {
					Type:        schema.TypeBool,
					Description: "Whether metadata column is virtual.",
					Optional:    true,
					Computed:    true,
				},
			},
		},
	}
}

func constraintsSchemaDataSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramConstraintsName: {
					Type:        schema.TypeString,
					Description: "Name of the constraint.",
					Optional:    true,
					Computed:    true,
				},
				paramConstraintsType: {
					Type:        schema.TypeString,
					Description: "Type of the constraint.",
					Optional:    true,
					Computed:    true,
				},
				paramConstraintsColumnNames: {
					Type:        schema.TypeSet,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "Constraint column names.",
					Optional:    true,
					Computed:    true,
				},
				paramConstraintsEnforced: {
					Type:        schema.TypeBool,
					Description: "Whether constraint is enforced.",
					Optional:    true,
					Computed:    true,
				},
			},
		},
	}
}

func materializedTableDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	displayName := d.Get(paramDisplayName).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Materialized Table %q", displayName), map[string]interface{}{flinkMaterializedTableLoggingKey: displayName})
	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClientInternal(restEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)
	if err != nil {
		return diag.Errorf("error reading Flink Materialized Table: %s", createDescriptiveError(err))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	flinkMaterializedTables, err := loadMaterializedTables(ctx, flinkRestClient)
	if err != nil {
		return diag.Errorf("error reading flink materialized table %q: %s", displayName, createDescriptiveError(err))
	}

	for _, flinkMaterializedTable := range flinkMaterializedTables {
		if flinkMaterializedTable.GetName() == displayName {
			fmtJson, err := json.Marshal(flinkMaterializedTable)
			if err != nil {
				return diag.Errorf("error reading flink materialized table %q: error marshaling %#v to json: %s", displayName, flinkMaterializedTable, createDescriptiveError(err))
			}
			if _, err := setMaterializedTableAttributes(d, flinkMaterializedTable, flinkRestClient); err != nil {
				tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Materialized Table using display name %q: %s", displayName, fmtJson))
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}
	return nil
}

func loadMaterializedTables(ctx context.Context, c *FlinkRestClient) ([]flinkgatewayv1.SqlV1MaterializedTable, error) {
	flinkMaterializedTables := make([]flinkgatewayv1.SqlV1MaterializedTable, 0)
	done := false
	pageToken := ""
	for !done {
		materializedTablesPageList, resp, err := executeListFlinkMaterializedTables(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading flink materialied table list: %s", createDescriptiveError(err, resp))
		}
		flinkMaterializedTables = append(flinkMaterializedTables, materializedTablesPageList.GetData()...)

		nextPageUrlString := materializedTablesPageList.GetMetadata().Next
		nextPageUrlStringNullable := flinkgatewayv1.NewNullableString(nextPageUrlString)

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString2 := *nextPageUrlStringNullable.Get()
			if nextPageUrlString2 == "" {
				done = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString2)
				if err != nil {
					return nil, fmt.Errorf("error reading Materialized Table: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			done = true
		}
	}
	return flinkMaterializedTables, nil
}
func executeListFlinkMaterializedTables(ctx context.Context, c *FlinkRestClient, pageToken string) (flinkgatewayv1.SqlV1MaterializedTableList, *http.Response, error) {
	if pageToken != "" {
		return c.apiClientInternal.MaterializedTablesSqlV1Api.ListSqlv1MaterializedTables(c.fgApiContext(ctx), c.organizationId, c.environmentId).PageSize(listFlinkArtifactsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.apiClientInternal.MaterializedTablesSqlV1Api.ListSqlv1MaterializedTables(c.fgApiContext(ctx), c.organizationId, c.environmentId).PageSize(listFlinkArtifactsPageSize).Execute()
	}
}
