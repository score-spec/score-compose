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
