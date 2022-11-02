package mocks3

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type TestRequest struct {
	URL            string
	ExpectedStatus int
	ExpectedBody   string
}

func (tc *TestRequest) Run(t *testing.T, httpClient *http.Client) {
	response, err := httpClient.Get(tc.URL)
	if err != nil {
		t.Fatalf("unexpected error from request: %v", err)
	}
	if response.StatusCode != tc.ExpectedStatus {
		t.Fatalf(
			"expected status: %v, but got status: %v",
			http.StatusText(tc.ExpectedStatus),
			http.StatusText(response.StatusCode),
		)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("error reading body: %v", err)
	}
	want := tc.ExpectedBody

	got := strings.TrimSpace(string(body))
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("content did not match expected; got %q, want %q\ndiff=%v", got, want, diff)
	}

}

func TestS3Get(t *testing.T) {
	mocks3 := New()
	bucket := mocks3.AddBucket("prod-artifacts-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com")

	bucket.AddObject("k1", Object{Contents: []byte("v1")})

	grid := []TestRequest{
		{
			URL:            "https://prod-artifacts-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com/k1",
			ExpectedStatus: http.StatusOK,
			ExpectedBody:   "v1",
		},
		{
			URL:            "https://prod-artifacts-k8s-io-ap-south-1.s3.dualstack.ap-south-1.amazonaws.com/not-found",
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   "",
		},
		{
			URL:            "https://not-a-bucket.s3.dualstack.ap-south-1.amazonaws.com/k1",
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   "",
		},
	}
	for _, tc := range grid {
		tc := tc
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			tc.Run(t, mocks3.HTTPClient())
		})
	}
}
