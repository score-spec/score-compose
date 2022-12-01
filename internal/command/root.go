/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package command

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/score-spec/score-compose/internal/version"
)

var (
	rootCmd = &cobra.Command{
		Use:   "score-compose",
		Short: "SCORE to docker-compose translator",
		Long: `SCORE is a specification for defining environment agnostic configuration for cloud based workloads.
This tool produces a docker-compose configuration file from the SCORE specification.
Complete documentation is available at https://score.dev`,
		Version: fmt.Sprintf("%s (build: %s; sha: %s)", version.Version, version.BuildTime, version.GitSHA),
	}
)

func init() {
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "%s" .Version}}
`)
}

func Execute() error {
	return rootCmd.Execute()
}
