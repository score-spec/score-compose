package provisioners

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
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
