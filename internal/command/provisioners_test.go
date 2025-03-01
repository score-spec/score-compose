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

	"github.com/score-spec/score-compose/internal/provisioners/loader"
	"github.com/stretchr/testify/assert"
)

var (
	update = flag.Bool("update", false, "update the golden files of this test")
)

func TestDisplayProvisioners(t *testing.T) {
	tests := []struct {
		name             string
		fixture          string
		format           string
		expectedResponse string
		expectedError    string
	}{
		{
			name:             "display provisioners in table format",
			fixture:          "provisioners.custom.golden",
			format:           "table",
			expectedResponse: "provisioners.list.valid.table.golden",
			expectedError:    "",
		},
		{
			name:             "display provisioners in json format",
			fixture:          "provisioners.custom.golden",
			format:           "json",
			expectedResponse: "provisioners.list.valid.json.golden",
			expectedError:    "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provisioners, err := loader.LoadProvisionersFromDirectory("fixtures/", test.fixture)
			if err != nil {
				t.Fatal(err)
			}

			// Capture the output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			displayProvisioners(provisioners, test.format)

			w.Close()
			os.Stdout = old
			out, _ := io.ReadAll(r)

			got := string(out)

			expected, err := goldenValue(t, "testdata", test.expectedResponse, got, *update)
			if err != nil {
				t.Fatal(err)
			}

			if test.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
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
