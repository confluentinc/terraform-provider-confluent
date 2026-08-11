package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

// TestAccCiReportCanary fails on purpose. It exists only so this PR's CI run produces a real
// failed acceptance test, which is the only way to see what the Semaphore Test Report and the
// tflogs artifact actually contain for a failure. DELETE THIS FILE BEFORE MERGING.
//
// It is modelled on TestAccDataSourceGroup so that it exercises the same path a real failure
// takes: terraform actually runs and emits TF_LOG=debug output, and the assertion fails only
// afterwards. resource.Test skips unless TF_ACC is set, so `make test` skips this and `make all`
// still reaches `make testacc`, which is the target that sets TF_LOG_PATH_MASK.
func TestAccCiReportCanary(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readIPGroupResponse, _ := os.ReadFile("../testdata/ip_group/read_created_ip_group.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullIPGroupResourceLabel := fmt.Sprintf("data.confluent_ip_group.%s", testIPGroupResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIPGroup(mockServerUrl, testIPGroupID, testIPGroupResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Deliberately wrong. The stub returns testIPGroupName.
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramGroupName, "deliberately-wrong-value-to-exercise-ci-test-reporting"),
				),
			},
		},
	})
}
