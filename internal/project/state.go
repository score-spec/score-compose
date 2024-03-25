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

package project

import (
	"fmt"
	"maps"
	"reflect"
	"slices"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
)

// State is the mega-structure that contains the state of our workload specifications and resources.
// Score specs are added to this structure and it stores the current resource set.
type State struct {
	Workloads          map[string]ScoreWorkloadState      `yaml:"workloads"`
	Resources          map[ResourceUid]ScoreResourceState `yaml:"resources"`
	SharedState        map[string]interface{}             `yaml:"shared_state"`
	ComposeProjectName string                             `yaml:"compose_project"`
	MountsDirectory    string                             `yaml:"mounts_directory"`
}

type ScoreWorkloadState struct {
	// Spec is the final score spec after all overrides and images have been set. This is a validated score file.
	Spec score.Workload `yaml:"spec"`
	// File is the source score file if known.
	File *string `yaml:"file,omitempty"`
	// BuildConfigs is a stored set of container build configs for this workload. Any known container should inherit
	// the appropriate config when being converted.
	BuildConfigs map[string]compose.BuildConfig `yaml:"build_configs,omitempty"`
}

type ScoreResourceState struct {
	// Type is the resource type.
	Type string `yaml:"type"`
	// Class is the resource class or 'default' if not provided.
	Class string `yaml:"class"`
	// Id is the generated id for the resource, either <workload>.<resName> or <shared>.<id>. This is tracked so that
	// we can deduplicate and work out where a resource came from.
	Id string `yaml:"id"`

	Metadata map[string]interface{} `yaml:"metadata"`
	Params   map[string]interface{} `yaml:"params"`
	// SourceWorkload holds the workload name that had the best definition for this resource. "best" is either the
	// first one or the one with params defined.
	SourceWorkload string `yaml:"source_workload"`

	// ProvisionerUri is the resolved provisioner uri that should be found in the config. This is tracked so that
	// we identify which provisioner was used for a particular instance of the resource.
	ProvisionerUri string `yaml:"provisioner"`
	// State is the internal state local to this resource. It will be persisted to disk when possible.
	State map[string]interface{} `yaml:"state"`

	// Outputs is the current set of outputs for the resource. This is the output of calling the provider. It doesn't
	// get persisted to disk.
	Outputs map[string]interface{} `yaml:"-"`
	// OutputLookupFunc is function that allows certain in-process providers to defer any output generation. If this is
	// not provided, it will fallback to using what's in the outputs.
	OutputLookupFunc OutputLookupFunc `yaml:"-"`
}

type OutputLookupFunc func(keys ...string) (interface{}, error)

// WithWorkload returns a new copy of State with the workload added, if the workload already exists with the same name
// then it will be replaced.
// This is not a deep copy, but any writes are executed in a copy-on-write manner to avoid modifying the source.
func (s *State) WithWorkload(spec *score.Workload, filePath *string, contexts map[string]compose.BuildConfig) (*State, error) {
	out := *s
	if s.Workloads == nil {
		out.Workloads = make(map[string]ScoreWorkloadState)
	} else {
		out.Workloads = maps.Clone(s.Workloads)
	}
	out.Workloads[spec.Metadata["name"].(string)] = ScoreWorkloadState{
		Spec:         *spec,
		File:         filePath,
		BuildConfigs: maps.Clone(contexts),
	}
	return &out, nil
}

// WithPrimedResources returns a new copy of State with all workload resources resolved to at least their initial type,
// class and id. New resources will have an empty provider set. Existing resources will not be touched.
// This is not a deep copy, but any writes are executed in a copy-on-write manner to avoid modifying the source.
func (s *State) WithPrimedResources() (*State, error) {
	out := *s
	if s.Resources == nil {
		out.Resources = make(map[ResourceUid]ScoreResourceState)
	} else {
		out.Resources = maps.Clone(s.Resources)
	}

	primedResourceUids := make(map[ResourceUid]bool)
	for workloadName, workload := range s.Workloads {
		for resName, res := range workload.Spec.Resources {
			resUid := NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)
			if existing, ok := out.Resources[resUid]; !ok {
				out.Resources[resUid] = ScoreResourceState{
					Type:           resUid.Type(),
					Class:          resUid.Class(),
					Id:             resUid.Id(),
					Metadata:       res.Metadata,
					Params:         res.Params,
					SourceWorkload: workloadName,
					State:          map[string]interface{}{},
				}
				primedResourceUids[resUid] = true
			} else if !primedResourceUids[resUid] {
				existing.Metadata = res.Metadata
				existing.Params = res.Params
				existing.SourceWorkload = workloadName
				out.Resources[resUid] = existing
				primedResourceUids[resUid] = true
			} else {
				// multiple definitions of the same shared resource, let's check for conflicting params and metadata
				if res.Params != nil {
					if existing.Params != nil && !reflect.DeepEqual(existing.Params, res.Params) {
						return nil, fmt.Errorf("resource '%s': multiple definitions with different params", resUid)
					}
					existing.Params = res.Params
					existing.SourceWorkload = workloadName
				}
				if res.Metadata != nil {
					if existing.Metadata != nil && !reflect.DeepEqual(existing.Metadata, res.Metadata) {
						return nil, fmt.Errorf("resource '%s': multiple definitions with different metadata", resUid)
					}
					existing.Metadata = res.Metadata
				}
				out.Resources[resUid] = existing
			}
		}
	}
	return &out, nil
}

func (s *State) getResourceDependencies(workloadName, resName string, ru ResourceUid) (map[ResourceUid]bool, error) {
	outMap := make(map[ResourceUid]bool)
	res := s.Workloads[workloadName].Spec.Resources[resName]
	if res.Params == nil {
		return nil, nil
	}
	_, err := Substitute((map[string]interface{})(res.Params), func(ref string) (string, error) {
		parts := SplitRefParts(ref)
		if len(parts) > 1 && parts[0] == "resources" {
			rr, ok := s.Workloads[workloadName].Spec.Resources[parts[1]]
			if ok {
				outMap[NewResourceUid(workloadName, parts[1], rr.Type, rr.Class, rr.Id)] = true
			} else {
				return ref, fmt.Errorf("refers to unknown resource names '%s'", parts[1])
			}
		}
		return ref, nil
	})
	if err != nil {
		return nil, fmt.Errorf("workload '%s' resource '%s': %w", workloadName, resName, err)
	}
	return outMap, nil
}

// GetSortedResourceUids returns a topological sorting of the resource uids. The output order is deterministic and
// ensures that any resource output placeholder statements are strictly evaluated after their referenced resource.
// If cycles are detected an error will be thrown.
func (s *State) GetSortedResourceUids() ([]ResourceUid, error) {

	// We're implementing Kahn's algorithm (https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm).
	nodesWithNoIncomingEdges := make(map[ResourceUid]bool)
	incomingEdges := make(map[ResourceUid]map[ResourceUid]bool, len(s.Resources))

	// we must first gather all the dependencies of each resource
	// many resources won't have dependencies and they go straight into the no-incoming-edges set
	for workloadName, workload := range s.Workloads {
		for resName, res := range workload.Spec.Resources {
			resUid := NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)
			deps, err := s.getResourceDependencies(workloadName, resName, resUid)
			if err != nil {
				return nil, err
			}
			if len(deps) == 0 {
				nodesWithNoIncomingEdges[resUid] = true
			} else {
				incomingEdges[resUid] = deps
			}
		}
	}

	// set up the output list
	output := make([]ResourceUid, 0, len(nodesWithNoIncomingEdges)+len(incomingEdges))

	// now iterate through the nodes with no incoming edges and subtract them from the
	for len(nodesWithNoIncomingEdges) > 0 {

		// to get a stable set, we grab whatever is on the set and convert it to a sorted list
		subset := make([]ResourceUid, 0, len(nodesWithNoIncomingEdges))
		for uid, _ := range nodesWithNoIncomingEdges {
			subset = append(subset, uid)
		}
		clear(nodesWithNoIncomingEdges)
		slices.Sort(subset)

		// we can bulk append the subset to the output
		output = append(output, subset...)

		// remove a node from the no-incoming edges set
		for _, fromUid := range subset {
			// now find any nodes that had an edge going from this node to them
			for toUid, m := range incomingEdges {
				if m[fromUid] {
					// and remove the edge
					delete(m, fromUid)
					// if there are no incoming edges, then move it to the no-incoming-edges set
					if len(m) == 0 {
						delete(incomingEdges, toUid)
						nodesWithNoIncomingEdges[toUid] = true
					}
				}
			}
		}
	}
	// if we make no progress then there are cycles
	if len(incomingEdges) > 0 {
		return nil, fmt.Errorf("a cycle exists involving resource param placeholders")
	}
	return output, nil
}

func (s *State) GetSubstitutionContextForWorkload(workloadName string) (map[string]interface{}, error) {
	resourceOutputs, err := s.GetResourceOutputForWorkload(workloadName)
	if err != nil {
		return nil, err
	}
	metadata := s.Workloads[workloadName].Spec.Metadata
	return map[string]interface{}{
		"metadata":  (map[string]interface{})(metadata),
		"resources": resourceOutputs,
	}, nil
}

// GetResourceOutputForWorkload returns an output function per resource name in the given workload. This is for
// passing into the compose translation context to resolve placeholder references.
// This does not modify the state.
func (s *State) GetResourceOutputForWorkload(workloadName string) (map[string]OutputLookupFunc, error) {
	workload, ok := s.Workloads[workloadName]
	if !ok {
		return nil, fmt.Errorf("workload '%s': does not exist", workloadName)
	}
	out := make(map[string]OutputLookupFunc)

	for resName, res := range workload.Spec.Resources {
		resUid := NewResourceUid(workloadName, resName, res.Type, res.Class, res.Id)
		state, ok := s.Resources[resUid]
		if !ok {
			return nil, fmt.Errorf("workload '%s': resource '%s' (%s) is not primed", workloadName, resName, resUid)
		}
		out[resName] = state.OutputLookup
	}
	return out, nil
}

// OutputLookup is a function which can traverse an outputs tree to find a resulting key, this defers to the embedded
// output function if it exists.
func (s *ScoreResourceState) OutputLookup(keys ...string) (interface{}, error) {
	if s.OutputLookupFunc != nil {
		return s.OutputLookupFunc(keys...)
	} else if len(keys) == 0 {
		return nil, fmt.Errorf("at least one lookup key is required")
	}
	var resolvedValue interface{}
	resolvedValue = s.Outputs
	for _, k := range keys {
		ok := resolvedValue != nil
		if ok {
			var mapV map[string]interface{}
			mapV, ok = resolvedValue.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("cannot lookup key '%s', context is not a map", k)
			}
			resolvedValue, ok = mapV[k]
		}
		if !ok {
			return "", fmt.Errorf("key '%s' not found", k)
		}
	}
	return resolvedValue, nil
}
