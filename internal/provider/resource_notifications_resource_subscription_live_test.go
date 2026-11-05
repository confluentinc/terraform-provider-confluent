//go:build live_test && (all || core)

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
	"math/rand"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// The resource subscription tests don't call t.Parallel: all of them subscribe the same
// dedicated cluster to the same notification type, which the API allows only once.

func TestAccResourceSubscriptionLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true)
	randomSuffix := rand.Intn(100000)
	label := fmt.Sprintf("test_live_resource_subscription_%d", randomSuffix)
	address := fmt.Sprintf("confluent_notifications_resource_subscription.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSubscriptionLiveConfig(env, label, randomSuffix, "ENABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "resource", env.clusterCrn()),
					resource.TestCheckResourceAttr(address, "resource_type", "CLUSTER"),
					resource.TestCheckResourceAttr(address, "notification_type.0.id", env.notificationTypeId),
					resource.TestCheckResourceAttr(address, "integrations.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(address, "integrations.*", notificationsLiveIntegration, "id"),
					resource.TestCheckResourceAttr(address, "current_state", "ENABLED"),
				),
			},
			{
				ResourceName:      address,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceSubscriptionUpdateLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true)
	randomSuffix := rand.Intn(100000)
	label := fmt.Sprintf("test_live_resource_subscription_update_%d", randomSuffix)
	address := fmt.Sprintf("confluent_notifications_resource_subscription.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSubscriptionLiveConfig(env, label, randomSuffix, "ENABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "current_state", "ENABLED"),
				),
			},
			{
				Config: testAccResourceSubscriptionLiveConfig(env, label, randomSuffix, "DISABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "resource", env.clusterCrn()),
					resource.TestCheckResourceAttr(address, "current_state", "DISABLED"),
				),
			},
		},
	})
}

func TestAccResourceSubscriptionMinimalLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true)
	randomSuffix := rand.Intn(100000)
	label := fmt.Sprintf("test_live_resource_subscription_minimal_%d", randomSuffix)
	address := fmt.Sprintf("confluent_notifications_resource_subscription.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSubscriptionLiveConfig(env, label, randomSuffix, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "resource", env.clusterCrn()),
					resource.TestCheckResourceAttr(address, "resource_type", "CLUSTER"),
					resource.TestCheckResourceAttrSet(address, "current_state"),
				),
			},
		},
	})
}

// testAccResourceSubscriptionLiveConfig omits current_state when it's empty.
func testAccResourceSubscriptionLiveConfig(env notificationsLiveEnv, label string, randomSuffix int, currentState string) string {
	stateLine := ""
	if currentState != "" {
		stateLine = fmt.Sprintf("current_state = %q", currentState)
	}
	return env.providerConfig() + env.integrationConfig(randomSuffix) + fmt.Sprintf(`
	resource "confluent_notifications_resource_subscription" %q {
		resource      = %q
		resource_type = "CLUSTER"
		notification_type {
			id = %q
		}
		integrations = [%s.id]
		%s
	}
	`, label, env.clusterCrn(), env.notificationTypeId, notificationsLiveIntegration, stateLine)
}
