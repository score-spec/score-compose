package legacyvarprovider

import (
	"context"
	"testing"

	score "github.com/score-spec/score-go/types"
	"github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/project"
)

func TestMatch(t *testing.T) {
	prov := &Provider{Id: "foo"}
	assert.False(t, prov.Match("foo", "default", ""))
	assert.True(t, prov.Match("foo", "default", "foo"))
	assert.False(t, prov.Match("foo", "unknown", ""))
	assert.True(t, prov.Match("foo", "unknown", "foo"))
	assert.True(t, prov.Match("bar", "unknown", "foo"))
}

func TestProvisionAndLookup(t *testing.T) {
	prov := &Provider{}
	resState := &project.ScoreResourceState{Id: "foo"}
	assert.NoError(t, prov.Provision(context.Background(), "my-resource", score.Resource{}, map[string]interface{}{}, resState, nil))

	for _, tc := range []struct {
		Name          string
		Keys          []string
		Expected      string
		ExpectedError string
	}{
		{Name: "basic", Keys: []string{"hello"}, Expected: "${FOO_HELLO?required}"},
		{Name: "basic mixed case", Keys: []string{"HEllo"}, Expected: "${FOO_HELLO?required}"},
		{Name: "2 keys", Keys: []string{"foo", "bar-baz**"}, Expected: "${FOO_FOO_BAR_BAZ__?required}"},
		{Name: "no keys", Keys: []string{}, ExpectedError: "resource requires at least one lookup key"},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			res, err := resState.LookupOutput(tc.Keys...)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, res)
			}
		})
	}
	assert.Equal(t, map[string]bool{
		"FOO_HELLO":         true,
		"FOO_FOO_BAR_BAZ__": true,
	}, prov.Accessed())
}
