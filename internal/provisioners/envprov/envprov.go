// Copyright 2024 Humanitec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envprov

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"os"
	"strings"

	"github.com/score-spec/score-go/framework"

	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/util"
)

// The Provisioner is an environment provision which returns a suitable expression for accessing an environment variable
// within the compose project at deploy time. This provisioner also tracks what env vars are accessed so that they can
// be added to the .env file later.
type Provisioner struct {
	// LookupFunc is an environment variable LookupFunc function, if nil this will be defaulted to os.LookupEnv
	LookupFunc func(key string) (string, bool)
	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]string
}

func (e *Provisioner) Uri() string {
	return "builtin://environment"
}

func (e *Provisioner) Match(resUid framework.ResourceUid) bool {
	return resUid.Type() == "environment" && resUid.Class() == "default" && strings.Contains(resUid.Id(), ".")
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
	return &envVarResourceTracker{
		uid:    resUid,
		inner:  e,
		prefix: strings.ToUpper(resName),
	}
}

// envVarResourceTracker is a child object of EnvVarTracker and is used as a fallback behavior for resource types
// that are not supported natively: we treat them like environment variables instead with a prefix of the resource name.
type envVarResourceTracker struct {
	uid    framework.ResourceUid
	prefix string
	inner  *Provisioner
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

var _ provisioners.Provisioner = (*Provisioner)(nil)
var _ provisioners.Provisioner = (*envVarResourceTracker)(nil)
