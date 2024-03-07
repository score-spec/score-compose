package envvarprovider

import (
	"context"
	"fmt"
	"maps"

	compose "github.com/compose-spec/compose-go/types"

	"github.com/score-spec/score-compose/internal/project"
)

type Provider struct {
	// accessed is the map of accessed environment variables and the value they had at access time
	accessed map[string]bool
}

func (p *Provider) String() string {
	return "builtin://env_var"
}

func (p *Provider) Provision(ctx context.Context, uid string, sharedState map[string]interface{}, state *project.ScoreResourceState, project *compose.Project) error {
	state.OutputLookupFunc = func(keys ...string) (interface{}, error) {
		if len(keys) != 1 {
			return nil, fmt.Errorf("environment resource only supports a single lookup key")
		}
		envVarKey := keys[0]
		if p.accessed == nil {
			p.accessed = make(map[string]bool, 1)
		}
		p.accessed[envVarKey] = true
		return "${" + envVarKey + "}", nil
	}
	return nil
}

func (p *Provider) Match(resType, resClass, resId string) bool {
	return resType == "environment" && resClass == "default"
}

func (p *Provider) Accessed() map[string]bool {
	return maps.Clone(p.accessed)
}
