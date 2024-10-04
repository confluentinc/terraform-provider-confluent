package provider

import (
	"context"
	"encoding/json"
	"fmt"
	fa "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

var acceptedRuntimeLanguage = []string{"python", "java"}

func artifactResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: artifactCreate,
		ReadContext:   artifactRead,
		UpdateContext: artifactUpdate,
		DeleteContext: artifactDelete,
		Importer: &schema.ResourceImporter{
			StateContext: artifactImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID for flink artifact",
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The display name of flink artifact",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramClass: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The class for flink artifact",
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Description:  "The public cloud flink artifact name",
				// Suppress the diff shown if the value of "cloud" attribute are equal when both compared in lower case.
				// For example, AWS == aws
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.ToLower(old) == strings.ToLower(new) {
						return true
					}
					return false
				},
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider region that hosts the Flink artifact.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramEnvironment: environmentSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The content format for flink artifact",
			},
			paramArtifactFile: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The artifact file for flink artifact",
			},
			paramRuntimeLanguage: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(acceptedRuntimeLanguage, true),
				Description:  "The runtime language for flink artifact",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form description of the artifact.",
			},
		},
	}
}

func artifactCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	name := d.Get(paramDisplayName).(string)
	class := d.Get(paramClass).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	contentFormat := d.Get(paramContentFormat).(string)
	artifactFile := d.Get(paramArtifactFile).(string)
	runtimeLanguage := d.Get(paramRuntimeLanguage).(string)
	description := d.Get(paramDescription).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	request := fa.ArtifactV1PresignedUrlRequest{
		ContentFormat: fa.PtrString(contentFormat),
		Cloud:         fa.PtrString(cloud),
		Region:        fa.PtrString(region),
	}
	resp, _, err := getFlinkPresignedUrl(c.faApiContext(ctx), c, request)
	if err != nil {
		return diag.Errorf("error creating Flink Artifact: %s", createDescriptiveError(err))
	}

	if err := uploadFile(resp.GetUploadUrl(), artifactFile, resp.GetUploadFormData()); err != nil {
		return diag.Errorf("error uploading Flink Artifact: %s", createDescriptiveError(err))
	}

	createArtifactRequest := fa.InlineObject{
		DisplayName: name,
		Cloud:       cloud,
		Region:      region,
		Environment: environmentId,
		Class:       class,
		Description: fa.PtrString(description),
		UploadSource: fa.InlineObjectUploadSourceOneOf{
			ArtifactV1UploadSourcePresignedUrl: &fa.ArtifactV1UploadSourcePresignedUrl{
				Location: fa.PtrString("PRESIGNED_URL_LOCATION"),
				UploadId: fa.PtrString(resp.GetUploadId()),
			},
		},
		RuntimeLanguage: fa.PtrString(runtimeLanguage),
	}

	createArtifactRequestJson, err := json.Marshal(createArtifactRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Artifact: error marshaling %#v to json: %s", createArtifactRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Flink Artifact: %s", createArtifactRequestJson))

	createdArtifact, _, err := executeArtifactCreate(c.faApiContext(ctx), c, createArtifactRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Artifact %q: %s", createdArtifact.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdArtifact.GetId())

	createdArtifactJson, err := json.Marshal(createdArtifact)

	if err != nil {
		return diag.Errorf("error creating Flink Artifact: error marshaling %#v to json: %s", createdArtifactJson, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Artifact %q: %s", d.Id(), createdArtifactJson), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	return artifactRead(ctx, d, meta)
}

func getFlinkPresignedUrl(ctx context.Context, c *Client, request fa.ArtifactV1PresignedUrlRequest) (fa.ArtifactV1PresignedUrl, *http.Response, error) {
	resp := c.faClient.PresignedUrlsArtifactV1Api.PresignedUploadUrlArtifactV1PresignedUrl(c.faApiContext(ctx)).ArtifactV1PresignedUrlRequest(request)
	return resp.Execute()
}

func executeArtifactCreate(ctx context.Context, c *Client, artifact fa.InlineObject) (fa.ArtifactV1FlinkArtifact, *http.Response, error) {
	req := c.faClient.FlinkArtifactsArtifactV1Api.CreateArtifactV1FlinkArtifact(c.faApiContext(ctx)).Region(artifact.GetRegion()).Cloud(artifact.GetCloud())
	return req.Execute()
}

func executeArtifactRead(ctx context.Context, c *Client, region, cloud, artifactID string) (fa.ArtifactV1FlinkArtifact, *http.Response, error) {
	req := c.faClient.FlinkArtifactsArtifactV1Api.GetArtifactV1FlinkArtifact(c.faApiContext(ctx), artifactID).Region(region).Cloud(cloud)
	return req.Execute()
}

func artifactRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	artifactId := d.Id()

	if _, err := readArtifactAndSetAttributes(ctx, d, meta, d.Get(paramRegion).(string), d.Get(paramCloud).(string), artifactId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Flink Artifact %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readArtifactAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, region, cloud, artifactId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	artifact, resp, err := executeArtifactRead(c.faApiContext(ctx), c, region, cloud, artifactId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Artifact %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Artifact %q in TF state because Flink Artifact could not be found on the server", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	artifactJson, err := json.Marshal(artifact)
	if err != nil {
		return nil, fmt.Errorf("error reading Flink Artifact %q: error marshaling %#v to json: %s", artifactId, artifact, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Artifact %q: %s", d.Id(), artifactJson), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	if _, err := setArtifactAttributes(d, artifact); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setArtifactAttributes(d *schema.ResourceData, artifact fa.ArtifactV1FlinkArtifact) (*schema.ResourceData, error) {
	if err := d.Set(paramId, artifact.GetId()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDisplayName, artifact.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramClass, artifact.GetClass()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, artifact.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, artifact.GetRegion()); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, artifact.GetEnvironment(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramContentFormat, artifact.GetContentFormat()); err != nil {
		return nil, err
	}
	d.SetId(artifact.GetId())
	return d, nil
}

func artifactUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName) {
		return diag.Errorf("error updating Flink Artifact %q: only %q attribute can be updated for Flink Artifact", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateArtifactRequest := fa.NewArtifactV1FlinkArtifactUpdate()
	updateArtifactRequest.SetEnvironment(environmentId)

	if d.HasChange(paramDisplayName) {
		updateArtifactRequest.SetDisplayName(d.Get(paramDisplayName).(string))
	}

	updateArtifactRequestJson, err := json.Marshal(updateArtifactRequest)
	if err != nil {
		return diag.Errorf("error updating Flink Artifact %q: error marshaling %#v to json: %s", d.Id(), updateArtifactRequestJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Artifact %q: %s", d.Id(), updateArtifactRequestJson), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	req := c.faClient.FlinkArtifactsArtifactV1Api.UpdateArtifactV1FlinkArtifact(c.faApiContext(ctx), d.Id()).ArtifactV1FlinkArtifactUpdate(*updateArtifactRequest).Region(d.Get(paramRegion).(string)).Cloud(d.Get(paramCloud).(string))
	updatedArtifact, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Flink Artifact %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedArtifactJson, err := json.Marshal(updatedArtifact)
	if err != nil {
		return diag.Errorf("error updating Flink Artifact %q: error marshaling %#v to json: %s", d.Id(), updatedArtifact, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Flink Artifact %q: %s", d.Id(), updatedArtifactJson), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	return artifactRead(ctx, d, meta)
}

func artifactDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink atrifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	c := meta.(*Client)
	req := c.faClient.FlinkArtifactsArtifactV1Api.DeleteArtifactV1FlinkArtifact(c.faApiContext(ctx), d.Id()).Region(d.Get(paramRegion).(string)).Cloud(d.Get(paramCloud).(string))
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Flink artifact %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	return nil
}

func artifactImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	regionCloudAndArtifactId := d.Id()
	parts := strings.Split(regionCloudAndArtifactId, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Flink Artifact: invalid format: expected '<region>/<cloud>/<Flink Artifact ID>'")
	}

	artifactId := parts[2]
	region := parts[0]
	cloud := parts[1]
	d.SetId(artifactId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readArtifactAndSetAttributes(ctx, d, meta, region, cloud, artifactId); err != nil {
		return nil, fmt.Errorf("error importing Flink Artifact %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
