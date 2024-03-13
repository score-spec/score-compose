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
			"thing.default#example.one": {
				Type: "thing", Class: "default", Id: "example.one", State: map[string]interface{}{},
				SourceWorkload: "example",
			},
			"thing2.banana#example.two": {
				Type: "thing2", Class: "banana", Id: "example.two", State: map[string]interface{}{},
				SourceWorkload: "example",
			},
			"thing3.apple#dog": {
				Type: "thing3", Class: "apple", Id: "dog", State: map[string]interface{}{},
				Metadata:       map[string]interface{}{"annotations": score.ResourceMetadata{"foo": "bar"}},
				Params:         map[string]interface{}{"color": "green"},
				SourceWorkload: "example",
			},
			"thing4.default#elephant": {
				Type: "thing4", Class: "default", Id: "elephant", State: map[string]interface{}{},
				Metadata:       map[string]interface{}{"x": "y"},
				Params:         map[string]interface{}{"color": "blue"},
				SourceWorkload: "example",
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
				"thing.default#example1.one": {
					Type: "thing", Class: "default", Id: "example1.one", State: map[string]interface{}{},
					SourceWorkload: "example1",
				},
				"thing.default#example2.one": {
					Type: "thing", Class: "default", Id: "example2.one", State: map[string]interface{}{},
					SourceWorkload: "example2",
				},
				"thing2.default#dog": {
					Type: "thing2", Class: "default", Id: "dog", State: map[string]interface{}{},
					SourceWorkload: "example1",
				},
			}, next.Resources)
		})
	})

}

func TestGetSortedResourceUids(t *testing.T) {

	t.Run("empty", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Empty(t, ru)
	})

	t.Run("one", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res": {Type: "thing", Params: map[string]interface{}{}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Equal(t, []ResourceUid{"thing.default#eg.res"}, ru)
	})

	t.Run("one cycle", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res.blah}"}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		_, err = s.GetSortedResourceUids()
		assert.EqualError(t, err, "a cycle exists involving resource param placeholders")
	})

	t.Run("two unrelated", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res1": {Type: "thing", Params: map[string]interface{}{}},
				"res2": {Type: "thing", Params: map[string]interface{}{}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Equal(t, []ResourceUid{"thing.default#eg.res1", "thing.default#eg.res2"}, ru)
	})

	t.Run("two linked", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res1": {Type: "thing", Params: map[string]interface{}{"x": "${resources.res2.blah}"}},
				"res2": {Type: "thing", Params: map[string]interface{}{}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Equal(t, []ResourceUid{"thing.default#eg.res2", "thing.default#eg.res1"}, ru)
	})

	t.Run("two cycle", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res1": {Type: "thing", Params: map[string]interface{}{"x": "${resources.res2.blah}"}},
				"res2": {Type: "thing", Params: map[string]interface{}{"y": "${resources.res1.blah}"}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		_, err = s.GetSortedResourceUids()
		assert.EqualError(t, err, "a cycle exists involving resource param placeholders")
	})

	t.Run("three linked", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res1": {Type: "thing", Params: map[string]interface{}{"x": "${resources.res2.blah}"}},
				"res2": {Type: "thing", Params: map[string]interface{}{}},
				"res3": {Type: "thing", Params: map[string]interface{}{"x": "${resources.res1.blah}"}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Equal(t, []ResourceUid{"thing.default#eg.res2", "thing.default#eg.res1", "thing.default#eg.res3"}, ru)
	})

	t.Run("complex", func(t *testing.T) {
		s, err := new(State).WithWorkload(&score.Workload{
			Metadata: map[string]interface{}{"name": "eg"},
			Resources: map[string]score.Resource{
				"res1": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res2.blah} ${resources.res3.blah} ${resources.res4.blah} ${resources.res5.blah} ${resources.res6.blah}"}},
				"res2": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res3.blah} ${resources.res4.blah} ${resources.res5.blah} ${resources.res6.blah}"}},
				"res3": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res4.blah} ${resources.res5.blah} ${resources.res6.blah}"}},
				"res4": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res5.blah} ${resources.res6.blah}"}},
				"res5": {Type: "thing", Params: map[string]interface{}{"a": "${resources.res6.blah}"}},
				"res6": {Type: "thing", Params: map[string]interface{}{}},
			},
		}, nil, nil)
		assert.NoError(t, err)
		ru, err := s.GetSortedResourceUids()
		assert.NoError(t, err)
		assert.Equal(t, []ResourceUid{"thing.default#eg.res6", "thing.default#eg.res5", "thing.default#eg.res4", "thing.default#eg.res3", "thing.default#eg.res2", "thing.default#eg.res1"}, ru)
	})

}
