package provisioners

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

func TestApplyToStateAndProject(t *testing.T) {
	resUid := project.NewResourceUid("w", "r", "t", nil, nil)
	startState := &project.State{
		Resources: map[project.ResourceUid]project.ScoreResourceState{
			resUid: {},
		},
	}

	t.Run("set first provision with no outputs", func(t *testing.T) {
		td := t.TempDir()
		startState.MountsDirectory = td
		composeProject := &compose.Project{}
		output := &ProvisionOutput{}
		afterState, err := output.ApplyToStateAndProject(startState, resUid, composeProject)
		require.NoError(t, err)
		assert.Equal(t, project.ScoreResourceState{
			State:   map[string]interface{}{},
			Outputs: map[string]interface{}{},
		}, afterState.Resources[resUid])
	})

	t.Run("set first provision with some outputs", func(t *testing.T) {
		td := t.TempDir()
		startState.MountsDirectory = td
		composeProject := &compose.Project{}
		output := &ProvisionOutput{
			ResourceState:   map[string]interface{}{"a": "b", "c": nil},
			ResourceOutputs: map[string]interface{}{"x": "y"},
			SharedState:     map[string]interface{}{"i": "j", "k": nil},
			RelativeDirectories: map[string]bool{
				"one/two/three": true,
				"four":          false,
				"five":          true,
			},
			RelativeFileContents: map[string]*string{
				"one/two/three/thing.txt": util.Ref("hello-world"),
				"six/other.txt":           util.Ref("blah"),
				"something.md":            nil,
			},
			ComposeNetworks: map[string]compose.NetworkConfig{
				"some-network": {Name: "network"},
			},
			ComposeServices: map[string]compose.ServiceConfig{
				"some-service": {Name: "service"},
			},
			ComposeVolumes: map[string]compose.VolumeConfig{
				"some-volume": {Name: "volume"},
			},
		}
		afterState, err := output.ApplyToStateAndProject(startState, resUid, composeProject)
		require.NoError(t, err)
		assert.Equal(t, project.ScoreResourceState{
			State:   map[string]interface{}{"a": "b", "c": nil},
			Outputs: map[string]interface{}{"x": "y"},
		}, afterState.Resources[resUid])
		assert.Equal(t, map[string]interface{}{"i": "j"}, afterState.SharedState)
		assert.Len(t, composeProject.Networks, 1)
		assert.Len(t, composeProject.Volumes, 1)
		assert.Len(t, composeProject.Services, 1)
		paths := make([]string, 0)
		_ = filepath.WalkDir(td, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				if items, _ := os.ReadDir(path); len(items) > 0 {
					return nil
				}
				path, _ = filepath.Rel(td, path)
				paths = append(paths, path+"/")
			} else {
				path, _ = filepath.Rel(td, path)
				paths = append(paths, path)
			}
			return nil
		})
		slices.Sort(paths)
		assert.Equal(t, []string{
			"five/",
			"one/two/three/thing.txt",
			"six/other.txt",
		}, paths)
	})

}

func TestProvisionResourcesWithResourceParams(t *testing.T) {
	state := new(project.State)
	state, _ = state.WithWorkload(&score.Workload{
		Metadata: map[string]interface{}{"name": "w1"},
		Resources: map[string]score.Resource{
			"a": {Type: "a", Params: map[string]interface{}{"x": "${resources.b.key}"}},
			"b": {Type: "b"},
		},
	}, nil, nil)
	state, _ = state.WithPrimedResources()
	p := []Provisioner{
		NewEphemeralProvisioner("ephemeral://blah", "a.default#w1.a", func(ctx context.Context, input *Input) (*ProvisionOutput, error) {
			assert.Equal(t, map[string]interface{}{"x": "value"}, input.ResourceParams)
			return &ProvisionOutput{}, nil
		}),
		NewEphemeralProvisioner("ephemeral://blah", "b.default#w1.b", func(ctx context.Context, input *Input) (*ProvisionOutput, error) {
			return &ProvisionOutput{ResourceOutputs: map[string]interface{}{"key": "value"}}, nil
		}),
	}
	after, err := ProvisionResources(context.Background(), state, p, nil)
	if assert.NoError(t, err) {
		assert.Len(t, after.Resources, 2)
	}
}

func TestProvisionResourcesWithResourceParams_fail(t *testing.T) {
	state := new(project.State)
	state, _ = state.WithWorkload(&score.Workload{
		Metadata: map[string]interface{}{"name": "w1"},
		Resources: map[string]score.Resource{
			"a": {Type: "a", Params: map[string]interface{}{"x": "${resources.b.unknown}"}},
			"b": {Type: "b"},
		},
	}, nil, nil)
	state, _ = state.WithPrimedResources()
	p := []Provisioner{
		NewEphemeralProvisioner("ephemeral://blah", "a.default#w1.a", func(ctx context.Context, input *Input) (*ProvisionOutput, error) {
			return &ProvisionOutput{}, nil
		}),
		NewEphemeralProvisioner("ephemeral://blah", "b.default#w1.b", func(ctx context.Context, input *Input) (*ProvisionOutput, error) {
			return &ProvisionOutput{ResourceOutputs: map[string]interface{}{}}, nil
		}),
	}
	_, err := ProvisionResources(context.Background(), state, p, nil)
	assert.EqualError(t, err, "failed to substitute params for resource 'a.default#w1.a': x: invalid ref 'resources.b.unknown': key 'unknown' not found")
}
