package mockgcs

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

func TestGCSGet(t *testing.T) {
	gcs := New()
	bucket := gcs.AddBucket("test-bucket")

	bucket.AddObject("k1", Object{Contents: []byte("v1")})

	bucket.AddObject("with-override", Object{Contents: []byte("v1"), Override: func(response *http.Response) { response.StatusCode = http.StatusAccepted }})

	grid := []TestRequest{
		{
			URL:            "https://storage.googleapis.com/test-bucket/k1",
			ExpectedStatus: http.StatusOK,
			ExpectedBody:   "v1",
		},
		{
			URL:            "https://storage.googleapis.com/test-bucket/with-override",
			ExpectedStatus: http.StatusAccepted,
			ExpectedBody:   "v1",
		},
		{
			URL:            "https://storage.googleapis.com/test-bucket/bad-key",
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   "",
		},
		{
			URL:            "https://storage.googleapis.com/not-a-bucket/k1",
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   "",
		},
		{
			URL:            "https://incorrecthost.googleapis.com/not-a-bucket/k1",
			ExpectedStatus: http.StatusNotFound,
			ExpectedBody:   "",
		},
	}
	for _, tc := range grid {
		tc := tc
		t.Run(tc.URL, func(t *testing.T) {
			t.Parallel()
			tc.Run(t, gcs.HTTPClient())
		})
	}
}
