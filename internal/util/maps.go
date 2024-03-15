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

import "maps"

func isMap(a any) bool {
	_, isMap := a.(map[string]any)
	return isMap
}

// PatchMap performs a JSON Merge Patch as defined in https://datatracker.ietf.org/doc/html/rfc7386.
//
// This should return a new map without modifying the current or patch inputs.
// Notes:
//   - if new is nil, the output is an empty object - this allows for in-place
//   - if a key is not a map, it will be treated as scalar according to the
//     JSON Merge Patch strategy. This includes structs and slices.
func PatchMap(current map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	// small shortcut here
	if len(patch) == 0 {
		return current
	}
	out := maps.Clone(current)
	if out == nil {
		out = make(map[string]interface{})
	}
	for k, patchValue := range patch {
		if patchValue == nil {
			delete(out, k)
		} else if existingValue, ok := out[k]; ok && isMap(patchValue) {
			patchMap := patchValue.(map[string]interface{})
			if isMap(existingValue) {
				out[k] = PatchMap(existingValue.(map[string]interface{}), patchMap)
			} else {
				out[k] = PatchMap(map[string]interface{}{}, patchMap)
			}
		} else if isMap(patchValue) {
			patchMap := patchValue.(map[string]interface{})
			out[k] = PatchMap(map[string]interface{}{}, patchMap)
		} else {
			out[k] = patchValue
		}
	}
	return out
}
