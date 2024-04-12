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

package util

import (
	"errors"
	"fmt"
	"os"
)

// DeferredEnvironmentVariable allows us to change when a particular variable is interpolated depending on the subcommand
// or destination. This is because environment variables can ONLY be interpolated in the docker compose files and not
// within resource params, file contents at "compose up" time.
type DeferredEnvironmentVariable struct {
	Variable string
	Required bool
}

func (dev *DeferredEnvironmentVariable) Error() string {
	return fmt.Sprintf("environment variable '%s' must be resolved", dev.Variable)
}

// WrapImmediateSubstitutionFunction wraps the given substitution function so that it can lookup the environment variable
// immediately and use that if available. Or throws an error.
func WrapImmediateSubstitutionFunction(inner func(string) (string, error)) func(string) (string, error) {
	return func(key string) (string, error) {
		o, err := inner(key)
		if err != nil {
			var dev *DeferredEnvironmentVariable
			if errors.As(err, &dev) {
				if v, ok := os.LookupEnv(dev.Variable); ok {
					return v, nil
				}
			}
			return "", err
		}
		return o, nil
	}
}

// WrapDeferredSubstitutionFunction wraps the given substitution function so that it can be interpolated within the
// docker compose file.
func WrapDeferredSubstitutionFunction(inner func(string) (string, error)) func(string) (string, error) {
	return func(key string) (string, error) {
		o, err := inner(key)
		if err != nil {
			var dev *DeferredEnvironmentVariable
			if errors.As(err, &dev) {
				if dev.Required {
					return fmt.Sprintf("${%s?required}", dev.Variable), nil
				}
				return fmt.Sprintf("${%s}", dev.Variable), nil
			}
			return "", err
		}
		return o, nil
	}
}
