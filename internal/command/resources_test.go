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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResourcesHelp(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "--help"})
	assert.NoError(t, err)
	assert.Equal(t, `Subcommands related to provisioned resources

Usage:
  score-compose resources [command]

Available Commands:
  get-outputs Return the resource outputs
  list        List the resource uids

Flags:
  -h, --help   help for resources

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times

Use "score-compose resources [command] --help" for more information about a command.
`, stdout)
	assert.Equal(t, "", stderr)

	stdout2, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"help", "resources"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, stdout2)
	assert.Equal(t, "", stderr)
}

func TestResourcesExample(t *testing.T) {
	td := changeToTempDir(t)
	_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--no-sample"})
	assert.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  container:
    image: thing
resources:
  vol:
    type: volume
  dns:
    type: dns
`), 0600))
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)

	t.Run("list", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "list"})
		assert.NoError(t, err)
		assert.Equal(t, `dns.default#example.dns
volume.default#example.vol
`, stdout)
	})

	t.Run("get not found", func(t *testing.T) {
		_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "foo"})
		assert.EqualError(t, err, "no such resource 'foo'")
	})

	t.Run("get dns", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "dns.default#example.dns"})
		assert.NoError(t, err)
		var out map[string]interface{}
		if assert.NoError(t, json.Unmarshal([]byte(stdout), &out)) {
			assert.NotEmpty(t, out["host"].(string))
		}
	})

	t.Run("get vol", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "volume.default#example.vol"})
		assert.NoError(t, err)
		var out map[string]interface{}
		if assert.NoError(t, json.Unmarshal([]byte(stdout), &out)) {
			assert.NotEmpty(t, out["source"].(string))
		}
	})

	t.Run("format json", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "volume.default#example.vol", "--format", "json"})
		assert.NoError(t, err)
		var out map[string]interface{}
		assert.NoError(t, json.Unmarshal([]byte(stdout), &out))
		assert.True(t, strings.HasSuffix(stdout, "\n"))
	})

	t.Run("format yaml", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "volume.default#example.vol", "--format", "yaml"})
		assert.NoError(t, err)
		var out map[string]interface{}
		assert.NoError(t, yaml.Unmarshal([]byte(stdout), &out))
		assert.True(t, strings.HasSuffix(stdout, "\n"))
	})

	t.Run("format template", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "volume.default#example.vol", "--format", `{{ . | len }}`})
		assert.NoError(t, err)
		assert.Equal(t, "2\n", stdout)
	})

	t.Run("format template with newline", func(t *testing.T) {
		stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"resources", "get-outputs", "volume.default#example.vol", "--format", "{{ . | len }}\n"})
		assert.NoError(t, err)
		assert.Equal(t, "2\n", stdout)
	})
}
