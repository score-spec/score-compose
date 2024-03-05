package command

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
			if initCmdComposeProject != "" && sd.Config.ComposeProjectName != initCmdComposeProject {
				sd.Config.ComposeProjectName = initCmdComposeProject
				if err := sd.Persist(); err != nil {
					return fmt.Errorf("failed to persist new compose project name: %w", err)
				}
			}
		} else {
			slog.Info(fmt.Sprintf("Writing new state directory '%s'", project.DefaultRelativeStateDirectory))
			wd, _ := os.Getwd()
			sd := &project.StateDirectory{
				Path:   project.DefaultRelativeStateDirectory,
				Config: project.Config{ComposeProjectName: filepath.Base(wd)},
			}
			if initCmdComposeProject != "" {
				sd.Config.ComposeProjectName = initCmdComposeProject
			}
			if err := sd.Persist(); err != nil {
				return fmt.Errorf("failed to persist new compose project name: %w", err)
			}
		}

		if st, err := os.Stat(initCmdScoreFile); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to read existing Score file: %w", err)
			}
			slog.Info(fmt.Sprintf("Initial Score file '%s' does not exist - creating it", initCmdScoreFile))

			if err := os.WriteFile(initCmdScoreFile+".temp", []byte(DefaultScoreFileContent), 0755); err != nil {
				return fmt.Errorf("failed to write initial score file: %w", err)
			} else if err := os.Rename(initCmdScoreFile+".temp", initCmdScoreFile); err != nil {
				return fmt.Errorf("failed to complete writing initial Score file: %w", err)
			}
		} else if st.IsDir() || !st.Mode().IsRegular() {
			return fmt.Errorf("existing Score file is not a regular file")
		} else {
			slog.Info(fmt.Sprintf("Found existing Score file '%s'", initCmdScoreFile))
		}
		slog.Info(fmt.Sprintf("Read more about the Score specification at https://docs.score.dev/docs/"))

		return nil
	},
}

func init() {
	initCmd.Flags().StringP("file", "f", scoreFileDefault, "The score file to initialize")
	initCmd.Flags().StringP("project", "p", "", "Set the name of the docker compose project (defaults to the current directory name)")

	rootCmd.AddCommand(initCmd)
}
