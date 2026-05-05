package wiremock

import (
	"encoding/json"
)

// A Request is the part of StubRule describing the matching of the http request
type Request struct {
	urlMatcher           URLMatcherInterface
	method               string
	headers              map[string]ParamMatcherInterface
	queryParams          map[string]ParamMatcherInterface
	cookies              map[string]ParamMatcherInterface
	bodyPatterns         []ParamMatcher
	basicAuthCredentials *struct {
		username string
		password string
	}
}

// NewRequest constructs minimum possible Request
func NewRequest(method string, urlMatcher URLMatcherInterface) *Request {
	return &Request{
		method:     method,
		urlMatcher: urlMatcher,
	}
}

// WithMethod is fluent-setter for http verb
func (r *Request) WithMethod(method string) *Request {
	r.method = method
	return r
}

// WithURLMatched is fluent-setter url matcher
func (r *Request) WithURLMatched(urlMatcher URLMatcherInterface) *Request {
	r.urlMatcher = urlMatcher
	return r
}

// WithBodyPattern adds body pattern to list
func (r *Request) WithBodyPattern(matcher ParamMatcher) *Request {
	r.bodyPatterns = append(r.bodyPatterns, matcher)
	return r
}

// WithBasicAuth adds basic auth credentials to Request
func (r *Request) WithBasicAuth(username, password string) *Request {
	r.basicAuthCredentials = &struct {
		username string
		password string
	}{
		username: username,
		password: password,
	}
	return r
}

// WithQueryParam add param to query param list
func (r *Request) WithQueryParam(param string, matcher ParamMatcherInterface) *Request {
	if r.queryParams == nil {
		r.queryParams = map[string]ParamMatcherInterface{}
	}

	r.queryParams[param] = matcher
	return r
}

// WithHeader add header to header list
func (r *Request) WithHeader(header string, matcher ParamMatcherInterface) *Request {
	if r.headers == nil {
		r.headers = map[string]ParamMatcherInterface{}
	}

	r.headers[header] = matcher
	return r
}

// WithCookie is fluent-setter for cookie
func (r *Request) WithCookie(cookie string, matcher ParamMatcherInterface) *Request {
	if r.cookies == nil {
		r.cookies = map[string]ParamMatcherInterface{}
	}

	r.cookies[cookie] = matcher
	return r
}

// MarshalJSON gives valid JSON or error.
func (r *Request) MarshalJSON() ([]byte, error) {
	request := map[string]interface{}{
		"method":                        r.method,
		string(r.urlMatcher.Strategy()): r.urlMatcher.Value(),
	}
	if len(r.bodyPatterns) > 0 {
		bodyPatterns := make([]map[ParamMatchingStrategy]string, len(r.bodyPatterns))
		for i, bodyPattern := range r.bodyPatterns {
			bodyPatterns[i] = map[ParamMatchingStrategy]string{
				bodyPattern.Strategy(): bodyPattern.Value(),
			}
		}
		request["bodyPatterns"] = bodyPatterns
	}
	if len(r.headers) > 0 {
		headers := make(map[string]map[ParamMatchingStrategy]string, len(r.bodyPatterns))
		for key, header := range r.headers {
			headers[key] = map[ParamMatchingStrategy]string{
				header.Strategy(): header.Value(),
			}
		}
		request["headers"] = headers
	}
	if len(r.cookies) > 0 {
		cookies := make(map[string]map[ParamMatchingStrategy]string, len(r.cookies))
		for key, cookie := range r.cookies {
			cookies[key] = map[ParamMatchingStrategy]string{
				cookie.Strategy(): cookie.Value(),
			}
		}
		request["cookies"] = cookies
	}
	if len(r.queryParams) > 0 {
		params := make(map[string]map[ParamMatchingStrategy]string, len(r.queryParams))
		for key, param := range r.queryParams {
			params[key] = map[ParamMatchingStrategy]string{
				param.Strategy(): param.Value(),
			}
		}
		request["queryParameters"] = params
	}

	if r.basicAuthCredentials != nil {
		request["basicAuthCredentials"] = map[string]string{
			"password": r.basicAuthCredentials.password,
			"username": r.basicAuthCredentials.username,
		}
	}

	return json.Marshal(request)
}
