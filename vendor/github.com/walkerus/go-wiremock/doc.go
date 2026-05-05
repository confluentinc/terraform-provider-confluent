// Copyright 2020 go-wiremock maintainers
// license that can be found in the LICENSE file.

/*
Package wiremock is client for WireMock API.
WireMock is a simulator for HTTP-based APIs.
Some might consider it a service virtualization tool or a mock server.

HTTP request:

	POST /example?firstName=Jhon&lastName=Any string other than "Gray" HTTP/1.1
	Host: 0.0.0.0:8080
	x-session: somefingerprintsome
	Content-Type: application/json
	Content-Length: 23

	{"meta": "information"}

With response:

	Status: 400 Bad Request
	"Content-Type": "application/json"
	{"code": 400, "detail": "detail"}

Stub:

	client := wiremock.NewClient("http://0.0.0.0:8080")
	client.StubFor(wiremock.Post(wiremock.URLPathEqualTo("/example")).
		WithQueryParam("firstName", wiremock.EqualTo("Jhon")).
		WithQueryParam("lastName", wiremock.NotMatching("Gray")).
		WithBodyPattern(wiremock.EqualToJson(`{"meta": "information"}`)).
		WithHeader("x-session", wiremock.Matching("^\\S+fingerprint\\S+$")).
		WillReturn(
			`{"code": 400, "detail": "detail"}`,
			map[string]string{"Content-Type": "application/json"},
			400,
		).
		AtPriority(1))

The client should reset all made stubs after tests:

	client := wiremock.NewClient("http://0.0.0.0:8080")
	defer wiremock.Reset()
	client.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/example")))
	client.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/example/1")))
	// ...

To avoid conflicts, you can delete individual rules:

	client := wiremock.NewClient("http://0.0.0.0:8080")
	exampleStubRule := wiremock.Get(wiremock.URLPathEqualTo("/example"))
	client.StubFor(exampleStubRule)
	client.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/example/1")))
	// ...
	client.DeleteStub(exampleStubRule)

You can verify if a request has been made that matches the mapping.

	client := wiremock.NewClient("http://0.0.0.0:8080")
	exampleStubRule := wiremock.Get(wiremock.URLPathEqualTo("/example"))
	client.StubFor(exampleStubRule)
	// ...
	client.Verify(exampleStubRule.Request(), 1)

*/
package wiremock
