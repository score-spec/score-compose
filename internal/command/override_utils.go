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

package command

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

func parseDotPathParts(input string) []string {
	// support escaping dot's to insert elements with a . in them.
	input = strings.ReplaceAll(input, "\\\\", "\x01")
	input = strings.ReplaceAll(input, "\\.", "\x00")
	parts := strings.Split(input, ".")
	for i, part := range parts {
		part = strings.ReplaceAll(part, "\x00", ".")
		part = strings.ReplaceAll(part, "\x01", "\\")
		parts[i] = part
	}
	return parts
}

func writePathInStruct(input map[string]interface{}, path []string, isDelete bool, value interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("cannot change root node")
	}

	// the current position in the tree
	var current interface{} = input

	// a reference to the map that holds current
	var parentMap map[string]interface{}
	var parentKey string

	// first traverse to the right location
	for _, s := range path[:len(path)-1] {
		switch currentType := current.(type) {
		case map[string]interface{}:
			parentMap = currentType
			parentKey = s

			next, ok := currentType[s]
			if ok {
				current = next
			} else {
				currentType[s] = make(map[string]interface{})
				current = currentType[s]
			}
		case []interface{}:
			parentMap = nil

			idx, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("cannot index '%s' in array", s)
			} else if idx < 0 || idx >= len(currentType) {
				return fmt.Errorf("cannot set '%s' in array: out of range", s)
			}
			current = currentType[idx]
		default:
			return fmt.Errorf("cannot lookup property or index '%s' in %T", s, currentType)
		}
	}
	// then apply the change

	key := path[len(path)-1]
	switch currentType := current.(type) {
	case map[string]interface{}:
		if isDelete {
			delete(currentType, key)
		} else {
			currentType[key] = value
		}
	case []interface{}:
		// This is where the bulk of the complexity comes from. Parsing validating and then navigating the slices.
		idx, err := strconv.Atoi(key)
		if err != nil {
			return fmt.Errorf("cannot index '%s' in array", key)
		} else if idx < -1 || idx >= len(currentType) {
			return fmt.Errorf("cannot set '%s' in array: out of range", key)
		} else if isDelete {
			if idx == -1 {
				return fmt.Errorf("cannot delete '%s' in array", key)
			} else {
				if parentMap != nil {
					parentMap[parentKey] = slices.Delete(currentType, idx, idx+1)
				} else {
					return fmt.Errorf("override in nested arrays is not supported")
				}
			}
		} else {
			if idx == -1 {
				if parentMap != nil {
					parentMap[parentKey] = append(currentType, value)
				} else {
					return fmt.Errorf("override in nested arrays is not supported")
				}
			} else {
				currentType[idx] = value
			}
		}
	default:
		return fmt.Errorf("cannot lookup property or index '%s' in %T", key, currentType)
	}

	return nil
}
