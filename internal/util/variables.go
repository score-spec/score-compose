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
	"regexp"
)

// PrepareEnvVariables replaces the dollar sign ($) by double dollar sign ($$)
// in slice strings for using in the docker compose file
func PrepareEnvVariables(arr []string) []string {
	// create pattern matching string
	re := regexp.MustCompile(`(^|[[:graph:]][^$])\$([\w+|\{?])`)

	// prepare replace string
	replaceStr := "${1}$$$$${2}"

	for i := range arr {
		var done bool
		for !done {
			arr[i] = re.ReplaceAllString(arr[i], replaceStr)
			done = !re.MatchString(arr[i])
		}
	}
	return arr
}
