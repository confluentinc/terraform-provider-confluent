package provider

import (
	"context"
	"encoding/json"
	"fmt"
	flinkgateway "github.com/confluentinc/ccloud-sdk-go-v2-internal/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramQuery                    = "query"
	paramWatermarkColumnName      = "watermark_column_name"
	paramWatermarkExpression      = "watermark_expression"
	paramDistributedByColumnNames = "distributed_by_column_names"
	paramDistributedByBuckets     = "distributed_by_buckets"
	paramColumns                  = "columns"
	paramConstraints              = "constraints"
	paramColumnComputed           = "columns_computed"
	paramColumnPhysical           = "columns_physical"
	paramColumnMetadata           = "columns_metadata"
	paramComputedName             = "column_computed_name"
	paramComputedKind             = "column_computed_kind"
	paramComputedComment          = "column_computed_comment"
	paramComputedType             = "column_computed_type"
	paramComputedExpression       = "column_computed_expression"
	paramComputedVirtual          = "column_computed_virtual"
	paramPhysicalName             = "column_physical_name"
	paramPhysicalKind             = "column_physical_kind"
	paramPhysicalComment          = "column_physical_comment"
	paramPhysicalType             = "column_physical_type"
	paramMetadataName             = "column_metadat_name"
	paramMetadataKind             = "column_metadat_kind"
	paramMetadataComment          = "column_metadat_comment"
	paramMetadataType             = "column_metadat_type"
	paramMetadataKey              = "column_metadata_expression"
	paramMetadataVirtual          = "column_metadat_virtual"
	paramConstraintsType          = "kind"
	paramConstraintsName          = "name"
	paramConstraintsColumnNames   = "column_names"
	paramConstraintsEnforced      = "enforced"
)

func flinkMaterializedTableResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: materializedTableCreate,
		ReadContext:   materializedTableRead,
		UpdateContext: materializedTableUpdate,
		DeleteContext: materializedTableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: materializedTableImport,
		},
		Schema: map[string]*schema.Schema{
			paramOrganization: optionalIdBlockSchema(),
			paramEnvironment:  optionalIdBlockSchema(),
			paramComputePool:  optionalIdBlockSchemaUpdatable(),
			paramPrincipal:    optionalIdBlockSchemaUpdatable(),
			paramDisplayName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique name of the materialized table.",
			},
			paramKafkaCluster: {
				Type:        schema.TypeString,
				Description: "The Kafka Cluster Id hosting the Materialized Table's topic.",
				Required:    true,
			},
			paramQuery: {
				Type:        schema.TypeString,
				Description: "he query section of the latest Materialized Table.",
				Optional:    true,
			},
			paramWatermarkColumnName: {
				Type:        schema.TypeString,
				Description: "The name of the watermark columns.",
				Optional:    true,
			},
			paramWatermarkExpression: {
				Type:        schema.TypeString,
				Description: "The watermark expression.",
				Optional:    true,
			},
			paramDistributedByColumnNames: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "The names of the columns the table is distributed by.",
				Optional:    true,
			},
			paramDistributedByBuckets: {
				Type:        schema.TypeInt,
				Description: "The number of the buckets the table is distributed by.",
				Optional:    true,
			},
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Compute Pool cluster, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramStopped: {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
			paramCredentials: credentialsSchema(),
			paramColumns:     columnsSchema(),
			paramConstraints: constraintsSchema(),
		},
	}
}

func columnsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramColumnComputed: columnComputedSchema(),
				paramColumnPhysical: columnPhysicalSchema(),
				paramColumnMetadata: columnMetadataSchema(),
			},
		},
	}
}

func columnComputedSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramComputedName: {
					Type:        schema.TypeString,
					Description: "Name of the computed column.",
					Optional:    true,
				},
				paramComputedType: {
					Type:        schema.TypeString,
					Description: "Type of the computed column.",
					Optional:    true,
				},
				paramComputedComment: {
					Type:        schema.TypeString,
					Description: "Comment for the computed column.",
					Optional:    true,
				},
				paramComputedKind: {
					Type:        schema.TypeString,
					Description: "Kind of the computed column.",
					Optional:    true,
				},
				paramComputedExpression: {
					Type:        schema.TypeString,
					Description: "Expression of the computed column.",
					Optional:    true,
				},
				paramComputedVirtual: {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Whether computed column is virtual.",
					Optional:    true,
				},
			},
		},
	}
}

func columnPhysicalSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPhysicalName: {
					Type:        schema.TypeString,
					Description: "Name of the physical column.",
					Optional:    true,
				},
				paramPhysicalType: {
					Type:        schema.TypeString,
					Description: "Type of the physical column.",
					Optional:    true,
				},
				paramPhysicalComment: {
					Type:        schema.TypeString,
					Description: "Comment for the physical column.",
					Optional:    true,
				},
				paramPhysicalKind: {
					Type:        schema.TypeString,
					Description: "Kind of the physical column.",
					Optional:    true,
				},
			},
		},
	}
}

func columnMetadataSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramMetadataName: {
					Type:        schema.TypeString,
					Description: "Name of the metadata column.",
					Optional:    true,
				},
				paramMetadataType: {
					Type:        schema.TypeString,
					Description: "Type of the metadata column.",
					Optional:    true,
				},
				paramMetadataComment: {
					Type:        schema.TypeString,
					Description: "Comment for the metadata column.",
					Optional:    true,
				},
				paramMetadataKind: {
					Type:        schema.TypeString,
					Description: "Kind of the metadata column.",
					Optional:    true,
				},
				paramMetadataKey: {
					Type:        schema.TypeString,
					Description: "Metadata key of the metadata column.",
					Optional:    true,
				},
				paramMetadataVirtual: {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Whether metadata column is virtual.",
					Optional:    true,
				},
			},
		},
	}
}

func constraintsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramConstraintsName: {
					Type:        schema.TypeString,
					Description: "Name of the constraint.",
					Optional:    true,
				},
				paramConstraintsType: {
					Type:        schema.TypeString,
					Description: "Type of the constraint.",
					Optional:    true,
				},
				paramConstraintsColumnNames: {
					Type:        schema.TypeSet,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "Constraint column names.",
					Optional:    true,
				},
				paramConstraintsEnforced: {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Whether constraint is enforced.",
					Optional:    true,
				},
			},
		},
	}
}

func materializedTableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClientInternal(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)

	displayName := d.Get(paramDisplayName).(string)
	kafkaId := d.Get(paramKafkaCluster).(string)
	query := d.Get(paramQuery).(string)

	stopped := d.Get(paramStopped).(bool)

	var isComputed bool
	computedGet := d.Get(paramColumnComputed)
	if computedGet != nil {
		isComputed = len(computedGet.([]interface{})) > 0
	}

	var isPhysical bool
	physicalGet := d.Get(paramColumnPhysical)
	if physicalGet != nil {
		isPhysical = len(physicalGet.([]interface{})) > 0
	}

	var isMetadata bool
	metadataGet := d.Get(paramColumnMetadata)
	if metadataGet != nil {
		isMetadata = len(metadataGet.([]interface{})) > 0
	}

	table := flinkgateway.SqlV1MaterializedTable{
		Name:           displayName,
		EnvironmentId:  environmentId,
		OrganizationId: organizationId,

		Spec: flinkgateway.SqlV1MaterializedTableSpec{
			KafkaClusterId: &kafkaId,
			ComputePoolId:  &computePoolId,
			Principal:      &principalId,
			Query:          &query,
			Stopped:        &stopped,
		},
	}
	table.Spec.Watermark = &flinkgateway.SqlV1MaterializedTableWatermark{}
	table.Spec.DistributedBy = &flinkgateway.SqlV1MaterializedTableDistribution{}

	var computedColumn []flinkgateway.SqlV1MaterializedTableColumnDetails
	var physicalColumn []flinkgateway.SqlV1MaterializedTableColumnDetails
	var metadataColumn []flinkgateway.SqlV1MaterializedTableColumnDetails

	if isComputed {
		computedColumn = expandComputedColumns(d)
	}
	if isPhysical {
		physicalColumn = expandPhysicalColumns(d)
	}
	if isMetadata {
		metadataColumn = expandMetadataColumns(d)
	}

	var columns []flinkgateway.SqlV1MaterializedTableColumnDetails
	for _, col := range computedColumn {
		columns = append(columns, col)
	}
	for _, col := range physicalColumn {
		columns = append(columns, col)
	}
	for _, col := range metadataColumn {
		columns = append(columns, col)
	}

	if isPhysical || isComputed || isMetadata {
		table.Spec.SetColumns(columns)
	}

	watermarkColumnName := d.Get(paramWatermarkColumnName).(string)
	if watermarkColumnName != "" {
		table.Spec.Watermark.SetColumnName(watermarkColumnName)
	}

	watermarkColumnExpression := d.Get(paramWatermarkExpression).(string)
	if watermarkColumnExpression != "" {
		table.Spec.Watermark.SetExpression(watermarkColumnExpression)
	}

	distributedByBuckets := d.Get(paramDistributedByBuckets).(int)
	if distributedByBuckets != 0 {
		table.Spec.DistributedBy.SetBuckets(int32(distributedByBuckets))
	}

	distributedByColumns := getStringSet(d, paramDistributedByColumnNames)
	if len(distributedByColumns) > 0 {
		table.Spec.DistributedBy.SetColumnNames(distributedByColumns)
	}

	constraints := expandMaterializedTableConstraints(d, paramConstraints)
	if len(constraints) > 0 {
		table.Spec.SetConstraints(constraints)
	}

	createMaterializedTableRequestJson, err := json.Marshal(table)
	if err != nil {
		return diag.Errorf("error creating Flink Materialized Table: error marshaling %#v to json: %s", createMaterializedTableRequestJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Flink Materialized Table: %s", createMaterializedTableRequestJson))

	createdMaterializedTable, resp, err := executeMaterializedTableCreate(flinkRestClient.fgApiContext(ctx), flinkRestClient, table, organizationId, environmentId, kafkaId)
	if err != nil {
		return diag.Errorf("error creating Flink Materialized Table %q: %s", createdMaterializedTable.GetName(), createDescriptiveError(err, resp))
	}
	d.SetId(createFlinkMaterializedTableId(createdMaterializedTable.GetEnvironmentId(), createdMaterializedTable.Spec.GetKafkaClusterId(), createdMaterializedTable.GetName()))

	createdMaterializedTableJson, err := json.Marshal(createdMaterializedTable)
	if err != nil {
		return diag.Errorf("error creating Flink Materialized Table %q: error marshaling %#v to json: %s", d.Id(), createdMaterializedTable, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Materialized Table %q: %s", d.Id(), createdMaterializedTableJson), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	return materializedTableRead(ctx, d, meta)
}

func executeMaterializedTableCreate(ctx context.Context, c *FlinkRestClient, table flinkgateway.SqlV1MaterializedTable, orgId, environmentId, kafkaId string) (flinkgateway.SqlV1MaterializedTable, *http.Response, error) {
	req := c.apiClientInternal.MaterializedTablesSqlV1Api.CreateSqlv1MaterializedTable(c.fgApiContext(ctx), orgId, environmentId, kafkaId).SqlV1MaterializedTable(table)
	return req.Execute()
}

func executeMaterializedTableRead(ctx context.Context, c *FlinkRestClient, orgId, environmentId, kafkaId, tableName string) (flinkgateway.SqlV1MaterializedTable, *http.Response, error) {
	req := c.apiClientInternal.MaterializedTablesSqlV1Api.GetSqlv1MaterializedTable(c.fgApiContext(ctx), orgId, environmentId, kafkaId, tableName)
	return req.Execute()
}

func materializedTableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Materialized Table %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	materializedTableId := d.Id()
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

	kafkaId := d.Get(paramKafkaCluster).(string)

	if _, err := readMaterializedTableAndSetAttributes(ctx, d, organizationId, environmentId, kafkaId, materializedTableId, flinkRestClient); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Flink Materialized Table %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readMaterializedTableAndSetAttributes(ctx context.Context, d *schema.ResourceData, orgId, environmentId, kafkaId, materializedTableId string, c *FlinkRestClient) ([]*schema.ResourceData, error) {
	materializedTable, resp, err := executeMaterializedTableRead(c.fgApiContext(ctx), c, orgId, environmentId, kafkaId, getTableName(materializedTableId))
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Materialized Table %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Materialized Table %q in TF state because Flink Materialized Table could not be found on the server", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	materializedTableJson, err := json.Marshal(materializedTable)
	if err != nil {
		return nil, fmt.Errorf("error reading Flink Materialized Table %q: error marshaling %#v to json: %s", materializedTableId, materializedTable, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Materialed Table %q: %s", d.Id(), materializedTableJson), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	if _, err := setMaterializedTableAttributes(d, materializedTable, c); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Materialized Table %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setMaterializedTableAttributes(d *schema.ResourceData, materializedTable flinkgateway.SqlV1MaterializedTable, c *FlinkRestClient) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, materializedTable.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramKafkaCluster, materializedTable.Spec.GetKafkaClusterId()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramPrincipal, paramId, materializedTable.Spec.GetPrincipal(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramComputePool, paramId, materializedTable.Spec.GetComputePoolId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramQuery, materializedTable.Spec.GetQuery()); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.flinkApiKey, c.flinkApiSecret, d, c.externalAccessToken != nil); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramOrganization, paramId, materializedTable.GetOrganizationId(), d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, materializedTable.GetEnvironmentId(), d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramComputePool, paramId, materializedTable.Spec.GetComputePoolId(), d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramPrincipal, paramId, materializedTable.Spec.GetPrincipal(), d); err != nil {
			return nil, err
		}
	}

	if materializedTable.Spec.Watermark != nil {
		_ = d.Set(paramWatermarkColumnName, materializedTable.Spec.Watermark.GetColumnName())
		_ = d.Set(paramWatermarkExpression, materializedTable.Spec.Watermark.GetExpression())
	} else {
		_ = d.Set(paramWatermarkColumnName, nil)
		_ = d.Set(paramWatermarkExpression, nil)
	}

	if materializedTable.Spec.DistributedBy != nil {
		_ = d.Set(paramDistributedByColumnNames, materializedTable.Spec.DistributedBy.GetColumnNames())
		_ = d.Set(paramDistributedByBuckets, materializedTable.Spec.DistributedBy.GetBuckets())
	} else {
		_ = d.Set(paramDistributedByColumnNames, nil)
		_ = d.Set(paramDistributedByBuckets, nil)
	}

	err := d.Set(paramStopped, materializedTable.Spec.GetStopped())
	if err != nil {
		return nil, err
	}
	if materializedTable.Spec.GetColumns() != nil {
		columnsList := make([]map[string]interface{}, 0, len(materializedTable.Spec.GetColumns()))
		for _, col := range materializedTable.Spec.GetColumns() {
			m := map[string]interface{}{}

			if col.SqlV1ComputedColumn != nil {
				computedCol := col.SqlV1ComputedColumn
				computedVirtual := false
				if computedCol.Virtual != nil {
					computedVirtual = *computedCol.Virtual
				}

				m[paramColumnComputed] = []map[string]interface{}{
					{
						paramComputedName:       computedCol.Name,
						paramComputedType:       computedCol.Type,
						paramComputedComment:    computedCol.Comment,
						paramComputedKind:       computedCol.Kind,
						paramComputedExpression: computedCol.Expression,
						paramComputedVirtual:    computedVirtual,
					},
				}
			} else {
				m[paramColumnComputed] = []map[string]interface{}{}
			}

			if col.SqlV1PhysicalColumn != nil {
				physicalCol := col.SqlV1PhysicalColumn
				m[paramColumnPhysical] = []map[string]interface{}{
					{
						paramPhysicalName:    physicalCol.Name,
						paramPhysicalType:    physicalCol.Type,
						paramPhysicalComment: physicalCol.Comment,
						paramPhysicalKind:    physicalCol.Kind,
					},
				}
			} else {
				m[paramColumnPhysical] = []map[string]interface{}{}
			}

			if col.SqlV1MetadataColumn != nil {
				metadataCol := col.SqlV1MetadataColumn
				metadataVirtual := false
				if metadataCol.Virtual != nil {
					metadataVirtual = *metadataCol.Virtual
				}

				m[paramColumnMetadata] = []map[string]interface{}{
					{
						paramMetadataName:    metadataCol.Name,
						paramMetadataType:    metadataCol.Type,
						paramMetadataComment: metadataCol.Comment,
						paramMetadataKind:    metadataCol.Kind,
						paramMetadataKey:     metadataCol.MetadataKey,
						paramMetadataVirtual: metadataVirtual,
					},
				}
			} else {
				m[paramColumnMetadata] = []map[string]interface{}{}
			}

			columnsList = append(columnsList, m)
		}

		_ = d.Set(paramColumns, columnsList)
	} else {
		_ = d.Set(paramColumns, nil)
	}

	if materializedTable.Spec.GetConstraints() != nil {
		constraintsList := make([]map[string]interface{}, 0, len(materializedTable.Spec.GetConstraints()))

		for _, c := range materializedTable.Spec.GetConstraints() {
			m := map[string]interface{}{
				paramConstraintsName:        c.Name,
				paramConstraintsType:        c.Kind,
				paramConstraintsColumnNames: schema.NewSet(schema.HashString, toInterfaceSlice(*c.ColumnNames)),
				paramConstraintsEnforced:    c.Enforced,
			}
			constraintsList = append(constraintsList, m)
		}
		_ = d.Set(paramConstraints, constraintsList)
	} else {
		_ = d.Set(paramConstraints, nil)
	}

	d.SetId(createFlinkMaterializedTableId(materializedTable.GetEnvironmentId(), materializedTable.Spec.GetKafkaClusterId(), materializedTable.GetName()))
	return d, nil
}

func materializedTableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Materilaized Table %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Materialized Table: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Materialized Table: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Materialized Table: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Materialized Table: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Materialized Table: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClientInternal(restEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)

	kafkaId := d.Get(paramKafkaCluster).(string)

	req := flinkRestClient.apiClientInternal.MaterializedTablesSqlV1Api.DeleteSqlv1MaterializedTable(flinkRestClient.fgApiContext(ctx), organizationId, environmentId, kafkaId, getTableName(d.Id()))
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Flink Materialized Table %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Materialized Tablel %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	return nil
}

func materializedTableImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Materialized Table %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClientInternal(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)

	tableName := getTableName(d.Id())
	kafkaId := getKafkaId(d.Id())
	d.SetId(createFlinkMaterializedTableId(environmentId, kafkaId, tableName))

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readMaterializedTableAndSetAttributes(ctx, d, organizationId, environmentId, kafkaId, tableName, flinkRestClient); err != nil {
		return nil, fmt.Errorf("error importing Flink Materialized Table %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Materialized Table %q", d.Id()), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func materializedTableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramQuery, paramStopped, paramComputePool, paramPrincipal, paramColumns, paramWatermarkExpression, paramWatermarkColumnName, paramConstraints) {
		return diag.Errorf("error updating Flink connection %q: only %q, %q, %q, %q, %q, %q, %q, and %q attributes can be updated for Flink Compute Pool", d.Id(), paramQuery, paramStopped, paramComputePool, paramPrincipal, paramColumns, paramWatermarkExpression, paramWatermarkColumnName, paramConstraints)
	}

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClientInternal(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)

	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	name := d.Get(paramDisplayName).(string)
	kafkaId := d.Get(paramKafkaCluster).(string)

	table, _, err := executeMaterializedTableRead(flinkRestClient.fgApiContext(ctx), flinkRestClient, organizationId, environmentId, kafkaId, name)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	if d.HasChange(paramQuery) {
		table.Spec.SetQuery(d.Get(paramQuery).(string))
	}

	if d.HasChange(paramStopped) {
		table.Spec.SetStopped(d.Get(paramStopped).(bool))
	}

	if d.HasChange(paramWatermarkExpression) {
		table.Spec.Watermark.SetExpression(d.Get(paramWatermarkExpression).(string))
	}

	if d.HasChange(paramWatermarkColumnName) {
		table.Spec.Watermark.SetColumnName(d.Get(paramWatermarkColumnName).(string))
	}

	if d.HasChange(paramComputePool) {
		table.Spec.SetComputePoolId(computePoolId)
	}

	if d.HasChange(paramPrincipal) {
		table.Spec.SetPrincipal(principalId)
	}

	if d.HasChange(paramColumns) {
		columnsList := make([]map[string]interface{}, 0, len(expandComputedColumns(d))+len(expandPhysicalColumns(d))+len(expandMetadataColumns(d)))
		var isComputed bool
		computedGet := d.Get(paramColumnComputed)
		if computedGet != nil {
			isComputed = len(computedGet.([]interface{})) > 0
		}

		var isPhysical bool
		physicalGet := d.Get(paramColumnPhysical)
		if physicalGet != nil {
			isPhysical = len(physicalGet.([]interface{})) > 0
		}

		var isMetadata bool
		metadataGet := d.Get(paramColumnMetadata)
		if metadataGet != nil {
			isMetadata = len(metadataGet.([]interface{})) > 0
		}
		m := map[string]interface{}{}
		if isComputed {
			for _, col := range expandComputedColumns(d) {
				if col.SqlV1ComputedColumn != nil {
					computedCol := col.SqlV1ComputedColumn
					computedVirtual := false
					if computedCol.Virtual != nil {
						computedVirtual = *computedCol.Virtual
					}

					m[paramColumnComputed] = []map[string]interface{}{
						{
							paramComputedName:       computedCol.Name,
							paramComputedType:       computedCol.Type,
							paramComputedComment:    computedCol.Comment,
							paramComputedKind:       computedCol.Kind,
							paramComputedExpression: computedCol.Expression,
							paramComputedVirtual:    computedVirtual,
						},
					}
					columnsList = append(columnsList, m)
				} else {
					m[paramColumnComputed] = []map[string]interface{}{}
				}
			}
		}
		if isPhysical {
			for _, col := range expandComputedColumns(d) {
				if col.SqlV1PhysicalColumn != nil {
					physicalCol := col.SqlV1PhysicalColumn
					m[paramColumnPhysical] = []map[string]interface{}{
						{
							paramPhysicalName:    physicalCol.Name,
							paramPhysicalType:    physicalCol.Type,
							paramPhysicalComment: physicalCol.Comment,
							paramPhysicalKind:    physicalCol.Kind,
						},
					}
					columnsList = append(columnsList, m)

				} else {
					m[paramColumnPhysical] = []map[string]interface{}{}
				}
			}

		}
		if isMetadata {
			for _, col := range expandComputedColumns(d) {
				if col.SqlV1MetadataColumn != nil {
					metadataCol := col.SqlV1MetadataColumn
					metadataVirtual := false
					if metadataCol.Virtual != nil {
						metadataVirtual = *metadataCol.Virtual
					}

					m[paramColumnMetadata] = []map[string]interface{}{
						{
							paramMetadataName:    metadataCol.Name,
							paramMetadataType:    metadataCol.Type,
							paramMetadataComment: metadataCol.Comment,
							paramMetadataKind:    metadataCol.Kind,
							paramMetadataKey:     metadataCol.MetadataKey,
							paramMetadataVirtual: metadataVirtual,
						},
					}
					columnsList = append(columnsList, m)

				} else {
					m[paramColumnMetadata] = []map[string]interface{}{}
				}
			}

		}
		_ = d.Set(paramColumns, columnsList)
	}
	if d.HasChange(paramConstraints) {
		constraints := expandMaterializedTableConstraints(d, paramConstraints)
		if len(constraints) > 0 {
			constraintsList := make([]map[string]interface{}, 0, len(constraints))

			for _, c := range constraints {
				m := map[string]interface{}{
					paramConstraintsName:        c.Name,
					paramConstraintsType:        c.Kind,
					paramConstraintsColumnNames: schema.NewSet(schema.HashString, toInterfaceSlice(*c.ColumnNames)),
					paramConstraintsEnforced:    c.Enforced,
				}
				constraintsList = append(constraintsList, m)
			}

			_ = d.Set(paramConstraints, constraintsList)
			table.Spec.SetConstraints(constraints)
		}
	}

	updateMaterializedTableRequestJson, err := json.Marshal(table)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table %q: error marshaling %#v to json: %s", d.Id(), table, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Materialized Table %q: %s", d.Id(), updateMaterializedTableRequestJson), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})

	req := flinkRestClient.apiClientInternal.MaterializedTablesSqlV1Api.UpdateSqlv1MaterializedTable(flinkRestClient.fgApiContext(ctx), organizationId, environmentId, kafkaId, name).SqlV1MaterializedTable(table)
	updatedTable, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedTableJson, err := json.Marshal(updatedTable)
	if err != nil {
		return diag.Errorf("error updating Flink Materialized Table %q: error marshaling %#v to json: %s", d.Id(), updatedTable, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Flink Materialized Table %q: %s", d.Id(), updatedTableJson), map[string]interface{}{flinkMaterializedTableLoggingKey: d.Id()})
	return materializedTableRead(ctx, d, meta)
}

func createFlinkMaterializedTableId(environmentId, kafkaId, tableName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentId, kafkaId, tableName)
}

func getTableName(tableId string) string {
	parts := strings.Split(tableId, "/")
	tableName := parts[len(parts)-1]
	return tableName
}

func getKafkaId(tableId string) string {
	parts := strings.Split(tableId, "/")
	tableName := parts[len(parts)-2]
	return tableName
}

func getStringSet(d *schema.ResourceData, key string) []string {
	raw, ok := d.GetOk(key)
	if !ok || raw == nil {
		return nil
	}

	set := raw.(*schema.Set)
	result := make([]string, 0, set.Len())
	for _, v := range set.List() {
		if s, ok := v.(string); ok && s != "" {
			result = append(result, s)
		}
	}
	return result
}

func expandMaterializedTableConstraints(d *schema.ResourceData, key string) []flinkgateway.SqlV1MaterializedTableConstraint {
	raw, ok := d.GetOk(key)
	if !ok || raw == nil {
		return nil
	}

	list := raw.([]interface{})
	if len(list) == 0 {
		return nil
	}

	result := make([]flinkgateway.SqlV1MaterializedTableConstraint, 0, len(list))

	for _, v := range list {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		var c flinkgateway.SqlV1MaterializedTableConstraint
		if name, ok := m["name"].(string); ok && name != "" {
			c.Name = &name
		}
		if kind, ok := m["kind"].(string); ok && kind != "" {
			c.Kind = &kind
		}
		if enforced, ok := m["enforced"].(bool); ok {
			c.Enforced = &enforced
		}
		if rawSet, ok := m["column_names"].(*schema.Set); ok && rawSet.Len() > 0 {
			cols := make([]string, 0, rawSet.Len())
			for _, col := range rawSet.List() {
				if s, ok := col.(string); ok && s != "" {
					cols = append(cols, s)
				}
			}
			if len(cols) > 0 {
				c.ColumnNames = &cols
			}
		}
		result = append(result, c)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func expandComputedColumns(d *schema.ResourceData) []flinkgateway.SqlV1MaterializedTableColumnDetails {
	raw, ok := d.GetOk(paramColumnComputed)
	if !ok || raw == nil {
		return nil
	}

	list := raw.([]interface{})
	result := make([]flinkgateway.SqlV1MaterializedTableColumnDetails, 0, len(list))

	for _, v := range list {
		m := v.(map[string]interface{})

		col := flinkgateway.SqlV1MaterializedTableColumnDetails{}

		if name, ok := m[paramComputedName].(string); ok && name != "" {
			col.SqlV1ComputedColumn.Name = &name
		}
		if kind, ok := m[paramComputedKind].(string); ok && kind != "" {
			col.SqlV1ComputedColumn.Kind = kind
		}
		if expr, ok := m[paramComputedExpression].(string); ok && expr != "" {
			col.SqlV1ComputedColumn.Expression = expr
		}
		if virtual, ok := m[paramComputedVirtual].(bool); ok {
			col.SqlV1ComputedColumn.Virtual = &virtual
		}

		result = append(result, col)
	}

	return result
}

func expandPhysicalColumns(d *schema.ResourceData) []flinkgateway.SqlV1MaterializedTableColumnDetails {
	raw, ok := d.GetOk("physical_columns")
	if !ok || raw == nil {
		return nil
	}

	list := raw.([]interface{})
	result := make([]flinkgateway.SqlV1MaterializedTableColumnDetails, 0, len(list))

	for _, v := range list {
		m := v.(map[string]interface{})

		col := flinkgateway.SqlV1MaterializedTableColumnDetails{}

		if name, ok := m["name"].(string); ok && name != "" {
			col.SqlV1PhysicalColumn.Name = name
		}
		if typ, ok := m["type"].(string); ok && typ != "" {
			col.SqlV1PhysicalColumn.Type = typ
		}
		if comment, ok := m["comment"].(string); ok && comment != "" {
			col.SqlV1PhysicalColumn.Comment = &comment
		}
		if kind, ok := m["kind"].(string); ok && kind != "" {
			col.SqlV1PhysicalColumn.Kind = kind
		}

		result = append(result, col)
	}

	return result
}

func expandMetadataColumns(d *schema.ResourceData) []flinkgateway.SqlV1MaterializedTableColumnDetails {
	raw, ok := d.GetOk("metadata_columns")
	if !ok || raw == nil {
		return nil
	}

	list := raw.([]interface{})
	result := make([]flinkgateway.SqlV1MaterializedTableColumnDetails, 0, len(list))

	for _, v := range list {
		m := v.(map[string]interface{})

		col := flinkgateway.SqlV1MaterializedTableColumnDetails{}

		if name, ok := m["name"].(string); ok && name != "" {
			col.SqlV1MetadataColumn.Name = name
		}
		if typ, ok := m["type"].(string); ok && typ != "" {
			col.SqlV1MetadataColumn.Type = typ
		}
		if comment, ok := m["comment"].(string); ok && comment != "" {
			col.SqlV1MetadataColumn.Comment = &comment
		}
		if kind, ok := m["kind"].(string); ok && kind != "" {
			col.SqlV1MetadataColumn.Kind = kind
		}
		if key, ok := m["metadata_key"].(string); ok && key != "" {
			col.SqlV1MetadataColumn.MetadataKey = key
		}
		if virtual, ok := m["virtual"].(bool); ok {
			col.SqlV1MetadataColumn.Virtual = &virtual
		}

		result = append(result, col)
	}

	return result
}

func toInterfaceSlice(strs []string) []interface{} {
	out := make([]interface{}, len(strs))
	for i, s := range strs {
		out[i] = s
	}
	return out
}
