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
		"name": "test-name",
	}

	var resources = score.WorkloadResources{
		"env": score.Resource{
			Type: "environment",
		},
		"db": score.Resource{
			Type: "postgres",
		},
	}

	ctx, err := buildContext(meta, resources)
	assert.NoError(t, err)

	assert.Equal(t, "", ctx.mapVar(""))
	assert.Equal(t, "$", ctx.mapVar("$"))

	assert.Equal(t, "test-name", ctx.mapVar("metadata.name"))
	assert.Equal(t, "", ctx.mapVar("metadata.name.nil"))
	assert.Equal(t, "", ctx.mapVar("metadata.nil"))

	assert.Equal(t, "${DEBUG}", ctx.mapVar("resources.env.DEBUG"))

	assert.Equal(t, "db", ctx.mapVar("resources.db"))
	assert.Equal(t, "${DB_HOST}", ctx.mapVar("resources.db.host"))
	assert.Equal(t, "${DB_PORT}", ctx.mapVar("resources.db.port"))
	assert.Equal(t, "${DB_NAME}", ctx.mapVar("resources.db.name"))
	assert.Equal(t, "${DB_NAME_USER}", ctx.mapVar("resources.db.name.user"))

	assert.Equal(t, "", ctx.mapVar("resources.nil"))
	assert.Equal(t, "", ctx.mapVar("nil.db.name"))
}

func TestSubstitute(t *testing.T) {
	var meta = score.WorkloadMetadata{
		"name": "test-name",
	}

	var resources = score.WorkloadResources{
		"env": score.Resource{
			Type: "environment",
		},
		"db": score.Resource{
			Type: "postgres",
		},
	}

	ctx, err := buildContext(meta, resources)
	assert.NoError(t, err)

	assert.Empty(t, ctx.ListEnvVars())

	assert.Equal(t, "", ctx.Substitute(""))
	assert.Equal(t, "abc", ctx.Substitute("abc"))
	assert.Equal(t, "$abc", ctx.Substitute("$abc"))
	assert.Equal(t, "abc $ abc", ctx.Substitute("abc $$ abc"))
	assert.Equal(t, "${abc}", ctx.Substitute("$${abc}"))

	assert.Equal(t, "The name is 'test-name'", ctx.Substitute("The name is '${metadata.name}'"))
	assert.Equal(t, "The name is ''", ctx.Substitute("The name is '${metadata.nil}'"))

	assert.Equal(t, "resources.badref.DEBUG", ctx.Substitute("resources.badref.DEBUG"))

	assert.Equal(t, "db", ctx.Substitute("${resources.db}"))
	assert.Equal(t,
		"postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}",
		ctx.Substitute("postgresql://${resources.db.user}:${resources.db.password}@${resources.db.host}:${resources.db.port}/${resources.db.name}"))

	assert.Equal(t, map[string]interface{}{
		"DB_USER":     "",
		"DB_PASSWORD": "",
		"DB_HOST":     "",
		"DB_PORT":     "",
		"DB_NAME":     "",
	}, ctx.ListEnvVars())
}
