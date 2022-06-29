package provider

import (
	"net/http"
)

type ItsActuallyJsonRoundTripper struct {
	Transport http.RoundTripper
}

func (t *ItsActuallyJsonRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var transport http.RoundTripper
	if t.Transport != nil {
		transport = t.Transport
	} else {
		transport = http.DefaultTransport
	}

	resp, err := transport.RoundTrip(req)
	if resp != nil {
		resp.Header.Set("Content-Type", "application/json")
	}

	return resp, err
}
