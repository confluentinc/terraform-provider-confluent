package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const scenarioStateMaterializedTableDeleteConflicted = "The materialized table delete returned 409"

// TestAccFlinkMaterializedTableDeleteRetriesOn409 checks that a DELETE answered with 409 (pending operation) is retried
// after a 5-6s wait and that the delete then succeeds. The jitter itself is covered by the withJitter unit tests.
func TestAccFlinkMaterializedTableDeleteRetriesOn409(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockTestServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()
	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	const scenarioName = "confluent_flink_materialized_table Delete 409 Retry"
	const displayName = "table_group_by"
	readPath := fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/databases/%s/materialized-tables/%s", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkMaterializedTableDatabase, displayName)

	stub := func(s *wiremock.StubRule) {
		if err := wiremockClient.StubFor(s); err != nil {
			t.Fatal(err)
		}
	}
	createResponse, _ := os.ReadFile("../testdata/flink_materialized_table/create_materialized_table_server_derived_distribution.json")
	stub(wiremock.Post(wiremock.URLPathEqualTo(createFlinkMaterializedTablePath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateMaterializedTableHasBeenCreated).
		WillReturn(string(createResponse), contentTypeJSONHeader, http.StatusCreated))

	readResponse, _ := os.ReadFile("../testdata/flink_materialized_table/read_materialized_table_server_derived_distribution.json")
	stub(wiremock.Get(wiremock.URLPathEqualTo(readPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenCreated).
		WillReturn(string(readResponse), contentTypeJSONHeader, http.StatusOK))

	// First DELETE: 409 while the table's statement still has a pending operation.
	stub(wiremock.Delete(wiremock.URLPathEqualTo(readPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenCreated).
		WillSetStateTo(scenarioStateMaterializedTableDeleteConflicted).
		WillReturn(`{"errors":[{"status":"409","detail":"has a pending operation"}]}`, contentTypeJSONHeader, http.StatusConflict))

	// Retried DELETE: succeeds.
	stub(wiremock.Delete(wiremock.URLPathEqualTo(readPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableDeleteConflicted).
		WillSetStateTo(scenarioStateMaterializedTableHasBeenDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	readDeletedResponse, _ := os.ReadFile("../testdata/flink_materialized_table/read_deleted_materialized_table.json")
	stub(wiremock.Get(wiremock.URLPathEqualTo(readPath)).
		InScenario(scenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenDeleted).
		WillReturn(string(readDeletedResponse), contentTypeJSONHeader, http.StatusNotFound))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			if err := testAccCheckMaterializedTableDestroy(s, mockTestServerUrl); err != nil {
				return err
			}
			return checkDeleteRetriedAfter(mockTestServerUrl, readPath, 5*time.Second, 6*time.Second)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMaterializedTableServerDerivedDistributionConfig(mockTestServerUrl, "test", displayName),
				Check:  testAccCheckMaterializedTableExists("confluent_flink_materialized_table.test"),
			},
		},
	})
}

// checkDeleteRetriedAfter checks WireMock's request journal for exactly two DELETEs of path, the second sent between
// minGap and maxGap after the first (plus a little slack for request latency on a busy machine).
func checkDeleteRetriedAfter(mockServerUrl, path string, minGap, maxGap time.Duration) error {
	resp, err := http.Get(mockServerUrl + "/__admin/requests")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var journal struct {
		Requests []struct {
			Request struct {
				URL        string `json:"url"`
				Method     string `json:"method"`
				LoggedDate int64  `json:"loggedDate"`
			} `json:"request"`
		} `json:"requests"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&journal); err != nil {
		return err
	}
	var deletes []int64
	for _, r := range journal.Requests {
		if r.Request.Method == http.MethodDelete && r.Request.URL == path {
			deletes = append(deletes, r.Request.LoggedDate)
		}
	}
	if len(deletes) != 2 {
		return fmt.Errorf("expected 2 DELETE requests (409, then 204), got %d", len(deletes))
	}
	sort.Slice(deletes, func(i, j int) bool { return deletes[i] < deletes[j] })
	const slack = 1 * time.Second
	if gap := time.Duration(deletes[1]-deletes[0]) * time.Millisecond; gap < minGap || gap > maxGap+slack {
		return fmt.Errorf("expected the retried DELETE %v-%v after the 409, got %v", minGap, maxGap, gap)
	}
	return nil
}
