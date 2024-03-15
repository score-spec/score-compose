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

package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRelativeStateDirectory = ".score-compose"
	StateFileName                 = "state.yaml"
	MountsDirectoryName           = "mounts"
)

// The StateDirectory holds the local state of the score-compose project, including any configuration, extensions,
// plugins, or resource provisioning state when possible.
type StateDirectory struct {
	// The path to the .score-compose directory
	Path string
	// The current state file
	State State
}

// Persist ensures that the directory is created and that the current config file has been written with the latest settings.
func (sd *StateDirectory) Persist() error {
	if sd.Path == "" {
		return fmt.Errorf("path not set")
	}
	if err := os.Mkdir(sd.Path, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("failed to create directory '%s': %w", sd.Path, err)
	}
	if err := os.Mkdir(sd.State.MountsDirectory, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("failed to create mounts directory: %w", err)
	}
	out := new(bytes.Buffer)
	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	if err := enc.Encode(sd.State); err != nil {
		return fmt.Errorf("failed to encode content: %w", err)
	}

	// important that we overwrite this file atomically via an inode move
	if err := os.WriteFile(filepath.Join(sd.Path, StateFileName+".temp"), out.Bytes(), 0755); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	} else if err := os.Rename(filepath.Join(sd.Path, StateFileName+".temp"), filepath.Join(sd.Path, StateFileName)); err != nil {
		return fmt.Errorf("failed to complete writing state: %w", err)
	}
	return nil
}

// LoadStateDirectory loads the state directory for the given directory (usually PWD).
func LoadStateDirectory(directory string) (*StateDirectory, bool, error) {
	d := filepath.Join(directory, DefaultRelativeStateDirectory)
	content, err := os.ReadFile(filepath.Join(d, StateFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("state file couldn't be read: %w", err)
	}

	var out State
	dec := yaml.NewDecoder(bytes.NewReader(content))
	dec.KnownFields(true)
	if err := dec.Decode(&out); err != nil {
		return nil, true, fmt.Errorf("state file couldn't be decoded: %w", err)
	}
	return &StateDirectory{d, out}, true, nil
}
