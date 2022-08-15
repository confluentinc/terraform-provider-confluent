package provider

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
)

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
