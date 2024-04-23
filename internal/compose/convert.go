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
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

// ConvertSpec converts SCORE specification into docker-compose configuration.
func ConvertSpec(state *project.State, spec *score.Workload) (*compose.Project, error) {
	workloadName, ok := spec.Metadata["name"].(string)
	if !ok || len(workloadName) == 0 {
		return nil, errors.New("workload metadata is missing a name")
	}

	if len(spec.Containers) == 0 {
		return nil, errors.New("workload does not have any containers to convert into a compose service")
	}

	resourceOutputs, err := state.GetResourceOutputForWorkload(workloadName)
	if err != nil {
		return nil, err
	}

	substitutionFunction := framework.BuildSubstitutionFunction(spec.Metadata, resourceOutputs)
	immediateSubstitutionFunction := util.WrapImmediateSubstitutionFunction(substitutionFunction)
	deferredSubstitutionFunction := util.WrapDeferredSubstitutionFunction(substitutionFunction)

	var composeProject = compose.Project{
		Services: make(compose.Services),
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
			resolved, err := framework.SubstituteString(val, deferredSubstitutionFunction)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.variables.%s: %w", containerName, key, err)
			}
			env[key] = &resolved
		}

		var volumes []compose.ServiceVolumeConfig
		if len(cSpec.Volumes) > 0 {
			volumes = make([]compose.ServiceVolumeConfig, len(cSpec.Volumes))
			for idx, vol := range cSpec.Volumes {
				if vol.Path != nil && *vol.Path != "" {
					return nil, fmt.Errorf("containers.%s.volumes[%d].path: can't mount named volume with sub path '%s': not supported", containerName, idx, *vol.Path)
				}

				// The way volumes are linked to a resource is a bit of a special case. The goal is to confirm that the
				// resource exists and is a volume. We then extract the source property. This only applies to the pattern
				// of ${resources.my-volume} which can also be written as ${resources.my-volume.source}.
				resolvedVolumeSource, err := framework.SubstituteString(vol.Source, func(ref string) (string, error) {
					if parts := framework.SplitRefParts(ref); len(parts) == 2 && parts[0] == "resources" {
						resName := parts[1]
						if res, ok := spec.Resources[resName]; !ok {
							return "", fmt.Errorf("containers.%s.volumes[%d].source: resource '%s' does not exist", containerName, idx, resName)
						} else if res.Type != "volume" {
							return "", fmt.Errorf("containers.%s.volumes[%d].source: resource '%s' is not a volume", containerName, idx, resName)
						}
						ref += ".source"
					}
					return deferredSubstitutionFunction(ref)
				})
				if err != nil {
					return nil, fmt.Errorf("containers.%s.volumes[%d].source: %w", containerName, idx, err)
				}

				volumes[idx] = compose.ServiceVolumeConfig{
					Type:     "volume",
					Source:   resolvedVolumeSource,
					Target:   vol.Target,
					ReadOnly: util.DerefOr(vol.ReadOnly, false),
				}
			}
		}

		if len(cSpec.Files) > 0 {
			newVolumes, err := convertFilesIntoVolumes(state, workloadName, containerName, immediateSubstitutionFunction)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, newVolumes...)
		}

		// NOTE: Sorting is necessary for DeepEqual call within our Unit Tests to work reliably
		sort.Slice(volumes, func(i, j int) bool {
			return volumes[i].Source < volumes[j].Source
		})
		// END (NOTE)

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
			Volumes:     volumes,
		}

		if bc, ok := state.Workloads[workloadName].Extras.BuildConfigs[containerName]; ok {
			slog.Info(fmt.Sprintf("containers.%s: overriding container build config to context=%s", containerName, bc.Context))
			svc.Build = &bc
			svc.Image = ""
		}

		// if we are not the "first" service, then inherit the network from the first service
		if firstService == "" {
			firstService = svc.Name
			// We name the containers as (workload name)-(container name) but we want the name for the main network
			// interface for be (workload name). So we set the hostname itself. This means that workloads cannot have
			// the same name within the project. But that's already enforced elsewhere.
			svc.Hostname = workloadName
		} else {
			svc.Ports = nil
			svc.NetworkMode = "service:" + firstService
		}
		composeProject.Services[svc.Name] = svc
	}
	return &composeProject, nil
}

// convertFilesIntoVolumes converts the lists of files into a list of bind mounts in the mounts directory.
func convertFilesIntoVolumes(state *project.State, workloadName string, containerName string, substitutionFunction func(string) (string, error)) ([]compose.ServiceVolumeConfig, error) {
	input := state.Workloads[workloadName].Spec.Containers[containerName].Files
	mountsDirectory := state.Extras.MountsDirectory
	if mountsDirectory == "" || mountsDirectory == "/dev/null" {
		return nil, fmt.Errorf("files are not supported")
	}

	output := make([]compose.ServiceVolumeConfig, 0, len(input))
	var err error

	filesDir := filepath.Join(mountsDirectory, "files")
	if err = os.MkdirAll(filesDir, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("failed to ensure the files directory exists")
	}
	for idx, file := range input {
		var content []byte
		if file.Content != nil {
			content = []byte(*file.Content)
		} else if file.Source != nil {
			sourcePath := *file.Source
			if !filepath.IsAbs(sourcePath) && state.Workloads[workloadName].File != nil {
				sourcePath = filepath.Join(filepath.Dir(*state.Workloads[workloadName].File), sourcePath)
			}
			content, err = os.ReadFile(sourcePath)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%d].source: failed to read: %w", containerName, idx, err)
			}
		} else {
			return nil, fmt.Errorf("containers.%s.files[%d]: missing 'content' or 'source'", containerName, idx)
		}
		if file.NoExpand == nil || !*file.NoExpand {
			stringContent, err := framework.SubstituteString(string(content), substitutionFunction)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%d]: failed to substitute in content: %w", containerName, idx, err)
			}
			content = []byte(stringContent)
		}
		newName := fmt.Sprintf("%s-files-%d-%s", workloadName, idx, strings.Trim(filepath.Base(file.Target), string(filepath.Separator)))
		slog.Debug(fmt.Sprintf("Writing %d bytes of content for %s containers.%s.files[%d] to %s", len(content), workloadName, containerName, idx, filepath.Join(filesDir, newName)))

		// Parse and correct the file mode of the mount. If the user permissions do not allow write, then we enable the read only flag
		// on the bind mount so that we can still remove the file from disk on the outside without sudo.
		readOnly := false
		fileMode := os.FileMode(0644)
		if file.Mode != nil {
			newMode, err := strconv.ParseInt(*file.Mode, 8, 32)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%d]: failed to parse '%s' as octal", containerName, idx, *file.Mode)
			} else if newMode > 0777 {
				return nil, fmt.Errorf("containers.%s.files[%d]: mode must be <= 0777", containerName, idx)
			} else if newMode&0400 != 0400 {
				return nil, fmt.Errorf("containers.%s.files[%d]: mode must be at least 0400", containerName, idx)
			} else if newMode&0600 != 0600 {
				newMode = newMode | 0600
				readOnly = true
			}
			fileMode = os.FileMode(newMode)
		}

		if err := os.WriteFile(filepath.Join(filesDir, newName), content, fileMode); err != nil {
			return nil, fmt.Errorf("containers.%s.files[%d]: failed to write to disk: %w", containerName, idx, err)
		}

		output = append(output, compose.ServiceVolumeConfig{
			Type:     "bind",
			Source:   filepath.Join(filesDir, newName),
			Target:   file.Target,
			ReadOnly: readOnly,
		})
	}

	return output, nil
}
