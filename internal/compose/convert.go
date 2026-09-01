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

package compose

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"maps"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	score "github.com/score-spec/score-go/types"
	"gopkg.in/yaml.v3"

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
	// name appended as a suffix. We use the natural sort order of the container names and pick the first one that
	// is not expected to exit (i.e. does not have all before entries with ready: complete).
	containerNames := make([]string, 0, len(spec.Containers))
	for name := range spec.Containers {
		containerNames = append(containerNames, name)
	}
	sort.Strings(containerNames)

	// Determine which container should own the network namespace.
	// By definition (enforced by validation), there must be at least one container with no
	// 'before' entries — otherwise the before relationships would form a cycle.
	primaryContainer := containerNames[0]
	for _, name := range containerNames {
		if len(spec.Containers[name].Before) == 0 {
			primaryContainer = name
			break
		}
	}

	variablesSubstitutor := framework.Substituter{
		Replacer: deferredSubstitutionFunction,
		UnEscaper: func(s string) (string, error) {
			return s, nil
		},
	}

	for _, containerName := range containerNames {
		cSpec := spec.Containers[containerName]

		var env = make(compose.MappingWithEquals, len(cSpec.Variables))
		for key, val := range cSpec.Variables {
			resolved, err := variablesSubstitutor.SubstituteString(val)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.variables.%s: %w", containerName, key, err)
			}
			env[key] = &resolved
		}

		// replace dollar sign ($) by double dollar sign ($$) in command strings
		if len(cSpec.Command) > 0 {
			cSpec.Command = util.PrepareEnvVariables(cSpec.Command)
		}

		// replace dollar sign ($) by double dollar sign ($$) in command arguments
		if len(cSpec.Args) > 0 {
			cSpec.Args = util.PrepareEnvVariables(cSpec.Args)
		}

		var volumes []compose.ServiceVolumeConfig
		if len(cSpec.Volumes) > 0 {
			volumes = make([]compose.ServiceVolumeConfig, 0, len(cSpec.Volumes))
			for _, target := range slices.Sorted(maps.Keys(cSpec.Volumes)) {
			    vol := cSpec.Volumes[target]
				cfg, err := convertVolumeSourceIntoVolume(state, deferredSubstitutionFunction, workloadName, target, vol)
				if err != nil {
					return nil, fmt.Errorf("containers.%s.volumes[%s]: %w", containerName, target, err)
				}
				volumes = append(volumes, *cfg)
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

		var svc = compose.ServiceConfig{
			Name:        workloadName + "-" + containerName,
			Annotations: buildWorkloadAnnotations(workloadName, spec),
			Image:       cSpec.Image,
			Entrypoint:  cSpec.Command,
			Command:     cSpec.Args,
			Environment: env,
			Volumes:     volumes,
		}

		if cSpec.ReadinessProbe != nil {
			if hc, err := convertProbeToExec(cSpec.ReadinessProbe); err != nil {
				return nil, fmt.Errorf("containers.%s.readinessProbe: %w", containerName, err)
			} else if hc != nil {
				svc.HealthCheck = hc
			}
		} else if cSpec.LivenessProbe != nil {
			if hc, err := convertProbeToExec(cSpec.LivenessProbe); err != nil {
				return nil, fmt.Errorf("containers.%s.livenessProbe: %w", containerName, err)
			} else if hc != nil {
				svc.HealthCheck = hc
			}
		}

		if bc, ok := state.Workloads[workloadName].Extras.BuildConfigs[containerName]; ok {
			slog.Info(fmt.Sprintf("containers.%s: overriding container build config to context=%s", containerName, bc.Context))
			svc.Build = &bc
			svc.Image = ""
		}

		// if we are not the primary container, then inherit the network from the primary service.
		// However, skip network_mode for containers that are expected to exit (all their before
		// entries have ready: complete) to avoid circular dependencies.
		if containerName == primaryContainer {
			// We name the containers as (workload name)-(container name) but we want the name for the main network
			// interface for be (workload name). So we set the hostname itself. This means that workloads cannot have
			// the same name within the project. But that's already enforced elsewhere.
			svc.Hostname = workloadName
		} else if !isInitContainer(spec.Containers[containerName]) {
			svc.Ports = nil
			svc.NetworkMode = "service:" + workloadName + "-" + primaryContainer
		}
		composeProject.Services[svc.Name] = svc
	}

	// Invert before -> depends_on: if container A declares before: {B: {ready: complete}},
	// then service B depends_on A with the appropriate condition.
	for _, containerName := range containerNames {
		cSpec := spec.Containers[containerName]
		for targetContainerName, entry := range cSpec.Before {
			// Determine the compose condition from the ready field
			var condition string
			switch entry.Ready {
			case score.ContainerBeforeReadyComplete:
				condition = "service_completed_successfully"
			case score.ContainerBeforeReadyHealthy:
				condition = "service_healthy"
			case score.ContainerBeforeReadyStarted:
				condition = "service_started"
			default:
				return nil, fmt.Errorf("containers.%s.before.%s: unknown ready condition %q", containerName, targetContainerName, entry.Ready)
			}

			if entry.Ready == score.ContainerBeforeReadyHealthy && cSpec.ReadinessProbe == nil && cSpec.LivenessProbe == nil {
				return nil, fmt.Errorf("containers.%s.before: ready '%s' requires a readiness or liveness probe to be defined", containerName, score.ContainerBeforeReadyHealthy)
			}

			sourceServiceName := workloadName + "-" + containerName
			targetServiceName := workloadName + "-" + targetContainerName

			// Add depends_on to the target service
			svc := composeProject.Services[targetServiceName]
			if svc.DependsOn == nil {
				svc.DependsOn = make(compose.DependsOnConfig)
			}
			svc.DependsOn[sourceServiceName] = compose.ServiceDependency{
				Condition: condition,
				Required:  true,
			}
			composeProject.Services[targetServiceName] = svc
		}
	}

	return &composeProject, nil
}

// isInitContainer returns true if the container is expected to exit. This is determined
// by checking if all its before entries specify ready: complete.
func isInitContainer(c score.Container) bool {
	if len(c.Before) == 0 {
		return false
	}
	for _, entry := range c.Before {
		if entry.Ready != score.ContainerBeforeReadyComplete {
			return false
		}
	}
	return true
}

// buildWorkloadAnnotations returns an annotation set for the workload service.
func buildWorkloadAnnotations(name string, spec *score.Workload) map[string]string {
	var out map[string]string
	a, ok := spec.Metadata["annotations"].(map[string]interface{})
	if !ok {
		a, ok = spec.Metadata["annotations"].(score.WorkloadMetadata)
	}
	if ok {
		out = make(map[string]string, len(a))
		for k, v := range a {
			// type is validated by the spec
			out[k] = v.(string)
		}
	} else {
		out = make(map[string]string, 1)
	}
	out["compose.score.dev/workload-name"] = name
	return out
}

func convertProbeToExec(p *score.ContainerProbe) (*compose.HealthCheckConfig, error) {
	if p.Exec != nil {
		if len(p.Exec.Command) == 0 {
			return nil, fmt.Errorf("exec command is empty")
		}
		return &compose.HealthCheckConfig{
			Test:     append([]string{"CMD"}, p.Exec.Command...),
			Interval: util.Ref(compose.Duration(time.Second * 5)),
			Timeout:  util.Ref(compose.Duration(time.Second * 5)),
			Disable:  false,
		}, nil
	} else if p.HttpGet != nil {
		slog.Warn("httpGet container probe: not supported - ignoring")
		return nil, nil
	}
	return nil, fmt.Errorf("exec or httpGet must be specified")
}

// Annotation keys for opting into a custom pair of file-content placeholder delimiters.
// Both annotations must be set together. When set, Score expands placeholders bounded by
// the given start/end strings instead of the default "${" and "}". This is experimental and
// compose-only; if it works out in practice it may later be promoted to a spec property.
const (
	annotationPlaceholderStart = "compose.score.dev/experiment-placeholder-start"
	annotationPlaceholderEnd   = "compose.score.dev/experiment-placeholder-end"
)

// substituteWithDelimiters expands placeholders in src using a custom start/end pair instead of
// the default ${...} syntax. Used by the experimental compose.score.dev/experiment-placeholder-*
// annotations. There is no escape mechanism; if the chosen delimiters appear literally in the
// file they will be matched and expanded.
func substituteWithDelimiters(src, start, end string, replacer func(string) (string, error)) (string, error) {
	pattern := regexp.QuoteMeta(start) + "(.*?)" + regexp.QuoteMeta(end)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to build delimiter regex from start=%q end=%q: %w", start, end, err)
	}
	var subErr error
	result := re.ReplaceAllStringFunc(src, func(match string) string {
		groups := re.FindStringSubmatch(match)
		if len(groups) < 2 {
			return match
		}
		res, err := replacer(groups[1])
		if err != nil {
			subErr = errors.Join(subErr, err)
			return match
		}
		return res
	})
	return result, subErr
}

// readPlaceholderDelimiters extracts the experiment-placeholder-{start,end} annotations from a
// workload spec. It returns (start, end, true, nil) when both are set, ("", "", false, nil) when
// neither is set, and an error when exactly one is set.
func readPlaceholderDelimiters(spec score.Workload) (string, string, bool, error) {
	annotations, ok := spec.Metadata["annotations"].(map[string]interface{})
	if !ok {
		annotations, ok = spec.Metadata["annotations"].(score.WorkloadMetadata)
	}
	if !ok {
		return "", "", false, nil
	}
	startRaw, hasStart := annotations[annotationPlaceholderStart]
	endRaw, hasEnd := annotations[annotationPlaceholderEnd]
	if !hasStart && !hasEnd {
		return "", "", false, nil
	}
	if !hasStart || !hasEnd {
		return "", "", false, fmt.Errorf(
			"annotations %q and %q must be set together (got start=%t, end=%t)",
			annotationPlaceholderStart, annotationPlaceholderEnd, hasStart, hasEnd,
		)
	}
	start, ok := startRaw.(string)
	if !ok || start == "" {
		return "", "", false, fmt.Errorf("annotation %q must be a non-empty string", annotationPlaceholderStart)
	}
	end, ok := endRaw.(string)
	if !ok || end == "" {
		return "", "", false, fmt.Errorf("annotation %q must be a non-empty string", annotationPlaceholderEnd)
	}
	return start, end, true, nil
}

// convertFilesIntoVolumes converts the lists of files into a list of bind mounts in the mounts directory.
func convertFilesIntoVolumes(state *project.State, workloadName string, containerName string, substitutionFunction func(string) (string, error)) ([]compose.ServiceVolumeConfig, error) {
	spec := state.Workloads[workloadName].Spec
	input := spec.Containers[containerName].Files
	mountsDirectory := state.Extras.MountsDirectory
	if mountsDirectory == "" || mountsDirectory == "/dev/null" {
		return nil, fmt.Errorf("files are not supported")
	}

	// Check for the experimental custom-delimiter annotations on the workload. When set, file
	// content is expanded using those delimiters instead of the default ${...} syntax.
	// noExpand: true on a file still takes priority, and binaryContent is unaffected.
	customStart, customEnd, useCustomDelimiters, err := readPlaceholderDelimiters(spec)
	if err != nil {
		return nil, fmt.Errorf("workload %q: %w", workloadName, err)
	}

	output := make([]compose.ServiceVolumeConfig, 0, len(input))

	filesDir := filepath.Join(mountsDirectory, "files")
	if err = os.MkdirAll(filesDir, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("failed to ensure the files directory exists")
	}
	for _, target := range slices.Sorted(maps.Keys(input)) {
	    file := input[target]
		var content []byte
		if file.Content != nil {
			content = []byte(*file.Content)
		} else if file.Source != nil {
			sourcePath := *file.Source
			if !filepath.IsLocal(sourcePath) {
				return nil, fmt.Errorf("containers.%s.files[%s].source: must be a relative path within the Score file's directory", containerName, target)
			}
			if state.Workloads[workloadName].File != nil {
				sourcePath = filepath.Join(filepath.Dir(*state.Workloads[workloadName].File), sourcePath)
			}
			content, err = os.ReadFile(sourcePath)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%s].source: failed to read: %w", containerName, target, err)
			}
		} else if file.BinaryContent != nil {
			content, err = base64.StdEncoding.DecodeString(*file.BinaryContent)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%s].binaryContent: failed to decode base64: %w", containerName, target, err)
			}
		} else {
			return nil, fmt.Errorf("containers.%s.files[%s]: missing 'content', 'binaryContent', or 'source'", containerName, target)
		}
		if (file.NoExpand == nil || !*file.NoExpand) && utf8.Valid(content) && file.BinaryContent == nil {
			var stringContent string
			var subErr error
			if useCustomDelimiters {
				stringContent, subErr = substituteWithDelimiters(string(content), customStart, customEnd, substitutionFunction)
			} else {
				stringContent, subErr = framework.SubstituteString(string(content), substitutionFunction)
			}
			if subErr != nil {
				return nil, fmt.Errorf("containers.%s.files[%s]: failed to substitute in content: %w", containerName, target, subErr)
			}
			content = []byte(stringContent)
		}
		newName := fmt.Sprintf("%s-files-%s", workloadName, strings.Trim(filepath.Base(target), string(filepath.Separator)))
		slog.Debug(fmt.Sprintf("Writing %d bytes of content for %s containers.%s.files[%s] to %s", len(content), workloadName, containerName, target, filepath.Join(filesDir, newName)))

		// Parse and correct the file mode of the mount. If the user permissions do not allow write, then we enable the read only flag
		// on the bind mount so that we can still remove the file from disk on the outside without sudo.
		readOnly := false
		fileMode := os.FileMode(0644)
		if file.Mode != nil {
			newMode, err := strconv.ParseInt(*file.Mode, 8, 32)
			if err != nil {
				return nil, fmt.Errorf("containers.%s.files[%s]: failed to parse '%s' as octal", containerName, target, *file.Mode)
			} else if newMode > 0777 {
				return nil, fmt.Errorf("containers.%s.files[%s]: mode must be <= 0777", containerName, target)
			} else if newMode&0400 != 0400 {
				return nil, fmt.Errorf("containers.%s.files[%s]: mode must be at least 0400", containerName, target)
			} else if newMode&0600 != 0600 {
				newMode = newMode | 0600
				readOnly = true
			}
			fileMode = os.FileMode(newMode)
		}

		if err := os.WriteFile(filepath.Join(filesDir, newName), content, fileMode); err != nil {
			return nil, fmt.Errorf("containers.%s.files[%s]: failed to write to disk: %w", containerName, target, err)
		}

		output = append(output, compose.ServiceVolumeConfig{
			Type:     "bind",
			Source:   filepath.Join(filesDir, newName),
			Target:   target,
			ReadOnly: readOnly,
		})
	}

	return output, nil
}

func convertVolumeSourceIntoVolume(state *project.State, substitutionFunction func(string) (string, error), workloadName string, target string, vol score.ContainerVolume) (*compose.ServiceVolumeConfig, error) {
	spec := state.Workloads[workloadName].Spec

	// The way volumes are linked to a resource is a bit of a special case. The goal is to confirm that the
	// resource exists and has the outputs that we need.
	resolvedVolumeSource, err := framework.SubstituteString(vol.Source, func(ref string) (string, error) {
		if parts := framework.SplitRefParts(ref); len(parts) == 2 && parts[0] == "resources" {
			resName := parts[1]
			if res, ok := spec.Resources[resName]; ok {
				return string(framework.NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)), nil
			}
			return "", fmt.Errorf("resource '%s' does not exist", resName)
		}
		return substitutionFunction(ref)
	})
	if err != nil {
		return nil, err
	}

	outputVolume := &compose.ServiceVolumeConfig{
		Type:     "volume",
		Source:   resolvedVolumeSource,
		Target:   target,
		ReadOnly: util.DerefOr(vol.ReadOnly, false),
	}

	// now if the resolves source is a volume we can check the outputs or throw an error

	res, ok := state.Resources[framework.ResourceUid(resolvedVolumeSource)]
	if ok {
		volType, ok := res.Outputs["type"].(string)
		if !ok {
			return nil, fmt.Errorf("resource '%s' has no 'type' output", resolvedVolumeSource)
		}
		outputVolume.Type = volType
		raw, _ := json.Marshal(res.Outputs)
		dec := yaml.NewDecoder(bytes.NewReader(raw))
		dec.KnownFields(true)
		switch volType {
		case "volume":
			s := struct {
				Type   string                       `json:"type"`
				Source string                       `json:"source"`
				Volume *compose.ServiceVolumeVolume `json:"volume"`
			}{}
			if err := dec.Decode(&s); err != nil {
				return nil, fmt.Errorf("resource '%s' outputs cannot decode for volume: %w", resolvedVolumeSource, err)
			}
			outputVolume.Source = s.Source
			outputVolume.Volume = s.Volume
			if vol.Path != nil && *vol.Path != "" {
				if outputVolume.Volume == nil {
					outputVolume.Volume = &compose.ServiceVolumeVolume{}
				}
				outputVolume.Volume.Subpath = filepath.Join(outputVolume.Volume.Subpath, *vol.Path)
			}
		case "tmpfs":
			s := struct {
				Type  string                      `json:"type"`
				Tmpfs *compose.ServiceVolumeTmpfs `json:"tmpfs"`
			}{}
			if err := dec.Decode(&s); err != nil {
				return nil, fmt.Errorf("resource '%s' outputs cannot decode for tmpfs: %w", resolvedVolumeSource, err)
			}
			outputVolume.Tmpfs = s.Tmpfs
			if vol.Path != nil && *vol.Path != "" {
				return nil, fmt.Errorf("can't mount named tmpfs volume with sub path '%s': not supported", *vol.Path)
			}
		case "bind":
			s := struct {
				Type   string                     `json:"type"`
				Source string                     `json:"source"`
				Bind   *compose.ServiceVolumeBind `json:"bind"`
			}{}
			if err := dec.Decode(&s); err != nil {
				return nil, fmt.Errorf("resource '%s' outputs cannot decode for bind: %w", resolvedVolumeSource, err)
			}
			outputVolume.Source = s.Source
			if vol.Path != nil && *vol.Path != "" {
				outputVolume.Source = filepath.Join(outputVolume.Source, *vol.Path)
			}
			outputVolume.Bind = s.Bind
		default:
			return nil, fmt.Errorf("resource '%s' has invalid type '%s'", resolvedVolumeSource, volType)
		}
	}

	return outputVolume, nil
}
