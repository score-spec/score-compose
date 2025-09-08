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

package provisioners

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

func TestApplyToStateAndProject(t *testing.T) {
	resUid := framework.NewResourceUid("w", "r", "t", nil, nil)
	startState := &project.State{
		Resources: map[framework.ResourceUid]framework.ScoreResourceState[framework.NoExtras]{
			resUid: {},
		},
	}

	t.Run("set first provision with no outputs", func(t *testing.T) {
		td := t.TempDir()
		startState.Extras.MountsDirectory = td
		composeProject := &compose.Project{}
		output := &ProvisionOutput{}
		afterState, err := output.ApplyToStateAndProject(startState, resUid, composeProject)
		require.NoError(t, err)
		assert.Equal(t, framework.ScoreResourceState[framework.NoExtras]{
			State:   map[string]interface{}{},
			Outputs: map[string]interface{}{},
		}, afterState.Resources[resUid])
	})

	t.Run("set first provision with some outputs", func(t *testing.T) {
		td := t.TempDir()
		startState.Extras.MountsDirectory = td
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
		assert.Equal(t, framework.ScoreResourceState[framework.NoExtras]{
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

func TestProvisionResourcesWithNetworkService(t *testing.T) {
	state := new(project.State)
	state, _ = state.WithWorkload(&score.Workload{
		Metadata: map[string]interface{}{"name": "w1"},
		Service: &score.WorkloadService{
			Ports: score.WorkloadServicePorts{
				"web":  {Port: 80},
				"grpc": {Port: 9000, TargetPort: util.Ref(9001), Protocol: util.Ref(score.ServicePortProtocolUDP)},
			},
		},
		Containers: map[string]score.Container{
			"container-a": {},
			"container-b": {},
		},
		Resources: map[string]score.Resource{
			"a": {Type: "thing"},
		},
	}, nil, project.WorkloadExtras{})
	state, _ = state.WithPrimedResources()
	p := []Provisioner{
		NewEphemeralProvisioner("ephemeral://blah", "thing.default#w1.a", func(ctx context.Context, input *Input) (*ProvisionOutput, error) {
			assert.Equal(t, "w1", input.SourceWorkload)
			assert.Equal(t, map[string]NetworkService{
				"w1": {
					ServiceName: "w1",
					Ports: map[string]ServicePort{
						"web":  {Name: "web", Port: 80, TargetPort: 80, Protocol: score.ServicePortProtocolTCP},
						"80":   {Name: "web", Port: 80, TargetPort: 80, Protocol: score.ServicePortProtocolTCP},
						"grpc": {Name: "grpc", Port: 9000, TargetPort: 9001, Protocol: score.ServicePortProtocolUDP},
						"9000": {Name: "grpc", Port: 9000, TargetPort: 9001, Protocol: score.ServicePortProtocolUDP},
					},
				},
			}, input.WorkloadServices)
			return &ProvisionOutput{}, nil
		}),
	}
	after, err := ProvisionResources(context.Background(), state, p, nil)
	if assert.NoError(t, err) {
		assert.Len(t, after.Resources, 1)
	}
}

func TestProvisionResourcesWithResourceParams(t *testing.T) {
	state := new(project.State)
	state, _ = state.WithWorkload(&score.Workload{
		Metadata: map[string]interface{}{"name": "w1"},
		Resources: map[string]score.Resource{
			"a": {Type: "a", Params: map[string]interface{}{"x": "${resources.b.key}"}},
			"b": {Type: "b"},
		},
	}, nil, project.WorkloadExtras{})
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
	}, nil, project.WorkloadExtras{})
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
