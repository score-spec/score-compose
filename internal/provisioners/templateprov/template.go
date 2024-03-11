package templateprov

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	compose "github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/project"
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

func (p *Provisioner) Match(resUid project.ResourceUid) bool {
	if resUid.Type() != p.ResType {
		return false
	} else if p.ResClass != nil && resUid.Class() != *p.ResClass {
		return false
	} else if p.ResId != nil && resUid.Id() != *p.ResId {
		return false
	}
	return true
}

func renderTemplateAndDecode(raw string, data interface{}, out interface{}) error {
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
	if strings.TrimSpace(buffContents) == "" {
		return nil
	}
	dec := yaml.NewDecoder(strings.NewReader(buffContents))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		slog.Debug(fmt.Sprintf("template output was '%s' from template '%s'", buffContents, raw))
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

	MountsDirectory string
}

func (p *Provisioner) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	out := &provisioners.ProvisionOutput{}

	// The data payload that gets passed into each template
	data := Data{
		Uid:             input.ResourceUid,
		Type:            input.ResourceType,
		Class:           input.ResourceClass,
		Id:              input.ResourceId,
		Params:          input.ResourceParams,
		Metadata:        input.ResourceMetadata,
		State:           input.ResourceState,
		Shared:          input.SharedState,
		MountsDirectory: input.MountDirectoryPath,
	}

	init := make(map[string]interface{})
	if err := renderTemplateAndDecode(p.InitTemplate, &data, &init); err != nil {
		return nil, fmt.Errorf("init template failed: %w", err)
	}
	data.Init = init

	out.ResourceState = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.StateTemplate, &data, &out.ResourceState); err != nil {
		return nil, fmt.Errorf("state template failed: %w", err)
	}
	data.State = out.ResourceState

	out.SharedState = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.SharedStateTemplate, &data, &out.SharedState); err != nil {
		return nil, fmt.Errorf("shared template failed: %w", err)
	}
	data.Shared = util.PatchMap(data.Shared, out.SharedState)

	out.ResourceOutputs = make(map[string]interface{})
	if err := renderTemplateAndDecode(p.OutputsTemplate, &data, &out.ResourceOutputs); err != nil {
		return nil, fmt.Errorf("outputs template failed: %w", err)
	}

	out.RelativeDirectories = make(map[string]bool)
	if err := renderTemplateAndDecode(p.RelativeDirectoriesTemplate, &data, &out.RelativeDirectories); err != nil {
		return nil, fmt.Errorf("directories template failed: %w", err)
	}

	out.RelativeFileContents = make(map[string]*string)
	if err := renderTemplateAndDecode(p.RelativeFilesTemplate, &data, &out.RelativeFileContents); err != nil {
		return nil, fmt.Errorf("files template failed: %w", err)
	}

	out.ComposeNetworks = make(map[string]compose.NetworkConfig)
	if err := renderTemplateAndDecode(p.ComposeNetworksTemplate, &data, &out.ComposeNetworks); err != nil {
		return nil, fmt.Errorf("networks template failed: %w", err)
	}

	out.ComposeServices = make(map[string]compose.ServiceConfig)
	if err := renderTemplateAndDecode(p.ComposeServicesTemplate, &data, &out.ComposeServices); err != nil {
		return nil, fmt.Errorf("networks template failed: %w", err)
	}

	out.ComposeVolumes = make(map[string]compose.VolumeConfig)
	if err := renderTemplateAndDecode(p.ComposeVolumesTemplate, &data, &out.ComposeVolumes); err != nil {
		return nil, fmt.Errorf("volumes template failed: %w", err)
	}

	return out, nil
}

var _ provisioners.Provisioner = (*Provisioner)(nil)
