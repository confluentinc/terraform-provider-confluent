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
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"net/http"
	"time"
)

const (
	stateUp = "UP"
)

func waitForCreatedKafkaApiKeyToSync(ctx context.Context, c *KafkaRestClient) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateInProgress},
		Target:  []string{stateDone},
		Refresh: kafkaApiKeySyncStatus(ctx, c),
		// Default timeout for a resource
		// https://www.terraform.io/plugin/sdkv2/resources/retries-and-customizable-timeouts
		// Based on the tests, Kafka API Key takes about 2 minutes to sync
		Timeout:      20 * time.Minute,
		Delay:        1 * time.Minute,
		PollInterval: 1 * time.Minute,
		// Expects 2x http.StatusOK before exiting which adds PollInterval to the total time it takes to sync an API Key
		// but helps to ensure the API Key is synced across all brokers.
		ContinuousTargetOccurence: 2,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka API Key %q to sync", c.clusterApiKey), map[string]interface{}{apiKeyLoggingKey: c.clusterApiKey})
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return err
	}
	return nil
}

func waitForCreatedCloudApiKeyToSync(ctx context.Context, c *Client, cloudApiKey, cloudApiSecret string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateInProgress},
		Target:  []string{stateDone},
		Refresh: cloudApiKeySyncStatus(ctx, c, cloudApiKey, cloudApiSecret),
		// Default timeout for a resource
		// https://www.terraform.io/plugin/sdkv2/resources/retries-and-customizable-timeouts
		// Based on the tests, Cloud API Key takes about 10 seconds to sync (or even faster)
		Timeout:      20 * time.Minute,
		Delay:        15 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Cloud API Key %q to sync", cloudApiKey), map[string]interface{}{apiKeyLoggingKey: cloudApiKey})
	if _, err := stateConf.WaitForStateContext(c.orgApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKafkaClusterToProvision(ctx context.Context, c *Client, environmentId, clusterId, clusterType string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateProvisioning},
		Target:  []string{stateProvisioned},
		Refresh: kafkaClusterProvisionStatus(c.cmkApiContext(ctx), c, environmentId, clusterId),
		// https://docs.confluent.io/cloud/current/clusters/cluster-types.html#provisioning-time
		Timeout:      getTimeoutFor(clusterType),
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Cluster %q provisioning status to become %q", clusterId, stateProvisioned), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
	if _, err := stateConf.WaitForStateContext(c.cmkApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKsqlClusterToProvision(ctx context.Context, c *Client, environmentId, clusterId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateProvisioning},
		Target:       []string{stateProvisioned},
		Refresh:      ksqlClusterProvisionStatus(c.ksqlApiContext(ctx), c, environmentId, clusterId),
		Timeout:      ksqlCreateTimeout,
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for ksqlDB Cluster %q provisioning status to become %v", clusterId, []string{stateUp, stateProvisioned}), map[string]interface{}{ksqlClusterLoggingKey: clusterId})
	if _, err := stateConf.WaitForStateContext(c.ksqlApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForPrivateLinkAccessToProvision(ctx context.Context, c *Client, environmentId, privateLinkAccessId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateProvisioning},
		Target:       []string{stateReady},
		Refresh:      privateLinkAccessProvisionStatus(c.netApiContext(ctx), c, environmentId, privateLinkAccessId),
		Timeout:      networkingAPICreateTimeout,
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Private Link Access %q provisioning status to become %q", privateLinkAccessId, stateReady), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForNetworkToProvision(ctx context.Context, c *Client, environmentId, networkId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateProvisioning},
		Target:  []string{stateReady},
		Refresh: networkProvisionStatus(c.netApiContext(ctx), c, environmentId, networkId),
		Timeout: networkingAPICreateTimeout,
		// TODO: increase delay
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Network %q provisioning status to become %q", networkId, stateReady), map[string]interface{}{networkLoggingKey: networkId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForSchemaRegistryClusterToProvision(ctx context.Context, c *Client, environmentId, clusterId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateProvisioning},
		Target:  []string{stateProvisioned},
		Refresh: schemaRegistryClusterProvisionStatus(c.srcmApiContext(ctx), c, environmentId, clusterId),
		// https://docs.confluent.io/cloud/current/clusters/cluster-types.html#provisioning-time
		Timeout:      1 * time.Hour,
		Delay:        5 * time.Second,
		PollInterval: 30 * time.Second,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Schema Registry Cluster %q provisioning status to become %q", clusterId, stateProvisioned), map[string]interface{}{schemaRegistryClusterLoggingKey: clusterId})
	if _, err := stateConf.WaitForStateContext(c.srcmApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForConnectorToProvision(ctx context.Context, c *Client, displayName, environmentId, clusterId string) error {
	stateConf := &resource.StateChangeConf{
		// Allow PROVISIONING -> DEGRADED -> RUNNING transition
		Pending:      []string{stateProvisioning, stateDegraded},
		Target:       []string{stateRunning},
		Refresh:      connectorProvisionStatus(c.connectApiContext(ctx), c, displayName, environmentId, clusterId),
		Timeout:      connectAPICreateTimeout,
		Delay:        6 * time.Minute,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Connector %q=%q provisioning status to become %q", paramDisplayName, displayName, stateRunning))
	if _, err := stateConf.WaitForStateContext(c.connectApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForConnectorToChangeStatus(ctx context.Context, c *Client, displayName, environmentId, clusterId, currentStatus, targetStatus string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{currentStatus},
		Target:       []string{targetStatus},
		Refresh:      connectorUpdateStatus(c.connectApiContext(ctx), c, displayName, environmentId, clusterId),
		Timeout:      1 * time.Hour,
		Delay:        30 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Connector %q=%q status to become %q", paramDisplayName, displayName, targetStatus))
	if _, err := stateConf.WaitForStateContext(c.connectApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKafkaMirrorTopicToChangeStatus(ctx context.Context, c *KafkaRestClient, clusterId, linkName, mirrorTopicName, currentStatus, targetStatus string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{currentStatus},
		Target:       []string{targetStatus},
		Refresh:      kafkaMirrorTopicUpdateStatus(c.apiContext(ctx), c, clusterId, linkName, mirrorTopicName),
		Timeout:      5 * time.Minute,
		Delay:        2 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	kafkaMirrorTopicId := createKafkaMirrorTopicId(clusterId, linkName, mirrorTopicName)
	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Mirror Topic %q to be deleted", kafkaMirrorTopicId), map[string]interface{}{kafkaMirrorTopicLoggingKey: kafkaMirrorTopicId})
	if _, err := stateConf.WaitForStateContext(c.apiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForPeeringToProvision(ctx context.Context, c *Client, environmentId, peeringId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateProvisioning},
		Target:  []string{stateReady, statePendingAccept},
		Refresh: peeringProvisionStatus(c.netApiContext(ctx), c, environmentId, peeringId),
		Timeout: networkingAPICreateTimeout,
		// TODO: increase delay
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Peering %q provisioning status to become %q", peeringId, statePendingAccept), map[string]interface{}{networkLoggingKey: peeringId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForTransitGatewayAttachmentToProvision(ctx context.Context, c *Client, environmentId, transitGatewayAttachmentId string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateProvisioning},
		Target:  []string{stateReady, statePendingAccept},
		Refresh: transitGatewayAttachmentProvisionStatus(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId),
		Timeout: networkingAPICreateTimeout,
		// TODO: increase delay
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Transit Gateway Attachment %q provisioning status to become %q", transitGatewayAttachmentId, statePendingAccept), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKafkaClusterCkuUpdateToComplete(ctx context.Context, c *Client, environmentId, clusterId string, cku int32) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateInProgress},
		Target:  []string{stateDone},
		Refresh: kafkaClusterCkuUpdateStatus(c.cmkApiContext(ctx), c, environmentId, clusterId, cku),
		// https://docs.confluent.io/cloud/current/clusters/cluster-types.html#resizing-time
		Timeout:      getTimeoutFor(kafkaClusterTypeDedicated),
		Delay:        5 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Cluster %q CKU update", clusterId), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
	if _, err := stateConf.WaitForStateContext(c.cmkApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForPrivateLinkAccessToBeDeleted(ctx context.Context, c *Client, environmentId, privateLinkAccessId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      privateLinkAccessDeleteStatus(c.netApiContext(ctx), c, environmentId, privateLinkAccessId),
		Timeout:      networkingAPIDeleteTimeout,
		Delay:        1 * time.Minute,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Private Link Access %q to be deleted", privateLinkAccessId), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForPeeringToBeDeleted(ctx context.Context, c *Client, environmentId, peeringId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      peeringDeleteStatus(c.netApiContext(ctx), c, environmentId, peeringId),
		Timeout:      networkingAPIDeleteTimeout,
		Delay:        1 * time.Minute,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Peering %q to be deleted", peeringId), map[string]interface{}{peeringLoggingKey: peeringId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForTransitGatewayAttachmentToBeDeleted(ctx context.Context, c *Client, environmentId, transitGatewayAttachmentId string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      transitGatewayAttachmentDeleteStatus(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId),
		Timeout:      networkingAPIDeleteTimeout,
		Delay:        1 * time.Minute,
		PollInterval: 1 * time.Minute,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for Transit Gateway Attachment %q to be deleted", transitGatewayAttachmentId), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
	if _, err := stateConf.WaitForStateContext(c.netApiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKafkaTopicToBeDeleted(ctx context.Context, c *KafkaRestClient, topicName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      kafkaTopicDeleteStatus(c.apiContext(ctx), c, topicName),
		Timeout:      1 * time.Hour,
		Delay:        10 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	topicId := createKafkaTopicId(c.clusterId, topicName)
	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Topic %q to be deleted", topicId), map[string]interface{}{kafkaTopicLoggingKey: topicId})
	if _, err := stateConf.WaitForStateContext(c.apiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func waitForKafkaMirrorTopicToBeDeleted(ctx context.Context, c *KafkaRestClient, linkName, mirrorTopicName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      kafkaMirrorTopicDeleteStatus(c.apiContext(ctx), c, linkName, mirrorTopicName),
		Timeout:      1 * time.Hour,
		Delay:        10 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	kafkaMirrorTopicId := createKafkaMirrorTopicId(c.clusterId, linkName, mirrorTopicName)
	tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Topic %q to be deleted", kafkaMirrorTopicId), map[string]interface{}{kafkaMirrorTopicLoggingKey: kafkaMirrorTopicId})
	if _, err := stateConf.WaitForStateContext(c.apiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func kafkaTopicDeleteStatus(ctx context.Context, c *KafkaRestClient, topicName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		kafkaTopic, resp, err := c.apiClient.TopicV3Api.GetKafkaTopic(c.apiContext(ctx), c.clusterId, topicName).Execute()
		topicId := createKafkaTopicId(c.clusterId, topicName)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Topic %q: %s", topicId, createDescriptiveError(err)), map[string]interface{}{kafkaTopicLoggingKey: topicId})

			// 404 means that the topic has been deleted
			isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
			if isResourceNotFound {
				// Result (the 1st argument) can't be nil
				return 0, stateDone, nil
			} else {
				return nil, stateFailed, err
			}
		}
		return kafkaTopic, stateInProgress, nil
	}
}

func kafkaMirrorTopicDeleteStatus(ctx context.Context, c *KafkaRestClient, linkName, mirrorTopicName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		kafkaTopic, resp, err := c.apiClient.ClusterLinkingV3Api.ReadKafkaMirrorTopic(c.apiContext(ctx), c.clusterId, linkName, mirrorTopicName).Execute()
		kafkaMirrorTopicId := createKafkaMirrorTopicId(c.clusterId, linkName, mirrorTopicName)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Mirror Topic %q: %s", kafkaMirrorTopicId, createDescriptiveError(err)), map[string]interface{}{kafkaMirrorTopicLoggingKey: kafkaMirrorTopicId})

			// 404 means that the topic has been deleted
			isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
			if isResourceNotFound {
				// Result (the 1st argument) can't be nil
				return 0, stateDone, nil
			} else {
				return nil, stateFailed, err
			}
		}
		return kafkaTopic, stateInProgress, nil
	}
}

func waitForClusterLinkToBeDeleted(ctx context.Context, c *KafkaRestClient, linkName string) error {
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateInProgress},
		Target:       []string{stateDone},
		Refresh:      clusterLinkDeleteStatus(c.apiContext(ctx), c, linkName),
		Timeout:      1 * time.Hour,
		Delay:        10 * time.Second,
		PollInterval: 1 * time.Minute,
	}

	topicId := createClusterLinkId(c.clusterId, linkName)
	tflog.Debug(ctx, fmt.Sprintf("Waiting for Cluster Link %q to be deleted", topicId), map[string]interface{}{clusterLinkLoggingKey: topicId})
	if _, err := stateConf.WaitForStateContext(c.apiContext(ctx)); err != nil {
		return err
	}
	return nil
}

func clusterLinkDeleteStatus(ctx context.Context, c *KafkaRestClient, linkName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		clusterLink, resp, err := c.apiClient.ClusterLinkingV3Api.GetKafkaLink(c.apiContext(ctx), c.clusterId, linkName).Execute()
		clusterLinkId := createClusterLinkId(c.clusterId, linkName)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Cluster Link %q: %s", clusterLinkId, createDescriptiveError(err)), map[string]interface{}{clusterLinkLoggingKey: clusterLinkId})

			// 404 means that the cluster link has been deleted
			isResourceNotFound := ResponseHasExpectedStatusCode(resp, http.StatusNotFound)
			if isResourceNotFound {
				// Result (the 1st argument) can't be nil
				return 0, stateDone, nil
			} else {
				return nil, stateFailed, err
			}
		}
		return clusterLink, stateInProgress, nil
	}
}

func kafkaClusterCkuUpdateStatus(ctx context.Context, c *Client, environmentId string, clusterId string, desiredCku int32) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		cluster, _, err := executeKafkaRead(c.cmkApiContext(ctx), c, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Cluster %q: %s", clusterId, createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Cluster %q CKU update", clusterId), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
		// Wail until actual # of CKUs is the same as desired one
		// spec.cku is the user’s desired # of CKUs, and status.cku is the current # of CKUs in effect
		// because the change is still pending, for example
		// Use desiredCku on the off chance that API will not work as expected (i.e., spec.cku = status.cku during expansion).
		// CAPAC-293
		if cluster.Status.GetCku() == cluster.Spec.Config.CmkV2Dedicated.Cku && cluster.Status.GetCku() == desiredCku {
			return cluster, stateDone, nil
		}
		return cluster, stateInProgress, nil
	}
}

func kafkaClusterProvisionStatus(ctx context.Context, c *Client, environmentId string, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		cluster, _, err := executeKafkaRead(c.cmkApiContext(ctx), c, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Cluster %q: %s", clusterId, createDescriptiveError(err)), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Kafka Cluster %q provisioning status to become %q: current status is %q", clusterId, stateProvisioned, cluster.Status.GetPhase()), map[string]interface{}{kafkaClusterLoggingKey: clusterId})
		if cluster.Status.GetPhase() == stateProvisioning || cluster.Status.GetPhase() == stateProvisioned {
			return cluster, cluster.Status.GetPhase(), nil
		} else if cluster.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("kafka Cluster %q provisioning status is %q", clusterId, stateFailed)
		}
		// Kafka Cluster is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("kafka Cluster %q is an unexpected state %q", clusterId, cluster.Status.GetPhase())
	}
}

func ksqlClusterProvisionStatus(ctx context.Context, c *Client, environmentId, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		cluster, _, err := executeKsqlRead(c.ksqlApiContext(ctx), c, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading ksqlDB Cluster %q: %s", clusterId, createDescriptiveError(err)), map[string]interface{}{ksqlClusterLoggingKey: clusterId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for ksqlDB Cluster %q provisioning status to become %q: current status is %q", clusterId, stateProvisioned, cluster.Status.GetPhase()), map[string]interface{}{ksqlClusterLoggingKey: clusterId})
		if cluster.Status.GetPhase() == stateProvisioning || cluster.Status.GetPhase() == stateProvisioned {
			return cluster, cluster.Status.GetPhase(), nil
		} else if cluster.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("ksqlDB Cluster %q provisioning status is %q", clusterId, stateFailed)
		}
		// ksqlDB Cluster is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("ksqlDB Cluster %q is an unexpected state %q", clusterId, cluster.Status.GetPhase())
	}
}

func schemaRegistryClusterProvisionStatus(ctx context.Context, c *Client, environmentId string, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		cluster, _, err := executeSchemaRegistryClusterRead(c.srcmApiContext(ctx), c, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Schema Registry Cluster %q: %s", clusterId, createDescriptiveError(err)), map[string]interface{}{schemaRegistryClusterLoggingKey: clusterId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Schema Registry Cluster %q provisioning status to become %q: current status is %q", clusterId, stateProvisioned, cluster.Status.GetPhase()), map[string]interface{}{schemaRegistryClusterLoggingKey: clusterId})
		if cluster.Status.GetPhase() == stateProvisioning || cluster.Status.GetPhase() == stateProvisioned {
			return cluster, cluster.Status.GetPhase(), nil
		} else if cluster.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("schema Registry Cluster %q provisioning status is %q", clusterId, stateFailed)
		}
		// SR Cluster is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("schema Registry Cluster %q is an unexpected state %q", clusterId, cluster.Status.GetPhase())
	}
}

func privateLinkAccessProvisionStatus(ctx context.Context, c *Client, environmentId string, privateLinkAccessId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		privateLinkAccess, _, err := executePrivateLinkAccessRead(c.netApiContext(ctx), c, environmentId, privateLinkAccessId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Private Link Access %q: %s", privateLinkAccessId, createDescriptiveError(err)), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Private Link Access %q provisioning status to become %q: current status is %q", privateLinkAccessId, stateReady, privateLinkAccess.Status.GetPhase()), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
		if privateLinkAccess.Status.GetPhase() == stateProvisioning || privateLinkAccess.Status.GetPhase() == stateReady {
			return privateLinkAccess, privateLinkAccess.Status.GetPhase(), nil
		} else if privateLinkAccess.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("private Link Access %q provisioning status is %q: %s", privateLinkAccessId, stateFailed, privateLinkAccess.Status.GetErrorMessage())
		}
		// Private Link Access is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("private Link Access %q is an unexpected state %q: %s", privateLinkAccessId, privateLinkAccess.Status.GetPhase(), privateLinkAccess.Status.GetErrorMessage())
	}
}

func networkProvisionStatus(ctx context.Context, c *Client, environmentId string, networkId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		network, _, err := executeNetworkRead(c.netApiContext(ctx), c, environmentId, networkId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Network %q: %s", networkId, createDescriptiveError(err)), map[string]interface{}{networkLoggingKey: networkId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Network %q provisioning status to become %q: current status is %q", networkId, stateReady, network.Status.GetPhase()), map[string]interface{}{networkLoggingKey: networkId})
		if network.Status.GetPhase() == stateProvisioning || network.Status.GetPhase() == stateReady {
			return network, network.Status.GetPhase(), nil
		} else if network.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("network %q provisioning status is %q: %s", networkId, stateFailed, network.Status.GetErrorMessage())
		}
		// Network is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("network %q is an unexpected state %q: %s", networkId, network.Status.GetPhase(), network.Status.GetErrorMessage())
	}
}

func connectorProvisionStatus(ctx context.Context, c *Client, displayName, environmentId, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		connector, _, err := executeConnectorStatusCreate(c.connectApiContext(ctx), c, displayName, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Connector %q=%q: %s", paramDisplayName, displayName, createDescriptiveError(err)))
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Connector %q=%q provisioning status to become %q: current status is %q", paramDisplayName, displayName, stateRunning, connector.Connector.GetState()))
		if connector.Connector.GetState() == stateProvisioning ||
			connector.Connector.GetState() == stateDegraded ||
			connector.Connector.GetState() == stateRunning {
			return connector, connector.Connector.GetState(), nil
		}
		return nil, stateFailed, fmt.Errorf("connector %q=%q provisioning status is %q: %s", paramDisplayName, displayName, connector.Connector.GetState(), connector.Connector.GetTrace())
	}
}

func connectorUpdateStatus(ctx context.Context, c *Client, displayName, environmentId, clusterId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		connector, _, err := executeConnectorStatusCreate(c.connectApiContext(ctx), c, displayName, environmentId, clusterId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Connector %q=%q: %s", paramDisplayName, displayName, createDescriptiveError(err)))
			return nil, stateUnknown, err
		}
		return connector, connector.Connector.GetState(), nil
	}
}

func kafkaMirrorTopicUpdateStatus(ctx context.Context, c *KafkaRestClient, clusterId, linkName, mirrorTopicName string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		mirrorKafkaTopic, _, err := c.apiClient.ClusterLinkingV3Api.ReadKafkaMirrorTopic(c.apiContext(ctx), clusterId, linkName, mirrorTopicName).Execute()
		kafkaMirrorTopicId := createKafkaMirrorTopicId(clusterId, linkName, mirrorTopicName)

		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Kafka Mirror Topic %q: %s", kafkaMirrorTopicId, createDescriptiveError(err)), map[string]interface{}{kafkaMirrorTopicLoggingKey: kafkaMirrorTopicId})
			return nil, stateUnknown, err
		}
		return mirrorKafkaTopic, string(mirrorKafkaTopic.GetMirrorStatus()), nil
	}
}

func peeringProvisionStatus(ctx context.Context, c *Client, environmentId string, peeringId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		peering, _, err := executePeeringRead(c.netApiContext(ctx), c, environmentId, peeringId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Peering %q: %s", peeringId, createDescriptiveError(err)), map[string]interface{}{peeringLoggingKey: peeringId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Peering %q provisioning status to become %q: current status is %q", peeringId, statePendingAccept, peering.Status.GetPhase()), map[string]interface{}{peeringLoggingKey: peeringId})
		if peering.Status.GetPhase() == stateProvisioning || peering.Status.GetPhase() == stateReady || peering.Status.GetPhase() == statePendingAccept {
			return peering, peering.Status.GetPhase(), nil
		} else if peering.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("peering %q provisioning status is %q: %s", peeringId, stateFailed, peering.Status.GetErrorMessage())
		}
		// Peering is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("peering %q is an unexpected state %q: %s", peeringId, peering.Status.GetPhase(), peering.Status.GetErrorMessage())
	}
}

func transitGatewayAttachmentProvisionStatus(ctx context.Context, c *Client, environmentId string, transitGatewayAttachmentId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		transitGatewayAttachment, _, err := executeTransitGatewayAttachmentRead(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Peering %q: %s", transitGatewayAttachmentId, createDescriptiveError(err)), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
			return nil, stateUnknown, err
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for Peering %q provisioning status to become %q: current status is %q", transitGatewayAttachmentId, statePendingAccept, transitGatewayAttachment.Status.GetPhase()), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
		if transitGatewayAttachment.Status.GetPhase() == stateProvisioning || transitGatewayAttachment.Status.GetPhase() == stateReady || transitGatewayAttachment.Status.GetPhase() == statePendingAccept {
			return transitGatewayAttachment, transitGatewayAttachment.Status.GetPhase(), nil
		} else if transitGatewayAttachment.Status.GetPhase() == stateFailed {
			return nil, stateFailed, fmt.Errorf("transit Gateway Attachment %q provisioning status is %q: %s", transitGatewayAttachmentId, stateFailed, transitGatewayAttachment.Status.GetErrorMessage())
		}
		// Peering is in an unexpected state
		return nil, stateUnexpected, fmt.Errorf("transit Gateway Attachment %q is an unexpected state %q: %s", transitGatewayAttachmentId, transitGatewayAttachment.Status.GetPhase(), transitGatewayAttachment.Status.GetErrorMessage())
	}
}

func privateLinkAccessDeleteStatus(ctx context.Context, c *Client, environmentId, privateLinkAccessId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		privateLinkAccess, resp, err := executePrivateLinkAccessRead(c.netApiContext(ctx), c, environmentId, privateLinkAccessId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Private Link Access %q: %s", privateLinkAccessId, createDescriptiveError(err)), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})

			isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
			if isResourceNotFound {
				tflog.Debug(ctx, fmt.Sprintf("Finishing Private Link Access %q deletion process: Received %d status code when reading %q Private Link Access", privateLinkAccessId, resp.StatusCode, privateLinkAccessId), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
				return 0, stateDone, nil
			} else {
				tflog.Debug(ctx, fmt.Sprintf("Exiting Private Link Access %q deletion process: Failed when reading Private Link Access: %s: %s", privateLinkAccessId, createDescriptiveError(err), privateLinkAccess.Status.GetErrorMessage()), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
				return nil, stateFailed, err
			}
		}
		tflog.Debug(ctx, fmt.Sprintf("Performing Private Link Access %q deletion process: Private Link Access %d's status is %q", privateLinkAccessId, resp.StatusCode, privateLinkAccess.Status.GetPhase()), map[string]interface{}{privateLinkAccessLoggingKey: privateLinkAccessId})
		return privateLinkAccess, stateInProgress, nil
	}
}

func peeringDeleteStatus(ctx context.Context, c *Client, environmentId, peeringId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		peering, resp, err := executePeeringRead(c.netApiContext(ctx), c, environmentId, peeringId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Peering %q: %s", peeringId, createDescriptiveError(err)), map[string]interface{}{peeringLoggingKey: peeringId})

			isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
			if isResourceNotFound {
				tflog.Debug(ctx, fmt.Sprintf("Finishing Peering %q deletion process: Received %d status code when reading %q Peering", peeringId, resp.StatusCode, peeringId), map[string]interface{}{peeringLoggingKey: peeringId})
				return 0, stateDone, nil
			} else {
				tflog.Debug(ctx, fmt.Sprintf("Exiting Peering %q deletion process: Failed when reading Peering: %s: %s", peeringId, createDescriptiveError(err), peering.Status.GetErrorMessage()), map[string]interface{}{peeringLoggingKey: peeringId})
				return nil, stateFailed, err
			}
		}
		tflog.Debug(ctx, fmt.Sprintf("Performing Peering %q deletion process: Peering %d's status is %q", peeringId, resp.StatusCode, peering.Status.GetPhase()), map[string]interface{}{peeringLoggingKey: peeringId})
		return peering, stateInProgress, nil
	}
}

func transitGatewayAttachmentDeleteStatus(ctx context.Context, c *Client, environmentId, transitGatewayAttachmentId string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		transitGatewayAttachment, resp, err := executeTransitGatewayAttachmentRead(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading Transit Gateway Attachment %q: %s", transitGatewayAttachmentId, createDescriptiveError(err)), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})

			isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
			if isResourceNotFound {
				tflog.Debug(ctx, fmt.Sprintf("Finishing Transit Gateway Attachment %q deletion process: Received %d status code when reading %q Transit Gateway Attachment", transitGatewayAttachmentId, resp.StatusCode, transitGatewayAttachmentId), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
				return 0, stateDone, nil
			} else {
				tflog.Debug(ctx, fmt.Sprintf("Exiting Transit Gateway Attachment %q deletion process: Failed when reading Transit Gateway Attachment: %s: %s", transitGatewayAttachmentId, createDescriptiveError(err), transitGatewayAttachment.Status.GetErrorMessage()), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
				return nil, stateFailed, err
			}
		}
		tflog.Debug(ctx, fmt.Sprintf("Performing Transit Gateway Attachment %q deletion process: Transit Gateway Attachment %d's status is %q", transitGatewayAttachmentId, resp.StatusCode, transitGatewayAttachment.Status.GetPhase()), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})
		return transitGatewayAttachment, stateInProgress, nil
	}
}

func cloudApiKeySyncStatus(ctx context.Context, c *Client, cloudApiKey, cloudApiSecret string) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		_, resp, err := c.orgClient.EnvironmentsOrgV2Api.ListOrgV2Environments(orgApiContext(ctx, cloudApiKey, cloudApiSecret)).Execute()
		if resp != nil && resp.StatusCode == http.StatusOK {
			tflog.Debug(ctx, fmt.Sprintf("Finishing Cloud API Key %q sync process: Received %d status code when listing environments", cloudApiKey, resp.StatusCode), map[string]interface{}{apiKeyLoggingKey: cloudApiKey})
			return 0, stateDone, nil
			// Status codes for unsynced API Keys might change over time, so it's safer to rely on a timeout to fail
		} else if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized) {
			tflog.Debug(ctx, fmt.Sprintf("Performing Cloud API Key %q sync process: Received %d status code when listing environments", cloudApiKey, resp.StatusCode), map[string]interface{}{apiKeyLoggingKey: cloudApiKey})
			return 0, stateInProgress, nil
		} else if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Exiting Cloud API Key %q sync process: Failed when listing Environments: %s", cloudApiKey, createDescriptiveError(err)), map[string]interface{}{apiKeyLoggingKey: cloudApiKey})
			return nil, stateFailed, fmt.Errorf("error listing Environments using Cloud API Key %q: %s", cloudApiKey, createDescriptiveError(err))
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Exiting Cloud API Key %q sync process: Received unexpected response when listing Environments: %#v", cloudApiKey, resp), map[string]interface{}{apiKeyLoggingKey: cloudApiKey})
			return nil, stateUnexpected, fmt.Errorf("error listing Environments using Kafka API Key %q: received a response with unexpected %d status code", cloudApiKey, resp.StatusCode)
		}
	}
}

func kafkaApiKeySyncStatus(ctx context.Context, c *KafkaRestClient) resource.StateRefreshFunc {
	return func() (result interface{}, s string, err error) {
		_, resp, err := c.apiClient.TopicV3Api.ListKafkaTopics(kafkaRestApiContextWithClusterApiKey(ctx, c.clusterApiKey, c.clusterApiSecret), c.clusterId).Execute()
		if resp != nil && resp.StatusCode == http.StatusOK {
			tflog.Debug(ctx, fmt.Sprintf("Finishing Kafka API Key %q sync process: Received %d status code when listing Kafka Topics", c.clusterApiKey, resp.StatusCode), map[string]interface{}{apiKeyLoggingKey: c.clusterApiKey})
			return 0, stateDone, nil
			// Status codes for unsynced API Keys might change over time, so it's safer to rely on a timeout to fail
			// That said, now Kafka REST API returns http.StatusUnauthorized
		} else if resp != nil && (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized) {
			tflog.Debug(ctx, fmt.Sprintf("Performing Kafka API Key %q sync process: Received %d status code when listing Kafka Topics", c.clusterApiKey, resp.StatusCode), map[string]interface{}{apiKeyLoggingKey: c.clusterApiKey})
			return 0, stateInProgress, nil
		} else if err != nil {
			tflog.Debug(ctx, fmt.Sprintf("Exiting Kafka API Key %q sync process: Failed when listing Kafka Topics: %s", c.clusterApiKey, createDescriptiveError(err)), map[string]interface{}{apiKeyLoggingKey: c.clusterApiKey})
			return nil, stateFailed, fmt.Errorf("error listing Kafka Topics using Kafka API Key %q: %s", c.clusterApiKey, err)
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Exiting Kafka API Key %q sync process: Received unexpected response when listing Kafka Topics: %#v", c.clusterApiKey, resp), map[string]interface{}{apiKeyLoggingKey: c.clusterApiKey})
			return nil, stateUnexpected, fmt.Errorf("error listing Kafka Topics using Kafka API Key %q: received a response with unexpected %d status code", c.clusterApiKey, resp.StatusCode)
		}
	}
}
