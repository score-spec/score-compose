package legacyvarprovider

import (
	"context"
	"fmt"
	"maps"
	"strings"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/project"
)

type Provider struct {
	Id string

	// Prefix is an environment variable prefix to apply
	Prefix string

	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]bool
}

func (p *Provider) ProviderUri() string {
	return "builtin://legacy_var"
}

func (p *Provider) Provision(ctx context.Context, uid string, resource score.Resource, sharedState map[string]interface{}, state *project.ScoreResourceState, project *compose.Project) error {
	state.OutputLookupFunc = func(keys ...string) (interface{}, error) {
		if len(keys) < 1 {
			return nil, fmt.Errorf("resource requires at least one lookup key")
		}
		envVarKey := strings.Join(keys, "_")
		if p.Prefix == "" {
			envVarKey = state.Id + "_" + envVarKey
		} else {
			envVarKey = p.Prefix + envVarKey
		}
		envVarKey = strings.ToUpper(envVarKey)
		envVarKey = strings.Map(func(r rune) rune {
			if r == '_' || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return '_'
		}, envVarKey)

		envVarKey = strings.ReplaceAll(envVarKey, "-", "_")
		envVarKey = strings.ReplaceAll(envVarKey, ".", "_")
		if p.accessed == nil {
			p.accessed = make(map[string]bool, 1)
		}
		p.accessed[envVarKey] = true
		return "${" + envVarKey + "?required}", nil
	}
	return nil
}

func (p *Provider) Match(resType, resClass, resId string) bool {
	return p.Id == resId
}

func (p *Provider) Accessed() map[string]bool {
	return maps.Clone(p.accessed)
}
