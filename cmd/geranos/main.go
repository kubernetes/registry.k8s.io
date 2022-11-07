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

package main

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"k8s.io/klog/v2"
)

// see cmd/archeio, this matches the layout of GCR's GCS bucket
const blobKeyPrefix = "/containers/images/"

// one of the backing registries for registry.k8s.io
// TODO: make configurable later
const sourceRegistry = "us-central1-docker.pkg.dev/k8s-artifacts-prod/images"

// TODO: make configurable later
const s3Bucket = "s3:prod-registry-k8s-io-us-east-2"

func main() {
	repo, err := name.NewRepository(sourceRegistry)
	if err != nil {
		klog.Fatal(err)
	}
	s3Uploader, err := newS3Uploader()
	if err != nil {
		klog.Fatal(err)
	}
	// walk all images in the repo
	err = walkManifests(repo, func(ref name.Reference) error {
		klog.Infof("Processing image: %s", ref)
		image, err := remote.Image(ref)
		if err != nil {
			return err
		}
		// get all image blobs as v1.Layer
		layers, err := image.Layers()
		if err != nil {
			return err
		}
		configLayer, err := partial.ConfigLayer(image)
		if err != nil {
			return err
		}
		layers = append(layers, configLayer)
		// copy all layers
		for _, layer := range layers {
			if err := s3Uploader.CopyToS3(layer); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		klog.Fatal(err)
	} else {
		klog.Info("Done!")
	}
}

type walkManifestsFunc func(ref name.Reference) error

func walkManifests(repo name.Repository, walkManifest walkManifestsFunc) error {
	// TODO: detect if GCR/AR and if not implement something close enough using crane.Catalog + crane.ListTags\
	// google.Walk can walk manifests even for imsges without tags, which gets us _all_ reachable image content
	// but it depends on GCP specific APIs
	// see: https://github.com/opencontainers/distribution-spec/issues/222
	return google.Walk(repo, func(repo name.Repository, tags *google.Tags, err error) error {
		for digest := range tags.Manifests {
			ref, err := name.ParseReference(fmt.Sprintf("%s@%s", repo, digest))
			if err != nil {
				return err
			}
			if err := walkManifest(ref); err != nil {
				return err
			}
		}
		return nil
	})
}

type s3Uploader struct {
	svc          *s3.S3
	uploader     *s3manager.Uploader
	skipExisting bool
}

func newS3Uploader() (*s3Uploader, error) {
	sess := session.Must(session.NewSession())
	r := &s3Uploader{}
	r.svc = s3.New(sess)
	r.uploader = s3manager.NewUploaderWithClient(r.svc)
	return r, nil
}

func (s *s3Uploader) CopyToS3(layer v1.Layer) error {
	key, err := keyForLayer(layer)
	if err != nil {
		return err
	}
	if s.skipExisting {
		exists, err := s.blobExists(key)
		if err != nil {
			klog.Errorf("failed to check if blob exists: %v", err)
		} else if exists {
			return nil
		}
	}
	r, err := layer.Compressed()
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	return err
}

func keyForLayer(layer v1.Layer) (string, error) {
	digest, err := layer.Digest()
	if err != nil {
		return "", err
	}
	return blobKeyPrefix + digest.String(), nil
}

func (s *s3Uploader) blobExists(key string) (bool, error) {
	_, err := s.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// yes, we really have to typecast to compare against an undocument string
		// to check if the object doesn't exist vs an error making the call
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
