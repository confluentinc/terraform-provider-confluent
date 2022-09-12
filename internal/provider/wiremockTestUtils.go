package provider

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
)

type WiremockContainer struct {
	testcontainers.Container
	URI string
}

func setupWiremock(ctx context.Context) (*WiremockContainer, error) {
	port := nat.Port("8080")
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

func createWiremockContainer(ctx context.Context, containerPort string) (testcontainers.Container, error) {
	containerPortTcp := fmt.Sprintf("%s/tcp", containerPort)
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
	return wiremockContainer, err
}

func createWiremockClient(ctx context.Context, wiremockContainer testcontainers.Container, containerPort string) (*wiremock.Client, string, error) {
	host, err := wiremockContainer.Host(ctx)
	if err != nil {
		return nil, "", err
	}
	wiremockHttpMappedPort, err := wiremockContainer.MappedPort(ctx, nat.Port(containerPort))
	if err != nil {
		return nil, "", err
	}

	mockServerUrl := fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	return wiremock.NewClient(mockServerUrl), mockServerUrl, nil
}

func cleanUp(ctx context.Context, wiremockContainer testcontainers.Container, wiremockClient *wiremock.Client) {
	// nolint:errcheck
	wiremockContainer.Terminate(ctx)

	// nolint:errcheck
	wiremockClient.Reset()
	// nolint:errcheck
	wiremockClient.ResetAllScenarios()
}
