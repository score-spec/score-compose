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
	"path/filepath"
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
	currentState := &sd.State
	provisionerFiles, err := getProvisionerFiles(sd.Path, currentState.Extras.ComposeProjectName)
	if err != nil {
		return fmt.Errorf("failed to get provisioner file: %w", err)
	}

	resIds, err := currentState.GetSortedResourceUids()
	if err != nil {
		return fmt.Errorf("failed to get sorted resource uids: %w", err)
	}
	for _, resId := range resIds {
		_, ok := currentState.Resources[resId]
		if !ok {
			return fmt.Errorf("resource %s not found", resId)
		}
	}
	err = displayProvisioners(provisionerFiles)
	if err != nil {
		return fmt.Errorf("failed to display provisioners: %w", err)
	}
	return nil
}

func displayProvisioners(provisionerFiles []string) error {
	rows := [][]string{}
	provisioners := []provisioners.Provisioner{}
	for _, provisionerFile := range provisionerFiles {
		provisionerContent, err := os.ReadFile(provisionerFile)
		if err != nil {
			return fmt.Errorf("failed to read provisioner file: %w", err)
		}
		loadedProvisioners, err := loader.LoadProvisioners(provisionerContent)
		if err != nil {
			return fmt.Errorf("failed to load provisioners: %w", err)
		}
		provisioners = append(provisioners, loadedProvisioners...)
	}

	for _, provisioner := range provisioners {
		rows = append(rows, []string{provisioner.Type(), provisioner.Class(), strings.Join(provisioner.Params(), ", "), strings.Join(provisioner.Outputs(), ", ")})
	}

	if len(rows) == 0 {
		slog.Info("No provisioners found")
		return nil
	}

	headers := []string{"Type", "Class", "Params", "Outputs"}
	displayTable(headers, rows)

	return nil
}

func displayTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.SetAutoWrapText(false)
	table.Render()
}

// getProvisionerFile returns the path to the provisioner file matching the given hash,
// or falls back to the default file if no match is found
func getProvisionerFiles(path string, projectName string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read directory: %w", err)
	}

	slog.Debug(fmt.Sprintf("Looking for provisioner file for project '%s' in path '%s'", projectName, path))

	// Look for a file matching the hash
	customProvisionerFiles := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "Z-") && strings.HasSuffix(entry.Name(), ".provisioners.yaml") {
			customProvisionerFiles = append(customProvisionerFiles, filepath.Join(path, entry.Name()))
		}
	}
	if len(customProvisionerFiles) == 0 {
		defaultFile := filepath.Join(path, "zz-default.provisioners.yaml")
		if _, err := os.Stat(defaultFile); err != nil {
			return []string{}, fmt.Errorf("default provisioners file not found: %w", err)
		}
		return []string{defaultFile}, nil
	}
	return customProvisionerFiles, nil
}

func init() {
	provisionersGroup.AddCommand(provisionersList)
	rootCmd.AddCommand(provisionersGroup)
}
