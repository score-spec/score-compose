/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"errors"
	"fmt"
	"sort"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
)

// ConvertSpec converts SCORE specification into docker-compose configuration.
func ConvertSpec(spec *score.WorkloadSpec) (*compose.Project, ExternalVariables, error) {
	context, err := buildContext(spec.Metadata, spec.Resources)
	if err != nil {
		return nil, nil, fmt.Errorf("preparing context: %w", err)
	}

	for _, cSpec := range spec.Containers {
		var externalVars = ExternalVariables(context.ListEnvVars())
		var env = make(compose.MappingWithEquals, len(cSpec.Variables))
		for key, val := range cSpec.Variables {
			var envVarVal = context.Substitute(val)
			env[key] = &envVarVal
		}

		var dependsOn = make(compose.DependsOnConfig, len(spec.Resources))
		for name, res := range spec.Resources {
			if res.Type != "environment" && res.Type != "volume" {
				dependsOn[name] = compose.ServiceDependency{Condition: "service_started"}
			}
		}

		var ports []compose.ServicePortConfig
		if len(spec.Service.Ports) > 0 {
			ports = []compose.ServicePortConfig{}
			for _, pSpec := range spec.Service.Ports {
				var pubPort = fmt.Sprintf("%v", pSpec.Port)
				var tgtPort = pSpec.TargetPort
				if pSpec.TargetPort == 0 {
					tgtPort = pSpec.Port
				}
				ports = append(ports, compose.ServicePortConfig{
					Published: pubPort,
					Target:    uint32(tgtPort),
					Protocol:  pSpec.Protocol,
				})
			}
		}
		// NOTE: Sorting is necessary for DeepEqual call within our Unit Tests to work reliably
		sort.Slice(ports, func(i, j int) bool {
			return ports[i].Published < ports[j].Published
		})
		// END (NOTE)

		var volumes []compose.ServiceVolumeConfig
		if len(cSpec.Volumes) > 0 {
			volumes = make([]compose.ServiceVolumeConfig, len(cSpec.Volumes))
			for idx, vol := range cSpec.Volumes {
				if vol.Path != "" {
					return nil, nil, fmt.Errorf("can't mount named volume with sub path '%s': %w", vol.Path, errors.New("not supported"))
				}
				volumes[idx] = compose.ServiceVolumeConfig{
					Type:     "volume",
					Source:   context.Substitute(vol.Source),
					Target:   vol.Target,
					ReadOnly: vol.ReadOnly,
				}
			}
		}
		// NOTE: Sorting is necessary for DeepEqual call within our Unit Tests to work reliably
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Source < volumes[j].Source
		})
		// END (NOTE)

		var svc = compose.ServiceConfig{
			Name:        spec.Metadata.Name,
			Image:       cSpec.Image,
			Entrypoint:  cSpec.Command,
			Command:     cSpec.Args,
			Environment: env,
			DependsOn:   dependsOn,
			Ports:       ports,
			Volumes:     volumes,
		}

		var proj = compose.Project{
			Services: compose.Services{
				svc,
			},
		}

		// NOTE: Only one container per workload can be defined for compose.
		//       All other containers will be ignored by this tool.
		return &proj, externalVars, nil
	}

	return nil, nil, errors.New("workload does not have any containers to convert into a compose service")
}
