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
	apikeys "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	paramOwner               = "owner"
	paramResource            = "managed_resource"
	paramDisableWaitForReady = "disable_wait_for_ready"

	serviceAccountKind   = "ServiceAccount"
	userKind             = "User"
	clusterKind          = "Cluster"
	schemaRegistryKind   = "SchemaRegistry"
	ksqlDbKind           = "ksqlDB"
	cloudKindInLowercase = "cloud"

	iamApiVersion      = "iam/v2"
	cmkApiVersion      = "cmk/v2"
	srcmApiVersion     = "srcm/v2"
	ksqldbcmApiVersion = "ksqldbcm/v2"
)

var acceptedOwnerKinds = []string{serviceAccountKind, userKind}
var acceptedResourceKinds = []string{clusterKind}

var acceptedOwnerApiVersions = []string{iamApiVersion}
var acceptedResourceApiVersions = []string{cmkApiVersion, srcmApiVersion, ksqldbcmApiVersion}

func apiKeyResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: apiKeyCreate,
		ReadContext:   apiKeyRead,
		UpdateContext: apiKeyUpdate,
		DeleteContext: apiKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: apiKeyImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "A human-readable name for the API key.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "A free-form description of the API key.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramOwner: apiKeyOwnerSchema(),
			// The API Key resource represents Cloud API Key if paramResource is not set
			paramResource: apiKeyResourceSchema(),
			paramSecret: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The API Key Secret.",
			},
			paramDisableWaitForReady: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
	}
}

func apiKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	skipSync := d.Get(paramDisableWaitForReady).(bool)

	ownerId := extractStringValueFromBlock(d, paramOwner, paramId)
	ownerKind := extractStringValueFromBlock(d, paramOwner, paramKind)

	spec := apikeys.NewIamV2ApiKeySpec()
	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetOwner(apikeys.ObjectReference{Id: ownerId, Kind: &ownerKind})

	// If paramResource block is present, then the API Key is a resource-specific API key (Kafka, Schema Registry, and ksqlDB).
	// https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#resource-specific-api-keys
	// Otherwise, it's Cloud API Key.
	isResourceSpecificApiKey := len(d.Get(paramResource).([]interface{})) > 0

	// Will be set to "" if not found (e.g., for Cloud API Key)
	environmentId := extractStringValueFromNestedBlock(d, paramResource, paramEnvironment, paramId)
	if isResourceSpecificApiKey {
		resourceId := extractStringValueFromBlock(d, paramResource, paramId)
		resourceKind := extractStringValueFromBlock(d, paramResource, paramKind)
		spec.SetResource(apikeys.ObjectReference{Id: resourceId, Kind: &resourceKind})
	}

	createApiKeyRequest := apikeys.IamV2ApiKey{Spec: spec}
	createApiKeyRequestJson, err := json.Marshal(createApiKeyRequest)
	if err != nil {
		return diag.Errorf("error creating API Key: error marshaling %#v to json: %s", createApiKeyRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new API Key: %s", createApiKeyRequestJson))

	createdApiKey, _, err := executeApiKeysCreate(c.apiKeysApiContext(ctx), c, &createApiKeyRequest)
	if err != nil {
		return diag.Errorf("error creating API Key %q: %s", createdApiKey.GetId(), createDescriptiveError(err))
	}
	if err := validateApiKey(createdApiKey); err != nil {
		return diag.Errorf("Created API Key is malformed: %s", err)
	}

	if !skipSync {
		// Wait until the API Key is synced and is ready to use
		tflog.Debug(ctx, fmt.Sprintf("Waiting for API Key %q to sync", createdApiKey.GetId()), map[string]interface{}{apiKeyLoggingKey: createdApiKey.GetId()})
		if err := waitForApiKeyToSync(ctx, c, createdApiKey, isResourceSpecificApiKey, environmentId); err != nil {
			return diag.FromErr(createDescriptiveError(err))
		}
	}

	// Save the API Key Secret
	if err := d.Set(paramSecret, createdApiKey.Spec.GetSecret()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(createdApiKey.GetId())

	// Set the API Key Secret (sensitive value) to an empty string
	createdApiKey.Spec.SetSecret("")
	createdApiKeyJson, err := json.Marshal(createdApiKey)
	if err != nil {
		return diag.Errorf("error creating API Key %q: error marshaling %#v to json: %s", d.Id(), createdApiKey, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating API Key %q: %s", d.Id(), createdApiKeyJson), map[string]interface{}{apiKeyLoggingKey: d.Id()})

	return apiKeyRead(ctx, d, meta)
}

func executeApiKeysCreate(ctx context.Context, c *Client, apiKey *apikeys.IamV2ApiKey) (apikeys.IamV2ApiKey, *http.Response, error) {
	req := c.apiKeysClient.APIKeysIamV2Api.CreateIamV2ApiKey(c.apiKeysApiContext(ctx)).IamV2ApiKey(*apiKey)
	return req.Execute()
}

func apiKeyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	isUpdateAllowed := d.HasChange(paramDisplayName) || d.HasChange(paramDescription)
	if isUpdateAllowed {
		c := meta.(*Client)
		displayName := d.Get(paramDisplayName).(string)
		description := d.Get(paramDescription).(string)

		updateApiKeyRequest := apikeys.NewIamV2ApiKeyUpdate()
		updateSpec := apikeys.NewIamV2ApiKeySpecUpdate()

		if d.HasChange(paramDisplayName) {
			updateSpec.SetDisplayName(displayName)
		}
		if d.HasChange(paramDescription) {
			updateSpec.SetDescription(description)
		}
		updateApiKeyRequest.SetSpec(*updateSpec)

		updateApiKeyRequestJson, err := json.Marshal(updateApiKeyRequest)
		if err != nil {
			return diag.Errorf("error updating API Key %q: error marshaling %#v to json: %s", d.Id(), updateApiKeyRequest, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Updating API Key %q: %s", d.Id(), updateApiKeyRequestJson), map[string]interface{}{apiKeyLoggingKey: d.Id()})

		req := c.apiKeysClient.APIKeysIamV2Api.UpdateIamV2ApiKey(c.apiKeysApiContext(ctx), d.Id()).IamV2ApiKeyUpdate(*updateApiKeyRequest)
		updatedApiKey, _, err := req.Execute()

		if err != nil {
			return diag.Errorf("error updating API Key %q: %s", d.Id(), createDescriptiveError(err))
		}

		updatedApiKeyJson, err := json.Marshal(updatedApiKey)
		if err != nil {
			return diag.Errorf("error updating API Key %q: error marshaling %#v to json: %s", d.Id(), updatedApiKey, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Finished updating API Key %q: %s", d.Id(), updatedApiKeyJson), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	} else {
		return diag.Errorf("only %s, %s attributes can be updated for an API Key", paramDisplayName, paramDescription)
	}

	return apiKeyRead(ctx, d, meta)
}

func apiKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.apiKeysClient.APIKeysIamV2Api.DeleteIamV2ApiKey(c.apiKeysApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting API Key %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})

	return nil
}

func apiKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	c := meta.(*Client)
	apiKey, resp, err := executeApiKeysRead(c.apiKeysApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading API Key %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{apiKeyLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing API Key %q in TF state because API Key could not be found on the server", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	apiKeyJson, err := json.Marshal(apiKey)
	if err != nil {
		return diag.Errorf("error reading API Key %q: error marshaling %#v to json: %s", d.Id(), apiKey, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched API Key %q: %s", d.Id(), apiKeyJson), map[string]interface{}{apiKeyLoggingKey: d.Id()})

	if _, err := setApiKeyAttributes(d, apiKey); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})

	return nil
}

func setApiKeyAttributes(d *schema.ResourceData, apiKey apikeys.IamV2ApiKey) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, apiKey.Spec.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, apiKey.Spec.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := setOwner(apiKey, d); err != nil {
		return nil, createDescriptiveError(err)
	}
	// Check whether the API Key is a resource-specific API key (Kafka, Schema Registry, and ksqlDB).
	// https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#resource-specific-api-keys
	// Otherwise, it's Cloud API Key.
	isResourceSpecificApiKey := strings.ToLower(apiKey.Spec.Resource.GetKind()) != cloudKindInLowercase
	if isResourceSpecificApiKey {
		environmentId := extractStringValueFromNestedBlock(d, paramResource, paramEnvironment, paramId)
		if err := setManagedResource(apiKey, environmentId, d); err != nil {
			return nil, createDescriptiveError(err)
		}
	}
	// Explicitly set paramDisableWaitForReady to the default value if unset
	if _, ok := d.GetOk(paramDisableWaitForReady); !ok {
		if err := d.Set(paramDisableWaitForReady, d.Get(paramDisableWaitForReady)); err != nil {
			return nil, createDescriptiveError(err)
		}
	}
	d.SetId(apiKey.GetId())
	return d, nil
}

func setOwner(apiKey apikeys.IamV2ApiKey, d *schema.ResourceData) error {
	return d.Set(paramOwner, []interface{}{map[string]interface{}{
		paramId:         apiKey.Spec.Owner.GetId(),
		paramKind:       apiKey.Spec.Owner.GetKind(),
		paramApiVersion: apiKey.Spec.Owner.GetApiVersion(),
	}})
}

func setManagedResource(apiKey apikeys.IamV2ApiKey, environmentId string, d *schema.ResourceData) error {
	// Have to be careful here in case Schema Registry and ksqlDB don't use paramEnvironment
	kind := apiKey.Spec.Resource.GetKind()
	// Hack for API Key Mgmt API that temporarily returns schemaRegistryKind / ksqlDbKind instead of clusterKind
	if kind == schemaRegistryKind || kind == ksqlDbKind {
		kind = clusterKind
	}
	if environmentId != "" {
		return d.Set(paramResource, []interface{}{map[string]interface{}{
			paramId:         apiKey.Spec.Resource.GetId(),
			paramKind:       kind,
			paramApiVersion: apiKey.Spec.Resource.GetApiVersion(),
			paramEnvironment: []interface{}{map[string]interface{}{
				paramId: environmentId,
			}},
		}})
	} else {
		return d.Set(paramResource, []interface{}{map[string]interface{}{
			paramId:         apiKey.Spec.Resource.GetId(),
			paramKind:       kind,
			paramApiVersion: apiKey.Spec.Resource.GetApiVersion(),
		}})
	}
}

func executeApiKeysRead(ctx context.Context, c *Client, apiKeyId string) (apikeys.IamV2ApiKey, *http.Response, error) {
	req := c.apiKeysClient.APIKeysIamV2Api.GetIamV2ApiKey(c.apiKeysApiContext(ctx), apiKeyId)
	return req.Execute()
}

func apiKeyOwnerSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Required:    true,
		ForceNew:    true,
		Description: "The owner to which the API Key belongs. The owner can be one of 'iam.v2.User', 'iam.v2.ServiceAccount'.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The unique identifier for the referred owner.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(u-|sa-)"), "the owner ID must be of the form 'u-' or 'sa-'"),
				},
				paramKind: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The kind of the referred owner.",
					ValidateFunc: validation.StringInSlice(acceptedOwnerKinds, false),
				},
				paramApiVersion: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The API version of the referred owner.",
					ValidateFunc: validation.StringInSlice(acceptedOwnerApiVersions, false),
				},
			},
		},
	}
}

func apiKeyResourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		// If the resource is not specified, then Cloud API Key gets created
		Optional:    true,
		ForceNew:    true,
		Description: "The resource associated with this object. The only resource that is supported is 'cmk.v2.Cluster', 'srcm.v2.Cluster'.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The unique identifier for the referred resource.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(lkc-|lsrc-|lksqlc-)"), "the resource ID must be of the form 'lkc-' or 'lsrc-' or 'lksqlc-'"),
				},
				paramKind: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The kind of the referred resource.",
					ValidateFunc: validation.StringInSlice(acceptedResourceKinds, false),
				},
				paramApiVersion: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The API version of the referred owner.",
					ValidateFunc: validation.StringInSlice(acceptedResourceApiVersions, false),
				},
				paramEnvironment: environmentSchema(),
			},
		},
	}
}

// extractStringValueFromBlock() returns the string for the given key, or "" if the key doesn't exist in the configuration.
// Schema definition's required:true property guarantees that all required resource attributes are set
// hence extractStringValueFromBlock() doesn't return errors (similar to d.Get())
func extractStringValueFromBlock(d *schema.ResourceData, blockName string, attribute string) string {
	// d.Get() will return "" if the key is not present
	return d.Get(fmt.Sprintf("%s.0.%s", blockName, attribute)).(string)
}

func extractStringValueFromNestedBlock(d *schema.ResourceData, outerBlockName string, innerBlockName string, attribute string) string {
	// d.Get() will return "" if the key is not present
	return d.Get(fmt.Sprintf("%s.0.%s.0.%s", outerBlockName, innerBlockName, attribute)).(string)
}

func validateApiKey(apiKey apikeys.IamV2ApiKey) error {
	if _, ok := apiKey.GetIdOk(); !ok {
		return fmt.Errorf("API Key ID is either empty or nil")
	}
	if _, ok := apiKey.Spec.GetSecretOk(); !ok {
		return fmt.Errorf("API Key Secret is either empty or nil")
	}
	return nil
}

// Send a GetCluster request to CMK API to find out rest_endpoint for a given (environmentId, clusterId) pair
func fetchHttpEndpointOfKafkaCluster(ctx context.Context, c *Client, environmentId, clusterId string) (string, error) {
	cluster, _, err := executeKafkaRead(c.cmkApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return "", fmt.Errorf("error reading Kafka Cluster %q: %s", clusterId, createDescriptiveError(err))
	}
	if restEndpoint := cluster.Spec.GetHttpEndpoint(); len(restEndpoint) > 0 {
		return restEndpoint, nil
	} else {
		return "", fmt.Errorf("rest_endpoint is nil or empty for Kafka Cluster %q", clusterId)
	}
}

// Send a GetCluster request to SRCM API to find out rest_endpoint for a given (environmentId, clusterId) pair
func fetchHttpEndpointOfSchemaRegistryCluster(ctx context.Context, c *Client, environmentId, clusterId string) (string, error) {
	cluster, _, err := executeSchemaRegistryClusterRead(c.srcmApiContext(ctx), c, environmentId, clusterId)
	if err != nil {
		return "", fmt.Errorf("error reading Schema Registry Cluster %q: %s", clusterId, createDescriptiveError(err))
	}
	if restEndpoint := cluster.Spec.GetHttpEndpoint(); len(restEndpoint) > 0 {
		return restEndpoint, nil
	} else {
		return "", fmt.Errorf("rest_endpoint is nil or empty for Schema Registry Cluster %q", clusterId)
	}
}

func isKafkaApiKey(apiKey apikeys.IamV2ApiKey) bool {
	return apiKey.Spec.Resource.GetKind() == clusterKind && apiKey.Spec.Resource.GetApiVersion() == cmkApiVersion
}

func isSchemaRegistryApiKey(apiKey apikeys.IamV2ApiKey) bool {
	// At the moment, API Key Mgmt API temporarily returns schemaRegistryKind instead of clusterKind
	return (apiKey.Spec.Resource.GetKind() == clusterKind || apiKey.Spec.Resource.GetKind() == schemaRegistryKind) && apiKey.Spec.Resource.GetApiVersion() == srcmApiVersion
}

func isKsqlDbClusterApiKey(apiKey apikeys.IamV2ApiKey) bool {
	// At the moment, API Key Mgmt API temporarily returns ksqlDbKind instead of clusterKind
	return (apiKey.Spec.Resource.GetKind() == clusterKind || apiKey.Spec.Resource.GetKind() == ksqlDbKind) && apiKey.Spec.Resource.GetApiVersion() == ksqldbcmApiVersion
}

func waitForApiKeyToSync(ctx context.Context, c *Client, createdApiKey apikeys.IamV2ApiKey, isResourceSpecificApiKey bool, environmentId string) error {
	// For Kafka API Key use Kafka REST API's List Topics request and wait for http.StatusOK
	// For Cloud API Key use Org API's List Environments request and wait for http.StatusOK

	if isResourceSpecificApiKey {
		if isKafkaApiKey(createdApiKey) {
			clusterId := createdApiKey.Spec.Resource.GetId()
			restEndpoint, err := fetchHttpEndpointOfKafkaCluster(ctx, c, environmentId, clusterId)
			if err != nil {
				return fmt.Errorf("error fetching Kafka Cluster %q's %q attribute: %s", clusterId, paramRestEndpoint, createDescriptiveError(err))
			}
			kafkaRestClient := c.kafkaRestClientFactory.CreateKafkaRestClient(restEndpoint, clusterId, createdApiKey.GetId(), createdApiKey.Spec.GetSecret(), false, false)
			if err := waitForCreatedKafkaApiKeyToSync(ctx, kafkaRestClient); err != nil {
				return fmt.Errorf("error waiting for Kafka API Key %q to sync: %s", createdApiKey.GetId(), createDescriptiveError(err))
			}
		} else if isSchemaRegistryApiKey(createdApiKey) {
			clusterId := createdApiKey.Spec.Resource.GetId()
			restEndpoint, err := fetchHttpEndpointOfSchemaRegistryCluster(ctx, c, environmentId, clusterId)
			if err != nil {
				return fmt.Errorf("error fetching Schema Registry Cluster %q's %q attribute: %s", clusterId, paramRestEndpoint, createDescriptiveError(err))
			}
			schemaRegistryRestClient := c.schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(restEndpoint, clusterId, createdApiKey.GetId(), createdApiKey.Spec.GetSecret(), false)
			if err := waitForCreatedSchemaRegistryApiKeyToSync(ctx, schemaRegistryRestClient); err != nil {
				return fmt.Errorf("error waiting for Schema Registry API Key %q to sync: %s", createdApiKey.GetId(), createDescriptiveError(err))
			}
		} else if isKsqlDbClusterApiKey(createdApiKey) {
			// Currently, there are no data plane API for ksqlDB clusters so there is no endpoint we could leverage
			// to check whether the Cluster API Key is synced which is why we're adding time.Sleep() here.
			// TODO: SVCF-3560
			time.Sleep(5 * time.Minute)
		} else {
			resourceJson, err := json.Marshal(createdApiKey.Spec.GetResource())
			if err != nil {
				return fmt.Errorf("unexpected API Key %q's resource: error marshaling %#v to json: %s", createdApiKey.GetId(), createdApiKey.Spec.GetResource(), createDescriptiveError(err))
			}
			return fmt.Errorf("unexpected API Key %q's resource: %s", createdApiKey.GetId(), resourceJson)
		}
	} else {
		// Cloud API Key
		if err := waitForCreatedCloudApiKeyToSync(ctx, c, createdApiKey.GetId(), createdApiKey.Spec.GetSecret()); err != nil {
			return fmt.Errorf("error waiting for Cloud API Key %q to sync: %s", createdApiKey.GetId(), createDescriptiveError(err))
		}
	}
	return nil
}

func apiKeyImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})

	apiKeySecret := os.Getenv("API_KEY_SECRET")
	if apiKeySecret == "" {
		return nil, fmt.Errorf("error importing API Key %q: API_KEY_SECRET environment variable is empty but it must be set", d.Id())
	}

	envIdAndClusterAPIKeyId := d.Id()
	parts := strings.Split(envIdAndClusterAPIKeyId, "/")
	if len(parts) == 1 {
		tflog.Debug(ctx, fmt.Sprintf("Importing Cloud API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	} else if len(parts) == 2 {
		environmentId := parts[0]
		clusterApiKeyId := parts[1]

		d.SetId(clusterApiKeyId)
		// Preset environmentId when importing Cluster API Key
		if err := d.Set(paramResource, []interface{}{map[string]interface{}{
			paramEnvironment: []interface{}{map[string]interface{}{
				paramId: environmentId,
			}},
		}}); err != nil {
			return nil, createDescriptiveError(err)
		}

		tflog.Debug(ctx, fmt.Sprintf("Importing Cluster API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	} else {
		return nil, fmt.Errorf("error importing API Key: invalid format: expected '<API Key ID> for Cloud API Key or <env ID>/API Key ID>' for Cluster API Key")
	}

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := apiKeyRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing API Key %q: %s", d.Id(), diagnostics[0].Summary)
	}
	if err := d.Set(paramSecret, apiKeySecret); err != nil {
		return nil, createDescriptiveError(err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing API Key %q", d.Id()), map[string]interface{}{apiKeyLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
