/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"

	score "github.com/score-spec/score-go/types"
)

// namedObjectMap ia an utility type that prints an object name when converted to string
type namedObjectMap map[string]string

// String converts object to string
func (r namedObjectMap) String() string {
	return fmt.Sprintf("%v", r[".name"])
}

// templatesContext ia an utility type that provides a context for '${...}' templates substitution
type templatesContext map[string]interface{}

// buildContext initializes a new templatesContext instance
func buildContext(metadata score.WorkloadMeta, resources score.ResourcesSpecs) (templatesContext, error) {
	var metadataMap = make(map[string]interface{})
	if decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &metadataMap,
	}); err == nil {
		decoder.Decode(metadata)
	} else {
		return nil, err
	}

	var resourcesMap = make(map[string]namedObjectMap)
	for resName, res := range resources {
		var resProps = namedObjectMap{
			".name": resName,
		}

		for propName, prop := range res.Properties {
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

			resProps[propName] = fmt.Sprintf("${%s}", envVar)
		}

		resourcesMap[resName] = resProps
	}

	var ctx = map[string]interface{}{
		"metadata":  metadataMap,
		"resources": resourcesMap,
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

	// NOTE: os.Expand(..) would invoke a callback function with "$" as an argument for escaped sequences.
	//       "$${abc}" is treated as "$$" pattern and "{abc}" static text.
	//       The first segment (pattern) would trigger a callback function call.
	//       By returning "$" value we would ensure that escaped sequences would remain in the source text.
	//       For example "$${abc}" would result in "${abc}" after os.Expand(..) call.
	if ref == "$" {
		return ref
	}

	var v reflect.Value
	var val interface{} = context
	for _, key := range strings.Split(ref, ".") {
		if v = reflect.ValueOf(val); v.Kind() != reflect.Map {
			return ""
		}
		if v = v.MapIndex(reflect.ValueOf(key)); !v.IsValid() {
			return ""
		}
		val = v.Interface()
	}

	return fmt.Sprintf("%v", val)
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
	for _, objMap := range context["resources"].(map[string]namedObjectMap) {
		for _, ref := range objMap {
			if matches := envVarPattern.FindStringSubmatch(ref); len(matches) == 3 {
				vars[matches[1]] = matches[2]
			}
		}
	}
	return vars
}
