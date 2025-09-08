// Copyright 2024 The Score Authors
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

package cmdprov

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/score-spec/score-go/framework"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/provisioners"
)

type Provisioner struct {
	ProvisionerUri string   `yaml:"uri"`
	ResType        string   `yaml:"type"`
	ResClass       *string  `yaml:"class,omitempty"`
	ResId          *string  `yaml:"id,omitempty"`
	Args           []string `yaml:"args"`
	ResourceParams []string `yaml:"params,omitempty"`
	ResOutputs     []string `yaml:"outputs,omitempty"`
	ResDescription string   `yaml:"description,omitempty"`

	// SupportedParams is a list of parameters that the provisioner expects to be passed in.
	SupportedParams []string `yaml:"supported_params,omitempty"`
	// ExpectedOutputs is a list of expected outputs that the provisioner should return.
	ExpectedOutputs []string `yaml:"expected_outputs,omitempty"`
}

func (p *Provisioner) Description() string {
	return p.ResDescription
}

func (p *Provisioner) Uri() string {
	return p.ProvisionerUri
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

func decodeBinary(uri string) (string, error) {
	parts, _ := url.Parse(uri)
	pathParts := strings.Split(parts.EscapedPath(), "/")
	switch parts.Hostname() {
	case "":
		return string(filepath.Separator) + filepath.Join(pathParts...), nil
	case "~":
		hd, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve user home directory: %w", err)
		}
		pathParts = slices.Insert(pathParts, 0, hd)
	case ".":
		pwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current working directory: %w", err)
		}
		pathParts = slices.Insert(pathParts, 0, pwd)
	case "..":
		pwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current working directory: %w", err)
		}
		pathParts = slices.Insert(pathParts, 0, filepath.Dir(pwd))
	default:
		if len(pathParts) > 1 {
			return "", fmt.Errorf("direct command reference cannot contain additional path parts")
		}
		b, err := exec.LookPath(parts.Hostname())
		if err != nil {
			return "", fmt.Errorf("failed to find '%s' on path: %w", parts.Hostname(), err)
		}
		pathParts = slices.Insert(pathParts, 0, b)
	}
	return filepath.Join(pathParts...), nil
}

func (p *Provisioner) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	bin, err := decodeBinary(p.Uri())
	if err != nil {
		return nil, err
	}

	rawInput, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to encode json input: %w", err)
	}
	outputBuffer := new(bytes.Buffer)

	cmd := exec.CommandContext(ctx, bin, p.Args...)
	slog.Debug(fmt.Sprintf("Executing '%s %v' for command provisioner", bin, p.Args))
	cmd.Stdin = bytes.NewReader(rawInput)
	cmd.Stdout = outputBuffer
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute cmd provisioner: %w", err)
	}

	var output provisioners.ProvisionOutput
	dec := json.NewDecoder(bytes.NewReader(outputBuffer.Bytes()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&output); err != nil {
		slog.Debug("Output from command provisioner:\n" + outputBuffer.String())
		return nil, fmt.Errorf("failed to decode output from cmd provisioner: %w", err)
	}

	return &output, nil
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

	parts, err := url.Parse(p.ProvisionerUri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	} else if parts.User != nil {
		return nil, fmt.Errorf("cmd provisioner uri cannot contain user info")
	} else if len(parts.Query()) != 0 {
		return nil, fmt.Errorf("cmd provisioner uri cannot contain query params")
	} else if parts.Port() != "" {
		return nil, fmt.Errorf("cmd provisioner uri cannot contain a port")
	}

	return p, nil
}
