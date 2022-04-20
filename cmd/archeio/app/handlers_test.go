/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO: exhaustive tests for new functionality

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
		handler: MakeHandler(defaultUpstreamRegistry),
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
			// when redirecting, results from k8s.gcr.io
			{
				name:     "/v2/ returns 401 from gcr, with following redirect",
				request:  request{path: "/v2/", redirect: true},
				expected: expected{url: defaultUpstreamRegistry + "/v2/", statusCode: http.StatusUnauthorized},
			},
		},
		tests: defaultTestFuncs(t),
	}
	suite.runTestSuite(t)
}

func TestDoV2(t *testing.T) {
	doV2 := makeV2Handler(defaultUpstreamRegistry)
	suite := &suite{
		handler: http.HandlerFunc(doV2),
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
