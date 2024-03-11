package templateprov

import (
	"context"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/util"
)

func TestProvision(t *testing.T) {
	td := t.TempDir()
	resUid := project.NewResourceUid("w", "r", "thing", nil, nil)
	p, err := Parse(map[string]interface{}{
		"uri":  "template://example",
		"type": resUid.Type(),
		"init": `
a: {{ .Uid }}
b: {{ .Type }}
`,
		"state": `
a: {{ .Init.a }}
b: stuff
`,
		"shared": `
c: 1
`,
		"outputs": `
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
		MountDirectoryPath: td,
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, map[string]interface{}{
		"a": "thing.default#w.r",
		"b": "stuff",
	}, out.ResourceState)
	assert.Equal(t, map[string]interface{}{"c": 1}, out.SharedState)
	assert.Equal(t, map[string]interface{}{"b": "STUFF", "c": 1}, out.ResourceOutputs)
	assert.Equal(t, map[string]bool{"blah": true}, out.RelativeDirectories)
	assert.Equal(t, map[string]*string{"blah/foo": util.Ref("content")}, out.RelativeFileContents)
	assert.Equal(t, map[string]compose.NetworkConfig{"default": {Driver: "default"}}, out.ComposeNetworks)
	assert.Equal(t, map[string]compose.VolumeConfig{"some-vol": {Driver: "default"}}, out.ComposeVolumes)
	assert.Equal(t, map[string]compose.ServiceConfig{"some-svc": {Name: "foo"}}, out.ComposeServices)
}
