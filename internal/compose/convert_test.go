/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"errors"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
	assert "github.com/stretchr/testify/assert"

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
						Ports: []compose.ServicePortConfig{
							{
								Published: "80",
								Target:    8080,
							},
							{
								Published: "8080",
								Target:    8080,
								Protocol:  "udp",
							},
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
						Ports: []compose.ServicePortConfig{
							{Target: 80, Published: "8080"},
							{Target: 81, Published: "8081"},
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
			proj, vars, err := ConvertSpec(tt.Source)

			if tt.Error != nil {
				// On Error
				//
				assert.ErrorContains(t, err, tt.Error.Error())
			} else {
				// On Success
				//
				assert.NoError(t, err)
				assert.Equal(t, tt.Project, proj)
				assert.Equal(t, tt.Vars, vars.Accessed())
			}
		})
	}
}
