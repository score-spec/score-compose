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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
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
					"annotations": map[string]interface{}{
						"extra.name/annotation": "foo",
					},
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
						Name: "test-backend",
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
							"extra.name/annotation":           "foo",
						},
						Hostname: "test",
						Image:    "busybox",
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
						Volumes: map[string]score.ContainerVolume{
							"/mnt/data": {
								Source:   "${resources.data}",
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
						Type: "mysql",
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
						Name: "test-backend",
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
						Hostname: "test",
						Image:    "busybox",
						Environment: compose.MappingWithEquals{
							"DEBUG":             stringPtr("${DEBUG}"),
							"LOGS_LEVEL":        stringPtr("$${LOGS_LEVEL}"),
							"DOMAIN_NAME":       stringPtr("${SOME_DNS_DOMAIN_NAME?required}"),
							"CONNECTION_STRING": stringPtr("postgresql://${APP_DB_HOST?required}:${APP_DB_PORT?required}/${APP_DB_NAME?required}"),
						},
						Volumes: []compose.ServiceVolumeConfig{
							{
								Type:     "volume",
								Source:   "example",
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
			Name: "Should convert all resources references with mysql database",
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
							"CONNECTION_STRING": "mysql://${resources.app-db.host}:${resources.app-db.port}/${resources.app-db.name}",
						},
						Volumes: map[string]score.ContainerVolume{
							"/mnt/data": {
								Source:   "${resources.data}",
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
						Type: "mysql",
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
						Name: "test-backend",
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
						Hostname: "test",
						Image:    "busybox",
						Environment: compose.MappingWithEquals{
							"DEBUG":             stringPtr("${DEBUG}"),
							"LOGS_LEVEL":        stringPtr("$${LOGS_LEVEL}"),
							"DOMAIN_NAME":       stringPtr("${SOME_DNS_DOMAIN_NAME?required}"),
							"CONNECTION_STRING": stringPtr("mysql://${APP_DB_HOST?required}:${APP_DB_PORT?required}/${APP_DB_NAME?required}"),
						},
						Volumes: []compose.ServiceVolumeConfig{
							{
								Type:     "volume",
								Source:   "example",
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
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
						Name:     "test-backend",
						Hostname: "test",
						Image:    "busybox",
						Environment: compose.MappingWithEquals{
							"PORT": stringPtr("81"),
						},
					},
					"test-frontend": {
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
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
		{
			Name: "Should convert SCORE to docker-compose spec with env variables",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Containers: score.WorkloadContainers{
					"backend": score.Container{
						Image: "busybox",
						Command: []string{
							"/bin/sh",
							"-c",
						},
						Args: []string{
							"echo hello $A ${B} $C world",
						},
						Variables: map[string]string{
							"A": "cat",
							"B": "dog",
						},
					},
				},
			},
			Project: &compose.Project{
				Services: compose.Services{
					"test-backend": {
						Name: "test-backend",
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
						Image:    "busybox",
						Hostname: "test",
						Entrypoint: compose.ShellCommand{
							"/bin/sh",
							"-c",
						},
						Command: compose.ShellCommand{
							"echo hello $$A $${B} $$C world",
						},
						Environment: compose.MappingWithEquals{
							"A": stringPtr("cat"),
							"B": stringPtr("dog"),
						},
					},
				},
			},
		},

		{
			Name: "Volume with sub-path should succeed",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{
					"name": "test",
				},
				Containers: score.WorkloadContainers{
					"backend": score.Container{
						Image: "busybox",
						Volumes: map[string]score.ContainerVolume{
							"/mnt/data": {
								Source:   "${resources.data}",
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
			Project: &compose.Project{
				Services: compose.Services{
					"test-backend": {
						Name: "test-backend",
						Annotations: map[string]string{
							"compose.score.dev/workload-name": "test",
						},
						Image:       "busybox",
						Hostname:    "test",
						Environment: compose.MappingWithEquals{},
						Volumes: []compose.ServiceVolumeConfig{
							{
								Type:     "volume",
								Source:   "example",
								Target:   "/mnt/data",
								ReadOnly: true,
								Volume: &compose.ServiceVolumeVolume{
									Subpath: "sub/path",
								},
							},
						},
					},
				},
			},
		},

		// Errors handling
		//
		{
			Name: "Should report an error for volume that doesn't exist in resources",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{"name": "test"},
				Containers: score.WorkloadContainers{
					"test": score.Container{
						Image:   "busybox",
						Volumes: map[string]score.ContainerVolume{"/mnt/data": {Source: "${resources.data}"}},
					},
				},
			},
			Error: errors.New("containers.test.volumes[/mnt/data]: resource 'data' does not exist"),
		},

		{
			Name: "Should report an error for volume resource that isn't a volume",
			Source: &score.Workload{
				Metadata: score.WorkloadMetadata{"name": "test"},
				Containers: score.WorkloadContainers{
					"test": score.Container{
						Image:   "busybox",
						Volumes: map[string]score.ContainerVolume{"/mnt/data": {Source: "${resources.data}"}},
					},
				},
				Resources: map[string]score.Resource{"data": {Type: "thing"}},
			},
			Error: errors.New("containers.test.volumes[/mnt/data]: resource 'thing.default#test.data' has no 'type' output"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {

			state := &project.State{
				Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{},
				Resources: map[framework.ResourceUid]framework.ScoreResourceState[framework.NoExtras]{},
			}
			state, _ = state.WithWorkload(tt.Source, nil, project.WorkloadExtras{})
			state, _ = state.WithPrimedResources()

			evt := new(envprov.Provisioner)
			state.Resources["environment.default#test.env"] = framework.ScoreResourceState[framework.NoExtras]{OutputLookupFunc: evt.LookupOutput}
			po, _ := evt.GenerateSubProvisioner("app-db", "").Provision(context.TODO(), nil)
			state.Resources["mysql.default#test.app-db"] = framework.ScoreResourceState[framework.NoExtras]{OutputLookupFunc: po.OutputLookupFunc}
			po, _ = evt.GenerateSubProvisioner("some-dns", "").Provision(context.TODO(), nil)
			state.Resources["dns.default#test.some-dns"] = framework.ScoreResourceState[framework.NoExtras]{OutputLookupFunc: po.OutputLookupFunc}
			state.Resources["volume.default#test.data"] = framework.ScoreResourceState[framework.NoExtras]{Outputs: map[string]interface{}{
				"type":   "volume",
				"source": "example",
			}}

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
	assert.NoError(t, os.MkdirAll(filepath.Join(td, "subdir"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(td, "subdir/original.txt"), []byte(`first ${metadata.name} second`), 0644))
	state := &project.State{
		Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{"my-workload": {
			Spec: score.Workload{
				Containers: map[string]score.Container{
					"my-container": {
						Files: map[string]score.ContainerFile{
							"/ant.txt": {Source: util.Ref("original.txt")},
							"/bat.txt": {Source: util.Ref("original.txt"), NoExpand: util.Ref(true)},
							"/cat.txt": {Source: util.Ref("original.txt"), NoExpand: util.Ref(false)},
							"/dog.txt": {Content: util.Ref("third ${metadata.name} fourth")},
							"/eel.txt": {Content: util.Ref("third ${metadata.name} fourth"), NoExpand: util.Ref(true)},
							"/fox.txt": {Content: util.Ref("third ${metadata.name} fourth"), NoExpand: util.Ref(false)},
							"/goat.txt": {BinaryContent: util.Ref("ZmlmdGggJHttZXRhZGF0YS5uYW1lfSBzaXh0aA==")},
						},
					},
				},
			},
			File: util.Ref(filepath.Join(td, "subdir/score.yaml")),
		}},
		Extras: project.StateExtras{
			MountsDirectory: td,
		},
	}
	out, err := convertFilesIntoVolumes(
		state, "my-workload", "my-container",
		func(s string) (string, error) {
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
		{Type: "bind", Target: "/ant.txt", Source: filepath.Join(td, "files", "my-workload-files-ant.txt")},
		{Type: "bind", Target: "/bat.txt", Source: filepath.Join(td, "files", "my-workload-files-bat.txt")},
		{Type: "bind", Target: "/cat.txt", Source: filepath.Join(td, "files", "my-workload-files-cat.txt")},
		{Type: "bind", Target: "/dog.txt", Source: filepath.Join(td, "files", "my-workload-files-dog.txt")},
		{Type: "bind", Target: "/eel.txt", Source: filepath.Join(td, "files", "my-workload-files-eel.txt")},
		{Type: "bind", Target: "/fox.txt", Source: filepath.Join(td, "files", "my-workload-files-fox.txt")},
		{Type: "bind", Target: "/goat.txt", Source: filepath.Join(td, "files", "my-workload-files-goat.txt")},
	}, out)
	for k, v := range map[string]string{
		"my-workload-files-ant.txt": "first blah second",
		"my-workload-files-bat.txt": "first ${metadata.name} second",
		"my-workload-files-cat.txt": "first blah second",
		"my-workload-files-dog.txt": "third blah fourth",
		"my-workload-files-eel.txt": "third ${metadata.name} fourth",
		"my-workload-files-fox.txt": "third blah fourth",
		// NOTE: binaryContent doesn't support placeholders
		"my-workload-files-goat.txt": "fifth ${metadata.name} sixth",
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
	state := &project.State{
		Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{"my-workload": {
			Spec: score.Workload{
				Containers: map[string]score.Container{
					"my-container": {
						Files: map[string]score.ContainerFile{
							"/ant.txt": {Source: util.Ref(filepath.Join(td, "original.txt"))},
						},
					},
				},
			},
			File: util.Ref(filepath.Join(td, "score.yaml")),
		}},
		Extras: project.StateExtras{
			MountsDirectory: td,
		},
	}
	_, err := convertFilesIntoVolumes(
		state, "my-workload", "my-container",
		func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, fmt.Sprintf("containers.my-container.files[/ant.txt].source: failed to read: open %s/original.txt: no such file or directory", td))
}

func TestConvertFilesIntoVolumes_source_missing(t *testing.T) {
	td := t.TempDir()
	state := &project.State{
		Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{"my-workload": {
			Spec: score.Workload{
				Containers: map[string]score.Container{
					"my-container": {
						Files: map[string]score.ContainerFile{
							"/ant.txt": {},
						},
					},
				},
			},
			File: util.Ref(filepath.Join(td, "score.yaml")),
		}},
		Extras: project.StateExtras{
			MountsDirectory: td,
		},
	}
	_, err := convertFilesIntoVolumes(
		state, "my-workload", "my-container",
		func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, "containers.my-container.files[/ant.txt]: missing 'content', 'binaryContent', or 'source'")
}

func TestConvertFilesIntoVolumes_expand_bad(t *testing.T) {
	td := t.TempDir()
	state := &project.State{
		Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{"my-workload": {
			Spec: score.Workload{
				Containers: map[string]score.Container{
					"my-container": {
						Files: map[string]score.ContainerFile{
							"/ant.txt": {Content: util.Ref("${metadata.blah}")},
						},
					},
				},
			},
			File: util.Ref(filepath.Join(td, "score.yaml")),
		}},
		Extras: project.StateExtras{
			MountsDirectory: td,
		},
	}
	_, err := convertFilesIntoVolumes(
		state, "my-workload", "my-container",
		func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.EqualError(t, err, "containers.my-container.files[/ant.txt]: failed to substitute in content: unknown key")
}

func TestConvertFiles_with_mode(t *testing.T) {
	td := t.TempDir()
	state := &project.State{
		Workloads: map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{"my-workload": {
			Spec: score.Workload{
				Containers: map[string]score.Container{
					"my-container": {
						Files: map[string]score.ContainerFile{
							"/ant.txt": {Content: util.Ref("stuff")},
							"/bat.txt": {Content: util.Ref("stuff"), Mode: util.Ref("0600")},
							"/cat.txt": {Content: util.Ref("stuff"), Mode: util.Ref("0755")},
							"/dog.txt": {Content: util.Ref("stuff"), Mode: util.Ref("0444")},
						},
					},
				},
			},
			File: util.Ref(filepath.Join(td, "score.yaml")),
		}},
		Extras: project.StateExtras{
			MountsDirectory: td,
		},
	}
	out, err := convertFilesIntoVolumes(
		state, "my-workload", "my-container",
		func(s string) (string, error) {
			return "", fmt.Errorf("unknown key")
		},
	)
	assert.NoError(t, err)
	st, err := os.Stat(filepath.Join(td, "files/my-workload-files-ant.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0644, int(st.Mode()))
	assert.False(t, out[0].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-bat.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0600, int(st.Mode()))
	assert.False(t, out[1].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-cat.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0755, int(st.Mode()))
	assert.False(t, out[2].ReadOnly)
	st, err = os.Stat(filepath.Join(td, "files/my-workload-files-dog.txt"))
	assert.NoError(t, err)
	assert.Equal(t, 0644, int(st.Mode()))
	assert.True(t, out[3].ReadOnly)
}
