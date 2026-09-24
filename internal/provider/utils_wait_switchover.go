// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	switchoverv1 "github.com/confluentinc/ccloud-sdk-go-v2/switchover/v1"
)

// The Switchover API is asynchronous: create and delete return 202 and the pair/endpoint keep
// changing afterwards. Terraform's dependency ordering only helps if Create/Delete return once the
// server has actually finished, so the two operations that have a dependent waiting on them (pair
// create -> endpoint create; endpoint delete -> pair delete) wait for a terminal state the same way
// confluent_kafka_cluster does (see waitForKafkaClusterToProvision / waitForKafkaClusterToBeDeleted).

// waitForSwitchoverPairToProvision blocks until a newly created pair leaves PROVISIONING. Without it
// a confluent_switchover_endpoint created in the same apply is rejected with a 409 because its
// parent is still provisioning.
func waitForSwitchoverPairToProvision(ctx context.Context, c *Client, environmentId, pairId string) error {
	delay, pollInterval := getDelayAndPollInterval(5*time.Second, 15*time.Second, c.isAcceptanceTestMode)
	stateConf := &resource.StateChangeConf{
		Pending:      []string{stateProvisioning},
		Target:       []string{stateReadyToFailover},
		Refresh:      switchoverPairProvisionStatus(ctx, c, environmentId, pairId),
		Timeout:      switchoverPairCreateTimeout,
		Delay:        delay,
		PollInterval: pollInterval,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for switchover pair %q to become %q", pairId, stateReadyToFailover), map[string]interface{}{switchoverPairLoggingKey: pairId})
	_, err := stateConf.WaitForStateContext(ctx)
	return err
}

func switchoverPairProvisionStatus(ctx context.Context, c *Client, environmentId, pairId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		pair, resp, err := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.GetSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), pairId).Environment(environmentId).Execute()
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error reading switchover pair %q: %s", pairId, createDescriptiveError(err, resp)), map[string]interface{}{switchoverPairLoggingKey: pairId})
			return nil, stateUnknown, err
		}

		phase := pair.Status.GetPhase()
		tflog.Debug(ctx, fmt.Sprintf("Waiting for switchover pair %q to become %q: current phase is %q", pairId, stateReadyToFailover, phase), map[string]interface{}{switchoverPairLoggingKey: pairId})
		switch phase {
		case stateProvisioning, stateReadyToFailover:
			return pair, phase, nil
		case stateFailed:
			return nil, stateFailed, fmt.Errorf("switchover pair %q provisioning failed: %s", pairId, formatSwitchoverPairConditions(pair.Status.GetConditions()))
		}
		return nil, stateUnexpected, fmt.Errorf("switchover pair %q is in an unexpected phase %q", pairId, phase)
	}
}

// waitForSwitchoverEndpointToBeDeleted blocks until GET on the endpoint returns 404. Without it the
// parent pair's delete, which Terraform issues right after this one returns, is rejected with a 409
// because the endpoint still exists on the server.
func waitForSwitchoverEndpointToBeDeleted(ctx context.Context, c *Client, environmentId, endpointId string) error {
	delay, pollInterval := getDelayAndPollInterval(5*time.Second, 15*time.Second, c.isAcceptanceTestMode)
	stateConf := &resource.StateChangeConf{
		Pending: []string{stateInProgress},
		Target:  []string{stateDone},
		Refresh: func() (interface{}, string, error) {
			endpoint, resp, err := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.GetSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx), endpointId).Environment(environmentId).Execute()
			if err != nil {
				if isNonKafkaRestApiResourceNotFound(resp) {
					// Result (the 1st argument) can't be nil
					return 0, stateDone, nil
				}
				return nil, stateFailed, err
			}
			return endpoint, stateInProgress, nil
		},
		Timeout:      switchoverEndpointDeleteTimeout,
		Delay:        delay,
		PollInterval: pollInterval,
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for switchover endpoint %q to be deleted", endpointId), map[string]interface{}{switchoverEndpointLoggingKey: endpointId})
	_, err := stateConf.WaitForStateContext(ctx)
	return err
}

func formatSwitchoverPairConditions(conditions []switchoverv1.SwitchoverV1Condition) string {
	if len(conditions) == 0 {
		return "no conditions reported"
	}
	parts := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		part := fmt.Sprintf("%s=%s", condition.GetType(), condition.GetStatus())
		if reason := condition.GetReason(); reason != "" {
			part += fmt.Sprintf(" (%s)", reason)
		}
		if message := condition.GetMessage(); message != "" {
			part += ": " + message
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}
