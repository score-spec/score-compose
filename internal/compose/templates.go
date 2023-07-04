/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"

	score "github.com/score-spec/score-go/types"
)

// templatesContext ia an utility type that provides a context for '${...}' templates substitution
type templatesContext struct {
	meta      map[string]interface{}
	resources score.ResourcesSpecs

	// env map is populated dynamically with any refenced variable used by Substitute
	env map[string]interface{}
}

// buildContext initializes a new templatesContext instance
func buildContext(metadata score.WorkloadMeta, resources score.ResourcesSpecs) (*templatesContext, error) {
	var metadataMap = make(map[string]interface{})
	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &metadataMap,
	}); err != nil {
		return nil, err
	} else {
		decoder.Decode(metadata)
	}

	return &templatesContext{
		meta:      metadataMap,
		resources: resources,

		env: make(map[string]interface{}),
	}, nil
}

// Substitute replaces all matching '${...}' templates in a source string
func (ctx *templatesContext) Substitute(src string) string {
	return os.Expand(src, ctx.mapVar)
}

// MapVar replaces objects and properties references with corresponding values
// Returns an empty string if the reference can't be resolved
func (ctx *templatesContext) mapVar(ref string) string {
	if ref == "" {
		return ""
	}

	// NOTE: os.Expand(..) would invoke a callback function with "$" as an argument for escaped sequences.
	//       "$${abc}" is treated as "$$" pattern and "{abc}" static text.
	//       The first segment (pattern) would trigger a callback function call.
	//       By returning "$" value we would ensure that escaped sequences would remain in the source text.
	//       For example "$${abc}" would result in "${abc}" after os.Expand(..) call.
	if ref == "$" {
		return ref
	}

	var segments = strings.SplitN(ref, ".", 2)
	switch segments[0] {
	case "metadata":
		if len(segments) == 2 {
			if val, exists := ctx.meta[segments[1]]; exists {
				return fmt.Sprintf("%v", val)
			}
		}

	case "resources":
		if len(segments) == 2 {
			segments = strings.SplitN(segments[1], ".", 2)
			var resName = segments[0]
			if res, exists := ctx.resources[resName]; exists {
				if len(segments) == 1 {
					return resName
				} else {
					var propName = segments[1]

					var envVar string
					switch res.Type {
					case "environment":
						envVar = strings.ToUpper(propName)
					default:
						envVar = strings.ToUpper(fmt.Sprintf("%s_%s", resName, propName))
					}
					envVar = strings.Replace(envVar, "-", "_", -1)
					envVar = strings.Replace(envVar, ".", "_", -1)

					ctx.env[envVar] = ""
					return fmt.Sprintf("${%s}", envVar)
				}
			}
		}
	}

	log.Printf("Warning: Can not resolve '%s' reference.", ref)
	return ""
}

// ListEnvVars reports all environment variables used by templatesContext
func (ctx *templatesContext) ListEnvVars() map[string]interface{} {
	return ctx.env
}
