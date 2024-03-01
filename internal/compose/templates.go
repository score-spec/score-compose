/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
)

var (
	placeholderRegEx = regexp.MustCompile(`\$(\$|{([a-zA-Z0-9.\-_\\]+)})`)
)

// templatesContext ia an utility type that provides a context for '${...}' templates substitution
type templatesContext struct {
	meta      map[string]interface{}
	resources map[string]ResourceWithOutputs
}

// buildContext initializes a new templatesContext instance
func buildContext(metadata map[string]interface{}, resources map[string]ResourceWithOutputs) (*templatesContext, error) {
	return &templatesContext{
		meta:      maps.Clone(metadata),
		resources: maps.Clone(resources),
	}, nil
}

// Substitute replaces all matching '${...}' templates in a source string
func (ctx *templatesContext) Substitute(src string) (string, error) {
	var err error
	result := placeholderRegEx.ReplaceAllStringFunc(src, func(str string) string {
		// WORKAROUND: ReplaceAllStringFunc(..) does not provide match details
		//             https://github.com/golang/go/issues/5690
		var matches = placeholderRegEx.FindStringSubmatch(str)

		// SANITY CHECK
		if len(matches) != 3 {
			err = errors.Join(err, fmt.Errorf("could not find a proper match in previously captured string fragment"))
			return src
		}

		// EDGE CASE: Captures "$$" sequences and empty templates "${}"
		if matches[2] == "" {
			return matches[1]
		} else if matches[2] == "$" {
			return matches[2]
		}

		result, subErr := ctx.mapVar(matches[2])
		err = errors.Join(err, subErr)
		return result
	})
	return result, err
}

// MapVar replaces objects and properties references with corresponding values
// Returns an empty string if the reference can't be resolved
func (ctx *templatesContext) mapVar(ref string) (string, error) {
	subRef := strings.Replace(ref, `\.`, "\000", -1)
	parts := strings.Split(subRef, ".")
	for i, part := range parts {
		parts[i] = strings.Replace(part, "\000", ".", -1)
	}

	var resolvedValue interface{}
	var remainingParts []string

	switch parts[0] {
	case "metadata":
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid ref '%s': requires at least a metadata key to lookup", ref)
		}
		if rv, ok := ctx.meta[parts[1]]; ok {
			resolvedValue = rv
			remainingParts = parts[2:]
			for _, part := range remainingParts {
				mapV, ok := resolvedValue.(map[string]interface{})
				if !ok {
					return "", fmt.Errorf("invalid ref '%s': cannot lookup a key in %T", ref, resolvedValue)
				}
				resolvedValue, ok = mapV[part]
				if !ok {
					return "", fmt.Errorf("invalid ref '%s': key '%s' does not exist", ref, part)
				}
			}
		} else {
			return "", fmt.Errorf("invalid ref '%s': unknown metadata key '%s'", ref, parts[1])
		}
	case "resources":
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid ref '%s': requires at least a resource name to lookup", ref)
		}
		rv, ok := ctx.resources[parts[1]]
		if !ok {
			return "", fmt.Errorf("invalid ref '%s': no known resource '%s'", ref, parts[1])
		} else if len(parts) == 2 {
			return "", fmt.Errorf("invalid ref '%s': an output key is required", ref)
		} else if rv2, err := rv.LookupOutput(parts[2:]...); err != nil {
			return "", err
		} else {
			resolvedValue = rv2
		}
	default:
		return "", fmt.Errorf("invalid ref '%s': unknown reference root", ref)
	}

	if asString, ok := resolvedValue.(string); ok {
		return asString, nil
	}
	// TODO: work out how we might support other types here in the future
	raw, err := json.Marshal(resolvedValue)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
