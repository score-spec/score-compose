package project

import (
	"context"
	"fmt"
	"slices"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
)

// ProvisionResources calls provisionResource for each workload and resource and modifies the current state and
// compose project in place.
func (s *State) ProvisionResources(
	ctx context.Context,
	providers []ConfiguredResourceProvider,
	project *compose.Project,
) error {
	if s.SharedState == nil {
		s.SharedState = make(map[string]interface{})
	}
	for _, workloadName := range s.SortedWorkloadNames() {
		// for deterministic order - sort the res names
		resNames := make([]string, 0, len(s.ScoreWorkloads[workloadName].Spec.Resources))
		for resName := range s.ScoreWorkloads[workloadName].Spec.Resources {
			resNames = append(resNames, resName)
		}
		slices.Sort(resNames)
		for _, resName := range resNames {
			resource := s.ScoreWorkloads[workloadName].Spec.Resources[resName]
			if err := s.provisionResource(ctx, providers, project, workloadName, resName, resource); err != nil {
				return fmt.Errorf("failed to provision resource '%s' in workload '%s': %w", resName, workloadName, err)
			}
		}
	}
	return nil
}

// provisionResource is the core resource provisioning procedure. It performs the main functions:
// 1. find the provider
// 2. find or initialise the starting state
// 3. call the provider
func (s *State) provisionResource(
	ctx context.Context,
	providers []ConfiguredResourceProvider,
	project *compose.Project,
	workloadName string,
	resourceName string,
	resource score.Resource,
) error {
	// Determine the resource unique id which we can use to track this instance of a resource and still find it
	// deterministically when we need to resolve outputs.
	resClass, resId := GenerateResourceClassAndId(workloadName, resourceName, &resource)
	resUid := GenerateResourceUidFromParts(resource.Type, resClass, resId)

	// Lookup the matching provider first. We can't do this without it.
	provider, ok := FindFirstMatchingProvider(providers, resource.Type, resClass, resId)
	if !ok {
		return fmt.Errorf("no provider matches resource type '%s', class '%s', id '%s'", resource.Type, resClass, resId)
	}

	// Lookup the current state in our tree so that we can pass that into the create or update function
	currentState, ok := s.Resources[resUid]
	if !ok {
		currentState = ScoreResourceState{
			Type:     resource.Type,
			Class:    resClass,
			Id:       resId,
			Provider: provider.String(),
		}
	}

	// This is an undefined behavior here. We could potentially actively delete the old resource before provisioning a
	// new one, but we don't know if the old provider is even there any more. Probably better to just fail.
	if currentState.Provider != provider.String() {
		return fmt.Errorf("the resource was previous provisioned by a different provider - please reset all state and generate again")
	}

	// Call the provider to do its changes
	if err := provider.Provision(ctx, resUid, s.SharedState, &currentState, project); err != nil {
		return err
	}
	if s.Resources == nil {
		s.Resources = make(map[string]ScoreResourceState)
	}
	s.Resources[resUid] = currentState

	return nil
}
