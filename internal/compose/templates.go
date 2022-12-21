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
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"

	score "github.com/score-spec/score-go/types"
)

// templatesContext ia an utility type that provides a context for '${...}' templates substitution
type templatesContext map[string]string

// buildContext initializes a new templatesContext instance
func buildContext(metadata score.WorkloadMeta, resources score.ResourcesSpecs) (templatesContext, error) {
	var ctx = make(map[string]string)

	var metadataMap = make(map[string]interface{})
	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &metadataMap,
	}); err != nil {
		return nil, err
	} else {
		decoder.Decode(metadata)
		for key, val := range metadataMap {
			var ref = fmt.Sprintf("metadata.%s", key)
			if _, exists := ctx[ref]; exists {
				return nil, fmt.Errorf("ambiguous property reference '%s'", ref)
			}
			ctx[ref] = fmt.Sprintf("%v", val)
		}
	}

	for resName, res := range resources {
		ctx[fmt.Sprintf("resources.%s", resName)] = resName

		for propName, prop := range res.Properties {
			var ref = fmt.Sprintf("resources.%s.%s", resName, propName)

			var envVar string
			switch res.Type {
			case "environment":
				envVar = strings.ToUpper(propName)
			default:
				envVar = strings.ToUpper(fmt.Sprintf("%s_%s", resName, propName))
			}

			envVar = strings.Replace(envVar, "-", "_", -1)
			envVar = strings.Replace(envVar, ".", "_", -1)

			if prop.Default != nil {
				envVar += fmt.Sprintf("-%v", prop.Default)
			} else if prop.Required {
				envVar += "?err"
			}

			ctx[ref] = fmt.Sprintf("${%s}", envVar)
		}
	}

	return ctx, nil
}

// Substitute replaces all matching '${...}' templates in a source string
func (context templatesContext) Substitute(src string) string {
	return os.Expand(src, context.mapVar)
}

// MapVar replaces objects and properties references with corresponding values
// Returns an empty string if the reference can't be resolved
func (context templatesContext) mapVar(ref string) string {
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

	if res, ok := context[ref]; ok {
		return res
	}

	log.Printf("Warning: Can not resolve '%s'. Resource or property is not declared.", ref)
	return ""
}

// composeEnvVarReferencePattern defines the rule for compose environment variable references
// Possible documented references for compose v3.5:
//   - ${ENV_VAR}
//   - ${ENV_VAR?err}
//   - ${ENV_VAR-default}
var envVarPattern = regexp.MustCompile(`\$\{(\w+)(?:\-(.+?)|\?.+)?\}$`)

// ListEnvVars reports all environment variables used by templatesContext
func (context templatesContext) ListEnvVars() map[string]interface{} {
	var vars = make(map[string]interface{})
	for _, ref := range context {
		if matches := envVarPattern.FindStringSubmatch(ref); len(matches) == 3 {
			vars[matches[1]] = matches[2]
		}
	}
	return vars
}
