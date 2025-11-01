package provider

import (
	"context"
	"encoding/json"
	"fmt"
	fcpm "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	paramDefaultPoolEnabled = "default_compute_pool_enabled"
	paramMaxCFU             = "default_max_cfu"
)

func computePoolConfigResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: computePoolConfigCreate,
		ReadContext:   computePoolConfigRead,
		UpdateContext: computePoolConfigUpdate,
		DeleteContext: computePoolConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: computePoolConfigImport,
		},
		Schema: map[string]*schema.Schema{
			paramDefaultPoolEnabled: {
				Type:         schema.TypeBool,
				Description:  "Whether default compute pools are enabled for the organization.",
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{paramDefaultPoolEnabled, paramMaxCFU},
			},
			paramMaxCFU: {
				Type:         schema.TypeInt,
				Description:  "Maximum number of Confluent Flink Units (CFU).",
				Optional:     true,
				Computed:     true,
				AtLeastOneOf: []string{paramDefaultPoolEnabled, paramMaxCFU},
			},
			paramApiVersion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramKind: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func computePoolConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	defaultPool := d.Get(paramDefaultPoolEnabled).(bool)
	maxCFU := d.Get(paramMaxCFU).(int)

	computePoolRequest := fcpm.NewFcpmV2OrgComputePoolConfigUpdate()
	spec := fcpm.FcpmV2OrgComputePoolConfigSpec{}

	if _, ok := d.GetOk(paramMaxCFU); ok {
		spec.SetDefaultPoolMaxCfu(int32(maxCFU))
	}

	if _, ok := d.GetOkExists(paramDefaultPoolEnabled); ok {
		spec.SetDefaultPoolEnabled(defaultPool)
	}

	computePoolRequest.SetSpec(spec)
	createComputePoolJson, err := json.Marshal(computePoolRequest)
	if err != nil {
		return diag.Errorf("error creating Compute Pool Config %q: error marshaling %#v to json: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, computePoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Compute Pool Request: %s", createComputePoolJson))

	req := c.fcpmClient.OrgComputePoolConfigsFcpmV2Api.UpdateFcpmV2OrgComputePoolConfig(c.fcpmApiContext(ctx)).FcpmV2OrgComputePoolConfigUpdate(*computePoolRequest)
	config, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, createDescriptiveError(err, resp))
	}

	d.SetId(config.GetOrganizationId())

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Compute Pool Config %q", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}))

	return computePoolConfigRead(ctx, d, meta)

}

func computePoolConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramMaxCFU, paramDefaultPoolEnabled) {
		return diag.Errorf("error updating Flink Compute Config Pool %q: only %q, and %q attributes can be updated for Flink Compute Pool Config", d.Id(), paramMaxCFU, paramDefaultPoolEnabled)
	}

	c := meta.(*Client)
	updateComputePoolRequest := fcpm.NewFcpmV2OrgComputePoolConfigUpdate()
	updateSpec := fcpm.FcpmV2OrgComputePoolConfigSpec{}

	if d.HasChange(paramMaxCFU) {
		updateSpec.SetDefaultPoolMaxCfu(int32(d.Get(paramMaxCFU).(int)))
	}
	if d.HasChange(paramDefaultPoolEnabled) {
		updateSpec.SetDefaultPoolEnabled(d.Get(paramDefaultPoolEnabled).(bool))
	}

	updateComputePoolRequest.SetSpec(updateSpec)
	updateComputePoolRequestJson, err := json.Marshal(updateComputePoolRequest)
	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool Config %q: error marshaling %#v to json: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, updateComputePoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, updateComputePoolRequestJson))

	req := c.fcpmClient.OrgComputePoolConfigsFcpmV2Api.UpdateFcpmV2OrgComputePoolConfig(c.fcpmApiContext(ctx)).FcpmV2OrgComputePoolConfigUpdate(*updateComputePoolRequest)
	updatedComputePool, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, createDescriptiveError(err, resp))
	}

	updatedComputePoolJson, err := json.Marshal(updatedComputePool)
	if err != nil {
		return diag.Errorf("error updating Flink Compute Pool Config %q: error marshaling %#v to json: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, updatedComputePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Flink Compute Pool %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, updatedComputePoolJson))
	return computePoolConfigRead(ctx, d, meta)
}

func computePoolConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Compute Pool Config %q", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}))

	if _, err := readComputePoolConfigAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Flink Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, createDescriptiveError(err)))
	}
	return nil
}

func readComputePoolConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	computePoolConfig, resp, err := executeComputePoolConfigRead(c.fcpmApiContext(ctx), c)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Compute Pool %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, createDescriptiveError(err, resp)))
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Compute Pool Config in TF state because Flink Compute Pool could not be found on the server"))
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	computePoolJson, err := json.Marshal(computePoolConfig)
	if err != nil {
		return nil, fmt.Errorf("error reading Flink Compute Pool Config %q: error marshaling %#v to json: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, computePoolConfig, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Compute Pool Config %q: %s", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}, computePoolJson))

	if _, err := setComputePoolConfigAttributes(d, computePoolConfig); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Compute Pool Config %q", map[string]interface{}{computePoolLoggingConfigKey: d.Id()}))

	return []*schema.ResourceData{d}, nil
}

func setComputePoolConfigAttributes(d *schema.ResourceData, computePool fcpm.FcpmV2OrgComputePoolConfig) (*schema.ResourceData, error) {
	if err := d.Set(paramMaxCFU, computePool.Spec.GetDefaultPoolMaxCfu()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDefaultPoolEnabled, computePool.Spec.GetDefaultPoolEnabled()); err != nil {
		return nil, err
	}

	if err := d.Set(paramApiVersion, computePool.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, computePool.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}

	return d, nil
}

func executeComputePoolConfigRead(ctx context.Context, c *Client) (fcpm.FcpmV2OrgComputePoolConfig, *http.Response, error) {
	req := c.fcpmClient.OrgComputePoolConfigsFcpmV2Api.GetFcpmV2OrgComputePoolConfig(c.fcpmApiContext(ctx))
	return req.Execute()
}

func computePoolConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Compute Pool Config %q", d.Id()), map[string]interface{}{computePoolLoggingConfigKey: d.Id()})

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Compute Pool Config %q", d.Id()), map[string]interface{}{computePoolLoggingConfigKey: d.Id()})
	return nil
}

func computePoolConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Compute Pool Config %q", d.Id()), map[string]interface{}{computePoolLoggingConfigKey: d.Id()})

	orgId := d.Id()

	d.SetId(orgId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readComputePoolConfigAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing Flink Compute Pool Config %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Compute Pool Config %q", d.Id()), map[string]interface{}{computePoolLoggingConfigKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
