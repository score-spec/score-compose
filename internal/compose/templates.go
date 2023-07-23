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
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"

	score "github.com/score-spec/score-go/types"
)

var (
	placeholderRegEx = regexp.MustCompile(`\$(\$|{([a-zA-Z0-9.\-_\[\]"'#]+)})`)
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
	return placeholderRegEx.ReplaceAllStringFunc(src, func(str string) string {
		// WORKAROUND: ReplaceAllStringFunc(..) does not provide match details
		//             https://github.com/golang/go/issues/5690
		var matches = placeholderRegEx.FindStringSubmatch(str)

		// SANITY CHECK
		if len(matches) != 3 {
			log.Printf("Error: could not find a proper match in previously captured string fragment")
			return src
		}

		// EDGE CASE: Captures "$$" sequences and empty templates "${}"
		if matches[2] == "" {
			return matches[1]
		}

		return ctx.mapVar(matches[2])
	})
}

// MapVar replaces objects and properties references with corresponding values
// Returns an empty string if the reference can't be resolved
func (ctx *templatesContext) mapVar(ref string) string {
	if ref == "" || ref == "$" {
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
