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

package templateprov

import (
	"context"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/util"
)

func TestProvision(t *testing.T) {
	td := t.TempDir()
	resUid := framework.NewResourceUid("w", "r", "thing", nil, nil)
	p, err := Parse(map[string]interface{}{
		"uri":              "template://example",
		"type":             resUid.Type(),
		"expected_outputs": []string{"b", "c"},
		"supported_params": []string{"ptest"},
		"init": `
a: {{ .Uid }}
b: {{ .Type }}
`,
		"state": `
a: {{ .Init.a }}
b: stuff
p: {{ .ComposeProjectName }}
`,
		"shared": `
c: 1
`,
		"outputs": `
{{ if not .Params.ptest }}{{ end }}
b: {{ .State.b | upper }}
c: {{ .Shared.c }}
`,
		"directories": `{"blah": true}`,
		"files":       `{"blah/foo": "content"}`,
		"networks": `
default:
  driver: default
`,
		"volumes": `
some-vol:
  driver: default
`,
		"services": `
some-svc:
  name: foo
`,
		"info_logs": `
- This is a message
- This is another message
`,
	})
	require.NoError(t, err)
	out, err := p.Provision(context.Background(), &provisioners.Input{
		ResourceUid:        string(resUid),
		ResourceType:       resUid.Type(),
		ResourceClass:      resUid.Class(),
		ResourceId:         resUid.Id(),
		ResourceParams:     map[string]interface{}{"pk": "pv"},
		ResourceMetadata:   map[string]interface{}{"mk": "mv"},
		ResourceState:      map[string]interface{}{"sk": "sv"},
		SharedState:        map[string]interface{}{"ssk": "ssv"},
		ComposeProjectName: "test",
		MountDirectoryPath: td,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, map[string]interface{}{
		"a": "thing.default#w.r",
		"b": "stuff",
		"p": "test",
	}, out.ResourceState)
	assert.Equal(t, map[string]interface{}{"c": 1}, out.SharedState)
	assert.Equal(t, map[string]interface{}{"b": "STUFF", "c": 1}, out.ResourceOutputs)
	assert.Equal(t, map[string]bool{"blah": true}, out.RelativeDirectories)
	assert.Equal(t, map[string]*string{"blah/foo": util.Ref("content")}, out.RelativeFileContents)
	assert.Equal(t, map[string]compose.NetworkConfig{"default": {Driver: "default"}}, out.ComposeNetworks)
	assert.Equal(t, map[string]compose.VolumeConfig{"some-vol": {Driver: "default"}}, out.ComposeVolumes)
	assert.Equal(t, map[string]compose.ServiceConfig{"some-svc": {Name: "foo"}}, out.ComposeServices)
	assert.Equal(t, []string{"b", "c"}, p.Outputs())
	assert.Equal(t, []string{"ptest"}, p.Params())
	assert.Equal(t, "(any)", p.Class())
	assert.Equal(t, resUid.Type(), p.Type())
}
