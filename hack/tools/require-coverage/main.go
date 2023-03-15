/*
Copyright 2023 The Kubernetes Authors.

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

// A small utility to enforce code coverage levels
// hack/make-rules/test.sh && (cd ./hack/tools && go run ./require-coverage)

package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/cover"

	"k8s.io/apimachinery/pkg/util/sets"
)

// TODO: instead of fully excluding files, maybe we should have a more
// flexible pattern of minimum coverage?
//
// For now the goal is to require 100% coverage for production serving code.
// See also: cmd/archeio/docs/testing.md
//
// Reviewers should be wary of approving additions to this list.
var knownFailingFiles = sets.NewString(
	// this code is used only at development time and integration testing it
	// is probably excessive
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs/internal/ranges2go/main.go",
	// TODO: this is reasonable to test and has poor coverage currently
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs/internal/ranges2go/parse_gcp.go",
	// TODO: this is reasonable to test but shy of 100% coverage, mostly error handling ...
	"k8s.io/registry.k8s.io/pkg/net/cloudcidrs/internal/ranges2go/gen.go",
	// geranos is not easily tested and is not in the blocking path in production
	// we should still test it better
	"k8s.io/registry.k8s.io/cmd/geranos/main.go",
	"k8s.io/registry.k8s.io/cmd/geranos/ratelimitroundtrip.go",
	"k8s.io/registry.k8s.io/cmd/geranos/s3uploader.go",
	"k8s.io/registry.k8s.io/cmd/geranos/schemav1.go",
	"k8s.io/registry.k8s.io/cmd/geranos/walkimages.go",
	// integration test utilites
	"k8s.io/registry.k8s.io/internal/integration/paths.go",
	"k8s.io/registry.k8s.io/internal/integration/bins.go",
	// TODO: we can cover this
	"k8s.io/registry.k8s.io/cmd/archeio/main.go",
)

func main() {
	fmt.Println("Checking coverage ...")
	profiles, err := cover.ParseProfiles("./../../bin/all.cov")
	if err != nil {
		panic(err)
	}
	failedAny := false
	needToRemove := []string{}
	for _, profile := range profiles {
		coverage := coverPercent(profile)
		file := profile.FileName
		if coverage < 100.0 {
			if !knownFailingFiles.Has(file) {
				failedAny = true
				fmt.Printf("FAILED: %s %v%%\n", file, coverage)
			} else {
				fmt.Printf("IGNORE: %s %v%%\n", file, coverage)
			}
		} else {
			if knownFailingFiles.Has(file) {
				needToRemove = append(needToRemove, file)
			}
			fmt.Printf("PASSED: %s %v%%\n", file, coverage)
		}
	}
	if failedAny {
		fmt.Println("Failed required coverage levels for one or more go files")
		os.Exit(-1)
	} else {
		fmt.Println("All code coverage either acceptable or ignored")
	}
	if len(needToRemove) > 0 {
		fmt.Println("FAILED: The following files are now passing and must be removed frmo the ignored list:")
		for _, file := range needToRemove {
			fmt.Println(file)
		}
		os.Exit(-1)
	}
}

func coverPercent(profile *cover.Profile) float64 {
	totalStatements := 0
	coveredStatements := 0
	for _, block := range profile.Blocks {
		totalStatements += block.NumStmt
		if block.Count > 0 {
			coveredStatements += block.NumStmt
		}
	}
	return float64(coveredStatements) / float64(totalStatements) * 100
}
