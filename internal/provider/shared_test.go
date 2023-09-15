package provider

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"os"
	"sync"
	"testing"
)

type TestContext struct {
	ctx               context.Context
	wiremockContainer testcontainers.Container
	wiremockUrl       string
}

var tc *TestContext
var once sync.Once

func GetTestContext() *TestContext {
	once.Do(func() {
		tc = &TestContext{}
		tc.Setup()
	})
	return tc
}

func (tc *TestContext) Setup() {
	tc.ctx = context.Background()

	port := nat.Port("8080")
	req := testcontainers.ContainerRequest{
		Image:        "wiremock/wiremock:2.32.0-alpine",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForListeningPort(port),
		// docker run -it --rm -p 8080:8080 wiremock/wiremock --verbose
	}
	tc.wiremockContainer, _ = testcontainers.GenericContainer(tc.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	mappedPort, _ := tc.wiremockContainer.MappedPort(tc.ctx, port)

	hostIP, _ := tc.wiremockContainer.Host(tc.ctx)

	tc.wiremockUrl = fmt.Sprintf("http://%s:%s", hostIP, mappedPort.Port())
}

func (tc *TestContext) Teardown() {
	if err := tc.wiremockContainer.Terminate(tc.ctx); err != nil {
		log.Printf("Error while stopping the wiremock/wiremock container: %s", err)
	}
}

func TestMain(t *testing.M) {
	// Setup
	log.Println("TestMain: Starting setup of the wiremock/wiremock container...")
	tc = GetTestContext()
	// You can perform any setup operations here that are common to all tests.
	log.Println("TestMain: the wiremock/wiremock container started")

	defer tc.Teardown()

	// Run tests
	log.Println("TestMain: Running tests...")
	exitCode := t.Run()
	log.Printf("TestMain: Tests completed with exit code %d", exitCode)

	// Teardown
	// You can perform any cleanup operations here that are common to all tests.
	// For example, closing database connections, releasing resources, etc.

	// Exit with the appropriate exit code
	os.Exit(exitCode)
}
