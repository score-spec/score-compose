// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compose

import (
	"bufio"
	"bytes"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	assert "github.com/stretchr/testify/assert"
)

func TestYamlEncode(t *testing.T) {
	var tests = []struct {
		Name   string
		Source *compose.Project
		Output []byte
		Error  error
	}{
		{
			Name: "Should encode the docker-compose spec",
			Source: &compose.Project{
				Services: compose.Services{
					"test": {
						Name:  "test",
						Image: "busybox",
						Command: compose.ShellCommand{
							"/bin/sh",
							"-c",
							"while true; echo ...sleeping 10 sec...; sleep 10; done",
						},
					},
				},
			},
			Output: []byte(`services:
  test:
    command:
      - /bin/sh
      - -c
      - while true; echo ...sleeping 10 sec...; sleep 10; done
    image: busybox
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			buf := bytes.Buffer{}
			w := bufio.NewWriter(&buf)

			err := WriteYAML(w, tt.Source)
			w.Flush()

			if tt.Error != nil {
				// On Error
				//
				assert.ErrorContains(t, err, tt.Error.Error())
			} else {
				// On Success
				//
				assert.NoError(t, err)
				assert.Equal(t, tt.Output, buf.Bytes())
			}
		})
	}
}
