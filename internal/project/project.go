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
)

// The StateDirectory holds the local state of the score-compose project, including any configuration, extensions,
// plugins, or resource provisioning state when possible.
type StateDirectory struct {
	// The path to the .score-compose directory
	Path string
	// The current State file
	State State
}

type Config struct {
	ComposeProjectName string `yaml:"composeProject"`
}

// Persist ensures that the directory is created and that the current config file has been written with the latest settings.
func (sd *StateDirectory) Persist() error {
	if sd.Path == "" {
		return fmt.Errorf("path not set")
	}
	if err := os.Mkdir(sd.Path, 0755); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("failed to create directory '%s': %w", sd.Path, err)
	}
	out := new(bytes.Buffer)
	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)
	if err := enc.Encode(sd.State); err != nil {
		return fmt.Errorf("failed to encode content: %w", err)
	}

	// important that we overwrite this file atomically via a inode move
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

	stateContent, err := os.ReadFile(filepath.Join(d, StateFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("config file couldn't be read: %w", err)
	}
	var state State
	dec := yaml.NewDecoder(bytes.NewReader(stateContent))
	dec.KnownFields(true)
	if err := dec.Decode(&state); err != nil {
		return nil, true, fmt.Errorf("config file couldn't be decoded: %w", err)
	}
	return &StateDirectory{Path: d, State: state}, true, nil
}
