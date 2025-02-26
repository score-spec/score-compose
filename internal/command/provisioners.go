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
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/loader"
	"github.com/score-spec/score-compose/internal/util"
)

var (
	provisionersGroup = &cobra.Command{
		Use:   "provisioners",
		Short: "Subcommands related to provisioners",
	}
	provisionersList = &cobra.Command{
		Use:   "list [--format|-f table|json]",
		Short: "List the provisioners",
		Long: `The list command will list out the provisioners. This requires an active score compose state
after 'init' or 'generate' has been run. The list of provisioners will be empty if no provisioners are defined.
`,
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		RunE:          listProvisioners,
	}
)

type provisionerOutput struct {
	Type    string
	Class   string
	Params  string
	Outputs string
}

func listProvisioners(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	wd, _ := os.Getwd()
	sd, ok, err := project.LoadStateDirectory(wd)
	if err != nil {
		return fmt.Errorf("failed to load existing state directory: %w", err)
	} else if !ok {
		return fmt.Errorf("no state directory found, run 'score-compose init' first")
	}
	slog.Debug(fmt.Sprintf("Listing provisioners in project '%s'", sd.State.Extras.ComposeProjectName))

	provisioners, err := loader.LoadProvisionersFromDirectory(sd.Path, loader.DefaultSuffix)
	if err != nil {
		return fmt.Errorf("failed to load provisioners in %s: %w", sd.Path, err)
	}

	if len(provisioners) == 0 {
		slog.Info("No provisioners found")
		return nil
	}

	outputFormat := cmd.Flag("format").Value.String()

	displayProvisioners(provisioners, outputFormat)

	return nil
}

func displayProvisioners(loadedProvisioners []provisioners.Provisioner, outputFormat string) {
	var outputFormatter util.OutputFormatter

	sortedProvisioners := sortProvisionersByType(loadedProvisioners)

	switch outputFormat {
	case "json":
		var outputs []provisionerOutput
		for _, provisioner := range sortedProvisioners {
			outputs = append(outputs, provisionerOutput{
				Type:    provisioner.Type(),
				Class:   provisioner.Class(),
				Params:  strings.Join(provisioner.Params(), ", "),
				Outputs: strings.Join(provisioner.Outputs(), ", "),
			})
		}
		outputFormatter = &util.JSONOutputFormatter[[]provisionerOutput]{Data: outputs}
	default:
		rows := [][]string{}

		for _, provisioner := range sortedProvisioners {
			rows = append(rows, []string{provisioner.Type(), provisioner.Class(), strings.Join(provisioner.Params(), ", "), strings.Join(provisioner.Outputs(), ", ")})
		}

		headers := []string{"Type", "Class", "Params", "Outputs"}

		outputFormatter = &util.TableOutputFormatter{
			Headers: headers,
			Rows:    rows,
		}
	}
	outputFormatter.Display()
}

func sortProvisionersByType(provisioners []provisioners.Provisioner) []provisioners.Provisioner {
	sort.Slice(provisioners, func(i, j int) bool {
		return provisioners[i].Type() < provisioners[j].Type()
	})
	return provisioners
}

func init() {
	provisionersList.Flags().StringP("format", "f", "table", "Format of the output: table (default), json")
	provisionersGroup.AddCommand(provisionersList)
	rootCmd.AddCommand(provisionersGroup)
}
