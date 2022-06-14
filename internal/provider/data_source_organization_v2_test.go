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
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	organizationDataSourceScenarioName = "confluent_organization_v2 Data Source Lifecycle"
	organizationDataSourceLabel        = "test_organization_data_source_label"

	expectedOrgResourceName = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	expectedOrgId           = "1111aaaa-11aa-11aa-11aa-111111aaaaaa"
)

var fullOrganizationDataSourceLabel = fmt.Sprintf("data.confluent_organization_v2.%s", organizationDataSourceLabel)

func TestAccDataSourceOrganization(t *testing.T) {
	containerPort := "8080"
	containerPortTcp := fmt.Sprintf("%s/tcp", containerPort)
	ctx := context.Background()
	listeningPort := wait.ForListeningPort(nat.Port(containerPortTcp))
	req := testcontainers.ContainerRequest{
		Image:        "rodolpheche/wiremock",
		ExposedPorts: []string{containerPortTcp},
		WaitingFor:   listeningPort,
	}
	wiremockContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	// nolint:errcheck
	defer wiremockContainer.Terminate(ctx)

	host, err := wiremockContainer.Host(ctx)
	require.NoError(t, err)

	wiremockHttpMappedPort, err := wiremockContainer.MappedPort(ctx, nat.Port(containerPort))
	require.NoError(t, err)

	confluentCloudBaseUrl := fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	wiremockClient := wiremock.NewClient(confluentCloudBaseUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readEnvironmentsResponse, _ := ioutil.ReadFile("../testdata/organization/read_environments.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(organizationDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readEnvironmentsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceOrganizationConfig(confluentCloudBaseUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrganizationExists(fullOrganizationDataSourceLabel),
					resource.TestCheckResourceAttr(fullOrganizationDataSourceLabel, "id", expectedOrgId),
					resource.TestCheckResourceAttr(fullOrganizationDataSourceLabel, "resource_name", expectedOrgResourceName),
				),
			},
		},
	})
}

func testAccCheckDataSourceOrganizationConfig(confluentCloudBaseUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_organization_v2" "%s" {
	}
	`, confluentCloudBaseUrl, organizationDataSourceLabel)
}

func testAccCheckOrganizationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s organization has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s organization", n)
		}

		return nil
	}
}
