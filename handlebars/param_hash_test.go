// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestParameterAndHashLiterals(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"var":  func(_ any, options *handlebars.Options) (any, error) { return javaLiteral(options.HashValue("h")), nil },
		"varp": func(_ any, options *handlebars.Options) (any, error) { return javaLiteral(options.Param(0)), nil },
	}
	runRenderCases(t, []renderCase{
		{"true parameter", "{{varp . true}}", object(), "Boolean:true", helpers, nil, nil},
		{"false parameter", "{{varp . false}}", object(), "Boolean:false", helpers, nil, nil},
		{"true hash", "{{var h=true}}", object(), "Boolean:true", helpers, nil, nil},
		{"false hash", "{{var h=false}}", object(), "Boolean:false", helpers, nil, nil},
		{"integer parameter", "{{varp . 9}}", object(), "Integer:9", helpers, nil, nil},
		{"integer hash", "{{var h=9}}", object(), "Integer:9", helpers, nil, nil},
		{"string parameter", "{{varp . \"Hey!\"}}", object(), "String:Hey!", helpers, nil, nil},
		{"string hash", "{{var h=\"Hey!\"}}", object(), "String:Hey!", helpers, nil, nil},
		{"single quoted hash", "{{var h='Hey!'}}", object(), "String:Hey!", helpers, nil, nil},
		{"reference parameter", "{{varp . ref}}", object("ref", "."), "String:.", helpers, nil, nil},
		{"reference hash", "{{var h=ref}}", object("ref", "."), "String:.", helpers, nil, nil},
	})
}

func javaLiteral(value any) string {
	name := fmt.Sprintf("%T", value)
	switch value.(type) {
	case bool:
		name = "Boolean"
	case int32:
		name = "Integer"
	case string:
		name = "String"
	}
	return name + ":" + fmt.Sprint(value)
}
