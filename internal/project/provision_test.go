package project

import (
	"context"
	"testing"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/ref"
)

func TestMergeResource(t *testing.T) {
	resA := score.Resource{
		Type:   "foo",
		Class:  ref.Ref("bar"),
		Id:     ref.Ref("x"),
		Params: map[string]interface{}{"a": "b"},
		Metadata: map[string]interface{}{
			"c": "d",
		},
	}
	resB := score.Resource{
		Type:   "foo",
		Class:  ref.Ref("bar"),
		Id:     ref.Ref("x"),
		Params: map[string]interface{}{"a": "b"},
		Metadata: map[string]interface{}{
			"d": "f",
		},
	}
	resC, err := mergeResource(resA, resB)
	assert.NoError(t, err)
	assert.Equal(t, score.ResourceMetadata{"c": "d"}, resA.Metadata)
	assert.Equal(t, score.ResourceMetadata{"d": "f"}, resB.Metadata)
	assert.Equal(t, score.ResourceMetadata{"c": "d", "d": "f"}, resC.Metadata)
}

func TestProvision_nils(t *testing.T) {
	state := &State{}
	assert.NoError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{}, nil))
}

func TestProvision_no_resources(t *testing.T) {
	state := &State{}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Containers: map[string]score.Container{
			"default": {
				Image: "bananas",
			},
		},
	}, nil))
	assert.NoError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{}, nil))
}

func TestProvision_no_providers(t *testing.T) {
	state := &State{}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Containers: map[string]score.Container{
			"default": {
				Image: "bananas",
			},
		},
		Resources: map[string]score.Resource{
			"something": {
				Type: "foo",
			},
		},
	}, nil))
	assert.EqualError(
		t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{}, nil),
		"failed to provision resource 'foo.default#example.something': no provider matches resource type 'foo', class 'default', id 'example.something'",
	)
}

func TestProvisionResources_basic(t *testing.T) {
	state := &State{}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Resources: map[string]score.Resource{
			"something": {
				Type: "example",
			},
		},
	}, nil))
	assert.NoError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{&MonotonicCountingProvider{Type: "example"}}, nil))
	assert.Equal(t, ScoreResourceState{
		Type:     "example",
		Class:    "default",
		Id:       "example.something",
		State:    map[string]interface{}{"value": 1},
		Outputs:  map[string]interface{}{"value": 1},
		Provider: "builtin://monotonic-number",
	}, state.Resources["example.default#example.something"])
	assert.Equal(t, map[string]interface{}{"builtin://monotonic-number_last": 1}, state.SharedState)
}

func TestProvisionResources_multiple(t *testing.T) {
	state := &State{}
	composeProject := &compose.Project{Environment: map[string]string{}}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Resources: map[string]score.Resource{
			"four": {
				Type: "example",
				Id:   ref.Ref("thing"),
			},
			"one": {
				Type: "example",
			},
			"two": {
				Type: "example",
			},
			"three": {
				Type:   "example",
				Id:     ref.Ref("thing"),
				Params: map[string]interface{}{"a": "b"},
			},
		},
	}, nil))
	assert.NoError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{
		&MonotonicCountingProvider{Type: "example"},
	}, composeProject))

	assert.Len(t, state.Resources, 3)
	numbers := make(map[int]bool)
	for _, resourceState := range state.Resources {
		assert.Equal(t, "example", resourceState.Type)
		assert.Equal(t, "default", resourceState.Class)
		numbers[resourceState.Outputs["value"].(int)] = true
	}
	assert.Equal(t, map[int]bool{1: true, 2: true, 3: true}, numbers)

	assert.Equal(t, compose.Mapping{
		"example.one": "1",
		"example.two": "2",
		"thing":       "3",
	}, composeProject.Environment)
}

func TestProvisionResources_conflict_shared_params(t *testing.T) {
	state := &State{}
	composeProject := &compose.Project{Environment: map[string]string{}}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Resources: map[string]score.Resource{
			"one": {
				Type:   "example",
				Id:     ref.Ref("thing"),
				Params: map[string]interface{}{"a": "b"},
			},
			"two": {
				Type:   "example",
				Id:     ref.Ref("thing"),
				Params: map[string]interface{}{"a": "c"},
			},
		},
	}, nil))
	assert.EqualError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{
		&MonotonicCountingProvider{Type: "example"},
	}, composeProject), "cannot provision shared resource 'example.default#thing': there are multiple definitions with different params")
}
