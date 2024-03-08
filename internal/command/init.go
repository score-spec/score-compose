package command

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/project"
)

const DefaultScoreFileContent = `# Score provides a developer-centric and platform-agnostic 
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
`,
	Example: `
  # Define a score file to generate
  score-compose init --file score2.yaml

  # Or override the docker compose project name
  score-compose init --project score-compose2`,

	// don't print the errors - we print these ourselves in main()
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// load flag values
		initCmdScoreFile, _ := cmd.Flags().GetString("file")
		initCmdComposeProject, _ := cmd.Flags().GetString("project")

		sd, ok, err := project.LoadStateDirectory(".")
		if err != nil {
			return fmt.Errorf("failed to load existing state directory: %w", err)
		} else if ok {
			slog.Info(fmt.Sprintf("Found existing state directory '%s'", sd.Path))
			if initCmdComposeProject != "" && sd.State.ComposeProjectName != initCmdComposeProject {
				cleanedInitCmdComposeProject := cleanComposeProjectName(initCmdComposeProject)
				if cleanedInitCmdComposeProject != initCmdComposeProject {
					return fmt.Errorf("invalid value for --project, it must match ^[a-z0-9][a-z0-9_-]*$")
				}
				sd.State.ComposeProjectName = cleanedInitCmdComposeProject
				if err := sd.Persist(); err != nil {
					return fmt.Errorf("failed to persist new compose project name: %w", err)
				}
			}
		} else {
			wd, _ := os.Getwd()
			sd := &project.StateDirectory{
				Path: project.DefaultRelativeStateDirectory,
				State: project.State{
					ComposeProjectName: cleanComposeProjectName(filepath.Base(wd)),
					ScoreWorkloads:     map[string]project.ScoreWorkloadState{},
					Resources:          map[string]project.ScoreResourceState{},
				},
			}
			if initCmdComposeProject != "" {
				sd.State.ComposeProjectName = initCmdComposeProject
			}
			slog.Info(fmt.Sprintf("Writing new state directory '%s' with project name '%s'", project.DefaultRelativeStateDirectory, sd.State.ComposeProjectName))
			if err := sd.Persist(); err != nil {
				return fmt.Errorf("failed to persist new compose project name: %w", err)
			}
		}

		if _, err := os.ReadFile(initCmdScoreFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to check existing Score file: %w", err)
			}
			slog.Info(fmt.Sprintf("Initial Score file '%s' does not exist - creating it", initCmdScoreFile))
			if err := os.WriteFile(initCmdScoreFile+".temp", []byte(DefaultScoreFileContent), 0755); err != nil {
				return fmt.Errorf("failed to write initial score file: %w", err)
			} else if err := os.Rename(initCmdScoreFile+".temp", initCmdScoreFile); err != nil {
				return fmt.Errorf("failed to complete writing initial Score file: %w", err)
			}
		} else {
			slog.Info(fmt.Sprintf("Found existing Score file '%s'", initCmdScoreFile))
		}
		slog.Info(fmt.Sprintf("Read more about the Score specification at https://docs.score.dev/docs/"))

		return nil
	},
}

func cleanComposeProjectName(input string) string {
	input = strings.ToLower(input)
	isFirst := true
	input = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (!isFirst && (r == '_') || (r == '-')) {
			isFirst = false
			return r
		}
		isFirst = false
		return -1
	}, input)
	return input
}

func init() {
	initCmd.Flags().StringP("file", "f", scoreFileDefault, "The score file to initialize")
	initCmd.Flags().StringP("project", "p", "", "Set the name of the docker compose project (defaults to the current directory name)")

	rootCmd.AddCommand(initCmd)
}
