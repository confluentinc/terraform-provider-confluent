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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	zclCty "github.com/zclconf/go-cty/cty"
)

const (
	tfConfigurationFileName = "main.tf"
	tfStateFileName         = "terraform.tfstate"
	tfLockFileName          = ".terraform.lock.hcl"
	paramResources          = "resources"
	paramOutputPath         = "output_path"
	defaultTfStateFile      = "terraform.tfstate"
	defaultVariablesTfFile  = "variables.tf"
	defaultOutputPath       = "./imported_confluent_infrastructure"
)

type ImporterMode int

const (
	Cloud ImporterMode = iota
	Kafka
	SchemaRegistry
)

var ImportableResources = []string{
	// Cloud
	"confluent_service_account",
	"confluent_kafka_cluster",
	"confluent_environment",
	"confluent_connector",

	// Kafka
	"confluent_kafka_topic",
	"confluent_kafka_acl",

	// Schema Registry
	"confluent_schema",
}

func tfImporterResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: tfImporterCreate,
		ReadContext:   tfImporterRead,
		DeleteContext: tfImporterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			paramResources: {
				Description: "A list of resources to Import. Defaults to all Importable resources.",
				Type:        schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(ImportableResources, false),
				},
				Optional: true,
				ForceNew: true,
			},
			paramOutputPath: {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  defaultOutputPath,
			},
			// TODO: add paramFormat = HCL (default) | JSON?
		},
	}
}

func tfImporterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	outputPath := d.Get(paramOutputPath).(string)
	resourcesToImport := convertToStringSlice(d.Get(paramResources).([]interface{}))

	importerMode := Cloud

	client := meta.(*Client)
	if client.isKafkaMetadataSet && client.isKafkaClusterIdSet {
		importerMode = Kafka
	} else if client.isSchemaRegistryMetadataSet {
		importerMode = SchemaRegistry
	}

	overrideUserAgent(client)

	tflog.Debug(ctx, fmt.Sprintf("Creating TF Importer %q", outputPath), map[string]interface{}{tfImporterLoggingKey: outputPath})

	instances, err := loadAllInstances(ctx, resourcesToImport, outputPath, importerMode, meta)
	if err != nil {
		return err
	}

	// Generate JSON: {"confluent_service_account": {"test_12345": {state}}}
	resourceJsonMaps := make(map[string]map[string]map[string]interface{})
	resourceHclBlocks := make([][]byte, 0)
	for i := range instances {
		if resourceJsonMaps[instances[i].ResourceName] == nil {
			resourceJsonMaps[instances[i].ResourceName] = make(map[string]map[string]interface{})
		}

		if len(resourceJsonMaps[instances[i].ResourceName]) > 0 {
			nextIndex := len(resourceJsonMaps[instances[i].ResourceName]) + 1
			instances[i].Name = instances[i].Name + "_" + strconv.Itoa(nextIndex)
		}

		jsonResult, err := instanceStateToJson(instances[i].State, instances[i].ComputedOnlyProperties, instances[i].CtyType)
		if err != nil {
			return err
		}
		resourceJsonMaps[instances[i].ResourceName][instances[i].Name] = jsonResult

		resourceHclBlocks = append(resourceHclBlocks, instanceStateToHclBlock(instances[i].ResourceName, instances[i].Name, jsonResult))
	}

	if err := setupOutputFolder(importerMode, outputPath); err != nil {
		return diag.FromErr(err)
	}

	// 1. Save Terraform configuration file
	// 2. Save Terraform state file
	// 3. Run terraform refresh to upgrade terraform state from v3 to v4.

	// Follow https://github.com/hashicorp/terraform/issues/15608
	if err := writeHclConfig(resourceHclBlocks, importerMode, outputPath); err != nil {
		return err
	}

	// new resource.name_hash + resource.state or something
	if err := writeTfState(ctx, instances, outputPath); err != nil {
		return err
	}

	d.SetId(outputPath)

	tflog.Debug(ctx, fmt.Sprintf("Finished creating TF Importer %q: %s", d.Id(), outputPath), map[string]interface{}{tfImporterLoggingKey: d.Id()})

	return nil
}

func setupOutputFolder(mode ImporterMode, outputPath string) error {
	// Create or recreate an output folder
	if err := createOrRecreateDirectory(outputPath); err != nil {
		return err
	}

	if err := ImportVariablesTf(mode, outputPath); err != nil {
		return err
	}
	return nil
}

func loadAllInstances(ctx context.Context, resourcesToImport []string, outputPath string, mode ImporterMode, meta interface{}) ([]instanceData, diag.Diagnostics) {
	importers, err := getResourceImporters(resourcesToImport, mode)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	client := meta.(*Client) // Client from the original provider with all the credentials set

	if err := buildImporterInstanceIdMaps(ctx, importers, client, outputPath); err != nil {
		return nil, err
	}

	// A list of all instances of every resource that should be saved
	// in Terraform state file and Terraform configuration file
	var instances []instanceData

	fakeProvider := New("", "")() // Fake Provider to infer resource schemas

	for resource, importer := range importers {
		resourceInstances, err := loadInstances(ctx, resource, importer, fakeProvider, meta)
		if err != nil {
			return nil, err
		}
		instances = append(instances, resourceInstances...)
	}
	if len(instances) == 0 {
		return nil, diag.Errorf("0 resources were imported. Please verify that the provided API keys have sufficient read access to the target resources.")
	}
	return instances, nil
}

func getResourceImporters(instancesToImport []string, mode ImporterMode) (map[string]*Importer, error) {
	cloudSupportedImporters := map[string]*Importer{
		// TODO: add more resources
		"confluent_service_account": serviceAccountImporter(),
		"confluent_environment":     environmentImporter(),
		"confluent_kafka_cluster":   kafkaClusterImporter(),
		"confluent_connector":       connectorImporter(),
	}

	kafkaSupportedImporters := map[string]*Importer{
		"confluent_kafka_acl":   kafkaAclImporter(),
		"confluent_kafka_topic": kafkaTopicImporter(),
	}

	schemaRegistrySupportedImporters := map[string]*Importer{
		"confluent_schema": schemaImporter(),
	}

	if len(instancesToImport) == 0 {
		if mode == Kafka {
			return kafkaSupportedImporters, nil
		} else if mode == Cloud {
			return cloudSupportedImporters, nil
		} else if mode == SchemaRegistry {
			return schemaRegistrySupportedImporters, nil
		}
	}

	importers := make(map[string]*Importer, len(instancesToImport))

	for _, instanceToImport := range instancesToImport {
		if importer, ok := kafkaSupportedImporters[instanceToImport]; ok {
			importers[instanceToImport] = importer
		} else if importer, ok := cloudSupportedImporters[instanceToImport]; ok {
			importers[instanceToImport] = importer
		} else if importer, ok := schemaRegistrySupportedImporters[instanceToImport]; ok {
			importers[instanceToImport] = importer
		}
	}

	if mapContainsAllKeys(kafkaSupportedImporters, importers) ||
		mapContainsAllKeys(cloudSupportedImporters, importers) ||
		mapContainsAllKeys(schemaRegistrySupportedImporters, importers) {
		return importers, nil
	}
	return nil, fmt.Errorf("cloud, kafka, and schema registry resources can't be imported simultaneously. Remove %q attribute from "+
		"'confluent_tf_importer' resource in your Terraform configuration to use the default set of resources for a given "+
		"importer mode or specify just Cloud or just Kafka, or just Schema Registry resources for %q attribute", paramResources, paramResources)
}

type instanceData struct {
	State                  *terraform.InstanceState
	ComputedOnlyProperties []string
	Name                   string
	ResourceName           string
	CtyType                cty.Type
}

func writeTfState(ctx context.Context, resources []instanceData, outputPath string) diag.Diagnostics {
	stateFilePath, err := getFilePath(defaultTfStateFile, outputPath)
	if err != nil {
		return diag.FromErr(err)
	}

	tfstate := terraform.NewState()
	for _, resource := range resources {
		resourceState := &terraform.ResourceState{
			Type:    resource.ResourceName,
			Primary: resource.State,
		}
		tfstate.RootModule().Resources[resource.ResourceName+"."+resource.Name] = resourceState
	}

	data, err := json.MarshalIndent(tfstate, "", "  ")
	if err != nil {
		return diag.Errorf("Failed to encode state as JSON: %v", err)
	}

	tflog.Info(ctx, fmt.Sprintf("Writing Import state file to %s", stateFilePath))
	if err := writeToFile(data, stateFilePath); err != nil {
		return err
	}

	// Upgrade TF state from v3 to v4:
	// https://discuss.hashicorp.com/t/unqualified-provider-aws/18554/10
	stateReplaceCommandStr := "terraform state replace-provider registry.terraform.io/-/confluent registry.terraform.io/confluentinc/confluent"
	cmd, err := commandWithResolvedSymlink(ctx, "terraform")
	if err != nil {
		return diag.Errorf("The process was successfully completed, but there's one more manual step you'll "+
			"need to take. Please run the following terraform CLI command: %q in %s folder.", stateReplaceCommandStr,
			outputPath)
	}

	cmd.Args = append(cmd.Args, []string{
		"state",
		"replace-provider",
		"-auto-approve",
		"-state=" + stateFilePath,
		"registry.terraform.io/-/confluent",
		"registry.terraform.io/confluentinc/confluent",
	}...)

	tflog.Info(ctx, fmt.Sprintf("Running %q in %s", stateReplaceCommandStr, outputPath))

	if err = cmd.Run(); err != nil {
		return diag.Errorf("The process was successfully completed, but there's one more manual step you'll "+
			"need to take. Please run the following terraform CLI command: %q in %s folder.", stateReplaceCommandStr,
			outputPath)
	}
	return nil
}

func commandWithResolvedSymlink(ctx context.Context, commandName string) (*exec.Cmd, error) {
	path, err := exec.LookPath(commandName)
	if err != nil {
		return nil, fmt.Errorf("failed to find %s path: %w", commandName, err)
	}

	fileInfo, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to Lstat %s path: %w", commandName, err)
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s path symlink: %w", commandName, err)
		}
	}

	return exec.CommandContext(ctx, path), nil
}

func loadInstances(ctx context.Context, resourceName string, importer *Importer, fakeProvider *schema.Provider, meta interface{}) ([]instanceData, diag.Diagnostics) {
	resourceSchema := fakeProvider.ResourcesMap[resourceName]
	if resourceSchema == nil {
		return nil, diag.Errorf("Resource type %s not defined", resourceName)
	}
	// cty.Object(map[string]cty.Type{"api_version":cty.String, "description":cty.String, "display_name":cty.String, "id":cty.String, "kind":cty.String})
	ctyType := resourceSchema.CoreConfigSchema().ImpliedType()

	configSchema := resourceSchema.CoreConfigSchema()
	var computedOnlyAttributes []string
	for attributeName, attributeSchema := range configSchema.Attributes {
		if attributeSchema.Computed && !attributeSchema.Optional && !attributeSchema.Required {
			computedOnlyAttributes = append(computedOnlyAttributes, attributeName)
		}
	}

	var resources []instanceData
	for instanceId, instanceName := range importer.InstanceIdMap {
		if resourceName == "confluent_kafka_topic" || resourceName == "confluent_kafka_acl" {
			// APIF-2043: TEMPORARY HACK
			// Sleep for 0.5s to avoid sending too many requests
			SleepIfNotTestMode(500*time.Millisecond, meta.(*Client).isAcceptanceTestMode)
		}

		instanceState, err := getInstanceState(ctx, resourceSchema, instanceId, meta)
		if err != nil {
			return nil, diag.Errorf("Failed to get state for %s instance %s: %v", resourceName, instanceId, err)
		}

		if instanceState == nil {
			tflog.Info(ctx, fmt.Sprintf("Resource %s was deleted while this script was running. Skipping.", instanceName))
			continue
		}

		resources = append(resources, instanceData{
			State:                  instanceState,
			ComputedOnlyProperties: computedOnlyAttributes,
			// test123
			Name: instanceName,
			// confluent_service_account
			ResourceName: resourceName,
			// cty.Object(map[string]cty.Type{"api_version":cty.String, "description":cty.String, "display_name":cty.String, "id":cty.String, "kind":cty.String})
			CtyType: ctyType,
		})
	}

	return resources, nil
}

func getInstanceState(ctx context.Context, resource *schema.Resource, instanceId string, meta interface{}) (*terraform.InstanceState, diag.Diagnostics) {
	// instanceID is an ID that is suitable for terraform import {instanceID} operation
	instanceState := &terraform.InstanceState{ID: instanceId}
	if resource.Importer != nil && resource.Importer.StateContext != nil {
		// Passing meta is super smart
		resourceData, err := resource.Importer.StateContext(ctx, resource.Data(instanceState), meta)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		if len(resourceData) > 0 {
			instanceState = resourceData[0].State()
		}
	}

	state, err := resource.RefreshWithoutUpgrade(ctx, instanceState, meta)
	if err != nil {
		return nil, err
	}
	if state.ID == "" {
		return nil, nil
	}
	return state, nil
}

func tfImporterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting TF Importer %q", d.Id()), map[string]interface{}{tfImporterLoggingKey: d.Id()})
	outputPath := d.Id()
	if err := os.Remove(fmt.Sprintf("%s/%s", outputPath, tfConfigurationFileName)); err != nil {
		return diag.Errorf("error deleting TF Importer %q's TF configuration file: %s", d.Id(), createDescriptiveError(err))
	}
	if err := os.Remove(fmt.Sprintf("%s/%s", outputPath, tfStateFileName)); err != nil {
		return diag.Errorf("error deleting TF Importer %q's TF state file: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting TF Importer %q", d.Id()), map[string]interface{}{tfImporterLoggingKey: d.Id()})

	return nil
}

func tfImporterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading TF Importer %q", d.Id()), map[string]interface{}{tfImporterLoggingKey: d.Id()})
	outputPath := d.Id()
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		tflog.Warn(ctx, fmt.Sprintf("Removing TF Importer %q in TF state because %q path does not exist", d.Id(), outputPath), map[string]interface{}{tfImporterLoggingKey: d.Id()})
		d.SetId("")
		return nil
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading TF Importer %q", outputPath), map[string]interface{}{tfImporterLoggingKey: d.Id()})

	return nil
}

// Importer is an interface that every Importable resource should implement
type Importer struct {
	LoadInstanceIds LoadInstanceIdsFunc

	// Map of resource id->names. This is set after a call to LoadInstanceIds
	InstanceIdMap InstanceIdsToNameMap
}

// LoadInstanceIdsFunc is a method that
// returns a map of all instances of a specific resource
// where key is an instance ID and value is an instance name.
// For example, this method might return { "sa-abc123": "app-manager", "sa-xyz123: "test" }
// for confluent_service_account resource.
type LoadInstanceIdsFunc func(context.Context, *Client) (InstanceIdsToNameMap, diag.Diagnostics)

type InstanceId string
type InstanceName string

type InstanceIdsToNameMap map[string]string

func (r *Importer) LoadInstances(ctx context.Context, client *Client) diag.Diagnostics {
	result, err := r.LoadInstanceIds(ctx, client)
	if err != nil {
		return err
	}

	r.InstanceIdMap = result
	return nil
}

func buildImporterInstanceIdMaps(ctx context.Context, importers map[string]*Importer, client *Client, outputPath string) diag.Diagnostics {
	for resourceName, importer := range importers {
		if err := importer.LoadInstances(ctx, client); err != nil {
			return nil
		}
		tflog.Info(ctx, fmt.Sprintf("Loaded %d instances of resource %s", len(importer.InstanceIdMap), resourceName), map[string]interface{}{tfImporterLoggingKey: outputPath})
	}
	return nil
}

func ImportVariablesTf(mode ImporterMode, outputPath string) error {
	variablesFilePath, err := getFilePath(defaultVariablesTfFile, outputPath)
	if err != nil {
		return err
	}

	variablesContent := `
variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}
`
	if mode == Kafka {
		variablesContent = `
variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_api_key" {
  description = "Kafka API Key"
  type        = string
  sensitive   = true
}

variable "kafka_api_secret" {
  description = "Kafka API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_rest_endpoint" {
  description = "The REST Endpoint of the Kafka cluster"
  type        = string
}

variable "kafka_id" {
  description = "The ID the the Kafka cluster of the form 'lkc-'"
  type        = string
}
`
	} else if mode == SchemaRegistry {
		variablesContent = `
variable "schema_registry_api_key" {
  description = "Schema Registry API Key"
  type        = string
  sensitive   = true
}

variable "schema_registry_api_secret" {
  description = "Schema Registry API Secret"
  type        = string
  sensitive   = true
}

variable "schema_registry_rest_endpoint" {
  description = "The REST Endpoint of the Schema Registry cluster"
  type        = string
}

variable "schema_registry_id" {
  description = "The ID the the Schema Registry cluster of the form 'lsrc-'"
  type        = string
}
`
	}
	if err := ioutil.WriteFile(variablesFilePath, []byte(variablesContent), os.ModePerm); err != nil {
		return fmt.Errorf("error writing to file %s: %s", variablesFilePath, err)
	}
	return nil
}

func writeHclConfig(resourceNameHclBlocksSlice [][]byte, mode ImporterMode, outputPath string) diag.Diagnostics {
	filePath, err := getFilePath(tfConfigurationFileName, outputPath)
	if err != nil {
		return diag.FromErr(err)
	}

	fileWithHeader := createHclFileWithHeader(mode)
	resourceNameHclBlocksSlice = prependHeaderBytes(fileWithHeader, resourceNameHclBlocksSlice)

	return writeHclToFile(resourceNameHclBlocksSlice, filePath)
}

func createHclFileWithHeader(mode ImporterMode) *hclwrite.File {
	file := hclwrite.NewEmptyFile()
	body := file.Body()

	tfBlock := body.AppendNewBlock("terraform", nil)
	requiredProvidersBlock := tfBlock.Body().AppendNewBlock("required_providers", nil)
	requiredProvidersBlock.Body().SetAttributeValue("confluent", zclCty.ObjectVal(map[string]zclCty.Value{
		"source":  zclCty.StringVal("confluentinc/confluent"),
		"version": zclCty.StringVal("2.33.0"),
	}))

	providerBlock := body.AppendNewBlock("provider", []string{"confluent"})
	providerBody := providerBlock.Body()
	if mode == Cloud {
		providerBody.SetAttributeRaw("cloud_api_key", hclwrite.Tokens{
			{Bytes: []byte(" var.confluent_cloud_api_key")},
		})
		providerBody.SetAttributeRaw("cloud_api_secret", hclwrite.Tokens{
			{Bytes: []byte(" var.confluent_cloud_api_secret")},
		})
	} else if mode == Kafka {
		providerBody.SetAttributeRaw("cloud_api_key", hclwrite.Tokens{
			{Bytes: []byte(" var.confluent_cloud_api_key")},
		})
		providerBody.SetAttributeRaw("cloud_api_secret", hclwrite.Tokens{
			{Bytes: []byte(" var.confluent_cloud_api_secret")},
		})
		providerBody.SetAttributeRaw("kafka_id", hclwrite.Tokens{
			{Bytes: []byte(" var.kafka_id")},
		})
		providerBody.SetAttributeRaw("kafka_rest_endpoint", hclwrite.Tokens{
			{Bytes: []byte(" var.kafka_rest_endpoint")},
		})
		providerBody.SetAttributeRaw("kafka_api_key", hclwrite.Tokens{
			{Bytes: []byte(" var.kafka_api_key")},
		})
		providerBody.SetAttributeRaw("kafka_api_secret", hclwrite.Tokens{
			{Bytes: []byte(" var.kafka_api_secret")},
		})
	} else if mode == SchemaRegistry {
		providerBody.SetAttributeRaw("schema_registry_id", hclwrite.Tokens{
			{Bytes: []byte(" var.schema_registry_id")},
		})
		providerBody.SetAttributeRaw("schema_registry_rest_endpoint", hclwrite.Tokens{
			{Bytes: []byte(" var.schema_registry_rest_endpoint")},
		})
		providerBody.SetAttributeRaw("schema_registry_api_key", hclwrite.Tokens{
			{Bytes: []byte(" var.schema_registry_api_key")},
		})
		providerBody.SetAttributeRaw("schema_registry_api_secret", hclwrite.Tokens{
			{Bytes: []byte(" var.schema_registry_api_secret")},
		})
	}
	return file
}

func prependHeaderBytes(file *hclwrite.File, resourceNameHclBlocksSlice [][]byte) [][]byte {
	headerBytes := file.Bytes()

	if len(resourceNameHclBlocksSlice) == 0 {
		return [][]byte{headerBytes}
	}

	return append([][]byte{headerBytes}, resourceNameHclBlocksSlice...)
}

func writeToFile(bytes []byte, path string) diag.Diagnostics {
	err := ioutil.WriteFile(path, bytes, os.ModePerm)
	if err != nil {
		return diag.Errorf("Error writing file %s: %v", path, err)
	}
	return nil
}

func writeHclToFile(bytes [][]byte, path string) diag.Diagnostics {
	file, err := os.Create(path)
	if err != nil {
		return diag.Errorf("error creating file %s: %s", path, err)
	}
	defer file.Close()

	for _, v := range bytes {
		if _, err := file.Write(append(v, '\n')); err != nil {
			return diag.Errorf("error writing to file %s: %s", path, err)
		}
	}
	return nil
}

func instanceStateToJson(state *terraform.InstanceState, computedOnlyProperties []string, ctyType cty.Type) (map[string]interface{}, diag.Diagnostics) {
	stateVal, err := schema.StateValueFromInstanceState(state, ctyType)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	jsonMap, err := schema.StateValueToJSONMap(stateVal, ctyType)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	for _, attributeName := range computedOnlyProperties {
		delete(jsonMap, attributeName)
	}

	// Remove empty blocks like network { id = "" } from Terraform configuration file
	filteredMap := make(map[string]interface{})
	for k, v := range jsonMap {
		remove := false
		if slice, ok := v.([]interface{}); ok {
			if len(slice) == 1 {
				if nestedMap, ok := slice[0].(map[string]interface{}); ok && len(nestedMap) == 1 {
					if idValue, exists := nestedMap["id"]; exists && idValue == "" {
						remove = true
					}
				}
			}
		}

		if !remove {
			filteredMap[k] = v
		}
	}

	delete(filteredMap, "id")

	return filteredMap, nil
}

func instanceStateToHclBlock(resourceName, instanceName string, json map[string]interface{}) []byte {
	f := hclwrite.NewEmptyFile()
	block := f.Body().AppendNewBlock("resource", []string{resourceName, instanceName})
	body := block.Body()

	addBody(body, json)

	lifecycleBlock := body.AppendNewBlock("lifecycle", nil)
	lifecycleBody := lifecycleBlock.Body()
	lifecycleBody.SetAttributeValue("prevent_destroy", zclCty.BoolVal(true))

	return bytes.ReplaceAll(f.Bytes(), []byte("$$"), []byte("$"))
}

func setInterfaceArray(body *hclwrite.Body, k string, v []interface{}) {
	var listItems []zclCty.Value
	for _, val := range v {
		if valMap, ok := val.(map[string]interface{}); ok {
			addBlock(body, k, valMap)
		} else {
			listItems = append(listItems, getCtyValue(val))
		}
	}

	if len(listItems) > 0 {
		body.SetAttributeValue(k, zclCty.ListVal(listItems))
	}
}

func addBody(body *hclwrite.Body, json map[string]interface{}) {
	for k, v := range json {
		addValue(body, k, v)
	}
}

func addBlock(body *hclwrite.Body, k string, valMap map[string]interface{}) {
	block := body.AppendNewBlock(k, nil)
	for key, value := range valMap {
		addValue(block.Body(), key, value)
	}
}

func addValue(body *hclwrite.Body, k string, v interface{}) {
	switch vTyped := v.(type) {
	case []interface{}:
		setInterfaceArray(body, k, vTyped)
	default:
		if ctyVal := getCtyValue(v); !ctyVal.IsNull() {
			body.SetAttributeValue(k, ctyVal)
		}
	}
}

func getCtyValue(v interface{}) zclCty.Value {
	switch vTyped := v.(type) {
	case string:
		return zclCty.StringVal(vTyped)
	case bool:
		return zclCty.BoolVal(vTyped)
	case int:
		return zclCty.NumberIntVal(int64(vTyped))
	case int32:
		return zclCty.NumberIntVal(int64(vTyped))
	case int64:
		return zclCty.NumberIntVal(vTyped)
	case float32:
		return zclCty.NumberFloatVal(float64(vTyped))
	case float64:
		return zclCty.NumberFloatVal(vTyped)
	case map[string]interface{}:
		return createHclObject(vTyped)
	default:
		return zclCty.NilVal
	}
}

func createHclObject(v map[string]interface{}) zclCty.Value {
	obj := make(map[string]zclCty.Value)
	for key, val := range v {
		if ctyVal := getCtyValue(val); !ctyVal.IsNull() {
			obj[key] = ctyVal
		}
	}
	if len(obj) == 0 {
		return zclCty.NilVal
	}
	return zclCty.ObjectVal(obj)
}

func createOrRecreateDirectory(outputPath string) error {
	path := outputPath
	if _, err := os.Stat(path); err == nil {
		if outputPath == defaultOutputPath {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to remove existing directory %s: %s", path, err)
			}
		} else {
			if err := os.RemoveAll(filepath.Join(path, ".terraform")); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove subfolder '.terraform' in existing directory %s: %s", path, err)
			}

			if err := os.Remove(filepath.Join(path, tfConfigurationFileName)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %q file in existing directory %s: %s", tfConfigurationFileName, path, err)
			}

			if err := os.Remove(filepath.Join(path, tfStateFileName)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %q file in existing directory %s: %s", tfStateFileName, path, err)
			}

			if err := os.Remove(filepath.Join(path, defaultTfStateFile)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %q file in existing directory %s: %s", defaultTfStateFile, path, err)
			}

			if err := os.Remove(filepath.Join(path, defaultVariablesTfFile)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %q file in existing directory %s: %s", defaultVariablesTfFile, path, err)
			}

			if err := os.Remove(filepath.Join(path, tfLockFileName)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %q file in existing directory %s: %s", tfLockFileName, path, err)
			}
		}
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create a path with directory %s: %s", path, err)
		}
	}
	return nil
}

func getFilePath(filename, outputPath string) (string, error) {
	path := filepath.Join(outputPath, filename)
	// Ensure that the resulting path is different from the input directory.
	if path == outputPath {
		return "", fmt.Errorf("failed to create file path with directory %s", outputPath)
	}
	return path, nil
}

func toValidTerraformResourceName(input string) string {
	// Replace invalid characters with underscores
	var sb strings.Builder
	for i, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			sb.WriteRune(unicode.ToLower(r))
		} else if i > 0 {
			sb.WriteRune('_')
		}
	}

	// Ensure the first character is a letter
	output := sb.String()

	// Resource names must start with a letter or underscore, and may contain only letters, digits, underscores, and dashes.
	// https://developer.hashicorp.com/terraform/language/resources/syntax
	if len(output) == 0 || !unicode.IsLetter(rune(output[0])) {
		output = "importer_" + output
	}

	// https://cloud.google.com/docs/terraform/best-practices-for-terraform#naming-convention
	output = strings.ReplaceAll(output, "-", "_")

	return output
}

func mapContainsAllKeys(map1, map2 map[string]*Importer) bool {
	for key := range map2 {
		if _, ok := map1[key]; !ok {
			return false
		}
	}
	return true
}

func overrideUserAgent(client *Client) {
	const importer = "TFImporter"

	// Add "importer" suffix to the default user agent
	client.apiKeysClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.apiKeysClient.GetConfig().UserAgent, importer)
	client.byokClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.byokClient.GetConfig().UserAgent, importer)
	client.cmkClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.cmkClient.GetConfig().UserAgent, importer)
	client.connectClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.connectClient.GetConfig().UserAgent, importer)
	client.iamClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.iamClient.GetConfig().UserAgent, importer)
	client.iamV1Client.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.iamV1Client.GetConfig().UserAgent, importer)
	client.mdsClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.mdsClient.GetConfig().UserAgent, importer)
	client.netClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.netClient.GetConfig().UserAgent, importer)
	client.oidcClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.oidcClient.GetConfig().UserAgent, importer)
	client.orgClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.orgClient.GetConfig().UserAgent, importer)
	client.srcmClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.srcmClient.GetConfig().UserAgent, importer)
	client.ksqlClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.ksqlClient.GetConfig().UserAgent, importer)
	client.quotasClient.GetConfig().UserAgent = fmt.Sprintf("%s %s", client.quotasClient.GetConfig().UserAgent, importer)

	client.userAgent = fmt.Sprintf("%s %s", client.userAgent, importer)
	client.kafkaRestClientFactory.userAgent = fmt.Sprintf("%s %s", client.kafkaRestClientFactory.userAgent, importer)
}

// TODO: add docs
