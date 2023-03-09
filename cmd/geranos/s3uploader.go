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
	"context"
	"encoding/base64"
	"encoding/hex"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"k8s.io/klog/v2"
)

// see cmd/archeio, this matches the layout of GCR's GCS bucket
// containers/images/sha256:$layer_digest
const blobKeyPrefix = "containers/images/"

type s3Uploader struct {
	svc            *s3.S3
	uploader       *s3manager.Uploader
	reuploadLayers bool
	dryRun         bool
}

func newS3Uploader(dryRun bool) (*s3Uploader, error) {
	cfg := []*aws.Config{}
	// force anonymous configs for dry run uploaders
	if dryRun {
		cfg = append(cfg, &aws.Config{
			Credentials: credentials.AnonymousCredentials,
		})
	}
	sess, err := session.NewSession(cfg...)
	if err != nil {
		return nil, err
	}
	r := &s3Uploader{
		dryRun: dryRun,
		svc:    s3.New(sess),
	}
	r.uploader = s3manager.NewUploaderWithClient(r.svc)
	return r, nil
}

func (s *s3Uploader) CopyToS3(bucket string, layer v1.Layer) error {
	digest, err := layer.Digest()
	if err != nil {
		return err
	}
	key := keyForLayer(digest)
	if !s.reuploadLayers {
		exists, err := s.blobExists(bucket, key)
		if err != nil {
			klog.Errorf("failed to check if blob exists: %v", err)
		} else if exists {
			klog.V(4).Infof("Layer already exists: %s", key)
			return nil
		}
	}
	_ = gcpRateLimiter.Wait(context.Background())
	r, err := layer.Compressed()
	if err != nil {
		return err
	}
	defer r.Close()
	uploadInput := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   r,
	}
	// TODO: what if it isn't sha256?
	// We also depend on this in cmd/archeio currently
	if digest.Algorithm == "SHA256" {
		b, err := hex.DecodeString(digest.Hex)
		if err != nil {
			return err
		}
		uploadInput.ChecksumSHA256 = aws.String(base64.StdEncoding.EncodeToString(b))
	}
	// skip actually uploading if this is a dry-run, otherwise finally upload
	klog.Infof("Uploading layer: %s", key)
	if s.dryRun {
		return nil
	}
	_, err = s.uploader.Upload(uploadInput)
	return err
}

func keyForLayer(digest v1.Hash) string {
	return blobKeyPrefix + digest.String()
}

func (s *s3Uploader) blobExists(bucket, key string) (bool, error) {
	_, err := s.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
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
