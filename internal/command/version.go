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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/score-spec/score-compose/internal/version"
	"github.com/spf13/cobra"
)

const (
	versionCmdFileNoLogo         = "no-logo"
	versionCmdFileNoUpdatesCheck = "no-updates-check"
	logo                         = `
                   ...    .............           
               .......   .............            
           .........     ............             
       .........        .....                     
      .......           ....   ..                 
       ..........      .....   ......             
           ........   ......   ..........         
              .....   .....       ..........      
                      ....          ........      
              ...........       .........         
           ..............    .........            
         ................   ......                
         ...............    ..                    
          ............
	`
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version for " + ScoreImplementationName + " and new version to update if available.",
	Args:  cobra.NoArgs,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		if noLogo, _ := cmd.Flags().GetBool(versionCmdFileNoLogo); !noLogo {
			fmt.Println(logo)
		}

		fmt.Println(ScoreImplementationName, version.BuildVersionString())

		if noUpdateCheck, _ := cmd.Flags().GetBool(versionCmdFileNoUpdatesCheck); !noUpdateCheck {
			if newer, err := checkForNewerVersion(version.Version); err == nil && newer != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nA newer version is available: %s\nUpdate at: https://github.com/score-spec/%s/releases/tag/%s\n", newer, ScoreImplementationName, newer)
			}
		}

		return nil
	},
}

// checkForNewerVersion queries the GitHub releases API and returns the tag name of the latest
// release if it is newer than currentVersion. Returns an empty string if no newer version is found
// or if the current version cannot be parsed.
func checkForNewerVersion(currentVersion string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/score-spec/" + ScoreImplementationName + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status from releases API: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if isNewerVersion(currentVersion, release.TagName) {
		return release.TagName, nil
	}
	return "", nil
}

// isNewerVersion reports whether latestVersion is strictly greater than currentVersion.
// Both versions may optionally start with a "v" prefix and are expected to follow semver
// (MAJOR.MINOR.PATCH). Non-numeric or unparseable segments are treated as 0.
func isNewerVersion(currentVersion, latestVersion string) bool {
	current := parseSemver(currentVersion)
	latest := parseSemver(latestVersion)
	for i := range current {
		if latest[i] > current[i] {
			return true
		}
		if latest[i] < current[i] {
			return false
		}
	}
	return false
}

// parseSemver splits a version string (with an optional leading "v") into its three numeric
// components [major, minor, patch]. Components that cannot be parsed are treated as 0.
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, _ := strconv.Atoi(parts[i])
		out[i] = n
	}
	return out
}

func init() {
	versionCmd.Flags().Bool(versionCmdFileNoLogo, false, "Do not show the Score logo")
	versionCmd.Flags().Bool(versionCmdFileNoUpdatesCheck, false, "Do not check for a new version")
	rootCmd.AddCommand(versionCmd)
}
