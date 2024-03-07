package legacyvarprovider

import (
	"context"
	"fmt"
	"maps"
	"strings"

	compose "github.com/compose-spec/compose-go/types"

	"github.com/score-spec/score-compose/internal/project"
)

type Provider struct {
	Type  string
	Class string
	Id    string

	// Prefix is an environment variable prefix to apply
	Prefix string

	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]bool
}

func (p *Provider) String() string {
	return "builtin://legacy_var"
}

func (p *Provider) Provision(ctx context.Context, uid string, sharedState map[string]interface{}, state *project.ScoreResourceState, project *compose.Project) error {
	state.OutputLookupFunc = func(keys ...string) (interface{}, error) {
		if len(keys) < 1 {
			return nil, fmt.Errorf("resource requires at least one lookup key")
		}
		envVarKey := strings.ToUpper(strings.Join(keys, "_"))
		envVarKey = strings.Map(func(r rune) rune {
			if r == '_' || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return '_'
		}, envVarKey)

		envVarKey = strings.ReplaceAll(envVarKey, "-", "_")
		envVarKey = strings.ReplaceAll(envVarKey, ".", "_")
		envVarKey = p.Prefix + envVarKey
		if p.accessed == nil {
			p.accessed = make(map[string]bool, 1)
		}
		p.accessed[envVarKey] = true
		return "${" + envVarKey + "?required}", nil
	}
	return nil
}

func (p *Provider) Match(resType, resClass, resId string) bool {
	return resType == p.Type && (p.Class == "" || (p.Class == resClass)) && (p.Id == "" || (p.Id == resId))
}

func (p *Provider) Accessed() map[string]bool {
	return maps.Clone(p.accessed)
}
