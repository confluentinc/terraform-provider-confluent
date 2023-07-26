package provider

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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
