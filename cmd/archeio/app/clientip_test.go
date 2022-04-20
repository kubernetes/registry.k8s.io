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
	"net/netip"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		Name        string
		Request     http.Request
		ExpectedIP  netip.Addr
		ExpectError bool
	}{
		{
			Name: "NO X-Forwarded-For",
			Request: http.Request{
				RemoteAddr: "127.0.0.1:8888",
			},
			ExpectedIP: netip.MustParseAddr("127.0.0.1"),
		},
		{
			Name: "X-Forwarded-For without client-supplied",
			Request: http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"8.8.8.8,8.8.8.9"},
				},
				RemoteAddr: "127.0.0.1:8888",
			},
			ExpectedIP: netip.MustParseAddr("8.8.8.8"),
		},
		{
			Name: "X-Forwarded-For with clean client-supplied",
			Request: http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"127.0.0.1,8.8.8.8,8.8.8.9"},
				},
				RemoteAddr: "127.0.0.1:8888",
			},
			ExpectedIP: netip.MustParseAddr("8.8.8.8"),
		},
		{
			Name: "X-Forwarded-For with garbage client-supplied",
			Request: http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"asd;lfkjaasdf;lk,,8.8.8.8,8.8.8.9"},
				},
				RemoteAddr: "127.0.0.1:8888",
			},
			ExpectedIP: netip.MustParseAddr("8.8.8.8"),
		},
		{
			Name: "Bogus crafted non-cloud X-Forwarded-For with no commas",
			Request: http.Request{
				Header: http.Header{
					"X-Forwarded-For": []string{"8.8.8.8"},
				},
				RemoteAddr: "127.0.0.1:8888",
			},
			ExpectError: true,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			ip, err := getClientIP(&tc.Request)
			if err != nil {
				if !tc.ExpectError {
					t.Fatalf("unexpted error: %v", err)
				}
			} else if tc.ExpectError {
				t.Fatal("expected error but err was nil")
			} else if ip != tc.ExpectedIP {
				t.Fatalf("IP does not match expected IP got: %q, expected: %q", ip, tc.ExpectedIP)
			}
		})
	}
}
