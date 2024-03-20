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

package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
	assert "github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners/envprov"
	"github.com/score-spec/score-compose/internal/util"
)

func TestScoreConvert(t *testing.T) {
	var stringPtr = func(s string) *string {
		return &s
	}

	var tests = []struct {
		Name    string
		Source  *score.Workload
		Project *compose.Project
		Vars    map[string]string
		Error   error
	}{
		// Success path
		//
		{
			Name: "Should convert SCORE to docker-compose spec",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Service: &score.WorkloadService{
					Ports: score.WorkloadServicePorts{
						"www": score.ServicePort{
							Port:       80,
							TargetPort: util.Ref(8080),
						},
						"admin": score.ServicePort{
							Port:     8080,
							Protocol: util.Ref(score.ServicePortProtocolUDP),
						},
					},
				},
				Containers: score.WorkloadContainers{
					"backend": score.Container{
						Image: "busybox",
						Command: []string{
							"/bin/sh",
						},
						Args: []string{
							"-c",
							"while true; echo ...sleeping 10 sec...; sleep 10; done",
						},
						Variables: map[string]string{
							"CONNECTION_STRING": "test connection string",
						},
					},
				},
			},
			Project: &compose.Project{
				Services: compose.Services{
					"test-backend": {
						Name:  "test-backend",
						Image: "busybox",
						Entrypoint: compose.ShellCommand{
							"/bin/sh",
						},
						Command: compose.ShellCommand{
							"-c",
							"while true; echo ...sleeping 10 sec...; sleep 10; done",
						},
						Environment: compose.MappingWithEquals{
							"CONNECTION_STRING": stringPtr("test connection string"),
						},
					},
				},
			},
		},
		{
			Name: "Should convert all resources references",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Containers: score.WorkloadContainers{
					"backend": score.Container{
						Image: "busybox",
						Variables: map[string]string{
							"DEBUG":             "${resources.env.DEBUG}",
							"LOGS_LEVEL":        "$${LOGS_LEVEL}",
							"DOMAIN_NAME":       "${resources.some-dns.domain_name}",
							"CONNECTION_STRING": "postgresql://${resources.app-db.host}:${resources.app-db.port}/${resources.app-db.name}",
						},
						Volumes: []score.ContainerVolumesElem{
							{
								Source:   "data",
								Target:   "/mnt/data",
								ReadOnly: util.Ref(true),
							},
						},
					},
				},
				Resources: map[string]score.Resource{
					"env": {
						Type: "environment",
					},
					"app-db": {
						Type: "postgress",
					},
					"some-dns": {
						Type: "dns",
					},
					"data": {
						Type: "volume",
					},
				},
			},
			Project: &compose.Project{
				Services: compose.Services{
					"test-backend": {
						Name:  "test-backend",
						Image: "busybox",
						Environment: compose.MappingWithEquals{
							"DEBUG":             stringPtr("${DEBUG}"),
							"LOGS_LEVEL":        stringPtr("${LOGS_LEVEL}"),
							"DOMAIN_NAME":       stringPtr("${SOME_DNS_DOMAIN_NAME?required}"),
							"CONNECTION_STRING": stringPtr("postgresql://${APP_DB_HOST?required}:${APP_DB_PORT?required}/${APP_DB_NAME?required}"),
						},
						Volumes: []compose.ServiceVolumeConfig{
							{
								Type:     "volume",
								Source:   "data",
								Target:   "/mnt/data",
								ReadOnly: true,
							},
						},
					},
				},
			},
			Vars: map[string]string{
				"DEBUG":                "",
				"APP_DB_HOST":          "",
				"APP_DB_PORT":          "",
				"APP_DB_NAME":          "",
				"SOME_DNS_DOMAIN_NAME": "",
			},
		},
		{
			Name: "Should support multiple containers",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Containers: score.WorkloadContainers{
					"frontend": score.Container{
						Image: "busybox",
						Variables: map[string]string{
							"PORT": "80",
						},
					},
					"backend": score.Container{
						Image: "busybox",
						Variables: map[string]string{
							"PORT": "81",
						},
					},
				},
				Service: &score.WorkloadService{
					Ports: map[string]score.ServicePort{
						"frontend": {Port: 8080, TargetPort: util.Ref(80)},
						"backend":  {Port: 8081, TargetPort: util.Ref(81)},
					},
				},
			},
			Project: &compose.Project{
				Services: compose.Services{
					"test-backend": {
						Name:  "test-backend",
						Image: "busybox",
						Environment: compose.MappingWithEquals{
							"PORT": stringPtr("81"),
						},
					},
					"test-frontend": {
						Name:  "test-frontend",
						Image: "busybox",
						Environment: compose.MappingWithEquals{
							"PORT": stringPtr("80"),
						},
						NetworkMode: "service:test-backend",
					},
				},
			},
		},

		// Errors handling
		//
		{
			Name: "Should report an error for volumes with sub path (not supported)",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Containers: score.WorkloadContainers{
					"backend": score.Container{
						Image: "busybox",
						Volumes: []score.ContainerVolumesElem{
							{
								Source:   "data",
								Target:   "/mnt/data",
								Path:     util.Ref("sub/path"),
								ReadOnly: util.Ref(true),
							},
						},
					},
				},
				Resources: map[string]score.Resource{
					"data": {
						Type: "volume",
					},
				},
			},
			Error: errors.New("not supported"),
		},

		{
			Name: "Should report an error for volume that doesn't exist in resources",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{"name": "test"},
				Containers: score.WorkloadContainers{
					"test": score.Container{
						Image:   "busybox",
						Volumes: []score.ContainerVolumesElem{{Source: "data", Target: "/mnt/data"}},
					},
				},
			},
			Error: errors.New("containers.test.volumes[0].source: resource 'data' does not exist"),
		},

		{
			Name: "Should report an error for volume resource that isn't a volume",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{"name": "test"},
				Containers: score.WorkloadContainers{
					"test": score.Container{
						Image:   "busybox",
						Volumes: []score.ContainerVolumesElem{{Source: "data", Target: "/mnt/data"}},
					},
				},
				Resources: map[string]score.Resource{"data": {Type: "thing"}},
			},
			Error: errors.New("containers.test.volumes[0].source: resource 'data' is not a volume"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			state := &project.State{
				Workloads: map[string]project.ScoreWorkloadState{
					"test": {Spec: score.Workload{Resources: map[string]score.Resource{
						"env":      {Type: "environment"},
						"app-db":   {Type: "thing"},
						"some-dns": {Type: "thing"},
					}}},
				},
				Resources: map[project.ResourceUid]project.ScoreResourceState{},
			}
			evt := new(envprov.Provisioner)
			state.Resources["environment.default#test.env"] = project.ScoreResourceState{OutputLookupFunc: evt.LookupOutput}
			po, _ := evt.GenerateSubProvisioner("app-db", "").Provision(nil, nil)
			state.Resources["thing.default#test.app-db"] = project.ScoreResourceState{OutputLookupFunc: po.OutputLookupFunc}
			po, _ = evt.GenerateSubProvisioner("some-dns", "").Provision(nil, nil)
			state.Resources["thing.default#test.some-dns"] = project.ScoreResourceState{OutputLookupFunc: po.OutputLookupFunc}

			proj, err := ConvertSpec(state, tt.Source)

			if tt.Error != nil {
				// On Error
				//
				assert.ErrorContains(t, err, tt.Error.Error())
			} else {
				// On Success
				//
				assert.NoError(t, err)
				assert.Equal(t, tt.Project, proj)
				assert.Equal(t, tt.Vars, evt.Accessed())
			}
		})
	}
}

func TestConvertFilesIntoVolumes_nominal(t *testing.T) {
	td := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(td, "original.txt"), []byte(`first ${metadata.name} second`), 0644))
	out, err := convertFilesIntoVolumes(
		"my-workload", "my-container",
		[]score.ContainerFilesElem{
			{Target: "/ant.txt", Source: util.Ref(filepath.Join(td, "original.txt"))},
			{Target: "/bat.txt", Source: util.Ref(filepath.Join(td, "original.txt")), NoExpand: util.Ref(true)},
			{Target: "/cat.txt", Source: util.Ref(filepath.Join(td, "original.txt")), NoExpand: util.Ref(false)},
			{Target: "/dog.txt", Content: util.Ref("third ${metadata.name} fourth")},
			{Target: "/eel.txt", Content: util.Ref("third ${metadata.name} fourth"), NoExpand: util.Ref(true)},
			{Target: "/fox.txt", Content: util.Ref("third ${metadata.name} fourth"), NoExpand: util.Ref(false)},
		}, td, func(s string) (string, error) {
			switch s {
			case "metadata.name":
				return "blah", nil
			default:
				return "", fmt.Errorf("unknown key")
			}
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, []compose.ServiceVolumeConfig{
		{Type: "bind", Target: "/ant.txt", Source: filepath.Join(td, "files", "my-workload-files-0-ant.txt")},
		{Type: "bind", Target: "/bat.txt", Source: filepath.Join(td, "files", "my-workload-files-1-bat.txt")},
		{Type: "bind", Target: "/cat.txt", Source: filepath.Join(td, "files", "my-workload-files-2-cat.txt")},
		{Type: "bind", Target: "/dog.txt", Source: filepath.Join(td, "files", "my-workload-files-3-dog.txt")},
		{Type: "bind", Target: "/eel.txt", Source: filepath.Join(td, "files", "my-workload-files-4-eel.txt")},
		{Type: "bind", Target: "/fox.txt", Source: filepath.Join(td, "files", "my-workload-files-5-fox.txt")},
	}, out)
	for k, v := range map[string]string{
		"my-workload-files-0-ant.txt": "first blah second",
		"my-workload-files-1-bat.txt": "first ${metadata.name} second",
		"my-workload-files-2-cat.txt": "first blah second",
		"my-workload-files-3-dog.txt": "third blah fourth",
		"my-workload-files-4-eel.txt": "third ${metadata.name} fourth",
		"my-workload-files-5-fox.txt": "third blah fourth",
	} {
		t.Run(k, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(td, "files", k))
			assert.NoError(t, err)
			assert.Equal(t, v, string(raw))
		})
	}

}

func TestConvertFilesIntoVolumes_file_missing(t *testing.T) {
	td := t.TempDir()
	_, err := convertFilesIntoVolumes(
		"my-workload", "my-container",
		[]score.ContainerFilesElem{
			{Target: "/ant.txt", Source: util.Ref(filepath.Join(td, "original.txt"))},
		}, td, func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, fmt.Sprintf("containers.my-container.files[0].source: failed to read: open %s/original.txt: no such file or directory", td))
}

func TestConvertFilesIntoVolumes_source_missing(t *testing.T) {
	td := t.TempDir()
	_, err := convertFilesIntoVolumes(
		"my-workload", "my-container",
		[]score.ContainerFilesElem{
			{Target: "/ant.txt"},
		}, td, func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, "containers.my-container.files[0]: missing 'content' or 'source'")
}

func TestConvertFilesIntoVolumes_expand_bad(t *testing.T) {
	td := t.TempDir()
	_, err := convertFilesIntoVolumes(
		"my-workload", "my-container",
		[]score.ContainerFilesElem{
			{Target: "/ant.txt", Content: util.Ref("${metadata.blah}")},
		}, td, func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, "containers.my-container.files[0]: failed to substitute in content: unknown key")
}

func TestConvertFiles_with_mode(t *testing.T) {
	td := t.TempDir()
	out, err := convertFilesIntoVolumes(
		"my-workload", "my-container",
		[]score.ContainerFilesElem{
			{Target: "/ant.txt", Content: util.Ref("stuff")},
			{Target: "/bat.txt", Content: util.Ref("stuff"), Mode: util.Ref("0600")},
			{Target: "/cat.txt", Content: util.Ref("stuff"), Mode: util.Ref("0755")},
			{Target: "/dog.txt", Content: util.Ref("stuff"), Mode: util.Ref("0444")},
		}, td, func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.NoError(t, err)
	st, err := os.Stat(filepath.Join(td, "files/my-workload-files-0-ant.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0644, int(st.Mode()))
	assert.False(t, out[0].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-1-bat.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0600, int(st.Mode()))
	assert.False(t, out[1].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-2-cat.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0755, int(st.Mode()))
	assert.False(t, out[2].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-3-dog.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0644, int(st.Mode()))
	assert.True(t, out[3].ReadOnly)
}
