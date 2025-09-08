// Copyright 2024 The Score Authors
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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/score-spec/score-go/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/loader"
)

func TestInitHelp(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--help"})
	assert.NoError(t, err)
	assert.Equal(t, `The init subcommand will prepare the current directory for working with score-compose and prepare any local
files or configuration needed to be successful.

A directory named .score-compose will be created if it doesn't exist. This file stores local state and generally should
not be checked into source control. Add it to your .gitignore file if you use Git as version control.

The project name will be used as a Docker compose project name when the final compose files are written. This name
acts as a namespace when multiple score files and containers are used.

Custom provisioners can be installed by uri using the --provisioners flag. The provisioners will be installed and take
precedence in the order they are defined over the default provisioners. If init has already been called with provisioners
the new provisioners will take precedence.

To adjust the way the compose project is generated, or perform post processing actions, you can use the --patch-templates
flag to provide one or more template files by uri. Each template file is stored in the project and then evaluated as a 
Golang text/template and should output a yaml/json encoded array of patches. Each patch is an object with required 'op' 
(set or delete), 'patch' (a dot-separated json path), a 'value' if the 'op' == 'set', and an optional 'description' for 
showing in the logs. The template has access to '.Compose' and '.Workloads'.

Usage:
  score-compose init [flags]

Examples:

  # Define a score file to generate
  score-compose init --file score2.yaml

  # Or override the docker compose project name
  score-compose init --project score-compose2

  # Or disable the default score file generation if you already have a score file
  score-compose init --no-sample

  # Optionally loading in provisoners from a remote url
  score-compose init --provisioners https://raw.githubusercontent.com/user/repo/main/example.yaml

  # Optionally adding a couple of patching templates
  score-compose init --patch-templates ./patching.tpl --patch-templates https://raw.githubusercontent.com/user/repo/main/example.tpl

URI Retrieval:
  The --provisioners and --patch-templates arguments support URI retrieval for pulling the contents from a URI on disk
  or over the network. These support:
    - HTTP        : http://host/file
    - HTTPS       : https://host/file
    - Git (SSH)   : git-ssh://git@host/repo.git/file
    - Git (HTTPS) : git-https://host/repo.git/file
    - OCI         : oci://[registry/][namespace/]repository[:tag|@digest][#file]
    - Local File  : /path/to/local/file
    - Stdin       : - (read from standard input)

Flags:
  -f, --file string                   The score file to initialize (default "./score.yaml")
  -h, --help                          help for init
      --no-default-provisioners       Disable generation of the default provisioners file
      --no-sample                     Disable generation of the sample score file
      --patch-templates stringArray   Patching template files to include. May be specified multiple times. Supports URI retrieval.
  -p, --project string                Set the name of the docker compose project (defaults to the current directory name)
      --provisioners stringArray      Provisioner files to install. May be specified multiple times. Supports URI retrieval.

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
`, stdout)
	assert.Equal(t, "", stderr)

	stdout2, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"help", "init"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, stdout2)
	assert.Equal(t, "", stderr)
}

func TestInitNominal(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	stdout, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{"run"})
	assert.NoError(t, err)
	assert.Equal(t, `services:
  example-hello-world:
    annotations:
      compose.score.dev/workload-name: example
    environment:
      EXAMPLE_VARIABLE: example-value
    hostname: example
    image: nginx:latest
    ports:
      - target: 80
        published: "8080"
`, stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	sd, ok, err := project.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, project.DefaultRelativeStateDirectory, sd.Path)
		assert.Equal(t, filepath.Base(td), sd.State.Extras.ComposeProjectName)
		assert.Equal(t, filepath.Join(project.DefaultRelativeStateDirectory, "mounts"), sd.State.Extras.MountsDirectory)
		assert.Equal(t, map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{}, sd.State.Workloads)
		assert.Equal(t, map[framework.ResourceUid]framework.ScoreResourceState[framework.NoExtras]{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
}

func TestInitNoSample(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--no-sample"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	_, err = os.Stat("score.yaml")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestInitNominal_custom_file_and_project(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--file", "score2.yaml", "--project", "bananas"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	_, err = os.Stat("score.yaml")
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat("score2.yaml")
	assert.NoError(t, err)

	sd, ok, err := project.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, project.DefaultRelativeStateDirectory, sd.Path)
		assert.Equal(t, "bananas", sd.State.Extras.ComposeProjectName)
	}
}

func TestInitNominal_bad_project(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--project", "-this-is-invalid-"})
	assert.EqualError(t, err, "invalid value for --project, it must match ^[a-z0-9][a-z0-9_-]*$")
	assert.Equal(t, "", stdout)
	assert.Equal(t, "", stderr)
}

func TestInitNominal_run_twice(t *testing.T) {
	td := t.TempDir()

	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	// first init
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--file", "score2.yaml", "--project", "bananas"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	// check default provisioners exists and overwrite it with an empty array
	dpf, err := os.Stat(filepath.Join(td, ".score-compose", "zz-default.provisioners.yaml"))
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(td, ".score-compose", dpf.Name()), []byte("[]"), 0644))

	// init again
	stdout, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	// verify that default provisioners was not overwritten again
	dpf, err = os.Stat(filepath.Join(td, ".score-compose", dpf.Name()))
	assert.NoError(t, err)
	assert.Equal(t, 2, int(dpf.Size()))

	_, err = os.Stat("score.yaml")
	assert.NoError(t, err)
	_, err = os.Stat("score2.yaml")
	assert.NoError(t, err)

	sd, ok, err := project.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, project.DefaultRelativeStateDirectory, sd.Path)
		assert.Equal(t, "bananas", sd.State.Extras.ComposeProjectName)
		assert.Equal(t, filepath.Join(project.DefaultRelativeStateDirectory, "mounts"), sd.State.Extras.MountsDirectory)
		assert.Equal(t, map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{}, sd.State.Workloads)
		assert.Equal(t, map[framework.ResourceUid]framework.ScoreResourceState[framework.NoExtras]{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
}

func TestInitWithProvisioners(t *testing.T) {
	td := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()

	td2 := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td2, "one.provisioners.yaml"), []byte(`
- uri: template://one
  type: thing
  outputs: "{}"
`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td2, "two.provisioners.yaml"), []byte(`
- uri: template://two
  type: thing
  outputs: "{}"
`), 0644))

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--provisioners", filepath.Join(td2, "one.provisioners.yaml"), "--provisioners", "file://" + filepath.Join(td2, "two.provisioners.yaml")})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	provs, err := loader.LoadProvisionersFromDirectory(filepath.Join(td, ".score-compose"), loader.DefaultSuffix)
	assert.NoError(t, err)
	expectedProvisionerUris := []string{"template://one", "template://two"}
	for _, expectedUri := range expectedProvisionerUris {
		assert.True(t, slices.ContainsFunc(provs, func(p provisioners.Provisioner) bool {
			return p.Uri() == expectedUri
		}), fmt.Sprintf("Expected provisioner '%s' not found", expectedUri))
	}
}

func TestInitWithPatchingFiles(t *testing.T) {
	td := t.TempDir()
	wd, _ := os.Getwd()
	require.NoError(t, os.Chdir(td))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "patch-templates-1"), []byte(`[]`), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "patch-templates-2"), []byte(`[]`), 0644))

	t.Run("new", func(t *testing.T) {
		_, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--patch-templates", filepath.Join(td, "patch-templates-1"), "--patch-templates", filepath.Join(td, "patch-templates-2")})
		assert.NoError(t, err)
		t.Log(stderr)
		sd, ok, err := project.LoadStateDirectory(".")
		assert.NoError(t, err)
		if assert.True(t, ok) {
			assert.Len(t, sd.State.Extras.PatchingTemplates, 2)
		}
	})

	t.Run("update", func(t *testing.T) {
		_, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--patch-templates", filepath.Join(td, "patch-templates-2")})
		assert.NoError(t, err)
		t.Log(stderr)
		sd, ok, err := project.LoadStateDirectory(".")
		assert.NoError(t, err)
		if assert.True(t, ok) {
			assert.Len(t, sd.State.Extras.PatchingTemplates, 1)
		}
	})

	t.Run("bad patch", func(t *testing.T) {
		assert.NoError(t, os.WriteFile(filepath.Join(td, "patch-templates-3"), []byte(`{{ what is this }}`), 0644))
		_, _, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--patch-templates", filepath.Join(td, "patch-templates-3")})
		assert.Error(t, err, "failed to parse template: template: :1: function \"what\" not defined")
	})
}
