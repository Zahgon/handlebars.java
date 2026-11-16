// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestBlocks(t *testing.T) {
	people := []any{object("text", "goodbye"), object("text", "Goodbye"), object("text", "GOODBYE")}
	runRenderCases(t, []renderCase{
		{"array", "{{#goodbyes}}{{text}}! {{/goodbyes}}cruel {{world}}!", object("goodbyes", people, "world", "world"), "goodbye! Goodbye! GOODBYE! cruel world!", nil, nil, nil},
		{"empty array", "{{#goodbyes}}{{text}}! {{/goodbyes}}cruel {{world}}!", object("goodbyes", []any{}, "world", "world"), "cruel world!", nil, nil, nil},
		{"index", "{{#goodbyes}}{{@index}}. {{text}}! {{/goodbyes}}cruel {{world}}!", object("goodbyes", people, "world", "world"), "0. goodbye! 1. Goodbye! 2. GOODBYE! cruel world!", nil, nil, nil},
		{"parent lookup", "{{#goodbyes}}{{text}} cruel {{../name}}! {{/goodbyes}}", object("goodbyes", people, "name", "Alan"), "goodbye cruel Alan! Goodbye cruel Alan! GOODBYE cruel Alan! ", nil, nil, nil},
		{"deep parent lookup", "{{#outer}}Goodbye {{#inner}}cruel {{../../omg}}{{/inner}}{{/outer}}", object("omg", "OMG!", "outer", []any{object("inner", []any{object("text", "goodbye")})}), "Goodbye cruel OMG!", nil, nil, nil},
		{"undefined block", "{{#_empty}}shouldn't render{{/_empty}}", object(), "", nil, nil, nil},
		{"helper block", "{{#goodbyes}}{{text}}! {{/goodbyes}}cruel {{world}}!", object("world", "world"), "GOODBYE! cruel world!", map[string]handlebars.Helper{
			"goodbyes": func(_ any, options *handlebars.Options) (any, error) {
				return options.ApplyFn(object("text", "GOODBYE"))
			},
		}, nil, nil},
		{"helper current context", "{{#form}}<p>{{name}}</p>{{/form}}", object("name", "Yehuda"), "<form><p>Yehuda</p></form>", map[string]handlebars.Helper{
			"form": func(context any, options *handlebars.Options) (any, error) {
				body, err := options.ApplyFn(context)
				return "<form>" + body + "</form>", err
			},
		}, nil, nil},
		{"helper parameter context", "{{#form yehuda}}<p>{{name}}</p>{{/form}}", object("yehuda", object("name", "Yehuda")), "<form><p>Yehuda</p></form>", map[string]handlebars.Helper{
			"form": func(context any, options *handlebars.Options) (any, error) {
				body, err := options.ApplyFn(context)
				return "<form>" + body + "</form>", err
			},
		}, nil, nil},
	})
}
