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

package loader

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/score-spec/score-compose/internal/provisioners"
	"github.com/score-spec/score-compose/internal/provisioners/cmdprov"
	"github.com/score-spec/score-compose/internal/provisioners/templateprov"
)

const DefaultSuffix = ".provisioners.yaml"

// LoadProvisioners loads a list of provisioners from the raw contents from a yaml file.
func LoadProvisioners(raw []byte) ([]provisioners.Provisioner, error) {
	var intermediate []map[string]interface{}
	if err := yaml.NewDecoder(bytes.NewReader(raw)).Decode(&intermediate); err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}
	out := make([]provisioners.Provisioner, 0, len(intermediate))
	for i, m := range intermediate {
		uri, _ := m["uri"].(string)
		u, err := url.Parse(uri)
		if err != nil {
			return nil, fmt.Errorf("%d: invalid uri '%s'", i, u)
		} else if u.Scheme == "" {
			return nil, fmt.Errorf("%d: missing uri schema '%s'", i, u)
		}
		switch u.Scheme {
		case "template":
			if p, err := templateprov.Parse(m); err != nil {
				return nil, fmt.Errorf("%d: %s: failed to parse: %w", i, uri, err)
			} else {
				slog.Debug(fmt.Sprintf("Loaded provisioner %s", p.Uri()))
				out = append(out, p)
			}
		case "cmd":
			if p, err := cmdprov.Parse(m); err != nil {
				return nil, fmt.Errorf("%d: %s: failed to parse: %w", i, uri, err)
			} else {
				slog.Debug(fmt.Sprintf("Loaded provisioner %s", p.Uri()))
				out = append(out, p)
			}
		default:
			return nil, fmt.Errorf("%d: unsupported provisioner type '%s'", i, u.Scheme)
		}
	}
	return out, nil
}

// LoadProvisionersFromDirectory loads all providers we can find in files that end in the common suffix.
func LoadProvisionersFromDirectory(path string, suffix string) ([]provisioners.Provisioner, error) {
	slog.Debug(fmt.Sprintf("Loading providers with suffix %s in directory '%s'", suffix, path))
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]provisioners.Provisioner, 0)
	for _, item := range items {
		if !item.IsDir() && strings.HasSuffix(item.Name(), suffix) {
			raw, err := os.ReadFile(filepath.Join(path, item.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read '%s': %w", item.Name(), err)
			}
			p, err := LoadProvisioners(raw)
			if err != nil {
				return nil, fmt.Errorf("failed to load '%s': %w", item.Name(), err)
			}
			out = append(out, p...)
		}
	}
	return out, nil
}
