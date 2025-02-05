// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	update = flag.Bool("update", false, "update the golden files of this test")
)

func TestGetProvisionerFiles(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(dir string) error
		expectedFiles []string
		expectedError string
	}{
		{
			name: "custom provisioner files",
			setup: func(dir string) error {
				files := []string{
					"Z-custom1.provisioners.yaml",
					"Z-custom2.provisioners.yaml",
				}
				for _, file := range files {
					if _, err := os.Create(filepath.Join(dir, file)); err != nil {
						return err
					}
				}
				return nil
			},
			expectedFiles: []string{
				"Z-custom1.provisioners.yaml",
				"Z-custom2.provisioners.yaml",
			},
			expectedError: "",
		},
		{
			name: "default provisioner file",
			setup: func(dir string) error {
				if _, err := os.Create(filepath.Join(dir, "zz-default.provisioners.yaml")); err != nil {
					return err
				}
				return nil
			},
			expectedFiles: []string{
				"zz-default.provisioners.yaml",
			},
			expectedError: "",
		},
		{
			name: "no provisioner files",
			setup: func(dir string) error {
				return nil
			},
			expectedFiles: nil,
			expectedError: "default provisioners file not found",
		},
		{
			name: "directory read error",
			setup: func(dir string) error {
				return os.RemoveAll(dir)
			},
			expectedFiles: nil,
			expectedError: "failed to read directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "test-get-provisioner-files")
			assert.NoError(t, err)
			defer os.RemoveAll(dir)

			if tt.setup != nil {
				err = tt.setup(dir)
				assert.NoError(t, err)
			}

			files, err := getProvisionerFiles(dir, "test-project")
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				for i, file := range tt.expectedFiles {
					tt.expectedFiles[i] = filepath.Join(dir, file)
				}
				assert.ElementsMatch(t, tt.expectedFiles, files)
			}
		})
	}
}

func TestDisplayProvisioners(t *testing.T) {
	tests := []struct {
		name             string
		fixture          string
		expectedResponse string
		expectedError    string
	}{
		{
			name:             "valid provisioner files",
			fixture:          "provisioners.custom.golden",
			expectedResponse: "provisioners.list.valid.golden",
			expectedError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []string{filepath.Join("fixtures", "/", tt.fixture)}

			// Capture the output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := displayProvisioners(files)

			w.Close()
			os.Stdout = old
			out, _ := io.ReadAll(r)

			got := string(out)

			expected, err := goldenValue(t, "testdata", tt.expectedResponse, got, *update)
			if err != nil {
				t.Fatal(err)
			}

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, got)
			}
		})
	}
}

func goldenValue(t *testing.T, path string, goldenFile string, actual string, update bool) (string, error) {
	t.Helper()
	goldenPath := filepath.Join(path, goldenFile)

	f, err := os.OpenFile(goldenPath, os.O_RDWR, 0644)
	defer f.Close()

	if update {
		_, err := f.WriteString(actual)
		if err != nil {
			return "", fmt.Errorf("failed to update golden file %s: %w", goldenPath, err)
		}

		return actual, nil
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read golden file %s: %w", goldenPath, err)
	}

	return string(content), nil
}
