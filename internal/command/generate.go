package command

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/score-spec/score-go/loader"
	"github.com/score-spec/score-go/schema"
	score "github.com/score-spec/score-go/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/compose"
	"github.com/score-spec/score-compose/internal/project"
)

const (
	generateCmdFallbackFlag      = "fallback-to-env-var-resource-types"
	generateCmdOverridesFileFlag = "overrides-file"
	generateCmdOverridePathFlag  = "override-path"
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
  # Use default values
  score-compose generate

  # Specify Score files
  score-compose generate score.yaml *.score.yaml`,

	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		sd, ok, err := project.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if !ok {
			return fmt.Errorf("state directory does not exist, please run \"score-compose init\" first")
		}
		slog.Info(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.Config.ComposeProjectName))

		// find the input score files
		inputFiles := []string{scoreFileDefault}
		if len(args) > 0 {
			inputFiles = args
		}
		slices.Sort(inputFiles)
		slog.Debug("Input Score files", "files", inputFiles)

		// first load all the score files, parse them with a dummy yaml decoder to find the workload name, reject any
		// with invalid or duplicate names.
		workloadNames, workloadSpecs, err := loadRawScoreFiles(inputFiles)
		if err != nil {
			return err
		}
		slog.Debug("Input Workload names", "names", workloadNames)
		if len(workloadNames) == 0 {
			return fmt.Errorf("at least one Score file must be provided")
		}

		// Now read and apply any overrides files to the score files
		if v, _ := cmd.Flags().GetStringArray(generateCmdOverridesFileFlag); len(v) > 0 {
			for _, overrideFileEntry := range v {
				parts := strings.SplitN(overrideFileEntry, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("--%s '%s' is invalid, expected a =-separated name and file", generateCmdOverridesFileFlag, overrideFileEntry)
				} else if spec, ok := workloadSpecs[parts[0]]; !ok {
					return fmt.Errorf("--%s '%s' is invalid, unknown workload '%s'", generateCmdOverridesFileFlag, overrideFileEntry, parts[0])
				} else if raw, err := os.ReadFile(parts[1]); err != nil {
					return fmt.Errorf("--%s '%s' is invalid, failed to read file: %w", generateCmdOverridesFileFlag, overrideFileEntry, err)
				} else {
					slog.Info(fmt.Sprintf("Applying overrides from %s to workload %s", parts[1], parts[0]))
					var out map[string]interface{}
					if err := yaml.Unmarshal(raw, &out); err != nil {
						return fmt.Errorf("--%s '%s' is invalid: failed to decode yaml: %w", generateCmdOverridesFileFlag, overrideFileEntry, err)
					} else if err := mergo.Merge(spec, out, mergo.WithOverride); err != nil {
						return fmt.Errorf("--%s '%s' failed to apply: %w", generateCmdOverridesFileFlag, overrideFileEntry, err)
					}
				}
			}
		}

		// Now read, parse, and apply any override properties to the score files
		if v, _ := cmd.Flags().GetStringArray(generateCmdOverridePathFlag); len(v) > 0 {
			for _, overridePropertyEntry := range v {
				parts := strings.SplitN(overridePropertyEntry, "=", 3)
				if len(parts) != 3 {
					return fmt.Errorf("--%s '%s' is invalid, expected a =-separated name, path, and value", generateCmdOverridePathFlag, overridePropertyEntry)
				} else if spec, ok := workloadSpecs[parts[0]]; !ok {
					return fmt.Errorf("--%s '%s' is invalid, unknown workload '%s'", generateCmdOverridePathFlag, overridePropertyEntry, parts[0])
				} else if parts[2] == "" {
					slog.Info(fmt.Sprintf("Overriding '%s' in workload '%s'", parts[1], parts[0]))
					if err := writePathInStruct(spec, parseDotPathParts(parts[1]), true, nil); err != nil {
						return fmt.Errorf("--%s '%s' could not be applied: %w", generateCmdOverridePathFlag, overridePropertyEntry, err)
					}
				} else {
					var value interface{}
					if err := json.Unmarshal([]byte(parts[2]), &value); err != nil {
						return fmt.Errorf("--%s '%s' is invalid, failed to unmarshal value as json: %w", generateCmdOverridesFileFlag, overridePropertyEntry, err)
					}
					slog.Info(fmt.Sprintf("Overriding '%s' in workload '%s'", parts[1], parts[0]))
					if err := writePathInStruct(spec, parseDotPathParts(parts[1]), false, value); err != nil {
						return fmt.Errorf("--%s '%s' could not be applied: %w", generateCmdOverridePathFlag, overridePropertyEntry, err)
					}
				}
			}
		}

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
		}

		var fallbackTypes []string
		if v, _ := cmd.Flags().GetString(generateCmdFallbackFlag); v != "" {
			fallbackTypes = strings.Split(v, ",")
		}

		superProject := types.Project{
			Name:     sd.Config.ComposeProjectName,
			Services: make(types.Services, 0),
			Volumes:  map[string]types.VolumeConfig{},
			Networks: map[string]types.NetworkConfig{},
		}

		// Now convert them in order
		for _, workloadName := range workloadNames {
			spec := workloadSpecs[workloadName]
			var out score.Workload
			if err := loader.MapSpec(&out, spec); err != nil {
				return fmt.Errorf("failed to convert '%s' to structure: %w", workloadName, err)
			}

			resources, err := compose.GenerateResourceOutputs(&out, new(compose.EnvVarTracker), fallbackTypes)
			if err != nil {
				return fmt.Errorf("failed to convert workload '%s': %s", workloadName, err)
			}

			slog.Info(fmt.Sprintf("Converting workload '%s' to Docker compose", workloadName))
			converted, err := compose.ConvertSpec(&out, resources)
			if err != nil {
				return fmt.Errorf("failed to convert workload '%s' to Docker compose: %w", workloadName, err)
			}

			for _, service := range converted.Services {
				if slices.ContainsFunc(superProject.Services, func(config types.ServiceConfig) bool {
					return config.Name == service.Name
				}) {
					return fmt.Errorf("failed to add converted workload '%s': duplicate service name '%s'", workloadName, service.Name)
				}
				superProject.Services = append(superProject.Services, service)
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

		if meta, ok := out["metadata"].(map[string]interface{}); ok {
			if name, ok := meta["name"].(string); ok && name != "" {
				if _, ok := workloadToRawScore[name]; ok {
					return nil, nil, fmt.Errorf("workload name '%s' in file '%s' is used more than once", name, fileName)
				}
				workloadNames = append(workloadNames, name)
				workloadToRawScore[name] = out
				continue
			}
		}
		return nil, nil, fmt.Errorf("failed to find metadata.name in file '%s' please set a workload name", fileName)
	}
	return workloadNames, workloadToRawScore, nil
}

func init() {
	// TODO: it would be nice to have some better help rendering here, but we'd need to determine the terminal width
	//		 and use a small library to wrap the lines with some indentation.

	generateCommand.Flags().StringP(
		"output", "o", "compose.yaml",
		"The output file to write the composed compose file to",
	)

	generateCommand.Flags().StringArray(
		generateCmdOverridesFileFlag, []string{},
		"This option can be used one or more times to specify a Score overrides file per workload which will be merged with the "+
			"existing Score specification. For example --overrides-file=hello-world=./service.overrides.score.yaml "+
			"would merge the contents of the file with the Score workload with the name \"hello-world\". An error will "+
			"be thrown if a workload with the name does not exist.",
	)
	generateCommand.Flags().StringArray(
		generateCmdOverridePathFlag, []string{},
		"This option can be used one or more times to specify paths in the Score workloads to set or remove. Each "+
			"argument should provide the workload name, a .-separated path, and a value to set at the path, for example "+
			"--override-path=hello-world=some.key.subkey=value. The value may be json-encoded to indicate non-string "+
			"types. If the value is empty, the value at the given path will be removed.",
	)
	generateCommand.Flags().String(
		generateCmdFallbackFlag, "",
		"In previous versions of score-compose, resource references were resolved to environment variables. "+
			"This option can be used to specify which resource types can fallback to this behavior. By default none "+
			"can, but this option can be set to \"*\" or a comma-seperated list fo types.",
	)

	rootCmd.AddCommand(generateCommand)
}
