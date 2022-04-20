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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func EnsureBinsInPath(binDir string) error {
	path := os.Getenv("PATH")
	// if bins are already at front of PATH, do nothing
	if strings.HasPrefix(path, binDir+string(os.PathSeparator)) {
		return nil
	}
	// otherwise prepend and set
	newPath := binDir + string(os.PathListSeparator) + path
	return os.Setenv("PATH", newPath)
}

// EnsureCrane ensures crane is available in PATH for testing
// under rootPath/bin
// See also: EnsureBinsInPath
func EnsureCrane(rootPath string) error {
	// ensure $REPO_ROOT/bin is in the front of $PATH
	root, err := ModuleRootDir()
	if err != nil {
		return fmt.Errorf("failed to detect path to project root: %w", err)
	}
	binDir := rootToBinDir(root)
	if err := EnsureBinsInPath(binDir); err != nil {
		return fmt.Errorf("failed to ensure PATH: %w", err)
	}
	// install crane
	// nolint:gosec // we *want* user supplied command arguments ...
	cmd := exec.Command(
		"go", "build",
		"-o", filepath.Join(binDir, "crane"),
		"github.com/google/go-containerregistry/cmd/crane",
	)
	cmd.Dir = rootToToolsDir(root)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install crane: %w", err)
	}
	return nil
}
