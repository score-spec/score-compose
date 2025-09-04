// Copyright 2024 The Score Authors
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

	"github.com/score-spec/score-compose/internal/patching"
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
	initCmdFileFlag                  = "file"
	initCmdFileProjectFlag           = "project"
	initCmdFileNoSampleFlag          = "no-sample"
	initCmdProvisionerFlag           = "provisioners"
	initCmdPatchTemplateFlag         = "patch-templates"
	initCmdNoDefaultProvisionersFlag = "no-default-provisioners"
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

To adjust the way the compose project is generated, or perform post processing actions, you can use the --patch-templates
flag to provide one or more template files by uri. Each template file is stored in the project and then evaluated as a 
Golang text/template and should output a yaml/json encoded array of patches. Each patch is an object with required 'op' 
(set or delete), 'patch' (a dot-separated json path), a 'value' if the 'op' == 'set', and an optional 'description' for 
showing in the logs. The template has access to '.Compose' and '.Workloads'.
`,
	Example: `
  # Define a score file to generate
  score-compose init --file score2.yaml

  # Or override the docker compose project name
  score-compose init --project score-compose2

  # Or disable the default score file generation if you already have a score file
  score-compose init --no-sample

  # Optionally loading in provisoners from a remote url
  score-compose init --provisioners https://raw.githubusercontent.com/user/repo/main/example.yaml

  # Optionally adding a couple of patching templates
  score-compose init --patch-templates ./patching.tpl --patch-templates https://raw.githubusercontent.com/user/repo/main/example.tpl

URI Retrieval:
  The --provisioners and --patch-templates arguments support URI retrieval for pulling the contents from a URI on disk
  or over the network. These support:
    - HTTP        : http://host/file
    - HTTPS       : https://host/file
    - Git (SSH)   : git-ssh://git@host/repo.git/file
    - Git (HTTPS) : git-https://host/repo.git/file
    - OCI         : oci://[registry/][namespace/]repository[:tag|@digest][#file]
    - Local File  : /path/to/local/file
    - Stdin       : - (read from standard input)`,

	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// load flag values
		initCmdScoreFile, _ := cmd.Flags().GetString(initCmdFileFlag)
		initCmdComposeProject, _ := cmd.Flags().GetString(initCmdFileProjectFlag)
		initCmdPatchingFiles, _ := cmd.Flags().GetStringArray(initCmdPatchTemplateFlag)

		// validate project
		if initCmdComposeProject != "" {
			cleanedInitCmdComposeProject := cleanComposeProjectName(initCmdComposeProject)
			if cleanedInitCmdComposeProject != initCmdComposeProject {
				return fmt.Errorf("invalid value for --project, it must match ^[a-z0-9][a-z0-9_-]*$")
			}
		}

		var templates []string
		for _, u := range initCmdPatchingFiles {
			slog.Info(fmt.Sprintf("Fetching patch template from %s", u))
			content, err := uriget.GetFile(cmd.Context(), u)
			if err != nil {
				return fmt.Errorf("error fetching patch template from %s: %w", u, err)
			} else if err = patching.ValidatePatchTemplate(string(content)); err != nil {
				return fmt.Errorf("error parsing patch template from %s: %w", u, err)
			}
			templates = append(templates, string(content))
		}

		sd, ok, err := project.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if ok {
			slog.Info(fmt.Sprintf("Found existing state directory '%s'", sd.Path))
			var hasChanges bool
			if initCmdComposeProject != "" && sd.State.Extras.ComposeProjectName != initCmdComposeProject {
				sd.State.Extras.ComposeProjectName = initCmdComposeProject
				hasChanges = true
			}
			if len(templates) > 0 {
				sd.State.Extras.PatchingTemplates = templates
				hasChanges = true
			}
			if hasChanges {
				if err := sd.Persist(); err != nil {
					return fmt.Errorf("failed to persist new state file: %w", err)
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
			if len(templates) > 0 {
				sd.State.Extras.PatchingTemplates = templates
			}
			slog.Info(fmt.Sprintf("Writing new state directory '%s' with project name '%s'", sd.Path, sd.State.Extras.ComposeProjectName))
			if err := sd.Persist(); err != nil {
				return fmt.Errorf("failed to persist new compose project name: %w", err)
			}

			// create and write the default provisioners file if it doesn't already exist
			disableDefaultProvisioners, err := cmd.Flags().GetBool(initCmdNoDefaultProvisionersFlag)
			if err != nil {
				return fmt.Errorf("failed to parse --%s flag: %w", initCmdNoDefaultProvisionersFlag, err)
			}

			if !disableDefaultProvisioners {
				defaultProvisioners := filepath.Join(sd.Path, "zz-default.provisioners.yaml")

				if _, err := os.Stat(defaultProvisioners); err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return fmt.Errorf("failed to check for existing default provisioners file: %w", err)
					}

					f, err := os.OpenFile(defaultProvisioners, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
					if err != nil {
						return fmt.Errorf("failed to create default provisioners file: %w", err)
					}
					defer f.Close()

					slog.Info("Writing default provisioners file", "path", defaultProvisioners)
					if _, err := f.WriteString(defaultProvisionersContent); err != nil {
						return fmt.Errorf("failed to write default provisioners content: %w", err)
					}
				} else {
					slog.Info("Default provisioners file already exists, skipping", "path", defaultProvisioners)
				}
			} else {
				slog.Info("Skipping default provisioners due to --no-default-provisioners flag")
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

		slog.Info("Read more about the Score specification at https://docs.score.dev/docs/")

		return nil
	},
}

func init() {
	initCmd.Flags().StringP(initCmdFileFlag, "f", scoreFileDefault, "The score file to initialize")
	initCmd.Flags().StringP(initCmdFileProjectFlag, "p", "", "Set the name of the docker compose project (defaults to the current directory name)")
	initCmd.Flags().Bool(initCmdFileNoSampleFlag, false, "Disable generation of the sample score file")
	initCmd.Flags().StringArray(initCmdProvisionerFlag, nil, "Provisioner files to install. May be specified multiple times. Supports URI retrieval.")
	initCmd.Flags().StringArray(initCmdPatchTemplateFlag, nil, "Patching template files to include. May be specified multiple times. Supports URI retrieval.")
	initCmd.Flags().Bool(initCmdNoDefaultProvisionersFlag, false, "Disable generation of the default provisioners file")

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
