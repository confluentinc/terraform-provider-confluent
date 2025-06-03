// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceIPGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIpGroupRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The ID of the IP group.",
				ExactlyOneOf: []string{paramId, paramGroupName},
			},
			paramGroupName: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "A human readable name for an IP Group.",
				ExactlyOneOf: []string{paramId, paramGroupName},
			},
			paramCIDRBlocks: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "A set of CIDR blocks to include in the IP group.",
			},
		},
	}
}

func dataSourceIpGroupRead(d *schema.ResourceData, m interface{}) error {
	// Implement data source read logic here
	return nil
}
