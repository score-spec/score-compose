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

	compose "github.com/compose-spec/compose-go/v2/types"

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

	// -- current state --

	ResourceState map[string]interface{} `json:"resource_state"`
	SharedState   map[string]interface{} `json:"shared_state"`

	// -- configuration --

	MountDirectoryPath string `json:"mount_directory_path"`
}

// ProvisionOutput is the output returned from a provisioner implementation.
type ProvisionOutput struct {
	ResourceState        map[string]interface{}           `json:"resource_state"`
	ResourceOutputs      map[string]interface{}           `json:"resource_outputs"`
	SharedState          map[string]interface{}           `json:"shared_state"`
	RelativeDirectories  map[string]bool                  `json:"relative_directories"`
	RelativeFileContents map[string]*string               `json:"relative_file_contents"`
	ComposeNetworks      map[string]compose.NetworkConfig `json:"compose_networks"`
	ComposeVolumes       map[string]compose.VolumeConfig  `json:"compose_volumes"`
	ComposeServices      map[string]compose.ServiceConfig `json:"compose_services"`

	// For testing and legacy reasons, built in provisioners can set a direct lookup function
	OutputLookupFunc project.OutputLookupFunc `json:"-"`
}

type Provisioner interface {
	Uri() string
	Match(resUid project.ResourceUid) bool
	Provision(ctx context.Context, input *Input) (*ProvisionOutput, error)
}

// ApplyToStateAndProject takes the outputs of a provisioning request and applies to the state, file tree, and docker
// compose project.
func (po *ProvisionOutput) ApplyToStateAndProject(state *project.State, resUid project.ResourceUid, project *compose.Project) (*project.State, error) {
	out := *state
	out.Resources = maps.Clone(state.Resources)

	existing, ok := out.Resources[resUid]
	if !ok {
		return nil, fmt.Errorf("failed to apply to state - unknown res uid")
	}

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
		dst := filepath.Join(state.MountsDirectory, relativePath)
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
		dst := filepath.Join(state.MountsDirectory, relativePath)
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

func ProvisionResources(ctx context.Context, state *project.State, provisioners []Provisioner, composeProject *compose.Project) (*project.State, error) {
	out := state

	for resUid, resState := range state.Resources {
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

		output, err := provisioner.Provision(ctx, &Input{
			ResourceUid:        string(resUid),
			ResourceType:       resUid.Type(),
			ResourceClass:      resUid.Class(),
			ResourceId:         resUid.Id(),
			ResourceParams:     resState.Params,
			ResourceMetadata:   resState.Metadata,
			ResourceState:      resState.State,
			SharedState:        out.SharedState,
			MountDirectoryPath: state.MountsDirectory,
		})
		if err != nil {
			return nil, fmt.Errorf("resource '%s': failed to provision: %w", resUid, err)
		}

		out, err = output.ApplyToStateAndProject(out, resUid, composeProject)
		if err != nil {
			return nil, fmt.Errorf("resource '%s': failed to apply outputs: %w", resUid, err)
		}
	}

	return out, nil
}
