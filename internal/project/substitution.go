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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// placeholderRegEx will search for ${...} with any sequence of characters between them.
	placeholderRegEx = regexp.MustCompile(`\$(\$|{([^}]*)})`)
)

func SplitRefParts(ref string) []string {
	subRef := strings.Replace(ref, `\.`, "\000", -1)
	parts := strings.Split(subRef, ".")
	for i, part := range parts {
		parts[i] = strings.Replace(part, "\000", ".", -1)
	}
	return parts
}

// SubstituteString replaces all matching '${...}' templates in a source string with whatever is returned
// from the inner function.
func SubstituteString(src string, inner func(string) (string, error)) (string, error) {
	var err error
	result := placeholderRegEx.ReplaceAllStringFunc(src, func(str string) string {
		// WORKAROUND: ReplaceAllStringFunc(..) does not provide match details
		//             https://github.com/golang/go/issues/5690
		var matches = placeholderRegEx.FindStringSubmatch(str)

		// SANITY CHECK
		if len(matches) != 3 {
			err = errors.Join(err, fmt.Errorf("could not find a proper match in previously captured string fragment"))
			return src
		}

		// support escaped dollars
		if matches[1] == "$" {
			return matches[1]
		}

		result, subErr := inner(matches[2])
		err = errors.Join(err, subErr)
		return result
	})
	return result, err
}

// Substitute does the same thing as SubstituteString but recursively through a map. It returns a copy of the original map.
func Substitute(source interface{}, inner func(string) (string, error)) (interface{}, error) {
	if source == nil {
		return nil, nil
	}
	switch v := source.(type) {
	case string:
		return SubstituteString(v, inner)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, v := range v {
			v2, err := Substitute(v, inner)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}
			out[k] = v2
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, i2 := range v {
			i3, err := Substitute(i2, inner)
			if err != nil {
				return nil, fmt.Errorf("%d: %w", i, err)
			}
			out[i] = i3
		}
		return out, nil
	default:
		return source, nil
	}
}

func mapLookupOutput(ctx map[string]interface{}) func(keys ...string) (interface{}, error) {
	return func(keys ...string) (interface{}, error) {
		var resolvedValue interface{}
		resolvedValue = ctx
		for _, k := range keys {
			mapV, ok := resolvedValue.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("cannot lookup key '%s', context is not a map", k)
			}
			resolvedValue, ok = mapV[k]
			if !ok {
				return "", fmt.Errorf("key '%s' not found", k)
			}
		}
		return resolvedValue, nil
	}
}

func BuildSubstitutionFunction(metadata map[string]interface{}, resources map[string]OutputLookupFunc) func(string) (string, error) {
	metadataLookup := mapLookupOutput(metadata)
	return func(ref string) (string, error) {
		parts := SplitRefParts(ref)
		var resolvedValue interface{}
		switch parts[0] {
		case "metadata":
			if len(parts) < 2 {
				return "", fmt.Errorf("invalid ref '%s': requires at least a metadata key to lookup", ref)
			}
			if rv, err := metadataLookup(parts[1:]...); err != nil {
				return "", fmt.Errorf("invalid ref '%s': %w", ref, err)
			} else {
				resolvedValue = rv
			}
		case "resources":
			if len(parts) < 2 {
				return "", fmt.Errorf("invalid ref '%s': requires at least a resource name to lookup", ref)
			}
			rv, ok := resources[parts[1]]
			if !ok {
				return "", fmt.Errorf("invalid ref '%s': no known resource '%s'", ref, parts[1])
			} else if len(parts) == 2 {
				// TODO: deprecate this - this is an annoying and nonsensical legacy thing
				return parts[1], nil
			} else if rv2, err := rv(parts[2:]...); err != nil {
				return "", fmt.Errorf("invalid ref '%s': %w", ref, err)
			} else {
				resolvedValue = rv2
			}
		default:
			return "", fmt.Errorf("invalid ref '%s': unknown reference root, use $$ to escape the substitution", ref)
		}

		if asString, ok := resolvedValue.(string); ok {
			return asString, nil
		}
		// TODO: work out how we might support other types here in the future
		raw, err := json.Marshal(resolvedValue)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
}
