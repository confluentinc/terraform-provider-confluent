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

// The resource preference tests don't call t.Parallel: the API allows one preference per
// resource, and all of them target the same dedicated cluster.

func TestAccResourcePreferenceLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true, false)
	label := fmt.Sprintf("test_live_resource_preference_%d", rand.Intn(100000))
	address := fmt.Sprintf("confluent_notifications_resource_preference.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePreferenceLiveConfig(env, label, "DISABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "resource", env.clusterCrn()),
					resource.TestCheckResourceAttr(address, "resource_type", "CLUSTER"),
					resource.TestCheckResourceAttr(address, "current_state", "DISABLED"),
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

func TestAccResourcePreferenceUpdateLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true, false)
	label := fmt.Sprintf("test_live_resource_preference_update_%d", rand.Intn(100000))
	address := fmt.Sprintf("confluent_notifications_resource_preference.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePreferenceLiveConfig(env, label, "DISABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "current_state", "DISABLED"),
				),
			},
			{
				Config: testAccResourcePreferenceLiveConfig(env, label, "ENABLED"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNotificationsLiveExists(address),
					resource.TestCheckResourceAttr(address, "resource", env.clusterCrn()),
					resource.TestCheckResourceAttr(address, "current_state", "ENABLED"),
				),
			},
		},
	})
}

func TestAccResourcePreferenceMinimalLive(t *testing.T) {
	env := loadNotificationsLiveEnv(t, true, false)
	label := fmt.Sprintf("test_live_resource_preference_minimal_%d", rand.Intn(100000))
	address := fmt.Sprintf("confluent_notifications_resource_preference.%s", label)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourcePreferenceLiveConfig(env, label, ""),
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

// testAccResourcePreferenceLiveConfig omits current_state when it's empty.
func testAccResourcePreferenceLiveConfig(env notificationsLiveEnv, label, currentState string) string {
	stateLine := ""
	if currentState != "" {
		stateLine = fmt.Sprintf("current_state = %q", currentState)
	}
	return env.providerConfig() + fmt.Sprintf(`
	resource "confluent_notifications_resource_preference" %q {
		resource      = %q
		resource_type = "CLUSTER"
		%s
	}
	`, label, env.clusterCrn(), stateLine)
}
