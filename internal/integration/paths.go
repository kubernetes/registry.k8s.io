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

package integration

import (
	"errors"
	"os"
	"path/filepath"
)

func ModuleRootDir() (string, error) {
	return moduleRootDir(os.Getwd)
}

func moduleRootDir(getWD func() (string, error)) (string, error) {
	// in a test, the working directory will be the test package source dir
	wd, err := getWD()
	if err != nil {
		return "", err
	}
	// walk upwards looking for module file
	currDir := wd
	for {
		// stop at first existing go.mod
		_, err := os.Stat(filepath.Join(currDir, "go.mod"))
		if err == nil {
			return currDir, nil
		}
		// if we get back the same path, we've hit the disk / volume root
		nextDir := filepath.Dir(currDir)
		if nextDir == currDir {
			return "", errors.New("walked to disk root without finding module root")
		}
		currDir = nextDir
	}
}
