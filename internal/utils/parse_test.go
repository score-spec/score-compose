/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package utils

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestTryParseJsonValue(t *testing.T) {
	assert.Equal(t, "", TryParseJsonValue(""))
	assert.Equal(t, "", TryParseJsonValue("\"\""))

	assert.Equal(t, nil, TryParseJsonValue("null"))
	assert.Equal(t, "null", TryParseJsonValue("\"null\""))

	assert.Equal(t, float64(123), TryParseJsonValue("123"))
	assert.Equal(t, float64(-123.23), TryParseJsonValue("-123.23"))
	assert.Equal(t, "123", TryParseJsonValue("\"123\""))
	assert.Equal(t, "123 123", TryParseJsonValue("123 123"))

	assert.Equal(t, true, TryParseJsonValue("true"))
	assert.Equal(t, false, TryParseJsonValue("false"))
	assert.Equal(t, "false", TryParseJsonValue("\"false\""))
	assert.Equal(t, "true 23", TryParseJsonValue("true 23"))

	assert.Equal(t, "abc", TryParseJsonValue("abc"))
	assert.Equal(t, "abc", TryParseJsonValue("\"abc\""))
	assert.Equal(t, "{\"key\": \"value\"}", TryParseJsonValue("{\"key\": \"value\"}"))
	assert.Equal(t, "[1, 2, 3]", TryParseJsonValue("[1, 2, 3]"))
}
