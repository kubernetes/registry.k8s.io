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

// ranges2go generates a go source file with pre-parsed AWS IP ranges data.
// See also genrawdata.sh for downloading the raw data to this binary.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// overridable for make verify
	outputPath := os.Getenv("OUT_FILE")
	dataDir := os.Getenv("DATA_DIR")
	excludedAWSRegions := strings.Split(os.Getenv("EXCLUDED_AWS_REGIONS"), ",")
	if len(excludedAWSRegions) > 0 {
		fmt.Printf("Please make sure the following excluded AWS regions are added after they have been released: %v", excludedAWSRegions)
	}
	if outputPath == "" {
		outputPath = "./zz_generated_range_data.go"
	}
	if dataDir == "" {
		dataDir = "./internal/ranges2go/data"
	}
	// read in data
	awsRaw := mustReadFile(filepath.Join(dataDir, "aws-ip-ranges.json"))
	gcpRaw := mustReadFile(filepath.Join(dataDir, "gcp-cloud.json"))
	// parse raw AWS IP range data
	awsRTP, err := parseAWS(awsRaw, excludedAWSRegions)
	if err != nil {
		panic(err)
	}
	// parse GCP IP range data
	gcpRTP, err := parseGCP(gcpRaw)
	if err != nil {
		panic(err)
	}
	// emit file
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	cloudToRTP := map[string]regionsToPrefixes{
		"AWS": awsRTP,
		"GCP": gcpRTP,
	}
	if err := generateRangesGo(f, cloudToRTP); err != nil {
		panic(err)
	}
}

func mustReadFile(filePath string) string {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	return string(contents)
}
