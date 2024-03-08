package project

import (
	"fmt"

	score "github.com/score-spec/score-go/types"
)

// State is the mega-structure that contains the state of our workload specifications and resources.
// Score specs are added to this structure and it stores the current resource set.
type State struct {
	ScoreWorkloads map[string]ScoreWorkloadState `yaml:"score_workloads"`
	Resources      map[string]ScoreResourceState `yaml:"resources"`
	SharedState    map[string]interface{}        `yaml:"shared_state"`
}

type ScoreWorkloadState struct {
	// Spec is the final score spec after all overrides and images have been set. This is a validated score file.
	Spec score.Workload `yaml:"spec"`
	// File is the source score file it known.
	File *string `yaml:"file,omitempty"`
}

type ScoreResourceState struct {
	// Type is the resource type.
	Type string `yaml:"type"`
	// Class is the resource class or 'default' if not provided.
	Class string `yaml:"class"`
	// Id is the generated id for the resource, either <workload>.<resName> or <shared>.<id>. This is tracked so that
	// we can deduplicate and work out where a resource came from.
	Id string `yaml:"id"`
	// Provider is the resolved provider uri that should be found in the providers file. This is tracked so that
	// we identify which provider was used for a particular instance of the resource.
	Provider string `yaml:"provider"`
	// State is the internal state local to this resource. It will be persisted to disk when possible.
	State map[string]interface{} `yaml:"state"`

	// Outputs is the current set of outputs for the resource. This is the output of calling the provider. It doesn't
	// get persisted to disk.
	Outputs map[string]interface{} `yaml:"-"`
	// OutputLookupFunc is function that allows certain in-process providers to defer any output generation. If this is
	// not provided, it will fallback to using what's in the outputs.
	OutputLookupFunc OutputLookupFunc `yaml:"-"`
}

func (s *State) AppendWorkload(spec *score.Workload, filePath *string) error {
	if s.ScoreWorkloads == nil {
		s.ScoreWorkloads = map[string]ScoreWorkloadState{}
	}
	s.ScoreWorkloads[spec.Metadata["name"].(string)] = ScoreWorkloadState{
		Spec: *spec,
		File: filePath,
	}
	return nil
}

func (s *State) CollectResourceOutputs(spec *score.Workload) (map[string]OutputLookupFunc, error) {
	workloadName := spec.Metadata["name"].(string)
	output := make(map[string]OutputLookupFunc)
	for resName, resource := range spec.Resources {
		coordinate := NewResourceCoordinate(workloadName, resName, &resource)
		state, ok := s.Resources[coordinate.Uid()]
		if ok {
			output[resName] = state.LookupOutput
		} else {
			return nil, fmt.Errorf("no resource provisioned for '%s'", coordinate.Uid())
		}
	}
	return output, nil
}

func (s *ScoreResourceState) LookupOutput(keys ...string) (interface{}, error) {
	if s.OutputLookupFunc != nil {
		return s.OutputLookupFunc(keys...)
	}
	var resolvedValue interface{}
	resolvedValue = s.Outputs
	for _, k := range keys {
		mapV, ok := resolvedValue.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("cannot lookup key '%s', context is not a map", k)
		}
		resolvedValue, ok = mapV[k]
		if !ok {
			return "", fmt.Errorf("key '%s' not found", k)
		}
	}
	return resolvedValue, nil
}

type OutputLookupFunc func(keys ...string) (interface{}, error)
