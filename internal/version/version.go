/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package version

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strconv"
)

var (
	Version             string = "0.0.0"
	semverPattern              = regexp.MustCompile(`^(?:v?)(\d+)(?:\.(\d+))?(?:\.(\d+))?$`)
	constraintAndSemver        = regexp.MustCompile("^(>|>=|=)?" + semverPattern.String()[1:])
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

func semverToI(x string) (int, error) {
	cpm := semverPattern.FindStringSubmatch(x)
	if cpm == nil {
		return 0, fmt.Errorf("invalid version: %s", x)
	}
	major, _ := strconv.Atoi(cpm[1])
	minor, patch := 999, 999
	if len(cpm) > 2 {
		minor, _ = strconv.Atoi(cpm[2])
		if len(cpm) > 3 {
			patch, _ = strconv.Atoi(cpm[3])
		}
	}
	return (major*1_000+minor)*1_000 + patch, nil
}

func AssertVersion(constraint string, current string) error {
	if currentI, err := semverToI(current); err != nil {
		return fmt.Errorf("current version is missing or invalid '%s'", current)
	} else if m := constraintAndSemver.FindStringSubmatch(constraint); m == nil {
		return fmt.Errorf("invalid constraint '%s'", constraint)
	} else {
		op := m[1]
		compareI, err := semverToI(m[0][len(op):])
		if err != nil {
			return fmt.Errorf("failed to parse constraint: %w", err)
		}
		match := false
		switch op {
		case ">":
			match = currentI > compareI
		case ">=":
			match = currentI >= compareI
		case "=":
			match = currentI == compareI
		}
		if !match {
			return fmt.Errorf("current version %s does not match requested constraint %s", current, constraint)
		}
		return nil
	}
}
