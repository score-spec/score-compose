package compose

import (
	"fmt"
	"strings"
)

type ResourceWithOutputs interface {
	LookupOutput(keys ...string) (interface{}, error)
}

type StaticResource struct {
	Outputs map[string]interface{}
}

func (sr *StaticResource) LookupOutput(keys ...string) (interface{}, error) {
	resolvedValue := interface{}(sr.Outputs)
	remainingKeys := keys
	for partIndex, part := range remainingKeys {
		mapV, ok := resolvedValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot lookup a key in %T", resolvedValue)
		}
		resolvedValue, ok = mapV[part]
		if !ok {
			return nil, fmt.Errorf("output '%s' does not exist", strings.Join(keys[:partIndex], "."))
		}
	}
	return resolvedValue, nil
}
