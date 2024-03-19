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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
)

func TestGenerateHelp(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "--help"})
	assert.NoError(t, err)
	assert.Equal(t, `The generate command will convert Score files in the current Score compose project into a combined Docker compose
manifest. All resources and links between Workloads will be resolved and provisioned as required.

By default this command looks for score.yaml in the current directory, but can take explicit file names as positional
arguments.

"score-compose init" MUST be run first. An error will be thrown if the project directory is not present.

Usage:
  score-compose generate [flags]

Examples:

  # Specify Score files
  score-compose generate score.yaml *.score.yaml

  # Regenerate without adding new score files
  score-compose generate

  # Provide overrides when one score file is provided
  score-compose generate score.yaml --override-file=./overrides.score.yaml --override-property=metadata.key=value

Flags:
      --build stringArray               An optional build context to use for the given container --build=container=./dir or --build=container={'"context":"./dir"}
  -h, --help                            help for generate
      --image string                    An optional container image to use for any container with image == '.'
  -o, --output string                   The output file to write the composed compose file to (default "compose.yaml")
      --override-property stringArray   An optional set of path=key overrides to set or remove
      --overrides-file string           An optional file of Score overrides to merge in

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
`, stdout)
	assert.Equal(t, "", stderr)

	stdout2, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"help", "generate"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, stdout2)
	assert.Equal(t, "", stderr)
}

func changeToDir(t *testing.T, dir string) string {
	t.Helper()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})
	return dir
}

func changeToTempDir(t *testing.T) string {
	return changeToDir(t, t.TempDir())
}

func TestGenerateWithoutInit(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"generate"})
	assert.EqualError(t, err, "state directory does not exist, please run \"score-compose init\" first")
	assert.Equal(t, "", stdout)
}

func TestGenerateWithoutScoreFiles(t *testing.T) {
	_ = changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate"})
	assert.EqualError(t, err, "the project is empty, please provide a score file to generate from")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerateWithBadFile(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile(filepath.Join(td, "thing"), []byte(`"blah"`), 0644))

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "thing"})
	assert.EqualError(t, err, "failed to decode 'thing' as yaml: yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `blah` into map[string]interface {}")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerateWithBadScore(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile(filepath.Join(td, "thing"), []byte(`{}`), 0644))

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "thing"})
	assert.EqualError(t, err, "validation errors in workload '': jsonschema: '' does not validate with https://score.dev/schemas/score#/required: missing properties: 'apiVersion', 'metadata', 'containers'")
	assert.Equal(t, "", stdout)
}

func TestInitAndGenerate_with_sample(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// write overrides file
	assert.NoError(t, os.WriteFile(filepath.Join(td, "overrides.yaml"), []byte(`{"resources": {"foo": {"type": "environment"}}}`), 0644))
	// generate
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "-o", "compose-output.yaml",
		"--overrides-file", "overrides.yaml",
		"--override-property", "containers.hello-world.variables.THING=${resources.foo.THING}",
		"--", "score.yaml",
	})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
	assert.NoError(t, err)
	expectedOutput := `name: "001"
services:
    example-hello-world:
        environment:
            EXAMPLE_VARIABLE: example-value
            THING: ${THING}
        image: nginx:latest
`
	assert.Equal(t, expectedOutput, string(raw))
	// generate again just for luck
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "-o", "compose-output.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err = os.ReadFile(filepath.Join(td, "compose-output.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, string(raw))

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 1)
}

func TestInitAndGenerate_with_image_override(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// write new score file
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: .
`), 0644))

	t.Run("generate but fail due to missing override", func(t *testing.T) {
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--", "score.yaml",
		})
		assert.EqualError(t, err, "failed to convert 'example' because container 'example' has no image and neither --image nor --build was provided")
	})

	t.Run("generate with image", func(t *testing.T) {
		// generate with image
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--image", "busybox:latest", "--", "score.yaml",
		})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		expectedOutput := `name: "001"
services:
    example-example:
        image: busybox:latest
`
		assert.Equal(t, expectedOutput, string(raw))
		// generate again just for luck
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "-o", "compose-output.yaml"})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err = os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, string(raw))
	})

	t.Run("generate with raw build context", func(t *testing.T) {
		// generate with build context
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--build", "example=./dir", "--", "score.yaml",
		})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		expectedOutput := `name: "001"
services:
    example-example:
        build:
            context: ./dir
`
		assert.Equal(t, expectedOutput, string(raw))
		// generate again just for luck
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "-o", "compose-output.yaml"})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err = os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, string(raw))
	})

	t.Run("generate with json build context", func(t *testing.T) {
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--build", `example={"context":"./dir","args":{"DEBUG":"true"}}`, "--", "score.yaml",
		})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		expectedOutput := `name: "001"
services:
    example-example:
        build:
            context: ./dir
            args:
                DEBUG: "true"
`
		assert.Equal(t, expectedOutput, string(raw))
	})

}

func TestInitAndGenerate_with_files(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: foo
    files:
    - target: /blah.txt
      source: ./original.txt
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "original.txt"), []byte(`first ${metadata.name} second`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, `name: "001"
services:
    example-example:
        image: foo
        volumes:
            - type: bind
              source: .score-compose/mounts/files/example-files-0-blah.txt
              target: /blah.txt
`, string(raw))
}

func TestGenerateRedisResource(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: foo
    variables:
      CONN_STR_1: "redis://${resources.cache1.username}:${resources.cache1.password}@${resources.cache1.host}:${resources.cache1.port}"
      CONN_STR_2: "redis://${resources.cache2.username}:${resources.cache2.password}@${resources.cache2.host}:${resources.cache2.port}"
resources:
  cache1:
    type: redis
  cache2:
    type: redis
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 2)
	assert.Contains(t, sd.State.Resources["redis.default#example.cache1"].State, "serviceName")
	assert.Contains(t, sd.State.Resources["redis.default#example.cache1"].State, "password")
	assert.Contains(t, sd.State.Resources["redis.default#example.cache2"].State, "serviceName")
	assert.NotEqual(t, sd.State.Resources["redis.default#example.cache1"].State, sd.State.Resources["redis.default#example.cache2"].State)
	assert.Len(t, sd.State.SharedState, 0)

	t.Run("validate compose spec", func(t *testing.T) {
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

func TestGeneratePostgresResource(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: foo
    variables:
      CONN_STR_1: "postgres://${resources.db1.username}:${resources.db1.password}@${resources.db1.host}:${resources.db1.port}/${resources.db1.name}"
      CONN_STR_2: "postgres://${resources.db2.username}:${resources.db2.password}@${resources.db2.host}:${resources.db2.port}/${resources.db2.name}"
resources:
  db1:
    type: postgres
  db2:
    type: postgres
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 2)
	assert.Contains(t, sd.State.Resources["postgres.default#example.db1"].State, "database")
	assert.Contains(t, sd.State.Resources["postgres.default#example.db1"].State, "username")
	assert.Contains(t, sd.State.Resources["postgres.default#example.db1"].State, "password")
	assert.Contains(t, sd.State.Resources["postgres.default#example.db2"].State, "database")
	assert.NotEqual(t, sd.State.Resources["postgres.default#example.db1"].State, sd.State.Resources["postgres.default#example.db2"].State)
	assert.Contains(t, sd.State.SharedState, "default-provisioners-postgres-instance")

	t.Run("validate compose spec", func(t *testing.T) {
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

func TestGenerateS3Resource(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: foo
    variables:
      output: ${resources.bucket1.endpoint} ${resources.bucket1.region} ${resources.bucket1.bucket} ${resources.bucket1.access_key_id} ${resources.bucket1.secret_key}
resources:
  bucket1:
    metadata:
      annotations:
        compose.score.dev/publish-port: "9001"
    type: s3
  bucket2:
    type: s3
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 2)
	assert.Contains(t, sd.State.Resources["s3.default#example.bucket1"].State, "bucket")
	assert.Contains(t, sd.State.Resources["s3.default#example.bucket2"].State, "bucket")
	assert.NotEqual(t, sd.State.Resources["s3.default#example.bucket1"].State, sd.State.Resources["postgres.default#example.bucket2"].State)
	assert.Contains(t, sd.State.SharedState, "default-provisioners-minio-instance")

	t.Run("validate compose spec", func(t *testing.T) {
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

func TestInitAndGenerate_with_depends_on(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile("score.yaml", []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: foo
resources:
  thing:
    type: thing
`), 0644))

	assert.NoError(t, os.WriteFile(".score-compose/00-custom.provisioners.yaml", []byte(`
- uri: template://blah
  type: thing
  services: |
    init_service:
      image: thing
      labels:
        dev.score.compose.labels.is-init-container: "true"
    generic_service:
      image: other
    service_with_healthcheck:
      image: something
      healthcheck:
        test: ["CMD", "boo"]
`), 0644))
	// generate
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, `name: "001"
services:
    example-example:
        depends_on:
            wait-for-resources:
                condition: service_started
                required: false
        image: foo
    generic_service:
        image: other
    init_service:
        image: thing
        labels:
            dev.score.compose.labels.is-init-container: "true"
    service_with_healthcheck:
        healthcheck:
            test:
                - CMD
                - boo
        image: something
    wait-for-resources:
        command:
            - echo
        depends_on:
            generic_service:
                condition: service_started
                required: true
            init_service:
                condition: service_completed_successfully
                required: true
            service_with_healthcheck:
                condition: service_healthy
                required: true
        image: alpine
`, string(raw))

	t.Run("validate compose spec", func(t *testing.T) {
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

func TestInitAndGenerate_with_dependent_resources(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// write custom providers
	assert.NoError(t, os.WriteFile(filepath.Join(td, ".score-compose", "00-custom.provisioners.yaml"), []byte(`
- uri: template://foo
  type: foo
  outputs: |
    blah: value
  services: |
    foo-service:
      image: foo-image
- uri: template://bar
  type: bar
  services: |
    bar-service:
      image: {{ .Params.x }}
`), 0644))

	// write custom score file
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: busybox
resources:
  first:
    type: foo
  second:
    type: bar
    params:
      x: ${resources.first.blah}
`), 0644))

	// generate
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, `name: "001"
services:
    bar-service:
        image: value
    example-example:
        depends_on:
            wait-for-resources:
                condition: service_started
                required: false
        image: busybox
    foo-service:
        image: foo-image
    wait-for-resources:
        command:
            - echo
        depends_on:
            bar-service:
                condition: service_started
                required: true
            foo-service:
                condition: service_started
                required: true
        image: alpine
`, string(raw))
}

func TestInitAndGenerateWithNetworkServicesAcrossWorkloads(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// write custom providers
	assert.NoError(t, os.WriteFile(filepath.Join(td, ".score-compose", "00-custom.provisioners.yaml"), []byte(`
- uri: template://default-provisioners/workload-port
  type: workload-port
  init: |
    {{ if not .Params.workload }}{{ fail "'workload' param required" }}{{ end }}
    {{ if not .Params.port }}{{ fail "'port' param required - the name of the remote port" }}{{ end }}
    {{ $x := index .WorkloadServices .Params.workload }}
    {{ if not $x.ServiceName }}{{ fail "unknown workload" }}{{ end }}
    {{ $y := index $x.Ports .Params.port }}
    {{ if not $y.Name }}{{ fail "unknown port" }}{{ end }}
  state: |
    {{ $x := index .WorkloadServices .Params.workload }}
    hostname: {{ $x.ServiceName | quote }}
    {{ $y := index $x.Ports .Params.port }}
    port: {{ $y.TargetPort }}
`),
		0644,
	))

	t.Run("fail unknown workload", func(t *testing.T) {
		assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: busybox
resources:
  first:
    type: workload-port
    params:
      workload: example-2
      port: web
`), 0644))

		// generate
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
		assert.EqualError(t, err, "failed to provision: resource 'workload-port.default#example.first': failed to provision: init template failed: failed to execute template: template: :4:30: executing \"\" at <fail \"unknown workload\">: error calling fail: unknown workload")
	})

	t.Run("fail unknown port", func(t *testing.T) {
		assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: busybox
resources:
  first:
    type: workload-port
    params:
      workload: example
      port: web
`), 0644))

		// generate
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
		assert.EqualError(t, err, "failed to provision: resource 'workload-port.default#example.first': failed to provision: init template failed: failed to execute template: template: :6:23: executing \"\" at <fail \"unknown port\">: error calling fail: unknown port")
	})

	t.Run("succeed", func(t *testing.T) {
		assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: busybox
service:
  ports:
    web:
      port: 8080
      targetPort: 80
resources:
  first:
    type: workload-port
    params:
      workload: example
      port: web
`), 0644))

		// generate
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
		assert.NoError(t, err)

		// check that state was persisted
		sd, ok, err := project.LoadStateDirectory(td)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Len(t, sd.State.Workloads, 1)
		assert.Len(t, sd.State.Resources, 1)
		assert.Equal(t, map[string]interface{}{
			"hostname": "example-example",
			"port":     80,
		}, sd.State.Resources["workload-port.default#example.first"].State)
	})

}
