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
	ConfigFileName                = "config.yaml"
)

// The StateDirectory holds the local state of the score-compose project, including any configuration, extensions,
// plugins, or resource provisioning state when possible.
type StateDirectory struct {
	// The path to the .score-compose directory
	Path string
	// The current config read from the config.yaml file
	Config Config
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
	if err := enc.Encode(sd.Config); err != nil {
		return fmt.Errorf("failed to encode content: %w", err)
	}

	// important that we overwrite this file atomically via a inode move
	if err := os.WriteFile(filepath.Join(sd.Path, ConfigFileName+".temp"), out.Bytes(), 0755); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	} else if err := os.Rename(filepath.Join(sd.Path, ConfigFileName+".temp"), filepath.Join(sd.Path, ConfigFileName)); err != nil {
		return fmt.Errorf("failed to complete writing config: %w", err)
	}
	return nil
}

// LoadStateDirectory loads the state directory for the given directory (usually PWD).
func LoadStateDirectory(directory string) (*StateDirectory, bool, error) {
	d := filepath.Join(directory, DefaultRelativeStateDirectory)
	if st, err := os.Stat(d); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to stat '%s': %w", d, err)
	} else if !st.IsDir() {
		return nil, false, fmt.Errorf("path '%s' is not a directory", d)
	}

	content, err := os.ReadFile(filepath.Join(d, ConfigFileName))
	if err != nil {
		return nil, true, fmt.Errorf("config file couldn't be read: %w", err)
	}

	var out Config
	dec := yaml.NewDecoder(bytes.NewReader(content))
	dec.KnownFields(true)
	if err := dec.Decode(&out); err != nil {
		return nil, true, fmt.Errorf("config file couldn't be decoded: %w", err)
	}
	return &StateDirectory{d, out}, true, nil
}
