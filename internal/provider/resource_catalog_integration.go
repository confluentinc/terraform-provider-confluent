// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"strings"

	tableflow "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramLastSyncAt                  = "last_sync_at"
	paramCatalogIntegrationAwsGlue   = "catalog_integration_aws_glue"
	paramCatalogIntegrationSnowflake = "catalog_integration_snowflake"
	paramEndpoint                    = "endpoint"
	paramClientId                    = "client_id"
	paramClientSecret                = "client_secret"
	paramWarehouse                   = "warehouse"
	paramAllowedScope                = "allowed_scope"

	awsGlueSpecKind   = "AwsGlue"
	snowflakeSpecKind = "Snowflake"
)

var acceptedCatalogIntegrationConnectionTypes = []string{paramCatalogIntegrationAwsGlue, paramCatalogIntegrationSnowflake}

func catalogIntegrationResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: catalogIntegrationCreate,
		ReadContext:   catalogIntegrationRead,
		UpdateContext: catalogIntegrationUpdate,
		DeleteContext: catalogIntegrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: catalogIntegrationImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the catalog integration.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramSuspended: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates whether the Catalog Integration should be suspended.",
			},
			paramLastSyncAt: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time at which the catalog was last synced. It is represented in RFC3339 format and is in UTC.",
			},
			paramKafkaCluster:                requiredKafkaClusterBlockSchema(),
			paramEnvironment:                 environmentSchema(),
			paramCredentials:                 credentialsSchema(),
			paramCatalogIntegrationAwsGlue:   catalogIntegrationAwsGlueSchema(),
			paramCatalogIntegrationSnowflake: catalogIntegrationSnowflakeSchema(),
		},
	}
}

func catalogIntegrationAwsGlueSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "The catalog integration Glue connection configuration.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramProviderIntegrationId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedCatalogIntegrationConnectionTypes,
	}
}

func catalogIntegrationSnowflakeSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "The catalog integration connection configuration for Snowflake Open Catalog.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramEndpoint: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The catalog integration connection endpoint for Snowflake Open Catalog.",
				},
				paramClientId: {
					Type:      schema.TypeString,
					Required:  true,
					Sensitive: true,
				},
				paramClientSecret: {
					Type:      schema.TypeString,
					Required:  true,
					Sensitive: true,
				},
				paramWarehouse: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Warehouse name of the Snowflake Open Catalog.",
				},
				paramAllowedScope: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "Allowed scope of the Snowflake Open Catalog.",
				},
			},
		},
		MinItems:     1,
		MaxItems:     1,
		ExactlyOneOf: acceptedCatalogIntegrationConnectionTypes,
	}
}

func catalogIntegrationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet)

	isAwsGlue := len(d.Get(paramCatalogIntegrationAwsGlue).([]interface{})) > 0
	isSnowflake := len(d.Get(paramCatalogIntegrationSnowflake).([]interface{})) > 0

	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	catalogIntegrationSpec := tableflow.NewTableflowV1CatalogIntegrationSpec()
	catalogIntegrationSpec.SetDisplayName(displayName)
	catalogIntegrationSpec.SetEnvironment(tableflow.GlobalObjectReference{Id: environmentId})
	catalogIntegrationSpec.SetKafkaCluster(tableflow.EnvScopedObjectReference{Id: clusterId})
	if isAwsGlue {
		catalogIntegrationSpec.SetConfig(tableflow.TableflowV1CatalogIntegrationSpecConfigOneOf{
			TableflowV1CatalogIntegrationAwsGlueSpec: &tableflow.TableflowV1CatalogIntegrationAwsGlueSpec{
				Kind:                  awsGlueSpecKind,
				ProviderIntegrationId: extractStringValueFromBlock(d, paramCatalogIntegrationAwsGlue, paramProviderIntegrationId),
			},
		})
	} else if isSnowflake {
		catalogIntegrationSpec.SetConfig(tableflow.TableflowV1CatalogIntegrationSpecConfigOneOf{
			TableflowV1CatalogIntegrationSnowflakeSpec: &tableflow.TableflowV1CatalogIntegrationSnowflakeSpec{
				Kind:         snowflakeSpecKind,
				Endpoint:     extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramEndpoint),
				ClientId:     extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramClientId),
				ClientSecret: extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramClientSecret),
				Warehouse:    extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramWarehouse),
				AllowedScope: extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramAllowedScope),
			},
		})
	}

	createCatalogIntegrationRequest := tableflow.NewTableflowV1CatalogIntegration()
	createCatalogIntegrationRequest.SetSpec(*catalogIntegrationSpec)

	createCatalogIntegrationRequestJson, err := json.Marshal(createCatalogIntegrationRequest)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: error marshaling %#v to json: %s", createCatalogIntegrationRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Catalog Integration: %s", createCatalogIntegrationRequestJson))

	req := tableflowRestClient.apiClient.CatalogIntegrationsTableflowV1Api.CreateTableflowV1CatalogIntegration(tableflowRestClient.apiContext(ctx)).TableflowV1CatalogIntegration(*createCatalogIntegrationRequest)
	createdCatalogIntegration, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}

	d.SetId(createdCatalogIntegration.GetId())

	// No SLA on status transition from PENDING -> CONNECTED
	//if err := waitForTableflowCatalogIntegrationToProvision(tableflowRestClient.apiContext(ctx), tableflowRestClient, environmentId, clusterId, d.Id(), c.isAcceptanceTestMode); err != nil {
	//	return diag.Errorf("error waiting for Catalog Integration %q to provision: %s", d.Id(), createDescriptiveError(err))
	//}

	createdCatalogIntegrationJson, err := json.Marshal(createdCatalogIntegration)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration %q: error marshaling %#v to json: %s", d.Id(), createdCatalogIntegration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Catalog Integration %q: %s", d.Id(), createdCatalogIntegrationJson), map[string]interface{}{catalogIntegrationKey: d.Id()})

	return catalogIntegrationRead(ctx, d, meta)
}

func executeCatalogIntegrationRead(ctx context.Context, c *TableflowRestClient, environmentId, clusterId, id string) (tableflow.TableflowV1CatalogIntegration, *http.Response, error) {
	return c.apiClient.CatalogIntegrationsTableflowV1Api.GetTableflowV1CatalogIntegration(c.apiContext(ctx), id).Environment(environmentId).SpecKafkaCluster(clusterId).Execute()
}

func catalogIntegrationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Catalog Integration %q", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet)

	catalogIntegrationId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	if _, err := readCatalogIntegrationAndSetAttributes(ctx, d, tableflowRestClient, environmentId, clusterId, catalogIntegrationId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Catalog Integration %q: %s", catalogIntegrationId, createDescriptiveError(err)))
	}

	return nil
}

func readCatalogIntegrationAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *TableflowRestClient, environmentId, clusterId, catalogIntegrationId string) ([]*schema.ResourceData, error) {
	catalogIntegration, resp, err := executeCatalogIntegrationRead(c.apiContext(ctx), c, environmentId, clusterId, catalogIntegrationId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Catalog Integration %q: %s", catalogIntegrationId, createDescriptiveError(err)), map[string]interface{}{catalogIntegrationKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Catalog Integration %q in TF state because Catalog Integration could not be found on the server", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	catalogIntegrationJson, err := json.Marshal(catalogIntegration)
	if err != nil {
		return nil, fmt.Errorf("error reading Catalog Integration %q: error marshaling %#v to json: %s", catalogIntegrationId, catalogIntegration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Catalog Integration %q: %s", d.Id(), catalogIntegrationJson), map[string]interface{}{catalogIntegrationKey: d.Id()})

	if _, err := setCatalogIntegrationAttributes(d, c, catalogIntegration); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Catalog Integration %q", catalogIntegrationId), map[string]interface{}{catalogIntegrationKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setCatalogIntegrationAttributes(d *schema.ResourceData, c *TableflowRestClient, catalogIntegration tableflow.TableflowV1CatalogIntegration) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, catalogIntegration.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSuspended, catalogIntegration.Spec.GetSuspended()); err != nil {
		return nil, err
	}
	if err := d.Set(paramLastSyncAt, catalogIntegration.Status.GetLastSyncAt()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, catalogIntegration.GetSpec().Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramKafkaCluster, paramId, catalogIntegration.GetSpec().KafkaCluster.GetId(), d); err != nil {
		return nil, createDescriptiveError(err)
	}

	if catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationAwsGlueSpec != nil {
		if err := d.Set(paramCatalogIntegrationAwsGlue, []interface{}{map[string]interface{}{
			paramProviderIntegrationId: catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationAwsGlueSpec.GetProviderIntegrationId(),
		}}); err != nil {
			return nil, err
		}
	} else if catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationSnowflakeSpec != nil {
		// We cannot read these two values from the backend, so read the stored value instead to prevent drift
		var clientId, clientSecret string
		if currentClientId, ok := d.GetOk(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramClientId)); ok {
			clientId = currentClientId.(string)
		}
		if currentClientSecret, ok := d.GetOk(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramClientSecret)); ok {
			clientSecret = currentClientSecret.(string)
		}

		if err := d.Set(paramCatalogIntegrationSnowflake, []interface{}{map[string]interface{}{
			paramEndpoint:     catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationSnowflakeSpec.GetEndpoint(),
			paramWarehouse:    catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationSnowflakeSpec.GetWarehouse(),
			paramAllowedScope: catalogIntegration.Spec.GetConfig().TableflowV1CatalogIntegrationSnowflakeSpec.GetAllowedScope(),
			paramClientId:     clientId,
			paramClientSecret: clientSecret,
		}}); err != nil {
			return nil, err
		}
	}

	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.tableflowApiKey, c.tableflowApiSecret, d); err != nil {
			return nil, err
		}
	}

	d.SetId(catalogIntegration.GetId())
	return d, nil
}

func catalogIntegrationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Catalog Integration %q", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)
	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet)

	req := tableflowRestClient.apiClient.CatalogIntegrationsTableflowV1Api.DeleteTableflowV1CatalogIntegration(tableflowRestClient.apiContext(ctx), d.Id()).Environment(environmentId).SpecKafkaCluster(clusterId)
	_, err = req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Catalog Integration %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Catalog Integration %q", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})

	return nil
}

func catalogIntegrationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramCatalogIntegrationAwsGlue, paramCatalogIntegrationSnowflake) {
		return diag.Errorf("error updating Catalog Integration %q: only %q, %q, %q, %q, %q, %q, %q attributes can be updated for Catalog Integration", d.Id(), paramDisplayName, paramProviderIntegrationId, paramEndpoint, paramWarehouse, paramAllowedScope, paramClientId, paramClientSecret)
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	clusterId := extractStringValueFromBlock(d, paramKafkaCluster, paramId)

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, false)
	if err != nil {
		return diag.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet)

	updateCatalogIntegrationSpec := tableflow.NewTableflowV1CatalogIntegrationSpec()
	updateCatalogIntegrationSpec.SetEnvironment(tableflow.GlobalObjectReference{Id: environmentId})
	updateCatalogIntegrationSpec.SetKafkaCluster(tableflow.EnvScopedObjectReference{Id: clusterId})
	if d.HasChange(paramDisplayName) {
		updateCatalogIntegrationSpec.SetDisplayName(d.Get(paramDisplayName).(string))
	}
	if d.HasChange(paramCatalogIntegrationAwsGlue) && d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationAwsGlue, paramProviderIntegrationId)) {
		updateCatalogIntegrationSpec.SetConfig(tableflow.TableflowV1CatalogIntegrationAwsGlueSpecAsTableflowV1CatalogIntegrationSpecConfigOneOf(&tableflow.TableflowV1CatalogIntegrationAwsGlueSpec{
			Kind:                  awsGlueSpecKind,
			ProviderIntegrationId: extractStringValueFromBlock(d, paramCatalogIntegrationAwsGlue, paramProviderIntegrationId),
		}))
	}
	if d.HasChange(paramCatalogIntegrationSnowflake) {
		updateCatalogIntegrationSpec.SetConfig(tableflow.TableflowV1CatalogIntegrationSnowflakeSpecAsTableflowV1CatalogIntegrationSpecConfigOneOf(&tableflow.TableflowV1CatalogIntegrationSnowflakeSpec{
			Kind: snowflakeSpecKind,
		}))
		if d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramEndpoint)) {
			updateCatalogIntegrationSpec.Config.TableflowV1CatalogIntegrationSnowflakeSpec.SetEndpoint(extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramEndpoint))
		}
		if d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramWarehouse)) {
			updateCatalogIntegrationSpec.Config.TableflowV1CatalogIntegrationSnowflakeSpec.SetWarehouse(extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramWarehouse))
		}
		if d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramAllowedScope)) {
			updateCatalogIntegrationSpec.Config.TableflowV1CatalogIntegrationSnowflakeSpec.SetAllowedScope(extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramAllowedScope))
		}
		if d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramClientId)) {
			updateCatalogIntegrationSpec.Config.TableflowV1CatalogIntegrationSnowflakeSpec.SetClientId(extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramClientId))
		}
		if d.HasChange(fmt.Sprintf("%s.0.%s", paramCatalogIntegrationSnowflake, paramClientSecret)) {
			updateCatalogIntegrationSpec.Config.TableflowV1CatalogIntegrationSnowflakeSpec.SetClientSecret(extractStringValueFromBlock(d, paramCatalogIntegrationSnowflake, paramClientSecret))
		}
	}

	updateCatalogIntegration := tableflow.NewTableflowV1CatalogIntegration()
	updateCatalogIntegration.SetSpec(*updateCatalogIntegrationSpec)

	updateCatalogIntegrationJson, err := json.Marshal(updateCatalogIntegration)
	if err != nil {
		return diag.Errorf("error updating Catalog Integration %q: error marshaling %#v to json: %s", d.Id(), updateCatalogIntegration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Catalog Integration %q: %s", d.Id(), updateCatalogIntegrationJson), map[string]interface{}{catalogIntegrationKey: d.Id()})

	req := tableflowRestClient.apiClient.CatalogIntegrationsTableflowV1Api.UpdateTableflowV1CatalogIntegration(tableflowRestClient.apiContext(ctx), d.Id()).TableflowV1CatalogIntegration(*updateCatalogIntegration)
	updatedCatalogIntegration, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error updating Catalog Integration %q: %s", d.Id(), createDescriptiveError(err))
	}

	UpdatedCatalogIntegrationJson, err := json.Marshal(updatedCatalogIntegration)
	if err != nil {
		return diag.Errorf("error updating Catalog Integration %q: error marshaling %#v to json: %s", d.Id(), updatedCatalogIntegration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Catalog Integration %q: %s", d.Id(), UpdatedCatalogIntegrationJson), map[string]interface{}{catalogIntegrationKey: d.Id()})
	return catalogIntegrationRead(ctx, d, meta)
}

func catalogIntegrationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Catalog Integration %q", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})

	c := meta.(*Client)

	tableflowApiKey, tableflowApiSecret, err := extractTableflowApiKeyAndApiSecret(c, d, true)
	if err != nil {
		return nil, fmt.Errorf("error creating Catalog Integration: %s", createDescriptiveError(err))
	}
	tableflowRestClient := c.tableflowRestClientFactory.CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret, c.isTableflowMetadataSet)

	envIDAndClusterIDAndCatalogIntegrationId := d.Id()
	parts := strings.Split(envIDAndClusterIDAndCatalogIntegrationId, "/")

	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Catalog Integration: invalid format: expected '<env ID>/<Kafka cluster ID>/<Catalog integration ID>'")
	}

	environmentId := parts[0]
	clusterId := parts[1]
	d.SetId(parts[2])

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readCatalogIntegrationAndSetAttributes(ctx, d, tableflowRestClient, environmentId, clusterId, d.Id()); err != nil {
		return nil, fmt.Errorf("error importing Catalog Integration %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Catalog Integration %q", d.Id()), map[string]interface{}{catalogIntegrationKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
