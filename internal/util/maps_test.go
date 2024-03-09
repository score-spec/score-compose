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
