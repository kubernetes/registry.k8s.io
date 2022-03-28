package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type request struct {
	path     string
	redirect bool
}

type expected struct {
	url        string
	statusCode int
}

type scenario struct {
	request  request
	expected expected
}

var (
	defaultUpstreamRegistry = "https://k8s.gcr.io"
)

type suite struct {
	handler   http.Handler
	scenarios []scenario
	tests     []func(resp *http.Response, scenario scenario)
}

func (s *suite) runTestSuite(t *testing.T) {
	for _, sc := range s.scenarios {
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
	}
}

func TestMakeHandler(t *testing.T) {
	suite := &suite{
		handler: makeHandler(defaultUpstreamRegistry),
		scenarios: []scenario{
			{
				request:  request{path: "/", redirect: false},
				expected: expected{url: "/", statusCode: http.StatusNotFound},
			},
			// when not redirecting
			{
				request:  request{path: "/v2/", redirect: false},
				expected: expected{url: "/v2/", statusCode: http.StatusPermanentRedirect},
			},
			{
				request:  request{path: "/v1/", redirect: false},
				expected: expected{url: "/v1/", statusCode: http.StatusPermanentRedirect},
			},
			// when redirecting, results from k8s.gcr.io
			{
				request:  request{path: "/v2/", redirect: true},
				expected: expected{url: "/v2/", statusCode: http.StatusUnauthorized},
			},
			{
				request:  request{path: "/v1/", redirect: true},
				expected: expected{url: "/v1/", statusCode: http.StatusNotFound},
			},
		},
		tests: []func(resp *http.Response, sc scenario){
			func(resp *http.Response, sc scenario) {
				if resp.StatusCode != sc.expected.statusCode {
					t.Errorf("Expected status code '%v' but received '%v', scenario: %#v, resp: %#v", resp.StatusCode, sc.expected.statusCode, sc, resp)
				}
			},
			func(resp *http.Response, sc scenario) {
				if resp.Request.URL.Path != sc.expected.url {
					t.Errorf("Expected path '%v' but received '%v', scenario: %#v, resp: %#v", resp.Request.URL.Path, sc.expected.url, sc, resp)
				}
			},
		},
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
				request:  request{path: "/v2/", redirect: false},
				expected: expected{url: "/v2/", statusCode: http.StatusPermanentRedirect},
			},
		},
		tests: []func(resp *http.Response, sc scenario){
			func(resp *http.Response, sc scenario) {
				if resp.StatusCode != sc.expected.statusCode {
					t.Errorf("Expected status code '%v' but received '%v', scenario: %#v, resp: %#v", resp.StatusCode, sc.expected.statusCode, sc, resp)
				}
			},
			func(resp *http.Response, sc scenario) {
				if resp.Request.URL.Path != sc.expected.url {
					t.Errorf("Expected path '%v' but received '%v', scenario: %#v, resp: %#v", resp.Request.URL.Path, sc.expected.url, sc, resp)
				}
			},
		},
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
				request:  request{path: "/v1/", redirect: false},
				expected: expected{url: "/v1/", statusCode: http.StatusPermanentRedirect},
			},
		},
		tests: []func(resp *http.Response, sc scenario){
			func(resp *http.Response, sc scenario) {
				if resp.StatusCode != sc.expected.statusCode {
					t.Errorf("Expected status code '%v' but received '%v', scenario: %#v, resp: %#v", resp.StatusCode, sc.expected.statusCode, sc, resp)
				}
			},
			func(resp *http.Response, sc scenario) {
				if resp.Request.URL.Path != sc.expected.url {
					t.Errorf("Expected path '%v' but received '%v', scenario: %#v, resp: %#v", resp.Request.URL.Path, sc.expected.url, sc, resp)
				}
			},
		},
	}
	suite.runTestSuite(t)
}
