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

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/imdario/mergo"
	"github.com/score-spec/score-go/loader"
	"github.com/score-spec/score-go/schema"
	score "github.com/score-spec/score-go/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/compose"
	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/envprov"
	provloader "github.com/score-spec/score-compose/internal/provisioners/loader"
)

const (
	generateCmdOverridesFileFlag    = "overrides-file"
	generateCmdOverridePropertyFlag = "override-property"
	generateCmdImageFlag            = "image"
	generateCmdBuildFlag            = "build"
)

var generateCommand = &cobra.Command{
	Use:   "generate",
	Args:  cobra.ArbitraryArgs,
	Short: "Convert one or more Score files into a Docker compose manifest",
	Long: `The generate command will convert Score files in the current Score compose project into a combined Docker compose
manifest. All resources and links between Workloads will be resolved and provisioned as required.

By default this command looks for score.yaml in the current directory, but can take explicit file names as positional
arguments.

"score-compose init" MUST be run first. An error will be thrown if the project directory is not present.
`,
	Example: `
  # Specify Score files
  score-compose generate score.yaml *.score.yaml

  # Regenerate without adding new score files
  score-compose generate

  # Provide overrides when one score file is provided
  score-compose generate score.yaml --override-file=./overrides.score.yaml --override-property=metadata.key=value`,

	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		inputFiles := args
		slices.Sort(inputFiles)
		slog.Debug("Input Score files", "files", inputFiles)

		// first load all the score files, parse them with a dummy yaml decoder to find the workload name, reject any
		// with invalid or duplicate names.
		workloadNames, workloadSpecs, err := loadRawScoreFiles(inputFiles)
		if err != nil {
			return err
		}
		slog.Debug("Input Workload names", "names", workloadNames)

		// Now read and apply any overrides files to the score files
		if v, _ := cmd.Flags().GetString(generateCmdOverridesFileFlag); v != "" {
			if len(workloadNames) == 0 {
				return fmt.Errorf("--%s cannot be used without providing a score file", generateCmdOverridesFileFlag)
			} else if len(workloadNames) > 1 {
				return fmt.Errorf("--%s cannot be used when multiple score files are provided", generateCmdOverridesFileFlag)
			}
			if err := parseAndApplyOverrideFile(v, generateCmdOverridesFileFlag, workloadSpecs[workloadNames[0]]); err != nil {
				return err
			}
		}

		// Now read, parse, and apply any override properties to the score files
		if v, _ := cmd.Flags().GetStringArray(generateCmdOverridePropertyFlag); len(v) > 0 {
			if len(workloadNames) == 0 {
				return fmt.Errorf("--%s cannot be used without providing a score file", generateCmdOverridesFileFlag)
			} else if len(workloadNames) > 1 {
				return fmt.Errorf("--%s cannot be used when multiple score files are provided", generateCmdOverridesFileFlag)
			}
			for _, overridePropertyEntry := range v {
				if err := parseAndApplyOverrideProperty(overridePropertyEntry, generateCmdOverridePropertyFlag, workloadSpecs[workloadNames[0]]); err != nil {
					return err
				}
			}
		}

		sd, ok, err := project.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if !ok {
			return fmt.Errorf("state directory does not exist, please run \"score-compose init\" first")
		}
		slog.Info(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.State.ComposeProjectName))

		currentState := &sd.State

		// Now validate with score spec
		for workloadName, spec := range workloadSpecs {
			// Ensure transforms are applied (be a good citizen)
			if changes, err := schema.ApplyCommonUpgradeTransforms(spec); err != nil {
				return fmt.Errorf("failed to upgrade spec: %w", err)
			} else if len(changes) > 0 {
				for _, change := range changes {
					slog.Info(fmt.Sprintf("Applying backwards compatible upgrade to '%s': %s", workloadName, change))
				}
			}
			if err := schema.Validate(spec); err != nil {
				return fmt.Errorf("validation errors in workload '%s': %w", workloadName, err)
			}
			slog.Info(fmt.Sprintf("Validated workload '%s'", workloadName))

			var out score.Workload
			if err := loader.MapSpec(&out, spec); err != nil {
				return fmt.Errorf("failed to convert '%s' to structure: %w", workloadName, err)
			}

			// Gather container build contexts, these will be stored and added to the generated compose output later
			containerBuildContexts := make(map[string]types.BuildConfig)
			if v, _ := cmd.Flags().GetStringArray(generateCmdBuildFlag); len(v) > 0 {
				for _, buildFlag := range v {
					parts := strings.SplitN(buildFlag, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid --%s '%s': expected 2 =-separated parts", generateCmdBuildFlag, buildFlag)
					} else if _, ok := out.Containers[parts[0]]; !ok {
						return fmt.Errorf("invalid --%s '%s': unknown container '%s'", generateCmdBuildFlag, buildFlag, parts[0])
					}
					if strings.HasPrefix(parts[1], "{") {
						var out types.BuildConfig
						dec := json.NewDecoder(strings.NewReader(parts[1]))
						dec.DisallowUnknownFields()
						if err := dec.Decode(&out); err != nil {
							return fmt.Errorf("invalid --%s '%s': %w", generateCmdBuildFlag, buildFlag, err)
						}
						containerBuildContexts[parts[0]] = out
					} else {
						containerBuildContexts[parts[0]] = types.BuildConfig{Context: parts[1]}
					}
				}
			}

			// Apply image if container image is .
			for containerName, container := range out.Containers {
				if container.Image == "." {
					if v, _ := cmd.Flags().GetString(generateCmdImageFlag); v != "" {
						container.Image = v
						slog.Info("Set container image for workload '%s' container '%s' to %s from --%s", workloadName, containerName, v, generateCmdImageFlag)
						out.Containers[containerName] = container
					} else if _, ok := containerBuildContexts[containerName]; !ok {
						return fmt.Errorf("failed to convert '%s' because container '%s' has no image and neither --%s nor --%s was provided", workloadName, containerName, generateCmdImageFlag, generateCmdBuildFlag)
					}
				}
			}

			currentState, err = currentState.WithWorkload(&out, nil, containerBuildContexts)
			if err != nil {
				return fmt.Errorf("failed to add workload '%s': %w", workloadName, err)
			}
		}

		if len(currentState.Workloads) == 0 {
			return fmt.Errorf("the project is empty, please provide a score file to generate from")
		}

		loadedProvisioners, err := provloader.LoadProvisionersFromDirectory(sd.Path, provloader.DefaultSuffix)
		if err != nil {
			return fmt.Errorf("failed to load provisioners: %w", err)
		} else if len(loadedProvisioners) > 0 {
			slog.Info(fmt.Sprintf("Successfully loaded %d resource provisioners", len(loadedProvisioners)))
		}

		// append the env var provisioner
		environmentProvisioner := new(envprov.Provisioner)
		loadedProvisioners = append(loadedProvisioners, environmentProvisioner)

		currentState, err = currentState.WithPrimedResources()
		if err != nil {
			return fmt.Errorf("failed to prime resources: %w", err)
		}

		superProject := &types.Project{
			Name:     sd.State.ComposeProjectName,
			Services: make(types.Services, 0),
			Volumes:  map[string]types.VolumeConfig{},
			Networks: map[string]types.NetworkConfig{},
		}

		currentState, err = provisioners.ProvisionResources(context.Background(), currentState, loadedProvisioners, superProject)
		if err != nil {
			return fmt.Errorf("failed to provision: %w", err)
		} else if len(currentState.Resources) > 0 {
			slog.Info(fmt.Sprintf("Provisioned %d resources", len(currentState.Resources)))
		}

		waitServiceName, hasWaitService := injectWaitService(superProject)

		for workloadName, workloadState := range currentState.Workloads {

			slog.Info(fmt.Sprintf("Converting workload '%s' to Docker compose", workloadName))
			converted, err := compose.ConvertSpec(currentState, &workloadState.Spec)
			if err != nil {
				return fmt.Errorf("failed to convert workload '%s' to Docker compose: %w", workloadName, err)
			}

			for serviceName, service := range converted.Services {
				if _, ok := superProject.Services[serviceName]; ok {
					return fmt.Errorf("failed to add converted workload '%s': duplicate service name '%s'", workloadName, serviceName)
				}
				if hasWaitService {
					if service.DependsOn == nil {
						service.DependsOn = make(types.DependsOnConfig)
					}
					service.DependsOn[waitServiceName] = types.ServiceDependency{Condition: "service_started"}
				}
				superProject.Services[serviceName] = service
			}
			for volumeName, volume := range converted.Volumes {
				if _, ok := superProject.Volumes[volumeName]; ok {
					return fmt.Errorf("failed to add converted workload '%s': duplicate volume name '%s'", workloadName, volumeName)
				}
				superProject.Volumes[volumeName] = volume
			}
			for networkName, network := range converted.Networks {
				if _, ok := superProject.Networks[networkName]; ok {
					return fmt.Errorf("failed to add converted workload '%s': duplicated network name '%s'", workloadName, networkName)
				}
				superProject.Networks[networkName] = network
			}
		}

		sd.State = *currentState
		if err := sd.Persist(); err != nil {
			return fmt.Errorf("failed to persist updated state directory: %w", err)
		}

		raw, _ := yaml.Marshal(superProject)

		v, _ := cmd.Flags().GetString("output")
		if v == "" {
			return fmt.Errorf("no output file specified")
		} else if v == "-" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(raw))
		} else if err := os.WriteFile(v+".temp", raw, 0755); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		} else if err := os.Rename(v+".temp", v); err != nil {
			return fmt.Errorf("failed to complete writing output file: %w", err)
		}
		return nil
	},
}

// loadRawScoreFiles loads raw score specs as yaml from the given files and finds all the workload names. It throws
// errors if it failed to read, load, or if names are duplicated.
func loadRawScoreFiles(fileNames []string) ([]string, map[string]map[string]interface{}, error) {
	workloadNames := make([]string, 0, len(fileNames))
	workloadToRawScore := make(map[string]map[string]interface{}, len(fileNames))

	for _, fileName := range fileNames {
		var out map[string]interface{}
		raw, err := os.ReadFile(fileName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read '%s': %w", fileName, err)
		} else if err := yaml.Unmarshal(raw, &out); err != nil {
			return nil, nil, fmt.Errorf("failed to decode '%s' as yaml: %w", fileName, err)
		}

		var workloadName string
		if meta, ok := out["metadata"].(map[string]interface{}); ok {
			workloadName, _ = meta["name"].(string)
			if _, ok := workloadToRawScore[workloadName]; ok {
				return nil, nil, fmt.Errorf("workload name '%s' in file '%s' is used more than once", workloadName, fileName)
			}
		}
		workloadNames = append(workloadNames, workloadName)
		workloadToRawScore[workloadName] = out
	}
	return workloadNames, workloadToRawScore, nil
}

func init() {
	generateCommand.Flags().StringP("output", "o", "compose.yaml", "The output file to write the composed compose file to")
	generateCommand.Flags().String(generateCmdOverridesFileFlag, "", "An optional file of Score overrides to merge in")
	generateCommand.Flags().StringArray(generateCmdOverridePropertyFlag, []string{}, "An optional set of path=key overrides to set or remove")
	generateCommand.Flags().String(generateCmdImageFlag, "", "An optional container image to use for any container with image == '.'")
	generateCommand.Flags().StringArray(generateCmdBuildFlag, []string{}, "An optional build context to use for the given container --build=container=./dir or --build=container={'\"context\":\"./dir\"}")
	rootCmd.AddCommand(generateCommand)
}

func parseAndApplyOverrideFile(entry string, flagName string, spec map[string]interface{}) error {
	if raw, err := os.ReadFile(entry); err != nil {
		return fmt.Errorf("--%s '%s' is invalid, failed to read file: %w", flagName, entry, err)
	} else {
		slog.Info(fmt.Sprintf("Applying overrides from %s to workload", entry))
		var out map[string]interface{}
		if err := yaml.Unmarshal(raw, &out); err != nil {
			return fmt.Errorf("--%s '%s' is invalid: failed to decode yaml: %w", flagName, entry, err)
		} else if err := mergo.Merge(&spec, out, mergo.WithOverride); err != nil {
			return fmt.Errorf("--%s '%s' failed to apply: %w", flagName, entry, err)
		}
	}
	return nil
}

func parseAndApplyOverrideProperty(entry string, flagName string, spec map[string]interface{}) error {
	parts := strings.SplitN(entry, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("--%s '%s' is invalid, expected a =-separated path and value", flagName, entry)
	}
	if parts[1] == "" {
		slog.Info(fmt.Sprintf("Overriding '%s' in workload", parts[0]))
		if err := writePathInStruct(spec, parseDotPathParts(parts[0]), true, nil); err != nil {
			return fmt.Errorf("--%s '%s' could not be applied: %w", flagName, entry, err)
		}
	} else {
		var value interface{}
		if err := yaml.Unmarshal([]byte(parts[1]), &value); err != nil {
			return fmt.Errorf("--%s '%s' is invalid, failed to unmarshal value as json: %w", flagName, entry, err)
		}
		slog.Info(fmt.Sprintf("Overriding '%s' in workload", parts[0]))
		if err := writePathInStruct(spec, parseDotPathParts(parts[0]), false, value); err != nil {
			return fmt.Errorf("--%s '%s' could not be applied: %w", flagName, entry, err)
		}
	}
	return nil
}

// injectWaitService injects a service into the compose project which waits for all other services to be started,
// healthy, or complete depending on their definition. The workload services may then wait for this.
// This will return an empty string and false if there are no applicable services.
func injectWaitService(p *types.Project) (string, bool) {
	if len(p.Services) == 0 {
		return "", false
	}
	newService := types.ServiceConfig{
		Name:      "wait-for-resources",
		Image:     "alpine",
		Command:   types.ShellCommand{"echo"},
		DependsOn: make(types.DependsOnConfig),
	}
	for otherServiceName, otherService := range p.Services {
		condition := "service_started"
		if otherService.HealthCheck != nil {
			condition = "service_healthy"
		} else if v := otherService.Labels["dev.score.compose.labels.is-init-container"]; v == "true" {
			// annoyingly we can't tell based on the definition whether a service is designed to stop or not,
			// so we'll use this label as a best effort indicator.
			condition = "service_completed_successfully"
		}
		newService.DependsOn[otherServiceName] = types.ServiceDependency{
			Condition: condition,
			Required:  true,
		}
	}
	if p.Services == nil {
		p.Services = make(types.Services)
	}
	p.Services[newService.Name] = newService
	return newService.Name, true
}
