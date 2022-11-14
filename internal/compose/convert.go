/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"
)

// ConvertSpec converts SCORE specification into docker-compose configuration.
func ConvertSpec(spec *score.WorkloadSpec) (*compose.Project, ExternalVariables, error) {

	for _, cSpec := range spec.Containers {
		var externalVars = ExternalVariables(resourcesMap(spec.Resources).listVars())
		var env = make(compose.MappingWithEquals, len(cSpec.Variables))
		for key, val := range cSpec.Variables {
			var envVarVal = os.Expand(val, resourcesMap(spec.Resources).mapVar)
			env[key] = &envVarVal
		}

		var dependsOn = make(compose.DependsOnConfig, len(spec.Resources))
		for name, res := range spec.Resources {
			if res.Type != "environment" && res.Type != "volume" {
				dependsOn[name] = compose.ServiceDependency{Condition: "service_started"}
			}
		}

		var ports []compose.ServicePortConfig
		if len(spec.Service.Ports) > 0 {
			ports = []compose.ServicePortConfig{}
			for _, pSpec := range spec.Service.Ports {
				var pubPort = fmt.Sprintf("%v", pSpec.Port)
				var tgtPort = pSpec.TargetPort
				if pSpec.TargetPort == 0 {
					tgtPort = pSpec.Port
				}
				ports = append(ports, compose.ServicePortConfig{
					Published: pubPort,
					Target:    uint32(tgtPort),
					Protocol:  pSpec.Protocol,
				})
			}
		}

		var volumes []compose.ServiceVolumeConfig
		if len(cSpec.Volumes) > 0 {
			volumes = make([]compose.ServiceVolumeConfig, len(cSpec.Volumes))
			for idx, vol := range cSpec.Volumes {
				if vol.Path != "" {
					return nil, nil, fmt.Errorf("can't mount named volume with sub path '%s': %w", vol.Path, errors.New("not supported"))
				}
				volumes[idx] = compose.ServiceVolumeConfig{
					Type:     "volume",
					Source:   resourceRefRegex.ReplaceAllString(vol.Source, "$1"),
					Target:   vol.Target,
					ReadOnly: vol.ReadOnly,
				}
			}
		}

		var svc = compose.ServiceConfig{
			Name:        spec.Metadata.Name,
			Image:       cSpec.Image,
			Entrypoint:  cSpec.Command,
			Command:     cSpec.Args,
			Environment: env,
			DependsOn:   dependsOn,
			Ports:       ports,
			Volumes:     volumes,
		}

		var proj = compose.Project{
			Services: compose.Services{
				svc,
			},
		}

		// NOTE: Only one container per workload can be defined for compose.
		//       All other containers will be ignored by this tool.
		return &proj, externalVars, nil
	}

	return nil, nil, errors.New("workload does not have any containers to convert into a compose service")
}

// resourceRefRegex extracts the resource ID from the resource reference: '${resources.RESOURCE_ID}'
var resourceRefRegex = regexp.MustCompile(`\${resources\.(.+)}`)

// resourcesMap is an internal utility type to group some helper methods.
type resourcesMap map[string]score.ResourceSpec

// listResourcesVars lists all available environment variables based on the declared resources properties.
func (r resourcesMap) listVars() map[string]interface{} {
	var vars = make(map[string]interface{})
	for resName, res := range r {
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

			vars[envVar] = prop.Default
		}
	}
	return vars
}

// mapResourceVar maps resources properties references.
// Returns an empty string if the reference can't be resolved.
func (r resourcesMap) mapVar(ref string) string {
	if ref == "$" {
		return ref
	}

	var segments = strings.SplitN(ref, ".", 3)
	if segments[0] != "resources" || len(segments) != 3 {
		return ""
	}

	var resName = segments[1]
	var propName = segments[2]
	if res, ok := r[resName]; ok {
		if prop, ok := res.Properties[propName]; ok {
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

			return fmt.Sprintf("${%s}", envVar)
		}
	}

	return ""
}
