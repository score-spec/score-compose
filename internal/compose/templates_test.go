/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package compose

import (
	"testing"

	score "github.com/score-spec/score-go/types"
	assert "github.com/stretchr/testify/assert"
)

func TestMapVar(t *testing.T) {
	var meta = score.WorkloadMetadata{
		"name":  "test-name",
		"other": map[string]interface{}{"key": "value"},
	}
	evt := new(EnvVarTracker)
	evt.lookup = func(key string) (string, bool) {
		if key == "DEBUG" {
			return "something", true
		}
		return "", false
	}
	ctx, err := buildContext(meta, map[string]ResourceWithOutputs{
		"env":    evt,
		"db":     evt.GenerateResource("db"),
		"static": resourceWithStaticOutputs{"x": "a"},
	})
	assert.NoError(t, err)

	for _, tc := range []struct {
		Input         string
		Expected      string
		ExpectedError string
	}{
		{Input: "missing", ExpectedError: "invalid ref 'missing': unknown reference root"},
		{Input: "metadata.name", Expected: "test-name"},
		{Input: "metadata", ExpectedError: "invalid ref 'metadata': requires at least a metadata key to lookup"},
		{Input: "metadata.other", Expected: "{\"key\":\"value\"}"},
		{Input: "metadata.other.key", Expected: "value"},
		{Input: "metadata.missing", ExpectedError: "invalid ref 'metadata.missing': key 'missing' not found"},
		{Input: "metadata.name.foo", ExpectedError: "invalid ref 'metadata.name.foo': cannot lookup key 'foo', context is not a map"},
		{Input: "resources.env", Expected: "env"},
		{Input: "resources.env.DEBUG", Expected: "${DEBUG}"},
		{Input: "resources.missing", ExpectedError: "invalid ref 'resources.missing': no known resource 'missing'"},
		{Input: "resources.db", Expected: "db"},
		{Input: "resources.db.host", Expected: "${DB_HOST?required}"},
		{Input: "resources.db.port", Expected: "${DB_PORT?required}"},
		{Input: "resources.db.name", Expected: "${DB_NAME?required}"},
		{Input: "resources.db.name.user", Expected: "${DB_NAME_USER?required}"},
		{Input: "resources.static", Expected: "static"},
		{Input: "resources.static.x", Expected: "a"},
		{Input: "resources.static.y", ExpectedError: "invalid ref 'resources.static.y': key 'y' not found"},
	} {
		t.Run(tc.Input, func(t *testing.T) {
			res, err := ctx.mapVar(tc.Input)
			if tc.ExpectedError != "" {
				assert.EqualError(t, err, tc.ExpectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, res)
			}
		})
	}

	assert.Equal(t, map[string]string{
		"DEBUG":        "something",
		"DB_HOST":      "",
		"DB_NAME":      "",
		"DB_NAME_USER": "",
		"DB_PORT":      "",
	}, evt.Accessed())
}

func TestSubstitute(t *testing.T) {
	var meta = score.WorkloadMetadata{
		"name": "test-name",
	}
	evt := new(EnvVarTracker)
	evt.lookup = func(key string) (string, bool) {
		if key == "DEBUG" {
			return "something", true
		}
		return "", false
	}
	ctx, err := buildContext(meta, map[string]ResourceWithOutputs{
		"env": evt,
		"db":  evt.GenerateResource("db"),
	})
	assert.NoError(t, err)

	for _, tc := range []struct {
		Input         string
		Expected      string
		ExpectedError string
	}{
		{Input: "", Expected: ""},
		{Input: "abc", Expected: "abc"},
		{Input: "$abc", Expected: "$abc"},
		{Input: "abc $$ abc", Expected: "abc $ abc"},
		{Input: "$${abc}", Expected: "${abc}"},
		{Input: "my name is ${metadata.name}", Expected: "my name is test-name"},
		{Input: "my name is ${metadata.thing\\.two}", ExpectedError: "invalid ref 'metadata.thing\\.two': key 'thing.two' not found"},
		{
			Input:    "postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}",
			Expected: "postgresql://${DB_USER?required}:${DB_PASSWORD?required}@${DB_HOST?required}:${DB_PORT?required}/${DB_NAME?required}",
		},
	} {
		t.Run(tc.Input, func(t *testing.T) {
			res, err := ctx.Substitute(tc.Input)
			if tc.ExpectedError != "" {
				if !assert.EqualError(t, err, tc.ExpectedError) {
					assert.Equal(t, "", res)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.Expected, res)
			}
		})
	}

	assert.Equal(t, map[string]string{
		"DB_HOST":     "",
		"DB_NAME":     "",
		"DB_PASSWORD": "",
		"DB_PORT":     "",
		"DB_USER":     "",
	}, evt.Accessed())
}
