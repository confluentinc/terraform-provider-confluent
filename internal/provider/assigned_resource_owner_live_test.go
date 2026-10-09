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
	"strings"
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

// testAccCheckAssignedResourceOwnerLive verifies what the WireMock tests cannot: that creating the
// resource with assigned_resource_owner really granted that principal ResourceOwner on it.
//
// It lists the principal's ResourceOwner role bindings across the whole organization, using the
// organization's CRN followed by "/*" as `confluent iam rbac role-binding list --inclusive` does.
// It then looks for a binding whose CRN ends in the resource's ID. Matching on the ID avoids
// building each resource type's CRN by hand. That is not safe: the API's metadata.resource_name is
// not always the CRN role bindings use, since a service account's has no organization segment.
// It fails if no such binding appears within assignedResourceOwnerRoleBindingTimeout, listing the
// bindings it did find.
func testAccCheckAssignedResourceOwnerLive(resourceName string) resource.TestCheckFunc {
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
		organizationCrn, err := assignedResourceOwnerOrganizationCrnLive(ctx, c)
		if err != nil {
			return fmt.Errorf("reading the organization's CRN: %s", err)
		}

		crnPattern := organizationCrn + "/*"
		principal := fmt.Sprintf("User:%s", owner)
		resourceIdSuffix := "=" + rs.Primary.ID
		deadline := time.Now().Add(assignedResourceOwnerRoleBindingTimeout)
		for {
			roleBindings, resp, err := c.mdsV2Client.RoleBindingsIamV2Api.ListIamV2RoleBindings(c.mdsV2ApiContext(ctx)).
				CrnPattern(crnPattern).
				Principal(principal).
				RoleName(assignedResourceOwnerRoleName).
				Execute()
			if err != nil {
				return fmt.Errorf("listing %s role bindings for %s on %s: %s", assignedResourceOwnerRoleName, principal, crnPattern, createDescriptiveError(err, resp))
			}
			var found []string
			for _, roleBinding := range roleBindings.GetData() {
				if strings.HasSuffix(roleBinding.GetCrnPattern(), resourceIdSuffix) {
					return nil
				}
				found = append(found, roleBinding.GetCrnPattern())
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("no %s role binding for %s on %s after %s; its %s role bindings in the organization: %v",
					assignedResourceOwnerRoleName, principal, rs.Primary.ID, assignedResourceOwnerRoleBindingTimeout, assignedResourceOwnerRoleName, found)
			}
			time.Sleep(assignedResourceOwnerRoleBindingPollInterval)
		}
	}
}

// assignedResourceOwnerOrganizationCrnLive returns the organization's CRN the way the
// confluent_organization data source derives it: from an environment's CRN, without the
// environment segment.
func assignedResourceOwnerOrganizationCrnLive(ctx context.Context, c *Client) (string, error) {
	environments, resp, err := c.orgV2Client.EnvironmentsOrgV2Api.ListOrgV2Environments(c.orgV2ApiContext(ctx)).Execute()
	if err != nil {
		return "", createDescriptiveError(err, resp)
	}
	if len(environments.GetData()) == 0 {
		return "", fmt.Errorf("no environments found")
	}
	return extractOrgResourceName(environments.GetData()[0].Metadata.GetResourceName())
}
