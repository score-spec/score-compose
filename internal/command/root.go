/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package command

import (
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/logging"
	"github.com/score-spec/score-compose/internal/version"
)

var (
	rootCmd = &cobra.Command{
		Use:   "score-compose",
		Short: "SCORE to docker-compose translator",
		Long: `SCORE is a specification for defining environment agnostic configuration for cloud based workloads.
This tool produces a docker-compose configuration file from the SCORE specification.
Complete documentation is available at https://score.dev`,
		// don't print the errors - we print these ourselves in main()
		SilenceErrors: true,

		// This function always runs for all subcommands
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if q, _ := cmd.Flags().GetBool("quiet"); q {
				slog.SetDefault(slog.New(&logging.SimpleHandler{Level: slog.LevelError, Writer: io.Discard}))
			} else if v, _ := cmd.Flags().GetCount("verbose"); v == 0 {
				slog.SetDefault(slog.New(&logging.SimpleHandler{Level: slog.LevelInfo, Writer: cmd.ErrOrStderr()}))
			} else if v == 1 {
				slog.SetDefault(slog.New(&logging.SimpleHandler{Level: slog.LevelDebug, Writer: cmd.ErrOrStderr()}))
			} else if v == 2 {
				slog.SetDefault(slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
					Level: slog.LevelDebug, AddSource: true,
				})))
			}
			return nil
		},
	}
)

func init() {
	rootCmd.Version = version.BuildVersionString()
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s" .Version}}
`)
	rootCmd.PersistentFlags().Bool("quiet", false, "Mute any logging output")
	rootCmd.PersistentFlags().CountP("verbose", "v", "Increase log verbosity and detail by specifying this flag one or more times")
}

func Execute() error {
	return rootCmd.Execute()
}
