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
  -h, --help                            help for generate
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

func changeToTempDir(t *testing.T) string {
	t.Helper()
	td := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})
	return td
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
        ports:
            - target: 80
              published: "8080"
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
