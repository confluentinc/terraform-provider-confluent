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
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The IP filtering quota is enforced by the API and can be raised above 25, so the provider
// must not cap cidr_blocks / ip_groups client-side. These tests run the plan-time validation
// (no API calls), so they fail if a regeneration restores a spec-derived MaxItems: 25.
const ipFilteringOverOldCap = 26

func TestIPGroupCidrBlocksHasNoClientSideMaxItems(t *testing.T) {
	cidrs := make([]interface{}, ipFilteringOverOldCap)
	for i := range cidrs {
		cidrs[i] = fmt.Sprintf("10.%d.0.0/16", i)
	}
	rc := terraform.NewResourceConfigRaw(map[string]interface{}{
		"group_name":  "many-cidrs",
		"cidr_blocks": cidrs,
	})
	if diags := ipGroupResource().Validate(rc); diags.HasError() {
		t.Fatalf("%d cidr_blocks must pass plan-time validation, got: %v", ipFilteringOverOldCap, diags)
	}
}

func TestIPFilterIPGroupsHasNoClientSideMaxItems(t *testing.T) {
	ids := make([]interface{}, ipFilteringOverOldCap)
	for i := range ids {
		ids[i] = fmt.Sprintf("ipg-%05d", i)
	}
	rc := terraform.NewResourceConfigRaw(map[string]interface{}{
		"filter_name":    "many-groups",
		"resource_group": "management",
		"ip_groups":      ids,
	})
	if diags := ipFilterResource().Validate(rc); diags.HasError() {
		t.Fatalf("%d ip_groups must pass plan-time validation, got: %v", ipFilteringOverOldCap, diags)
	}
}

// MinItems stays: the relaxed ceiling must not have loosened the floor.
func TestIPFilteringKeepsMinItems(t *testing.T) {
	if diags := ipGroupResource().Validate(terraform.NewResourceConfigRaw(map[string]interface{}{
		"group_name":  "empty",
		"cidr_blocks": []interface{}{},
	})); !diags.HasError() {
		t.Fatal("an empty cidr_blocks must still be rejected")
	}
	if diags := ipFilterResource().Validate(terraform.NewResourceConfigRaw(map[string]interface{}{
		"filter_name":    "empty",
		"resource_group": "management",
		"ip_groups":      []interface{}{},
	})); !diags.HasError() {
		t.Fatal("an empty ip_groups must still be rejected")
	}
}
