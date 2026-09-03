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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// unknownConfigValue is helper/schema's sentinel for a value that is not resolved at plan time.
// The SDK keeps it in an internal package (internal/configs/hcl2shim.UnknownVariableValue), so it
// cannot be imported here; the constant itself is stable and is what a raw config carries for an
// interpolation whose result is not yet known.
const unknownConfigValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// azurePrivateLinkNetworkState is an existing AZURE PrivateLink network whose zones the API
// returned in a different order than the config lists them. Such a network cannot be created
// through the provider with an explicit zones list, but it can be imported, or created through
// the CLI or API and then adopted -- and its zones are then in state.
//
// Every attribute setNetworkAttributes writes is present, because that is what a refresh leaves
// behind. A sparser state is not merely unrealistic, it changes the code path under test: an
// absent Optional+Computed+ForceNew attribute (cidr, reserved_cidr, zone_info) diffs as
// NewComputed, finalizeDiff turns that into RequiresNew, and helper/schema then re-runs
// CustomizeDiff against a nil state -- so the test would exercise the replacement path while
// appearing to exercise the reorder path.
func azurePrivateLinkNetworkState() *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: "n-abc123",
		Attributes: map[string]string{
			"id":                 "n-abc123",
			"display_name":       "legacy-azure-pl",
			"cloud":              "AZURE",
			"region":             "eastus",
			"connection_types.#": "1",
			"connection_types.0": "PRIVATELINK",
			// API order, deliberately not the config order below.
			"zones.#":                                "3",
			"zones.0":                                "3",
			"zones.1":                                "1",
			"zones.2":                                "2",
			"environment.#":                          "1",
			"environment.0.id":                       "env-abc",
			"cidr":                                   "",
			"reserved_cidr":                          "",
			"zone_info.#":                            "0",
			"dns_config.#":                           "1",
			"dns_config.0.resolution":                "PRIVATE",
			"gateway.#":                              "1",
			"gateway.0.id":                           "gw-abc123",
			"resource_name":                          "crn://confluent.cloud/network=n-abc123",
			"dns_domain":                             "abc123.eastus.azure.confluent.cloud",
			"endpoint_suffix":                        "",
			"zonal_subdomains.%":                     "0",
			"aws.#":                                  "0",
			"gcp.#":                                  "0",
			"azure.#":                                "1",
			"azure.0.private_link_service_aliases.%": "0",
		},
	}
}

func networkConfig(overrides map[string]interface{}) *terraform.ResourceConfig {
	raw := map[string]interface{}{
		"display_name":     "legacy-azure-pl",
		"cloud":            "AZURE",
		"region":           "eastus",
		"connection_types": []interface{}{"PRIVATELINK"},
		"zones":            []interface{}{"1", "2", "3"},
		"environment":      []interface{}{map[string]interface{}{"id": "env-abc"}},
	}
	for k, v := range overrides {
		raw[k] = v
	}
	return terraform.NewResourceConfigRaw(raw)
}

// TestNetworkCustomizeDiffSkipsValidationOnZonesReorder locks in that a zones reorder on an
// existing network is not treated as a replacement.
//
// The API derives zones_info from the user-supplied zones list and may return the list reordered,
// which is the whole reason suppressNetworkZonesReorder exists. Because that reorder makes
// HasChange(paramZones) true, running validateNetworkOnCreateOrReplace first would classify it as
// a replace and re-validate a network that already exists -- rejecting, for instance, an AZURE
// network carrying explicit zones, which plans cleanly on the published provider. The suppression
// therefore has to run first.
func TestNetworkCustomizeDiffSkipsValidationOnZonesReorder(t *testing.T) {
	diff, err := networkResource().Diff(context.Background(), azurePrivateLinkNetworkState(), networkConfig(nil), nil)
	if err != nil {
		t.Fatalf("a zones reorder on an existing network must not fail the plan, got: %s", err)
	}
	if diff != nil {
		if attr, ok := diff.Attributes["zones.#"]; ok && attr.RequiresNew {
			t.Fatalf("a zones reorder must not force replacement, got diff: %#v", diff.Attributes)
		}
	}
}

// TestNetworkCustomizeDiffValidatesOnForceNewReplacement is the other half of the reorder test: it
// locks in that dropping the explicit ForceNew-field check did not stop replacements from being
// validated.
//
// Changing region (Required+ForceNew) makes the first diff pass RequiresNew, so helper/schema nils
// out the state, re-diffs, and re-runs CustomizeDiff -- this time with Id() == "". The invalid
// combination in the config (zones on an AZURE network) must be caught on that second pass, which
// is where the hand-written networkCreate would have caught it.
func TestNetworkCustomizeDiffValidatesOnForceNewReplacement(t *testing.T) {
	_, err := networkResource().Diff(
		context.Background(),
		azurePrivateLinkNetworkState(),
		networkConfig(map[string]interface{}{"region": "westus"}),
		nil,
	)
	if err == nil {
		t.Fatal("a ForceNew replacement must still be validated, got no error")
	}
	if !strings.Contains(err.Error(), "zones can only be specified for AWS networks") {
		t.Fatalf("expected the zones compatibility error, got: %s", err)
	}
}

// TestNetworkCustomizeDiffDefersValidationOnUnknownValues locks in that the cross-field checks are
// skipped while any of the three inputs they read is unknown at plan time.
//
// ResourceDiff.Get cannot distinguish "unknown" from "empty" -- both read back as the zero value --
// so validating an unknown rejects a valid config. Terraform re-plans with the values resolved
// immediately before applying, so the checks still run before the API call.
func TestNetworkCustomizeDiffDefersValidationOnUnknownValues(t *testing.T) {
	testCases := []struct {
		name      string
		overrides map[string]interface{}
	}{
		{
			// cloud unknown reads as "", which matches neither AWS nor GCP, so
			// validateNetworkZones would reject the zones list.
			name: "cloud unknown",
			overrides: map[string]interface{}{
				"cloud": unknownConfigValue,
				"zones": []interface{}{"use1-az1", "use1-az2", "use1-az3"},
			},
		},
		{
			// A wholly unknown connection_types list reads as empty, so validateNetworkZones
			// finds no zone-capable connection type.
			name: "connection_types wholly unknown",
			overrides: map[string]interface{}{
				"cloud":            "AWS",
				"connection_types": unknownConfigValue,
				"zones":            []interface{}{"use1-az1", "use1-az2", "use1-az3"},
			},
		},
		{
			// The count is known and only the element is unknown, so NewValueKnown on the list
			// itself reports known while the element reads as "" -- which verifyListValues then
			// rejects. This is why networkDiffListKnown checks the elements too.
			name: "connection_types element unknown",
			overrides: map[string]interface{}{
				"cloud":            "AWS",
				"connection_types": []interface{}{unknownConfigValue},
				"zones":            []interface{}{"use1-az1", "use1-az2", "use1-az3"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Empty state, so diff.Id() is "" and this is a create.
			_, err := networkResource().Diff(context.Background(), &terraform.InstanceState{}, networkConfig(testCase.overrides), nil)
			if err != nil {
				t.Fatalf("validation must be deferred while a value is unknown, got: %s", err)
			}
		})
	}
}

// TestNetworkCustomizeDiffValidatesOnCreate confirms the unknown-value guard did not disable the
// checks for configs that are fully known -- the cases the hand-written networkCreate rejected.
func TestNetworkCustomizeDiffValidatesOnCreate(t *testing.T) {
	testCases := []struct {
		name          string
		overrides     map[string]interface{}
		expectedError string
	}{
		{
			name:          "zones on an AZURE network",
			overrides:     nil,
			expectedError: "zones can only be specified for AWS networks",
		},
		{
			name: "connection type outside the accepted set",
			overrides: map[string]interface{}{
				"cloud":            "AWS",
				"connection_types": []interface{}{"NOT_A_CONNECTION_TYPE"},
				"zones":            []interface{}{},
			},
			expectedError: "expected NOT_A_CONNECTION_TYPE to be one of",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := networkResource().Diff(context.Background(), &terraform.InstanceState{}, networkConfig(testCase.overrides), nil)
			if err == nil {
				t.Fatalf("expected an error containing %q, got none", testCase.expectedError)
			}
			if !strings.Contains(err.Error(), testCase.expectedError) {
				t.Fatalf("expected an error containing %q, got: %s", testCase.expectedError, err)
			}
		})
	}
}
