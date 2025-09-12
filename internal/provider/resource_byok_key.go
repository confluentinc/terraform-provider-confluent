package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	byok "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
)

const (
	paramAzureKeyId         = "key_identifier"
	paramAzureKeyVaultId    = "key_vault_id"
	paramAzureTenantId      = "tenant_id"
	paramAzureApplicationId = "application_id"
	paramGcpSecurityGroup   = "security_group"

	paramAwsKeyArn = "key_arn"
	paramGcpKeyId  = "key_id"
	paramAwsRoles  = "roles"

	kindAws   = "AwsKey"
	kindAzure = "AzureKey"
	kindGcp   = "GcpKey"
)

var ()

func byokResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: byokCreate,
		ReadContext:   byokRead,
		DeleteContext: byokDelete,
		Importer: &schema.ResourceImporter{
			StateContext: byokImport,
		},
		Schema: map[string]*schema.Schema{
			paramAws:   awsKeySchema(),
			paramAzure: azureKeySchema(),
			paramGcp:   gcpKeySchema(),
		},
	}
}

func awsKeySchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAwsKeyArn: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramAwsRoles: {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Computed: true,
				},
			},
		},
		ForceNew: true,
		Optional: true,
		Computed: true,
		MinItems: 1,
		MaxItems: 1,
	}
}

func gcpKeySchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramGcpKeyId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramGcpSecurityGroup: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		ForceNew: true,
		Optional: true,
		Computed: true,
		MinItems: 1,
		MaxItems: 1,
	}
}

func azureKeySchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAzureKeyId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramAzureKeyVaultId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramAzureTenantId: {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				paramAzureApplicationId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		ForceNew: true,
		Optional: true,
		Computed: true,
		MinItems: 1,
		MaxItems: 1,
	}
}

func byokCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	createByokKeyRequest := byok.NewByokV1Key()
	_, isAwsKey := d.GetOk(paramAws)
	_, isAzureKey := d.GetOk(paramAzure)
	_, isGcpKey := d.GetOk(paramGcp)

	var key string

	switch {
	case isAwsKey:
		key = extractStringValueFromBlock(d, paramAws, paramAwsKeyArn)
		byokKeyOneOf := byok.ByokV1AwsKeyAsByokV1KeyKeyOneOf(byok.NewByokV1AwsKey(key, kindAws))
		createByokKeyRequest.SetKey(byokKeyOneOf)

	case isAzureKey:
		key = extractStringValueFromBlock(d, paramAzure, paramAzureKeyId)
		keyVaultId := extractStringValueFromBlock(d, paramAzure, paramAzureKeyVaultId)
		tenantId := extractStringValueFromBlock(d, paramAzure, paramAzureTenantId)
		byokKeyOneOf := byok.ByokV1AzureKeyAsByokV1KeyKeyOneOf(byok.NewByokV1AzureKey(key, keyVaultId, kindAzure, tenantId))
		createByokKeyRequest.SetKey(byokKeyOneOf)

	case isGcpKey:
		key = extractStringValueFromBlock(d, paramGcp, paramGcpKeyId)
		byokKeyOneOf := byok.ByokV1GcpKeyAsByokV1KeyKeyOneOf(byok.NewByokV1GcpKey(key, kindGcp))
		createByokKeyRequest.SetKey(byokKeyOneOf)
	default:
		return diag.Errorf("error creating BYOK Key: expected one of %s, %s, %s params", paramAws, paramAzure, paramGcp)
	}

	createByokKeyRequestJson, err := json.Marshal(createByokKeyRequest)
	if err != nil {
		return diag.Errorf("error creating BYOK Key: error marshaling %#v to json: %s", createByokKeyRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new BYOK Key: %s", createByokKeyRequestJson))

	createdKey, resp, err := executeKeyCreate(ctx, c, *createByokKeyRequest)
	if err != nil {
		return diag.Errorf("error creating BYOK Key %q: %s", key, createDescriptiveError(err, resp))
	}
	d.SetId(createdKey.GetId())

	createdKeyJson, err := json.Marshal(createdKey)
	if err != nil {
		return diag.Errorf("error creating BYOK Key %q: error marshaling %#v to json: %s", d.Id(), createdKey, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating BYOK BYOK Key %q: %s", d.Id(), createdKeyJson), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	return byokRead(ctx, d, meta)
}

func byokDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	c := meta.(*Client)

	req := c.byokClient.KeysByokV1Api.DeleteByokV1Key(c.byokApiContext(ctx), d.Id())
	resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error deleting BYOK Key %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	return nil
}

func byokRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})
	if _, err := readKeyAndSetAttributes(ctx, d, meta); err != nil {
		return diag.FromErr(fmt.Errorf("error reading BYOK Key %q: %s", d.Id(), createDescriptiveError(err)))
	}
	return nil
}

func executeKeyCreate(ctx context.Context, c *Client, key byok.ByokV1Key) (byok.ByokV1Key, *http.Response, error) {
	req := c.byokClient.KeysByokV1Api.CreateByokV1Key(c.byokApiContext(ctx)).ByokV1Key(key)
	return req.Execute()
}

func executeKeyRead(ctx context.Context, c *Client, id string) (byok.ByokV1Key, *http.Response, error) {
	req := c.byokClient.KeysByokV1Api.GetByokV1Key(c.byokApiContext(ctx), id)
	return req.Execute()
}

func readKeyAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	key, resp, err := executeKeyRead(ctx, c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading BYOK Key %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{byokKeyLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing BYOK Key %q in TF state because BYOK Key could not be found on the server", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}
		return nil, fmt.Errorf("error reading BYOK Key %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	keyJson, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("error reading BYOK Key %q: error marshaling %#v to json: %s", key.GetId(), key, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched BYOK Key %q: %s", d.Id(), keyJson), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	if _, err := setKeyAttributes(d, key); err != nil {
		return nil, fmt.Errorf("error setting BYOK Key attributes %q: %s", d.Id(), createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished reading BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setKeyAttributes(d *schema.ResourceData, byokKey byok.ByokV1Key) (*schema.ResourceData, error) {
	oneOfKeys := byokKey.GetKey()

	switch {
	case oneOfKeys.ByokV1AzureKey != nil:
		if err := d.Set(paramAzure, []interface{}{map[string]interface{}{
			paramAzureApplicationId: oneOfKeys.ByokV1AzureKey.GetApplicationId(),
			paramAzureKeyId:         oneOfKeys.ByokV1AzureKey.GetKeyId(),
			paramAzureKeyVaultId:    oneOfKeys.ByokV1AzureKey.GetKeyVaultId(),
			paramAzureTenantId:      oneOfKeys.ByokV1AzureKey.GetTenantId(),
		}}); err != nil {
			return nil, err
		}
	case oneOfKeys.ByokV1AwsKey != nil:
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramAwsKeyArn: oneOfKeys.ByokV1AwsKey.GetKeyArn(),
			paramAwsRoles:  oneOfKeys.ByokV1AwsKey.GetRoles(),
		}}); err != nil {
			return nil, err
		}
	case oneOfKeys.ByokV1GcpKey != nil:
		if err := d.Set(paramGcp, []interface{}{map[string]interface{}{
			paramGcpKeyId:         oneOfKeys.ByokV1GcpKey.GetKeyId(),
			paramGcpSecurityGroup: oneOfKeys.ByokV1GcpKey.GetSecurityGroup(),
		}}); err != nil {
			return nil, err
		}
	}

	d.SetId(byokKey.GetId())
	return d, nil
}

func byokImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readKeyAndSetAttributes(ctx, d, meta); err != nil {
		return nil, fmt.Errorf("error importing BYOK Key %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing BYOK Key %q", d.Id()), map[string]interface{}{byokKeyLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
