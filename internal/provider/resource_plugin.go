package provider

import (
	"context"
	"encoding/json"
	"fmt"
	ccpm "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

func pluginResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: pluginCreate,
		ReadContext:   pluginRead,
		UpdateContext: pluginUpdate,
		DeleteContext: pluginDelete,
		Importer: &schema.ResourceImporter{
			StateContext: pluginImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human-readable name for the Plugin.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form description of the Plugin.",
			},
			paramCloud: {
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
			},
			paramRuntimeLanguage: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Runtime language of the Plugin",
			},
			paramEnvironment: environmentSchema(),
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The schema version of this representation of a resource.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object the plugin represents.",
			},
		},
	}
}

func pluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	cloud := d.Get(paramCloud).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	createPluginRequest := ccpm.NewCcpmV1CustomConnectPlugin()

	pluginSpec := ccpm.NewCcpmV1CustomConnectPluginSpec()
	pluginSpec.SetEnvironment(ccpm.EnvScopedObjectReference{Id: environmentId})
	pluginSpec.SetCloud(cloud)
	pluginSpec.SetDescription(description)
	pluginSpec.SetDisplayName(displayName)

	createPluginRequest.SetSpec(*pluginSpec)

	pluginRequestJson, err := json.Marshal(createPluginRequest)
	if err != nil {
		return diag.Errorf("error creating Plugin: error marshaling %#v to json: %s", createPluginRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Plugin: %s", pluginRequestJson))
	createdPlugin, resp, err := executePluginCreate(c.ccpmApiContext(ctx), c, createPluginRequest)
	if err != nil {
		return diag.Errorf("error creating Plugin %q: %s", displayName, createDescriptiveError(err, resp))
	}
	d.SetId(createdPlugin.GetId())

	createdPluginJson, err := json.Marshal(createdPlugin)
	if err != nil {
		return diag.Errorf("error creating Plugin %q: error marshaling %#v to json: %s", d.Id(), createdPluginJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Plugin %q: %s", d.Id(), createdPluginJson), map[string]interface{}{pluginLoggingKey: d.Id()})

	return pluginRead(ctx, d, meta)
}

func pluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription) {
		return diag.Errorf("error updating Plugin %q: only %q, %q attributes can be updated for Custom Connector Plugin", d.Id(), paramDisplayName, paramDescription)
	}
	updatePluginRequest := ccpm.NewCcpmV1CustomConnectPluginUpdate()
	updatePluginSpec := ccpm.NewCcpmV1CustomConnectPluginSpecUpdate()
	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updatePluginSpec.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updatePluginSpec.SetDescription(updatedDescription)
	}
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updatePluginSpec.SetEnvironment(ccpm.EnvScopedObjectReference{Id: environmentId})

	if d.HasChange(paramDisplayName) || d.HasChange(paramDescription) {
		updatePluginRequest.SetSpec(*updatePluginSpec)
	}
	updatePluginRequestJson, err := json.Marshal(updatePluginRequest)
	if err != nil {
		return diag.Errorf("error updating Plugin %q: error marshaling %#v to json: %s", d.Id(), updatePluginRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Plugin %q: %s", d.Id(), updatePluginRequestJson), map[string]interface{}{pluginLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedPlugin, resp, err := c.ccpmClient.CustomConnectPluginsCcpmV1Api.UpdateCcpmV1CustomConnectPlugin(c.ccpmApiContext(ctx), d.Id()).CcpmV1CustomConnectPluginUpdate(*updatePluginRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Plugin %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedPluginJson, err := json.Marshal(updatedPlugin)
	if err != nil {
		return diag.Errorf("error updating Plugin %q: error marshaling %#v to json: %s", d.Id(), updatedPlugin, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Plugin %q: %s", d.Id(), updatedPluginJson), map[string]interface{}{pluginLoggingKey: d.Id()})

	return pluginRead(ctx, d, meta)
}

func pluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	_, err := readPluginAndSetAttributes(ctx, d, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Plugin %q: %s", d.Id(), createDescriptiveError(err))
	}

	return nil
}

func readPluginAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *Client, environmentId string) ([]*schema.ResourceData, error) {
	plugin, resp, err := executePluginRead(c.ccpmApiContext(ctx), c, d.Id(), environmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Plugin %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{pluginLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Plugin %q in TF state because Plugin could not be found on the server", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, createDescriptiveError(err)
	}
	pluginJson, err := json.Marshal(plugin)
	if err != nil {
		return nil, fmt.Errorf("error reading Plugin %q: error marshaling %#v to json: %s", d.Id(), plugin, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Plugin %q: %s", d.Id(), pluginJson), map[string]interface{}{pluginLoggingKey: d.Id()})

	if _, err := setPluginAttributes(d, plugin); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setPluginAttributes(d *schema.ResourceData, plugin ccpm.CcpmV1CustomConnectPlugin) (*schema.ResourceData, error) {
	spec := plugin.GetSpec()
	if err := d.Set(paramDisplayName, spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, spec.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCloud, spec.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, spec.GetEnvironment().Id, d); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, plugin.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, plugin.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRuntimeLanguage, spec.GetRuntimeLanguage()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(plugin.GetId())
	return d, nil
}

func pluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})
	c := meta.(*Client)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	req := c.ccpmClient.CustomConnectPluginsCcpmV1Api.DeleteCcpmV1CustomConnectPlugin(c.ccpmApiContext(ctx), d.Id()).Environment(environmentId)
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Plugin %q: %s", d.Id(), createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})
	return nil
}

func executePluginCreate(ctx context.Context, c *Client, customConnectorPlugin *ccpm.CcpmV1CustomConnectPlugin) (ccpm.CcpmV1CustomConnectPlugin, *http.Response, error) {
	req := c.ccpmClient.CustomConnectPluginsCcpmV1Api.CreateCcpmV1CustomConnectPlugin(c.ccpmApiContext(ctx)).CcpmV1CustomConnectPlugin(*customConnectorPlugin)
	return req.Execute()
}

func executePluginRead(ctx context.Context, c *Client, pluginId, envID string) (ccpm.CcpmV1CustomConnectPlugin, *http.Response, error) {
	req := c.ccpmClient.CustomConnectPluginsCcpmV1Api.GetCcpmV1CustomConnectPlugin(c.ccpmApiContext(ctx), pluginId).Environment(envID)
	return req.Execute()
}

func pluginImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})

	envIDAndPluginId := d.Id()
	parts := strings.Split(envIDAndPluginId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Plugin: invalid format: expected '<env ID>/<plugin ID>'")
	}

	environmentId := parts[0]
	pluginId := parts[1]
	d.SetId(pluginId)

	d.MarkNewResource()
	if _, err := readPluginAndSetAttributes(ctx, d, meta.(*Client), environmentId); err != nil {
		return nil, fmt.Errorf("error importing Plugin %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Plugin %q", d.Id()), map[string]interface{}{pluginLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
