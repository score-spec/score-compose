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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatchMap(t *testing.T) {

	t.Run("nils", func(t *testing.T) {
		assert.Nil(t, PatchMap(nil, nil))
	})

	t.Run("patch some into nil", func(t *testing.T) {
		assert.Equal(t, map[string]interface{}{
			"a": "b",
			"d": map[string]interface{}{
				"e": "f",
				"h": map[string]interface{}{},
			},
			"i": []interface{}{"a", "b"},
		}, PatchMap(nil, map[string]interface{}{
			"a": "b",
			"c": nil,
			"d": map[string]interface{}{
				"e": "f",
				"g": nil,
				"h": map[string]interface{}{},
			},
			"i": []interface{}{"a", "b"},
		}))
	})

	t.Run("patch some into some", func(t *testing.T) {
		before := map[string]interface{}{
			"a": "x",
			"d": map[string]interface{}{
				"h": map[string]interface{}{
					"x": "y",
				},
			},
		}
		assert.Equal(t, map[string]interface{}{
			"a": "b",
			"d": map[string]interface{}{
				"e": "f",
				"h": map[string]interface{}{
					"x": "y",
				},
			},
			"i": []interface{}{"a", "b"},
		}, PatchMap(before, map[string]interface{}{
			"a": "b",
			"c": nil,
			"d": map[string]interface{}{
				"e": "f",
				"g": nil,
				"h": map[string]interface{}{},
			},
			"i": []interface{}{"a", "b"},
		}))
		assert.Equal(t, map[string]interface{}{
			"a": "x",
			"d": map[string]interface{}{
				"h": map[string]interface{}{
					"x": "y",
				},
			},
		}, before)
	})

}
