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
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	sr "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramSchemaRegistryCluster               = "schema_registry_cluster"
	schemaRegistryAPIWaitAfterCreateOrDelete = 10 * time.Second
	paramFormat                              = "format"
	avroFormat                               = "AVRO"
	jsonFormat                               = "JSON"
	protobufFormat                           = "PROTOBUF"
	paramVersion                             = "version"
	// unique on a subject level
	paramSchemaIdentifier             = "schema_identifier"
	paramSchema                       = "schema"
	paramSchemaReference              = "schema_reference"
	paramSubjectName                  = "subject_name"
	paramHardDelete                   = "hard_delete"
	paramHardDeleteDefaultValue       = false
	paramRecreateOnUpdate             = "recreate_on_update"
	paramRecreateOnUpdateDefaultValue = false

	latestSchemaVersionAndPlaceholderForSchemaIdentifier = "latest"
)

var acceptedSchemaFormats = []string{avroFormat, jsonFormat, protobufFormat}

const schemaNotCompatibleErrorMessage = `Compatibility check on the schema has failed against one or more versions in the subject, depending on how the compatibility is set.
See https://docs.confluent.io/platform/current/schema-registry/avro.html#sr-compatibility-types for details.
For example, if compatibility on the subject is set to BACKWARD, FORWARD, or FULL, the compatibility check is against the latest version.
If compatibility is set to one of the TRANSITIVE types, the check is against all previous versions.`

func schemaResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: schemaCreate,
		ReadContext:   schemaRead,
		UpdateContext: schemaUpdate,
		DeleteContext: schemaDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schemaImport,
		},
		Schema: map[string]*schema.Schema{
			// ID = lsrc-123/subjectName/schemaName/schema_identifier
			paramSchemaRegistryCluster: schemaRegistryClusterBlockSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramSubjectName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the Schema Registry Subject.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFormat: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The format of the Schema.",
				ValidateFunc: validation.StringInSlice(acceptedSchemaFormats, false),
			},
			paramSchema: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The definition of the Schema.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramVersion: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version number of the Schema.",
			},
			paramSchemaIdentifier: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Globally unique identifier of the Schema returned for a creation request. It should be used to retrieve this schema from the schemas resource and is different from the schema’s version which is associated with the subject.",
			},
			paramSchemaReference: {
				Description: "The list of references to other Schemas.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramSubjectName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The name of the referenced Schema Registry Subject (for example, \"User\").",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramName: {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The name of the Schema references (for example, \"io.confluent.kafka.example.User\"). For Avro, the reference name is the fully qualified schema name, for JSON Schema it is a URL, and for Protobuf, it is the name of another Protobuf file.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramVersion: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "The version of the referenced Schema.",
						},
					},
				},
			},
			paramHardDelete: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     paramHardDeleteDefaultValue,
				Description: "Controls whether a schema should be soft or hard deleted. Set it to `true` if you want to hard delete a schema on destroy. Defaults to `false` (soft delete).",
			},
			paramRecreateOnUpdate: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     paramRecreateOnUpdateDefaultValue,
				Description: "Controls whether a schema should be recreated on update.",
			},
		},
		CustomizeDiff: customdiff.Sequence(SetSchemaDiff),
	}
}

func SetSchemaDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if !diff.HasChange(paramSchema) {
		return nil
	}

	oldObj, newObj := diff.GetChange(paramSchema)
	oldSchema := oldObj.(string)
	newSchema := newObj.(string)

	client := meta.(*Client)

	var restEndpoint, clusterId, clusterApiKey, clusterApiSecret string

	// We could have used interfaces here to reuse code from schemaCreate()
	// but it's probably a safer approach to duplicate code here since debug messages are different / no imports either.
	if client.isSchemaRegistryMetadataSet {
		restEndpoint = client.schemaRegistryRestEndpoint
		clusterId = client.schemaRegistryClusterId
		clusterApiKey = client.schemaRegistryApiKey
		clusterApiSecret = client.schemaRegistryApiSecret
	} else {
		restEndpoint = diff.Get(paramRestEndpoint).(string)
		clusterId = diff.Get(fmt.Sprintf("%s.0.%s", paramSchemaRegistryCluster, paramId)).(string)
		clusterApiKey = diff.Get(fmt.Sprintf("%s.0.%s", paramCredentials, paramKey)).(string)
		clusterApiSecret = diff.Get(fmt.Sprintf("%s.0.%s", paramCredentials, paramSecret)).(string)
	}

	if restEndpoint == "" || clusterId == "" || clusterApiKey == "" || clusterApiSecret == "" {
		// Skip checks since these attributes reference other resources attributes that are unknown before "terraform apply"
		return nil
	}

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	subjectName := diff.Get(paramSubjectName).(string)
	format := diff.Get(paramFormat).(string)
	schemaContent := newSchema
	schemaReferences := buildSchemaReferences(diff.Get(paramSchemaReference).(*schema.Set).List())

	createSchemaRequest := sr.NewRegisterSchemaRequest()
	createSchemaRequest.SetSchemaType(format)
	createSchemaRequest.SetSchema(schemaContent)
	createSchemaRequest.SetReferences(schemaReferences)

	err := schemaValidateCheck(ctx, schemaRegistryRestClient, createSchemaRequest, subjectName)
	if err != nil {
		return err
	}

	// Skip a schema lookup check if the schema doesn't exist yet
	if diff.Id() == "" {
		return nil
	}

	// Return an error for a schema update when recreate_on_update=true
	// User wants to edit / evolve a schema. See https://docs.confluent.io/cloud/current/sr/schemas-manage.html#editing-schemas for more details.
	// This is a fix for https://github.com/confluentinc/terraform-provider-confluent/issues/235
	shouldRecreateOnUpdate := diff.Get(paramRecreateOnUpdate).(bool)
	if shouldRecreateOnUpdate {
		return fmt.Errorf("error updating Schema %q: reimport the current resource instance and set %s = false to evolve a schema using the same resource instance.\nIn this case, on an update resource instance will reference the updated (latest) schema by overriding %s, %s and %s attributes and the old schema will be orphaned.", diff.Id(), paramRecreateOnUpdate, paramSchemaIdentifier, paramSchema, paramVersion)
	}

	return schemaLookupCheck(ctx, diff, schemaRegistryRestClient, createSchemaRequest, subjectName, oldSchema)
}

func schemaLookupCheck(ctx context.Context, diff *schema.ResourceDiff, c *SchemaRegistryRestClient, createSchemaRequest *sr.RegisterSchemaRequest, subjectName, oldSchema string) error {
	createSchemaRequestJson, err := json.Marshal(createSchemaRequest)
	if err != nil {
		return fmt.Errorf("error customizing diff Schema: error marshaling %#v to json: %s", createSchemaRequest, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Customizing diff new Schema: %s", createSchemaRequestJson))

	registeredSchema, schemaExists, err := schemaLookup(ctx, c, createSchemaRequest, subjectName)
	if err != nil {
		return fmt.Errorf("error customizing diff Schema: %s", createDescriptiveError(err))
	}
	if !schemaExists {
		return nil
	}

	schemaIdentifier := diff.Get(paramSchemaIdentifier).(int)
	if int(registeredSchema.GetId()) == schemaIdentifier {
		// Two schemas that are semantically equivalent
		// https://docs.confluent.io/platform/current/schema-registry/serdes-develop/index.html#schema-normalization

		// Set old value to paramSchema to avoid TF drift
		if err := diff.SetNew(paramSchema, oldSchema); err != nil {
			return fmt.Errorf("error customizing diff Schema: %s", createDescriptiveError(err))
		}
	}
	return nil
}

func schemaValidateCheck(ctx context.Context, c *SchemaRegistryRestClient, createSchemaRequest *sr.RegisterSchemaRequest, subjectName string) error {
	createSchemaRequestJson, err := json.Marshal(createSchemaRequest)
	if err != nil {
		return fmt.Errorf("error validating Schema: error marshaling %#v to json: %s", createSchemaRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Validating new Schema: %s", createSchemaRequestJson))
	validationResponse, _, err := executeSchemaValidate(ctx, c, createSchemaRequest, subjectName)
	if err != nil {
		return fmt.Errorf("error validating Schema: error sending validation request: %s", createDescriptiveError(err))
	}
	// Validation has failed
	if !validationResponse.GetIsCompatible() {
		if len(validationResponse.GetMessages()) > 0 {
			return fmt.Errorf("error validating Schema: error validating a schema: %v", validationResponse.GetMessages())
		}
		return fmt.Errorf("error validating Schema: error validating a schema: %s", schemaNotCompatibleErrorMessage)
	}
	return nil
}

func schemaLookup(ctx context.Context, c *SchemaRegistryRestClient, createSchemaRequest *sr.RegisterSchemaRequest, subjectName string) (*sr.Schema, bool, error) {
	// https://github.com/confluentinc/terraform-provider-confluent/issues/196#issuecomment-1426437871
	// Try both normalize=false and normalize=true
	nonNormalizedSchema, schemaExists, err := schemaLookupByNormalize(ctx, c, createSchemaRequest, subjectName, false)
	if err != nil {
		return nil, false, fmt.Errorf("error looking up Schema: %s", createDescriptiveError(err))
	}
	if schemaExists {
		return nonNormalizedSchema, schemaExists, nil
	}
	normalizedSchema, schemaExists, err := schemaLookupByNormalize(ctx, c, createSchemaRequest, subjectName, true)
	if err != nil {
		return nil, false, fmt.Errorf("error looking up Schema: %s", createDescriptiveError(err))
	}
	if schemaExists {
		return normalizedSchema, schemaExists, nil
	}
	// Requested schema doesn't exist
	return nil, false, nil
}

func schemaLookupByNormalize(ctx context.Context, c *SchemaRegistryRestClient, createSchemaRequest *sr.RegisterSchemaRequest, subjectName string, shouldNormalize bool) (*sr.Schema, bool, error) {
	srSchema, resp, err := executeSchemaLookup(ctx, c, createSchemaRequest, subjectName, shouldNormalize)

	if resp != nil {
		if http.StatusNotFound == resp.StatusCode {
			// Requested schema doesn't exist
			return nil, false, nil
		} else if http.StatusUnprocessableEntity == resp.StatusCode {
			// TF Provider shouldn't fail
			tflog.Warn(ctx, fmt.Sprintf("Warning looking up Schema %#v: 422 Unprocessable Entity", createSchemaRequest))
			return nil, false, nil
		}
	}

	if err != nil {
		return nil, false, createDescriptiveError(err)
	}

	return &srSchema, true, nil
}

func schemaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)
	format := d.Get(paramFormat).(string)
	schemaContent := d.Get(paramSchema).(string)
	schemaReferences := buildSchemaReferences(d.Get(paramSchemaReference).(*schema.Set).List())

	createSchemaRequest := sr.NewRegisterSchemaRequest()
	createSchemaRequest.SetSchemaType(format)
	createSchemaRequest.SetSchema(schemaContent)
	createSchemaRequest.SetReferences(schemaReferences)
	createSchemaRequestJson, err := json.Marshal(createSchemaRequest)
	if err != nil {
		return diag.Errorf("error creating Schema: error marshaling %#v to json: %s", createSchemaRequest, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Validating new Schema: %s", createSchemaRequestJson))
	validationResponse, _, err := executeSchemaValidate(ctx, schemaRegistryRestClient, createSchemaRequest, subjectName)
	if err != nil {
		return diag.Errorf("error creating Schema: error sending validation request: %s", createDescriptiveError(err))
	}
	// Validation has failed
	if !validationResponse.GetIsCompatible() {
		// Set old value to paramSchema to avoid TF drift
		// It will be applicable if schemaCreate() is called from schemaUpdate()
		// since d.SetId() is called in the end of schemaCreate()
		oldObj, _ := d.GetChange(paramSchema)
		oldSchema := oldObj.(string)
		d.Set(paramSchema, oldSchema)

		if len(validationResponse.GetMessages()) > 0 {
			return diag.Errorf("error creating Schema: error validating a schema: %v", validationResponse.GetMessages())
		}
		return diag.Errorf("error creating Schema: error validating a schema: %s", schemaNotCompatibleErrorMessage)
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema: %s", createSchemaRequestJson))

	registeredSchema, _, err := executeSchemaCreate(ctx, schemaRegistryRestClient, createSchemaRequest, subjectName)

	if err != nil {
		return diag.Errorf("error creating Schema: %s", createDescriptiveError(err))
	}

	// Save the schema content
	if err := d.Set(paramSchema, schemaContent); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	schemaId := createSchemaId(schemaRegistryRestClient.clusterId, subjectName, registeredSchema.GetId(), d.Get(paramRecreateOnUpdate).(bool))
	d.SetId(schemaId)

	// https://github.com/confluentinc/terraform-provider-confluent/issues/40#issuecomment-1048782379
	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	return schemaRead(ctx, d, meta)
}

func schemaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Remove subject when deleting the last schema
	deletionType := "soft"
	isHardDeleteEnabled := d.Get(paramHardDelete).(bool)
	if isHardDeleteEnabled {
		deletionType = "hard"
	}

	tflog.Debug(ctx, fmt.Sprintf("%s deleting Schema %q", deletionType, d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error %s deleting Schema: %s", deletionType, createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error %s deleting Schema: %s", deletionType, createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error %s deleting Schema: %s", deletionType, createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)
	schemaVersion := d.Get(paramVersion).(int)

	// Both soft and hard delete requires a user to run a soft delete first
	err = executeSchemaDelete(schemaRegistryRestClient.apiContext(ctx), schemaRegistryRestClient, subjectName, strconv.Itoa(schemaVersion), false)

	if err != nil {
		return diag.Errorf("error %s deleting Schema %q: %s", deletionType, d.Id(), createDescriptiveError(err))
	}

	if isHardDeleteEnabled {
		err = executeSchemaDelete(schemaRegistryRestClient.apiContext(ctx), schemaRegistryRestClient, subjectName, strconv.Itoa(schemaVersion), true)

		if err != nil {
			return diag.Errorf("error %s deleting Schema %q: %s", deletionType, d.Id(), createDescriptiveError(err))
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished %s deleting Schema %q", deletionType, d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	return nil
}

func schemaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	schemaIdentifier, err := extractSchemaIdentifierFromTfId(d.Id())
	if err != nil {
		return diag.Errorf("error reading Schema %q: %s", d.Id(), createDescriptiveError(err))
	}
	subjectName, err := extractSubjectNameFromTfId(d.Id())
	if err != nil {
		return diag.Errorf("error reading Schema %q: %s", d.Id(), createDescriptiveError(err))
	}

	_, err = readSchemaRegistryConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName, schemaIdentifier)
	if err != nil {
		return diag.Errorf("error reading Schema: %s", createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	return nil
}

func schemaUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramCredentials, paramConfigs, paramHardDelete, paramSchema, paramSchemaReference) {
		return diag.Errorf("error updating Schema %q: only %q, %q, %q, %q and %q blocks can be updated for Schema", d.Id(), paramCredentials, paramConfigs, paramHardDelete, paramSchema, paramSchemaReference)
	}

	if d.HasChanges(paramSchema, paramSchemaReference) {
		oldSchema, _ := d.GetChange(paramSchema)
		oldSchemaReference, _ := d.GetChange(paramSchemaReference)

		// User wants to edit / evolve a schema. See https://docs.confluent.io/cloud/current/sr/schemas-manage.html#editing-schemas for more details.
		shouldRecreateOnUpdate := d.Get(paramRecreateOnUpdate).(bool)
		if shouldRecreateOnUpdate {
			// At this point new schema and schema_reference is saved to TF file,
			// so we need to revert it to the old value to avoid TF drift.
			if err := d.Set(paramSchema, oldSchema); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			if err := d.Set(paramSchemaReference, oldSchemaReference); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return diag.Errorf("error updating Schema %q: reimport the current resource instance and set %s = false to evolve a schema using the same resource instance.\nIn this case, on an update resource instance will reference the updated (latest) schema by overriding %s, %s and %s attributes and the old schema will be orphaned.", d.Id(), paramRecreateOnUpdate, paramSchemaIdentifier, paramSchema, paramVersion)
		}
		// Create a new schema and make existing resource instance point to it.
		return schemaCreate(ctx, d, meta)
	}

	return schemaRead(ctx, d, meta)
}

func createSchemaId(clusterId, subjectName string, identifier int32, shouldRecreateOnUpdate bool) string {
	if !shouldRecreateOnUpdate {
		// https://docs.confluent.io/platform/current/schema-registry/develop/api.html#get--subjects-(string-%20subject)-versions-(versionId-%20version)
		return fmt.Sprintf("%s/%s/latest", clusterId, subjectName)
	} else {
		return fmt.Sprintf("%s/%s/%d", clusterId, subjectName, identifier)
	}
}

func extractSchemaIdentifierFromTfId(terraformId string) (string, error) {
	_, _, identifier, err := extractSubjectInfoFromTfId(terraformId)

	if err != nil {
		return "", fmt.Errorf("error extracting Schema Identifier from Resource ID: %s", err.Error())
	}

	return identifier, nil
}

func extractSubjectNameFromTfId(terraformId string) (string, error) {
	_, name, _, err := extractSubjectInfoFromTfId(terraformId)

	if err != nil {
		return "", fmt.Errorf("error extracting Subject Name from Resource ID: %s", err.Error())
	}

	return name, nil
}

func extractSubjectInfoFromTfId(terraformId string) (string, string, string, error) {
	parts := strings.Split(terraformId, "/")
	length := len(parts)

	if length < 3 {
		return "", "", "", fmt.Errorf("invalid format: expected '<Schema Registry cluster ID>/<subject name>/<schema identifier>'")
	}

	clusterId := parts[0]
	identifier := parts[length-1]
	name := strings.Join(parts[1:length-1], "/")

	return clusterId, name, identifier, nil
}

func schemaImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema: %s", createDescriptiveError(err))
	}

	clusterIDAndSubjectNameAndSchemaIdentifier := d.Id()
	clusterId, subjectName, schemaIdentifier, err := extractSubjectInfoFromTfId(clusterIDAndSubjectNameAndSchemaIdentifier)

	if err != nil {
		return nil, fmt.Errorf("error importing Schema: %s", err.Error())
	}

	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	schemaSr, err := readSchemaRegistryConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName, schemaIdentifier)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema %q: %s", d.Id(), createDescriptiveError(err))
	}
	schemaContent := os.Getenv("SCHEMA_CONTENT")
	if schemaContent == "" {
		schemaContent = schemaSr.GetSchema()
	}
	if err := d.Set(paramSchema, schemaContent); err != nil {
		return nil, createDescriptiveError(err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func loadIdForLatestSchema(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string) (string, error) {
	latestSchema, _, err := c.apiClient.SubjectVersionsV1Api.GetSchemaByVersion(c.apiContext(ctx), subjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier).Execute()
	if err != nil {
		return "", fmt.Errorf("error loading the latest Schema: %s", createDescriptiveError(err))
	}
	return strconv.Itoa(int(latestSchema.GetId())), nil
}

func isLatestSchema(schemaIdentifier string) bool {
	return schemaIdentifier == latestSchemaVersionAndPlaceholderForSchemaIdentifier
}

func loadSchema(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string, schemaIdentifier string) (*sr.Schema, bool, error) {
	// Option #1: find the schema identifier of the latest schema
	var err error
	if isLatestSchema(schemaIdentifier) {
		schemaIdentifier, err = loadIdForLatestSchema(ctx, d, c, subjectName)
		if err != nil {
			return nil, false, fmt.Errorf("error loading the latest Schema: %s", createDescriptiveError(err))
		}
	}

	// Option #2: load schemas and filter by schemaIdentifier
	// Load a schema by ID (schemaIdentifier is not -1 anymore)
	// Sample Response (it doesn't include soft deleted schemas):
	//  [{"subject": "test2", "version": 5, "id": 100004, "schema": "{\"type\":\"record\",...}]}"},
	//   {"subject": "test2", "version": 6, "id": 100006, "schema": "{\"type\":\"record\",...}]}"}]
	// Search for all subjects in all contexts
	schemas, _, err := c.apiClient.SchemasV1Api.GetSchemas(c.apiContext(ctx)).SubjectPrefix(":*:").Execute()
	if err != nil {
		return nil, false, fmt.Errorf("error loading Schemas: %s", createDescriptiveError(err))
	}
	schemasJson, err := json.Marshal(schemas)
	if err != nil {
		return nil, false, fmt.Errorf("error marshaling %#v to json: %s", schemas, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schemas %q: %s", d.Id(), schemasJson), map[string]interface{}{schemaLoggingKey: d.Id()})
	srSchema, exists := findSchemaById(schemas, schemaIdentifier, subjectName)
	return &srSchema, exists, nil
}

func readSchemaRegistryConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string, schemaIdentifier string) (*sr.Schema, error) {
	isLatestSchemaBool := isLatestSchema(schemaIdentifier)
	srSchema, exists, err := loadSchema(ctx, d, c, subjectName, schemaIdentifier)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema %q: %s", d.Id(), createDescriptiveError(err))
	}
	if !exists {
		if !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema %q in TF state because Schema could not be found on the server", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		} else {
			return nil, fmt.Errorf("error reading Schema %q: Schema could not be found on the server", d.Id())
		}
	}
	schemaJson, err := json.Marshal(srSchema)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema %q: error marshaling %#v to json: %s", d.Id(), srSchema, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema %q: %s", d.Id(), schemaJson), map[string]interface{}{schemaLoggingKey: d.Id()})

	if err := d.Set(paramSubjectName, srSchema.GetSubject()); err != nil {
		return nil, err
	}
	// The schema format: AVRO is the default (if no schema type is shown on the response, the type is AVRO), PROTOBUF, JSONSCHEMA
	if srSchema.GetSchemaType() == "" {
		srSchema.SetSchemaType(avroFormat)
	}
	if err := d.Set(paramFormat, srSchema.GetSchemaType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramVersion, srSchema.GetVersion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSchemaIdentifier, srSchema.GetId()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSchemaReference, buildTfSchemaReferences(srSchema.GetReferences())); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
			return nil, err
		}
	}

	// Explicitly set paramHardDelete to the default value if unset
	if _, ok := d.GetOk(paramHardDelete); !ok {
		if err := d.Set(paramHardDelete, paramHardDeleteDefaultValue); err != nil {
			return nil, createDescriptiveError(err)
		}
	}

	// Explicitly set paramRecreateOnUpdate to the default value if unset
	if _, ok := d.GetOk(paramRecreateOnUpdate); !ok {
		if err := d.Set(paramRecreateOnUpdate, !isLatestSchemaBool); err != nil {
			return nil, createDescriptiveError(err)
		}
	}

	d.SetId(createSchemaId(c.clusterId, srSchema.GetSubject(), srSchema.GetId(), d.Get(paramRecreateOnUpdate).(bool)))

	return srSchema, nil
}

func findSchemaById(schemas []sr.Schema, schemaIdentifier string, subjectName string) (sr.Schema, bool) {
	// 'schema' collides with a package name
	for _, srSchema := range schemas {
		if strconv.Itoa(int(srSchema.GetId())) == schemaIdentifier && srSchema.GetSubject() == subjectName {
			return srSchema, true
		}
	}
	return sr.Schema{}, false
}

func executeSchemaValidate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.RegisterSchemaRequest, subjectName string) (sr.CompatibilityCheckResponse, *http.Response, error) {
	return c.apiClient.CompatibilityV1Api.TestCompatibilityForSubject(c.apiContext(ctx), subjectName).RegisterSchemaRequest(*requestData).Verbose(true).Execute()
}

func executeSchemaLookup(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.RegisterSchemaRequest, subjectName string, shouldNormalize bool) (sr.Schema, *http.Response, error) {
	return c.apiClient.SubjectsV1Api.LookUpSchemaUnderSubject(c.apiContext(ctx), subjectName).RegisterSchemaRequest(*requestData).Normalize(shouldNormalize).Execute()
}

func executeSchemaCreate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.RegisterSchemaRequest, subjectName string) (sr.RegisterSchemaResponse, *http.Response, error) {
	return c.apiClient.SubjectVersionsV1Api.Register(c.apiContext(ctx), subjectName).RegisterSchemaRequest(*requestData).Execute()
}

func executeSchemaDelete(ctx context.Context, c *SchemaRegistryRestClient, subjectName, schemaVersion string, isHardDelete bool) error {
	_, _, err := c.apiClient.SubjectVersionsV1Api.DeleteSchemaVersion(c.apiContext(ctx), subjectName, schemaVersion).Permanent(isHardDelete).Execute()
	if err != nil {
		if isHardDelete {
			return fmt.Errorf("error hard deleting Schema: %s", createDescriptiveError(err))
		} else {
			return fmt.Errorf("error soft deleting Schema: %s", createDescriptiveError(err))
		}
	} else {
		time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)
		return nil
	}
}

func buildSchemaReferences(tfReferences []interface{}) []sr.SchemaReference {
	references := make([]sr.SchemaReference, len(tfReferences))
	for index, tfReference := range tfReferences {
		reference := sr.NewSchemaReference()
		tfReferenceMap := tfReference.(map[string]interface{})
		if subjectName, exists := tfReferenceMap[paramSubjectName].(string); exists {
			reference.SetSubject(subjectName)
		}
		if referenceName, exists := tfReferenceMap[paramName].(string); exists {
			reference.SetName(referenceName)
		}
		if version, exists := tfReferenceMap[paramVersion].(int); exists {
			reference.SetVersion(int32(version))
		}
		references[index] = *reference
	}
	return references
}

func buildTfSchemaReferences(schemaReferences []sr.SchemaReference) *[]map[string]interface{} {
	tfSchemaReferences := make([]map[string]interface{}, len(schemaReferences))
	for i, schemaReference := range schemaReferences {
		tfSchemaReferences[i] = *buildTfSchemaReference(schemaReference)
	}
	return &tfSchemaReferences
}

func buildTfSchemaReference(schemaReference sr.SchemaReference) *map[string]interface{} {
	tfSchemaReference := make(map[string]interface{})
	tfSchemaReference[paramSubjectName] = schemaReference.GetSubject()
	tfSchemaReference[paramName] = schemaReference.GetName()
	tfSchemaReference[paramVersion] = schemaReference.GetVersion()
	return &tfSchemaReference
}

func schemaRegistryClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The Schema Registry cluster ID (e.g., `lsrc-abc123`).",
				},
			},
		},
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
	}
}

func extractSchemaRegistryRestEndpoint(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isSchemaRegistryMetadataSet {
		return client.schemaRegistryRestEndpoint, nil
	}
	if isImportOperation {
		restEndpoint := getEnv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", "")
		if restEndpoint != "" {
			return restEndpoint, nil
		} else {
			return "", fmt.Errorf("one of provider.schema_registry_rest_endpoint (defaults to SCHEMA_REGISTRY_REST_ENDPOINT environment variable) or IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT environment variable must be set")
		}
	}
	restEndpoint := d.Get(paramRestEndpoint).(string)
	if restEndpoint != "" {
		return restEndpoint, nil
	}
	return "", fmt.Errorf("one of provider.schema_registry_rest_endpoint (defaults to SCHEMA_REGISTRY_REST_ENDPOINT environment variable) or resource.rest_endpoint must be set")
}

func extractSchemaRegistryClusterApiKeyAndApiSecret(client *Client, d *schema.ResourceData, isImportOperation bool) (string, string, error) {
	if client.isSchemaRegistryMetadataSet {
		return client.schemaRegistryApiKey, client.schemaRegistryApiSecret, nil
	}
	if isImportOperation {
		clusterApiKey := getEnv("IMPORT_SCHEMA_REGISTRY_API_KEY", "")
		clusterApiSecret := getEnv("IMPORT_SCHEMA_REGISTRY_API_SECRET", "")
		if clusterApiKey != "" && clusterApiSecret != "" {
			return clusterApiKey, clusterApiSecret, nil
		} else {
			return "", "", fmt.Errorf("one of (provider.schema_registry_api_key, provider.schema_registry_api_secret), (SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET environment variables) or (IMPORT_SCHEMA_REGISTRY_API_KEY, IMPORT_SCHEMA_REGISTRY_API_SECRET environment variables) must be set")
		}
	}
	clusterApiKey, clusterApiSecret := extractClusterApiKeyAndApiSecretFromCredentialsBlock(d)
	if clusterApiKey != "" {
		return clusterApiKey, clusterApiSecret, nil
	}
	return "", "", fmt.Errorf("one of (provider.schema_registry_api_key, provider.schema_registry_api_secret), (SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET environment variables) or (resource.credentials.key, resource.credentials.secret) must be set")
}

func extractSchemaRegistryClusterId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isSchemaRegistryMetadataSet {
		return client.schemaRegistryClusterId, nil
	}
	if isImportOperation {
		clusterId := getEnv("IMPORT_SCHEMA_REGISTRY_ID", "")
		if clusterId != "" {
			return clusterId, nil
		} else {
			return "", fmt.Errorf("one of provider.schema_registry_id (defaults to SCHEMA_REGISTRY_ID environment variable) or IMPORT_SCHEMA_REGISTRY_ID environment variable must be set")
		}
	}
	clusterId := extractStringValueFromBlock(d, paramSchemaRegistryCluster, paramId)
	if clusterId != "" {
		return clusterId, nil
	}
	return "", fmt.Errorf("one of provider.schema_registry_id (defaults to SCHEMA_REGISTRY_ID environment variable) or resource.schema_registry_cluster.id must be set")
}

func schemaImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllSchemas,
	}
}

func loadAllSchemas(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	schemaRegistryRestClient := client.schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(client.schemaRegistryRestEndpoint, client.schemaRegistryClusterId, client.schemaRegistryApiKey, client.schemaRegistryApiSecret, true)

	subjects, _, err := schemaRegistryRestClient.apiClient.SubjectsV1Api.List(schemaRegistryRestClient.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Subjects for Schema Registry Cluster %q: %s", schemaRegistryRestClient.clusterId, createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err))
	}
	subjectsJson, err := json.Marshal(subjects)
	if err != nil {
		return nil, diag.Errorf("error reading Subjects for Schema Registry Cluster %q: error marshaling %#v to json: %s", schemaRegistryRestClient.clusterId, subjects, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Subjects for Schema Registry Cluster %q: %s", schemaRegistryRestClient.clusterId, subjectsJson), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})

	for _, subjectName := range subjects {
		// using schemaSr as schema collides with the package name
		schemaSr, _, err := loadSchema(schemaRegistryRestClient.apiContext(ctx), &schema.ResourceData{}, schemaRegistryRestClient, subjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading the latest Schema for Subject %q: %s", schemaSr.GetSubject(), createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})
			return nil, diag.FromErr(createDescriptiveError(err))
		}
		schemaJson, err := json.Marshal(schemaSr)
		if err != nil {
			return nil, diag.Errorf("error reading the latest Schema for Subject %q: error marshaling %#v to json: %s", schemaSr.GetSubject(), schemaSr, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Fetched the latest Schema for Subject %q: %s", schemaSr.GetSubject(), schemaJson), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})

		instanceId := createSchemaId(schemaRegistryRestClient.clusterId, schemaSr.GetSubject(), schemaSr.GetId(), false)
		instances[instanceId] = toValidTerraformResourceName(schemaSr.GetSubject())
	}

	return instances, nil
}
