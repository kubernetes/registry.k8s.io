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
	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"k8s.io/klog/v2"
)

// see cmd/archeio, this matches the layout of GCR's GCS bucket
const blobKeyPrefix = "/containers/images/"

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

func (s *s3Uploader) CopyToS3(bucket string, layer v1.Layer) error {
	key, err := keyForLayer(layer)
	if err != nil {
		return err
	}
	if s.skipExisting {
		exists, err := s.blobExists(bucket, key)
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
		Bucket: aws.String(bucket),
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
