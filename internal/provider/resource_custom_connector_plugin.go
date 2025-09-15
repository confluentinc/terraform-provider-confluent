// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	ccp "github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	paramDocumentationLink         = "documentation_link"
	paramConnectorClass            = "connector_class"
	paramConnectorType             = "connector_type"
	paramSensitiveConfigProperties = "sensitive_config_properties"
	paramFilename                  = "filename"
	presignedUrlLocation           = "PRESIGNED_URL_LOCATION"
)

func customConnectorPluginResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: customConnectorPluginCreate,
		ReadContext:   customConnectorPluginRead,
		UpdateContext: customConnectorPluginUpdate,
		DeleteContext: customConnectorPluginDelete,
		Importer: &schema.ResourceImporter{
			StateContext: customConnectorPluginImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human-readable name for the Custom Connector Plugin.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form description of the Custom Connector Plugin.",
			},
			paramDocumentationLink: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A documentation link of the Custom Connector Plugin.",
			},
			paramConnectorClass: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Java class or alias for connector. You can get connector class from connector documentation provided by developer.",
			},
			paramConnectorType: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Custom Connector type.",
				ValidateFunc: validation.StringInSlice([]string{"SOURCE", "SINK"}, false),
			},
			paramSensitiveConfigProperties: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "A list of sensitive properties where a sensitive property is a connector configuration property that must be hidden after a user enters property value when setting up connector.",
			},
			paramFilename: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The path to the file that will be created.",
			},
			paramCloud: {
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
			},
		},
	}
}

func customConnectorPluginUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramDocumentationLink) {
		return diag.Errorf("error updating Custom Connector Plugin %q: only %q, %q, %q attributes can be updated for Custom Connector Plugin", d.Id(), paramDisplayName, paramDescription, paramDocumentationLink)
	}

	updateCustomConnectorPluginRequest := ccp.NewConnectV1CustomConnectorPluginUpdate()

	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateCustomConnectorPluginRequest.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updateCustomConnectorPluginRequest.SetDescription(updatedDescription)
	}
	if d.HasChange(paramDocumentationLink) {
		updatedDocumentationLink := d.Get(paramDocumentationLink).(string)
		updateCustomConnectorPluginRequest.SetDocumentationLink(updatedDocumentationLink)
	}

	updateCustomConnectorPluginRequestJson, err := json.Marshal(updateCustomConnectorPluginRequest)
	if err != nil {
		return diag.Errorf("error updating Custom Connector Plugin %q: error marshaling %#v to json: %s", d.Id(), updateCustomConnectorPluginRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Custom Connector Plugin %q: %s", d.Id(), updateCustomConnectorPluginRequestJson), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedCustomConnectorPlugin, resp, err := c.ccpClient.CustomConnectorPluginsConnectV1Api.UpdateConnectV1CustomConnectorPlugin(c.ccpApiContext(ctx), d.Id()).ConnectV1CustomConnectorPluginUpdate(*updateCustomConnectorPluginRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Custom Connector Plugin %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedCustomConnectorPluginJson, err := json.Marshal(updatedCustomConnectorPlugin)
	if err != nil {
		return diag.Errorf("error updating Custom Connector Plugin %q: error marshaling %#v to json: %s", d.Id(), updatedCustomConnectorPlugin, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Custom Connector Plugin %q: %s", d.Id(), updatedCustomConnectorPluginJson), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	return customConnectorPluginRead(ctx, d, meta)
}

func customConnectorPluginCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	documentationLink := d.Get(paramDocumentationLink).(string)
	connectorClass := d.Get(paramConnectorClass).(string)
	connectorType := d.Get(paramConnectorType).(string)
	sensitiveConfigProperties := convertToStringSlice(d.Get(paramSensitiveConfigProperties).(*schema.Set).List())
	filename := d.Get(paramFilename).(string)
	cloud := d.Get(paramCloud).(string)

	// Part 1: Get Upload ID
	uploadID, err := uploadCustomConnectorPlugin(c.ccpApiContext(ctx), c, filename, cloud)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin: %s", createDescriptiveError(err))
	}

	// Part 2: Creating Custom Connector Plugin
	createCustomConnectorPluginRequest := ccp.NewConnectV1CustomConnectorPlugin()

	if cloud != "" {
		createCustomConnectorPluginRequest.SetCloud(cloud)
	}

	createCustomConnectorPluginRequest.SetDisplayName(displayName)
	createCustomConnectorPluginRequest.SetDescription(description)
	createCustomConnectorPluginRequest.SetDocumentationLink(documentationLink)
	createCustomConnectorPluginRequest.SetConnectorClass(connectorClass)
	createCustomConnectorPluginRequest.SetConnectorType(connectorType)
	createCustomConnectorPluginRequest.SetSensitiveConfigProperties(sensitiveConfigProperties)
	createCustomConnectorPluginRequest.SetUploadSource(ccp.ConnectV1CustomConnectorPluginUploadSourceOneOf{
		ConnectV1UploadSourcePresignedUrl: ccp.NewConnectV1UploadSourcePresignedUrl(presignedUrlLocation, uploadID),
	})

	createCustomConnectorPluginRequestJson, err := json.Marshal(createCustomConnectorPluginRequest)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin: error marshaling %#v to json: %s", createCustomConnectorPluginRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Custom Connector Plugin: %s", createCustomConnectorPluginRequestJson))

	createdCustomConnectorPlugin, resp, err := executeCustomConnectorPluginCreate(c.ccpApiContext(ctx), c, createCustomConnectorPluginRequest)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin %q: %s", displayName, createDescriptiveError(err, resp))
	}
	d.SetId(createdCustomConnectorPlugin.GetId())

	createdCustomConnectorPluginJson, err := json.Marshal(createdCustomConnectorPlugin)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin %q: error marshaling %#v to json: %s", d.Id(), createdCustomConnectorPlugin, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Custom Connector Plugin %q: %s", d.Id(), createdCustomConnectorPluginJson), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	return customConnectorPluginRead(ctx, d, meta)
}

// https://github.com/confluentinc/cli/blob/main/internal/connect/command_custom_plugin_create.go#L78
func uploadCustomConnectorPlugin(ctx context.Context, c *Client, filename, cloud string) (string, error) {
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if extension != "zip" && extension != "jar" {
		return "", fmt.Errorf(`error uploading Custom Connector Plugin: only file extensions ".jar" and ".zip" are allowed`)
	}

	createPresignedUrlRequest := *ccp.NewConnectV1PresignedUrlRequest()
	createPresignedUrlRequest.SetContentFormat(extension)
	if cloud != "" {
		createPresignedUrlRequest.SetCloud(cloud)
	}

	createdPresignedUrl, _, err := c.ccpClient.PresignedUrlsConnectV1Api.PresignedUploadUrlConnectV1PresignedUrl(c.ccpApiContext(ctx)).ConnectV1PresignedUrlRequest(createPresignedUrlRequest).Execute()
	if err != nil {
		return "", fmt.Errorf(`error uploading Custom Connector Plugin: error fetching presigned upload URL: %s`, err)
	}

	if err := uploadFile(createdPresignedUrl.GetUploadUrl(), filename, createdPresignedUrl.GetUploadFormData(), createdPresignedUrl.GetContentFormat(), cloud, false); err != nil {
		return "", fmt.Errorf(`error uploading Custom Connector Plugin: error uploading a file: %s`, err)
	}
	return createdPresignedUrl.GetUploadId(), nil
}

func executeCustomConnectorPluginCreate(ctx context.Context, c *Client, customConnectorPlugin *ccp.ConnectV1CustomConnectorPlugin) (ccp.ConnectV1CustomConnectorPlugin, *http.Response, error) {
	req := c.ccpClient.CustomConnectorPluginsConnectV1Api.CreateConnectV1CustomConnectorPlugin(c.ccpApiContext(ctx)).ConnectV1CustomConnectorPlugin(*customConnectorPlugin)
	return req.Execute()
}

func customConnectorPluginDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.ccpClient.CustomConnectorPluginsConnectV1Api.DeleteConnectV1CustomConnectorPlugin(c.ccpApiContext(ctx), d.Id())
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Custom Connector Plugin %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	return nil
}

func executeCustomConnectorPluginRead(ctx context.Context, c *Client, customConnectorPluginId string) (ccp.ConnectV1CustomConnectorPlugin, *http.Response, error) {
	req := c.ccpClient.CustomConnectorPluginsConnectV1Api.GetConnectV1CustomConnectorPlugin(c.ccpApiContext(ctx), customConnectorPluginId)
	return req.Execute()
}

func customConnectorPluginRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})
	c := meta.(*Client)
	filename := d.Get(paramFilename).(string)

	_, err := readCustomConnectorPluginAndSetAttributes(ctx, d, c, filename)
	if err != nil {
		return diag.Errorf("error reading Custom Connector Plugin %q: %s", d.Id(), createDescriptiveError(err))
	}

	return nil
}

func readCustomConnectorPluginAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *Client, filename string) ([]*schema.ResourceData, error) {
	customConnectorPlugin, resp, err := executeCustomConnectorPluginRead(c.ccpApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Custom Connector Plugin %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Custom Connector Plugin %q in TF state because Custom Connector Plugin could not be found on the server", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, createDescriptiveError(err)
	}
	customConnectorPluginJson, err := json.Marshal(customConnectorPlugin)
	if err != nil {
		return nil, fmt.Errorf("error reading Custom Connector Plugin %q: error marshaling %#v to json: %s", d.Id(), customConnectorPlugin, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Custom Connector Plugin %q: %s", d.Id(), customConnectorPluginJson), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	if _, err := setCustomConnectorPluginAttributes(d, customConnectorPlugin, filename); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setCustomConnectorPluginAttributes(d *schema.ResourceData, customConnectorPlugin ccp.ConnectV1CustomConnectorPlugin, filename string) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, customConnectorPlugin.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, customConnectorPlugin.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCloud, customConnectorPlugin.GetCloud()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDocumentationLink, customConnectorPlugin.GetDocumentationLink()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramConnectorClass, customConnectorPlugin.GetConnectorClass()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramConnectorType, customConnectorPlugin.GetConnectorType()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramSensitiveConfigProperties, customConnectorPlugin.GetSensitiveConfigProperties()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramFilename, filename); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(customConnectorPlugin.GetId())
	return d, nil
}

func customConnectorPluginImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})
	filename := getEnv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_FILENAME", "")
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readCustomConnectorPluginAndSetAttributes(ctx, d, meta.(*Client), filename); err != nil {
		return nil, fmt.Errorf("error importing Custom Connector Plugin %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Custom Connector Plugin %q", d.Id()), map[string]interface{}{customConnectorPluginLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
