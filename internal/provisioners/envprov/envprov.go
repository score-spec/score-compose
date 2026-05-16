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

package envprov

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/score-spec/score-go/framework"
	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/util"
)

// The Provisioner is an environment provision which returns a suitable expression for accessing an environment variable
// within the compose project at deploy time. This provisioner also tracks what env vars are accessed so that they can
// be added to the .env file later.
//
// It can be used in two modes:
//   - Legacy mode: created via new(Provisioner), all fields are zero-valued, methods use hardcoded defaults.
//   - YAML-loaded mode: created via Parse(), fields are populated from the provisioners YAML file.
type Provisioner struct {
	ProvisionerUri  string   `yaml:"uri"`
	ResType         string   `yaml:"type"`
	ResClass        *string  `yaml:"class,omitempty"`
	ResDescription  string   `yaml:"description,omitempty"`
	SupportedParams []string `yaml:"supported_params,omitempty"`
	ExpectedOutputs []string `yaml:"expected_outputs,omitempty"`
	// LookupFunc is an environment variable LookupFunc function, if nil this will be defaulted to os.LookupEnv
	LookupFunc func(key string) (string, bool) `yaml:"-"`
	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]string
}

// Parse loads a Provisioner from raw YAML map data, following the same pattern as cmdprov.Parse and templateprov.Parse.
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
	}
	if p.ResType == "" {
		p.ResType = "environment"
	}
	return p, nil
}

func (e *Provisioner) Uri() string {
	if e.ProvisionerUri != "" {
		return e.ProvisionerUri
	}
	return "builtin://environment"
}

func (e *Provisioner) Match(resUid framework.ResourceUid) bool {
	if e.ProvisionerUri == "" {
		// Legacy mode: preserve original matching behavior
		return resUid.Type() == "environment" && resUid.Class() == "default" && strings.Contains(resUid.Id(), ".")
	}
	// YAML-loaded mode: standard type/class matching (same as cmdprov/templateprov)
	if resUid.Type() != e.ResType {
		return false
	}
	if e.ResClass != nil && resUid.Class() != *e.ResClass {
		return false
	}
	return true
}

func (e *Provisioner) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	if len(input.ResourceParams) > 0 {
		return nil, fmt.Errorf("no params expected")
	}
	return &provisioners.ProvisionOutput{OutputLookupFunc: e.LookupOutput}, nil
}

func (e *Provisioner) Accessed() map[string]string {
	return maps.Clone(e.accessed)
}

func (e *Provisioner) lookupOutput(required bool, envVarKey string) (interface{}, error) {
	if e.LookupFunc == nil {
		e.LookupFunc = os.LookupEnv
	}
	if e.accessed == nil {
		e.accessed = make(map[string]string, 1)
	}

	if v, ok := e.LookupFunc(envVarKey); ok {
		e.accessed[envVarKey] = v
	} else {
		e.accessed[envVarKey] = ""
	}
	return nil, &util.DeferredEnvironmentVariable{Variable: envVarKey, Required: required}
}

func (e *Provisioner) LookupOutput(keys ...string) (interface{}, error) {
	if len(keys) != 1 {
		return nil, fmt.Errorf("environment resource only supports a single lookup key")
	}
	return e.lookupOutput(false, keys[0])
}

func (e *Provisioner) GenerateSubProvisioner(resName string, resUid framework.ResourceUid) provisioners.Provisioner {
	class := "default"
	if strings.Contains(string(resUid), ".") {
		class = resUid.Class()
	}

	rType := ""
	if strings.Contains(string(resUid), "#") {
		rType = resUid.Type()
	}

	return &envVarResourceTracker{
		uid:      resUid,
		inner:    e,
		prefix:   strings.ToUpper(resName),
		resClass: class,
		resType:  rType,
	}
}

// envVarResourceTracker is a child object of EnvVarTracker and is used as a fallback behavior for resource types
// that are not supported natively: we treat them like environment variables instead with a prefix of the resource name.
type envVarResourceTracker struct {
	uid            framework.ResourceUid
	prefix         string
	inner          *Provisioner
	resClass       string
	resType        string
	resDescription string
}

func (e *envVarResourceTracker) Description() string {
	return e.resDescription
}

func (e *envVarResourceTracker) Uri() string {
	return "builtin://environment/" + url.PathEscape(string(e.uid))
}

func (e *envVarResourceTracker) Match(resUid framework.ResourceUid) bool {
	return e.uid == resUid
}

func (e *envVarResourceTracker) Provision(ctx context.Context, input *provisioners.Input) (*provisioners.ProvisionOutput, error) {
	return &provisioners.ProvisionOutput{
		OutputLookupFunc: e.LookupOutput,
	}, nil
}

func (e *envVarResourceTracker) LookupOutput(keys ...string) (interface{}, error) {
	if len(keys) < 1 {
		return nil, fmt.Errorf("at least one output lookup key is required")
	}
	sb := new(strings.Builder)
	_, _ = sb.WriteString(e.prefix)
	for _, k := range keys {
		_, _ = sb.WriteString("_")
		_, _ = sb.WriteString(k)
	}
	k := strings.ReplaceAll(sb.String(), "-", "_")
	k = strings.ReplaceAll(k, ".", "_")
	k = strings.ToUpper(k)
	return e.inner.lookupOutput(true, k)
}

func (e *envVarResourceTracker) Class() string {
	return e.resClass
}

func (e *envVarResourceTracker) Type() string {
	return e.resType
}

func (p *Provisioner) Class() string {
	if p.ProvisionerUri != "" && p.ResClass == nil {
		return "(any)"
	}
	if p.ResClass != nil {
		return *p.ResClass
	}
	return "default"
}

func (p *Provisioner) Type() string {
	if p.ResType != "" {
		return p.ResType
	}
	return "environment"
}

func (p *Provisioner) Outputs() []string {
	if p.ExpectedOutputs == nil {
		return nil
	}
	outputs := make([]string, len(p.ExpectedOutputs))
	copy(outputs, p.ExpectedOutputs)
	slices.Sort(outputs)
	return outputs
}

func (p *Provisioner) Params() []string {
	if p.SupportedParams == nil {
		return nil
	}
	params := make([]string, len(p.SupportedParams))
	copy(params, p.SupportedParams)
	slices.Sort(params)
	return params
}

func (e *envVarResourceTracker) Outputs() []string {
	return nil
}

func (e *envVarResourceTracker) Params() []string {
	return nil
}

func (p *Provisioner) Description() string {
	return p.ResDescription
}

var _ provisioners.Provisioner = (*Provisioner)(nil)
var _ provisioners.Provisioner = (*envVarResourceTracker)(nil)
