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

package cmdprov

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/provisioners"
)

func TestParseUri_success(t *testing.T) {
	for _, k := range []string{
		"cmd://python",
		"cmd:///absolute/path/here",
		"cmd://./relative/path",
		"cmd://../parent/path",
		"cmd://~/path",
	} {
		t.Run(k, func(t *testing.T) {
			out, err := Parse(map[string]interface{}{"uri": k, "type": "foo"})
			if assert.NoError(t, err) {
				assert.Equal(t, k, out.Uri())
			}
		})
	}
}

func TestParseUri_fail(t *testing.T) {
	for k, v := range map[string]string{
		"":                      "uri not set",
		"cmd://x:x":             "failed to parse url: parse \"cmd://x:x\": invalid port \":x\" after host",
		"cmd://something@foo":   "cmd provisioner uri cannot contain user info",
		"cmd://something:80":    "cmd provisioner uri cannot contain a port",
		"cmd://something?foo=x": "cmd provisioner uri cannot contain query params",
	} {
		t.Run(k, func(t *testing.T) {
			_, err := Parse(map[string]interface{}{"uri": k, "type": "foo"})
			assert.EqualError(t, err, v)
		})
	}
}

func TestDecodeBinary_success(t *testing.T) {
	dir, _ := os.Getwd()
	gogo, _ := exec.LookPath("go")
	for k, v := range map[string]string{
		"cmd://./thing":        dir + "/thing",
		"cmd://../thing/foo":   filepath.Dir(dir) + "/thing/foo",
		"cmd://~/path":         os.Getenv("HOME") + "/path",
		"cmd:///absolute/path": "/absolute/path",
		"cmd://go":             gogo,
	} {
		t.Run(k, func(t *testing.T) {
			out, err := decodeBinary(k)
			if assert.NoError(t, err) {
				assert.Equal(t, v, out)
			}
		})
	}
}

func TestDecodeBinary_fail(t *testing.T) {
	for k, v := range map[string]string{
		"cmd://absolutely-unknown-score-compose-provisioner": "failed to find 'absolutely-unknown-score-compose-provisioner' on path: exec: \"absolutely-unknown-score-compose-provisioner\": executable file not found in $PATH",
		"cmd://something/foo":                                "direct command reference cannot contain additional path parts",
	} {
		t.Run(k, func(t *testing.T) {
			_, err := decodeBinary(k)
			assert.EqualError(t, err, v)
		})
	}
}

func TestProvision_success(t *testing.T) {
	p, err := Parse(map[string]interface{}{
		"uri":  "cmd://sh",
		"type": "thing",
		"args": []string{"-c", "echo '{\"resource_outputs\":' `cat` '}'"},
	})
	require.NoError(t, err)
	po, err := p.Provision(context.Background(), &provisioners.Input{
		ResourceUid: "thing.default#w.r",
	})
	require.NoError(t, err)
	assert.Equal(t, "thing.default#w.r", po.ResourceOutputs["resource_uid"])
}

func TestProvision_fail_command(t *testing.T) {
	p, err := Parse(map[string]interface{}{
		"uri":  "cmd://sh",
		"type": "thing",
		"args": []string{"-c", "false"},
	})
	require.NoError(t, err)
	_, err = p.Provision(context.Background(), &provisioners.Input{
		ResourceUid: "thing.default#w.r",
	})
	require.EqualError(t, err, "failed to execute cmd provisioner: exit status 1")
}

func TestProvision_fail_decode(t *testing.T) {
	p, err := Parse(map[string]interface{}{
		"uri":  "cmd://sh",
		"type": "thing",
		"args": []string{"-c", "echo bananas"},
	})
	require.NoError(t, err)
	_, err = p.Provision(context.Background(), &provisioners.Input{
		ResourceUid: "thing.default#w.r",
	})
	require.EqualError(t, err, "failed to decode output from cmd provisioner: invalid character 'b' looking for beginning of value")
}
