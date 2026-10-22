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
	"testing"

	networkinggatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestSetGatewayAttributesGcpEgressPrivateServiceConnect(t *testing.T) {
	spec := networkinggatewayv1.NewNetworkingV1GatewaySpec()
	spec.SetDisplayName("egress")
	spec.SetEnvironment(*networkinggatewayv1.NewObjectReference("env-abc123", "", ""))
	spec.SetConfig(networkinggatewayv1.NetworkingV1GcpEgressPrivateServiceConnectGatewaySpecAsNetworkingV1GatewaySpecConfigOneOf(
		networkinggatewayv1.NewNetworkingV1GcpEgressPrivateServiceConnectGatewaySpec("GcpEgressPrivateServiceConnectGatewaySpec", "us-east4")))

	gcpStatus := networkinggatewayv1.NewNetworkingV1GcpEgressPrivateServiceConnectGatewayStatus("GcpEgressPrivateServiceConnectGatewayStatus")
	gcpStatus.SetProject("confluent-project")
	status := networkinggatewayv1.NewNetworkingV1GatewayStatus("READY")
	status.SetCloudGateway(networkinggatewayv1.NetworkingV1GcpEgressPrivateServiceConnectGatewayStatusAsNetworkingV1GatewayStatusCloudGatewayOneOf(gcpStatus))

	gateway := networkinggatewayv1.NewNetworkingV1Gateway()
	gateway.SetId("gw-abc123")
	gateway.SetSpec(*spec)
	gateway.SetStatus(*status)

	d := schema.TestResourceDataRaw(t, gatewayResource().Schema, map[string]interface{}{})
	if _, err := setGatewayAttributes(d, *gateway); err != nil {
		t.Fatalf("setGatewayAttributes: %v", err)
	}
	if got := d.Get("gcp_egress_private_service_connect_gateway.0.region"); got != "us-east4" {
		t.Errorf("region = %v, want us-east4", got)
	}
	if got := d.Get("gcp_egress_private_service_connect_gateway.0.project"); got != "confluent-project" {
		t.Errorf("project = %v, want confluent-project", got)
	}
}
