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
	"path/filepath"
	"regexp"
	"strings"
)

var acceptedRuntimeLanguage = []string{"python", "java"}
var pattern = "^(([a-zA-Z][a-zA-Z_$0-9]*(\\.[a-zA-Z][a-zA-Z_$0-9]*)*)\\.)?([a-zA-Z][a-zA-Z_$0-9]*)$"

func artifactResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: artifactCreate,
		ReadContext:   artifactRead,
		DeleteContext: artifactDelete,
		Importer: &schema.ResourceImporter{
			StateContext: artifactImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The Unique name of the Flink Artifact per cloud, region, environment scope.",
				ValidateFunc: validation.StringLenBetween(1, 60),
			},
			paramClass: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Java class or alias for the Flink Artifact as provided by developer.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile(pattern), "The class must be in the required format"),
				Deprecated:   "No longer required.",
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Description:  "Cloud provider where the Flink Artifact archive is uploaded.",
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The Cloud provider region the Flink Artifact archive is uploaded.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramEnvironment: environmentSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Archive format of the Flink Artifact (JAR or ZIP).",
			},
			paramArtifactFile: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(val.(string)), "."))
					if extension != "zip" && extension != "jar" {
						errs = append(errs, fmt.Errorf("%q must be have extension .jar or .zip", key))
					}
					return
				},
				Description: "The artifact file for Flink Artifact in JAR or ZIP format.",
			},
			paramRuntimeLanguage: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(acceptedRuntimeLanguage, true),
				Default:      "JAVA",
				Description:  "Runtime language of the Flink Artifact as Python or Java. The default runtime language is Java.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the Flink Artifact.",
			},
			paramDocumentationLink: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Documentation link of the Flink Artifact.",
			},
			paramVersions: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of versions for this Flink Artifact.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramVersion: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The version of this Flink Artifact.",
						},
					},
				},
			},
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
	documentationLink := d.Get(paramDocumentationLink).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	request := fa.ArtifactV1PresignedUrlRequest{
		Cloud:       fa.PtrString(cloud),
		Region:      fa.PtrString(region),
		Environment: fa.PtrString(environmentId),
	}
	if contentFormat != "" {
		request.SetContentFormat(contentFormat)
	}

	resp, _, err := getFlinkPresignedUrl(c.faApiContext(ctx), c, request)
	if err != nil {
		return diag.Errorf("error uploading Flink Artifact: error fetching presigned upload URL %s", createDescriptiveError(err))
	}

	if err := uploadFile(resp.GetUploadUrl(), artifactFile, resp.GetUploadFormData()); err != nil {
		return diag.Errorf("error uploading Flink Artifact: %s", createDescriptiveError(err))
	}

	createArtifactRequest := fa.InlineObject{
		DisplayName: name,
		Cloud:       cloud,
		Region:      region,
		Class:       &class,
		Environment: environmentId,
		UploadSource: fa.InlineObjectUploadSourceOneOf{
			ArtifactV1UploadSourcePresignedUrl: &fa.ArtifactV1UploadSourcePresignedUrl{
				Location: fa.PtrString("PRESIGNED_URL_LOCATION"),
				UploadId: fa.PtrString(resp.GetUploadId()),
			},
		},
	}
	if description != "" {
		createArtifactRequest.SetDescription(description)
	}
	if documentationLink != "" {
		createArtifactRequest.SetDocumentationLink(documentationLink)
	}
	if runtimeLanguage != "" {
		createArtifactRequest.SetRuntimeLanguage(runtimeLanguage)
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
	req := c.faClient.FlinkArtifactsArtifactV1Api.CreateArtifactV1FlinkArtifact(c.faApiContext(ctx)).Region(artifact.GetRegion()).Cloud(artifact.GetCloud()).InlineObject(artifact)
	return req.Execute()
}

func executeArtifactRead(ctx context.Context, c *Client, region, cloud, artifactID, envId string) (fa.ArtifactV1FlinkArtifact, *http.Response, error) {
	req := c.faClient.FlinkArtifactsArtifactV1Api.GetArtifactV1FlinkArtifact(c.faApiContext(ctx), artifactID).Region(region).Cloud(cloud).Environment(envId)
	return req.Execute()
}

func artifactRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	artifactId := d.Id()

	if _, err := readArtifactAndSetAttributes(ctx, d, meta, d.Get(paramRegion).(string), d.Get(paramCloud).(string), artifactId, d.Get(paramArtifactFile).(string), extractStringValueFromBlock(d, paramEnvironment, paramId)); err != nil {
		return diag.FromErr(fmt.Errorf("error reading flink artifact %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readArtifactAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, region, cloud, artifactId, artifactFile, envId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	artifact, resp, err := executeArtifactRead(c.faApiContext(ctx), c, region, cloud, artifactId, envId)
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
		return nil, fmt.Errorf("error reading flink artifact %q: error marshaling %#v to json: %s", artifactId, artifact, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Artifact %q: %s", d.Id(), artifactJson), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	if _, err := setArtifactAttributes(d, artifact, artifactFile); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}
func getVersions(versionsStruct []fa.ArtifactV1FlinkArtifactVersion) []map[string]string {
	versions := []map[string]string{}
	for i := 0; i < len(versionsStruct); i++ {
		versions = append(versions, make(map[string]string))
		versions[i][paramVersion] = versionsStruct[i].GetVersion()
	}
	return versions
}
func setArtifactAttributes(d *schema.ResourceData, artifact fa.ArtifactV1FlinkArtifact, artifactFile string) (*schema.ResourceData, error) {
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

	if err := d.Set(paramVersions, getVersions(artifact.GetVersions())); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, artifact.GetEnvironment(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramContentFormat, artifact.GetContentFormat()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, artifact.GetDescription()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDocumentationLink, artifact.GetDocumentationLink()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRuntimeLanguage, artifact.GetRuntimeLanguage()); err != nil {
		return nil, err
	}
	if artifactFile != "" {
		if err := d.Set(paramArtifactFile, artifactFile); err != nil {
			return nil, err
		}
	}
	if err := d.Set(paramApiVersion, artifact.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, artifact.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(artifact.GetId())

	return d, nil
}

func artifactDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	c := meta.(*Client)
	req := c.faClient.FlinkArtifactsArtifactV1Api.DeleteArtifactV1FlinkArtifact(c.faApiContext(ctx), d.Id()).Region(d.Get(paramRegion).(string)).Cloud(d.Get(paramCloud).(string)).Environment(extractStringValueFromBlock(d, paramEnvironment, paramId))
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting flink artifact %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	return nil
}

func artifactImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})

	regionCloudAndArtifactId := d.Id()
	parts := strings.Split(regionCloudAndArtifactId, "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("error importing flink artifact: invalid format: expected '<Environment ID>/<region>/<cloud>/<Flink Artifact ID>'")
	}

	artifactId := parts[3]
	region := parts[1]
	cloud := parts[2]
	envId := parts[0]
	artifactFile := getEnv("IMPORT_ARTIFACT_FILENAME", "")
	d.SetId(artifactId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readArtifactAndSetAttributes(ctx, d, meta, region, cloud, artifactId, artifactFile, envId); err != nil {
		return nil, fmt.Errorf("error importing flink artifact %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Artifact %q", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
