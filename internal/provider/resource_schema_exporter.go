// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var acceptedSchemaExporterStatus = []string{stateRunning, statePaused}

const (
	paramContextType                      = "context_type"
	paramContext                          = "context"
	paramSubjectRenameFormat              = "subject_rename_format"
	paramSubjects                         = "subjects"
	paramResetOnUpdate                    = "reset_on_update"
	paramResetOnUpdateDefaultValue        = false
	paramBasicAuthCredentialsSourceValue  = "USER_INFO"
	paramDestinationSchemaRegistryCluster = "destination_schema_registry_cluster"
	basicAuthCredentialsSourceConfig      = "basic.auth.credentials.source"
	schemaRegistryUrlConfig               = "schema.registry.url"
	basicAuthUserInfoConfig               = "basic.auth.user.info"

	bearerAuthClientId          = "bearer.auth.client.id"
	bearerAuthClientSecret      = "bearer.auth.client.secret"
	bearerAuthIssuerEndpointUrl = "bearer.auth.issuer.endpoint.url"
	bearerAuthCredentialsSource = "bearer.auth.credentials.source"
	bearerAuthScope             = "bearer.auth.scope"
	bearerAuthIdentityPoolId    = "bearer.auth.identity.pool.id"
	bearerAuthLogicalCluster    = "bearer.auth.logical.cluster"

	schemaExporterAPICreateTimeout = 12 * time.Hour
)

func schemaExporterResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   schemaExporterRead,
		CreateContext: schemaExporterCreate,
		DeleteContext: schemaExporterDelete,
		UpdateContext: schemaExporterUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schemaExporterImport,
		},
		Schema: map[string]*schema.Schema{
			paramSchemaRegistryCluster: schemaRegistryClusterBlockSchema(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Schema Registry cluster, for example, `https://psrc-00000.us-central1.gcp.confluent.cloud:443`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
			paramName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			paramContextType: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramContext: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramSubjectRenameFormat: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramSubjects: {
				Type:     schema.TypeSet,
				Computed: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			paramDestinationSchemaRegistryCluster: destinationSchemaRegistryClusterBlockSchema(),
			paramConfigs: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
			},
			paramSensitiveConfig: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Sensitive: true,
				Optional:  true,
				Computed:  true,
				ForceNew:  false,
			},
			paramResetOnUpdate: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  paramResetOnUpdateDefaultValue,
			},
			paramStatus: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(acceptedSchemaExporterStatus, false),
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(schemaExporterAPICreateTimeout),
		},
		CustomizeDiff: customdiff.Sequence(resourceCredentialBlockValidationWithOAuth),
	}
}

func destinationSchemaRegistryClusterBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Required: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Optional:     true,
					ForceNew:     true,
					Computed:     true,
					ValidateFunc: validation.StringIsNotEmpty,
					DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
						// During API key -> OAuth migration, ignore diffs on id as it is not required for API key/secret authentication
						// In this scenario, resource should not be recreated
						// Only suppress during updates (when resource already exists), not during creation
						if d.Id() != "" && old == "" && new != "" {
							return true
						}
						return false
					},
					Description: "The ID of the destination Schema Registry cluster. Required when using OAuth authentication.",
				},
				paramRestEndpoint: {
					Type:     schema.TypeString,
					Required: true,
				},
				paramCredentials: {
					Type:      schema.TypeList,
					Optional:  true,
					MinItems:  1,
					MaxItems:  1,
					Sensitive: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							paramKey: {
								Type:         schema.TypeString,
								Required:     true,
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
							paramSecret: {
								Type:         schema.TypeString,
								Required:     true,
								Sensitive:    true,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
		},
	}
}

func constructDestinationSRClusterRequest(d *schema.ResourceData, meta interface{}) map[string]string {
	client := meta.(*Client)
	// Merge non-sensitive and sensitive configs so that values from both blocks reach the API.
	mergedConfigs, _, _ := extractSRExporterProperties(d)
	configs := mergedConfigs
	if _, ok := configs[schemaRegistryUrlConfig]; !ok {
		configs[schemaRegistryUrlConfig] = extractStringValueFromBlock(d, paramDestinationSchemaRegistryCluster, paramRestEndpoint)
	}

	// OAuth specific configurations
	if client.isOAuthEnabled {
		destinationClusterId := extractStringValueFromBlock(d, paramDestinationSchemaRegistryCluster, paramId)
		applyOAuthDefaults(configs, client.oauthToken, destinationClusterId)
		return configs
	}

	configs[basicAuthCredentialsSourceConfig] = paramBasicAuthCredentialsSourceValue
	destinationSRClusterApiKey := extractStringValueFromNestedBlock(d, paramDestinationSchemaRegistryCluster, paramCredentials, paramKey)
	destinationSRClusterApiSecret := extractStringValueFromNestedBlock(d, paramDestinationSchemaRegistryCluster, paramCredentials, paramSecret)
	configs[basicAuthUserInfoConfig] = fmt.Sprintf("%s:%s", destinationSRClusterApiKey, destinationSRClusterApiSecret)
	return configs
}

// applyOAuthDefaults sets OAuth bearer config values from the provider-level OAuthToken
// only if they are not already specified in the user's config block.
func applyOAuthDefaults(configs map[string]string, token *OAuthToken, destinationClusterId string) {
	if _, ok := configs[bearerAuthClientId]; !ok {
		configs[bearerAuthClientId] = token.ClientId
	}
	if _, ok := configs[bearerAuthClientSecret]; !ok {
		configs[bearerAuthClientSecret] = token.ClientSecret
	}
	if _, ok := configs[bearerAuthIssuerEndpointUrl]; !ok {
		configs[bearerAuthIssuerEndpointUrl] = token.TokenUrl
	}
	if _, ok := configs[bearerAuthCredentialsSource]; !ok {
		configs[bearerAuthCredentialsSource] = configOAuthBearer
	}
	if _, ok := configs[bearerAuthIdentityPoolId]; !ok {
		configs[bearerAuthIdentityPoolId] = token.IdentityPoolId
	}
	if _, ok := configs[bearerAuthLogicalCluster]; !ok {
		configs[bearerAuthLogicalCluster] = destinationClusterId
	}
	// The Scope field is optional for Okta, but required for Azure Entra ID
	// setting arbitrary values may cause exporter exception from backend service
	if _, ok := configs[bearerAuthScope]; !ok && token.Scope != "" {
		configs[bearerAuthScope] = token.Scope
	}
}

func schemaExporterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Schema Exporter: %s", createDescriptiveError(err))
	}
	c := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	subjects := convertToStringSlice(d.Get(paramSubjects).(*schema.Set).List())
	exporterId := createExporterId(clusterId, d.Get(paramName).(string))
	name := d.Get(paramName).(string)

	_, sensitiveConfigs, _ := extractSRExporterProperties(d)
	// basic.auth.user.info is handled internally via destination_schema_registry_cluster credentials,
	// so ignore it in sensitive_config to avoid drift.
	delete(sensitiveConfigs, basicAuthUserInfoConfig)

	er := sr.NewExporterReference()
	er.SetName(name)
	if v := d.Get(paramContext).(string); v != "" {
		er.SetContext(v)
	}
	if v := d.Get(paramContextType).(string); v != "" {
		er.SetContextType(v)
	}
	if v := d.Get(paramSubjectRenameFormat).(string); v != "" {
		er.SetSubjectRenameFormat(v)
	}
	er.SetSubjects(subjects)
	er.SetConfig(constructDestinationSRClusterRequest(d, meta))

	if err := d.Set(paramSensitiveConfig, sensitiveConfigs); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	request := c.apiClient.ExportersV1Api.RegisterExporter(c.apiContext(ctx)).ExporterReference(*er)
	requestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Schema Exporter: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Schema Exporter: %s", requestJson))

	createdExporter, resp, err := request.Execute()
	if err != nil {
		return diag.Errorf("error creating Schema Exporter: %s", createDescriptiveError(err, resp))
	}

	if err := waitForSchemaExporterToProvision(c.apiContext(ctx), c, exporterId, name); err != nil {
		return diag.Errorf("error waiting for Schema Exporter %q to provision: %s", exporterId, createDescriptiveError(err, resp))
	}

	d.SetId(exporterId)

	createdExporterJson, err := json.Marshal(createdExporter)
	if err != nil {
		return diag.Errorf("error creating Schema Exporter %q: error marshaling %#v to json: %s", exporterId, exporterId, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Schema Exporter %q: %s", exporterId, createdExporterJson), map[string]interface{}{schemaExporterLoggingKey: exporterId})

	return schemaExporterRead(ctx, d, meta)
}

func schemaExporterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get(paramName).(string)
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Exporter %q", name))
	if _, err := readSchemaExporterAndSetAttributes(ctx, d, meta, false); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Schema Exporter %q: %s", name, createDescriptiveError(err)))
	}

	return nil
}

func readSchemaExporterAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, isImportOperation bool) ([]*schema.ResourceData, error) {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, isImportOperation)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, isImportOperation)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, isImportOperation)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Exporter: %s", createDescriptiveError(err))
	}
	c := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	name := d.Get(paramName).(string)
	id := createExporterId(clusterId, name)

	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Exporter %q=%q", paramId, id), map[string]interface{}{schemaExporterLoggingKey: id})

	request := c.apiClient.ExportersV1Api.GetExporterInfoByName(c.apiContext(ctx), name)
	exporter, resp, err := request.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Exporter %q: %s", id, createDescriptiveError(err, resp)), map[string]interface{}{schemaExporterLoggingKey: id})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Schema Exporter %q in TF state because Schema Exporter could not be found on the server", id), map[string]interface{}{schemaExporterLoggingKey: id})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	exporterJson, err := json.Marshal(exporter)
	if err != nil {
		return nil, fmt.Errorf("error reading Schema Exporter %q: error marshaling %#v to json: %s", id, exporterJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Schema Exporter %q: %s", id, exporterJson), map[string]interface{}{schemaExporterLoggingKey: id})

	// Read and set the Schema Exporter status with a different API call
	if err := readSchemaExporterStatusAndSetAttributes(ctx, d, c, id, name); err != nil {
		return nil, err
	}

	if _, err := setSchemaExporterAttributes(d, clusterId, exporter, c, meta, isImportOperation); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Schema Exporter %q", id), map[string]interface{}{schemaExporterLoggingKey: id})

	return []*schema.ResourceData{d}, nil
}

func schemaExporterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Schema Exporter: %s", createDescriptiveError(err))
	}
	c := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	name := d.Get(paramName).(string)
	id := createExporterId(clusterId, name)

	if d.HasChange(paramStatus) {
		isPaused := d.Get(paramStatus).(string) == statePaused
		if isPaused {
			// pause the exporter first before making any changes
			_, resp, err := c.apiClient.ExportersV1Api.PauseExporterByName(c.apiContext(ctx), name).Execute()
			if err != nil {
				return diag.Errorf("error pausing Schema Exporter (Failed to pause the exporter): %s", createDescriptiveError(err, resp))
			}
		}
	}

	if d.HasChanges(paramContextType, paramContext, paramSubjectRenameFormat, paramSubjects, paramConfigs, paramSensitiveConfig, paramDestinationSchemaRegistryCluster) {
		// pause the exporter whenever there's an update on configs
		// https://github.com/confluentinc/terraform-provider-confluent/issues/321
		_, resp, err := c.apiClient.ExportersV1Api.PauseExporterByName(c.apiContext(ctx), name).Execute()
		if err != nil {
			return diag.Errorf("error pausing Schema Exporter (Failed to pause the exporter): %s", createDescriptiveError(err, resp))
		}

		subjects := convertToStringSlice(d.Get(paramSubjects).(*schema.Set).List())

		req := sr.NewExporterUpdateRequest()
		if v := d.Get(paramContext).(string); v != "" {
			req.SetContext(v)
		}
		if v := d.Get(paramContextType).(string); v != "" {
			req.SetContextType(v)
		}
		if v := d.Get(paramSubjectRenameFormat).(string); v != "" {
			req.SetSubjectRenameFormat(v)
		}
		req.SetSubjects(subjects)
		req.SetConfig(constructDestinationSRClusterRequest(d, meta))

		request := c.apiClient.ExportersV1Api.UpdateExporterInfo(c.apiContext(ctx), name).ExporterUpdateRequest(*req)
		requestJson, err := json.Marshal(request)
		if err != nil {
			return diag.Errorf("error updating Schema Exporter: error marshaling %#v to json: %s", request, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Schema Exporter: %s", requestJson))

		updatedExporter, resp, err := request.Execute()
		if err != nil {
			return diag.Errorf("error updating Schema Exporter: %s", createDescriptiveError(err, resp))
		}
		updatedExporterJson, err := json.Marshal(updatedExporter)
		if err != nil {
			return diag.Errorf("error updating Schema Exporter: error marshaling %#v to json: %s", request, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Schema Exporter: %s", updatedExporterJson))

		isReset := d.Get(paramResetOnUpdate).(bool)
		if isReset {
			_, resp, err := c.apiClient.ExportersV1Api.ResetExporterByName(c.apiContext(ctx), name).Execute()
			if err != nil {
				return diag.Errorf("error updating Schema Exporter (Failed to reset the exporter): %s", createDescriptiveError(err, resp))
			}
		}

		error := resumeExporter(ctx, d, c, name, id)
		if error != nil {
			return error
		}
	}

	if d.HasChange(paramStatus) {
		error := resumeExporter(ctx, d, c, name, id)
		if error != nil {
			return error
		}
	}

	d.SetId(id)
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Schema Exporter %q", id), map[string]interface{}{schemaExporterLoggingKey: id})
	return schemaExporterRead(ctx, d, meta)
}

func resumeExporter(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, name string, id string) diag.Diagnostics {
	isRunning := d.Get(paramStatus).(string) == stateRunning
	if isRunning {
		// resume the exporter last after making any changes
		_, resp, err := c.apiClient.ExportersV1Api.ResumeExporterByName(c.apiContext(ctx), name).Execute()
		if err != nil && (resp == nil || resp.StatusCode != http.StatusConflict) {
			return diag.Errorf("error resuming Schema Exporter (Failed to resume the exporter): %s", createDescriptiveError(err, resp))
		}

		if err := waitForSchemaExporterToProvision(c.apiContext(ctx), c, id, name); err != nil {
			return diag.Errorf("error waiting for Schema Exporter %q to updating: %s", id, createDescriptiveError(err, resp))
		}
		status, resp, err := c.apiClient.ExportersV1Api.GetExporterStatusByName(c.apiContext(ctx), name).Execute()
		if err != nil {
			return diag.Errorf("error resuming Schema Exporter (Failed to read status): %s", createDescriptiveError(err, resp))
		}
		if status.GetTrace() != "" {
			return diag.Errorf("error resuming Schema Exporter %q: %s", id, status.GetTrace())
		}
	}
	return nil
}

func schemaExporterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractSchemaRegistryRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterId, err := extractSchemaRegistryClusterId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Exporter: %s", createDescriptiveError(err))
	}
	clusterApiKey, clusterApiSecret, err := extractSchemaRegistryClusterApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Schema Exporter: %s", createDescriptiveError(err))
	}

	name := d.Get(paramName).(string)
	id := createExporterId(clusterId, name)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Schema Exporter %q=%q", paramId, id), map[string]interface{}{schemaExporterLoggingKey: id})

	c := meta.(*Client).schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret, meta.(*Client).isSchemaRegistryMetadataSet, meta.(*Client).oauthToken)

	// pause the exporter first
	_, resp, err := c.apiClient.ExportersV1Api.PauseExporterByName(c.apiContext(ctx), name).Execute()
	if err != nil {
		return diag.Errorf("error deleting Schema Exporter (failed to pause the exporter): %s", createDescriptiveError(err, resp))
	}

	request := c.apiClient.ExportersV1Api.DeleteExporter(c.apiContext(ctx), name)
	resp, err = request.Execute()
	if err != nil {
		return diag.Errorf("error deleting Schema Exporter %q: %s", id, createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Schema Exporter %q", id), map[string]interface{}{schemaExporterLoggingKey: id})

	return nil
}

func schemaExporterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("error importing Schema Exporter: Schema Exporter id is missing")
	}

	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Schema Exporter: invalid format: expected '<Schema Registry Cluster Id>/<Schema Exporter Name>'")
	}
	if err := d.Set(paramName, parts[1]); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Imporing Schema Exporter %q=%q", paramId, id), map[string]interface{}{schemaExporterLoggingKey: id})
	d.MarkNewResource()
	if _, err := readSchemaExporterAndSetAttributes(ctx, d, meta, true); err != nil {
		return nil, fmt.Errorf("error importing Schema Exporter %q: %s", id, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func setSchemaExporterAttributes(d *schema.ResourceData, clusterId string, exporter sr.ExporterReference, c *SchemaRegistryRestClient, meta interface{}, isImportOperation bool) (*schema.ResourceData, error) {
	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.clusterApiKey, c.clusterApiSecret, d, c.externalAccessToken != nil); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramSchemaRegistryCluster, paramId, c.clusterId, d); err != nil {
			return nil, err
		}
	}

	if err := d.Set(paramName, exporter.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramContextType, exporter.GetContextType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramContext, exporter.GetContext()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSubjectRenameFormat, exporter.GetSubjectRenameFormat()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSubjects, exporter.GetSubjects()); err != nil {
		return nil, err
	}

	configs := exporter.GetConfig()
	if err := setDestinationSchemaRegistryClusterAttributes(d, configs, meta.(*Client).isOAuthEnabled); err != nil {
		return nil, err
	}

	filteredConfigs := filterExporterConfigs(configs, d, isImportOperation)
	if err := d.Set(paramConfigs, filteredConfigs); err != nil {
		return nil, err
	}

	if isImportOperation {
		// Sensitive config values are redacted by the API, so set to empty on import
		// (same pattern as the connector resource).
		if err := d.Set(paramSensitiveConfig, make(map[string]string)); err != nil {
			return nil, err
		}
	}

	d.SetId(createExporterId(clusterId, exporter.GetName()))
	return d, nil
}

func readSchemaExporterStatusAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *SchemaRegistryRestClient, id, name string) error {
	tflog.Debug(ctx, fmt.Sprintf("Reading Schema Exporter Status %q", name))
	status, resp, err := c.apiClient.ExportersV1Api.GetExporterStatusByName(c.apiContext(ctx), name).Execute()
	if err != nil {
		return fmt.Errorf("error reading Schema Exporter %q Status: %s", id, createDescriptiveError(err, resp))
	}
	switch state := status.GetState(); state {
	case stateRunning, statePaused:
		// valid states at this point
		if err := d.Set(paramStatus, state); err != nil {
			return err
		}
	}
	return nil
}

func setDestinationSchemaRegistryClusterAttributes(d *schema.ResourceData, configs map[string]string, isOAuthEnabled bool) error {
	destinationClusterId := configs[bearerAuthLogicalCluster]
	destinationClusterEndpoint := configs[schemaRegistryUrlConfig]
	destinationSRClusterApiKey := extractStringValueFromNestedBlock(d, paramDestinationSchemaRegistryCluster, paramCredentials, paramKey)
	destinationSRClusterApiSecret := extractStringValueFromNestedBlock(d, paramDestinationSchemaRegistryCluster, paramCredentials, paramSecret)

	if isOAuthEnabled {
		if err := d.Set(paramDestinationSchemaRegistryCluster, []interface{}{map[string]interface{}{
			paramId:           destinationClusterId,
			paramRestEndpoint: destinationClusterEndpoint,
		},
		}); err != nil {
			return err
		}
	} else {
		if err := d.Set(paramDestinationSchemaRegistryCluster, []interface{}{map[string]interface{}{
			paramRestEndpoint: destinationClusterEndpoint,
			paramCredentials: []interface{}{map[string]interface{}{
				paramKey:    destinationSRClusterApiKey,
				paramSecret: destinationSRClusterApiSecret,
			}},
		}}); err != nil {
			return err
		}
	}
	return nil
}

// exporterBoilerplateConfigs are config keys auto-added by the API that are managed
// by the destination_schema_registry_cluster block and should not appear in config state.
var exporterBoilerplateConfigs = []string{
	schemaRegistryUrlConfig,
	basicAuthUserInfoConfig,
	basicAuthCredentialsSourceConfig,
	bearerAuthClientId,
	bearerAuthClientSecret,
	bearerAuthIssuerEndpointUrl,
	bearerAuthCredentialsSource,
	bearerAuthScope,
	bearerAuthIdentityPoolId,
	bearerAuthLogicalCluster,
}

// filterExporterConfigs filters the API response config map for storage in TF state.
//
// On import: keeps all non-redacted, non-boilerplate keys from the API response so users
// get a populated config block they can adopt into their .tf files.
//
// On normal read: only keeps keys that already exist in state, preserving "[hidden]" values
// from state. This avoids drift from boilerplate keys the API auto-adds.
func filterExporterConfigs(apiConfigs map[string]string, d *schema.ResourceData, isImportOperation bool) map[string]string {
	if isImportOperation {
		filtered := make(map[string]string)
		boilerplate := make(map[string]bool, len(exporterBoilerplateConfigs))
		for _, key := range exporterBoilerplateConfigs {
			boilerplate[key] = true
		}
		for key, value := range apiConfigs {
			if boilerplate[key] || value == "[hidden]" {
				continue
			}
			filtered[key] = value
		}
		return filtered
	}

	// Normal read: only keep keys already in state
	currentConfigs := convertToStringStringMap(d.Get(paramConfigs).(map[string]interface{}))
	filtered := make(map[string]string)
	for key := range currentConfigs {
		if apiValue, ok := apiConfigs[key]; ok {
			if apiValue == "[hidden]" {
				filtered[key] = currentConfigs[key]
			} else {
				filtered[key] = apiValue
			}
		}
	}
	return filtered
}

func createExporterId(clusterId, exporterName string) string {
	return fmt.Sprintf("%s/%s", clusterId, exporterName)
}

func extractSRExporterProperties(d *schema.ResourceData) (map[string]string, map[string]string, map[string]string) {
	sensitiveProperties := convertToStringStringMap(d.Get(paramSensitiveConfig).(map[string]interface{}))
	nonsensitiveProperties := convertToStringStringMap(d.Get(paramConfigs).(map[string]interface{}))

	// Merge both configs
	properties := lo.Assign(
		nonsensitiveProperties,
		sensitiveProperties,
	)

	return properties, sensitiveProperties, nonsensitiveProperties
}
