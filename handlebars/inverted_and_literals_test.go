// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestInvertedSections(t *testing.T) {
	input := "{{#goodbyes}}{{this}}{{/goodbyes}}{{^goodbyes}}Right On!{{/goodbyes}}"
	runRenderCases(t, []renderCase{
		{"unset", input, object(), "Right On!", nil, nil, nil},
		{"false", input, object("goodbyes", false), "Right On!", nil, nil, nil},
		{"empty", input, object("goodbyes", []any{}), "Right On!", nil, nil, nil},
	})
}

func TestLiteralParameters(t *testing.T) {
	hello := func(context any, options *handlebars.Options) (any, error) {
		return "Hello " + stringify(context) + " " + stringify(options.Param(0)) + " times: " + stringify(options.Param(1)) + " " + stringify(options.Param(2)), nil
	}
	runRenderCases(t, []renderCase{
		{"simple literals", "Message: {{hello \"world\" 12 true false}}", object(), "Message: Hello world 12 times: true false", map[string]handlebars.Helper{"hello": hello}, nil, nil},
		{"escaped quote", "Message: {{{quote \"\\\"world\\\"\"}}}", object(), "Message: Hello \"world\"", map[string]handlebars.Helper{"quote": func(context any, _ *handlebars.Options) (any, error) { return "Hello " + stringify(context), nil }}, nil, nil},
		{"apostrophe", "Message: {{{quote \"Alan's world\"}}}", object(), "Message: Hello Alan's world", map[string]handlebars.Helper{"quote": func(context any, _ *handlebars.Options) (any, error) { return "Hello " + stringify(context), nil }}, nil, nil},
		{"double quoted newline", "Message: {{{words \"before\" \"multi\nline\" \"after\"}}}", object(), "Message: Hello before multi\nline after", map[string]handlebars.Helper{"words": wordsHelper}, nil, nil},
		{"single quoted newline", "Message: {{{words 'before' 'multi\nline' 'after'}}}", object(), "Message: Hello before multi\nline after", map[string]handlebars.Helper{"words": wordsHelper}, nil, nil},
	})
}

func wordsHelper(context any, options *handlebars.Options) (any, error) {
	return "Hello " + stringify(context) + " " + stringify(options.Param(0)) + " " + stringify(options.Param(1)), nil
}

func stringify(value any) string {
	return fmt.Sprint(value)
}
