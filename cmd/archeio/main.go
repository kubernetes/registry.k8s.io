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
	"flag"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"k8s.io/klog/v2"

	"k8s.io/registry.k8s.io/cmd/archeio/internal/app"
)

func main() {
	// klog setup
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// make it possible to override k8s.gcr.io without rebuilding in the future
	registryConfig := app.RegistryConfig{
		UpstreamUsGAR: app.Registry{
			Endpoint:  getEnv("UPSTREAM_US_GCP_ENDPOINT", "https://gcr.io"),
			Namespace: getEnv("UPSTREAM_US_GCP_NAMESPACE", "datadoghq"),
		},
		UpstreamEuGAR: app.Registry{
			Endpoint:  getEnv("UPSTREAM_EU_GCP_ENDPOINT", "https://eu.gcr.io"),
			Namespace: getEnv("UPSTREAM_EU_GCP_NAMESPACE", "datadoghq"),
		},
		UpstreamAsiaGAR: app.Registry{
			Endpoint:  getEnv("UPSTREAM_AP_GCP_GCR_ENDPOINT", "https://asia.gcr.io"),
			Namespace: getEnv("UPSTREAM_AP_GCP_GCR_NAMESPACE", "datadoghq"),
		},
		UpstreamACR: app.Registry{
			// Azure does not use a registry path, the endpoint is already datadoghq.azurecr.io
			Endpoint: getEnv("UPSTREAM_AZ_ENDPOINT", "https://datadoghq.azurecr.io"),
		},
		UpstreamCDN: app.Registry{
			// CloudFront does not use a registry path, the endpoint is already d3o2h7i3xf2t1t.cloudfront.net
			Endpoint: getEnv("UPSTREAM_CDN_ENDPOINT", "https://d3o2h7i3xf2t1t.cloudfront.net"),
		},
		InfoURL:    "https://docs.datadoghq.com/",
		PrivacyURL: "https://www.datadoghq.com/legal/privacy/",
	}

	handler := app.MakeHandler(registryConfig)

	klog.InfoS("Starting AWS Lambda handler with API Gateway v1")
	klog.InfoS("registry", "configuration", registryConfig)

	// Lambda mode - use the API Gateway v1 adapter
	adapter := httpadapter.New(handler)
	lambda.Start(adapter.ProxyWithContext)
}

// getEnv returns defaultValue if key is not set, else the value of os.LookupEnv(key)
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
