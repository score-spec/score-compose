// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioners

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/score-spec/score-go/framework"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/project"
	"github.com/score-spec/score-compose/internal/util"
)

// Input is the set of thins passed to the provisioner implementation. It provides context, previous state, and shared
// state used by all resources.
type Input struct {
	// -- aspects from the resource declaration --

	ResourceUid      string                 `json:"resource_uid"`
	ResourceType     string                 `json:"resource_type"`
	ResourceClass    string                 `json:"resource_class"`
	ResourceId       string                 `json:"resource_id"`
	ResourceParams   map[string]interface{} `json:"resource_params"`
	ResourceMetadata map[string]interface{} `json:"resource_metadata"`

	// -- aspects of the workloads --

	// SourceWorkload is the name of the workload that first defined this resource or carries the params definition.
	SourceWorkload string `json:"source_workload"`
	// WorkloadServices is a map from workload name to the network NetworkService of another workload which defines
	// the hostname and the set of ports it exposes.
	WorkloadServices map[string]NetworkService `json:"workload_services"`

	// -- current state --

	ResourceState map[string]interface{} `json:"resource_state"`
	SharedState   map[string]interface{} `json:"shared_state"`

	// -- configuration --

	ComposeProjectName string `json:"compose_project_name"`
	MountDirectoryPath string `json:"mount_directory_path"`
}

type ServicePort struct {
	// Name is the name of the port from the workload specification
	Name string `json:"name"`
	// Port is the numeric port intended to be published
	Port int `json:"port"`
	// TargetPort is the port on the workload that hosts the actual traffic
	TargetPort int `json:"target_port"`
	// Protocol is TCP or UDP.
	Protocol score.ServicePortProtocol `json:"protocol"`
}

// NetworkService describes how to contact ports exposed by another workload
type NetworkService struct {
	ServiceName string                 `yaml:"service_name"`
	Ports       map[string]ServicePort `json:"ports"`
}

// ProvisionOutput is the output returned from a provisioner implementation.
type ProvisionOutput struct {
	ProvisionerUri       string                           `json:"-"`
	ResourceState        map[string]interface{}           `json:"resource_state"`
	ResourceOutputs      map[string]interface{}           `json:"resource_outputs"`
	SharedState          map[string]interface{}           `json:"shared_state"`
	RelativeDirectories  map[string]bool                  `json:"relative_directories"`
	RelativeFileContents map[string]*string               `json:"relative_file_contents"`
	ComposeNetworks      map[string]compose.NetworkConfig `json:"compose_networks"`
	ComposeVolumes       map[string]compose.VolumeConfig  `json:"compose_volumes"`
	ComposeServices      map[string]compose.ServiceConfig `json:"compose_services"`

	// For testing and legacy reasons, built in provisioners can set a direct lookup function
	OutputLookupFunc framework.OutputLookupFunc `json:"-"`
}

type Provisioner interface {
	Uri() string
	Match(resUid framework.ResourceUid) bool
	Provision(ctx context.Context, input *Input) (*ProvisionOutput, error)
	Class() string
	Type() string
	Params() []string
	Outputs() []string
	Description() string
}

type ephemeralProvisioner struct {
	uri         string
	matchUid    framework.ResourceUid
	provision   func(ctx context.Context, input *Input) (*ProvisionOutput, error)
	class       string
	eType       string
	params      []string
	outputs     []string
	description string
}

func (e *ephemeralProvisioner) Description() string {
	return e.description
}

func (e *ephemeralProvisioner) Uri() string {
	return e.uri
}

func (e *ephemeralProvisioner) Match(resUid framework.ResourceUid) bool {
	return resUid == e.matchUid
}
func (e *ephemeralProvisioner) Provision(ctx context.Context, input *Input) (*ProvisionOutput, error) {
	return e.provision(ctx, input)
}

func (e *ephemeralProvisioner) Outputs() []string {
	return e.outputs
}

func (e *ephemeralProvisioner) Class() string {
	return e.class
}

func (e *ephemeralProvisioner) Type() string {
	return e.eType
}

func (e *ephemeralProvisioner) Params() []string {
	return e.params
}

// NewEphemeralProvisioner is mostly used for internal testing and uses the given provisioner function to provision an exact resource.
func NewEphemeralProvisioner(uri string, matchUid framework.ResourceUid, inner func(ctx context.Context, input *Input) (*ProvisionOutput, error)) Provisioner {
	return &ephemeralProvisioner{uri: uri, matchUid: matchUid, provision: inner}
}

// ApplyToStateAndProject takes the outputs of a provisioning request and applies to the state, file tree, and docker
// compose project.
func (po *ProvisionOutput) ApplyToStateAndProject(state *project.State, resUid framework.ResourceUid, project *compose.Project) (*project.State, error) {
	slog.Debug(
		fmt.Sprintf("Provisioned resource '%s'", resUid),
		"outputs", po.ResourceOutputs,
		"#directories", len(po.RelativeDirectories),
		"#files", len(po.RelativeFileContents),
		"#volumes", len(po.ComposeVolumes),
		"#networks", len(po.ComposeNetworks),
		"#services", len(po.ComposeServices),
	)

	out := *state
	out.Resources = maps.Clone(state.Resources)

	existing, ok := out.Resources[resUid]
	if !ok {
		return nil, fmt.Errorf("failed to apply to state - unknown res uid")
	}

	// Update the provisioner string
	existing.ProvisionerUri = po.ProvisionerUri

	// State must ALWAYS be updated. If we don't get state back, we assume it's now empty.
	if po.ResourceState != nil {
		existing.State = po.ResourceState
	} else {
		existing.State = make(map[string]interface{})
	}

	// Same with outputs, it must ALWAYS be updated.
	if po.ResourceOutputs != nil {
		existing.Outputs = po.ResourceOutputs
	} else {
		existing.Outputs = make(map[string]interface{})
	}

	if po.OutputLookupFunc != nil {
		existing.OutputLookupFunc = po.OutputLookupFunc
	}

	if po.SharedState != nil {
		out.SharedState = util.PatchMap(state.SharedState, po.SharedState)
	}

	for relativePath, b := range po.RelativeDirectories {
		relativePath = filepath.Clean(relativePath)
		if !filepath.IsLocal(relativePath) {
			return nil, fmt.Errorf("failing to write non relative volume directory '%s'", relativePath)
		}
		dst := filepath.Join(state.Extras.MountsDirectory, relativePath)
		if b {
			slog.Debug(fmt.Sprintf("Ensuring mount directory '%s' exists", dst))
			if err := os.MkdirAll(dst, 0755); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, fmt.Errorf("failed to create volume directory '%s': %w", dst, err)
			}
		} else {
			slog.Debug(fmt.Sprintf("Ensuring mount directory '%s' no longer exists", dst))
			if err := os.RemoveAll(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to delete volume directory '%s': %w", dst, err)
			}
		}
	}

	for relativePath, b := range po.RelativeFileContents {
		relativePath = filepath.Clean(relativePath)
		if !filepath.IsLocal(relativePath) {
			return nil, fmt.Errorf("failing to write non relative volume directory '%s'", relativePath)
		}
		dst := filepath.Join(state.Extras.MountsDirectory, relativePath)
		if b != nil {
			slog.Debug(fmt.Sprintf("Ensuring mount file '%s' exists", dst))
			if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil && !errors.Is(err, os.ErrExist) {
				return nil, fmt.Errorf("failed to create directories for file '%s': %w", dst, err)
			}
			if err := os.WriteFile(dst, []byte(*b), 0644); err != nil {
				return nil, fmt.Errorf("failed to write file '%s': %w", dst, err)
			}
		} else {
			slog.Debug(fmt.Sprintf("Ensuring mount file '%s' no longer exists", dst))
			if err := os.Remove(dst); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to delete file '%s': %w", dst, err)
			}
		}
	}

	for networkName, network := range po.ComposeNetworks {
		if project.Networks == nil {
			project.Networks = make(compose.Networks)
		}
		project.Networks[networkName] = network
	}
	for volumeName, volume := range po.ComposeVolumes {
		if project.Volumes == nil {
			project.Volumes = make(compose.Volumes)
		}
		project.Volumes[volumeName] = volume
	}
	for serviceName, service := range po.ComposeServices {
		if project.Services == nil {
			project.Services = make(compose.Services)
		}
		project.Services[serviceName] = service
	}

	out.Resources[resUid] = existing
	return &out, nil
}

func buildWorkloadServices(state *project.State) map[string]NetworkService {
	out := make(map[string]NetworkService, len(state.Workloads))
	for workloadName, workloadState := range state.Workloads {
		// setup ports exposure
		ns := NetworkService{
			// the hostname of a workload is the <workload name>
			ServiceName: workloadName,
			Ports:       make(map[string]ServicePort),
		}
		if workloadState.Spec.Service != nil {
			for s, port := range (*workloadState.Spec.Service).Ports {
				ns.Ports[s] = ServicePort{
					Name:       s,
					Port:       port.Port,
					TargetPort: util.DerefOr(port.TargetPort, port.Port),
					Protocol:   util.DerefOr(port.Protocol, score.ServicePortProtocolTCP),
				}
			}
			// Also add unique ports using a str-converted port number - this expands compatibility by allowing users
			// to indicate the named port using its port number as a secondary name.
			for s, port := range (*workloadState.Spec.Service).Ports {
				p2 := strconv.Itoa(port.Port)
				if _, ok := ns.Ports[p2]; !ok {
					ns.Ports[p2] = ns.Ports[s]
				}
			}
		}
		out[workloadName] = ns
	}
	return out
}

func ProvisionResources(ctx context.Context, state *project.State, provisioners []Provisioner, composeProject *compose.Project) (*project.State, error) {
	out := state

	// provision in sorted order
	orderedResources, err := out.GetSortedResourceUids()
	if err != nil {
		return nil, fmt.Errorf("failed to determine sort order for provisioning: %w", err)
	}

	workloadServices := buildWorkloadServices(state)

	for _, resUid := range orderedResources {
		resState := out.Resources[resUid]
		provisionerIndex := slices.IndexFunc(provisioners, func(provisioner Provisioner) bool {
			return provisioner.Match(resUid)
		})
		if provisionerIndex < 0 {
			return nil, fmt.Errorf("resource '%s' is not supported by any provisioner", resUid)
		}
		provisioner := provisioners[provisionerIndex]
		if resState.ProvisionerUri != "" && resState.ProvisionerUri != provisioner.Uri() {
			return nil, fmt.Errorf("resource '%s' was previously provisioned by a different provider - undefined behavior", resUid)
		}

		var params map[string]interface{}
		if len(resState.Params) > 0 {
			resOutputs, err := out.GetResourceOutputForWorkload(resState.SourceWorkload)
			if err != nil {
				return nil, fmt.Errorf("failed to find resource params for resource '%s': %w", resUid, err)
			}
			sf := framework.BuildSubstitutionFunction(out.Workloads[resState.SourceWorkload].Spec.Metadata, resOutputs)
			sf = util.WrapImmediateSubstitutionFunction(sf)
			rawParams, err := framework.Substitute(resState.Params, sf)
			if err != nil {
				return nil, fmt.Errorf("failed to substitute params for resource '%s': %w", resUid, err)
			}
			params = rawParams.(map[string]interface{})
		}

		output, err := provisioner.Provision(ctx, &Input{
			ResourceUid:        string(resUid),
			ResourceType:       resUid.Type(),
			ResourceClass:      resUid.Class(),
			ResourceId:         resUid.Id(),
			ResourceParams:     params,
			ResourceMetadata:   resState.Metadata,
			ResourceState:      resState.State,
			SourceWorkload:     resState.SourceWorkload,
			WorkloadServices:   workloadServices,
			SharedState:        out.SharedState,
			ComposeProjectName: out.Extras.ComposeProjectName,
			MountDirectoryPath: out.Extras.MountsDirectory,
		})
		if err != nil {
			return nil, fmt.Errorf("resource '%s': failed to provision: %w", resUid, err)
		}

		output.ProvisionerUri = provisioner.Uri()
		out, err = output.ApplyToStateAndProject(out, resUid, composeProject)
		if err != nil {
			return nil, fmt.Errorf("resource '%s': failed to apply outputs: %w", resUid, err)
		}
	}

	return out, nil
}
