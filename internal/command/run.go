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
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/imdario/mergo"
	"github.com/score-spec/score-go/framework"
	"github.com/spf13/cobra"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"

	loader "github.com/score-spec/score-go/loader"
	schema "github.com/score-spec/score-go/schema"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/compose"
	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/envprov"
	"github.com/score-spec/score-compose/internal/util"
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

	// Log a deprecation warning
	slog.Warn("\n-----\nDeprecation notice: 'score-compose run' will be deprecated in the future - we recommend that you migrate to 'score-compose init' and 'score-compose generate'\n-----")

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

	// Build a fake score-compose init state. We don't actually need to store or persist this because we're not doing
	// anything iterative or stateful.
	state := &project.State{Extras: project.StateExtras{MountsDirectory: "/dev/null"}}
	state, err = state.WithWorkload(&spec, &scoreFile, project.WorkloadExtras{})
	if err != nil {
		return fmt.Errorf("failed to add score file to state: %w", err)
	}

	// Prime the resources with initial state and validate any issues
	state, err = state.WithPrimedResources()
	if err != nil {
		return fmt.Errorf("failed to prime resources: %w", err)
	}

	// Instead of actually calling the resource provisioning system, we skip it and fill in the supported resources
	// ourselves.
	provisionerList, envProvisioner, err := buildLegacyProvisioners(spec.Metadata["name"].(string), state)
	if err != nil {
		return err
	}

	state, err = provisioners.ProvisionResources(context.Background(), state, provisionerList, nil)
	if err != nil {
		return fmt.Errorf("failed to provision resources: %w", err)
	}

	// Build docker-compose configuration
	//
	slog.Info("Building docker-compose configuration")
	proj, err := compose.ConvertSpec(state, &spec)
	if err != nil {
		return fmt.Errorf("building docker-compose configuration: %w", err)
	}

	// Deprecated behavior of the run command which used to publish ports
	// Todo: remove this once score-compose run is removed
	if spec.Service != nil && len(spec.Service.Ports) > 0 {
		ports := make([]types.ServicePortConfig, 0)
		for _, pSpec := range spec.Service.Ports {
			var pubPort = fmt.Sprintf("%v", pSpec.Port)
			var protocol string
			if pSpec.Protocol != nil {
				protocol = strings.ToLower(string(*pSpec.Protocol))
			}
			ports = append(ports, types.ServicePortConfig{
				Published: pubPort,
				Target:    uint32(util.DerefOr(pSpec.TargetPort, pSpec.Port)),
				Protocol:  protocol,
			})
		}
		for serviceName, service := range proj.Services {
			if service.NetworkMode == "" {
				service.Ports = ports
				proj.Services[serviceName] = service
			}
		}
		sort.Slice(ports, func(i, j int) bool {
			return ports[i].Published < ports[j].Published
		})
	}

	// Override 'image' reference with 'build' instructions
	//
	if buildCtx != "" {
		slog.Info(fmt.Sprintf("Applying build context '%s' for service images", buildCtx))
		// We add the build context to all services and containers here and make a big assumption that all are
		// using the image we are building here and now. If this is unexpected, users should use a more complex
		// overrides file.
		for serviceName, service := range proj.Services {
			service.Build = &types.BuildConfig{Context: buildCtx}
			service.Image = ""
			proj.Services[serviceName] = service
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
		for key, val := range envProvisioner.Accessed() {
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

func buildLegacyProvisioners(workloadName string, state *project.State) ([]provisioners.Provisioner, *envprov.Provisioner, error) {
	envProv := new(envprov.Provisioner)
	out := []provisioners.Provisioner{envProv}
	for resName, res := range state.Workloads[workloadName].Spec.Resources {
		resUid := framework.NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)
		if resUid.Type() == "environment" {
			// handled by env prov which is already added above
		} else if resUid.Type() == "volume" && resUid.Class() == "default" {
			out = append(out, &legacyVolumeProvisioner{MatchResourceUid: resUid})
		} else {
			slog.Warn(fmt.Sprintf("resources.%s: '%s.%s' is not directly supported in score-compose, references will be converted to environment variables", resName, resUid.Type(), resUid.Class()))
			out = append(out, envProv.GenerateSubProvisioner(resName, resUid))
		}
	}
	return out, envProv, nil
}

type legacyVolumeProvisioner struct {
	MatchResourceUid framework.ResourceUid
}

func (l *legacyVolumeProvisioner) Uri() string {
	return "builtin://legacy-volume"
}

func (l *legacyVolumeProvisioner) Match(resUid framework.ResourceUid) bool {
	return l.MatchResourceUid == resUid
}

func (l *legacyVolumeProvisioner) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	return &provisioners.ProvisionOutput{ResourceOutputs: map[string]interface{}{}}, nil
}
