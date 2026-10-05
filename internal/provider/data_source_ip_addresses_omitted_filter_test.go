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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestIpAddressesOmittedFilterBlockIsSafe guards the read path when the Optional `filter {}` block
// is omitted entirely — `data "confluent_ip_addresses" "x" {}`.
//
// The read reaches each filter with an unchecked type assertion on
// `d.Get("filter.0.<attr>")`, which reads as if it would panic on a nil when the block is absent.
// It does not: SDKv2's `d.Get` on an address inside a missing block returns the *zero value of
// that address's schema type*, never nil — so a TypeList filter comes back as an empty
// `[]interface{}`, the assertion holds, and `len(...) > 0` correctly declines to send the filter.
// This asserts that directly, so a future template change that swapped the read shape (or a
// schema change that retyped a filter) fails here rather than at a customer's apply.
//
// The published data source has shipped this exact shape since 2023; the end-to-end counterpart is
// TestAccDataSourceIpAddresses, which drives the same code through Terraform core.
func TestIpAddressesOmittedFilterBlockIsSafe(t *testing.T) {
	d := schema.TestResourceDataRaw(t, ipAddressesDataSource().Schema, map[string]interface{}{})

	for _, attr := range []string{paramClouds, paramRegions, paramServices, paramAddressTypes} {
		path := fmt.Sprintf("%s.0.%s", paramFilter, attr)
		raw, ok := d.Get(path).([]interface{})
		if !ok {
			t.Fatalf("d.Get(%q) returned %T, not []interface{} — the generated read would panic", path, d.Get(path))
		}
		if len(raw) != 0 {
			t.Errorf("d.Get(%q) = %#v, want empty", path, raw)
		}
		if got := convertToStringSlice(raw); len(got) != 0 {
			t.Errorf("convertToStringSlice(%q) = %#v, want empty (an empty filter must not be sent)", path, got)
		}
	}
}
