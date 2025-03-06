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
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/score-spec/score-go/framework"
	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

const (
	getOutputsCmdFormatFlag = "format"
)

var (
	resourcesGroup = &cobra.Command{
		Use:   "resources",
		Short: "Subcommands related to provisioned resources",
	}
	listResources = &cobra.Command{
		Use:   "list",
		Short: "List the resource uids",
		Long: `The list command will list out the provisioned resource uids. This requires an active score compose state
after 'init' or 'generate' has been run. The list of uids will be empty if no resources are provisioned.
`,
		Args:          cobra.ExactArgs(0),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			sd, ok, err := project.LoadStateDirectory(".")
			if err != nil {
				return fmt.Errorf("failed to load existing state directory: %w", err)
			} else if !ok {
				return fmt.Errorf("state directory does not exist, please run \"score-compose init\" first")
			}
			slog.Debug(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.State.Extras.ComposeProjectName))
			currentState := &sd.State
			resIds, err := currentState.GetSortedResourceUids()
			if err != nil {
				return fmt.Errorf("failed to sort resources: %w", err)
			}
			return displayResourcesList(resIds, *currentState, cmd)
		},
	}
	getResourceOutputs = &cobra.Command{
		Use:   "get-outputs TYPE.CLASS#ID",
		Short: "Return the resource outputs",
		Long: `The get-outputs command will print the outputs of the resource from the last provisioning. The data will
be returned as json.
`,
		Args:          cobra.ExactArgs(1),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			sd, ok, err := project.LoadStateDirectory(".")
			if err != nil {
				return fmt.Errorf("failed to load existing state directory: %w", err)
			} else if !ok {
				return fmt.Errorf("state directory does not exist, please run \"score-compose init\" first")
			}
			slog.Debug(fmt.Sprintf("Loaded state directory with docker compose project '%s'", sd.State.Extras.ComposeProjectName))
			if res, ok := sd.State.Resources[framework.ResourceUid(args[0])]; ok {
				outputs := res.Outputs
				if outputs == nil {
					outputs = make(map[string]interface{})
				}
				return displayResourcesOutputs(outputs, cmd)
			}
			resourceOuptuts, err := getResourceOutputsByUid(framework.ResourceUid(args[0]), &sd.State)
			if err != nil {
				return fmt.Errorf("no such resource '%s'", args[0])
			}
			return displayResourcesOutputs(resourceOuptuts, cmd)
		},
	}
)

func getResourceOutputsByUid(uid framework.ResourceUid, state *project.State) (map[string]interface{}, error) {
	if res, ok := state.Resources[uid]; ok {
		outputs := res.Outputs
		if outputs == nil {
			outputs = make(map[string]interface{})
		}
		return outputs, nil
	}
	return nil, fmt.Errorf("no such resource '%s'", uid)
}

func getResourceOutputsKeys(uid framework.ResourceUid, state *project.State) ([]string, error) {
	outputs, err := getResourceOutputsByUid(uid, state)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(outputs))
	for key, _ := range outputs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys, nil
}

func displayResourcesOutputs(outputs map[string]interface{}, cmd *cobra.Command) error {
	outputFormat := cmd.Flags().Lookup(getOutputsCmdFormatFlag).Value.String()
	var outputFormatter util.OutputFormatter
	switch outputFormat {
	case "json":
		outputFormatter = &util.JSONOutputFormatter[map[string]interface{}]{Data: outputs, Out: cmd.OutOrStdout()}
	case "yaml":
		outputFormatter = &util.YAMLOutputFormatter[map[string]interface{}]{Data: outputs, Out: cmd.OutOrStdout()}
	default:
		// ensure there is a new line at the end if one is not already present
		if !strings.HasSuffix(outputFormat, "\n") {
			outputFormat += "\n"
		}
		prepared, err := template.New("").Funcs(sprig.FuncMap()).Parse(outputFormat)
		if err != nil {
			return fmt.Errorf("failed to parse format template: %w", err)
		}
		if err := prepared.Execute(cmd.OutOrStdout(), outputs); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}
		return nil
	}

	return outputFormatter.Display()
}

func displayResourcesList(resources []framework.ResourceUid, state project.State, cmd *cobra.Command) error {
	outputFormat := cmd.Flag("format").Value.String()
	var outputFormatter util.OutputFormatter

	switch outputFormat {
	case "json":
		type jsonData struct {
			UID     string
			Outputs []string
		}
		var outputs []jsonData
		for _, resource := range resources {

			keys, err := getResourceOutputsKeys(resource, &state)
			if err != nil {
				return fmt.Errorf("failed to get outputs for resource '%s': %w", resource, err)
			}
			outputs = append(outputs, jsonData{
				UID:     string(resource),
				Outputs: keys,
			})
		}
		outputFormatter = &util.JSONOutputFormatter[[]jsonData]{Data: outputs, Out: cmd.OutOrStdout()}
	default:
		var rows [][]string
		for _, resource := range resources {
			keys, err := getResourceOutputsKeys(resource, &state)
			if err != nil {
				return fmt.Errorf("failed to get outputs for resource '%s': %w", resource, err)
			}
			row := []string{string(resource), strings.Join(keys, ", ")}
			rows = append(rows, row)
		}
		outputFormatter = &util.TableOutputFormatter{
			Headers: []string{"UID", "Outputs"},
			Rows:    rows,
			Out:     cmd.OutOrStdout(),
		}
	}

	return outputFormatter.Display()
}

func init() {
	getResourceOutputs.Flags().StringP(getOutputsCmdFormatFlag, "f", "json", "Format of the output: json, yaml, or a Go template with sprig functions")
	resourcesGroup.AddCommand(listResources)
	listResources.Flags().StringP("format", "f", "table", "Format of the output: table (default), json")
	resourcesGroup.AddCommand(getResourceOutputs)
	rootCmd.AddCommand(resourcesGroup)
}
