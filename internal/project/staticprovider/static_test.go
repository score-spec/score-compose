package staticprovider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/score-spec/score-compose/internal/project"
)

func TestMatch(t *testing.T) {
	prov := &Provider{Type: "foo"}
	assert.True(t, prov.Match("foo", "default", ""))
	assert.True(t, prov.Match("foo", "default", "foo"))
	assert.True(t, prov.Match("foo", "unknown", ""))
	assert.True(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))

	prov = &Provider{Type: "foo", Class: "bar"}
	assert.True(t, prov.Match("foo", "bar", ""))
	assert.True(t, prov.Match("foo", "bar", "foo"))
	assert.False(t, prov.Match("foo", "unknown", ""))
	assert.False(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))

	prov = &Provider{Type: "foo", Class: "bar", Id: "foo"}
	assert.False(t, prov.Match("foo", "bar", ""))
	assert.True(t, prov.Match("foo", "bar", "foo"))
	assert.False(t, prov.Match("foo", "unknown", ""))
	assert.False(t, prov.Match("foo", "unknown", "foo"))
	assert.False(t, prov.Match("bar", "unknown", "foo"))
}

func TestProvisionAndLookup(t *testing.T) {
	prov := &Provider{Outputs: map[string]interface{}{"a": map[string]interface{}{"b": "hello"}}}
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
		{Name: "basic", Keys: []string{"a", "b"}, Expected: "hello"},
		{Name: "not found", Keys: []string{"g"}, ExpectedError: "key 'g' not found"},
		{Name: "bad type", Keys: []string{"a", "b", "c"}, ExpectedError: "cannot lookup key 'c', context is not a map"},
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
}
