package patching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"
)

type PatchOperation struct {
	Op          string      `json:"op"`
	Path        string      `json:"path"`
	Value       interface{} `json:"value,omitempty"`
	Description string      `json:"description,omitempty"`
}

func yamlRoundTrip[k any, v any](input *k) (*v, error) {
	raw, err := yaml.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	var out *v
	if err := yaml.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return out, nil
}

type patchTemplateInput struct {
	Compose   map[string]interface{}
	Resources map[string]interface{}
}

func ValidatePatchTemplate(content string) error {
	if _, err := template.New("").Funcs(sprig.FuncMap()).Parse(content); err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	return nil
}

func PatchServices(composeProject *types.Project, rawTemplate string) (*types.Project, error) {
	tmpl, err := template.New("").Funcs(sprig.FuncMap()).Parse(rawTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	buff := &bytes.Buffer{}
	composeInputs, err := yamlRoundTrip[types.Project, map[string]interface{}](composeProject)
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(buff, patchTemplateInput{
		Compose: *composeInputs,
	}); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	var patches []PatchOperation
	templatedPatches := strings.TrimSpace(buff.String())
	if templatedPatches == "" {
		return composeProject, nil
	}

	yamlDecoder := yaml.NewDecoder(strings.NewReader(templatedPatches))
	yamlDecoder.KnownFields(true)
	if err := yamlDecoder.Decode(&patches); err != nil {
		slog.Debug("Raw patch output", slog.String("raw", templatedPatches))
		return nil, fmt.Errorf("failed to unmarshal patches from template execution output: %w", err)
	}
	jsonInput, _ := json.Marshal(composeInputs)
	for i, p := range patches {
		if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
			slog.Debug("Applying patch", slog.String("operation", p.Op), slog.String("path", p.Path), slog.Any("value", p.Value), slog.Any("description", p.Description))
		} else {
			desc := p.Description
			if desc != "" {
				desc = " (" + desc + ")"
			}
			slog.Info(fmt.Sprintf("Applying patch to %s%s", p.Path, desc))
		}
		switch p.Op {
		case "set":
			jsonInput, err = sjson.SetBytes(jsonInput, p.Path, p.Value)
		case "delete":
			jsonInput, err = sjson.DeleteBytes(jsonInput, p.Path)
		default:
			err = fmt.Errorf("unknown operation: %s", p.Op)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to perform patch operation %d: '%s '%s': %w", i+1, p.Op, p.Path, err)
		}
	}

	var output *types.Project
	decoder := json.NewDecoder(bytes.NewReader(jsonInput))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patched output: %w", err)
	}
	return output, nil
}
