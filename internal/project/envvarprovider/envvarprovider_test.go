package envvarprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/project"
)

func TestMatch(t *testing.T) {
	prov := new(Provider)
	assert.True(t, prov.Match("environment", "default", ""))
	assert.True(t, prov.Match("environment", "default", "foo"))
	assert.False(t, prov.Match("environment", "unknown", ""))
	assert.False(t, prov.Match("environment", "unknown", "foo"))
	assert.False(t, prov.Match("other", "unknown", "foo"))
}

func TestProvisionAndLookup(t *testing.T) {
	prov := new(Provider)
	resState := new(project.ScoreResourceState)
	assert.NoError(t, prov.Provision(
		context.Background(),
		"my-resource",
		map[string]interface{}{},
		resState,
		nil,
	))

	for _, tc := range []struct {
		Name          string
		Keys          []string
		Expected      string
		ExpectedError string
	}{
		{Name: "basic", Keys: []string{"hello"}, Expected: "${hello}"},
		{Name: "basic mixed case", Keys: []string{"HEllo"}, Expected: "${HEllo}"},
		{Name: "2 keys", Keys: []string{"foo", "bar"}, ExpectedError: "environment resource only supports a single lookup key"},
		{Name: "no keys", Keys: []string{}, ExpectedError: "environment resource only supports a single lookup key"},
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
		"hello": true,
		"HEllo": true,
	}, prov.Accessed())
}
