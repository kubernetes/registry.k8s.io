package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	defaultUpstreamRegistry = "https://k8s.gcr.io"
)

type request struct {
	path     string
	redirect bool
}

type expected struct {
	url        string
	path       string
	statusCode int
}

type scenario struct {
	name     string
	request  request
	expected expected
}

type suite struct {
	handler   http.Handler
	scenarios []scenario
	tests     []func(resp *http.Response, scenario scenario)
}

func (s *suite) runTestSuite(t *testing.T) {
	t.Run("test suite", func(t *testing.T) {
		for _, sc := range s.scenarios {
			t.Run(sc.name, func(t *testing.T) {
				t.Parallel()
				server := httptest.NewServer(s.handler)
				defer server.Close()
				client := server.Client()
				if sc.request.redirect == false {
					client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					}
				}
				resp, err := client.Get(server.URL + sc.request.path)
				if err != nil {
					t.Errorf("Error requesting fake backend, %v", err)
				}
				for _, test := range s.tests {
					test(resp, sc)
				}
			})
		}
	})
}

func defaultTestFuncs(t *testing.T) []func(resp *http.Response, sc scenario) {
	return []func(resp *http.Response, sc scenario){
		func(resp *http.Response, sc scenario) {
			if resp.StatusCode != sc.expected.statusCode {
				t.Errorf("Expected status code '%v' but received '%v', scenario: %#v, resp: %#v", resp.StatusCode, sc.expected.statusCode, sc, resp)
			}
		},
		func(resp *http.Response, sc scenario) {
			if sc.expected.path != "" && resp.Request.URL.Path != sc.expected.path {
				t.Errorf("Expected path '%v' but received '%v', scenario: %#v, resp: %#v", resp.Request.URL.Path, sc.expected.url, sc, resp)
			}
			if sc.expected.url != "" && defaultUpstreamRegistry+resp.Request.URL.Path != sc.expected.url {
				t.Errorf("Expected url '%v' but received '%v', scenario: %#v, resp: %#v", defaultUpstreamRegistry+resp.Request.URL.Path, sc.expected.url, sc, resp)
			}
		},
	}
}

func TestMakeHandler(t *testing.T) {
	suite := &suite{
		handler: makeHandler(defaultUpstreamRegistry),
		scenarios: []scenario{
			{
				name:     "root is not found",
				request:  request{path: "/", redirect: false},
				expected: expected{path: "/", statusCode: http.StatusNotFound},
			},
			// when not redirecting
			{
				name:     "/v2/ returns 308 without following redirect",
				request:  request{path: "/v2/", redirect: false},
				expected: expected{url: defaultUpstreamRegistry + "/v2/", statusCode: http.StatusPermanentRedirect},
			},
			{
				name:     "/v1/ returns 308 without following redirect",
				request:  request{path: "/v1/", redirect: false},
				expected: expected{url: defaultUpstreamRegistry + "/v1/", statusCode: http.StatusPermanentRedirect},
			},
			// when redirecting, results from k8s.gcr.io
			{
				name:     "/v2/ returns 401 from gcr, with following redirect",
				request:  request{path: "/v2/", redirect: true},
				expected: expected{url: defaultUpstreamRegistry + "/v2/", statusCode: http.StatusUnauthorized},
			},
			{
				name:     "/v1/ returns 404 from gcr, with following redirect",
				request:  request{path: "/v1/", redirect: true},
				expected: expected{url: defaultUpstreamRegistry + "/v1/", statusCode: http.StatusNotFound},
			},
		},
		tests: defaultTestFuncs(t),
	}
	suite.runTestSuite(t)
}

func TestDoV2(t *testing.T) {
	suite := &suite{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			doV2(w, r, defaultUpstreamRegistry)
		}),
		scenarios: []scenario{
			{
				name:     "v2 handler returns 308 without following redirect",
				request:  request{path: "/v2/", redirect: false},
				expected: expected{url: defaultUpstreamRegistry + "/v2/", statusCode: http.StatusPermanentRedirect},
			},
		},
		tests: defaultTestFuncs(t),
	}
	suite.runTestSuite(t)
}

func TestDoV1(t *testing.T) {
	suite := &suite{
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			doV1(w, r, defaultUpstreamRegistry)
		}),
		scenarios: []scenario{
			{
				name:     "v1 handler returns 308 without following redirect",
				request:  request{path: "/v1/", redirect: false},
				expected: expected{url: defaultUpstreamRegistry + "/v1/", statusCode: http.StatusPermanentRedirect},
			},
		},
		tests: defaultTestFuncs(t),
	}
	suite.runTestSuite(t)
}
