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

  # Publish a port exposed by a workload for local testing
  score-compose generate score.yaml --publish 8080:my-workload:80

  # Publish a port from a resource host and port for local testing, the middle expression is RESOURCE_ID.OUTPUT_KEY
  score-compose generate score.yaml --publish 5432:postgres#my-workload.db.host:5432

Flags:
      --build stringArray               An optional build context to use for the given container --build=container=./dir or --build=container={"context":"./dir"}
      --env-file string                 Location to store a skeleton .env file for compose - this will override existing content
  -h, --help                            help for generate
      --image string                    An optional container image to use for any container with image == '.'
  -o, --output string                   The output file to write the composed compose file to (default "compose.yaml")
      --override-property stringArray   An optional set of path=key overrides to set or remove
      --overrides-file string           An optional file of Score overrides to merge in
      --publish stringArray             An optional set of HOST_PORT:<ref>:CONTAINER_PORT to publish on the host system.

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
        annotations:
            compose.score.dev/workload-name: example
        environment:
            EXAMPLE_VARIABLE: example-value
            THING: ${THING}
        hostname: example
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
	assert.Equal(t, "score.yaml", *sd.State.Workloads["example"].File)
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
        annotations:
            compose.score.dev/workload-name: example
        hostname: example
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
        annotations:
            compose.score.dev/workload-name: example
        build:
            context: ./dir
        hostname: example
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
        annotations:
            compose.score.dev/workload-name: example
        build:
            context: ./dir
            args:
                DEBUG: "true"
        hostname: example
`
		assert.Equal(t, expectedOutput, string(raw))
	})

	t.Run("generate with json build context and array args", func(t *testing.T) {
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--build", `example={"context":"./dir","args":["DEBUG"]}`, "--", "score.yaml",
		})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		expectedOutput := `name: "001"
services:
    example-example:
        annotations:
            compose.score.dev/workload-name: example
        build:
            context: ./dir
            args:
                DEBUG: null
        hostname: example
`
		assert.Equal(t, expectedOutput, string(raw))
	})

	t.Run("generate with yaml build context", func(t *testing.T) {
		stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
			"generate", "-o", "compose-output.yaml", "--build", `example={context: "./dir"}`, "--", "score.yaml",
		})
		assert.NoError(t, err)
		assert.Equal(t, "", stdout)
		raw, err := os.ReadFile(filepath.Join(td, "compose-output.yaml"))
		assert.NoError(t, err)
		expectedOutput := `name: "001"
services:
    example-example:
        annotations:
            compose.score.dev/workload-name: example
        build:
            context: ./dir
        hostname: example
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
        annotations:
            compose.score.dev/workload-name: example
        hostname: example
        image: foo
        volumes:
            - type: bind
              source: .score-compose/mounts/files/example-files-blah.txt
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
        annotations:
            compose.score.dev/workload-name: example
        depends_on:
            wait-for-resources:
                condition: service_completed_successfully
                required: true
        hostname: example
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
        annotations:
            compose.score.dev/workload-name: example
        depends_on:
            wait-for-resources:
                condition: service_completed_successfully
                required: true
        hostname: example
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
			"hostname": "example",
			"port":     80,
		}, sd.State.Resources["workload-port.default#example.first"].State)
	})

}

func TestInitAndGenerate_with_annotation_ref(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	assert.NoError(t, os.WriteFile("score.yaml", []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
  annotations:
    key.com/foo-bar: thing
containers:
  example:
    image: foo
    variables:
      REF: ${metadata.annotations.key\.com/foo-bar}
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
        annotations:
            compose.score.dev/workload-name: example
            key.com/foo-bar: thing
        environment:
            REF: thing
        hostname: example
        image: foo
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

func TestGenerateRouteResource(t *testing.T) {
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
service:
  ports:
    foo:
      port: 80
      targetPort: 8080
resources:
  r1:
    type: route
    params:
      host: localhost1
      path: /first
      port: foo
  r2:
    type: route
    params:
      host: localhost1
      path: /second
      port: foo
  r3:
    type: route
    metadata:
      annotations:
        compose.score.dev/route-provisioner-path-type: Exact
    params:
      host: localhost2
      path: /third
      port: 80
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 3)
	x := sd.State.SharedState["default-provisioners-routing-instance"].(map[string]interface{})
	instanceServiceName := x["instanceServiceName"].(string)
	assert.Contains(t, instanceServiceName, "routing-")
	delete(x, "instanceServiceName")
	assert.Equal(t, map[string]interface{}{
		"default-provisioners-routing-instance": map[string]interface{}{
			"hosts": map[string]interface{}{
				"localhost1": map[string]interface{}{
					"route.default#example.r1": map[string]interface{}{"path": "/first", "port": 8080, "target": "example:8080", "path_type": "Prefix"},
					"route.default#example.r2": map[string]interface{}{"path": "/second", "port": 8080, "target": "example:8080", "path_type": "Prefix"},
				},
				"localhost2": map[string]interface{}{
					"route.default#example.r3": map[string]interface{}{"path": "/third", "port": 8080, "target": "example:8080", "path_type": "Exact"},
				},
			},
			"instancePort": 8080,
		},
	}, sd.State.SharedState)

	// validate that the wildcard routes don't exist for /third
	raw, err := os.ReadFile(filepath.Join(td, ".score-compose", "mounts", instanceServiceName, "nginx.conf"))
	assert.NoError(t, err)
	assert.Contains(t, string(raw), `location ~ ^/first$`)
	assert.Contains(t, string(raw), `location ~ ^/first/.*`)
	assert.Contains(t, string(raw), `location ~ ^/second$`)
	assert.Contains(t, string(raw), `location ~ ^/second/.*`)
	assert.Contains(t, string(raw), `location ~ ^/third$`)
	assert.NotContains(t, string(raw), `location ~ ^/third/.*`)

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

func TestEnvVarsArentRequiredInVariables(t *testing.T) {
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
      ONE: ${resources.env.UNKNOWN_SCORE_VARIABLE}
resources:
  env:
    type: environment
`), 0644))
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, `name: "001"
services:
    example-example:
        annotations:
            compose.score.dev/workload-name: example
        environment:
            ONE: ${UNKNOWN_SCORE_VARIABLE}
        hostname: example
        image: foo
`, string(raw))
}

func TestEnvVarsMustResolveInsideFiles(t *testing.T) {
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
    - target: /some/file
      content: ${resources.env.UNKNOWN_SCORE_VARIABLE}
resources:
  env:
    type: environment
`), 0644))
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.EqualError(t, err, "failed to convert workload 'example' to Docker compose: containers.example.files[/some/file]: "+
		"failed to substitute in content: invalid ref 'resources.env.UNKNOWN_SCORE_VARIABLE': "+
		"environment variable 'UNKNOWN_SCORE_VARIABLE' must be resolved",
	)
}

func TestEnvVarsMustResolveInsideParams(t *testing.T) {
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
resources:
  env:
    type: environment
  data:
    type: volume
    params:
      x: ${resources.env.UNKNOWN_SCORE_VARIABLE}
`), 0644))
	_, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.EqualError(t, err, "failed to provision: failed to substitute params for resource 'volume.default#example.data': "+
		"x: invalid ref 'resources.env.UNKNOWN_SCORE_VARIABLE': "+
		"environment variable 'UNKNOWN_SCORE_VARIABLE' must be resolved",
	)
}

func TestInitAndGenerate_with_volume_types(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// write custom providers
	assert.NoError(t, os.WriteFile(filepath.Join(td, ".score-compose", "00-custom.provisioners.yaml"), []byte(`
- uri: template://docker-volume
  type: volume
  outputs: |
    type: volume
    source: named-volume
- uri: template://tmpfs-volume
  type: tmp-volume
  outputs: |
    type: tmpfs
    tmpfs:
      size: 10000000
- uri: template://bind-volume
  type: bind-volume
  outputs: |
    type: bind
    source: /dev/something
    bind:
      create_host_path: true
`), 0644))

	// write custom score file
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  example:
    image: busybox
    volumes:
    - target: /mnt/v1
      source: ${resources.v1}
    - target: /mnt/v2
      source: ${resources.v2}
      path: thing
    - target: /mnt/v3
      source: ${resources.v3}
      path: other/thing
resources:
  v1:
    type: tmp-volume
  v2:
    type: bind-volume
  v3:
    type: volume
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
        annotations:
            compose.score.dev/workload-name: example
        hostname: example
        image: busybox
        volumes:
            - type: bind
              source: /dev/something/thing
              target: /mnt/v2
              bind:
                create_host_path: true
            - type: volume
              source: named-volume
              target: /mnt/v3
              volume:
                subpath: other/thing
            - type: tmpfs
              source: tmp-volume.default#example.v1
              target: /mnt/v1
              tmpfs:
                size: "10000000"
`, string(raw))
}

func TestGenerateMongodbResource(t *testing.T) {
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
      CONN_STR_1: "mongodb://${resources.db.username}:${resources.db.password}@${resources.db.host}:${resources.db.port}/"
      CONN_STR_2: "${resources.db.connection}"
resources:
  db:
    type: mongodb
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 1)
	assert.Contains(t, sd.State.Resources["mongodb.default#example.db"].Outputs, "connection")
	assert.Contains(t, sd.State.Resources["mongodb.default#example.db"].Outputs, "username")
	assert.Contains(t, sd.State.Resources["mongodb.default#example.db"].Outputs, "password")
	assert.Contains(t, sd.State.Resources["mongodb.default#example.db"].Outputs, "host")
	assert.Contains(t, sd.State.Resources["mongodb.default#example.db"].Outputs, "port")

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

func TestGenerateMySQLResource(t *testing.T) {
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
      CONN_STR_1: "mysql://${resources.db1.username}:${resources.db1.password}@${resources.db1.host}:${resources.db1.port}/${resources.db1.name}"
      CONN_STR_2: "mysql://${resources.db2.username}:${resources.db2.password}@${resources.db2.host}:${resources.db2.port}/${resources.db2.name}"
resources:
  db1:
    type: mysql
  db2:
    type: mysql
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
	assert.Contains(t, sd.State.Resources["mysql.default#example.db1"].State, "database")
	assert.Contains(t, sd.State.Resources["mysql.default#example.db1"].State, "username")
	assert.Contains(t, sd.State.Resources["mysql.default#example.db1"].State, "password")
	assert.Contains(t, sd.State.Resources["mysql.default#example.db2"].State, "database")
	assert.NotEqual(t, sd.State.Resources["mysql.default#example.db1"].State, sd.State.Resources["mysql.default#example.db2"].State)
	assert.Contains(t, sd.State.SharedState, "default-provisioners-mysql-instance")

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

func TestGenerateKeepAnnotations(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
  annotations:
    example.com/fizz: buzz
containers:
  example:
    image: foo
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Contains(t, string(raw), `example.com/fizz: buzz`)

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	raw, err = os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Contains(t, string(raw), `example.com/fizz: buzz`)
}

func TestGenerateElasticsearchResource(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  hello:
    image: foo
resources:
  ecs:
    type: elasticsearch
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

	// check that state was persisted
	sd, ok, err := project.LoadStateDirectory(td)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, sd.State.Workloads, 1)
	assert.Len(t, sd.State.Resources, 1)
	assert.Contains(t, sd.State.Resources["elasticsearch.default#example.ecs"].Outputs, "host")
	assert.Contains(t, sd.State.Resources["elasticsearch.default#example.ecs"].Outputs, "port")
	assert.Contains(t, sd.State.Resources["elasticsearch.default#example.ecs"].Outputs, "username")
	assert.Contains(t, sd.State.Resources["elasticsearch.default#example.ecs"].Outputs, "password")

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

func TestGeneratePublishPorts(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
containers:
  hello:
    image: foo
resources:
  db1:
    type: postgres
  db2:
    type: postgres
    id: thing
`), 0644))

	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "score.yaml",
		"--publish", "8080:example:80",
		"--publish", "42:postgres#example.db1.host:13",
		"--publish", "43:postgres.default#example.db1.host:14",
		"--publish", "44:postgres.default#thing.host:15",
	})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)

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

	for _, tc := range [][2]string{
		{"something", "--publish[0] expected 3 :-separated parts"},
		{"x:foo:y", "--publish[0] could not parse host port 'x' as integer"},
		{"8080:foo:y", "--publish[0] could not parse container port 'y' as integer"},
		{"42:thing:13", "--publish[0] failed to find a workload named 'thing'"},
		{"42:x#y:13", "--publish[0] must match RES_UID.OUTPUT"},
		{"42:x#y.z:13", "--publish[0] failed to find a resource with uid 'x#y'"},
		{"42:postgres#thing.foo:13", "--publish[0] resource 'postgres.default#thing' has no output 'foo'"},
	} {
		t.Run("invalid publish "+tc[0], func(t *testing.T) {
			stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
				"generate", "score.yaml",
				"--publish", tc[0],
			})
			assert.EqualError(t, err, tc[1])
			assert.Equal(t, "", stdout)
		})
	}
}

func TestGenerateMultipleSpecsWithImage(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "scoreA.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-a
containers:
  hello:
    image: foo
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "scoreB.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-b
containers:
  hello:
    image: foo
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "--image", "nginx:latest", "scoreA.yaml", "scoreB.yaml",
	})
	assert.EqualError(t, err, "--image cannot be used when multiple score files are provided")
	assert.Equal(t, "", stdout)
}

func TestGenerateMultipleSpecsWithBuild(t *testing.T) {
	td := changeToTempDir(t)
	stdout, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "scoreA.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-a
containers:
  hello:
    image: foo
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "scoreB.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example-b
containers:
  hello:
    image: foo
`), 0644))
	stdout, _, err = executeAndResetCommand(context.Background(), rootCmd, []string{
		"generate", "--build", "foo=.", "scoreA.yaml", "scoreB.yaml",
	})
	assert.EqualError(t, err, "--build cannot be used when multiple score files are provided")
	assert.Equal(t, "", stdout)
}

func TestGenerateWithPatching(t *testing.T) {
	td := changeToTempDir(t)
	assert.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
apiVersion: score.dev/v1b1
metadata:
  name: example
  custom:
    privileged: true
containers:
  hello:
    image: foo
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "patch1.template"), []byte(`
{{ range $name, $spec := .Workloads }}
    {{ if (dig "metadata" "custom" "privileged" false $spec) }}
        {{ range $cname, $_ := $spec.containers }}
- op: set
  path: services.{{ $name }}-{{ $cname }}.privileged
  value: true
  description: Enable privileged mode on service containers
        {{ end }}
    {{ end }}
{{ end }}
---
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "patch2.template"), []byte(`
{{ range $name, $cfg := .Compose.services }}
- op: set
  path: services.{{ $name }}-future
  value: {{ toRawJson $cfg }}
  description: Rename service {{ $name }}
- op: delete
  path: services.{{ $name }}
  description: Delete service {{ $name }}
{{ end }}
---
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "patch3.template"), []byte(`
{{ range $name, $cfg := .Compose.services }}
- op: set
  path: services.{{ $name }}.read_only
  value: true
  description: Set services to read only root fs
{{ end }}
---
`), 0644))
	_, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--patch-templates", "patch1.template", "--patch-templates", "patch2.template", "--patch-templates", "patch3.template"})
	assert.NoError(t, err)
	t.Log(stderr)

	_, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{"generate", "score.yaml"})
	assert.NoError(t, err)
	t.Log(stderr)

	raw, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, string(raw), `name: "001"
services:
    example-hello-future:
        annotations:
            compose.score.dev/workload-name: example
        hostname: example
        image: foo
        privileged: true
        read_only: true
`)
}
