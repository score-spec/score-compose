package staticprovider

import (
	"context"

	compose "github.com/compose-spec/compose-go/types"
	score "github.com/score-spec/score-go/types"

	"github.com/score-spec/score-compose/internal/project"
)

type Provider struct {
	Type  string
	Class string
	Id    string

	Outputs map[string]interface{}
}

func (p *Provider) ProviderUri() string {
	return "builtin://static"
}

func (p *Provider) Provision(ctx context.Context, uid string, resource score.Resource, sharedState map[string]interface{}, state *project.ScoreResourceState, project *compose.Project) error {
	// Just set the outputs directly to what the provider returns
	state.Outputs = p.Outputs
	return nil
}

func (p *Provider) Match(resType, resClass, resId string) bool {
	return resType == p.Type && (p.Class == "" || (p.Class == resClass)) && (p.Id == "" || (p.Id == resId))
}
