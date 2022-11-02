package mocks3

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

// Server implements a simple mock s3 server
type Server struct {
	bucketsByHost map[string]*Bucket
}

// Bucket is a mock S3 bucket
type Bucket struct {
	objects map[string]*Object
}

// Object mocks an object in an S3 bucket
type Object struct {
	Contents []byte
}

// HTTPClient returns an http.Client bound to the mocks3 "server"
func (s *Server) HTTPClient() *http.Client {
	return &http.Client{Transport: s}
}

// New constructs a new mocks3 Server
func New() *Server {
	return &Server{
		bucketsByHost: make(map[string]*Bucket),
	}
}

// AddBucket creates a new bucket, for the given S3 hostname
func (s *Server) AddBucket(bucketHost string) *Bucket {
	bucket := &Bucket{
		objects: make(map[string]*Object),
	}
	s.bucketsByHost[bucketHost] = bucket
	return bucket
}

// AddObject adds an object to an existing bucket.
func (b *Bucket) AddObject(objectPath string, obj Object) {
	b.objects[objectPath] = &obj
}

// RoundTrip implements the (mock) S3 protocol
func (s *Server) RoundTrip(request *http.Request) (*http.Response, error) {
	host := request.Host

	bucket := s.bucketsByHost[host]
	if bucket != nil {
		url := request.URL

		objectKey := strings.TrimPrefix(url.Path, "/")

		if object := bucket.objects[objectKey]; object != nil {
			return s.serveObject(request, object)
		}
		return s.serve404(request)
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
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(object.Contents)),
	}
	return httpResponse, nil
}
