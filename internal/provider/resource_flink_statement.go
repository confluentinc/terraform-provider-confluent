// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	fgb "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1beta1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramStatementName = "statement_name"
	paramStatement     = "statement"
	paramComputePool   = "compute_pool"
	paramProperties    = "properties"
	paramStopped       = "stopped"

	stateCompleted = "COMPLETED"
	statePending   = "PENDING"
	stateFailing   = "FAILING"

	paramResourceVersion = "resource_version"

	exampleFlinkRestEndpoint = "https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123"
)

func flinkStatementResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: flinkStatementCreate,
		ReadContext:   flinkStatementRead,
		UpdateContext: flinkStatementUpdate,
		DeleteContext: flinkStatementDelete,
		Importer: &schema.ResourceImporter{
			StateContext: flinkStatementImport,
		},
		Schema: map[string]*schema.Schema{
			paramComputePool: optionalIdBlockSchema(),
			paramPrincipal:   optionalIdBlockSchema(),
			paramStatementName: {
				Type:        schema.TypeString,
				Description: "The unique identifier of the Statement.",
				Optional:    true,
				Computed:    true,
			},
			paramStatement: {
				Type:         schema.TypeString,
				Description:  "The raw SQL text of the Statement.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramProperties: {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
			},
			paramStopped: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicates whether the statement should be stopped.",
			},
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Compute Pool cluster, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1beta1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramResourceVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A system generated string that uniquely identifies the version of this resource.",
			},
			paramCredentials: credentialsSchema(),
		},
	}
}

func flinkStatementCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	computePoolRestEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	flinkRegionRestEndpoint, organizationId, environmentId, err := extractFlinkAttributes(computePoolRestEndpoint)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(flinkRegionRestEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)

	statementName := d.Get(paramStatementName).(string)
	if len(statementName) == 0 {
		statementName = generateFlinkStatementName()
	}

	statement := d.Get(paramStatement).(string)
	properties := convertToStringStringMap(d.Get(paramProperties).(map[string]interface{}))

	spec := fgb.NewSqlV1beta1StatementSpec()
	spec.SetStatement(statement)
	spec.SetProperties(properties)
	spec.SetComputePoolId(computePoolId)
	spec.SetPrincipal(principalId)

	createFlinkStatementRequest := fgb.NewSqlV1beta1Statement()
	createFlinkStatementRequest.SetName(statementName)
	createFlinkStatementRequest.SetSpec(*spec)

	createFlinkStatementRequestJson, err := json.Marshal(createFlinkStatementRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: error marshaling %#v to json: %s", createFlinkStatementRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Flink Statement: %s", createFlinkStatementRequestJson))

	createdFlinkStatement, _, err := executeFlinkStatementCreate(flinkRestClient.apiContext(ctx), flinkRestClient, createFlinkStatementRequest)
	if err != nil {
		return diag.Errorf("error creating Flink Statement %q: %s", createdFlinkStatement.GetName(), createDescriptiveError(err))
	}
	d.SetId(createFlinkStatementId(flinkRestClient.environmentId, createdFlinkStatement.Spec.GetComputePoolId(), createdFlinkStatement.GetName()))

	if err := waitForFlinkStatementToProvision(flinkRestClient.apiContext(ctx), flinkRestClient, createdFlinkStatement.GetName()); err != nil {
		return diag.Errorf("error waiting for Flink Statement %q to provision: %s", createdFlinkStatement.GetName(), createDescriptiveError(err))
	}

	createdFlinkStatementJson, err := json.Marshal(createdFlinkStatement)
	if err != nil {
		return diag.Errorf("error creating Flink Statement %q: error marshaling %#v to json: %s", createdFlinkStatement.GetName(), createdFlinkStatement, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Statement %q: %s", createdFlinkStatement.GetName(), createdFlinkStatementJson), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	return flinkStatementRead(ctx, d, meta)
}

func executeFlinkStatementCreate(ctx context.Context, c *FlinkRestClient, requestData *fgb.SqlV1beta1Statement) (fgb.SqlV1beta1Statement, *http.Response, error) {
	req := c.apiClient.StatementsSqlV1beta1Api.CreateSqlv1beta1Statement(c.apiContext(ctx), c.organizationId, c.environmentId).SqlV1beta1Statement(*requestData)
	return req.Execute()
}

func executeFlinkStatementRead(ctx context.Context, c *FlinkRestClient, statementName string) (fgb.SqlV1beta1Statement, *http.Response, error) {
	req := c.apiClient.StatementsSqlV1beta1Api.GetSqlv1beta1Statement(c.apiContext(ctx), c.organizationId, c.environmentId, statementName)
	return req.Execute()
}

func flinkStatementRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	computePoolRestEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	flinkRegionRestEndpoint, organizationId, environmentId, err := extractFlinkAttributes(computePoolRestEndpoint)
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(flinkRegionRestEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)
	statementName, err := parseStatementName(d.Id())
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}

	if _, err := readFlinkStatementAndSetAttributes(ctx, d, flinkRestClient, statementName); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Flink Statement %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func flinkStatementUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramStopped) {
		return diag.Errorf("error updating Flink Statement %q: only %q attribute can be updated for Flink Statement", d.Id(), paramStopped)
	}
	computePoolRestEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	flinkRegionRestEndpoint, organizationId, environmentId, err := extractFlinkAttributes(computePoolRestEndpoint)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(flinkRegionRestEndpoint, organizationId, environmentId, "", computePoolId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)
	updatedStopped := d.Get(paramStopped).(bool)
	statementName := d.Get(paramStatementName).(string)
	resourceVersion := d.Get(paramResourceVersion).(string)
	updatedSpec := fgb.NewSqlV1beta1StatementSpec()
	updatedSpec.SetStatement(statementName)
	updatedSpec.SetComputePoolId(flinkRestClient.computePoolId)
	updatedSpec.SetPrincipal(flinkRestClient.principalId)
	updatedSpec.SetStopped(updatedStopped)
	updateFlinkStatementRequest := fgb.NewSqlV1beta1Statement()
	updateFlinkStatementRequest.SetName(statementName)
	updatedMetadata := fgb.NewObjectMetaWithDefaults()
	updatedMetadata.SetResourceVersion(resourceVersion)
	updateFlinkStatementRequest.SetMetadata(*updatedMetadata)
	updateFlinkStatementRequest.SetSpec(*updatedSpec)
	updateFlinkStatementRequestJson, err := json.Marshal(updateFlinkStatementRequest)
	if err != nil {
		return diag.Errorf("error updating Flink Statement %q: error marshaling %#v to json: %s", statementName, updateFlinkStatementRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Statement %q: %s", statementName, updateFlinkStatementRequestJson), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
	req := flinkRestClient.apiClient.StatementsSqlV1beta1Api.UpdateSqlv1beta1Statement(flinkRestClient.apiContext(ctx), organizationId, environmentId, statementName).SqlV1beta1Statement(*updateFlinkStatementRequest)
	_, err = req.Execute()
	if err != nil {
		return diag.Errorf("error updating Flink Statement %q: %s", statementName, createDescriptiveError(err))
	}
	if err := waitForFlinkStatementToBeStopped(flinkRestClient.apiContext(ctx), flinkRestClient, statementName); err != nil {
		return diag.Errorf("error waiting for Flink Statement %q to be stopped: %s", statementName, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Flink Statement %q", statementName), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
	return flinkStatementRead(ctx, d, meta)
}

func readFlinkStatementAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *FlinkRestClient, statementName string) ([]*schema.ResourceData, error) {
	statement, resp, err := executeFlinkStatementRead(c.apiContext(ctx), c, statementName)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Statement %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Statement %q in TF state because Flink Statement could not be found on the server", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	statementJson, err := json.Marshal(statement)
	if err != nil {
		return nil, fmt.Errorf("error reading Flink Statement %q: error marshaling %#v to json: %s", statementName, statement, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Statement %q: %s", d.Id(), statementJson), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	if _, err := setFlinkStatementAttributes(d, c, statement); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setFlinkStatementAttributes(d *schema.ResourceData, c *FlinkRestClient, statement fgb.SqlV1beta1Statement) (*schema.ResourceData, error) {
	if err := d.Set(paramStatementName, statement.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatement, statement.Spec.GetStatement()); err != nil {
		return nil, err
	}
	if err := d.Set(paramProperties, statement.Spec.GetProperties()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStopped, statement.Spec.GetStopped()); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceVersion, statement.Metadata.GetResourceVersion()); err != nil {
		return nil, err
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.flinkApiKey, c.flinkApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, constructComputePoolRestEndpoint(c.restEndpoint, c.organizationId, c.environmentId)); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramComputePool, paramId, statement.Spec.GetComputePoolId(), d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramPrincipal, paramId, statement.Spec.GetPrincipal(), d); err != nil {
			return nil, err
		}
	}
	d.SetId(createFlinkStatementId(statement.GetEnvironmentId(), statement.Spec.GetComputePoolId(), statement.GetName()))
	return d, nil
}

func flinkStatementDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	computePoolRestEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	flinkRegionRestEndpoint, organizationId, environmentId, err := extractFlinkAttributes(computePoolRestEndpoint)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(flinkRegionRestEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)
	statementName := d.Get(paramStatementName).(string)

	req := flinkRestClient.apiClient.StatementsSqlV1beta1Api.DeleteSqlv1beta1Statement(flinkRestClient.apiContext(ctx), organizationId, environmentId, statementName)
	_, err = req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Flink Statement %q: %s", statementName, createDescriptiveError(err))
	}

	if err := waitForFlinkStatementToBeDeleted(flinkRestClient.apiContext(ctx), flinkRestClient, statementName); err != nil {
		return diag.Errorf("error waiting for Flink Statement %q to be deleted: %s", statementName, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Statement %q", statementName), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	return nil
}

func flinkStatementImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	computePoolRestEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, true)
	flinkRegionRestEndpoint, organizationId, environmentId, err := extractFlinkAttributes(computePoolRestEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(flinkRegionRestEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)

	statementName := d.Id()
	d.SetId(createFlinkStatementId(environmentId, computePoolId, statementName))

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readFlinkStatementAndSetAttributes(ctx, d, flinkRestClient, statementName); err != nil {
		return nil, fmt.Errorf("error importing Flink Statement %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func optionalIdBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
			},
		},
	}
}

func extractFlinkRestEndpoint(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkRestEndpoint, nil
	}
	if isImportOperation {
		restEndpoint := getEnv("IMPORT_FLINK_REST_ENDPOINT", "")

		// Trim outer quotes from the retrieved values.
		restEndpoint = strings.Trim(restEndpoint, "\"")

		if restEndpoint != "" {
			return restEndpoint, nil
		} else {
			return "", fmt.Errorf("one of provider.flink_rest_endpoint (defaults to FLINK_REST_ENDPOINT environment variable) or IMPORT_FLINK_REST_ENDPOINT environment variable must be set")
		}
	}
	restEndpoint := d.Get(paramRestEndpoint).(string)
	if restEndpoint != "" {
		return restEndpoint, nil
	}
	return "", fmt.Errorf("one of provider.flink_rest_endpoint (defaults to FLINK_REST_ENDPOINT environment variable) or resource.rest_endpoint must be set")
}

func extractFlinkApiKeyAndApiSecret(client *Client, d *schema.ResourceData, isImportOperation bool) (string, string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkApiKey, client.flinkApiSecret, nil
	}
	if isImportOperation {
		clusterApiKey := getEnv("IMPORT_FLINK_API_KEY", "")
		clusterApiSecret := getEnv("IMPORT_FLINK_API_SECRET", "")

		// Trim outer quotes from the retrieved values.
		clusterApiKey = strings.Trim(clusterApiKey, "\"")
		clusterApiSecret = strings.Trim(clusterApiSecret, "\"")

		if clusterApiKey != "" && clusterApiSecret != "" {
			return clusterApiKey, clusterApiSecret, nil
		} else {
			return "", "", fmt.Errorf("one of (provider.flink_api_key, provider.flink_api_secret), (FLINK_API_KEY, FLINK_API_SECRET environment variables) or (IMPORT_FLINK_API_KEY, IMPORT_FLINK_API_SECRET environment variables) must be set")
		}
	}
	clusterApiKey, clusterApiSecret := extractClusterApiKeyAndApiSecretFromCredentialsBlock(d)
	if clusterApiKey != "" {
		return clusterApiKey, clusterApiSecret, nil
	}
	return "", "", fmt.Errorf("one of (provider.flink_api_key, provider.flink_api_secret), (FLINK_API_KEY, FLINK_API_SECRET environment variables) or (resource.credentials.key, resource.credentials.secret) must be set")
}

func extractFlinkComputePoolId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkComputePoolId, nil
	}
	if isImportOperation {
		computePoolId := getEnv("IMPORT_FLINK_COMPUTE_POOL_ID", "")
		if computePoolId != "" {
			return computePoolId, nil
		} else {
			return "", fmt.Errorf("one of provider.flink_compute_pool_id (defaults to FLINK_COMPUTE_POOL_ID environment variable) or IMPORT_FLINK_COMPUTE_POOL_ID environment variable must be set")
		}
	}
	computePoolId := extractStringValueFromBlock(d, paramComputePool, paramId)
	if computePoolId != "" {
		return computePoolId, nil
	}
	return "", fmt.Errorf("one of provider.flink_compute_pool_id (defaults to FLINK_COMPUTE_POOL_ID environment variable) or resource.compute_pool.id must be set")
}

func extractFlinkPrincipalId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkPrincipalId, nil
	}
	if isImportOperation {
		principalId := getEnv("IMPORT_FLINK_PRINCIPAL_ID", "")
		if principalId != "" {
			return principalId, nil
		} else {
			return "", fmt.Errorf("one of provider.flink_principal_id (defaults to FLINK_PRINCIPAL_ID environment variable) or IMPORT_FLINK_PRINCIPAL_ID environment variable must be set")
		}
	}
	principalId := extractStringValueFromBlock(d, paramPrincipal, paramId)
	if principalId != "" {
		return principalId, nil
	}
	return "", fmt.Errorf("one of provider.flink_principal_id (defaults to FLINK_PRINCIPAL_ID environment variable) or resource.principal.id must be set")
}

func createFlinkStatementId(environmentId, computePoolId, statementName string) string {
	return fmt.Sprintf("%s/%s/%s", environmentId, computePoolId, statementName)
}
