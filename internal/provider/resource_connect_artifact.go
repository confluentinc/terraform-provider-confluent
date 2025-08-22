package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	cam "github.com/confluentinc/ccloud-sdk-go-v2/cam/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	stateProcessing           = "PROCESSING"
	stateWaitingForProcessing = "WAITING_FOR_PROCESSING"
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
				Description:  "The unique name of the Connect Artifact per cloud, environment scope.",
				ValidateFunc: validation.StringLenBetween(1, 60),
			},
			paramCloud: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Cloud provider where the Connect Artifact archive is uploaded.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				// Suppress the diff shown if the value of "cloud" attribute are equal when both compared in lower case.
				// For example, AWS == aws
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if strings.ToLower(old) == strings.ToLower(new) {
						return true
					}
					return false
				},
			},
			paramEnvironment: environmentSchema(),
			paramContentFormat: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Archive format of the Connect Artifact. Supported formats are JAR and ZIP.",
				ValidateFunc: validation.StringInSlice([]string{
					"JAR",
					"ZIP",
				}, false),
			},
			paramArtifactFile: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					extension := strings.ToLower(strings.TrimPrefix(filepath.Ext(val.(string)), "."))
					if extension != "jar" && extension != "zip" {
						errs = append(errs, fmt.Errorf("%q must have extension .jar or .zip", key))
					}
					return
				},
				Description: "The artifact file for Connect Artifact in JAR or ZIP format.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Description of the Connect Artifact.",
			},
			paramStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Connect Artifact.",
			},
		},
	}
}

func connectArtifactCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	name := d.Get(paramDisplayName).(string)
	cloud := d.Get(paramCloud).(string)
	contentFormat := d.Get(paramContentFormat).(string)
	artifactFile := d.Get(paramArtifactFile).(string)
	description := d.Get(paramDescription).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	// Step 1: Get presigned URL
	request := cam.CamV1PresignedUrlRequest{
		Cloud:         cam.PtrString(cloud),
		Environment:   cam.PtrString(environmentId),
		ContentFormat: cam.PtrString(contentFormat),
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

	// Wait for the Connect Artifact to be ready
	if err := waitForConnectArtifactToProvision(ctx, c, environmentId, d.Id(), cloud); err != nil {
		return diag.Errorf("error waiting for Connect Artifact to be ready: %s", createDescriptiveError(err))
	}

	return connectArtifactRead(ctx, d, meta)
}

func getConnectPresignedUrl(ctx context.Context, c *Client, request cam.CamV1PresignedUrlRequest) (cam.CamV1PresignedUrl, *http.Response, error) {
	resp := c.camClient.PresignedUrlsCamV1Api.PresignedUploadUrlCamV1PresignedUrl(c.camApiContext(ctx)).CamV1PresignedUrlRequest(request)
	return resp.Execute()
}

func executeConnectArtifactCreate(ctx context.Context, c *Client, artifact cam.CamV1ConnectArtifactSpec) (cam.CamV1ConnectArtifact, *http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.CreateCamV1ConnectArtifact(c.camApiContext(ctx)).
		SpecCloud(artifact.GetCloud()).
		Environment(artifact.GetEnvironment()).
		CamV1ConnectArtifact(cam.CamV1ConnectArtifact{Spec: &artifact})
	return req.Execute()
}

func executeConnectArtifactRead(ctx context.Context, c *Client, cloud, artifactID, envId string) (cam.CamV1ConnectArtifact, *http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.GetCamV1ConnectArtifact(c.camApiContext(ctx), artifactID).
		SpecCloud(cloud).
		Environment(envId)
	return req.Execute()
}

func connectArtifactRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	artifactId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading Connect Artifact: environment is required and must be specified")
	}

	if _, err := readConnectArtifactAndSetAttributes(ctx, d, meta, d.Get(paramCloud).(string), artifactId, d.Get(paramArtifactFile).(string), environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading connect artifact %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readConnectArtifactAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, cloud, artifactId, artifactFile, envId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	artifact, resp, err := executeConnectArtifactRead(c.camApiContext(ctx), c, cloud, artifactId, envId)
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

	// Set the status attribute if it exists in the schema
	if _, ok := d.GetOk(paramStatus); ok && artifact.Status != nil {
		if err := d.Set(paramStatus, artifact.Status.GetPhase()); err != nil {
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
	if environmentId == "" {
		return diag.Errorf("error deleting Connect Artifact: environment is required and must be specified")
	}
	_, err := executeConnectArtifactDelete(c.camApiContext(ctx), c, d.Id(), d.Get(paramCloud).(string), environmentId)

	if err != nil {
		return diag.Errorf("error deleting connect artifact %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	return nil
}

func executeConnectArtifactDelete(ctx context.Context, c *Client, artifactID, cloud, envId string) (*http.Response, error) {
	req := c.camClient.ConnectArtifactsCamV1Api.DeleteCamV1ConnectArtifact(c.camApiContext(ctx), artifactID).
		SpecCloud(cloud).
		Environment(envId)
	return req.Execute()
}

func connectArtifactImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})

	cloudAndArtifactId := d.Id()
	parts := strings.Split(cloudAndArtifactId, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing connect artifact: invalid format: expected '<Environment ID>/<cloud>/<Connect Artifact ID>'")
	}

	artifactId := parts[2]
	cloud := parts[1]
	envId := parts[0]
	artifactFile := getEnv("IMPORT_ARTIFACT_FILENAME", "")
	d.SetId(artifactId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readConnectArtifactAndSetAttributes(ctx, d, meta, cloud, artifactId, artifactFile, envId); err != nil {
		return nil, fmt.Errorf("error importing connect artifact %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Connect Artifact %q", d.Id()), map[string]interface{}{connectArtifactLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func waitForConnectArtifactToProvision(ctx context.Context, c *Client, environmentId, artifactId, cloud string) error {
	delay, pollInterval := getDelayAndPollInterval(5*time.Second, 1*time.Minute, c.isAcceptanceTestMode)
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateProvisioning, stateProcessing},
		Target:       []string{stateProvisioned, stateReady},
		Refresh:      connectArtifactProvisionStatus(c.camApiContext(ctx), c, environmentId, artifactId, cloud),
		Timeout:      1 * time.Hour,
		Delay:        delay,
		PollInterval: pollInterval,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Connect Artifact %q provisioning status to become %q", artifactId, stateProvisioned), map[string]interface{}{connectArtifactLoggingKey: artifactId})
	if _, err := stateConf.WaitForStateContext(c.camApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func connectArtifactProvisionStatus(ctx context.Context, c *Client, environmentId, artifactId, cloud string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		artifact, _, err := executeConnectArtifactRead(ctx, c, cloud, artifactId, environmentId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Connect Artifact %q: %s", artifactId, createDescriptiveError(err)), map[string]interface{}{connectArtifactLoggingKey: artifactId})
			return nil, stateUnknown, err
		}

		// Check if the artifact has a status field
		if artifact.Status == nil {
			// If no status field, assume it's provisioned
			return artifact, stateProvisioned, nil
		}

		phase := artifact.Status.GetPhase()
		tflog.Debug(ctx, fmt.Sprintf("Waiting for Connect Artifact %q provisioning status to become %q: current status is %q", artifactId, stateProvisioned, phase), map[string]interface{}{connectArtifactLoggingKey: artifactId})

		if phase == stateProcessing || phase == stateProvisioning || phase == stateProvisioned || phase == stateReady {
			return artifact, phase, nil
		} else if phase == stateWaitingForProcessing || phase == stateFailed {
			return nil, phase, fmt.Errorf("connect artifact %q provisioning status is %q", artifactId, phase)
		}
		// Connect Artifact is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("connect artifact %q is in an unexpected state %q", artifactId, phase)
	}
}
