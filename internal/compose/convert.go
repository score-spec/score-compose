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
	"log/slog"
	"sort"
	"strings"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

// ConvertSpec converts SCORE specification into docker-compose configuration.
func ConvertSpec(spec *score.Workload, containerBuildConfigs map[string]compose.BuildConfig, resources map[string]project.OutputLookupFunc) (*compose.Project, error) {
	workloadName, ok := spec.Metadata["name"].(string)
	if !ok || len(workloadName) == 0 {
		return nil, errors.New("workload metadata is missing a name")
	}

	if len(spec.Containers) == 0 {
		return nil, errors.New("workload does not have any containers to convert into a compose service")
	}

	var project = compose.Project{
		Services: make(compose.Services),
	}

	ctx, err := buildContext(spec.Metadata, resources)
	if err != nil {
		return nil, fmt.Errorf("preparing context: %w", err)
	}

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
				Target:    uint32(util.DerefOr(pSpec.TargetPort, pSpec.Port)),
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

	var firstService string
	for _, containerName := range containerNames {
		cSpec := spec.Containers[containerName]

		var env = make(compose.MappingWithEquals, len(cSpec.Variables))
		for key, val := range cSpec.Variables {
			resolved, err := ctx.Substitute(val)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.variables.%s: %w", containerName, key, err)
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
					return nil, fmt.Errorf("containers.%s.volumes[%d].path: can't mount named volume with sub path '%s': not supported", containerName, idx, *vol.Path)
				}

				resolvedVolumeSource, err := ctx.Substitute(vol.Source)
				if err != nil {
					return nil, fmt.Errorf("containers.%s.volumes[%d].source: %w", containerName, idx, err)
				}

				if res, ok := spec.Resources[resolvedVolumeSource]; !ok {
					return nil, fmt.Errorf("containers.%s.volumes[%d].source: resource '%s' does not exist", containerName, idx, resolvedVolumeSource)
				} else if res.Type != "volume" {
					return nil, fmt.Errorf("containers.%s.volumes[%d].source: resource '%s' is not a volume", containerName, idx, resolvedVolumeSource)
				}

				if outputFunc, ok := resources[resolvedVolumeSource]; ok {
					if v, err := outputFunc("source"); err != nil {
						slog.Warn(fmt.Sprintf("containers.%s.volumes[%d].source: failed to find 'source' key in volume resource '%s': %v", containerName, idx, resolvedVolumeSource, err))
					} else if sv, ok := v.(string); ok {
						resolvedVolumeSource = sv
					}
				}

				volumes[idx] = compose.ServiceVolumeConfig{
					Type:     "volume",
					Source:   resolvedVolumeSource,
					Target:   vol.Target,
					ReadOnly: util.DerefOr(vol.ReadOnly, false),
				}
			}
		}
		// NOTE: Sorting is necessary for DeepEqual call within our Unit Tests to work reliably
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Source < volumes[j].Source
		})
		// END (NOTE)

		// Files are not supported just yet
		if len(cSpec.Files) > 0 {
			return nil, fmt.Errorf("containers.%s.files: not supported", containerName)
		}

		// Docker compose without swarm/stack mode doesn't really support resource limits. There are optimistic
		// workarounds but they vary between specific versions of the CLI. Better to just ignore.
		if cSpec.Resources != nil {
			if cSpec.Resources.Requests != nil {
				slog.Warn(fmt.Sprintf("containers.%s.resources.requests: not supported - ignoring", containerName))
			}
			if cSpec.Resources.Limits != nil {
				slog.Warn(fmt.Sprintf("containers.%s.resources.limits: not supported - ignoring", containerName))
			}
		}

		if cSpec.ReadinessProbe != nil {
			slog.Warn(fmt.Sprintf("containers.%s.readinessProbe: not supported - ignoring", containerName))
		}
		if cSpec.LivenessProbe != nil {
			slog.Warn(fmt.Sprintf("containers.%s.livenessProbe: not supported - ignoring", containerName))
		}

		var svc = compose.ServiceConfig{
			Name:        workloadName + "-" + containerName,
			Image:       cSpec.Image,
			Entrypoint:  cSpec.Command,
			Command:     cSpec.Args,
			Environment: env,
			Ports:       ports,
			Volumes:     volumes,
		}

		if bc, ok := containerBuildConfigs[containerName]; ok {
			slog.Info(fmt.Sprintf("containers.%s: overriding container build config to context=%s", containerName, bc.Context))
			svc.Build = &bc
			svc.Image = ""
		}

		// if we are not the "first" service, then inherit the network from the first service
		if firstService == "" {
			firstService = svc.Name
		} else {
			svc.Ports = nil
			svc.NetworkMode = "service:" + firstService
		}
		project.Services[svc.Name] = svc
	}
	return &project, nil
}
