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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Both declarations below used to live in resource_peering.go / data_source_peering.go, which
// confluent_transit_gateway_attachment borrowed them from. Those two files are now generated, and
// the generated templates need neither: variant fields are addressed inline rather than through a
// package-level flattened-key var, and the network block is rendered inline rather than via a
// shared helper. Overwriting them therefore deletes both symbols out from under their remaining
// caller.
//
// They live in this companion file rather than in the transit_gateway_attachment files themselves
// because those are being generated too (provider #1234). A generated file carries a DO NOT EDIT
// header and is overwritten wholesale, so a declaration parked there would be silently dropped by
// the next regeneration; this file is hand-written and regeneration never touches it. Once
// confluent_transit_gateway_attachment is generated, nothing references either symbol and this
// file should be deleted outright.

// paramAwsRoutes is the flattened-key path to the AWS variant's routes list, byte-identical to the
// declaration it replaces.
var paramAwsRoutes = fmt.Sprintf("%s.0.%s", paramAws, paramRoutes)

// networkDataSourceSchema is byte-identical to the declaration it replaces.
func networkDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}
