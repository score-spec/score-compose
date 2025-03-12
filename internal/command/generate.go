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
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"

	"dario.cat/mergo"
	composeloader "github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	"github.com/score-spec/score-go/loader"
	"github.com/score-spec/score-go/schema"
	score "github.com/score-spec/score-go/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/compose"
	"github.com/score-spec/score-compose/internal/patching"
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
	generateCmdOutputFlag           = "output"
	generateCmdEnvFileFlag          = "env-file"
	generateCmdPublishFlag          = "publish"
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
  score-compose generate score.yaml --override-file=./overrides.score.yaml --override-property=metadata.key=value

  # Publish a port exposed by a workload for local testing
  score-compose generate score.yaml --publish 8080:my-workload:80

  # Publish a port from a resource host and port for local testing, the middle expression is RESOURCE_ID.OUTPUT_KEY
  score-compose generate score.yaml --publish 5432:postgres#my-workload.db.host:5432`,

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

		// Forbid --image and --build when multiple score files are provided
		if v, _ := cmd.Flags().GetString(generateCmdImageFlag); v != "" && len(workloadNames) > 1 {
			return fmt.Errorf("--%s cannot be used when multiple score files are provided", generateCmdImageFlag)
		}
		if v, _ := cmd.Flags().GetStringArray(generateCmdBuildFlag); len(v) > 0 && len(workloadNames) > 1 {
			return fmt.Errorf("--%s cannot be used when multiple score files are provided", generateCmdBuildFlag)
		}

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
				if workloadSpecs[workloadNames[0]], err = parseAndApplyOverrideProperty(overridePropertyEntry, generateCmdOverridePropertyFlag, workloadSpecs[workloadNames[0]]); err != nil {
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
		slog.Info(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.State.Extras.ComposeProjectName))

		currentState := &sd.State

		// Now validate with score spec
		for i, workloadName := range workloadNames {
			spec := workloadSpecs[workloadName]
			inputFile := inputFiles[i]

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
						var intermediate interface{}
						if err := yaml.Unmarshal([]byte(parts[1]), &intermediate); err != nil {
							return fmt.Errorf("invalid --%s '%s': %w", generateCmdBuildFlag, buildFlag, err)
						}
						var out types.BuildConfig
						if err := composeloader.Transform(intermediate, &out); err != nil {
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
						slog.Info(fmt.Sprintf("Set container image for workload '%s' container '%s' to %s from --%s", workloadName, containerName, v, generateCmdImageFlag))
						out.Containers[containerName] = container
					} else if _, ok := containerBuildContexts[containerName]; !ok {
						return fmt.Errorf("failed to convert '%s' because container '%s' has no image and neither --%s nor --%s was provided", workloadName, containerName, generateCmdImageFlag, generateCmdBuildFlag)
					}
				}
			}

			currentState, err = currentState.WithWorkload(&out, &inputFile, project.WorkloadExtras{BuildConfigs: containerBuildContexts})
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
			Name:     sd.State.Extras.ComposeProjectName,
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
					service.DependsOn[waitServiceName] = types.ServiceDependency{Condition: "service_completed_successfully", Required: true}
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

		for i, content := range currentState.Extras.PatchingTemplates {
			slog.Info(fmt.Sprintf("Applying patching template %d", i+1))
			superProject, err = patching.PatchServices(superProject, content)
			if err != nil {
				return fmt.Errorf("failed to patch template %d: %w", i+1, err)
			}
		}

		if v, err := cmd.Flags().GetStringArray(generateCmdPublishFlag); err != nil {
			return fmt.Errorf("failed to read publish property: %w", err)
		} else {
			for i, k := range v {
				parts := strings.Split(k, ":")
				if len(parts) <= 2 {
					return fmt.Errorf("--%s[%d] expected 3 :-separated parts", generateCmdPublishFlag, i)
				}
				// raw host port
				rhp := parts[0]
				// raw ref
				rr := strings.Join(parts[1:len(parts)-1], ":")
				// raw container port
				rcp := parts[len(parts)-1]

				hp, err := strconv.Atoi(rhp)
				if err != nil {
					return fmt.Errorf("--%s[%d] could not parse host port '%s' as integer", generateCmdPublishFlag, i, rhp)
				} else if hp <= 1 {
					return fmt.Errorf("--%s[%d] host port must be > 1", generateCmdPublishFlag, i)
				}

				cp, err := strconv.Atoi(rcp)
				if err != nil {
					return fmt.Errorf("--%s[%d] could not parse container port '%s' as integer", generateCmdPublishFlag, i, rcp)
				} else if cp <= 1 {
					return fmt.Errorf("--%s[%d] container port must be > 1", generateCmdPublishFlag, i)
				}

				if strings.Contains(rr, "#") {
					parts := strings.Split(rr, ".")
					if len(parts) < 2 {
						return fmt.Errorf("--%s[%d] must match RES_UID.OUTPUT", generateCmdPublishFlag, i)
					}
					rr = strings.Join(parts[0:len(parts)-1], ".")
					outputKey := parts[len(parts)-1]

					resUid := parseResourceUid(rr)
					res, ok := currentState.Resources[resUid]
					if !ok {
						return fmt.Errorf("--%s[%d] failed to find a resource with uid '%s'", generateCmdPublishFlag, i, rr)
					}

					if v, ok := res.Outputs[outputKey]; !ok {
						return fmt.Errorf("--%s[%d] resource '%s' has no output '%s'", generateCmdPublishFlag, i, resUid, outputKey)
					} else if sv, ok := v.(string); !ok {
						return fmt.Errorf("--%s[%d] resource '%s' output '%s' is not a string", generateCmdPublishFlag, i, resUid, outputKey)
					} else if config, ok := superProject.Services[sv]; !ok {
						return fmt.Errorf("--%s[%d] host '%s' does not exist", generateCmdPublishFlag, i, sv)
					} else {
						config.Ports = append(config.Ports, types.ServicePortConfig{
							Published: strconv.Itoa(hp),
							Target:    uint32(cp),
						})
						superProject.Services[sv] = config
						slog.Info(fmt.Sprintf("Published port %d of service '%s' to host port %d", cp, sv, hp))
					}
				} else if _, ok := currentState.Workloads[rr]; !ok {
					return fmt.Errorf("--%s[%d] failed to find a workload named '%s'", generateCmdPublishFlag, i, rr)
				} else {
					for sv, config := range superProject.Services {
						if config.Hostname == rr {
							config.Ports = append(config.Ports, types.ServicePortConfig{
								Published: strconv.Itoa(hp),
								Target:    uint32(cp),
							})
							superProject.Services[sv] = config
							slog.Info(fmt.Sprintf("Published port %d of service '%s' to host port %d", cp, sv, hp))
						}
					}
				}
			}
		}

		sd.State = *currentState
		if err := sd.Persist(); err != nil {
			return fmt.Errorf("failed to persist updated state directory: %w", err)
		}

		raw, _ := yaml.Marshal(superProject)

		v, _ := cmd.Flags().GetString(generateCmdOutputFlag)
		if v == "" {
			return fmt.Errorf("no output file specified")
		} else if v == "-" {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), string(raw))
		} else if err := os.WriteFile(v+".temp", raw, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		} else if err := os.Rename(v+".temp", v); err != nil {
			return fmt.Errorf("failed to complete writing output file: %w", err)
		}

		if v, _ := cmd.Flags().GetString(generateCmdEnvFileFlag); v != "" {
			content := new(strings.Builder)
			for k := range environmentProvisioner.Accessed() {
				_, _ = content.WriteString(k)
				_, _ = content.WriteRune('=')
				_, _ = content.WriteRune('\n')
			}
			slog.Info(fmt.Sprintf("Writing env var file to '%s'", v))
			if err := os.WriteFile(v, []byte(content.String()), 0644); err != nil {
				return fmt.Errorf("failed to write env var file: %w", err)
			}
		}
		return nil
	},
}

func parseResourceUid(raw string) framework.ResourceUid {
	parts := strings.SplitN(raw, "#", 2)
	firstParts := strings.SplitN(parts[0], ".", 2)
	secondParts := strings.SplitN(parts[1], ".", 2)
	resType := firstParts[0]
	var resClass *string
	if len(firstParts) > 1 {
		resClass = &firstParts[1]
	}
	if len(secondParts) == 1 {
		return framework.NewResourceUid("", "", resType, resClass, &parts[1])
	}
	return framework.NewResourceUid(secondParts[0], secondParts[1], resType, resClass, nil)
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
	generateCommand.Flags().StringP(generateCmdOutputFlag, "o", "compose.yaml", "The output file to write the composed compose file to")
	generateCommand.Flags().String(generateCmdOverridesFileFlag, "", "An optional file of Score overrides to merge in")
	generateCommand.Flags().StringArray(generateCmdOverridePropertyFlag, []string{}, "An optional set of path=key overrides to set or remove")
	generateCommand.Flags().String(generateCmdImageFlag, "", "An optional container image to use for any container with image == '.'")
	generateCommand.Flags().StringArray(generateCmdBuildFlag, []string{}, "An optional build context to use for the given container --build=container=./dir or --build=container={\"context\":\"./dir\"}")
	generateCommand.Flags().String(generateCmdEnvFileFlag, "", "Location to store a skeleton .env file for compose - this will override existing content")
	generateCommand.Flags().StringArray(generateCmdPublishFlag, []string{}, "An optional set of HOST_PORT:<ref>:CONTAINER_PORT to publish on the host system.")
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

func parseAndApplyOverrideProperty(entry string, flagName string, spec map[string]interface{}) (map[string]interface{}, error) {
	parts := strings.SplitN(entry, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("--%s '%s' is invalid, expected a =-separated path and value", flagName, entry)
	}
	if parts[1] == "" {
		slog.Info(fmt.Sprintf("Overriding '%s' in workload", parts[0]))
		after, err := framework.OverridePathInMap(spec, framework.ParseDotPathParts(parts[0]), true, nil)
		if err != nil {
			return nil, fmt.Errorf("--%s '%s' could not be applied: %w", flagName, entry, err)
		}
		return after, nil
	} else {
		var value interface{}
		if err := yaml.Unmarshal([]byte(parts[1]), &value); err != nil {
			return nil, fmt.Errorf("--%s '%s' is invalid, failed to unmarshal value as json: %w", flagName, entry, err)
		}
		slog.Info(fmt.Sprintf("Overriding '%s' in workload", parts[0]))
		after, err := framework.OverridePathInMap(spec, framework.ParseDotPathParts(parts[0]), false, value)
		if err != nil {
			return nil, fmt.Errorf("--%s '%s' could not be applied: %w", flagName, entry, err)
		}
		return after, nil
	}
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
