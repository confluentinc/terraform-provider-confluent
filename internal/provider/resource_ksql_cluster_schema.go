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
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// credentialIdentityBlockSchema is hand-written and referenced by the generated
// resource_ksql_cluster.go via terraform.objectref_schema_overrides. It is kept out of the
// generated file because it carries a ValidateFunc — `^(u-|sa-)` on the nested id — that the
// generator cannot derive: the credential_identity reference is typed by a shared spec
// ObjectReference that carries no per-referent pattern. Referencing the helper preserves the
// plan-time check the published resource enforced, exactly the case objectref_schema_overrides
// exists for (see requiredNetworkSchema).
func credentialIdentityBlockSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The credential_identity to which this belongs. The credential_identity can be one of iam.v2.User, iam.v2.ServiceAccount.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^(u-|sa-)"), "the credential identity must be of the form 'u-' or 'sa-'"),
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}
