package project

import (
	"context"
	"strconv"
	"testing"

	compose "github.com/compose-spec/compose-go/types"
	"github.com/stretchr/testify/assert"
)

// MonotonicCountingProvider is a dummy example, used for testing some of the internal provider logic.
// The point is to have a provider which generates a monotonic number per instance of the provider.
// It stores the last value in the shared state, holds the value per resource in the local state, and returns it as
// an output.
type MonotonicCountingProvider struct {
	Type  string
	Class string
	Id    string
}

func (p *MonotonicCountingProvider) String() string {
	return "builtin://monotonic-number"
}

func (p *MonotonicCountingProvider) Provision(ctx context.Context, uid string, sharedState map[string]interface{}, state *ScoreResourceState, project *compose.Project) error {
	if state.State == nil {
		state.State = map[string]interface{}{}
	}
	if v, ok := state.State["value"].(int); ok {
		state.Outputs = map[string]interface{}{"value": v}
	} else {
		sharedKey := p.String() + "_last"
		var lastV int
		if v, ok := sharedState[sharedKey].(int); ok {
			lastV = v
		}
		newV := lastV + 1
		state.State["value"] = newV
		sharedState[sharedKey] = newV
		state.Outputs = map[string]interface{}{"value": newV}
	}
	if project != nil {
		project.Environment[state.Id] = strconv.Itoa(state.Outputs["value"].(int))
	}
	return nil
}

func (p *MonotonicCountingProvider) Match(resType, resClass, resId string) bool {
	return resType == p.Type && (p.Class == "" || (p.Class == resClass)) && (p.Id == "" || (p.Id == resId))
}

func TestMonotonicCountingProviderMatch(t *testing.T) {
	prov := &MonotonicCountingProvider{Type: "foo"}
	assert.True(t, prov.Match("foo", "default", ""))
	assert.True(t, prov.Match("foo", "default", "foo"))
	assert.True(t, prov.Match("foo", "unknown", ""))
	assert.True(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))

	prov = &MonotonicCountingProvider{Type: "foo", Class: "bar"}
	assert.True(t, prov.Match("foo", "bar", ""))
	assert.True(t, prov.Match("foo", "bar", "foo"))
	assert.False(t, prov.Match("foo", "unknown", ""))
	assert.False(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))

	prov = &MonotonicCountingProvider{Type: "foo", Class: "bar", Id: "foo"}
	assert.False(t, prov.Match("foo", "bar", ""))
	assert.True(t, prov.Match("foo", "bar", "foo"))
	assert.False(t, prov.Match("foo", "unknown", ""))
	assert.False(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))
}

func TestMonotonicCountingProviderProvision(t *testing.T) {
	prov := &MonotonicCountingProvider{}
	sharedState := make(map[string]interface{})
	resState := new(ScoreResourceState)
	resState.Id = "foo"
	composeProject := &compose.Project{Environment: map[string]string{}}
	assert.NoError(t, prov.Provision(
		context.Background(),
		"my-resource",
		sharedState,
		resState,
		composeProject,
	))
	assert.Equal(t, map[string]interface{}{"builtin://monotonic-number_last": 1}, sharedState)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.State)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.Outputs)

	assert.NoError(t, prov.Provision(
		context.Background(),
		"my-resource",
		sharedState,
		resState,
		composeProject,
	))
	assert.Equal(t, map[string]interface{}{"builtin://monotonic-number_last": 1}, sharedState)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.State)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.Outputs)

	resState2 := new(ScoreResourceState)
	resState2.Id = "bar"
	assert.NoError(t, prov.Provision(
		context.Background(),
		"my-resource2",
		sharedState,
		resState2,
		composeProject,
	))
	assert.Equal(t, map[string]interface{}{"builtin://monotonic-number_last": 2}, sharedState)
	assert.Equal(t, map[string]interface{}{"value": 2}, resState2.State)
	assert.Equal(t, map[string]interface{}{"value": 2}, resState2.Outputs)

	assert.NoError(t, prov.Provision(
		context.Background(),
		"my-resource",
		sharedState,
		resState,
		composeProject,
	))
	assert.Equal(t, map[string]interface{}{"builtin://monotonic-number_last": 2}, sharedState)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.State)
	assert.Equal(t, map[string]interface{}{"value": 1}, resState.Outputs)

	assert.Equal(t, map[string]string{
		"foo": "1",
		"bar": "2",
	}, composeProject.Environment)
}
