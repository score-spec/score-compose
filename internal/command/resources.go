// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/score-spec/score-go/framework"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/project"
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
			for _, id := range resIds {
				_, _ = cmd.OutOrStdout().Write([]byte(id))
				_, _ = cmd.OutOrStdout().Write([]byte("\n"))
			}
			return nil
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
				formatValue := cmd.Flags().Lookup(getOutputsCmdFormatFlag).Value.String()
				switch formatValue {
				case "json":
					return json.NewEncoder(cmd.OutOrStdout()).Encode(outputs)
				case "yaml":
					return yaml.NewEncoder(cmd.OutOrStdout()).Encode(outputs)
				default:
					prepared, err := template.New("").Funcs(sprig.FuncMap()).Parse(formatValue)
					if err != nil {
						return fmt.Errorf("failed to parse format template: %w", err)
					}
					if err := prepared.Execute(cmd.OutOrStdout(), outputs); err != nil {
						return fmt.Errorf("failed to execute template: %w", err)
					}
					return nil
				}
			}
			return fmt.Errorf("no such resource '%s'", args[0])
		},
	}
)

func init() {
	getResourceOutputs.Flags().StringP(getOutputsCmdFormatFlag, "f", "json", "Format of the output: json, yaml, or a Go template with sprig functions")
	resourcesGroup.AddCommand(listResources)
	resourcesGroup.AddCommand(getResourceOutputs)
	rootCmd.AddCommand(resourcesGroup)
}
