/*
Apache Score
Copyright 2020 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"errors"
	"testing"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
	assert "github.com/stretchr/testify/assert"
)

func TestScoreConvert(t *testing.T) {
	var stringPtr = func(s string) *string {
		return &s
	}

	var tests = []struct {
		Name    string
		Source  *score.WorkloadSpec
		Project *compose.Project
		Vars    ExternalVariables
		Error   error
	}{
		// Success path
		//
		{
			Name: "Should convert SCORE to docker-compose spec",
			Source: &score.WorkloadSpec{
				Metadata: score.WorkloadMeta{
					Name: "test",
				},
				Service: score.ServiceSpec{
					Ports: score.ServicePortsSpecs{
						"www": score.ServicePortSpec{
							Port:       80,
							TargetPort: 8080,
						},
						"admin": score.ServicePortSpec{
							Port:     8080,
							Protocol: "UDP",
						},
					},
				},
				Containers: score.ContainersSpecs{
					"backend": score.ContainerSpec{
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
					{
						Name:  "test",
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
						DependsOn: make(compose.DependsOnConfig, 0),
						Ports: []compose.ServicePortConfig{
							{
								Published: "80",
								Target:    8080,
							},
							{
								Published: "8080",
								Target:    8080,
								Protocol:  "UDP",
							},
						},
					},
				},
			},
			Vars: ExternalVariables{},
		},
		{
			Name: "Should convert all resources references",
			Source: &score.WorkloadSpec{
				Metadata: score.WorkloadMeta{
					Name: "test",
				},
				Containers: score.ContainersSpecs{
					"backend": score.ContainerSpec{
						Image: "busybox",
						Variables: map[string]string{
							"DEBUG":             "${resources.env.DEBUG}",
							"LOGS_LEVEL":        "$${LOGS_LEVEL}",
							"DOMAIN_NAME":       "${resources.dns.domain_name}",
							"CONNECTION_STRING": "postgresql://${resources.app-db.host}:${resources.app-db.port}/${resources.app-db.name}",
						},
						Volumes: []score.VolumeMountSpec{
							{
								Source:   "${resources.data}",
								Target:   "/mnt/data",
								ReadOnly: true,
							},
						},
					},
				},
				Resources: map[string]score.ResourceSpec{
					"env": {
						Type: "environment",
						Properties: map[string]score.ResourcePropertySpec{
							"DEBUG":      {Default: false, Required: false},
							"LOGS_LEVEL": {Default: "WARN", Required: false},
						},
					},
					"app-db": {
						Properties: map[string]score.ResourcePropertySpec{
							"host":      {Default: "localhost", Required: true},
							"port":      {Default: 5432, Required: false},
							"name":      {Required: true},
							"user.name": {Required: true, Secret: true},
							"password":  {Required: true, Secret: true},
						},
					},
					"dns": {
						Type: "dns",
						Properties: map[string]score.ResourcePropertySpec{
							"domain": {Required: false},
						},
					},
					"data": {
						Type: "volume",
					},
				},
			},
			Project: &compose.Project{
				Services: compose.Services{
					{
						Name:  "test",
						Image: "busybox",
						Environment: compose.MappingWithEquals{
							"DEBUG":             stringPtr("${DEBUG-false}"),
							"LOGS_LEVEL":        stringPtr("${LOGS_LEVEL}"),
							"DOMAIN_NAME":       stringPtr(""),
							"CONNECTION_STRING": stringPtr("postgresql://${APP_DB_HOST-localhost}:${APP_DB_PORT-5432}/${APP_DB_NAME?err}"),
						},
						Volumes: []compose.ServiceVolumeConfig{
							{
								Type:     "volume",
								Source:   "data",
								Target:   "/mnt/data",
								ReadOnly: true,
							},
						},
						DependsOn: compose.DependsOnConfig{
							"app-db": compose.ServiceDependency{Condition: "service_started"},
							"dns":    compose.ServiceDependency{Condition: "service_started"},
						},
					},
				},
			},
			Vars: ExternalVariables{
				"DEBUG":            false,
				"LOGS_LEVEL":       "WARN",
				"APP_DB_HOST":      "localhost",
				"APP_DB_PORT":      5432,
				"APP_DB_NAME":      nil,
				"APP_DB_USER_NAME": nil,
				"APP_DB_PASSWORD":  nil,
				"DNS_DOMAIN":       nil,
			},
		},

		// Errors handling
		//
		{
			Name: "Should report an error for volumes with sub path (not supported)",
			Source: &score.WorkloadSpec{
				Metadata: score.WorkloadMeta{
					Name: "test",
				},
				Containers: score.ContainersSpecs{
					"backend": score.ContainerSpec{
						Image: "busybox",
						Volumes: []score.VolumeMountSpec{
							{
								Source:   "${resources.data}",
								Path:     "sub/path",
								Target:   "/mnt/data",
								ReadOnly: true,
							},
						},
					},
				},
				Resources: map[string]score.ResourceSpec{
					"data": {
						Type: "volume",
					},
				},
			},
			Error: errors.New("not supported"),
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
				assert.Equal(t, tt.Vars, vars)
			}
		})
	}
}
