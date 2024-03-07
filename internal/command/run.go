/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/spf13/cobra"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"

	loader "github.com/score-spec/score-go/loader"
	schema "github.com/score-spec/score-go/schema"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/compose"
	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/project/envvarprovider"
	"github.com/score-spec/score-compose/internal/project/legacyvarprovider"
	"github.com/score-spec/score-compose/internal/project/staticprovider"
	"github.com/score-spec/score-compose/internal/ref"
)

const (
	scoreFileDefault     = "./score.yaml"
	overridesFileDefault = "./overrides.score.yaml"
)

var (
	scoreFile     string
	overridesFile string
	outFile       string
	envFile       string
	buildCtx      string

	overrideParams []string

	skipValidation bool
)

func init() {
	runCmd.Flags().StringVarP(&scoreFile, "file", "f", scoreFileDefault, "Source SCORE file")
	runCmd.Flags().StringVar(&overridesFile, "overrides", overridesFileDefault, "Overrides SCORE file")
	runCmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file")
	runCmd.Flags().StringVar(&envFile, "env-file", "", "Location to store sample .env file")
	runCmd.Flags().StringVar(&buildCtx, "build", "", "Replaces 'image' name with compose 'build' instruction")

	runCmd.Flags().StringArrayVarP(&overrideParams, "property", "p", nil, "Overrides selected property value")

	runCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "DEPRECATED: Disables Score file schema validation")

	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run [--file=score.yaml] [--output=compose.yaml]",
	Args:  cobra.NoArgs,
	Short: "Translate the SCORE file to docker-compose configuration",
	RunE:  run,
	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,
}

func run(cmd *cobra.Command, args []string) error {
	// Silence usage message if args are parsed correctly
	cmd.SilenceUsage = true

	// Open source file
	//
	slog.Info(fmt.Sprintf("Reading Score file '%s'", scoreFile))
	var err error
	var src *os.File
	if src, err = os.Open(scoreFile); err != nil {
		return err
	}
	defer src.Close()

	// Parse SCORE spec
	//
	slog.Info("Parsing Score specification")
	var srcMap map[string]interface{}
	if err = loader.ParseYAML(&srcMap, src); err != nil {
		return err
	}

	// Apply overrides from file (optional)
	//
	if overridesFile != "" {
		if ovr, err := os.Open(overridesFile); err == nil {
			defer ovr.Close()

			slog.Info(fmt.Sprintf("Loading Score overrides file '%s'", overridesFile))
			var ovrMap map[string]interface{}
			if err = loader.ParseYAML(&ovrMap, ovr); err != nil {
				return err
			}
			slog.Info("Applying Score overrides")
			if err := mergo.MergeWithOverwrite(&srcMap, ovrMap); err != nil {
				return fmt.Errorf("applying overrides fom '%s': %w", overridesFile, err)
			}
		} else if !os.IsNotExist(err) || overridesFile != overridesFileDefault {
			return err
		}
	}

	// Apply overrides from command line (optional)
	//
	for _, pstr := range overrideParams {
		jsonBytes, err := json.Marshal(srcMap)
		if err != nil {
			return fmt.Errorf("marshalling score spec: %w", err)
		}

		pmap := strings.SplitN(pstr, "=", 2)
		if len(pmap) <= 1 {
			var path = pmap[0]
			slog.Info(fmt.Sprintf("Applying Score properties override: removing '%s'", path))
			if jsonBytes, err = sjson.DeleteBytes(jsonBytes, path); err != nil {
				return fmt.Errorf("removing '%s': %w", path, err)
			}
		} else {
			var path = pmap[0]
			var val interface{}
			if err := yaml.Unmarshal([]byte(pmap[1]), &val); err != nil {
				val = pmap[1]
			}

			slog.Info(fmt.Sprintf("Applying Score properties override: overriding '%s' = '%s' (%T)", path, val, val))
			if jsonBytes, err = sjson.SetBytes(jsonBytes, path, val); err != nil {
				return fmt.Errorf("overriding '%s': %w", path, err)
			}
		}

		if err = json.Unmarshal(jsonBytes, &srcMap); err != nil {
			return fmt.Errorf("unmarshalling score spec: %w", err)
		}
	}

	// Apply upgrades to fix backports or backward incompatible things
	if changes, err := schema.ApplyCommonUpgradeTransforms(srcMap); err != nil {
		return fmt.Errorf("failed to upgrade spec: %w", err)
	} else if len(changes) > 0 {
		for _, change := range changes {
			slog.Info(fmt.Sprintf("Applying upgrade to specification: %s", change))
		}
	}

	// Validate SCORE spec
	//
	if !skipValidation {
		slog.Info("Validating final Score specification")
		if err := schema.Validate(srcMap); err != nil {
			return fmt.Errorf("validating workload spec: %w", err)
		}
	}

	// Convert SCORE spec
	//
	var spec score.Workload
	if err = loader.MapSpec(&spec, srcMap); err != nil {
		return fmt.Errorf("validating workload spec: %w", err)
	}

	// Build the legacy provides that are available for score-compose run to keep compatible behavior.
	//
	providers := make([]project.ConfiguredResourceProvider, 0)
	providers = append(providers, new(envvarprovider.Provider))
	providers = append(providers, &staticprovider.Provider{Type: "volume", Class: "default"})
	for resName, resource := range spec.Resources {
		if resource.Type != "environment" && resource.Type != "volume" {
			resClass, resId := project.GenerateResourceClassAndId(spec.Metadata["name"].(string), resName, &resource)
			slog.Warn(fmt.Sprintf("resources.%s: '%s.%s' is not directly supported in score-compose, references will be converted to environment variables", resName, resource.Type, resClass))
			providers = append(providers, &legacyvarprovider.Provider{
				Prefix: strings.ReplaceAll(strings.ToUpper(resName), "-", "_") + "_",
				Type:   resource.Type,
				Class:  resClass,
				Id:     resId,
			})
		}
	}

	// Initialise the score state
	state := new(project.State)
	_ = state.AppendWorkload(&spec, ref.Ref(scoreFile))

	// Provision resources and collect outputs. Note that the legacy drivers do not modify the score compose project
	// at all.
	if err := state.ProvisionResources(context.Background(), providers, nil); err != nil {
		return err
	}
	localResourceView, err := state.CollectResourceOutputs(&spec)
	if err != nil {
		return err
	}

	// Build docker-compose configuration
	//
	slog.Info("Building docker-compose configuration")
	proj, err := compose.ConvertSpec(&spec, localResourceView)
	if err != nil {
		return fmt.Errorf("building docker-compose configuration: %w", err)
	}

	// Override 'image' reference with 'build' instructions
	//
	if buildCtx != "" {
		slog.Info(fmt.Sprintf("Applying build context '%s' for service images", buildCtx))
		// We add the build context to all services and containers here and make a big assumption that all are
		// using the image we are building here and now. If this is unexpected, users should use a more complex
		// overrides file.
		for idx := range proj.Services {
			proj.Services[idx].Build = &types.BuildConfig{Context: buildCtx}
			proj.Services[idx].Image = ""
		}
	}

	// Open output file (optional)
	//
	var dest = cmd.OutOrStdout()
	if outFile != "" {
		slog.Info(fmt.Sprintf("Writing output compose file '%s'", outFile))
		destFile, err := os.Create(outFile)
		if err != nil {
			return err
		}
		defer destFile.Close()

		dest = io.MultiWriter(dest, destFile)
	}

	// Write docker-compose spec
	//
	if err = compose.WriteYAML(dest, proj); err != nil {
		return err
	}

	if envFile != "" {

		varsAndValues := make(map[string]string)
		for _, provider := range providers {
			if envTracker, ok := provider.(EnvVarTracker); ok {
				for k := range envTracker.Accessed() {
					varsAndValues[k] = os.Getenv(k)
				}
			}
		}

		// Open .env file
		//
		slog.Info(fmt.Sprintf("Writing output .env file '%s'", envFile))
		dest, err := os.Create(envFile)
		if err != nil {
			return err
		}
		defer dest.Close()

		// Write .env file
		//
		envVars := make([]string, 0)
		for key, val := range varsAndValues {
			var envVar = fmt.Sprintf("%s=%v\n", key, val)
			envVars = append(envVars, envVar)
		}
		sort.Strings(envVars)

		for _, envVar := range envVars {
			if _, err := dest.WriteString(envVar); err != nil {
				return err
			}
		}
	}

	return nil
}

type EnvVarTracker interface {
	Accessed() map[string]bool
}
