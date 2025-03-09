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

package templateprov

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/compose-spec/compose-go/v2/loader"
	compose "github.com/compose-spec/compose-go/v2/types"
	"github.com/mitchellh/mapstructure"
	"github.com/score-spec/score-go/framework"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/util"
)

// Provisioner is the decoded template provisioner.
// A template provisioner provisions a resource by evaluating a series of Go text/templates that have access to some
// input parameters, previous state, and utility functions. Each parameter is expected to return a JSON object.
type Provisioner struct {
	ProvisionerUri string  `yaml:"uri"`
	ResType        string  `yaml:"type"`
	ResClass       *string `yaml:"class,omitempty"`
	ResId          *string `yaml:"id,omitempty"`
	ResDescription string  `yaml:"description,omitempty"`

	// The InitTemplate is always evaluated first, it is used as temporary or working set data that may be needed in the
	// later templates. It has access to the resource inputs and previous state.
	InitTemplate string `yaml:"init,omitempty"`
	// StateTemplate generates the new state of the resource based on the init and previous state.
	StateTemplate string `yaml:"state,omitempty"`
	// SharedStateTemplate generates modifications to the shared state, based on the init and current state.
	SharedStateTemplate string `yaml:"shared,omitempty"`
	// OutputsTemplate generates the outputs of the resource, based on the init and current state.
	OutputsTemplate string `yaml:"outputs,omitempty"`

	// RelativeDirectoriesTemplate generates a set of directories to create (true) or delete (false). These may then
	// be used in mounting requests for volumes or service mounts.
	RelativeDirectoriesTemplate string `yaml:"directories,omitempty"`
	// RelativeFilesTemplate generates a set of file contents to write (non nil) or delete (nil) from the mounts
	// directory. These will then be used during service bind mounting.
	RelativeFilesTemplate string `yaml:"files,omitempty"`

	// ComposeNetworksTemplate generates a set of networks to add to the compose project. These will replace any with
	// the same name already.
	ComposeNetworksTemplate string `yaml:"networks,omitempty"`
	// ComposeVolumesTemplate generates a set of volumes to add to the compose project. These will replace any with
	// the same name already.
	ComposeVolumesTemplate string `yaml:"volumes,omitempty"`
	// ComposeServicesTemplate generates a set of services to add to the compose project. These will replace any with
	// the same name already.
	ComposeServicesTemplate string `yaml:"services,omitempty"`

	// InfoLogsTemplate allows the provisioner to return informational messages for the user which may help connecting or
	// testing the provisioned resource
	InfoLogsTemplate string `yaml:"info_logs,omitempty"`

	// SupportedParams is a list of parameters that the provisioner expects to be passed in.
	SupportedParams []string `yaml:"supported_params,omitempty"`

	// ExpectedOutputs is a list of expected outputs that the provisioner should return.
	ExpectedOutputs []string `yaml:"expected_outputs,omitempty"`
}

func Parse(raw map[string]interface{}) (*Provisioner, error) {
	p := new(Provisioner)
	intermediate, _ := yaml.Marshal(raw)
	dec := yaml.NewDecoder(bytes.NewReader(intermediate))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, err
	}
	if p.ProvisionerUri == "" {
		return nil, fmt.Errorf("uri not set")
	} else if p.ResType == "" {
		return nil, fmt.Errorf("type not set")
	}
	return p, nil
}

func (p *Provisioner) Uri() string {
	return p.ProvisionerUri
}

func (p *Provisioner) Description() string {
	return p.ResDescription
}

func (p *Provisioner) Class() string {
	if p.ResClass == nil {
		return "(any)"
	}
	return *p.ResClass
}

func (p *Provisioner) Type() string {
	return p.ResType
}

func (p *Provisioner) Params() []string {
	if p.SupportedParams == nil {
		return []string{}
	}
	params := make([]string, len(p.SupportedParams))
	copy(params, p.SupportedParams)
	slices.Sort(params)
	return params
}

func (p *Provisioner) Outputs() []string {
	if p.ExpectedOutputs == nil {
		return []string{}
	}
	outputs := make([]string, len(p.ExpectedOutputs))
	copy(outputs, p.ExpectedOutputs)
	slices.Sort(outputs)
	return outputs
}

func (p *Provisioner) Match(resUid framework.ResourceUid) bool {
	if resUid.Type() != p.ResType {
		return false
	} else if p.ResClass != nil && resUid.Class() != *p.ResClass {
		return false
	} else if p.ResId != nil && resUid.Id() != *p.ResId {
		return false
	}
	return true
}

func renderTemplateAndDecode(raw string, data interface{}, out interface{}, withComposeExtensions bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	prepared, err := template.New("").Funcs(sprig.FuncMap()).Parse(raw)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	buff := new(bytes.Buffer)
	if err := prepared.Execute(buff, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	buffContents := buff.String()
	if strings.TrimSpace(buff.String()) == "" {
		return nil
	}
	var intermediate interface{}
	if err := yaml.Unmarshal([]byte(buffContents), &intermediate); err != nil {
		slog.Debug(fmt.Sprintf("template output was '%s' from template '%s'", buffContents, raw))
		return fmt.Errorf("failed to decode output: %w", err)
	}
	if withComposeExtensions {
		err = loader.Transform(intermediate, &out)
	} else {
		err = mapstructure.Decode(intermediate, &out)
	}
	if err != nil {
		return fmt.Errorf("failed to decode output: %w", err)
	}
	return nil
}

// Data is the structure sent to each template during rendering.
type Data struct {
	Uid      string
	Type     string
	Class    string
	Id       string
	Params   map[string]interface{}
	Metadata map[string]interface{}

	Init   map[string]interface{}
	State  map[string]interface{}
	Shared map[string]interface{}

	SourceWorkload   string
	WorkloadServices map[string]provisioners.NetworkService

	ComposeProjectName string
	MountsDirectory    string
}

func (p *Provisioner) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	out := &provisioners.ProvisionOutput{}

	// The data payload that gets passed into each template
	data := Data{
		Uid:                input.ResourceUid,
		Type:               input.ResourceType,
		Class:              input.ResourceClass,
		Id:                 input.ResourceId,
		Params:             input.ResourceParams,
		Metadata:           input.ResourceMetadata,
		State:              input.ResourceState,
		Shared:             input.SharedState,
		SourceWorkload:     input.SourceWorkload,
		WorkloadServices:   input.WorkloadServices,
		ComposeProjectName: input.ComposeProjectName,
		MountsDirectory:    input.MountDirectoryPath,
	}

	init := make(map[string]interface{})
	if err := renderTemplateAndDecode(p.InitTemplate, &data, &init, false); err != nil {
		return nil, fmt.Errorf("init template failed: %w", err)
	}
	data.Init = init

	out.ResourceState = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.StateTemplate, &data, &out.ResourceState, false); err != nil {
		return nil, fmt.Errorf("state template failed: %w", err)
	}
	data.State = out.ResourceState

	out.SharedState = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.SharedStateTemplate, &data, &out.SharedState, false); err != nil {
		return nil, fmt.Errorf("shared template failed: %w", err)
	}
	data.Shared = util.PatchMap(data.Shared, out.SharedState)

	out.ResourceOutputs = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.OutputsTemplate, &data, &out.ResourceOutputs, false); err != nil {
		return nil, fmt.Errorf("outputs template failed: %w", err)
	}

	out.RelativeDirectories = make(map[string]bool)
	if err := renderTemplateAndDecode(p.RelativeDirectoriesTemplate, &data, &out.RelativeDirectories, false); err != nil {
		return nil, fmt.Errorf("directories template failed: %w", err)
	}

	out.RelativeFileContents = make(map[string]*string)
	if err := renderTemplateAndDecode(p.RelativeFilesTemplate, &data, &out.RelativeFileContents, false); err != nil {
		return nil, fmt.Errorf("files template failed: %w", err)
	}

	out.ComposeNetworks = make(map[string]compose.NetworkConfig)
	if err := renderTemplateAndDecode(p.ComposeNetworksTemplate, &data, &out.ComposeNetworks, true); err != nil {
		return nil, fmt.Errorf("networks template failed: %w", err)
	}

	out.ComposeServices = make(map[string]compose.ServiceConfig)
	if err := renderTemplateAndDecode(p.ComposeServicesTemplate, &data, &out.ComposeServices, true); err != nil {
		return nil, fmt.Errorf("services template failed: %w", err)
	}

	out.ComposeVolumes = make(map[string]compose.VolumeConfig)
	if err := renderTemplateAndDecode(p.ComposeVolumesTemplate, &data, &out.ComposeVolumes, true); err != nil {
		return nil, fmt.Errorf("volumes template failed: %w", err)
	}

	var infoLogs []string
	if err := renderTemplateAndDecode(p.InfoLogsTemplate, &data, &infoLogs, false); err != nil {
		return nil, fmt.Errorf("info logs template failed: %w", err)
	}
	for _, log := range infoLogs {
		slog.Info(fmt.Sprintf("%s: %s", input.ResourceUid, log))
	}

	return out, nil
}

var _ provisioners.Provisioner = (*Provisioner)(nil)
