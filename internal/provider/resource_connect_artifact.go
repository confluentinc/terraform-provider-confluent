package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	cam "github.com/confluentinc/ccloud-sdk-go-v2/cam/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func connectArtifactResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: connectArtifactCreate,
		ReadContext:   connectArtifactRead,
		DeleteContext: connectArtifactDelete,
		Importer: &schema.ResourceImporter{
			StateContext: connectArtifactImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The unique name of the Connect Artifact per cloud, region, environment scope.",
				ValidateFunc: validation.StringLenBetween(1, 60),
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Cloud provider where the Connect Artifact archive is uploaded.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The Cloud provider region the Connect Artifact archive is uploaded.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramEnvironment: environmentSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Archive format of the Connect Artifact (JAR).",
			},
			paramArtifactFile: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(val.(string)), "."))
					if extension != "jar" {
						errs = append(errs, fmt.Errorf("%q must have extension .jar", key))
					}
					return
				},
				Description: "The artifact file for Connect Artifact in JAR format.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the Connect Artifact.",
			},
			paramPlugins: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of classes present in the Connect Artifact uploaded.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"class": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The class name of the plugin.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the plugin.",
						},
					},
				},
			},
			paramUsages: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of resource crns where this Connect artifact is being used.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func connectArtifactCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	name := d.Get(paramDisplayName).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	contentFormat := d.Get(paramContentFormat).(string)
	artifactFile := d.Get(paramArtifactFile).(string)
	description := d.Get(paramDescription).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	// Step 1: Get presigned URL
	request := cam.CamV1PresignedUrlRequest{
		Cloud:       cam.PtrString(cloud),
		Region:      cam.PtrString(region),
		Environment: cam.PtrString(environmentId),
	}
	if contentFormat != "" {
		request.SetContentFormat(contentFormat)
	}
	resp, _, err := getConnectPresignedUrl(c.camApiContext(ctx), c, request)
	if err != nil {
		return diag.Errorf("error uploading Connect Artifact: error fetching presigned upload URL %s", createDescriptiveError(err))
	}

	// Step 2: Upload file to presigned URL
	if err := uploadFile(resp.GetUploadUrl(), artifactFile, resp.GetUploadFormData(), resp.GetContentFormat(), cloud, false); err != nil {
		return diag.Errorf("error uploading Connect Artifact: %s", createDescriptiveError(err))
	}

	// Step 3: Create artifact with upload ID
	createArtifactRequest := cam.CamV1ConnectArtifactSpec{
		DisplayName:   name,
		Cloud:         cloud,
		Region:        region,
		Environment:   environmentId,
		ContentFormat: cam.PtrString(contentFormat),
		UploadSource: &cam.CamV1ConnectArtifactSpecUploadSourceOneOf{
			CamV1UploadSourcePresignedUrl: &cam.CamV1UploadSourcePresignedUrl{
				Location: "PRESIGNED_URL_LOCATION",
				UploadId: resp.GetUploadId(),
			},
		},
	}
	if description != "" {
		createArtifactRequest.SetDescription(description)
	}

	createdArtifact, _, err := executeConnectArtifactCreate(c.camApiContext(ctx), c, createArtifactRequest)
	if err != nil {
		return diag.Errorf("error creating Connect Artifact: %s", createDescriptiveError(err))
	}
	d.SetId(createdArtifact.GetId())

	createdArtifactJson, err := json.Marshal(createdArtifact)
	if err != nil {
		return diag.Errorf("error creating Connect Artifact: error marshaling %#v to json: %s", createdArtifactJson, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Connect Artifact %q: %s", d.Id(), createdArtifactJson), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
	return connectArtifactRead(ctx, d, meta)
}

func getConnectPresignedUrl(ctx context.Context, c *Client, request cam.CamV1PresignedUrlRequest) (cam.CamV1PresignedUrl, *http.Response, error) {
	resp := c.camClient.PresignedUrlsCamV1Api.PresignedUploadUrlCamV1PresignedUrl(c.camApiContext(ctx)).CamV1PresignedUrlRequest(request)
	return resp.Execute()
}

func executeConnectArtifactCreate(ctx context.Context, c *Client, artifact cam.CamV1ConnectArtifactSpec) (cam.CamV1ConnectArtifact, *http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.CreateCamV1ConnectArtifact(c.camApiContext(ctx)).SpecCloud(artifact.GetCloud()).SpecRegion(artifact.GetRegion()).CamV1ConnectArtifact(cam.CamV1ConnectArtifact{Spec: &artifact})
	return req.Execute()
}

func executeConnectArtifactRead(ctx context.Context, c *Client, region, cloud, artifactID, envId string) (cam.CamV1ConnectArtifact, *http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.GetCamV1ConnectArtifact(c.camApiContext(ctx), artifactID).
		SpecRegion(region).
		SpecCloud(cloud).
		Environment(envId)
	return req.Execute()
}

func connectArtifactRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	artifactId := d.Id()

	if _, err := readConnectArtifactAndSetAttributes(ctx, d, meta, d.Get(paramRegion).(string), d.Get(paramCloud).(string), artifactId, d.Get(paramArtifactFile).(string), extractStringValueFromBlock(d, paramEnvironment, paramId)); err != nil {
		return diag.FromErr(fmt.Errorf("error reading connect artifact %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readConnectArtifactAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, region, cloud, artifactId, artifactFile, envId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	artifact, resp, err := executeConnectArtifactRead(c.camApiContext(ctx), c, region, cloud, artifactId, envId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Connect Artifact %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Connect Artifact %q in TF state because Connect Artifact could not be found on the server", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	artifactJson, err := json.Marshal(artifact)
	if err != nil {
		return nil, fmt.Errorf("error reading connect artifact %q: error marshaling %#v to json: %s", artifactId, artifact, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Connect Artifact %q: %s", d.Id(), artifactJson), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	if _, err := setConnectArtifactAttributes(d, artifact, artifactFile); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setConnectArtifactAttributes(d *schema.ResourceData, artifact cam.CamV1ConnectArtifact, artifactFile string) (*schema.ResourceData, error) {
	spec := artifact.GetSpec()
	if err := d.Set(paramDisplayName, spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, spec.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, spec.GetRegion()); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, spec.GetEnvironment(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramContentFormat, spec.GetContentFormat()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, spec.GetDescription()); err != nil {
		return nil, err
	}
	if artifactFile != "" {
		if err := d.Set(paramArtifactFile, artifactFile); err != nil {
			return nil, err
		}
	}
	d.SetId(artifact.GetId())

	return d, nil
}

func connectArtifactDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	_, err := executeConnectArtifactDelete(c.camApiContext(ctx), c, d.Id(), d.Get(paramRegion).(string), d.Get(paramCloud).(string), environmentId)

	if err != nil {
		return diag.Errorf("error deleting connect artifact %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	return nil
}

func executeConnectArtifactDelete(ctx context.Context, c *Client, artifactID, region, cloud, envId string) (*http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.DeleteCamV1ConnectArtifact(c.camApiContext(ctx), artifactID).
		SpecRegion(region).
		SpecCloud(cloud).
		Environment(envId)
	return req.Execute()
}

func connectArtifactImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	regionCloudAndArtifactId := d.Id()
	parts := strings.Split(regionCloudAndArtifactId, "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("error importing connect artifact: invalid format: expected '<Environment ID>/<region>/<cloud>/<Connect Artifact ID>'")
	}

	artifactId := parts[3]
	region := parts[1]
	cloud := parts[2]
	envId := parts[0]
	artifactFile := getEnv("IMPORT_ARTIFACT_FILENAME", "")
	d.SetId(artifactId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readConnectArtifactAndSetAttributes(ctx, d, meta, region, cloud, artifactId, artifactFile, envId); err != nil {
		return nil, fmt.Errorf("error importing connect artifact %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
