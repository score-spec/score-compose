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

func TestBuildContext(t *testing.T) {
	var meta = score.WorkloadMeta{
		Name: "test-name",
	}

	var resources = score.ResourcesSpecs{
		"env": score.ResourceSpec{
			Type: "environment",
			Properties: map[string]score.ResourcePropertySpec{
				"DEBUG": {Required: false, Default: true},
			},
		},
		"db": score.ResourceSpec{
			Type: "postgres",
			Properties: map[string]score.ResourcePropertySpec{
				"host": {Required: true, Default: "."},
				"port": {Required: true, Default: "5342"},
				"name": {Required: true},
			},
		},
	}

	context, err := buildContext(meta, resources)
	assert.NoError(t, err)

	assert.Equal(t, templatesContext{
		"metadata": map[string]interface{}{
			"name": "test-name",
		},
		"resources": map[string]namedObjectMap{
			"env": {
				".name": "env",
				"DEBUG": "${DEBUG-true}",
			},
			"db": {
				".name": "db",
				"host":  "${DB_HOST-.}",
				"port":  "${DB_PORT-5342}",
				"name":  "${DB_NAME?err}",
			},
		},
	}, context)
}

func TestMapVar(t *testing.T) {
	var context = templatesContext{
		"metadata": map[string]interface{}{
			"name": "test-name",
		},
		"resources": map[string]namedObjectMap{
			"env": {
				".name": "env",
				"DEBUG": "${DEBUG-true}",
			},
			"db": {
				".name": "db",
				"host":  "${DB_HOST-.}",
				"port":  "${DB_PORT-5342}",
				"name":  "${DB_NAME?err}",
			},
		},
	}

	assert.Equal(t, "", context.mapVar(""))
	assert.Equal(t, "$", context.mapVar("$"))

	assert.Equal(t, "test-name", context.mapVar("metadata.name"))
	assert.Equal(t, "", context.mapVar("metadata.name.nil"))
	assert.Equal(t, "", context.mapVar("metadata.nil"))

	assert.Equal(t, "${DEBUG-true}", context.mapVar("resources.env.DEBUG"))

	assert.Equal(t, "db", context.mapVar("resources.db"))
	assert.Equal(t, "${DB_HOST-.}", context.mapVar("resources.db.host"))
	assert.Equal(t, "${DB_PORT-5342}", context.mapVar("resources.db.port"))
	assert.Equal(t, "${DB_NAME?err}", context.mapVar("resources.db.name"))
	assert.Equal(t, "", context.mapVar("resources.db.name.nil"))
	assert.Equal(t, "", context.mapVar("resources.db.nil"))
	assert.Equal(t, "", context.mapVar("resources.nil"))
	assert.Equal(t, "", context.mapVar("nil.db.name"))
}

func TestSubstitute(t *testing.T) {
	var context = templatesContext{
		"metadata": map[string]interface{}{
			"name": "test-name",
		},
		"resources": map[string]namedObjectMap{
			"env": {
				".name": "env",
				"DEBUG": "${DEBUG-true}",
			},
			"db": {
				".name": "db",
				"host":  "${DB_HOST-.}",
				"port":  "${DB_PORT-5342}",
				"name":  "${DB_NAME?err}",
			},
		},
	}

	assert.Equal(t, "", context.Substitute(""))
	assert.Equal(t, "abc", context.Substitute("abc"))
	assert.Equal(t, "abc $ abc", context.Substitute("abc $$ abc"))
	assert.Equal(t, "${abc}", context.Substitute("$${abc}"))

	assert.Equal(t, "The name is 'test-name'", context.Substitute("The name is '${metadata.name}'"))
	assert.Equal(t, "The name is ''", context.Substitute("The name is '${metadata.nil}'"))

	assert.Equal(t, "resources.env.DEBUG", context.Substitute("resources.env.DEBUG"))

	assert.Equal(t, "db", context.Substitute("${resources.db}"))
	assert.Equal(t,
		"postgresql://:@${DB_HOST-.}:${DB_PORT-5342}/${DB_NAME?err}",
		context.Substitute("postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}"))
}

func TestEnvVarPattern(t *testing.T) {
	assert.Equal(t, []string(nil), envVarPattern.FindStringSubmatch(""))
	assert.Equal(t, []string(nil), envVarPattern.FindStringSubmatch("ENV_VAR"))
	assert.Equal(t, []string(nil), envVarPattern.FindStringSubmatch("${}"))

	assert.Equal(t, []string{"${ENV_VAR}", "ENV_VAR", ""}, envVarPattern.FindStringSubmatch("${ENV_VAR}"))
	assert.Equal(t, []string{"${ENV_VAR?err}", "ENV_VAR", ""}, envVarPattern.FindStringSubmatch("${ENV_VAR?err}"))
	assert.Equal(t, []string{"${ENV_VAR-default}", "ENV_VAR", "default"}, envVarPattern.FindStringSubmatch("${ENV_VAR-default}"))
}

func TestListEnvVars(t *testing.T) {
	var context = templatesContext{
		"metadata": map[string]interface{}{
			"name": "test-name",
		},
		"resources": map[string]namedObjectMap{
			"env": {
				".name": "env",
				"DEBUG": "${DEBUG-true}",
			},
			"db": {
				".name": "db",
				"host":  "${DB_HOST-.}",
				"port":  "${DB_PORT-5342}",
				"name":  "${DB_NAME?err}",
			},
		},
	}

	assert.Equal(t, map[string]interface{}{
		"DEBUG":   "true",
		"DB_HOST": ".",
		"DB_PORT": "5342",
		"DB_NAME": "",
	}, context.ListEnvVars())
}
