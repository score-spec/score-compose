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

package project

import (
	"fmt"
	"strings"
	"testing"

	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"
)

var substitutionFunction func(string) (string, error)

func init() {
	substitutionFunction = BuildSubstitutionFunction(score.WorkloadMetadata{
		"name":  "test-name",
		"other": map[string]interface{}{"key": "value"},
		"annotations": map[string]interface{}{
			"key.com/foo-bar": "thing",
		},
	}, map[string]OutputLookupFunc{
		"env": func(keys ...string) (interface{}, error) {
			if len(keys) != 1 {
				return nil, fmt.Errorf("fail")
			}
			return "${" + keys[0] + "}", nil
		},
		"db": func(keys ...string) (interface{}, error) {
			if len(keys) < 1 {
				return nil, fmt.Errorf("fail")
			}
			return "${DB_" + strings.ToUpper(strings.Join(keys, "_")) + "?required}", nil
		},
		"static": mapLookupOutput(map[string]interface{}{"x": "a"}),
	})
}

func TestSubstitutionFunction(t *testing.T) {
	for _, tc := range []struct {
		Input         string
		Expected      string
		ExpectedError string
	}{
		{Input: "missing", ExpectedError: "invalid ref 'missing': unknown reference root, use $$ to escape the substitution"},
		{Input: "metadata.name", Expected: "test-name"},
		{Input: "metadata", ExpectedError: "invalid ref 'metadata': requires at least a metadata key to lookup"},
		{Input: "metadata.other", Expected: "{\"key\":\"value\"}"},
		{Input: "metadata.other.key", Expected: "value"},
		{Input: "metadata.missing", ExpectedError: "invalid ref 'metadata.missing': key 'missing' not found"},
		{Input: "metadata.name.foo", ExpectedError: "invalid ref 'metadata.name.foo': cannot lookup key 'foo', context is not a map"},
		{Input: "metadata.annotations.key\\.com/foo-bar", Expected: "thing"},
		{Input: "resources.env", Expected: "env"},
		{Input: "resources.env.DEBUG", Expected: "${DEBUG}"},
		{Input: "resources.missing", ExpectedError: "invalid ref 'resources.missing': no known resource 'missing'"},
		{Input: "resources.db", Expected: "db"},
		{Input: "resources.db.host", Expected: "${DB_HOST?required}"},
		{Input: "resources.db.port", Expected: "${DB_PORT?required}"},
		{Input: "resources.db.name", Expected: "${DB_NAME?required}"},
		{Input: "resources.db.name.user", Expected: "${DB_NAME_USER?required}"},
		{Input: "resources.static", Expected: "static"},
		{Input: "resources.static.x", Expected: "a"},
		{Input: "resources.static.y", ExpectedError: "invalid ref 'resources.static.y': key 'y' not found"},
	} {
		t.Run(tc.Input, func(t *testing.T) {
			res, err := substitutionFunction(tc.Input)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, res)
			}
		})
	}
}

func TestSubstituteString(t *testing.T) {
	for _, tc := range []struct {
		Input         string
		Expected      string
		ExpectedError string
	}{
		{Input: "", Expected: ""},
		{Input: "abc", Expected: "abc"},
		{Input: "$abc", Expected: "$abc"},
		{Input: "abc $$ abc", Expected: "abc $ abc"},
		{Input: "$${abc}", Expected: "${abc}"},
		{Input: "$${abc .4t3298y *(^&(*}", Expected: "${abc .4t3298y *(^&(*}"},
		{Input: "my name is ${metadata.name}", Expected: "my name is test-name"},
		{Input: "my name is ${metadata.thing\\.two}", ExpectedError: "invalid ref 'metadata.thing\\.two': key 'thing.two' not found"},
		{Input: "my name is ${}", ExpectedError: "invalid ref '': unknown reference root, use $$ to escape the substitution"},
		{
			Input:    "postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}",
			Expected: "postgresql://${DB_USER?required}:${DB_PASSWORD?required}@${DB_HOST?required}:${DB_PORT?required}/${DB_NAME?required}",
		},
	} {
		t.Run(tc.Input, func(t *testing.T) {
			res, err := SubstituteString(tc.Input, substitutionFunction)
			if tc.ExpectedError != "" {
				if !assert.EqualError(t, err, tc.ExpectedError) {
					assert.Equal(t, "", res)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, res)
			}
		})
	}
}

func TestSubstituteMap_success(t *testing.T) {
	input := map[string]interface{}{
		"a": "1",
		"b": "${metadata.name}",
		"c": []interface{}{"1", "${metadata.name}", map[string]interface{}{"d": "${metadata.name}"}},
		"d": map[string]interface{}{
			"e": "1",
			"f": "${metadata.name}",
		},
	}
	output, err := Substitute(input, substitutionFunction)
	assert.NoError(t, err)
	assert.Equal(t, map[string]interface {
	}{
		"a": "1",
		"b": "test-name",
		"c": []interface{}{"1", "test-name", map[string]interface{}{"d": "test-name"}},
		"d": map[string]interface{}{
			"e": "1",
			"f": "test-name",
		},
	}, output)
}

func TestSubstituteMap_fail(t *testing.T) {
	_, err := Substitute(map[string]interface{}{
		"a": []interface{}{map[string]interface{}{"b": "${metadata.unknown}"}},
	}, substitutionFunction)
	assert.EqualError(t, err, "a: 0: b: invalid ref 'metadata.unknown': key 'unknown' not found")
}
