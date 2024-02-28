package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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
	}()

	nowOut, nowErr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.SetOut(nowOut)
	cmd.SetErr(nowErr)
	cmd.SetArgs(args)
	subCmd, err := cmd.ExecuteContextC(ctx)
	if subCmd != nil {
		subCmd.SetContext(nil)
		subCmd.SilenceUsage = false
		subCmd.SilenceErrors = false
		subCmd.Flags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
		})
	}
	return nowOut.String(), nowErr.String(), err
}

func TestExample(t *testing.T) {
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
`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.NoError(t, err)
	assert.NotEqual(t, "", stdout)
	assert.Equal(t, "", stderr)
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
}

func TestExample_invalid_spec(t *testing.T) {
	td := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(td, "score.yaml"), []byte(`
{}`), 0600))
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"run", "--file", filepath.Join(td, "score.yaml"), "--output", filepath.Join(td, "compose.yaml")})
	assert.EqualError(t, err, "validating workload spec: jsonschema: '' does not validate with https://score.dev/schemas/score#/required: missing properties: 'apiVersion', 'metadata', 'containers'")
	assert.Equal(t, "", stdout)
	assert.Equal(t, "", stderr)
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
	assert.EqualError(t, err, "building docker-compose configuration: can't mount named volume with sub path '/sub/path': not supported")
	assert.Equal(t, "", stdout)
	assert.Equal(t, "", stderr)
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
	assert.Equal(t, "", stderr)
}
