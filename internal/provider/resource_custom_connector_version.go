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
	"path/filepath"
	"regexp"
	"strings"
)

const (
	paramPluginId           = "plugin_id"
	paramConnectorClassName = "connector_class_name"
)

func customConnectorPluginVersionResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: customConnectorPluginVersionCreate,
		ReadContext:   customConnectorPluginVersionRead,
		DeleteContext: customConnectorPluginVersionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: customConnectorPluginVersionImport,
		},
		Schema: map[string]*schema.Schema{
			paramCloud: {
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
				ForceNew:     true,
			},
			paramFilename: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The path to the file that will be created.",
			},
			paramVersion: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^v"), "The version must start with 'v'"),
				Description:  "The version of the plugin.",
			},
			paramSensitiveConfigProperties: {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				ForceNew:    true,
				Description: "A list of sensitive properties where a sensitive property is a connector configuration property that must be hidden after a user enters property value when setting up connector.",
			},
			paramDocumentationLink: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "A documentation link of the Custom Connector Plugin.",
			},
			paramConnectorClass: {
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true,
				Description: "Java class or alias for connector. You can get connector class from connector documentation provided by developer.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramConnectorClassName: {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Java class or alias for connector. You can get connector class from connector documentation provided by developer.",
						},
						paramConnectorType: {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "Custom Connector type",
						},
					},
				},
			},
			paramPluginId: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The Plugin Id.",
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
				Description: "The object this REST resource represents.",
			},
		},
	}
}

func customConnectorPluginVersionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	filename := d.Get(paramFilename).(string)
	cloud := d.Get(paramCloud).(string)
	pluginId := d.Get(paramPluginId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	documentationLink := d.Get(paramDocumentationLink).(string)
	connectorClass := buildConnectorClass(d.Get(paramConnectorClass).(*schema.Set).List())
	version := d.Get(paramVersion).(string)
	sensitiveConfigProperties := convertToStringSlice(d.Get(paramSensitiveConfigProperties).(*schema.Set).List())
	//FOR STEP 1: model_ccpm_v1_custom_connect_plugin.go
	uploadID, err := uploadCustomConnectorVersionPlugin(c.ccpmApiContext(ctx), c, filename, cloud, environmentId)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin Version: %s", createDescriptiveError(err))
	}

	createCustomConnectorVersionRequest := ccpm.NewCcpmV1CustomConnectPluginVersion()

	createCustomConnectorVersionSpec := ccpm.NewCcpmV1CustomConnectPluginVersionSpec()
	createCustomConnectorVersionSpec.SetEnvironment(ccpm.EnvScopedObjectReference{Id: environmentId})
	createCustomConnectorVersionSpec.SetVersion(version)
	createCustomConnectorVersionSpec.SetDocumentationLink(documentationLink)
	createCustomConnectorVersionSpec.SetConnectorClasses(connectorClass)
	createCustomConnectorVersionSpec.SetSensitiveConfigProperties(sensitiveConfigProperties)
	createCustomConnectorVersionSpec.SetUploadSource(ccpm.CcpmV1CustomConnectPluginVersionSpecUploadSourceOneOf{
		CcpmV1UploadSourcePresignedUrl: ccpm.NewCcpmV1UploadSourcePresignedUrl(presignedUrlLocation, uploadID),
	})
	createCustomConnectorVersionRequest.SetSpec(*createCustomConnectorVersionSpec)

	createCustomConnectorPluginVersionRequestJson, err := json.Marshal(createCustomConnectorVersionRequest)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin Version: error marshaling %#v to json: %s", createCustomConnectorPluginVersionRequestJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Custom Connector Plugin Version: %s", createCustomConnectorPluginVersionRequestJson))

	createdCustomConnectorPluginVersion, resp, err := executeCustomConnectorPluginVersionCreate(c.ccpmApiContext(ctx), c, createCustomConnectorVersionRequest, pluginId)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin Version %q: %s", createdCustomConnectorPluginVersion.GetId(), createDescriptiveError(err, resp))
	}
	d.SetId(createdCustomConnectorPluginVersion.GetId())

	createdCustomConnectorPluginVersionJson, err := json.Marshal(createdCustomConnectorPluginVersion)
	if err != nil {
		return diag.Errorf("error creating Custom Connector Plugin Version %q: error marshaling %#v to json: %s", d.Id(), createdCustomConnectorPluginVersion, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Custom Connector Plugin Version %q: %s", d.Id(), createdCustomConnectorPluginVersionJson), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})

	return customConnectorPluginVersionRead(ctx, d, meta)
}

func customConnectorPluginVersionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
	c := meta.(*Client)
	pluginId := d.Get(paramPluginId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	req := c.ccpmClient.CustomConnectPluginVersionsCcpmV1Api.DeleteCcpmV1CustomConnectPluginVersion(c.ccpmApiContext(ctx), pluginId, d.Id()).Environment(environmentId)
	resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error deleting Custom Connector Plugin Version %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
	return nil
}

func customConnectorPluginVersionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
	c := meta.(*Client)
	filename := d.Get(paramFilename).(string)
	pluginId := d.Get(paramPluginId).(string)
	cloud := d.Get(paramCloud).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	_, err := readCustomConnectorPluginVersionAndSetAttributes(ctx, d, c, filename, pluginId, environmentId, cloud)
	if err != nil {
		return diag.Errorf("error reading Custom Connector Plugin Version %q: %s", d.Id(), createDescriptiveError(err))
	}
	return nil
}

func readCustomConnectorPluginVersionAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *Client, filename, pluginId, envId, cloud string) ([]*schema.ResourceData, error) {
	customConnectorPluginVersion, resp, err := executeCustomConnectorPluginVersionRead(c.ccpmApiContext(ctx), c, d.Id(), pluginId, envId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Custom Connector Plugin Version %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Custom Connector Plugin Version %q in TF state because Custom Connector Plugin could not be found on the server", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, createDescriptiveError(err)
	}
	customConnectorPluginVersionJson, err := json.Marshal(customConnectorPluginVersion)
	if err != nil {
		return nil, fmt.Errorf("error reading Custom Connector Plugin Version %q: error marshaling %#v to json: %s", d.Id(), customConnectorPluginVersionJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Custom Connector Plugin Version %q: %s", d.Id(), customConnectorPluginVersionJson), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})

	if _, err := setCustomConnectorPluginVersionAttributes(d, customConnectorPluginVersion, filename, pluginId, cloud); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setCustomConnectorPluginVersionAttributes(d *schema.ResourceData, customConnectorPlugin ccpm.CcpmV1CustomConnectPluginVersion, filename, pluginId, cloud string) (*schema.ResourceData, error) {
	spec := customConnectorPlugin.GetSpec()

	if err := d.Set(paramVersion, spec.GetVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, spec.GetEnvironment().Id, d); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDocumentationLink, spec.GetDocumentationLink()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramConnectorClass, buildTfConnectorClasses(spec.GetConnectorClasses())); err != nil {
		return nil, err
	}
	if err := d.Set(paramSensitiveConfigProperties, spec.GetSensitiveConfigProperties()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramFilename, filename); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramApiVersion, customConnectorPlugin.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, customConnectorPlugin.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCloud, cloud); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramPluginId, pluginId); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(customConnectorPlugin.GetId())
	return d, nil
}

func executeCustomConnectorPluginVersionCreate(ctx context.Context, c *Client, customConnectorPlugin *ccpm.CcpmV1CustomConnectPluginVersion, pluginId string) (ccpm.CcpmV1CustomConnectPluginVersion, *http.Response, error) {
	req := c.ccpmClient.CustomConnectPluginVersionsCcpmV1Api.CreateCcpmV1CustomConnectPluginVersion(ctx, pluginId).CcpmV1CustomConnectPluginVersion(*customConnectorPlugin)
	return req.Execute()
}

func executeCustomConnectorPluginVersionRead(ctx context.Context, c *Client, customConnectorPluginVersionId, pluginId, envId string) (ccpm.CcpmV1CustomConnectPluginVersion, *http.Response, error) {
	req := c.ccpmClient.CustomConnectPluginVersionsCcpmV1Api.GetCcpmV1CustomConnectPluginVersion(c.ccpmApiContext(ctx), pluginId, customConnectorPluginVersionId).Environment(envId)
	return req.Execute()
}

func uploadCustomConnectorVersionPlugin(ctx context.Context, c *Client, filename, cloud, environment string) (string, error) {
	extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if extension != "zip" && extension != "jar" {
		return "", fmt.Errorf(`error uploading Plugin: only file extensions ".jar" and ".zip" are allowed`)
	}

	createPresignedUrlRequest := *ccpm.NewCcpmV1PresignedUrl()
	createPresignedUrlRequest.SetContentFormat(extension)
	createPresignedUrlRequest.SetEnvironment(ccpm.EnvScopedObjectReference{Id: environment})
	if cloud != "" {
		createPresignedUrlRequest.SetCloud(cloud)
	}

	createdPresignedUrl, _, err := c.ccpmClient.PresignedUrlsCcpmV1Api.CreateCcpmV1PresignedUrl(c.ccpmApiContext(ctx)).CcpmV1PresignedUrl(createPresignedUrlRequest).Execute()
	if err != nil {
		return "", fmt.Errorf(`error uploading Plugin : error fetching presigned upload URL: %s`, err)
	}

	if err := uploadFile(createdPresignedUrl.GetUploadUrl(), filename, createdPresignedUrl.GetUploadFormData(), createdPresignedUrl.GetContentFormat(), cloud, false); err != nil {
		return "", fmt.Errorf(`error uploading Plugin: error uploading a file: %s`, err)
	}
	return createdPresignedUrl.GetUploadId(), nil
}

func customConnectorPluginVersionImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
	filename := getEnv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_VERSION_FILENAME", "")
	cloud := getEnv("IMPORT_CLOUD", "")
	envIDAndPluginIDAndVersionID := d.Id()
	parts := strings.Split(envIDAndPluginIDAndVersionID, "/")

	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Connector: invalid format: expected '<Environment ID>/<Plugin ID>/<Version ID>'")
	}

	environmentId := parts[0]
	pluginId := parts[1]
	versionId := parts[2]
	d.SetId(versionId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()

	if _, err := readCustomConnectorPluginVersionAndSetAttributes(ctx, d, meta.(*Client), filename, pluginId, environmentId, cloud); err != nil {
		return nil, fmt.Errorf("error importing Custom Connector Plugin Version %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Custom Connector Plugin Version %q", d.Id()), map[string]interface{}{customConnectorPluginVersionLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func buildConnectorClass(connectorClass []interface{}) []ccpm.CcpmV1ConnectorClass {
	classes := make([]ccpm.CcpmV1ConnectorClass, len(connectorClass))
	for index, tfClass := range connectorClass {
		class := ccpm.NewCcpmV1ConnectorClassWithDefaults()
		tfClassMap := tfClass.(map[string]interface{})
		if className, exists := tfClassMap[paramConnectorClassName].(string); exists {
			class.SetClassName(className)
		}
		if classType, exists := tfClassMap[paramConnectorType].(string); exists {
			class.SetType(classType)
		}
		classes[index] = *class
	}
	return classes
}

func buildTfConnectorClasses(classes []ccpm.CcpmV1ConnectorClass) *[]map[string]interface{} {
	tfClasses := make([]map[string]interface{}, len(classes))
	for i, class := range classes {
		tfClasses[i] = *buildTfClasses(class)
	}
	return &tfClasses
}

func buildTfClasses(class ccpm.CcpmV1ConnectorClass) *map[string]interface{} {
	tfClass := make(map[string]interface{})
	tfClass[paramConnectorClassName] = class.GetClassName()
	tfClass[paramConnectorType] = class.GetType()
	return &tfClass
}
