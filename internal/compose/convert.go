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
	"strings"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
)

// ConvertSpec converts SCORE specification into docker-compose configuration.
func ConvertSpec(spec *score.Workload) (*compose.Project, *EnvVarTracker, error) {
	// Track any uses of the environment resource or resources that are overridden with an env provider using the tracker.
	envVarTracker := NewEnvVarTracker()

	var project = compose.Project{
		Services: make(compose.Services, 0, len(spec.Containers)),
	}

	// this map holds the results of the provisioning process
	resources := make(map[string]ResourceWithOutputs)

	// The first thing we must do is validate or create the resources this workload depends on.
	// NOTE: this will soon be replaced by a much more sophisticated resource provisioning system!
	for resourceName, resourceSpec := range spec.Resources {
		if resourceSpec.Type == "environment" {
			if DerefOr(resourceSpec.Class, "default") != "default" {
				return nil, nil, fmt.Errorf("resources.%s: '%s.%s' is not supported in score-compose", resourceName, resourceSpec.Type, *resourceSpec.Class)
			}
			resources[resourceName] = envVarTracker
		} else {
			// TODO: only enable this if the type.class is in an allow-list or the allow-list is '*' - otherwise return an error
			resources[resourceName] = envVarTracker.GenerateResource(resourceName)
		}
	}

	ctx, err := buildContext(spec.Metadata, resources)
	if err != nil {
		return nil, nil, fmt.Errorf("preparing context: %w", err)
	}

	// This is already  validated by spec validation
	workloadName, _ := spec.Metadata["name"].(string)

	var ports []compose.ServicePortConfig
	if spec.Service != nil && len(spec.Service.Ports) > 0 {
		ports = []compose.ServicePortConfig{}
		for _, pSpec := range spec.Service.Ports {
			var pubPort = fmt.Sprintf("%v", pSpec.Port)
			var protocol string
			if pSpec.Protocol != nil {
				protocol = strings.ToLower(string(*pSpec.Protocol))
			}
			ports = append(ports, compose.ServicePortConfig{
				Published: pubPort,
				Target:    uint32(DerefOr(pSpec.TargetPort, pSpec.Port)),
				Protocol:  protocol,
			})
		}
	}

	// When multiple containers are specified we need to identify one container as the "main" container which will own
	// the network and use the native workload name. All other containers in this workload will have the container
	// name appended as a suffix. We use the natural sort order of the container names and pick the first one
	containerNames := make([]string, 0, len(spec.Containers))
	for name := range spec.Containers {
		containerNames = append(containerNames, name)
	}
	sort.Strings(containerNames)

	for _, containerName := range containerNames {
		cSpec := spec.Containers[containerName]

		var env = make(compose.MappingWithEquals, len(cSpec.Variables))
		for key, val := range cSpec.Variables {
			resolved, err := ctx.Substitute(val)
			if err != nil {
				return nil, nil, fmt.Errorf("containers.%s.variables.%s: %w", containerName, key, err)
			}
			env[key] = &resolved
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
				if vol.Path != nil && *vol.Path != "" {
					return nil, nil, fmt.Errorf("can't mount named volume with sub path '%s': %w", *vol.Path, errors.New("not supported"))
				}
				resolvedVolumeSource, err := ctx.Substitute(vol.Source)
				if err != nil {
					return nil, nil, fmt.Errorf("containers.%s.volumes[%d].source: %w", containerName, idx, err)
				}
				volumes[idx] = compose.ServiceVolumeConfig{
					Type:     "volume",
					Source:   resolvedVolumeSource,
					Target:   vol.Target,
					ReadOnly: DerefOr(vol.ReadOnly, false),
				}
			}
		}
		// NOTE: Sorting is necessary for DeepEqual call within our Unit Tests to work reliably
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Source < volumes[j].Source
		})
		// END (NOTE)

		var svc = compose.ServiceConfig{
			Name:        workloadName + "-" + containerName,
			Image:       cSpec.Image,
			Entrypoint:  cSpec.Command,
			Command:     cSpec.Args,
			Environment: env,
			Ports:       ports,
			Volumes:     volumes,
		}

		// if we are not the "first" service, then inherit the network from the first service
		if len(project.Services) > 0 {
			svc.Ports = nil
			svc.NetworkMode = "service:" + project.Services[0].Name
		}

		project.Services = append(project.Services, svc)
	}
	return &project, envVarTracker, nil
}
