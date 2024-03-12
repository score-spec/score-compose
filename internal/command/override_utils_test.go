package command

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDotPathParts(t *testing.T) {
	for _, tc := range []struct {
		Input    string
		Expected []string
	}{
		{"", []string{""}},
		{"a", []string{"a"}},
		{"a.b", []string{"a", "b"}},
		{"a.-1", []string{"a", "-1"}},
		{"a.b\\.c", []string{"a", "b.c"}},
		{"a.b\\\\.c", []string{"a", "b\\", "c"}},
	} {
		t.Run(tc.Input, func(t *testing.T) {
			assert.Equal(t, tc.Expected, parseDotPathParts(tc.Input))
		})
	}
}

func TestWritePathInStruct(t *testing.T) {
	for _, tc := range []struct {
		Name          string
		Spec          string
		Path          []string
		Delete        bool
		Value         interface{}
		Expected      string
		ExpectedError error
	}{
		{
			Name:     "simple object set",
			Spec:     `{"a":{"b":[{}]}}`,
			Path:     []string{"a", "b", "0", "c"},
			Value:    "hello",
			Expected: `{"a":{"b":[{"c":"hello"}]}}`,
		},
		{
			Name:     "simple object delete",
			Spec:     `{"a":{"b":[{"c":"hello"}]}}`,
			Path:     []string{"a", "b", "0", "c"},
			Delete:   true,
			Expected: `{"a":{"b":[{}]}}`,
		},
		{
			Name:     "simple array set",
			Spec:     `{"a":[{}]}`,
			Path:     []string{"a", "0"},
			Value:    "hello",
			Expected: `{"a":["hello"]}`,
		},
		{
			Name:     "simple array append",
			Spec:     `{"a":["hello"]}`,
			Path:     []string{"a", "-1"},
			Value:    "world",
			Expected: `{"a":["hello","world"]}`,
		},
		{
			Name:     "simple array delete",
			Spec:     `{"a":["hello", "world"]}`,
			Path:     []string{"a", "0"},
			Delete:   true,
			Expected: `{"a":["world"]}`,
		},
		{
			Name:     "build object via path",
			Spec:     `{}`,
			Path:     []string{"a", "b"},
			Value:    "hello",
			Expected: `{"a":{"b":"hello"}}`,
		},
		{
			Name:          "bad index str",
			Spec:          `{"a":[]}`,
			Path:          []string{"a", "b"},
			Value:         "hello",
			ExpectedError: fmt.Errorf("cannot index 'b' in array"),
		},
		{
			Name:          "index out of range",
			Spec:          `{"a": [0]}`,
			Path:          []string{"a", "2"},
			Value:         "hello",
			ExpectedError: fmt.Errorf("cannot set '2' in array: out of range"),
		},
		{
			Name:          "no append nested arrays",
			Spec:          `{"a":[[0]]}`,
			Path:          []string{"a", "0", "-1"},
			Value:         "hello",
			ExpectedError: fmt.Errorf("override in nested arrays is not supported"),
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			var inSpec map[string]interface{}
			assert.NoError(t, json.Unmarshal([]byte(tc.Spec), &inSpec))
			err := writePathInStruct(inSpec, tc.Path, tc.Delete, tc.Value)
			if tc.ExpectedError != nil {
				assert.EqualError(t, err, tc.ExpectedError.Error())
			} else {
				if assert.NoError(t, err) {
					outSpec, _ := json.Marshal(inSpec)
					assert.JSONEq(t, tc.Expected, string(outSpec))
				}
			}
		})
	}
}
