package project

import (
	"testing"

	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func mustLoadWorkload(t *testing.T, spec string) *score.Workload {
	t.Helper()
	var raw score.Workload
	require.NoError(t, yaml.Unmarshal([]byte(spec), &raw))
	return &raw
}

func mustAddWorkload(t *testing.T, s *State, spec string) *State {
	t.Helper()
	w := mustLoadWorkload(t, spec)
	n, err := s.WithWorkload(w, nil, nil)
	require.NoError(t, err)
	return n
}

func TestWithWorkload(t *testing.T) {
	start := new(State)

	t.Run("one", func(t *testing.T) {
		next, err := start.WithWorkload(mustLoadWorkload(t, `
metadata:
  name: example
containers:
  hello-world:
    image: hi
resources:
  foo:
    type: thing
`), nil, nil)
		require.NoError(t, err)
		assert.Len(t, start.Workloads, 0)
		assert.Len(t, next.Workloads, 1)
		assert.Nil(t, next.Workloads["example"].File, nil)
		assert.Equal(t, score.Workload{
			Metadata:   map[string]interface{}{"name": "example"},
			Containers: map[string]score.Container{"hello-world": {Image: "hi"}},
			Resources:  map[string]score.Resource{"foo": {Type: "thing"}},
		}, next.Workloads["example"].Spec)
	})

	t.Run("two", func(t *testing.T) {
		next1, err := start.WithWorkload(mustLoadWorkload(t, `
metadata:
  name: example1
containers:
  hello-world:
    image: hi
resources:
  foo:
    type: thing
`), nil, nil)
		require.NoError(t, err)
		next2, err := next1.WithWorkload(mustLoadWorkload(t, `
metadata:
  name: example2
containers:
  hello-world:
    image: hi
`), nil, nil)
		require.NoError(t, err)

		assert.Len(t, start.Workloads, 0)
		assert.Len(t, next1.Workloads, 1)
		assert.Len(t, next2.Workloads, 2)
	})
}

func TestWithPrimedResources(t *testing.T) {
	start := new(State)

	t.Run("empty", func(t *testing.T) {
		next, err := start.WithPrimedResources()
		require.NoError(t, err)
		assert.Len(t, next.Resources, 0)
	})

	t.Run("one workload - nominal", func(t *testing.T) {
		next := mustAddWorkload(t, start, `
metadata: {"name": "example"}
resources:
  one:
    type: thing
  two:
    type: thing2
    class: banana
  three:
    type: thing3
    class: apple
    id: dog
    metadata:
      annotations:
        foo: bar
    params:
      color: green
  four:
    type: thing4
    id: elephant
  five:
    type: thing4
    id: elephant
    metadata:
      x: y
    params:
      color: blue
`)
		next, err := next.WithPrimedResources()
		require.NoError(t, err)
		assert.Len(t, start.Resources, 0)
		assert.Equal(t, map[ResourceUid]ScoreResourceState{
			"thing.default#example.one": {Type: "thing", Class: "default", Id: "example.one", State: map[string]interface{}{}},
			"thing2.banana#example.two": {Type: "thing2", Class: "banana", Id: "example.two", State: map[string]interface{}{}},
			"thing3.apple#dog": {
				Type: "thing3", Class: "apple", Id: "dog", State: map[string]interface{}{},
				Metadata: map[string]interface{}{"annotations": score.ResourceMetadata{"foo": "bar"}},
				Params:   map[string]interface{}{"color": "green"},
			},
			"thing4.default#elephant": {
				Type: "thing4", Class: "default", Id: "elephant", State: map[string]interface{}{},
				Metadata: map[string]interface{}{"x": "y"},
				Params:   map[string]interface{}{"color": "blue"},
			},
		}, next.Resources)
	})

	t.Run("one workload - diff metadata", func(t *testing.T) {
		next := mustAddWorkload(t, start, `
metadata: {"name": "example"}
resources:
  one:
    type: thing
    id: elephant
    metadata:
      x: a
  two:
    type: thing
    id: elephant
    metadata:
      x: y
`)
		next, err := next.WithPrimedResources()
		require.EqualError(t, err, "resource 'thing.default#elephant': multiple definitions with different metadata")
		assert.Len(t, start.Resources, 0)
	})

	t.Run("one workload - diff params", func(t *testing.T) {
		next := mustAddWorkload(t, start, `
metadata: {"name": "example"}
resources:
  one:
    type: thing
    id: elephant
    params:
      x: a
  two:
    type: thing
    id: elephant
    params:
      x: y
`)
		next, err := next.WithPrimedResources()
		require.EqualError(t, err, "resource 'thing.default#elephant': multiple definitions with different params")
		assert.Len(t, start.Resources, 0)
	})

	t.Run("two workload - nominal", func(t *testing.T) {
		t.Run("one workload - nominal", func(t *testing.T) {
			next := mustAddWorkload(t, start, `
metadata: {"name": "example1"}
resources:
  one:
    type: thing
  two:
    type: thing2
    id: dog
`)
			next = mustAddWorkload(t, next, `
metadata: {"name": "example2"}
resources:
  one:
    type: thing
  two:
    type: thing2
    id: dog
`)
			next, err := next.WithPrimedResources()
			require.NoError(t, err)
			assert.Len(t, start.Resources, 0)
			assert.Len(t, next.Resources, 3)
			assert.Equal(t, map[ResourceUid]ScoreResourceState{
				"thing.default#example1.one": {Type: "thing", Class: "default", Id: "example1.one", State: map[string]interface{}{}},
				"thing.default#example2.one": {Type: "thing", Class: "default", Id: "example2.one", State: map[string]interface{}{}},
				"thing2.default#dog":         {Type: "thing2", Class: "default", Id: "dog", State: map[string]interface{}{}},
			}, next.Resources)
		})
	})

}
