package app

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"testing/iotest"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"k8s.io/registry.k8s.io/cmd/redirectserver/pkg/mockgcs"
)

func TestContentCache(t *testing.T) {
	ctx := context.Background()

	mockGCS := mockgcs.New()

	k1Count := 0

	root := GCSKey{Bucket: "some-bucket"}
	bucket := mockGCS.AddBucket(root.Bucket)
	bucket.AddObject("k1", mockgcs.Object{
		Contents: []byte("v1"),
		Override: func(response *http.Response) {
			k1Count++
		},
	})
	bucket.AddObject("inject-error", mockgcs.Object{
		Override: func(response *http.Response) {
			response.Body = io.NopCloser(iotest.ErrReader(fmt.Errorf("internal error")))
		},
	})

	storageClient, err := storage.NewClient(ctx, option.WithHTTPClient(mockGCS.HTTPClient()))
	if err != nil {
		t.Fatalf("error from storage.NewClient: %v", err)
	}
	contentCache := newContentCache(storageClient)

	goodKey := root.Join("k1")
	want := "v1"
	content1, err := contentCache.GetContents(ctx, goodKey)
	if err != nil {
		t.Fatalf("error getting %v: %v", goodKey, err)
	}
	if got := string(content1); got != want {
		t.Errorf("content of %v was not as expected; got %q, want %q", goodKey, got, want)
	}
	if k1Count != 1 {
		t.Errorf("count of requests to %v was %d, want 1", goodKey, k1Count)
	}

	content2, err := contentCache.GetContents(ctx, goodKey)
	if err != nil {
		t.Fatalf("error getting %v: %v", goodKey, err)
	}
	if got := string(content2); got != want {
		t.Errorf("content of %v was not as expected; got %q, want %q", goodKey, got, want)
	}
	// Should have been cached
	if k1Count != 1 {
		t.Errorf("count of requests to %v was %d, want 1", goodKey, k1Count)
	}

	badKey := root.Join("does-not-exist")
	_, err = contentCache.GetContents(ctx, badKey)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("unexpected error from GetContents; got %v, want IsNotExist", err)
	}

	errorKey := root.Join("inject-error")
	_, err = contentCache.GetContents(ctx, errorKey)
	if err == nil || os.IsNotExist(err) {
		t.Fatalf("unexpected error from GetContents; got %v, want internal error", err)
	}
}
