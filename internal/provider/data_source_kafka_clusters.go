package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func kafkaClustersDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: kafkaClustersDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramEnvironment: environmentDataSourceSchema(),
			"clusters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                 {Type: schema.TypeString, Computed: true},
						"display_name":       {Type: schema.TypeString, Computed: true},
						"cloud":              {Type: schema.TypeString, Computed: true},
						"region":             {Type: schema.TypeString, Computed: true},
						"availability":       {Type: schema.TypeString, Computed: true},
						"kind":               {Type: schema.TypeString, Computed: true},
						"api_version":        {Type: schema.TypeString, Computed: true},
						"bootstrap_endpoint": {Type: schema.TypeString, Computed: true},
						"rest_endpoint":      {Type: schema.TypeString, Computed: true},
						"rbac_crn":           {Type: schema.TypeString, Computed: true},
					},
				},
			},
		},
	}
}

func kafkaClustersDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	c := meta.(*Client)
	clusters, err := loadKafkaClusters(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Kafka Clusters: %s", createDescriptiveError(err))
	}

	var clusterList []map[string]interface{}
	for _, cluster := range clusters {
		// Compute rbac_crn using the helper
		rbacCrn, err := clusterCrnToRbacClusterCrn(cluster.Metadata.GetResourceName())
		if err != nil {
			rbacCrn = ""
		}
		clusterMap := map[string]interface{}{
			"id":                 cluster.GetId(),
			"display_name":       cluster.Spec.GetDisplayName(),
			"cloud":              cluster.Spec.GetCloud(),
			"region":             cluster.Spec.GetRegion(),
			"availability":       cluster.Spec.GetAvailability(),
			"kind":               cluster.GetKind(),
			"api_version":        cluster.GetApiVersion(),
			"bootstrap_endpoint": cluster.Spec.GetKafkaBootstrapEndpoint(),
			"rest_endpoint":      cluster.Spec.GetHttpEndpoint(),
			"rbac_crn":           rbacCrn,
		}
		clusterList = append(clusterList, clusterMap)
	}

	if err := d.Set("clusters", clusterList); err != nil {
		return diag.FromErr(err)
	}

	// Use environmentId as the data source ID for uniqueness
	d.SetId(environmentId)
	tflog.Debug(ctx, fmt.Sprintf("Set %d clusters for environment %s", len(clusterList), environmentId))
	return nil
}
