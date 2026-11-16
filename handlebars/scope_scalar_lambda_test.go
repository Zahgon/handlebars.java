// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestScopeResolutionAndHelperPrecedence(t *testing.T) {
	context := object("name", "an attribute", "inner", object("name", "an inner attribute"))
	helper := map[string]handlebars.Helper{"name": func(any, *handlebars.Options) (any, error) { return "helper response", nil }}
	runRenderCases(t, []renderCase{
		{"simple", "{{name}}", context, "an attribute", nil, nil, nil},
		{"qualified", "{{this.name}}", context, "an attribute", nil, nil, nil},
		{"inner", "{{#inner}}{{name}}{{/inner}}", context, "an inner attribute", nil, nil, nil},
		{"qualified inner", "{{#inner}}{{this.name}}{{/inner}}", context, "an inner attribute", nil, nil, nil},
		{"path inner", "{{inner.name}}", context, "an inner attribute", nil, nil, nil},
		{"helper precedence", "{{name}}", context, "helper response", helper, nil, nil},
		{"qualified bypasses helper", "{{this.name}}", context, "an attribute", helper, nil, nil},
		{"nested helper precedence", "{{#inner}}{{name}}{{/inner}}", context, "helper response", helper, nil, nil},
		{"nested qualified bypasses helper", "{{#inner}}{{this.name}}{{/inner}}", context, "an inner attribute", helper, nil, nil},
		{"current scope only", "{{#child}}{{this.value}}{{/child}}", object("value", "parent", "child", object()), "", nil, nil, nil},
		{"implicit parent expansion", "{{#child}}{{value}}{{/child}}", object("value", "parent", "child", object()), "parent", nil, nil, nil},
	})
}

func TestScalarContexts(t *testing.T) {
	for _, selector := range []string{".", "this"} {
		selector := selector
		t.Run(selector, func(t *testing.T) {
			runRenderCases(t, []renderCase{
				{"integer", "var i = {{" + selector + "}};", int32(10), "var i = 10;", nil, nil, nil},
				{"string", "var s = '{{" + selector + "}}';", "Hello", "var s = 'Hello';", nil, nil, nil},
				{"array", "{{#" + selector + "}}{{" + selector + "}} {{/" + selector + "}}", []any{int32(1), int32(2), int32(3)}, "1 2 3 ", nil, nil, nil},
				{"unescaped", "var s = '{{{" + selector + "}}}';", "<div>", "var s = '<div>';", nil, nil, nil},
				{"safe string", "var s = '{{" + selector + "}}';", handlebars.NewSafeString("<div>"), "var s = '<div>';", nil, nil, nil},
				{"character", "var s = '{{" + selector + "}}';", jvm.Char('c'), "var s = 'c';", nil, nil, nil},
				{"html character", "var s = '{{" + selector + "}}';", jvm.Char('<'), "var s = '&lt;';", nil, nil, nil},
				{"html string", "var s = '{{" + selector + "}}';", "<div>", "var s = '&lt;div&gt;';", nil, nil, nil},
				{"boolean", "if ({{" + selector + "}})", true, "if (true)", nil, nil, nil},
				{"decimal", "var d = {{" + selector + "}};", float64(1.34), "var d = 1.34;", nil, nil, nil},
			})
		})
	}
}

func TestFalsyContexts(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"null", "Hello {{world}}!", nil, "Hello !", nil, nil, nil},
		{"empty map", "Hello {{world}}!", object(), "Hello !", nil, nil, nil},
		{"empty list", "Hello {{world}}!", []any{}, "Hello !", nil, nil, nil},
		{"boolean", "Hello {{world}}!", true, "Hello !", nil, nil, nil},
		{"decimal", "Hello {{world}}!", float64(13.4), "Hello !", nil, nil, nil},
	})
}

func TestLambdas(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"non-string result", "{{lambda}}", object("lambda", handlebars.Lambda(func(any, handlebars.Template) (any, error) { return float64(3.5), nil })), "3.5", nil, nil, nil},
		{"string expansion", "{{lambda}}", object("lambda", handlebars.Lambda(func(any, handlebars.Template) (any, error) { return "{{name}}", nil }), "name", "lambda!"), "lambda!", nil, nil, nil},
		{"section raw body", "{{#lambda}}{{name}}{{/lambda}}", object("name", "world", "lambda", handlebars.Lambda(func(_ any, template handlebars.Template) (any, error) {
			return fmt.Sprintf("[%s]", template.Text()), nil
		})), "[world]", nil, nil, nil},
	})
}
