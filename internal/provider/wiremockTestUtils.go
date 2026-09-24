package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// wiremockScenarioHookPath is the URL path prefix for stubs that exist only to advance a WireMock
// scenario out of band (see triggerWiremockScenarioHook). Nothing in the provider calls it.
const wiremockScenarioHookPath = "/__test-hooks/"

// triggerWiremockScenarioHook fires a "hook" stub registered under wiremockScenarioHookPath whose
// only job is to move a scenario to its next state. Use it from a TestStep's PreConfig to simulate a
// change made outside Terraform (e.g. a failover flipping the pair's active member) before the
// step's refresh runs. The stub must be registered by the test with
// wiremock.Post(wiremock.URLPathEqualTo(wiremockScenarioHookPath + name)) and the desired
// InScenario / WhenScenarioStateIs / WillSetStateTo transition. Going through a stub rather than
// the scenarios admin API keeps this independent of the WireMock image version.
func triggerWiremockScenarioHook(mockServerUrl, name string) error {
	resp, err := http.Post(mockServerUrl+wiremockScenarioHookPath+name, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("triggering WireMock scenario hook %q: unexpected status %d (is the hook stub registered for the scenario's current state?)", name, resp.StatusCode)
	}
	return nil
}

type WiremockContainer struct {
	testcontainers.Container
	URI string
}

func setupWiremock(ctx context.Context) (*WiremockContainer, error) {
	port := "8080"
	req := testcontainers.ContainerRequest{
		Image:        "wiremock/wiremock:2.32.0-alpine",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForListeningPort(port),
		// docker run -it --rm -p 8080:8080 wiremock/wiremock --verbose
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, port)
	if err != nil {
		return nil, err
	}

	hostIP, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("http://%s:%s", hostIP, mappedPort.Port())

	return &WiremockContainer{Container: container, URI: uri}, nil
}
