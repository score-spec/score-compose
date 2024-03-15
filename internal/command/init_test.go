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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
)

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
    environment:
      EXAMPLE_VARIABLE: example-value
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
		assert.Equal(t, filepath.Base(td), sd.State.ComposeProjectName)
		assert.Equal(t, filepath.Join(project.DefaultRelativeStateDirectory, "mounts"), sd.State.MountsDirectory)
		assert.Equal(t, map[string]project.ScoreWorkloadState{}, sd.State.Workloads)
		assert.Equal(t, map[project.ResourceUid]project.ScoreResourceState{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
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
		assert.Equal(t, "bananas", sd.State.ComposeProjectName)
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

	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"init", "--file", "score2.yaml", "--project", "bananas"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	stdout, stderr, err = executeAndResetCommand(context.Background(), rootCmd, []string{"init"})
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	assert.NotEqual(t, "", strings.TrimSpace(stderr))

	_, err = os.Stat("score.yaml")
	assert.NoError(t, err)
	_, err = os.Stat("score2.yaml")
	assert.NoError(t, err)

	sd, ok, err := project.LoadStateDirectory(".")
	assert.NoError(t, err)
	if assert.True(t, ok) {
		assert.Equal(t, project.DefaultRelativeStateDirectory, sd.Path)
		assert.Equal(t, "bananas", sd.State.ComposeProjectName)
		assert.Equal(t, filepath.Join(project.DefaultRelativeStateDirectory, "mounts"), sd.State.MountsDirectory)
		assert.Equal(t, map[string]project.ScoreWorkloadState{}, sd.State.Workloads)
		assert.Equal(t, map[project.ResourceUid]project.ScoreResourceState{}, sd.State.Resources)
		assert.Equal(t, map[string]interface{}{}, sd.State.SharedState)
	}
}
