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
	sr "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	paramSchemaIdentifier = "schema_identifier"
	paramSchema           = "schema"
	paramSchemaReference  = "schema_reference"
	paramSubjectName      = "subject_name"
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
				Required:     true,
				ForceNew:     true,
				Description:  "The definition of the Schema.",
				ValidateFunc: validation.StringIsNotEmpty,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					format := d.Get(paramFormat).(string)
					if format == avroFormat || format == jsonFormat {
						normalizedOldJson, _ := structure.NormalizeJsonString(old)
						normalizedNewJson, _ := structure.NormalizeJsonString(new)
						return normalizedOldJson == normalizedNewJson
					} else if format == protobufFormat {
						return compareTwoProtos(new, old)
					}
					// There's an input validation for schema attribute on a schema level already,
					// so this line won't be run.
					return false
				},
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
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramSubjectName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "The name of the referenced Schema Registry Subject (for example, \"User\").",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "The name of the Schema references (for example, \"io.confluent.kafka.example.User\"). For Avro, the reference name is the fully qualified schema name, for JSON Schema it is a URL, and for Protobuf, it is the name of another Protobuf file.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramVersion: {
							Type:        schema.TypeInt,
							Required:    true,
							ForceNew:    true,
							Description: "The version of the referenced Schema.",
						},
					},
				},
			},
		},
	}
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
	schemaReferences := buildSchemaReferences(d.Get(paramSchemaReference).([]interface{}))

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
	if !validationResponse.GetIsCompatible() {
		return diag.Errorf("error creating Schema: error validating a schema: %s", schemaNotCompatibleErrorMessage)
	}

	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema: %s", createSchemaRequestJson))

	registeredSchema, _, err := executeSchemaCreate(ctx, schemaRegistryRestClient, createSchemaRequest, subjectName)

	if err != nil {
		return diag.Errorf("error creating Schema: %s", createDescriptiveError(err))
	}

	schemaId := createSchemaId(schemaRegistryRestClient.clusterId, subjectName, registeredSchema.GetId())
	d.SetId(schemaId)

	// https://github.com/confluentinc/terraform-provider-confluent/issues/40#issuecomment-1048782379
	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	return schemaRead(ctx, d, meta)
}

func schemaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Remove subject when deleting the last schema
	tflog.Debug(ctx, fmt.Sprintf("Soft deleting Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error soft deleting Schema: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error soft deleting Schema: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error soft deleting Schema: %s", createDescriptiveError(err))
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)
	subjectName := d.Get(paramSubjectName).(string)
	schemaVersion := d.Get(paramVersion).(int)

	_, _, err = schemaRegistryRestClient.apiClient.SubjectVersionsV1Api.DeleteSchemaVersion(schemaRegistryRestClient.apiContext(ctx), subjectName, strconv.Itoa(schemaVersion)).Execute()

	if err != nil {
		return diag.Errorf("error soft deleting Schema %q: %s", d.Id(), createDescriptiveError(err))
	}

	time.Sleep(schemaRegistryAPIWaitAfterCreateOrDelete)

	tflog.Debug(ctx, fmt.Sprintf("Finished soft deleting Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})

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
	if d.HasChangesExcept(paramCredentials, paramConfigs) {
		return diag.Errorf("error updating Schema %q: only %q and %q blocks can be updated for Schema", d.Id(), paramCredentials, paramConfigs)
	}
	return schemaRead(ctx, d, meta)
}

func createSchemaId(clusterId, subjectName string, identifier int32) string {
	return fmt.Sprintf("%s/%s/%d", clusterId, subjectName, identifier)
}

func extractSchemaIdentifierFromTfId(terraformId string) (int32, error) {
	parts := strings.Split(terraformId, "/")

	if len(parts) != 3 {
		return 0, fmt.Errorf("error extracting Schema Identifier from Resource ID: invalid format: expected '<Schema Registry cluster ID>/<subject name>/<schema identifier>'")
	}

	stringIdentifier := parts[2]
	identifier, err := strconv.Atoi(stringIdentifier)
	if err != nil {
		return 0, fmt.Errorf("error extracting Schema Identifier from Resource ID: invalid format: expected '<schema identifier>'=%q to be an int: %s", stringIdentifier, err)
	}
	return int32(identifier), nil
}

func extractSubjectNameFromTfId(terraformId string) (string, error) {
	parts := strings.Split(terraformId, "/")

	if len(parts) != 3 {
		return "", fmt.Errorf("error extracting Subject Name from Resource ID: invalid format: expected '<Schema Registry cluster ID>/<subject name>/<schema identifier>'")
	}

	return parts[1], nil
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
	parts := strings.Split(clusterIDAndSubjectNameAndSchemaIdentifier, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Schema: invalid format: expected '<SG cluster ID>/<subject name>/<schema identifier>'")
	}

	clusterId := parts[0]
	subjectName := parts[1]
	schemaIdentifier := parts[2]
	schemaIdentifierInt, err := strconv.Atoi(schemaIdentifier)
	if err != nil {
		return nil, fmt.Errorf("error importing Schema: invalid format: expected schema identifier from '<SG cluster ID>/<subject name>/<schema identifier>' to be an int")
	}
	schemaRegistryRestClient := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSchemaRegistryConfigAndSetAttributes(ctx, d, schemaRegistryRestClient, subjectName, int32(schemaIdentifierInt)); err != nil {
		return nil, fmt.Errorf("error importing Schema %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Schema %q", d.Id()), map[string]interface{}{schemaLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSchemaRegistryConfigAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, subjectName string, schemaIdentifier int32) ([]*schema.ResourceData, error) {
	// Sample Response (it doesn't include soft deleted schemas):
	//  [{"subject": "test2", "version": 5, "id": 100004, "schema": "{\"type\":\"record\",...}]}"},
	//   {"subject": "test2", "version": 6, "id": 100006, "schema": "{\"type\":\"record\",...}]}"}]
	schemas, _, err := c.apiClient.SchemasV1Api.GetSchemas(c.apiContext(ctx)).Execute()
	if err != nil {
		return nil, fmt.Errorf("error reading Schema %q: %s", d.Id(), createDescriptiveError(err))
	}
	schemasJson, err := json.Marshal(schemas)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema %q: error marshaling %#v to json: %s", d.Id(), schemas, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schemas %q: %s", d.Id(), schemasJson), map[string]interface{}{schemaLoggingKey: d.Id()})
	srSchema, exists := findSchemaById(schemas, schemaIdentifier, subjectName)
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
	if err := d.Set(paramSchema, srSchema.GetSchema()); err != nil {
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

	d.SetId(createSchemaId(c.clusterId, srSchema.GetSubject(), srSchema.GetId()))

	return []*schema.ResourceData{d}, nil
}

func findSchemaById(schemas []sr.Schema, schemaIdentifier int32, subjectName string) (sr.Schema, bool) {
	// 'schema' collides with a package name
	for _, srSchema := range schemas {
		if srSchema.GetId() == schemaIdentifier && srSchema.GetSubject() == subjectName {
			return srSchema, true
		}
	}
	return sr.Schema{}, false
}

func executeSchemaValidate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.RegisterSchemaRequest, subjectName string) (sr.CompatibilityCheckResponse, *http.Response, error) {
	return c.apiClient.CompatibilityV1Api.TestCompatibilityForSubject(c.apiContext(ctx), subjectName).RegisterSchemaRequest(*requestData).Execute()
}

func executeSchemaCreate(ctx context.Context, c *SchemaRegistryRestClient, requestData *sr.RegisterSchemaRequest, subjectName string) (sr.RegisterSchemaResponse, *http.Response, error) {
	return c.apiClient.SubjectVersionsV1Api.Register(c.apiContext(ctx), subjectName).RegisterSchemaRequest(*requestData).Execute()
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
