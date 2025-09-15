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
	"fmt"
	"strings"

	piv2 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	// Authorization resource attributes
	paramProviderIntegrationIdAuth = "provider_integration_id"

	// Azure-specific attributes
	paramAzureAuth                      = "azure"
	paramAzureCustomerTenantId          = "customer_azure_tenant_id"
	paramAzureConfluentMultiTenantAppId = "confluent_multi_tenant_app_id"

	// GCP-specific attributes
	paramGcpAuth                   = "gcp"
	paramGcpCustomerServiceAccount = "customer_google_service_account"
	paramGcpGoogleServiceAccount   = "google_service_account"
)

// providerIntegrationV2AuthorizationResource defines the authorization resource for PIM v2 integrations.
// This resource configures customer cloud provider settings (Azure tenant, GCP service account) and validates the integration.
func providerIntegrationV2AuthorizationResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: providerIntegrationV2AuthorizationCreate,
		ReadContext:   providerIntegrationV2AuthorizationRead,
		UpdateContext: providerIntegrationV2AuthorizationUpdate,
		DeleteContext: providerIntegrationV2AuthorizationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: providerIntegrationV2AuthorizationImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID for provider integration authorization.",
			},
			paramProviderIntegrationIdAuth: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The ID of the provider integration to authorize.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramAzureAuth: azureAuthConfigSchema(),
			paramGcpAuth:   gcpAuthConfigSchema(),

			paramEnvironment: environmentSchema(),
		},
	}
}

func azureAuthConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MinItems:    0,
		MaxItems:    1,
		Description: "Azure-specific configuration for the provider integration authorization.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			paramAzureCustomerTenantId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Customer's Azure Tenant ID.",
			},
			paramAzureConfluentMultiTenantAppId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Confluent Multi-Tenant App ID used to access customer Azure resources.",
			},
		}},
	}
}

func gcpAuthConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		MinItems:    0,
		MaxItems:    1,
		Description: "GCP-specific configuration for the provider integration authorization.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			paramGcpCustomerServiceAccount: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Customer's Google Service Account that Confluent Cloud impersonates.",
			},
			paramGcpGoogleServiceAccount: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Google Service Account that Confluent Cloud uses for impersonation.",
			},
		}},
	}
}

func providerIntegrationV2AuthorizationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	integrationId := d.Get(paramProviderIntegrationIdAuth).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Info(ctx, fmt.Sprintf("Starting authorization Create for integration %s", integrationId))

	// First, read the integration to get current state
	req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), integrationId).Environment(environmentId)
	integration, _, err := req.Execute()
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	// Check if integration is in valid status for authorization (DRAFT or CREATED)
	status := integration.GetStatus()
	if status != "DRAFT" && status != "CREATED" {
		return diag.Errorf("Provider integration %q must be in DRAFT or CREATED status for authorization, current status: %s", integrationId, status)
	}

	// Build config based on provider type
	var updateConfig piv2.PimV2IntegrationUpdateConfigOneOf
	provider := integration.GetProvider()

	switch provider {
	case providerAzure:
		azureConfig := d.Get(paramAzureAuth).([]interface{})
		if len(azureConfig) == 0 {
			return diag.Errorf("Azure configuration is required for provider %s", provider)
		}
		azureMap := azureConfig[0].(map[string]interface{})
		customerTenantId := azureMap[paramAzureCustomerTenantId].(string)

		azureIntegrationConfig := &piv2.PimV2AzureIntegrationConfig{
			Kind:                  "AzureIntegrationConfig",
			CustomerAzureTenantId: &customerTenantId,
		}
		updateConfig = piv2.PimV2AzureIntegrationConfigAsPimV2IntegrationUpdateConfigOneOf(azureIntegrationConfig)

	case providerGcp:
		gcpConfig := d.Get(paramGcpAuth).([]interface{})
		if len(gcpConfig) == 0 {
			return diag.Errorf("GCP configuration is required for provider %s", provider)
		}
		gcpMap := gcpConfig[0].(map[string]interface{})
		customerServiceAccount := gcpMap[paramGcpCustomerServiceAccount].(string)

		gcpIntegrationConfig := &piv2.PimV2GcpIntegrationConfig{
			Kind:                         "GcpIntegrationConfig",
			CustomerGoogleServiceAccount: &customerServiceAccount,
		}
		updateConfig = piv2.PimV2GcpIntegrationConfigAsPimV2IntegrationUpdateConfigOneOf(gcpIntegrationConfig)

	default:
		return diag.Errorf("Unsupported provider: %s", provider)
	}

	var updatedIntegration *piv2.PimV2Integration

	// Only PATCH if integration is still in DRAFT status
	if status == "DRAFT" {
		// PATCH the integration with the customer configuration to change status from DRAFT to CREATED
		tflog.Info(ctx, fmt.Sprintf("Updating provider integration %q with customer configuration", integrationId))

		updateReq := piv2.PimV2IntegrationUpdate{}
		updateReq.SetConfig(updateConfig)
		updateReq.SetEnvironment(piv2.ObjectReference{Id: environmentId})

		patchApiReq := c.piV2Client.IntegrationsPimV2Api.UpdatePimV2Integration(c.piV2ApiContext(ctx), integrationId).PimV2IntegrationUpdate(updateReq)
		patchedIntegration, _, err := patchApiReq.Execute()
		if err != nil {
			return diag.Errorf("Failed to update provider integration %q with customer configuration: %s", integrationId, createDescriptiveError(err))
		}
		updatedIntegration = &patchedIntegration
		tflog.Info(ctx, fmt.Sprintf("Successfully updated provider integration %q. Status: %s", integrationId, updatedIntegration.GetStatus()))
	} else {
		// Integration is already CREATED, skip PATCH and use existing integration
		tflog.Info(ctx, fmt.Sprintf("Integration %q is already in CREATED status, skipping configuration update", integrationId))
		updatedIntegration = &integration
	}

	// Set the resource ID and populate data first (so we get setup information even if validation fails)
	d.SetId(integrationId)

	// Read the integration data to populate outputs (including multi-tenant app ID)
	readDiags := providerIntegrationV2AuthorizationRead(ctx, d, meta)
	if readDiags.HasError() {
		return readDiags
	}

	// Always validate the integration
	return validateIntegrationSetup(ctx, c, integrationId, environmentId, status, updateConfig)
}

func providerIntegrationV2AuthorizationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.Id() == "" {
		return nil
	}

	c := meta.(*Client)
	integrationId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), integrationId).Environment(environmentId)
	integration, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading provider integration v2 authorization %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing provider integration v2 authorization %q in TF state because it could not be found on the server", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(createDescriptiveError(err))
	}

	// Set provider integration ID
	if err := d.Set(paramProviderIntegrationIdAuth, integration.GetId()); err != nil {
		return diag.FromErr(err)
	}

	// Set config based on provider type
	if integration.Config != nil {
		if integration.Config.PimV2AzureIntegrationConfig != nil {
			azureConfig := integration.Config.PimV2AzureIntegrationConfig
			if err := d.Set(paramAzureAuth, []interface{}{map[string]interface{}{
				paramAzureCustomerTenantId:          azureConfig.GetCustomerAzureTenantId(),
				paramAzureConfluentMultiTenantAppId: azureConfig.GetConfluentMultiTenantAppId(),
			}}); err != nil {
				return diag.FromErr(err)
			}
		}

		if integration.Config.PimV2GcpIntegrationConfig != nil {
			gcpConfig := integration.Config.PimV2GcpIntegrationConfig
			if err := d.Set(paramGcpAuth, []interface{}{map[string]interface{}{
				paramGcpCustomerServiceAccount: gcpConfig.GetCustomerGoogleServiceAccount(),
				paramGcpGoogleServiceAccount:   gcpConfig.GetGoogleServiceAccount(),
			}}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, integration.Environment.GetId(), d); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(integration.GetId())

	// Always validate to show warnings if cloud provider setup is incomplete
	return validateIntegrationSetup(ctx, c, integration.GetId(), environmentId, integration.GetStatus(), piv2.PimV2IntegrationUpdateConfigOneOf{})
}

func providerIntegrationV2AuthorizationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// For authorization resource, update means re-patching the config and re-validating
	tflog.Info(ctx, "UPDATE function called, delegating to CREATE")
	return providerIntegrationV2AuthorizationCreate(ctx, d, meta)
}

func providerIntegrationV2AuthorizationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Authorization resource delete doesn't delete the integration itself, only removes from Terraform state
	// The integration remains in CREATED status and continues to work
	tflog.Debug(ctx, fmt.Sprintf("Removing provider integration v2 authorization %q from TF state (integration itself remains)", d.Id()))
	return nil
}

func providerIntegrationV2AuthorizationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Expect format: <env-id>/<integration-id>
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing provider integration v2 authorization: invalid format: expected '<env ID>/<integration ID>'")
	}
	environmentId := parts[0]
	integrationId := parts[1]
	d.SetId(integrationId)
	if err := d.Set(paramProviderIntegrationIdAuth, integrationId); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, environmentId, d); err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("Imported provider integration v2 authorization %q (environment %q)", d.Id(), environmentId))
	return []*schema.ResourceData{d}, nil
}

// validateIntegrationSetup validates the integration configuration and returns warnings if setup is incomplete
func validateIntegrationSetup(ctx context.Context, c *Client, integrationId, environmentId, status string, updateConfig piv2.PimV2IntegrationUpdateConfigOneOf) diag.Diagnostics {
	tflog.Info(ctx, fmt.Sprintf("Validating integration %q configuration...", integrationId))

	// Build validate request with correct types
	validateReq := piv2.PimV2IntegrationValidateRequest{}
	validateReq.SetId(integrationId)
	validateReq.SetEnvironment(piv2.GlobalObjectReference{Id: environmentId})

	// For CREATED integrations, don't send config (it's already set)
	// For DRAFT integrations that were just patched, use the config from the update
	if status == "DRAFT" && (updateConfig.PimV2AzureIntegrationConfig != nil || updateConfig.PimV2GcpIntegrationConfig != nil) {
		// Convert config to validate request config type
		var validateConfig piv2.PimV2IntegrationValidateRequestConfigOneOf
		if updateConfig.PimV2AzureIntegrationConfig != nil {
			validateConfig.PimV2AzureIntegrationConfig = updateConfig.PimV2AzureIntegrationConfig
		}
		if updateConfig.PimV2GcpIntegrationConfig != nil {
			validateConfig.PimV2GcpIntegrationConfig = updateConfig.PimV2GcpIntegrationConfig
		}
		validateReq.SetConfig(validateConfig)
	}
	// For CREATED integrations, don't set config - the integration already has it

	validateApiReq := c.piV2Client.IntegrationsPimV2Api.ValidatePimV2Integration(c.piV2ApiContext(ctx)).PimV2IntegrationValidateRequest(validateReq)
	_, err := validateApiReq.Execute()
	if err != nil {
		// Return a warning instead of an error so the resource is created successfully
		// This allows the customer to get setup information they need
		tflog.Warn(ctx, fmt.Sprintf("Validation failed for integration %q: %s", integrationId, createDescriptiveError(err)))

		// Read the integration to determine the provider type for appropriate warning
		req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), integrationId).Environment(environmentId)
		integration, _, readErr := req.Execute()
		if readErr != nil {
			return diag.FromErr(createDescriptiveError(readErr))
		}

		// Generate provider-specific warning message
		provider := integration.GetProvider()
		switch provider {
		case providerAzure:
			return createAzureSetupWarning(integration)
		case providerGcp:
			return createGcpSetupWarning(integration)
		default:
			return diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "⏳ Cloud provider setup required",
					Detail:   fmt.Sprintf("Integration created successfully! Complete setup for %s provider and re-run 'terraform apply' to validate the connection.", provider),
				},
			}
		}
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully validated provider integration %q configuration", integrationId))
	return nil
}

// createAzureSetupWarning creates an Azure-specific setup warning with detailed instructions
func createAzureSetupWarning(integration piv2.PimV2Integration) diag.Diagnostics {
	appId := "unknown"
	if integration.Config != nil && integration.Config.PimV2AzureIntegrationConfig != nil {
		appId = integration.Config.PimV2AzureIntegrationConfig.GetConfluentMultiTenantAppId()
	}

	return diag.Diagnostics{
		{
			Severity: diag.Warning,
			Summary:  "⏳ Azure setup required",
			Detail:   fmt.Sprintf("Integration created successfully! Complete Azure setup:\n\n1. Run: az ad sp create --id %s\n2. Check Azure Portal → Enterprise Applications to ensure it appears\n3. Grant necessary permissions to the service principal\n4. Re-run 'terraform apply' to validate", appId),
		},
	}
}

// createGcpSetupWarning creates a GCP-specific setup warning with detailed instructions
func createGcpSetupWarning(integration piv2.PimV2Integration) diag.Diagnostics {
	confluentServiceAccount := "unknown"
	customerServiceAccount := "unknown"
	if integration.Config != nil && integration.Config.PimV2GcpIntegrationConfig != nil {
		confluentServiceAccount = integration.Config.PimV2GcpIntegrationConfig.GetGoogleServiceAccount()
		customerServiceAccount = integration.Config.PimV2GcpIntegrationConfig.GetCustomerGoogleServiceAccount()
	}

	return diag.Diagnostics{
		{
			Severity: diag.Warning,
			Summary:  "⏳ GCP setup required",
			Detail:   fmt.Sprintf("Integration created successfully! Complete GCP IAM setup:\n\n1. Grant Service Account Token Creator role:\n   gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \\\n     --member=\"serviceAccount:%s\" \\\n     --role=\"roles/iam.serviceAccountTokenCreator\" \\\n     --condition=\"expression=request.auth.claims.sub=='%s'\"\n\n2. Grant your service account (%s) necessary permissions:\n   • BigQuery: bigquery.datasets.get, bigquery.tables.*\n   • Storage: storage.objects.*, storage.buckets.get\n\n3. Re-run 'terraform apply' to validate\n\nNote: IAM changes may take 1-7 minutes to propagate.", confluentServiceAccount, confluentServiceAccount, customerServiceAccount),
		},
	}
}
