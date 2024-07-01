// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapImmediateSubstitutionFunction(t *testing.T) {

	sf := WrapImmediateSubstitutionFunction(func(s string) (string, error) {
		switch s {
		case "one":
			return "ONE", nil
		case "two":
			return "", &DeferredEnvironmentVariable{Variable: "UNKNOWN VARIABLE TWO"}
		case "three":
			return "", &DeferredEnvironmentVariable{Variable: "UNKNOWN VARIABLE THREE", Required: true}
		default:
			return "", fmt.Errorf("an original error")
		}
	})

	x, err := sf("one")
	assert.NoError(t, err)
	assert.Equal(t, "ONE", x)

	x, err = sf("two")
	assert.EqualError(t, err, "environment variable 'UNKNOWN VARIABLE TWO' must be resolved")
	assert.Equal(t, "", x)

	assert.NoError(t, os.Setenv("UNKNOWN VARIABLE TWO", "BANANA"))
	defer os.Unsetenv("UNKNOWN VARIABLE TWO")
	x, err = sf("two")
	assert.NoError(t, err)
	assert.Equal(t, "BANANA", x)

	x, err = sf("three")
	assert.EqualError(t, err, "environment variable 'UNKNOWN VARIABLE THREE' must be resolved")
	assert.Equal(t, "", x)

	x, err = sf("four")
	assert.EqualError(t, err, "an original error")
	assert.Equal(t, "", x)

}

func TestWrapDeferredSubstitutionFunction(t *testing.T) {

	sf := WrapDeferredSubstitutionFunction(func(s string) (string, error) {
		switch s {
		case "one":
			return "ONE", nil
		case "two":
			return "", &DeferredEnvironmentVariable{Variable: "UNKNOWN VARIABLE TWO"}
		case "three":
			return "", &DeferredEnvironmentVariable{Variable: "UNKNOWN VARIABLE THREE", Required: true}
		default:
			return "", fmt.Errorf("an original error")
		}
	})

	x, err := sf("one")
	assert.NoError(t, err)
	assert.Equal(t, "ONE", x)

	x, err = sf("two")
	assert.NoError(t, err)
	assert.Equal(t, "${UNKNOWN VARIABLE TWO}", x)

	x, err = sf("two")
	assert.NoError(t, err)
	assert.Equal(t, "${UNKNOWN VARIABLE TWO}", x)

	x, err = sf("three")
	assert.NoError(t, err)
	assert.Equal(t, "${UNKNOWN VARIABLE THREE?required}", x)

	x, err = sf("four")
	assert.EqualError(t, err, "an original error")
	assert.Equal(t, "", x)

}
