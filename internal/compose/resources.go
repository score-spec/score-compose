package compose

import (
	"fmt"
	"log/slog"
	"slices"

	score "github.com/score-spec/score-go/types"
)

// ResourceWithOutputs is an interface that resource implementations in the future may provide.
// The keys here are the parts of a .-separated path traversal down a tree to return some data from the outputs of
// the provisioned resource. If an error occurs looking up the output, an error should be thrown.
// nil is a valid result since some resources may return null in their outputs.
type ResourceWithOutputs interface {
	LookupOutput(keys ...string) (interface{}, error)
}

type resourceWithStaticOutputs map[string]interface{}

func (r resourceWithStaticOutputs) LookupOutput(keys ...string) (interface{}, error) {
	var resolvedValue interface{}
	resolvedValue = (map[string]interface{})(r)
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

var _ ResourceWithOutputs = (resourceWithStaticOutputs)(nil)

func GenerateOldStyleResourceOutputs(spec *score.Workload) (map[string]ResourceWithOutputs, *EnvVarTracker, error) {
	envVarTracker := new(EnvVarTracker)

	resources := make(map[string]ResourceWithOutputs)
	// The first thing we must do is validate or create the resources this workload depends on.
	// NOTE: this will soon be replaced by a much more sophisticated resource provisioning system!
	for resourceName, resourceSpec := range spec.Resources {
		resClass := "default"
		if resourceSpec.Class != nil {
			resClass = *resourceSpec.Class
		}
		if resourceSpec.Type == "environment" && resClass == "default" {
			resources[resourceName] = envVarTracker
		} else if resourceSpec.Type == "volume" && resClass == "default" {
			resources[resourceName] = resourceWithStaticOutputs{}
		} else {
			slog.Warn(fmt.Sprintf("resources.%s: '%s.%s' is not directly supported in score-compose, references will be converted to environment variables", resourceName, resourceSpec.Type, resClass))
			resources[resourceName] = envVarTracker.GenerateResource(resourceName)
		}
	}

	return resources, envVarTracker, nil
}

func GenerateResourceOutputs(spec *score.Workload, tracker *EnvVarTracker, fallbackGlobs []string) (map[string]ResourceWithOutputs, error) {
	workloadName := spec.Metadata["name"].(string)
	canFallback := func(t, c string) bool {
		return slices.Contains(fallbackGlobs, "*") || slices.Contains(fallbackGlobs, fmt.Sprintf("%s.*", t)) || slices.Contains(fallbackGlobs, fmt.Sprintf("%s.%s", t, c))
	}
	resources := make(map[string]ResourceWithOutputs)
	// The first thing we must do is validate or create the resources this workload depends on.
	// NOTE: this will soon be replaced by a much more sophisticated resource provisioning system!
	for resourceName, resourceSpec := range spec.Resources {
		resClass := "default"
		if resourceSpec.Class != nil {
			resClass = *resourceSpec.Class
		}
		if resourceSpec.Type == "environment" && resClass == "default" {
			resources[resourceName] = tracker
		} else if canFallback(resourceSpec.Type, resClass) {
			slog.Info(fmt.Sprintf("Using environment variable fallback for resource '%s' in workload '%s'", resourceName, workloadName))
			resources[resourceName] = tracker.GenerateResource(workloadName + "_" + resourceName)
		} else {
			return nil, fmt.Errorf("resources.%s '%s.%s' is not supported in score-compose", resourceName, resourceSpec.Type, resClass)
		}
	}
	return resources, nil
}
