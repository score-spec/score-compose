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
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/score-spec/score-go/framework"
	"github.com/score-spec/score-go/uriget"
	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/provisioners/loader"
)

const (
	DefaultScoreFileContent = `# Score provides a developer-centric and platform-agnostic 
# Workload specification to improve developer productivity and experience. 
# Score eliminates configuration management between local and remote environments.
#
# Specification reference: https://docs.score.dev/docs/reference/score-spec-reference/
---

# Score specification version
apiVersion: score.dev/v1b1

metadata:
  name: example

containers:
  hello-world:
    image: nginx:latest

    # Uncomment the following for a custom entrypoint command
    # command: []

    # Uncomment the following for custom arguments
    # args: []

    # Environment variables to inject into the container
    variables:
      EXAMPLE_VARIABLE: "example-value"

service:
  ports:
    # Expose the http port from nginx on port 8080
    www:
      port: 8080
      targetPort: 80

resources: {}
`
	initCmdFileFlag         = "file"
	initCmdFileProjectFlag  = "project"
	initCmdFileNoSampleFlag = "no-sample"
	initCmdProvisionerFlag  = "provisioners"
)

//go:embed default.provisioners.yaml
var defaultProvisionersContent string

var initCmd = &cobra.Command{
	Use:   "init",
	Args:  cobra.NoArgs,
	Short: "Initialise a new score-compose project with local state directory and score file",
	Long: `The init subcommand will prepare the current directory for working with score-compose and prepare any local
files or configuration needed to be successful.

A directory named .score-compose will be created if it doesn't exist. This file stores local state and generally should
not be checked into source control. Add it to your .gitignore file if you use Git as version control.

The project name will be used as a Docker compose project name when the final compose files are written. This name
acts as a namespace when multiple score files and containers are used.

Custom provisioners can be installed by uri using the --provisioners flag. The provisioners will be installed and take
precedence in the order they are defined over the default provisioners. If init has already been called with provisioners
the new provisioners will take precedence.
`,
	Example: `
  # Define a score file to generate
  score-compose init --file score2.yaml

  # Or override the docker compose project name
  score-compose init --project score-compose2

  # Or disable the default score file generation if you already have a score file
  score-compose init --no-sample

  # Optionally loading in provisoners from a remote url
  score-compose init --provisioners https://raw.githubusercontent.com/user/repo/main/example.yaml`,

	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// load flag values
		initCmdScoreFile, _ := cmd.Flags().GetString(initCmdFileFlag)
		initCmdComposeProject, _ := cmd.Flags().GetString(initCmdFileProjectFlag)

		// validate project
		if initCmdComposeProject != "" {
			cleanedInitCmdComposeProject := cleanComposeProjectName(initCmdComposeProject)
			if cleanedInitCmdComposeProject != initCmdComposeProject {
				return fmt.Errorf("invalid value for --project, it must match ^[a-z0-9][a-z0-9_-]*$")
			}
		}

		sd, ok, err := project.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if ok {
			slog.Info(fmt.Sprintf("Found existing state directory '%s'", sd.Path))
			if initCmdComposeProject != "" && sd.State.Extras.ComposeProjectName != initCmdComposeProject {
				sd.State.Extras.ComposeProjectName = initCmdComposeProject
				if err := sd.Persist(); err != nil {
					return fmt.Errorf("failed to persist new compose project name: %w", err)
				}
			}
		} else {

			slog.Info(fmt.Sprintf("Writing new state directory '%s'", project.DefaultRelativeStateDirectory))
			wd, _ := os.Getwd()
			sd = &project.StateDirectory{
				Path: project.DefaultRelativeStateDirectory,
				State: project.State{
					Workloads:   map[string]framework.ScoreWorkloadState[project.WorkloadExtras]{},
					Resources:   map[framework.ResourceUid]framework.ScoreResourceState[framework.NoExtras]{},
					SharedState: map[string]interface{}{},
					Extras: project.StateExtras{
						ComposeProjectName: cleanComposeProjectName(filepath.Base(wd)),
						MountsDirectory:    filepath.Join(project.DefaultRelativeStateDirectory, project.MountsDirectoryName),
					},
				},
			}
			if initCmdComposeProject != "" {
				sd.State.Extras.ComposeProjectName = initCmdComposeProject
			}
			slog.Info(fmt.Sprintf("Writing new state directory '%s' with project name '%s'", sd.Path, sd.State.Extras.ComposeProjectName))
			if err := sd.Persist(); err != nil {
				return fmt.Errorf("failed to persist new compose project name: %w", err)
			}

			// create and write the default provisioners file if it doesn't already exist
			dst := "zz-default" + loader.DefaultSuffix
			if f, err := os.Stat(filepath.Join(sd.Path, "99-default"+loader.DefaultSuffix)); err == nil {
				slog.Info(fmt.Sprintf("Default provisioners yaml file '%s' already exists, not overwriting it", f.Name()))
			} else if f, err := os.OpenFile(filepath.Join(sd.Path, dst), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644); err != nil {
				if !errors.Is(err, os.ErrExist) {
					return fmt.Errorf("failed to open default provisioners for writing: %w", err)
				}
				slog.Info(fmt.Sprintf("Default provisioners yaml file '%s' already exists, not overwriting it", dst))
			} else {
				defer f.Close()
				slog.Info(fmt.Sprintf("Writing default provisioners yaml file '%s'", dst))
				if _, err = f.WriteString(defaultProvisionersContent); err != nil {
					return fmt.Errorf("failed to write provisioners: %w", err)
				}
				_ = f.Close()
			}
		}

		if _, err := os.ReadFile(initCmdScoreFile); err != nil {
			if v, _ := cmd.Flags().GetBool(initCmdFileNoSampleFlag); v {
				slog.Info(fmt.Sprintf("Initial Score file '%s' does not exist - and sample generation is disabled", initCmdScoreFile))
			} else {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("failed to check existing Score file: %w", err)
				}
				slog.Info(fmt.Sprintf("Initial Score file '%s' does not exist - creating it", initCmdScoreFile))
				if err := os.WriteFile(initCmdScoreFile+".temp", []byte(DefaultScoreFileContent), 0755); err != nil {
					return fmt.Errorf("failed to write initial score file: %w", err)
				} else if err := os.Rename(initCmdScoreFile+".temp", initCmdScoreFile); err != nil {
					return fmt.Errorf("failed to complete writing initial Score file: %w", err)
				}
			}
		} else {
			slog.Info(fmt.Sprintf("Found existing Score file '%s'", initCmdScoreFile))
		}

		if v, _ := cmd.Flags().GetStringArray(initCmdProvisionerFlag); len(v) > 0 {
			for i, vi := range v {
				data, err := uriget.GetFile(cmd.Context(), vi)
				if err != nil {
					return fmt.Errorf("failed to load provisioner %d: %w", i+1, err)
				}

				var saveFilename string
				if vi == "-" {
					saveFilename = "from-stdin.provisioners.yaml"
				} else {
					saveFilename = vi
				}

				if err := loader.SaveProvisionerToDirectory(sd.Path, saveFilename, data); err != nil {
					return fmt.Errorf("failed to save provisioner %d: %w", i+1, err)
				}
			}
		}

		if provs, err := loader.LoadProvisionersFromDirectory(sd.Path, loader.DefaultSuffix); err != nil {
			return fmt.Errorf("failed to load existing provisioners: %w", err)
		} else {
			slog.Debug(fmt.Sprintf("Successfully loaded %d resource provisioners", len(provs)))
		}

		slog.Info(fmt.Sprintf("Read more about the Score specification at https://docs.score.dev/docs/"))

		return nil
	},
}

func init() {
	initCmd.Flags().StringP(initCmdFileFlag, "f", scoreFileDefault, "The score file to initialize")
	initCmd.Flags().StringP(initCmdFileProjectFlag, "p", "", "Set the name of the docker compose project (defaults to the current directory name)")
	initCmd.Flags().Bool(initCmdFileNoSampleFlag, false, "Disable generation of the sample score file")
	initCmd.Flags().StringArray(initCmdProvisionerFlag, nil, "Provisioner files to install. May be specified multiple times. Supports:\n"+
		"- HTTP        : http://host/file\n"+
		"- HTTPS       : https://host/file\n"+
		"- Git (SSH)   : git-ssh://git@host/repo.git/file\n"+
		"- Git (HTTPS) : git-https://host/repo.git/file\n"+
		"- OCI         : oci://[registry/][namespace/]repository[:tag|@digest][#file]\n"+
		"- Local File  : /path/to/local/file\n"+
		"- Stdin       : - (read from standard input)")

	rootCmd.AddCommand(initCmd)
}

func cleanComposeProjectName(input string) string {
	input = strings.ToLower(input)
	isFirst := true
	input = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (!isFirst && ((r == '_') || (r == '-'))) {
			isFirst = false
			return r
		}
		isFirst = false
		return -1
	}, input)
	return input
}
