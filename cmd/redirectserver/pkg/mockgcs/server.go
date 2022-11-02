package mockgcs

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

const expectedHost = "storage.googleapis.com"

// Server implements a simple mock Google Cloud Storage (GCS) server
type Server struct {
	buckets map[string]*Bucket
}

// Bucket is a (mock) bucket in a server
type Bucket struct {
	objects map[string]*Object
}

// Object is an object in a (mock) bucket
type Object struct {
	// Contents are the contents of this blob
	Contents []byte
	// Override can be set to force an error or similar
	Override func(response *http.Response)
}

// HTTPClient returns an http.Client bound to the mock server
func (s *Server) HTTPClient() *http.Client {
	return &http.Client{Transport: s}
}

// New constructs a new mock GCS server
func New() *Server {
	return &Server{
		buckets: make(map[string]*Bucket),
	}
}

// AddBucket creates a bucket with the specified name
func (s *Server) AddBucket(bucketName string) *Bucket {
	bucket := &Bucket{
		objects: make(map[string]*Object),
	}
	s.buckets[bucketName] = bucket
	return bucket
}

// AddObject creates an object in the bucket
func (b *Bucket) AddObject(objectPath string, obj Object) {
	b.objects[objectPath] = &obj
}

// RoundTrip implements the GCS protocol
func (s *Server) RoundTrip(request *http.Request) (*http.Response, error) {
	host := request.Host
	if host != expectedHost {
		return s.serve404(request)
	}

	url := request.URL

	pathTokens := strings.Split(strings.TrimPrefix(url.Path, "/"), "/")

	if len(pathTokens) >= 2 {
		bucket := pathTokens[0]
		objectKey := strings.Join(pathTokens[1:], "/")

		if bucket := s.buckets[bucket]; bucket != nil {
			if object := bucket.objects[objectKey]; object != nil {
				return s.serveObject(request, object)
			}
			return s.serve404(request)
		}
	}

	klog.Warningf("unhandled request: %s %s %#v", request.Method, request.URL, request)
	return s.serve404(request)
}

func (s *Server) serve404(_ *http.Request) (*http.Response, error) {
	httpResponse := &http.Response{
		Status:     http.StatusText(http.StatusNotFound),
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}
	return httpResponse, nil
}

func (s *Server) serveObject(_ *http.Request, object *Object) (*http.Response, error) {
	httpResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(object.Contents)),
		Status:     http.StatusText(http.StatusOK),
	}

	if object.Override != nil {
		object.Override(httpResponse)
	}

	return httpResponse, nil
}
