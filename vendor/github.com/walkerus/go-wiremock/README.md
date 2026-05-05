# go-wiremock
[![Actions Status](https://github.com/walkerus/go-wiremock/workflows/build/badge.svg)](https://github.com/walkerus/go-wiremock/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/walkerus/go-wiremock)](https://goreportcard.com/report/github.com/walkerus/go-wiremock)

The simple package to stub HTTP resource using [WireMock admin](http://wiremock.org/docs/api/)

## Documentation
[![GoDoc](https://godoc.org/github.com/walkerus/go-wiremock?status.svg)](http://godoc.org/github.com/walkerus/go-wiremock)

## Usage
```
docker run -it --rm -p 8080:8080 rodolpheche/wiremock
```
```go
package main

import (
    "testing"

    "github.com/walkerus/go-wiremock"
)

func TestSome(t *testing.T) {
    wiremockClient := wiremock.NewClient("http://0.0.0.0:8080")
    defer wiremockClient.Reset()
    
    // stubbing POST http://0.0.0.0:8080/example
    wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/example")).
            WithQueryParam("firstName", wiremock.EqualTo("Jhon")).
            WithQueryParam("lastName", wiremock.NotMatching("Black")).
            WithBodyPattern(wiremock.EqualToJson(`{"meta": "information"}`)).
            WithHeader("x-session", wiremock.Matching("^\\S+fingerprint\\S+$")).
            WillReturn(
                `{"code": 400, "detail": "detail"}`,
                map[string]string{"Content-Type": "application/json"},
                400,
            ).
            AtPriority(1))

    // scenario
    defer wiremockClient.ResetAllScenarios()
    wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/status")).
		WillReturn(
			`{"status": null}`,
			map[string]string{"Content-Type": "application/json"},
			200,
		).
		InScenario("Set status").
        WhenScenarioStateIs(wiremock.ScenarioStateStarted))

    wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/state")).
            WithBodyPattern(wiremock.EqualToJson(`{"status": "started"}`)).
            InScenario("Set status").
            WillSetStateTo("Status started"))

    statusStub := wiremock.Get(wiremock.URLPathEqualTo("/status")).
		WillReturn(
			`{"status": "started"}`,
			map[string]string{"Content-Type": "application/json"},
			200,
		).
		InScenario("Set status").
		WhenScenarioStateIs("Status started")
    wiremockClient.StubFor(statusStub)

    //testing code...
    
    verifyResult, _ := wiremockClient.Verify(statusStub.Request(), 1)
    if !verifyResult {
		//...
    }
    
    wiremockClient.DeleteStub(statusStub)
}
```