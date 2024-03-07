package project

import (
	"context"
	"testing"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"
)

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
		"failed to provision resource 'something' in workload 'example': no provider matches resource type 'foo', class 'default', id 'example.something'",
	)
}

type dummyProvider struct {
}

func (d *dummyProvider) String() string {
	return "dummy"
}

func (d *dummyProvider) Match(resType, resClass, resId string) bool {
	return resType == "example"
}

func (d *dummyProvider) Provision(ctx context.Context, uid string, sharedState map[string]interface{}, state *ScoreResourceState, project *compose.Project) error {
	state.Outputs = map[string]interface{}{"a": "b"}
	sharedState["key"] = "value"
	return nil
}

func TestProvisionResource(t *testing.T) {
	state := &State{}
	assert.NoError(t, state.AppendWorkload(&score.Workload{
		Metadata: score.WorkloadMetadata{"name": "example"},
		Resources: map[string]score.Resource{
			"something": {
				Type: "example",
			},
		},
	}, nil))
	assert.NoError(t, state.ProvisionResources(context.Background(), []ConfiguredResourceProvider{new(dummyProvider)}, nil))
	assert.Equal(t, ScoreResourceState{
		Type:     "example",
		Class:    "default",
		Id:       "example.something",
		Outputs:  map[string]interface{}{"a": "b"},
		Provider: "dummy",
	}, state.Resources["example.default#example.something"])
	assert.Equal(t, map[string]interface{}{"key": "value"}, state.SharedState)
}
