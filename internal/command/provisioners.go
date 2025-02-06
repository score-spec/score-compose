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

	"github.com/olekukonko/tablewriter"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/loader"
)

var (
	provisionersGroup = &cobra.Command{
		Use:   "provisioners",
		Short: "Subcommands related to provisioners",
	}
	provisionersList = &cobra.Command{
		Use:   "list",
		Short: "List the provisioners",
		Long: `The list command will list out the provisioners. This requires an active score compose state
after 'init' or 'generate' has been run. The list of provisioners will be empty if no provisioners are defined.
`,
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		RunE:          listProvisioners,
	}
)

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

	err = displayProvisioners(provisioners)
	if err != nil {
		return fmt.Errorf("failed to display provisioners: %w", err)
	}

	return nil
}

func displayProvisioners(loadedProvisioners []provisioners.Provisioner) error {
	rows := make([][]string, len(loadedProvisioners))

	sortedProvisioners := sortProvisionersByType(loadedProvisioners)
	for _, provisioner := range sortedProvisioners {
		rows = append(rows, []string{provisioner.Type(), provisioner.Class(), strings.Join(provisioner.Params(), ", "), strings.Join(provisioner.Outputs(), ", ")})
	}

	headers := []string{"Type", "Class", "Params", "Outputs"}
	displayTable(headers, rows)

	return nil
}

func sortProvisionersByType(provisioners []provisioners.Provisioner) []provisioners.Provisioner {
	sort.Slice(provisioners, func(i, j int) bool {
		return provisioners[i].Type() < provisioners[j].Type()
	})
	return provisioners
}

func displayTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.SetAutoWrapText(false)
	table.SetRowLine(true)
	table.SetCenterSeparator("+")
	table.SetColumnSeparator("|")
	table.SetRowSeparator("-")
	table.Render()
}

func init() {
	provisionersGroup.AddCommand(provisionersList)
	rootCmd.AddCommand(provisionersGroup)
}
