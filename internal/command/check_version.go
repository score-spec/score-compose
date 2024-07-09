package command

import (
	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/version"
)

var checkVersionCmd = &cobra.Command{
	Use:   "check-version [constraint]",
	Short: "Assert that the version of score-compose matches the required constraint",
	Long: `score-compose is commonly used in Makefiles and CI pipelines which may depend on a particular functionality
or a particular default provisioner provided by score-compose init. This command provides a common way to check that
the version of score-compose matches a required version.
`,
	Example: `
  # check that the version is exactly 1.2.3
  score-compose check-version =v1.2.3

  # check that the version is 1.3.0 or greater
  score-compose check-version >v1.2

  # check that the version is equal or greater to 1.2.3
  score-compose check-version >=1.2.3`,
	Args:              cobra.ExactArgs(1),
	SilenceErrors:     true,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return version.AssertVersion(args[0], version.Version)
	},
}

func init() {
	rootCmd.AddCommand(checkVersionCmd)
}
