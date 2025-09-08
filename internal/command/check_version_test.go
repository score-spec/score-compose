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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckVersionHelp(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"check-version", "--help"})
	assert.NoError(t, err)
	assert.Equal(t, `score-compose is commonly used in Makefiles and CI pipelines which may depend on a particular functionality
or a particular default provisioner provided by score-compose init. This command provides a common way to check that
the version of score-compose matches a required version.

Usage:
  score-compose check-version [constraint] [flags]

Examples:

  # check that the version is exactly 1.2.3
  score-compose check-version =v1.2.3

  # check that the version is 1.3.0 or greater
  score-compose check-version >v1.2

  # check that the version is equal or greater to 1.2.3
  score-compose check-version >=1.2.3

Flags:
  -h, --help   help for check-version

Global Flags:
      --quiet           Mute any logging output
  -v, --verbose count   Increase log verbosity and detail by specifying this flag one or more times
`, stdout)
	assert.Equal(t, "", stderr)

	stdout2, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"help", "check-version"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, stdout2)
	assert.Equal(t, "", stderr)
}

func TestCheckVersionPass(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"check-version", ">=0.0.0"})
	assert.NoError(t, err)
	assert.Equal(t, stdout, "")
	assert.Equal(t, "", stderr)
}

func TestCheckVersionFail(t *testing.T) {
	stdout, stderr, err := executeAndResetCommand(context.Background(), rootCmd, []string{"check-version", ">99"})
	assert.EqualError(t, err, "current version 0.0.0 does not match requested constraint >99")
	assert.Equal(t, stdout, "")
	assert.Equal(t, "", stderr)
}
