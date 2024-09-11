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

package loader

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// SaveProvisionerToDirectory saves the provisioner content (data) from the provisionerUrl to a new provisioners file
// in the path directory.
func SaveProvisionerToDirectory(path string, provisionerUrl string, data []byte) error {
	// First validate whether this file contains valid provisioner data.
	if _, err := LoadProvisioners(data); err != nil {
		return fmt.Errorf("invalid provisioners file: %w", err)
	}
	// Append a heading indicating the source and time
	data = append([]byte(fmt.Sprintf("# Downloaded from %s at %s\n", provisionerUrl, time.Now())), data...)
	hashValue := sha256.Sum256([]byte(provisionerUrl))
	hashName := base64.RawURLEncoding.EncodeToString(hashValue[:16]) + DefaultSuffix
	// We use a time prefix to always put the most recently downloaded files first lexicographically. So subtract
	// time from uint64 and convert it into a base64 two's complement binary representation.
	timePrefix := base64.RawURLEncoding.EncodeToString(binary.BigEndian.AppendUint64([]byte{}, uint64(math.MaxInt64-time.Now().UnixNano())))

	targetPath := filepath.Join(path, timePrefix+"."+hashName)
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	} else if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	slog.Info(fmt.Sprintf("Wrote provisioner from '%s' to %s", provisionerUrl, targetPath))

	// Remove any old files that have the same source.
	if items, err := os.ReadDir(path); err != nil {
		return err
	} else {
		for _, item := range items {
			if strings.HasSuffix(item.Name(), hashName) && !strings.HasPrefix(item.Name(), timePrefix) {
				if err := os.Remove(filepath.Join(path, item.Name())); err != nil {
					return fmt.Errorf("failed to remove old copy of provisioner loaded from '%s': %w", provisionerUrl, err)
				}
				slog.Debug(fmt.Sprintf("Removed old copy of provisioner loaded from '%s'", provisionerUrl))
			}
		}
	}

	return nil
}
