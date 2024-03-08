package project

import (
	"context"
	"slices"

	compose "github.com/compose-spec/compose-go/v2/types"
	score "github.com/score-spec/score-go/types"
)

// ProviderResourceMatcher works out whether this provider matches the resource
type ProviderResourceMatcher interface {
	Match(resType, resClass, resId string) bool
}

type ConfiguredResourceProvider interface {
	ProviderUri() string
	ProviderResourceMatcher
	Provision(ctx context.Context, uid string, resource score.Resource, sharedState map[string]interface{}, state *ScoreResourceState, project *compose.Project) error
}

func FindFirstMatchingProvider[k ProviderResourceMatcher](providers []k, resType, resClass, resId string) (k, bool) {
	// First find the first matching provider in the list which matches our resource. We rely on the higher layers
	// sorting these providers according to their priority or provenance.
	provInd := slices.IndexFunc(providers, func(provider k) bool {
		return provider.Match(resType, resClass, resId)
	})
	var zero k
	if provInd < 0 {
		return zero, false
	}
	return providers[provInd], true
}
