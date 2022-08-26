package provider

import (
	"context"
	"encoding/json"
	"fmt"
	ksql "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing ksqlDB cluster using ksqldbcm V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listKsqldbcmV2Clusters
	listKsqlClustersPageSize = 99
)

func ksqlDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: ksqlDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramUseDetailedProcessingLog: {
				Type:     schema.TypeBool,
				Computed: true,
			},
			paramCsu: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramHttpEndpoint: {
				Type:        schema.TypeString,
				Description: "The dataplane endpoint of the ksqlDB cluster.",
				Computed:    true,
			},
			paramStorage: {
				Type:        schema.TypeInt,
				Description: "Amount of storage (in GB) provisioned to this cluster.",
				Computed:    true,
			},
			paramTopicPrefix: {
				Type:        schema.TypeString,
				Description: "Topic name prefix used by this ksqlDB cluster. Used to assign ACLs for this ksqlDB cluster to use.",
				Computed:    true,
			},
			paramKafkaCluster:       optionalDataSourceSchema(),
			paramCredentialIdentity: optionalDataSourceSchema(),
		},
	}
}

func ksqlDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	clusterId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if clusterId != "" {
		return ksqlDataSourceReadUsingId(ctx, d, meta, environmentId, clusterId)
	} else if displayName != "" {
		return ksqlDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading ksqlDB cluster: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func ksqlDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading ksqlDB Cluster %q=%q", paramId, clusterId), map[string]interface{}{ksqlClusterLoggingKey: clusterId})

	c := meta.(*Client)
	ksqlCluster, _, err := executeKsqlRead(c.ksqlApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return diag.Errorf("error reading ksqlDB cluster %q: %s", clusterId, createDescriptiveError(err))
	}
	ksqlClusterJson, err := json.Marshal(ksqlCluster)
	if err != nil {
		return diag.Errorf("error reading ksqlDB Cluster %q: error marshaling %#v to json: %s", clusterId, ksqlCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched ksqlDb Cluster %q: %s", clusterId, ksqlClusterJson), map[string]interface{}{ksqlClusterLoggingKey: clusterId})

	if _, err := setKsqlAttributes(d, ksqlCluster); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func ksqlDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading ksqlDB Cluster %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	ksqlClusters, err := loadKsqlClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading ksqlDB Cluster %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleKsqlClustersWithTargetDisplayName(ksqlClusters, displayName) {
		return diag.Errorf("error reading ksqlDB Cluster: there are multiple ksqlDb Clusters with %q=%q", paramDisplayName, displayName)
	}

	for _, cluster := range ksqlClusters {
		if cluster.Spec.GetDisplayName() == displayName {
			if _, err := setKsqlAttributes(d, cluster); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading ksqlDB Cluster: ksqlDB Cluster with %q=%q was not found", paramDisplayName, displayName)
}

func orgHasMultipleKsqlClustersWithTargetDisplayName(clusters []ksql.KsqldbcmV2Cluster, displayName string) bool {
	var numberOfClustersWithTargetDisplayName = 0
	for _, cluster := range clusters {
		if cluster.Spec.GetDisplayName() == displayName {
			numberOfClustersWithTargetDisplayName += 1
		}
	}
	return numberOfClustersWithTargetDisplayName > 1
}

func loadKsqlClusters(ctx context.Context, c *Client, environmentId string) ([]ksql.KsqldbcmV2Cluster, error) {
	clusters := make([]ksql.KsqldbcmV2Cluster, 0)

	allClustersAreCollected := false
	pageToken := ""
	for !allClustersAreCollected {
		clustersPageList, _, err := executeListKsqlClusters(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading ksqlDB Clusters: %s", createDescriptiveError(err))
		}
		clusters = append(clusters, clustersPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := clustersPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allClustersAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading ksqlDB Clusters: %s", createDescriptiveError(err))
				}
			}
		} else {
			allClustersAreCollected = true
		}
	}
	return clusters, nil
}

func executeListKsqlClusters(ctx context.Context, c *Client, environmentId, pageToken string) (ksql.KsqldbcmV2ClusterList, *http.Response, error) {
	if pageToken != "" {
		return c.ksqlClient.ClustersKsqldbcmV2Api.ListKsqldbcmV2Clusters(c.ksqlApiContext(ctx)).Environment(environmentId).PageSize(listKsqlClustersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.ksqlClient.ClustersKsqldbcmV2Api.ListKsqldbcmV2Clusters(c.ksqlApiContext(ctx)).Environment(environmentId).PageSize(listKsqlClustersPageSize).Execute()
	}
}

func optionalDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}
