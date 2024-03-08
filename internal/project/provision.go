package project

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/imdario/mergo"
	score "github.com/score-spec/score-go/types"
)

// ProvisionResources calls provisionResource for each resource.
func (s *State) ProvisionResources(
	ctx context.Context,
	providers []ConfiguredResourceProvider,
	project *compose.Project,
) error {
	if s.SharedState == nil {
		s.SharedState = make(map[string]interface{})
	}

	// Collect and merge resources into one big list, the merging only applies to "shared" resources that have an id
	// field. We want to merge the annotations together, and warn if there are any 2 definitions with different
	// parameters.
	// This ensures that we only provision each resource once, and always with the same parameters, this reduces the
	// complexity of the providers.
	resolvedResources := make(map[ResourceCoordinate]score.Resource)
	for _, workloadName := range sortedKeys(s.ScoreWorkloads, strings.Compare) {
		for _, resName := range sortedKeys(s.ScoreWorkloads[workloadName].Spec.Resources, strings.Compare) {
			resource := s.ScoreWorkloads[workloadName].Spec.Resources[resName]
			coord := NewResourceCoordinate(workloadName, resName, &resource)
			if current, ok := resolvedResources[coord]; ok {
				current, err := mergeResource(current, resource)
				if err != nil {
					return fmt.Errorf("cannot provision shared resource '%s': %w", coord.Uid(), err)
				}
				resolvedResources[coord] = current
			} else {
				resolvedResources[coord] = resource
			}
		}
	}

	// Now we can go through in a stable uid order
	for _, coord := range sortedKeys(resolvedResources, func(a ResourceCoordinate, b ResourceCoordinate) int {
		return strings.Compare(a.Uid(), b.Uid())
	}) {
		rr := resolvedResources[coord]
		if err := s.provisionResource(ctx, providers, project, coord, rr); err != nil {
			return fmt.Errorf("failed to provision resource '%s': %w", coord.Uid(), err)
		}
	}

	return nil
}

// mergeResource merges 2 resources together and throws errors if they are incompatible. It returns a new copy and
// does not modify either source resource.
func mergeResource(a score.Resource, b score.Resource) (score.Resource, error) {
	out := score.Resource{
		Type:   a.Type,
		Class:  a.Class,
		Id:     b.Id,
		Params: a.Params,
	}
	if b.Params != nil {
		if a.Params != nil && !reflect.DeepEqual(b.Params, a.Params) {
			return a, fmt.Errorf("there are multiple definitions with different params")
		}
		out.Params = b.Params
	}

	for _, k := range []score.ResourceMetadata{a.Metadata, b.Metadata} {
		if k != nil {
			if out.Metadata == nil {
				out.Metadata = score.ResourceMetadata{}
			}
			if err := mergo.Merge(&out.Metadata, k, mergo.WithOverride); err != nil {
				return a, fmt.Errorf("failed to merge metadata: %w", err)
			}
		}
	}
	return out, nil
}

// provisionResource is the core resource provisioning procedure. It performs the main functions:
// 1. find the provider
// 2. find or initialise the starting state
// 3. call the provider
func (s *State) provisionResource(
	ctx context.Context,
	providers []ConfiguredResourceProvider,
	project *compose.Project,
	coord ResourceCoordinate,
	resource score.Resource,
) error {
	// Lookup the matching provider first. We can't do this without it.
	provider, ok := FindFirstMatchingProvider(providers, coord.Type(), coord.Class(), coord.Id())
	if !ok {
		return fmt.Errorf("no provider matches resource type '%s', class '%s', id '%s'", coord.Type(), coord.Class(), coord.Id())
	}

	// Lookup the current state in our tree so that we can pass that into the create or update function
	currentState, ok := s.Resources[coord.Uid()]
	if !ok {
		currentState = ScoreResourceState{
			Type:     coord.Type(),
			Class:    coord.Class(),
			Id:       coord.Id(),
			Provider: provider.ProviderUri(),
		}
	}

	// This is an undefined behavior here. We could potentially actively delete the old resource before provisioning a
	// new one, but we don't know if the old provider is even there any more. Probably better to just fail.
	if currentState.Provider != provider.ProviderUri() {
		return fmt.Errorf("the resource was previous provisioned by a different provider - please reset all state and generate again")
	}

	// Call the provider to do its changes
	if err := provider.Provision(ctx, coord.Uid(), resource, s.SharedState, &currentState, project); err != nil {
		return err
	}
	if s.Resources == nil {
		s.Resources = make(map[string]ScoreResourceState)
	}
	s.Resources[coord.Uid()] = currentState

	return nil
}

// sortedKeys returns the keys of the map, sorted by the given comparison function
func sortedKeys[M ~map[K]V, K comparable, V any](m M, cmp func(a K, b K) int) []K {
	if m == nil {
		return nil
	}
	output := make([]K, 0, len(m))
	for key := range m {
		output = append(output, key)
	}
	slices.SortFunc(output, cmp)
	return output
}
