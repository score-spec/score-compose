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
	beforeOut, beforeErr := cmd.OutOrStderr(), cmd.ErrOrStderr()
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
	assert.Equal(t, `Translate the SCORE file to docker-compose configuration

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
    command:
      - -c
      - while true; do echo Hello World!; sleep 5; done
    entrypoint:
      - /bin/sh
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

func TestRunExample02(t *testing.T) {
	td := t.TempDir()
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", "../../examples/02-environment/score.yaml", "--output", filepath.Join(td, "compose.yaml")})
	assert.NoError(t, err)

	expectedOutput := `services:
  hello-world-hello:
    command:
      - -c
      - while true; do echo Hello $${FRIEND}!; sleep 5; done
    entrypoint:
      - /bin/sh
    environment:
      FRIEND: ${NAME}
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

func TestRunExample03(t *testing.T) {
	td := t.TempDir()

	t.Run("a", func(t *testing.T) {
		stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", "../../examples/03-dependencies/service-a.yaml", "--output", filepath.Join(td, "compose-a.yaml")})
		assert.NoError(t, err)

		expectedOutput := `services:
  service-a-service-a:
    command:
      - -c
      - 'while true; do echo service-a: Hello $${FRIEND}! Connecting to $${CONNECTION_STRING}...; sleep 10; done'
    entrypoint:
      - /bin/sh
    environment:
      CONNECTION_STRING: postgresql://${DB_USER?required}:${DB_PASSWORD?required}@${DB_HOST?required}:${DB_PORT?required}/${DB_NAME?required}
      FRIEND: ${NAME}
    image: busybox
`

		assert.Equal(t, expectedOutput, stdout)
		assert.NotEqual(t, "", strings.TrimSpace(stderr))
		rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose-a.yaml"))
		require.NoError(t, err)
		assert.Equal(t, expectedOutput, string(rawComposeContent))
	})

	t.Run("b", func(t *testing.T) {
		stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", "../../examples/03-dependencies/service-b.yaml", "--output", filepath.Join(td, "compose-b.yaml")})
		assert.NoError(t, err)

		expectedOutput := `services:
  service-b-service-b:
    command:
      - -c
      - 'while true; do echo service-b: Hello $${FRIEND}!; sleep 5; done'
    entrypoint:
      - /bin/sh
    environment:
      FRIEND: ${NAME}
    image: busybox
`

		assert.Equal(t, expectedOutput, stdout)
		assert.NotEqual(t, "", strings.TrimSpace(stderr))
		rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose-b.yaml"))
		require.NoError(t, err)
		assert.Equal(t, expectedOutput, string(rawComposeContent))
	})

	t.Run("validate", func(t *testing.T) {
		if os.Getenv("NO_DOCKER") != "" {
			t.Skip("NO_DOCKER is set")
			return
		}

		baseContent, err := os.ReadFile("../../examples/03-dependencies/compose.yaml")
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(filepath.Join(td, "compose.yaml"), baseContent, 0644))

		dockerCmd, err := exec.LookPath("docker")
		require.NoError(t, err)

		cmd := exec.Command(dockerCmd, "compose", "-f", "compose.yaml", "-f", "compose-a.yaml", "-f", "compose-b.yaml", "convert", "--quiet", "--dry-run")
		cmd.Dir = td
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), "DB_HOST=host", "DB_PORT=80", "DB_NAME=default", "DB_USER=myuser", "DB_PASSWORD=password")
		assert.NoError(t, cmd.Run())
	})
}

func TestRunExample04(t *testing.T) {
	td := t.TempDir()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", "../../examples/04-extras/score.yaml", "--output", filepath.Join(td, "compose.yaml")})
	assert.NoError(t, err)

	expectedOutput := `services:
  web-app-hello:
    image: nginx
    ports:
      - target: 80
        published: "8000"
    volumes:
      - type: volume
        source: data
        target: /usr/share/nginx/html
        read_only: true
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

		baseContent, err := os.ReadFile("../../examples/04-extras/compose.yaml")
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(filepath.Join(td, "extras.yaml"), baseContent, 0644))

		dockerCmd, err := exec.LookPath("docker")
		require.NoError(t, err)

		cmd := exec.Command(dockerCmd, "compose", "-f", "compose.yaml", "-f", "extras.yaml", "convert", "--quiet", "--dry-run")
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
    build:
      context: ./test
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
    image: bananas:latest
`

	assert.Equal(t, expectedOutput, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))
	rawComposeContent, err := os.ReadFile(filepath.Join(td, "compose.yaml"))
	require.NoError(t, err)
	assert.Equal(t, expectedOutput, string(rawComposeContent))
}
