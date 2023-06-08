/*
Apache Score
Copyright 2022 The Apache Software Foundation

This product includes software developed at
The Apache Software Foundation (http://www.apache.org/).
*/
package utils

import (
	"strconv"
	"strings"
)

// TryParseJsonValue attempts to convert an input string into a simple JSON value (string, number, boolean, or null).
//
// Complex values (arrays and objects) are not supported and treated as strings.
// Quoted values are always treated as strings.
//
// Conversion rules:
//
//	null    -> nil
//	123     -> float64
//	"123"   -> string
//	false   -> boolean
//	"false" -> string
//	abc     -> string
//	"abc"   -> string
func TryParseJsonValue(str string) interface{} {
	if str == "null" {
		return nil
	} else if strings.HasPrefix(str, "\"") {
		return strings.Trim(str, "\"")
	}

	if val, err := strconv.ParseFloat(str, 64); err == nil {
		return val
	} else if val, err := strconv.ParseBool(str); err == nil {
		return val
	}

	return str
}
