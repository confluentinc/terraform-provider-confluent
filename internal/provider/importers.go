// Copyright 2021 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func connectorImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllConnectors,
	}
}

func loadAllConnectors(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	for _, environment := range environments {
		kafkaClusters, err := loadKafkaClusters(ctx, client, environment.GetId(), nil)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Clusters in Environment %q: %s", environment.GetId(), createDescriptiveError(err)))
			return instances, diag.FromErr(createDescriptiveError(err))
		}
		for _, kafkaCluster := range kafkaClusters {
			connectorNames, err := loadConnectorsByEnvironmentIdAndKafkaClusterId(ctx, client, environment.GetId(), kafkaCluster.GetId())
			if err != nil {
				tflog.Warn(ctx, fmt.Sprintf("Error reading Connectors in Environment %q and Kafka Cluster %q: %s", environment.GetId(), kafkaCluster.GetId(), createDescriptiveError(err)))
				return instances, diag.FromErr(createDescriptiveError(err))
			}
			connectorNamesJson, err := json.Marshal(connectorNames)
			if err != nil {
				return instances, diag.Errorf("error reading Connectors in Environment %q and Kafka Cluster %q: error marshaling %#v to json: %s", environment.GetId(), kafkaCluster.GetId(), connectorNames, createDescriptiveError(err))
			}
			tflog.Debug(ctx, fmt.Sprintf("Fetched Connectors in Environment %q and Kafka Cluster %q: %s", environment.GetId(), kafkaCluster.GetId(), connectorNamesJson))

			for _, connectorName := range connectorNames {
				instanceId := fmt.Sprintf("%s/%s/%s", environment.GetId(), kafkaCluster.GetId(), connectorName)
				instances[instanceId] = toValidTerraformResourceName(connectorName)
			}
		}
	}
	return instances, nil
}

func environmentImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllEnvironments,
	}
}

func loadAllEnvironments(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Environments: %s", createDescriptiveError(err)))
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	environmentsJson, err := json.Marshal(environments)
	if err != nil {
		return instances, diag.Errorf("error reading Environments: error marshaling %#v to json: %s", environments, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Environments: %s", environmentsJson))

	for _, environment := range environments {
		instanceId := environment.GetId()
		instances[instanceId] = toValidTerraformResourceName(environment.GetDisplayName())
	}

	return instances, nil
}

func serviceAccountImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllServiceAccounts,
	}
}

func loadAllServiceAccounts(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	serviceAccounts, err := loadServiceAccounts(ctx, client)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Service Accounts: %s", createDescriptiveError(err)))
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	serviceAccountsJson, err := json.Marshal(serviceAccounts)
	if err != nil {
		return instances, diag.Errorf("error reading Service Accounts: error marshaling %#v to json: %s", serviceAccounts, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Service Accounts: %s", serviceAccountsJson))

	for _, serviceAccount := range serviceAccounts {
		instanceId := serviceAccount.GetId()
		instances[instanceId] = toValidTerraformResourceName(serviceAccount.GetDisplayName())
	}

	return instances, nil
}

func kafkaAclImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaAcls,
	}
}

func loadAllKafkaAcls(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	kafkaRestClient := client.kafkaRestClientFactory.CreateKafkaRestClient(client.kafkaRestEndpoint, client.kafkaClusterId, client.kafkaApiKey, client.kafkaApiSecret, true, true, client.oauthToken)

	acls, resp, err := kafkaRestClient.apiClient.ACLV3Api.GetKafkaAcls(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()

	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka ACLs for Kafka Cluster %q: %s", kafkaRestClient.clusterId, createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err, resp))
	}
	kafkaAclsJson, err := json.Marshal(acls)
	if err != nil {
		return nil, diag.Errorf("error reading Kafka ACLs for Kafka Cluster %q: error marshaling %#v to json: %s", kafkaRestClient.clusterId, acls, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka ACLs for Kafka Cluster %q: %s", kafkaRestClient.clusterId, kafkaAclsJson))

	// APIF-2038: Kafka REST API only accepts integer ID at the moment
	serviceAccounts, resp, err := client.iamV1Client.ServiceAccountsV1Api.ListV1ServiceAccounts(client.iamV1ApiContext(ctx)).Execute()
	users, resp, err := client.iamV1Client.UsersV1Api.ListV1Users(client.iamV1ApiContext(ctx)).Execute()

	principalIdMap := make(map[int32]string)

	for _, principal := range serviceAccounts.GetUsers() {
		principalIdMap[principal.GetId()] = principal.GetResourceId()
	}
	for _, principal := range users.GetUsers() {
		principalIdMap[principal.GetId()] = principal.GetResourceId()
	}

	for _, aclData := range acls.GetData() {
		principalWithResourceId, err := principalWithIntegerIdToPrincipalWithResourceId(principalIdMap, aclData.GetPrincipal())
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("%s", createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
			continue
		}
		acl := Acl{
			ResourceType: aclData.GetResourceType(),
			ResourceName: aclData.GetResourceName(),
			PatternType:  aclData.GetPatternType(),
			Principal:    principalWithResourceId,
			Host:         aclData.GetHost(),
			Operation:    aclData.GetOperation(),
			Permission:   aclData.GetPermission(),
		}
		instanceId := createKafkaAclId(client.kafkaClusterId, acl)
		instances[instanceId] = toValidTerraformResourceName(createAclInstanceName(acl))
	}

	return instances, nil
}

func kafkaClusterImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaClusters,
	}
}

func loadAllKafkaClusters(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	environments, err := loadEnvironments(ctx, client)
	if err != nil {
		return instances, diag.FromErr(createDescriptiveError(err))
	}
	for _, environment := range environments {
		kafkaClusters, err := loadKafkaClusters(ctx, client, environment.GetId(), nil)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Clusters in Environment %q: %s", environment.GetId(), createDescriptiveError(err)))
			return instances, diag.FromErr(createDescriptiveError(err))
		}
		kafkaClustersJson, err := json.Marshal(kafkaClusters)
		if err != nil {
			return instances, diag.Errorf("error reading Kafka Clusters in Environment %q: error marshaling %#v to json: %s", environment.GetId(), kafkaClusters, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Clusters in Environment %q: %s", environment.GetId(), kafkaClustersJson))

		for _, kafkaCluster := range kafkaClusters {
			instanceId := fmt.Sprintf("%s/%s", environment.GetId(), kafkaCluster.GetId())
			instances[instanceId] = toValidTerraformResourceName(kafkaCluster.Spec.GetDisplayName())
		}
	}
	return instances, nil
}

// TODO: we might want to load all the resources instead
func kafkaTopicImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllKafkaTopics,
	}
}

func loadAllKafkaTopics(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	kafkaRestClient := client.kafkaRestClientFactory.CreateKafkaRestClient(client.kafkaRestEndpoint, client.kafkaClusterId, client.kafkaApiKey, client.kafkaApiSecret, true, true, client.oauthToken)

	topics, resp, err := kafkaRestClient.apiClient.TopicV3Api.ListKafkaTopics(kafkaRestClient.apiContext(ctx), kafkaRestClient.clusterId).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Topics for Kafka Cluster %q: %s", kafkaRestClient.clusterId, createDescriptiveError(err, resp)), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err, resp))
	}
	topicsJson, err := json.Marshal(topics)
	if err != nil {
		return nil, diag.Errorf("error reading Kafka Topics for Kafka Cluster %q: error marshaling %#v to json: %s", kafkaRestClient.clusterId, topics, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Kafka Topics for Kafka Cluster %q: %s", kafkaRestClient.clusterId, topicsJson), map[string]interface{}{kafkaClusterLoggingKey: kafkaRestClient.clusterId})

	for _, topic := range topics.GetData() {
		if shouldFilterOutTopic(topic.GetTopicName()) {
			continue
		}
		instanceId := createKafkaTopicId(kafkaRestClient.clusterId, topic.GetTopicName())
		instances[instanceId] = toValidTerraformResourceName(topic.GetTopicName())
	}

	return instances, nil
}

func schemaImporter() *Importer {
	return &Importer{
		LoadInstanceIds: loadAllSchemas,
	}
}

func loadAllSchemas(ctx context.Context, client *Client) (InstanceIdsToNameMap, diag.Diagnostics) {
	instances := make(InstanceIdsToNameMap)

	schemaRegistryRestClient := client.schemaRegistryRestClientFactory.CreateSchemaRegistryRestClient(client.schemaRegistryRestEndpoint, client.schemaRegistryClusterId, client.schemaRegistryApiKey, client.schemaRegistryApiSecret, true, client.oauthToken)

	subjects, resp, err := schemaRegistryRestClient.apiClient.SubjectsV1Api.List(schemaRegistryRestClient.apiContext(ctx)).Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Subjects for Schema Registry Cluster %q: %s", schemaRegistryRestClient.clusterId, createDescriptiveError(err, resp)), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})
		return nil, diag.FromErr(createDescriptiveError(err, resp))
	}
	subjectsJson, err := json.Marshal(subjects)
	if err != nil {
		return nil, diag.Errorf("error reading Subjects for Schema Registry Cluster %q: error marshaling %#v to json: %s", schemaRegistryRestClient.clusterId, subjects, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Subjects for Schema Registry Cluster %q: %s", schemaRegistryRestClient.clusterId, subjectsJson), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})

	for _, subjectName := range subjects {
		// using schemaSr as schema collides with the package name
		schemaSr, _, err := loadSchema(schemaRegistryRestClient.apiContext(ctx), &schema.ResourceData{}, schemaRegistryRestClient, subjectName, latestSchemaVersionAndPlaceholderForSchemaIdentifier)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading the latest Schema for Subject %q: %s", schemaSr.GetSubject(), createDescriptiveError(err, resp)), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})
			return nil, diag.FromErr(createDescriptiveError(err, resp))
		}
		schemaJson, err := json.Marshal(schemaSr)
		if err != nil {
			return nil, diag.Errorf("error reading the latest Schema for Subject %q: error marshaling %#v to json: %s", schemaSr.GetSubject(), schemaSr, createDescriptiveError(err))
		}
		tflog.Debug(ctx, fmt.Sprintf("Fetched the latest Schema for Subject %q: %s", schemaSr.GetSubject(), schemaJson), map[string]interface{}{schemaRegistryClusterLoggingKey: schemaRegistryRestClient.clusterId})

		instanceId := createSchemaId(schemaRegistryRestClient.clusterId, schemaSr.GetSubject(), schemaSr.GetId(), false)
		instances[instanceId] = toValidTerraformResourceName(schemaSr.GetSubject())
	}

	return instances, nil
}
