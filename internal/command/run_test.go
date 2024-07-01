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

package command

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// executeAndResetCommand is a test helper that runs and then resets a command for executing in another test.
func executeAndResetCommand(ctx context.Context, cmd *cobra.Command, args []string) (string, string, error) {
	beforeOut, beforeErr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	defer func() {
		cmd.SetOut(beforeOut)
		cmd.SetErr(beforeErr)
		// also have to remove completion commands which get auto added and bound to an output buffer
		for _, command := range cmd.Commands() {
			if command.Name() == "completion" {
				cmd.RemoveCommand(command)
				break
			}
		}
	}()

	nowOut, nowErr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.SetOut(nowOut)
	cmd.SetErr(nowErr)
	cmd.SetArgs(args)
	subCmd, err := cmd.ExecuteContextC(ctx)
	if subCmd != nil {
		subCmd.SetOut(nil)
		subCmd.SetErr(nil)
		subCmd.SetContext(nil)
		subCmd.SilenceUsage = false
		subCmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Value.Type() == "stringArray" {
				_ = f.Value.(pflag.SliceValue).Replace(nil)
			} else {
				_ = f.Value.Set(f.DefValue)
			}
		})
	}
	return nowOut.String(), nowErr.String(), err
}

func TestRunHelp(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--help"})
	assert.NoError(t, err)
	assert.Equal(t, `(Deprecated) Translate the SCORE file to docker-compose configuration

Usage:
  score-compose run [--file=score.yaml] [--output=compose.yaml] [flags]

Flags:
      --build string           Replaces 'image' name with compose 'build' instruction
      --env-file string        Location to store sample .env file
  -f, --file string            Source SCORE file (default "./score.yaml")
  -h, --help                   help for run
  -o, --output string          Output file
      --overrides string       Overrides SCORE file (default "./overrides.score.yaml")
  -p, --property stringArray   Overrides selected property value
      --skip-validation        DEPRECATED: Disables Score file schema validation

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
`, stdout)
	assert.Equal(t, "", stderr)

	stdout2, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"help", "run"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, stdout2)
	assert.Equal(t, "", stderr)
}

func TestRunExample(t *testing.T) {
	td := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-workload-name123
  extra-key: extra-value
service:
  ports:
    port-one:
      port: 1000
      protocol: TCP
      targetPort: 10000
    port-two2:
      port: 8000
containers:
  container-one1:
    image: localhost:4000/repo/my-image:tag
    command: ["/bin/sh", "-c"]
    args: ["hello", "world"]
    resources:
      requests:
        cpu: 1000m
        memory: 10Gi
      limits:
        cpu: "0.24"
        memory: 128M
    variables:
      SOME_VAR: some content here
    volumes:
    - source: volume-name
      target: /mnt/something
      readOnly: false
    - source: volume-two
      target: /mnt/something-else
    livenessProbe:
      httpGet:
        port: 8080
        path: /livez
    readinessProbe:
      httpGet:
        host: 127.0.0.1
        port: 80
        scheme: HTTP
        path: /readyz
        httpHeaders:
        - name: SOME_HEADER
          value: some-value-here
  container-two2:
    image: localhost:4000/repo/my-image:tag
resources:
  resource-one1:
    metadata:
      annotations:
        Default-Annotation: this is my annotation
        prefix.com/Another-Key_Annotation.2: something else
      extra-key: extra-value
    type: Resource-One
    class: default
    params:
      extra:
        data: here
  resource-two2:
    type: Resource-Two
  volume-name:
    type: volume
  volume-two:
    type: volume
`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.NoError(t, err)
	assert.NotEqual(t, "", stdout)
	for _, l := range []string{
		"WARN: resources.resource-one1: 'Resource-One.default' is not directly supported in score-compose, references will be converted to environment variables\n",
		"WARN: resources.resource-two2: 'Resource-Two.default' is not directly supported in score-compose, references will be converted to environment variables\n",
		"WARN: containers.container-one1.resources.requests: not supported - ignoring\n",
		"WARN: containers.container-one1.resources.limits: not supported - ignoring\n",
		"WARN: containers.container-one1.readinessProbe: not supported - ignoring\n",
		"WARN: containers.container-one1.livenessProbe: not supported - ignoring\n",
	} {
		assert.Contains(t, stderr, l)
	}

	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	var actualComposeContent map[string]interface{}
	assert.NoError(t, yaml.Unmarshal(rawComposeContent, &actualComposeContent))
	assert.Equal(t, map[string]interface{}{
		"services": map[string]interface{}{
			"example-workload-name123-container-one1": map[string]interface{}{
				"annotations": map[string]interface{}{
					"compose.score.dev/workload-name": "example-workload-name123",
				},
				"hostname":   "example-workload-name123",
				"image":      "localhost:4000/repo/my-image:tag",
				"entrypoint": []interface{}{"/bin/sh", "-c"},
				"command":    []interface{}{"hello", "world"},
				"environment": map[string]interface{}{
					"SOME_VAR": "some content here",
				},
				"ports": []interface{}{
					map[string]interface{}{"target": 10000, "published": "1000", "protocol": "tcp"},
					map[string]interface{}{"target": 8000, "published": "8000"},
				},
				"volumes": []interface{}{
					map[string]interface{}{"type": "volume", "source": "volume-name", "target": "/mnt/something"},
					map[string]interface{}{"type": "volume", "source": "volume-two", "target": "/mnt/something-else"},
				},
			},
			"example-workload-name123-container-two2": map[string]interface{}{
				"annotations": map[string]interface{}{
					"compose.score.dev/workload-name": "example-workload-name123",
				},
				"image":        "localhost:4000/repo/my-image:tag",
				"network_mode": "service:example-workload-name123-container-one1",
			},
		},
	}, actualComposeContent)

	t.Run("validate", func(t *testing.T) {
		if os.Getenv("NO_DOCKER") != "" {
			t.Skip("NO_DOCKER is set")
			return
		}

		dockerCmd, err := exec.LookPath("docker")
		require.NoError(t, err)

		assert.NoError(t, os.WriteFile(filepath.Join(td, "volume.yaml"), []byte(`
volumes:
  volume-name:
    driver: local
  volume-two:
    driver: local

`), 0644))

		cmd := exec.Command(dockerCmd, "compose", "-f", "compose.yaml", "-f", "volume.yaml", "convert", "--quiet", "--dry-run")
		cmd.Dir = td
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		assert.NoError(t, cmd.Run())
	})
}

func TestExample_invalid_spec(t *testing.T) {
	td := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
{}`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.EqualError(t, err, "validating workload spec: jsonschema: '' does not validate with https://score.dev/schemas/score#/required: missing properties: 'apiVersion', 'metadata', 'containers'")
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
}

func TestVolumeSubPathNotSupported(t *testing.T) {
	td := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-workload-name123
containers:
  container-one1:
    image: localhost:4000/repo/my-image:tag
    volumes:
    - source: volume-name
      target: /mnt/something
      path: /sub/path
`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.EqualError(t, err, "building docker-compose configuration: containers.container-one1.volumes[0].path: can't mount named volume with sub path '/sub/path': not supported")
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
}

func TestFilesNotSupported(t *testing.T) {
	td := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-workload-name123
containers:
  container-one1:
    image: localhost:4000/repo/my-image:tag
    files:
    - target: /mnt/something
      content: bananas
`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.EqualError(t, err, "building docker-compose configuration: files are not supported")
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
}

func TestInvalidWorkloadName(t *testing.T) {
	td := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: Invalid Name
containers:
  container-one1:
    image: localhost:4000/repo/my-image:tag
`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml")})
	assert.EqualError(t, err, "validating workload spec: jsonschema: '/metadata/name' does not validate with https://score.dev/schemas/score#/properties/metadata/properties/name/pattern: does not match pattern '^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$'")
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
}

func TestRunExample01(t *testing.T) {
	td := t.TempDir()
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", "../../examples/01-hello/score.yaml", "--output", filepath.Join(td, "compose.yaml")})
	assert.NoError(t, err)

	expectedOutput := `services:
  hello-world-hello:
    annotations:
      compose.score.dev/workload-name: hello-world
      your.custom/annotation: value
    command:
      - -c
      - while true; do echo Hello World!; sleep 5; done
    entrypoint:
      - /bin/sh
    hostname: hello-world
    image: busybox
`

	assert.Equal(t, expectedOutput, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(rawComposeContent))

	t.Run("validate", func(t *testing.T) {
		if os.Getenv("NO_DOCKER") != "" {
			t.Skip("NO_DOCKER is set")
			return
		}

		dockerCmd, err := exec.LookPath("docker")
		require.NoError(t, err)

		cmd := exec.Command(dockerCmd, "compose", "-f", "compose.yaml", "convert", "--quiet", "--dry-run")
		cmd.Dir = td
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		assert.NoError(t, cmd.Run())
	})
}

func TestRunWithBuild(t *testing.T) {
	td := t.TempDir()

	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: hello-world
containers:
  hello:
    image: busybox
`), 0600))

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml"), "--build", "./test",
	})
	assert.NoError(t, err)

	expectedOutput := `services:
  hello-world-hello:
    annotations:
      compose.score.dev/workload-name: hello-world
    build:
      context: ./test
    hostname: hello-world
`

	assert.Equal(t, expectedOutput, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(rawComposeContent))
}

func TestRunWithOverrides(t *testing.T) {
	td := t.TempDir()

	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: hello-world
containers:
  hello:
    image: busybox
`), 0600))

	assert.NoError(t, os.WriteFile(filepath.Join(td, "score-overrides.yaml"), []byte(`
containers:
  hello:
    image: nginx
`), 0600))

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml"), "--overrides", filepath.Join(td, "score-overrides.yaml"),
	})
	assert.NoError(t, err)

	expectedOutput := `services:
  hello-world-hello:
    annotations:
      compose.score.dev/workload-name: hello-world
    hostname: hello-world
    image: nginx
`

	assert.Equal(t, expectedOutput, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(rawComposeContent))
}

func TestRunWithPropertyOverrides(t *testing.T) {
	td := t.TempDir()

	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: hello-world
containers:
  hello:
    image: busybox
`), 0600))

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{
		"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml"), "--property", "containers.hello.image=bananas:latest",
	})
	assert.NoError(t, err)

	expectedOutput := `services:
  hello-world-hello:
    annotations:
      compose.score.dev/workload-name: hello-world
    hostname: hello-world
    image: bananas:latest
`

	assert.Equal(t, expectedOutput, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(rawComposeContent))
}
