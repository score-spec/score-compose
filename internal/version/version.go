/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Version string = "0.0.0"
)

// BuildVersionString constructs a version string by looking at the build metadata injected at build time.
// This is particularly useful when score-compose is stilled from the go module using go install.
func BuildVersionString() string {
	versionNumber, buildTime, gitSha, isDirtySuffix := Version, "local", "unknown", ""
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			versionNumber = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.time":
				buildTime = setting.Value
			case "vcs.revision":
				gitSha = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					isDirtySuffix = "-dirty"
				}
			}
		}
	}
	return fmt.Sprintf("%s (build: %s, sha: %s%s)", versionNumber, buildTime, gitSha, isDirtySuffix)
}
