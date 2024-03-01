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
		"env": evt,
		"db":  evt.GenerateResource("db"),
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
		{Input: "metadata.missing", ExpectedError: "invalid ref 'metadata.missing': unknown metadata key 'missing'"},
		{Input: "metadata.name.foo", ExpectedError: "invalid ref 'metadata.name.foo': cannot lookup a key in string"},
		{Input: "resources.env", ExpectedError: "invalid ref 'resources.env': an output key is required"},
		{Input: "resources.env.DEBUG", Expected: "${DEBUG}"},
		{Input: "resources.missing", ExpectedError: "invalid ref 'resources.missing': no known resource 'missing'"},
		{Input: "resources.db", ExpectedError: "invalid ref 'resources.db': an output key is required"},
		{Input: "resources.db.host", Expected: "${DB_HOST}"},
		{Input: "resources.db.port", Expected: "${DB_PORT}"},
		{Input: "resources.db.name", Expected: "${DB_NAME}"},
		{Input: "resources.db.name.user", Expected: "${DB_NAME_USER}"},
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
		{Input: "my name is ${metadata.thing\\.two}", ExpectedError: "invalid ref 'metadata.thing\\.two': unknown metadata key 'thing.two'"},
		{
			Input:    "postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}",
			Expected: "postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}",
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
