package compose

import "fmt"

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
