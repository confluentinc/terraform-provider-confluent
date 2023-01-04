package provider

import (
	"context"
	"encoding/json"
	"fmt"
	ksql "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	paramTopicPrefix              = "topic_prefix"
	paramCredentialIdentity       = "credential_identity"
	paramStorage                  = "storage"
	paramUseDetailedProcessingLog = "use_detailed_processing_log"
	ksqlCreateTimeout             = 12 * time.Hour
)

func ksqlResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: ksqlCreate,
		ReadContext:   ksqlRead,
		DeleteContext: ksqlDelete,
		Importer: &schema.ResourceImporter{
			StateContext: ksqlImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Description:  "The name of the ksqlDB cluster.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramUseDetailedProcessingLog: {
				Type:        schema.TypeBool,
				Description: "Controls whether the row data should be included in the processing log topic. Set it to `false` if you don't want to emit sensitive information to the processing log. Defaults to `true`.",
				Optional:    true,
				Default:     true,
				ForceNew:    true,
			},
			paramCsu: {
				Type:         schema.TypeInt,
				Description:  "The number of Confluent Streaming Units (CSUs) for the ksqlDB cluster.",
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(1),
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRestEndpoint: {
				Type:        schema.TypeString,
				Description: "The API endpoint of the ksqlDB cluster.",
				Computed:    true,
			},
			paramStorage: {
				Type:        schema.TypeInt,
				Description: "The amount of storage (in GB) provisioned to the ksqlDB cluster.",
				Computed:    true,
			},
			paramTopicPrefix: {
				Type:        schema.TypeString,
				Description: "Topic name prefix used by this ksqlDB cluster. Used to assign ACLs for this ksqlDB cluster to use.",
				Computed:    true,
			},
			paramKafkaCluster:       requiredKafkaClusterBlockSchema(),
			paramCredentialIdentity: credentialIdentityBlockSchema(),
			paramEnvironment:        environmentSchema(),
			paramResourceName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Confluent Resource Name of the ksqlDB cluster.",
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(ksqlCreateTimeout),
		},
	}
}

func ksqlCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	useDetailedProcessingLog := d.Get(paramUseDetailedProcessingLog).(bool)
	csu := d.Get(paramCsu).(int)
	kafkaClusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	credentialIdentityId := extractStringValueFromBlock(d, paramCredentialIdentity, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec := ksql.NewKsqldbcmV2ClusterSpec()
	spec.SetDisplayName(displayName)
	spec.SetUseDetailedProcessingLog(useDetailedProcessingLog)
	spec.SetCsu(int32(csu))
	// TODO: KSQL-1234: Remove placeholders from ObjectReference
	spec.SetKafkaCluster(ksql.ObjectReference{Id: kafkaClusterId, Environment: &environmentId, ResourceName: "_", Related: "_"})
	spec.SetCredentialIdentity(ksql.ObjectReference{Id: credentialIdentityId, ResourceName: "_", Related: "_"})
	spec.SetEnvironment(ksql.ObjectReference{Id: environmentId, ResourceName: "_", Related: "_"})

	createKsqlClusterRequest := ksql.KsqldbcmV2Cluster{Spec: spec}
	createKsqlClusterRequestJson, err := json.Marshal(createKsqlClusterRequest)
	if err != nil {
		return diag.Errorf("error creating ksqlDB Cluster: error marshaling %#v to json: %s", createKsqlClusterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new ksqlDB Cluster: %s", createKsqlClusterRequestJson))

	createdKsqlCluster, _, err := executeKsqlCreate(c.ksqlApiContext(ctx), c, &createKsqlClusterRequest)
	if err != nil {
		return diag.Errorf("error creating ksqlDB Cluster %q: %s", createdKsqlCluster.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdKsqlCluster.GetId())

	if err := waitForKsqlClusterToProvision(c.ksqlApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for ksqlDB Cluster %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdKsqlClusterJson, err := json.Marshal(createdKsqlCluster)
	if err != nil {
		return diag.Errorf("error creating ksqlDB Cluster %q: error marshaling %#v to json: %s", d.Id(), createdKsqlCluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating ksqlDB Cluster %q: %s", d.Id(), createdKsqlClusterJson), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	return ksqlRead(ctx, d, meta)
}

func ksqlRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	clusterId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readKsqlClusterAndSetAttributes(ctx, d, meta, environmentId, clusterId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading ksqlDB Cluster %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func ksqlDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	err := executeKsqlDelete(c.ksqlApiContext(ctx), c, environmentId, d.Id())
	if err != nil {
		return diag.Errorf("error deleting ksqlDB Cluster %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	return nil
}

func ksqlImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	envIDAndClusterID := d.Id()
	parts := strings.Split(envIDAndClusterID, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing ksqlDB Cluster: invalid format: expected '<env ID>/<ksqlDB cluster ID>'")
	}

	environmentId := parts[0]
	clusterId := parts[1]
	d.SetId(clusterId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readKsqlClusterAndSetAttributes(ctx, d, meta, environmentId, clusterId); err != nil {
		return nil, fmt.Errorf("error importing ksqlDB Cluster %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readKsqlClusterAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, clusterId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	cluster, resp, err := executeKsqlRead(c.ksqlApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading ksqlDB Cluster %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing ksqlDB Cluster %q in TF state because ksqlDB Cluster could not be found on the server", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	clusterJson, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("error reading ksqlDB Cluster %q: error marshaling %#v to json: %s", clusterId, cluster, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched ksqlDB Cluster %q: %s", d.Id(), clusterJson), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	if _, err := setKsqlAttributes(d, cluster); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading ksqlDB Cluster %q", d.Id()), map[string]interface{}{ksqlClusterLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func executeKsqlCreate(ctx context.Context, c *Client, cluster *ksql.KsqldbcmV2Cluster) (ksql.KsqldbcmV2Cluster, *http.Response, error) {
	req := c.ksqlClient.ClustersKsqldbcmV2Api.CreateKsqldbcmV2Cluster(c.ksqlApiContext(ctx)).KsqldbcmV2Cluster(*cluster)
	return req.Execute()
}

func executeKsqlRead(ctx context.Context, c *Client, environmentId, clusterId string) (ksql.KsqldbcmV2Cluster, *http.Response, error) {
	req := c.ksqlClient.ClustersKsqldbcmV2Api.GetKsqldbcmV2Cluster(c.ksqlApiContext(ctx), clusterId).Environment(environmentId)
	return req.Execute()
}

func executeKsqlDelete(ctx context.Context, c *Client, environmentId, clusterId string) error {
	req := c.ksqlClient.ClustersKsqldbcmV2Api.DeleteKsqldbcmV2Cluster(c.ksqlApiContext(ctx), clusterId).Environment(environmentId)
	_, err := req.Execute()
	return err
}

func credentialIdentityBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The credential_identity to which this belongs. The credential_identity can be one of iam.v2.User, iam.v2.ServiceAccount.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(u-|sa-)"), "the credential identity must be of the form 'u-' or 'sa-'"),
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func setKsqlAttributes(d *schema.ResourceData, cluster ksql.KsqldbcmV2Cluster) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, cluster.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramUseDetailedProcessingLog, cluster.Spec.GetUseDetailedProcessingLog()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCsu, cluster.Spec.GetCsu()); err != nil {
		return nil, createDescriptiveError(err)
	}

	if err := d.Set(paramApiVersion, cluster.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, cluster.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRestEndpoint, cluster.Status.GetHttpEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramStorage, cluster.Status.GetStorage()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramTopicPrefix, cluster.Status.GetTopicPrefix()); err != nil {
		return nil, createDescriptiveError(err)
	}

	if err := d.Set(paramResourceName, cluster.Metadata.GetResourceName()); err != nil {
		return nil, createDescriptiveError(err)
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, cluster.Spec.KafkaCluster.GetId(), d); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramCredentialIdentity, paramId, cluster.Spec.CredentialIdentity.GetId(), d); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, cluster.Spec.Environment.GetId(), d); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(cluster.GetId())
	return d, nil
}
