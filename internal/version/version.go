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
	"runtime"
	"strconv"
)

var (
	Version             string = "0.0.0"
	GitCommit           string = "unknown"
	BuildDate           string = "unknown"
	semverPattern              = regexp.MustCompile(`^(?:v?)(\d+)(?:\.(\d+))?(?:\.(\d+))?$`)
	constraintAndSemver        = regexp.MustCompile("^(>|>=|=)?" + semverPattern.String()[1:])
)

// BuildVersionString constructs a version string by looking at the build metadata injected at build time.
func BuildVersionString() string {
	return fmt.Sprintf("%s (%s - %s/%s)\ngit commit: %s\nbuild date: %s", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH, GitCommit, BuildDate)
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
