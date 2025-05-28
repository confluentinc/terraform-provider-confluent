package provider

import (
	"context"
	"encoding/json"
	"fmt"
	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const (
	paramApiKey          = "api_key"
	paramAwsAccessKey    = "aws_access_key"
	paramAwsSecretKey    = "aws_secret_key"
	paramAwsSessionToken = "aws_session_token"
	paramServiceKey      = "service_key"
	paramUsername        = "username"
	paramPassword        = "password"
)

var (
	acceptedTypes               = []string{"OPENAI", "AZUREML", "AZUREOPENAI", "BEDROCK", "SAGEMAKER", "GOOGLEAI", "VERTEXAI", "MONGODB", "PINECONE", "ELASTIC", "COUCHBASE"}
	ConnectionTypeSecretMapping = map[string][]string{
		"OPENAI":      {"api_key"},
		"AZUREML":     {"api_key"},
		"AZUREOPENAI": {"api_key"},
		"BEDROCK":     {"aws_access_key", "aws_secret_key", "aws_session_token"},
		"SAGEMAKER":   {"aws_access_key", "aws_secret_key", "aws_session_token"},
		"GOOGLEAI":    {"api_key"},
		"VERTEXAI":    {"service_key"},
		"MONGODB":     {"username", "password"},
		"ELASTIC":     {"api_key"},
		"PINECONE":    {"api_key"},
		"COUCHBASE":   {"username", "password"},
	}

	ConnectionSecretTypeMapping = map[string][]string{
		"api_key":           {"OPENAI", "AZUREML", "AZUREOPENAI", "GOOGLEAI", "ELASTIC", "PINECONE"},
		"aws_access_key":    {"BEDROCK", "SAGEMAKER"},
		"aws_secret_key":    {"BEDROCK", "SAGEMAKER"},
		"aws_session_token": {"BEDROCK", "SAGEMAKER"},
		"service_key":       {"VERTEXAI"},
		"username":          {"MONGODB", "COUCHBASE"},
		"password":          {"MONGODB", "COUCHBASE"},
	}

	ConnectionRequiredSecretMapping = map[string][]string{
		"OPENAI":      {"api_key"},
		"AZUREML":     {"api_key"},
		"AZUREOPENAI": {"api_key"},
		"BEDROCK":     {"aws_access_key", "aws_secret_key"},
		"SAGEMAKER":   {"aws_access_key", "aws_secret_key"},
		"GOOGLEAI":    {"api_key"},
		"VERTEXAI":    {"service_key"},
		"MONGODB":     {"username", "password"},
		"ELASTIC":     {"api_key"},
		"PINECONE":    {"api_key"},
		"COUCHBASE":   {"username", "password"},
	}
	ConnectionSecretBackendKeyMapping = map[string]string{
		"api_key":           "API_KEY",
		"aws_access_key":    "AWS_ACCESS_KEY_ID",
		"aws_secret_key":    "AWS_SECRET_ACCESS_KEY",
		"aws_session_token": "AWS_SESSION_TOKEN",
		"service_key":       "SERVICE_KEY",
		"username":          "USERNAME",
		"password":          "PASSWORD",
	}
)

func flinkConnectionResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: connectionCreate,
		ReadContext:   connectionRead,
		UpdateContext: connectionUpdate,
		DeleteContext: connectionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: connectionImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique name of the Flink Connection per organization, environment scope.",
			},
			paramType: {
				Type:         schema.TypeString,
				Description:  "The type of the flink connection.",
				ValidateFunc: validation.StringInSlice(acceptedTypes, false),
				Required:     true,
			},
			paramEndpoint: {
				Type:         schema.TypeString,
				Description:  "The endpoint of the flink connection.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
			},
			paramApiKey: {
				Type:        schema.TypeString,
				Description: "API key for the type.",
				Optional:    true,
			},
			paramAwsAccessKey: {
				Type:        schema.TypeString,
				Description: "Access key for the type.",
				Optional:    true,
			},
			paramAwsSecretKey: {
				Type:        schema.TypeString,
				Description: "Secret key for the type.",
				Optional:    true,
			},
			paramAwsSessionToken: {
				Type:        schema.TypeString,
				Description: "Session token for the type.",
				Optional:    true,
			},
			paramServiceKey: {
				Type:        schema.TypeString,
				Description: "Service Key for the type.",
				Optional:    true,
			},
			paramUsername: {
				Type:        schema.TypeString,
				Description: "Username for the type.",
				Optional:    true,
			},
			paramPassword: {
				Type:        schema.TypeString,
				Description: "Password for the type.",
				Optional:    true,
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The schema version of this representation of a resource.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The object this REST resource represents.",
			},
			paramOrganization: optionalIdBlockSchema(),
			paramEnvironment:  optionalIdBlockSchema(),
			paramComputePool:  optionalIdBlockSchemaUpdatable(),
			paramPrincipal:    optionalIdBlockSchemaUpdatable(),
			paramRestEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The REST endpoint of the Flink Connection cluster, for example, `https://flink.us-east-1.aws.confluent.cloud/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123`).",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^http"), "the REST endpoint must start with 'https://'"),
			},
			paramCredentials: credentialsSchema(),
		},
	}
}

func connectionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	flinkRestClient, errClient := getFlinkClient(d, meta)
	if errClient != nil {
		return errClient
	}

	name := d.Get(paramDisplayName).(string)
	connectionType := d.Get(paramType).(string)
	endpoint := d.Get(paramEndpoint).(string)

	secretMap, err := validateConnectionSecrets(connectionType, d)
	if err != nil {
		return diag.Errorf("error creating Flink Conection: %s", createDescriptiveError(err))
	}

	secretData, err := json.Marshal(secretMap)
	if err != nil {
		return diag.Errorf("error creating Flink Conection: %s", createDescriptiveError(err))
	}

	connection := flinkgatewayv1.SqlV1Connection{
		Name: flinkgatewayv1.PtrString(name),
		Spec: &flinkgatewayv1.SqlV1ConnectionSpec{
			ConnectionType: flinkgatewayv1.PtrString(strings.ToUpper(connectionType)),
			Endpoint:       flinkgatewayv1.PtrString(endpoint),
			AuthData: &flinkgatewayv1.SqlV1ConnectionSpecAuthDataOneOf{
				SqlV1PlaintextProvider: &flinkgatewayv1.SqlV1PlaintextProvider{
					Kind: flinkgatewayv1.PtrString("PlaintextProvider"),
					Data: flinkgatewayv1.PtrString(string(secretData[:])),
				},
			},
		},
	}
	createdConnection, _, err := executeConnectionCreate(flinkRestClient.apiContext(ctx), flinkRestClient, connection)
	if err != nil {
		return diag.Errorf("error creating Flink Connection: %s", createDescriptiveError(err))
	}
	d.SetId(createFlinkConnectionId(flinkRestClient.organizationId, flinkRestClient.environmentId, createdConnection.GetName()))
	createdConnectionJson, err := json.Marshal(createdConnection)
	if err != nil {
		return diag.Errorf("error creating Flink Connection: error marshaling %#v to json: %s", createdConnectionJson, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Flink Connection %q: %s", d.Id(), createdConnectionJson), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
	return connectionRead(ctx, d, meta)
}

func connectionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramApiKey, paramAwsAccessKey, paramAwsSecretKey, paramAwsSessionToken, paramServiceKey, paramUsername, paramPassword) {
		return diag.Errorf("error updating Flink connection %q: only auth token attributes can be updated for Flink Connection", d.Id())
	}

	flinkRestClient, errClient := getFlinkClient(d, meta)
	if errClient != nil {
		return errClient
	}

	name := d.Get(paramDisplayName).(string)
	connectionType := d.Get(paramType).(string)
	endpoint := d.Get(paramEndpoint).(string)

	secretMap, err := validateConnectionSecrets(connectionType, d)
	if err != nil {
		return diag.Errorf("error updating Flink Connection: %s", createDescriptiveError(err))
	}
	secretData, err := json.Marshal(secretMap)
	if err != nil {
		return diag.Errorf("error updating Flink Conection: %s", createDescriptiveError(err))
	}
	connection := flinkgatewayv1.SqlV1Connection{
		Name: flinkgatewayv1.PtrString(name),
		Spec: &flinkgatewayv1.SqlV1ConnectionSpec{
			ConnectionType: flinkgatewayv1.PtrString(strings.ToUpper(connectionType)),
			Endpoint:       flinkgatewayv1.PtrString(endpoint),
			AuthData: &flinkgatewayv1.SqlV1ConnectionSpecAuthDataOneOf{
				SqlV1PlaintextProvider: &flinkgatewayv1.SqlV1PlaintextProvider{
					Kind: flinkgatewayv1.PtrString("PlaintextProvider"),
					Data: flinkgatewayv1.PtrString(string(secretData[:])),
				},
			},
		},
	}
	updateConnectionRequestJson, err := json.Marshal(connection)
	if err != nil {
		return diag.Errorf("error updating Flink Connection %q: error marshaling %#v to json: %s", d.Id(), connection, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Flink Connection %q: %s", d.Id(), updateConnectionRequestJson), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})

	req := flinkRestClient.apiClient.ConnectionsSqlV1Api.UpdateSqlv1Connection(flinkRestClient.apiContext(ctx), flinkRestClient.organizationId, flinkRestClient.environmentId, name).SqlV1Connection(connection)
	_, err = req.Execute()
	if err != nil {
		return diag.Errorf("error updating Flink Connection %q: %s", d.Id(), createDescriptiveError(err))
	}
	return connectionRead(ctx, d, meta)
}

func executeConnectionCreate(ctx context.Context, c *FlinkRestClient, connection flinkgatewayv1.SqlV1Connection) (flinkgatewayv1.SqlV1Connection, *http.Response, error) {
	req := c.apiClient.ConnectionsSqlV1Api.CreateSqlv1Connection(c.apiContext(ctx), c.organizationId, c.environmentId).SqlV1Connection(connection)
	return req.Execute()
}

func executeConnectionRead(ctx context.Context, c *FlinkRestClient, connectionName string) (flinkgatewayv1.SqlV1Connection, *http.Response, error) {
	req := c.apiClient.ConnectionsSqlV1Api.GetSqlv1Connection(ctx, c.organizationId, c.environmentId, connectionName)
	return req.Execute()
}

func connectionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
	flinkRestClient, errClient := getFlinkClient(d, meta)
	if errClient != nil {
		return errClient
	}
	connectionName, err := parseConnectionName(d.Id())
	if err != nil {
		return diag.Errorf("error reading Flink Connection: %s", createDescriptiveError(err))
	}
	if _, err := readConnectionAndSetAttributes(ctx, d, flinkRestClient, connectionName); err != nil {
		return diag.FromErr(fmt.Errorf("error reading flink connection %q: %s", d.Id(), createDescriptiveError(err)))
	}
	return nil
}

func readConnectionAndSetAttributes(ctx context.Context, d *schema.ResourceData, c *FlinkRestClient, connectionName string) ([]*schema.ResourceData, error) {
	connection, resp, err := executeConnectionRead(c.apiContext(ctx), c, connectionName)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Flink Connection %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Flink Connection %q in TF state because Flink Connection could not be found on the server", d.Id()), map[string]interface{}{flinkArtifactLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	connectionJson, err := json.Marshal(connection)
	if err != nil {
		return nil, fmt.Errorf("error reading flink connection %q: error marshaling %#v to json: %s", d.Id(), connection, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Flink Connection %q: %s", d.Id(), connectionJson), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})

	if _, err := setConnectionAttributes(d, connection, c); err != nil {
		return nil, createDescriptiveError(err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished reading Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setConnectionAttributes(d *schema.ResourceData, connection flinkgatewayv1.SqlV1Connection, c *FlinkRestClient) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, connection.GetName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramType, connection.Spec.GetConnectionType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEndpoint, connection.Spec.GetEndpoint()); err != nil {
		return nil, err
	}
	if err := d.Set(paramApiKey, d.Get(paramApiKey)); err != nil {
		return nil, err
	}
	if err := d.Set(paramAwsAccessKey, d.Get(paramAwsAccessKey)); err != nil {
		return nil, err
	}
	if err := d.Set(paramAwsSessionToken, d.Get(paramAwsSessionToken)); err != nil {
		return nil, err
	}
	if err := d.Set(paramAwsSecretKey, d.Get(paramAwsSecretKey)); err != nil {
		return nil, err
	}
	if err := d.Set(paramServiceKey, d.Get(paramServiceKey)); err != nil {
		return nil, err
	}
	if err := d.Set(paramUsername, d.Get(paramUsername)); err != nil {
		return nil, err
	}
	if err := d.Set(paramPassword, d.Get(paramPassword)); err != nil {
		return nil, err
	}
	if err := d.Set(paramApiVersion, connection.GetApiVersion()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramKind, connection.GetKind()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if !c.isMetadataSetInProviderBlock {
		if err := setKafkaCredentials(c.flinkApiKey, c.flinkApiSecret, d, c.externalAccessToken != nil); err != nil {
			return nil, err
		}
		if err := d.Set(paramRestEndpoint, c.restEndpoint); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramOrganization, paramId, c.organizationId, d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, c.environmentId, d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramComputePool, paramId, c.computePoolId, d); err != nil {
			return nil, err
		}
		if err := setStringAttributeInListBlockOfSizeOne(paramPrincipal, paramId, c.principalId, d); err != nil {
			return nil, err
		}
	}
	d.SetId(createFlinkConnectionId(c.organizationId, c.environmentId, connection.GetName()))
	return d, nil
}

func connectionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})

	flinkRestClient, errClient := getFlinkClient(d, meta)
	if errClient != nil {
		return errClient
	}

	name := d.Get(paramDisplayName).(string)
	req := flinkRestClient.apiClient.ConnectionsSqlV1Api.DeleteSqlv1Connection(flinkRestClient.apiContext(ctx), flinkRestClient.organizationId, flinkRestClient.environmentId, name)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting flink connection %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
	return nil
}

func connectionImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})

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
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)

	orgIdEnvIDAndConnectionPoolId := d.Id()
	parts := strings.Split(orgIdEnvIDAndConnectionPoolId, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("error importing Flink Connection: invalid format: expected '<Organization ID>/<Environment ID>/<Flink Connection ID>'")
	}

	organizationId = parts[0]
	environmentId = parts[1]
	connectionName := parts[2]

	d.SetId(createFlinkConnectionId(organizationId, environmentId, connectionName))
	setImportData(d)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readConnectionAndSetAttributes(ctx, d, flinkRestClient, connectionName); err != nil {
		return nil, fmt.Errorf("error importing Flink Connection %q: %s", d.Id(), createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Flink Connection %q", d.Id()), map[string]interface{}{flinkConnectionLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func getFlinkClient(d *schema.ResourceData, meta interface{}) (*FlinkRestClient, diag.Diagnostics) {
	restEndpoint, err := extractFlinkRestEndpoint(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	organizationId, err := extractFlinkOrganizationId(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	environmentId, err := extractFlinkEnvironmentId(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	computePoolId, err := extractFlinkComputePoolId(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	principalId, err := extractFlinkPrincipalId(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkApiKey, flinkApiSecret, err := extractFlinkApiKeyAndApiSecret(meta.(*Client), d, false)
	if err != nil {
		return nil, diag.Errorf("error creating Flink Statement: %s", createDescriptiveError(err))
	}
	flinkRestClient := meta.(*Client).flinkRestClientFactory.CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret, meta.(*Client).isFlinkMetadataSet, meta.(*Client).oauthToken)
	return flinkRestClient, nil
}

func validateConnectionSecrets(connectionType string, d *schema.ResourceData) (map[string]string, error) {
	var connectionSecrets []string
	connectionSecrets = append(connectionSecrets, ConnectionTypeSecretMapping[connectionType]...)

	for key := range ConnectionSecretTypeMapping {
		secret := d.Get(key)
		if secret != "" && !slices.Contains(connectionSecrets, key) {
			return nil, fmt.Errorf("%s is invalid for connection %s", key, connectionType)
		}
	}

	requiredSecretKeys := ConnectionRequiredSecretMapping[connectionType]
	var optionalSecretKeys []string
	for _, secretKey := range ConnectionTypeSecretMapping[connectionType] {
		if !slices.Contains(requiredSecretKeys, secretKey) {
			optionalSecretKeys = append(optionalSecretKeys, secretKey)
		}
	}

	secretMap := map[string]string{}
	for _, requiredKey := range requiredSecretKeys {
		secret := d.Get(requiredKey).(string)
		if secret == "" {
			return nil, fmt.Errorf("must provide %s for type %s", requiredKey, connectionType)
		}
		backendKey, ok := ConnectionSecretBackendKeyMapping[requiredKey]
		if !ok {
			return nil, fmt.Errorf(`backend key not found for "%s"`, requiredKey)
		}
		secretMap[backendKey] = secret
	}

	for _, optionalSecretKey := range optionalSecretKeys {
		secret := d.Get(optionalSecretKey).(string)
		backendKey, ok := ConnectionSecretBackendKeyMapping[optionalSecretKey]
		if !ok {
			return nil, fmt.Errorf("backend key not found for %s", optionalSecretKey)
		}

		if secret != "" {
			secretMap[backendKey] = secret
		}
	}
	return secretMap, nil
}

func setImportData(d *schema.ResourceData) {
	err := d.Set(paramApiKey, getEnv("API_KEY", ""))
	if err != nil {
		return
	}
	err = d.Set(paramAwsAccessKey, getEnv("AWS_ACCESS_KEY_ID_CONNECTION", ""))
	if err != nil {
		return
	}
	err = d.Set(paramAwsSecretKey, getEnv("AWS_SECRET_ACCESS_KEY_CONNECTION", ""))
	if err != nil {
		return
	}
	err = d.Set(paramAwsSessionToken, getEnv("AWS_SESSION_TOKEN_CONNECTION", ""))
	if err != nil {
		return
	}
	err = d.Set(paramServiceKey, getEnv("SERVICE_KEY", ""))
	if err != nil {
		return
	}
	err = d.Set(paramUsername, getEnv("USERNAME", ""))
	if err != nil {
		return
	}
	err = d.Set(paramPassword, getEnv("PASSWORD", ""))
	if err != nil {
		return
	}
}

func createFlinkConnectionId(orgId, environmentId, connectionName string) string {
	return fmt.Sprintf("%s/%s/%s", orgId, environmentId, connectionName)
}

func parseConnectionName(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid ID format: expected '<Organization ID>/<Environment ID>/<Connection name>'")
	}
	return parts[2], nil
}
