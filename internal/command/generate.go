package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	"github.com/score-spec/score-compose/internal/project/envvarprovider"
	"github.com/score-spec/score-compose/internal/project/legacyvarprovider"
	"github.com/score-spec/score-compose/internal/ref"
)

const (
	generateCmdFallbackFlag         = "fallback-res-id-to-env-var"
	generateCmdOverridesFileFlag    = "overrides-file"
	generateCmdOverridePropertyFlag = "override-property"
)

var generateCommand = &cobra.Command{
	Use:   "generate",
	Args:  cobra.ArbitraryArgs,
	Short: "Add or update Score files in the current project and regenerate the Docker compose file",
	Long: `The generate command will add or update multiple Score workloads in the current Score compose project and
regenerate the output Docker compose file after all resources have been provisioned.

By default this command looks for score.yaml in the current directory, but can take explicit file names as positional
arguments as well as overrides to apply.

"score-compose init" MUST be run first. An error will be thrown if the project directory is not present.
`,
	Example: `
  # Use default values and import the default score.yaml file
  score-compose generate

  # Import multiple Score files if no additional options are required
  score-compose generate score.yaml *.score.yaml

  # Provide overrides when one score file is provided
  score-compose generate score.yaml --override-file=./overrides.score.yaml --override-property=metadata.key=value`,

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
		slog.Info(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.State.ComposeProjectName))

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
		if v, _ := cmd.Flags().GetString(generateCmdOverridesFileFlag); v != "" {
			if len(workloadSpecs) > 1 {
				return fmt.Errorf("--%s cannot be used when multiple Score files are imported", generateCmdOverridesFileFlag)
			}
			if err := parseAndApplyOverrideFile(v, generateCmdOverridesFileFlag, workloadSpecs[workloadNames[0]]); err != nil {
				return err
			}
		}

		// Now read, parse, and apply any override properties to the score files
		if v, _ := cmd.Flags().GetStringArray(generateCmdOverridePropertyFlag); len(v) > 0 {
			if len(workloadSpecs) > 1 {
				return fmt.Errorf("--%s cannot be used when multiple Score files are imported", generateCmdOverridePropertyFlag)
			}
			for _, overridePropertyEntry := range v {
				if err := parseAndApplyOverrideProperty(overridePropertyEntry, generateCmdOverridePropertyFlag, workloadSpecs[workloadNames[0]]); err != nil {
					return err
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

		workloads := make(map[string]score.Workload)
		// Now insert them into the state
		for i, workloadName := range workloadNames {
			var out score.Workload
			if err := loader.MapSpec(&out, workloadSpecs[workloadName]); err != nil {
				return fmt.Errorf("failed to convert '%s' to structure: %w", workloadName, err)
			}
			workloads[workloadName] = out
			absFilePath, err := filepath.Abs(inputFiles[i])
			if err != nil {
				return fmt.Errorf("failed to find absolute path: %w", err)
			}
			if err := sd.State.AppendWorkload(&out, ref.Ref(absFilePath)); err != nil {
				return fmt.Errorf("failed to add '%s' to state: %w", workloadName, err)
			}
		}

		providers := []project.ConfiguredResourceProvider{
			new(envvarprovider.Provider),
		}

		// TODO: load additional providers here

		if fallbackResIds, _ := cmd.Flags().GetString(generateCmdFallbackFlag); fallbackResIds != "" {
			parts := strings.Split(fallbackResIds, ",")
			for _, resId := range parts {
				if resId != "" {
					slog.Warn(fmt.Sprintf("Injecting legacy variable provider for resource id '%s'", resId))
					providers = append(providers, &legacyvarprovider.Provider{
						Id:     resId,
						Prefix: resId + "_",
					})
				}
			}
		}

		superProject := &types.Project{
			Name:     sd.State.ComposeProjectName,
			Services: make(types.Services),
			Volumes:  map[string]types.VolumeConfig{},
			Networks: map[string]types.NetworkConfig{},
		}

		if err := sd.State.ProvisionResources(context.Background(), providers, superProject); err != nil {
			return fmt.Errorf("failed to provision resources: %w", err)
		}

		// Now convert them in order
		for _, workloadName := range workloadNames {
			workload := workloads[workloadName]

			localResourceView, err := sd.State.CollectResourceOutputs(&workload)
			if err != nil {
				return fmt.Errorf("failed to setup local resources for '%s': %w", workloadName, err)
			}

			slog.Info(fmt.Sprintf("Converting workload '%s' to Docker compose", workloadName))
			converted, err := compose.ConvertSpec(&workload, localResourceView)
			if err != nil {
				return fmt.Errorf("failed to convert workload '%s' to Docker compose: %w", workloadName, err)
			}

			for serviceName, service := range converted.Services {
				if _, ok := superProject.Services[serviceName]; ok {
					return fmt.Errorf("failed to add converted workload '%s': duplicated service name '%s'", workloadName, serviceName)
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

		// finally persist the state
		if err := sd.Persist(); err != nil {
			return fmt.Errorf("faield to persist")
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

	generateCommand.Flags().StringP(
		"output", "o", "compose.yaml",
		"The output file to write the composed compose file to",
	)

	generateCommand.Flags().String(generateCmdOverridesFileFlag, "", "Merge Score overrides from another file")
	generateCommand.Flags().StringArray(generateCmdOverridePropertyFlag, []string{}, "Set or remove a path in the Score file, eg add.this.key=\"value\" or remove.this.key=")
	generateCommand.Flags().String(generateCmdFallbackFlag, "", "Enable fallback provider for the given ,-separated list of resource ids")

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
		slog.Info("Removing in workload")
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
