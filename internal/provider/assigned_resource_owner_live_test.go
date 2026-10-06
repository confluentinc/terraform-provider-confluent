//go:build live_test

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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	assignedResourceOwnerRoleName = "ResourceOwner"
	// Role bindings are eventually consistent, so the grant made by the create call is polled for
	// rather than expected on the first read.
	assignedResourceOwnerRoleBindingTimeout      = time.Minute
	assignedResourceOwnerRoleBindingPollInterval = 5 * time.Second
)

// assignedResourceOwnerCrnFunc returns the CRN of the resource in rs, read from the API's
// metadata.resource_name rather than assembled by hand, since each resource's CRN has a different
// shape.
type assignedResourceOwnerCrnFunc func(ctx context.Context, c *Client, rs *terraform.ResourceState) (string, error)

// testAccCheckAssignedResourceOwnerLive verifies what the WireMock tests cannot: that creating the
// resource with assigned_resource_owner really granted that principal ResourceOwner on it. It
// lists role bindings for the principal, the role and the resource's CRN, and fails if none
// appears within assignedResourceOwnerRoleBindingTimeout.
func testAccCheckAssignedResourceOwnerLive(resourceName string, crnOf assignedResourceOwnerCrnFunc) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}
		owner := rs.Primary.Attributes[paramAssignedResourceOwner]
		if owner == "" {
			return fmt.Errorf("%s has no %s in state", resourceName, paramAssignedResourceOwner)
		}

		c := testAccProvider.Meta().(*Client)
		ctx := context.Background()
		crn, err := crnOf(ctx, c, rs)
		if err != nil {
			return fmt.Errorf("reading the CRN of %s: %s", resourceName, err)
		}

		principal := fmt.Sprintf("User:%s", owner)
		deadline := time.Now().Add(assignedResourceOwnerRoleBindingTimeout)
		for {
			roleBindings, resp, err := c.mdsV2Client.RoleBindingsIamV2Api.ListIamV2RoleBindings(c.mdsV2ApiContext(ctx)).
				CrnPattern(crn).
				Principal(principal).
				RoleName(assignedResourceOwnerRoleName).
				Execute()
			if err != nil {
				return fmt.Errorf("listing %s role bindings for %s on %s: %s", assignedResourceOwnerRoleName, principal, crn, createDescriptiveError(err, resp))
			}
			if len(roleBindings.GetData()) > 0 {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("no %s role binding for %s on %s after %s", assignedResourceOwnerRoleName, principal, crn, assignedResourceOwnerRoleBindingTimeout)
			}
			time.Sleep(assignedResourceOwnerRoleBindingPollInterval)
		}
	}
}
