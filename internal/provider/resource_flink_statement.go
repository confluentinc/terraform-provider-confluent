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
	fgb "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"time"
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

	statementsAPICreateTimeout = 6 * time.Hour
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
			paramOrganization: optionalIdBlockSchema(),
			paramEnvironment:  optionalIdBlockSchema(),
			paramComputePool:  optionalIdBlockSchema(),
			paramPrincipal:    optionalIdBlockSchema(),
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
				Computed:    true,
				Description: "Indicates whether the statement should be stopped.",
			},
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Compute Pool cluster, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(statementsAPICreateTimeout),
		},
	}
}

func flinkStatementCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
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
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)

	statementName := d.Get(paramStatementName).(string)
	if len(statementName) == 0 {
		statementName = generateFlinkStatementName()
	}

	statement := d.Get(paramStatement).(string)
	properties := convertToStringStringMap(d.Get(paramProperties).(map[string]interface{}))

	spec := fgb.NewSqlV1StatementSpec()
	spec.SetStatement(statement)
	spec.SetProperties(properties)
	spec.SetComputePoolId(computePoolId)
	spec.SetPrincipal(principalId)

	createFlinkStatementRequest := fgb.NewSqlV1Statement()
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

	if err := waitForFlinkStatementToProvision(flinkRestClient.apiContext(ctx), flinkRestClient, createdFlinkStatement.GetName(), meta.(*Client).isAcceptanceTestMode); err != nil {
		return diag.Errorf("error waiting for Flink Statement %q to provision: %s", createdFlinkStatement.GetName(), createDescriptiveError(err))
	}

	createdFlinkStatementJson, err := json.Marshal(createdFlinkStatement)
	if err != nil {
		return diag.Errorf("error creating Flink Statement %q: error marshaling %#v to json: %s", createdFlinkStatement.GetName(), createdFlinkStatement, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Statement %q: %s", createdFlinkStatement.GetName(), createdFlinkStatementJson), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	return flinkStatementRead(ctx, d, meta)
}

func executeFlinkStatementCreate(ctx context.Context, c *FlinkRestClient, requestData *fgb.SqlV1Statement) (fgb.SqlV1Statement, *http.Response, error) {
	req := c.apiClient.StatementsSqlV1Api.CreateSqlv1Statement(c.apiContext(ctx), c.organizationId, c.environmentId).SqlV1Statement(*requestData)
	return req.Execute()
}

func executeFlinkStatementRead(ctx context.Context, c *FlinkRestClient, statementName string) (fgb.SqlV1Statement, *http.Response, error) {
	req := c.apiClient.StatementsSqlV1Api.GetSqlv1Statement(c.apiContext(ctx), c.organizationId, c.environmentId, statementName)
	return req.Execute()
}

func flinkStatementRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error reading Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
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
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)
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
	updatedStopped := d.Get(paramStopped).(bool)
	if updatedStopped == false {
		return diag.Errorf("error updating Flink Statement %q: Flink Statement cannot be resumed. Only "+
			"%s=false -> %s=true updates are supported.", d.Id(), paramStopped, paramStopped)
	}

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error updating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)

	statementName := d.Get(paramStatementName).(string)

	req := flinkRestClient.apiClient.StatementsSqlV1Api.GetSqlv1Statement(flinkRestClient.apiContext(ctx), flinkRestClient.organizationId, flinkRestClient.environmentId, statementName)
	statement, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Flink Statement: error fetching Flink Statement: %s", createDescriptiveError(err))
	}

	// The statement could be automatically stopped if no client has consumed the results for 5 minutes or more.
	// Therefore, we need to double-check whether the backend has already stopped the statement.
	shouldSendUpdateRequest := !statement.Spec.GetStopped()
	if shouldSendUpdateRequest {
		statement.Spec.SetStopped(true)
		updateFlinkStatementRequestJson, err := json.Marshal(statement)
		if err != nil {
			return diag.Errorf("error updating Flink Statement %q: error marshaling %#v to json: %s", statementName, statement, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating Flink Statement %q: %s", statementName, updateFlinkStatementRequestJson), map[string]interface{}{flinkStatementLoggingKey: d.Id()})
		req := flinkRestClient.apiClient.StatementsSqlV1Api.UpdateSqlv1Statement(flinkRestClient.apiContext(ctx), organizationId, environmentId, statementName).SqlV1Statement(statement)
		_, err = req.Execute()
		if err != nil {
			return diag.Errorf("error updating Flink Statement 123 %q: %s", statementName, createDescriptiveError(err))
		}
		if err := waitForFlinkStatementToBeStopped(flinkRestClient.apiContext(ctx), flinkRestClient, statementName, meta.(*Client).isAcceptanceTestMode); err != nil {
			return diag.Errorf("error waiting for Flink Statement %q to be stopped: %s", statementName, createDescriptiveError(err))
		}
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

func setFlinkStatementAttributes(d *schema.ResourceData, c *FlinkRestClient, statement fgb.SqlV1Statement) (*schema.ResourceData, error) {
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

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.flinkApiKey, c.flinkApiSecret, d); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramOrganization, paramId, statement.GetOrganizationId(), d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, statement.GetEnvironmentId(), d); err != nil {
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

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return diag.Errorf("error deleting Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, "", flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)
	statementName := d.Get(paramStatementName).(string)

	req := flinkRestClient.apiClient.StatementsSqlV1Api.DeleteSqlv1Statement(flinkRestClient.apiContext(ctx), organizationId, environmentId, statementName)
	_, err = req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Flink Statement %q: %s", statementName, createDescriptiveError(err))
	}

	if err := waitForFlinkStatementToBeDeleted(flinkRestClient.apiContext(ctx), flinkRestClient, statementName, meta.(*Client).isAcceptanceTestMode); err != nil {
		return diag.Errorf("error waiting for Flink Statement %q to be deleted: %s", statementName, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Statement %q", statementName), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	return nil
}

func flinkStatementImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Statement %q", d.Id()), map[string]interface{}{flinkStatementLoggingKey: d.Id()})

	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, true)
	if err != nil {
		return nil, fmt.Errorf("error importing Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet)

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

func extractFlinkOrganizationId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkOrganizationId, nil
	}
	if isImportOperation {
		organizationId := getEnv("IMPORT_CONFLUENT_ORGANIZATION_ID", "")
		if organizationId != "" {
			return organizationId, nil
		} else {
			return "", fmt.Errorf("one of provider.organization_id (defaults to CONFLUENT_ORGANIZATION_ID environment variable) or IMPORT_CONFLUENT_ORGANIZATION_ID environment variable must be set")
		}
	}
	organizationId := extractStringValueFromBlock(d, paramOrganization, paramId)
	if organizationId != "" {
		return organizationId, nil
	}
	return "", fmt.Errorf("one of provider.organization_id (defaults to CONFLUENT_ORGANIZATION_ID environment variable) or resource.organization.id must be set")
}

func extractFlinkEnvironmentId(client *Client, d *schema.ResourceData, isImportOperation bool) (string, error) {
	if client.isFlinkMetadataSet {
		return client.flinkEnvironmentId, nil
	}
	if isImportOperation {
		environmentId := getEnv("IMPORT_CONFLUENT_ENVIRONMENT_ID", "")
		if environmentId != "" {
			return environmentId, nil
		} else {
			return "", fmt.Errorf("one of provider.environment_id (defaults to CONFLUENT_ENVIRONMENT_ID environment variable) or IMPORT_CONFLUENT_ENVIRONMENT_ID environment variable must be set")
		}
	}
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId != "" {
		return environmentId, nil
	}
	return "", fmt.Errorf("one of provider.environment_id (defaults to CONFLUENT_ENVIRONMENT_ID environment variable) or resource.environment.id must be set")
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
