// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestIssues(t *testing.T) {
	conditional := func(engine *handlebars.Handlebars) { engine.RegisterConditionalHelpers() }
	stringHelpers := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	runRenderCases(t, []renderCase{
		{"Issue109 empty string section", "{{#empty}}truthy{{/empty}}", object("empty", ""), "", nil, nil, nil},
		{"Issue109 empty string inverted", "{{^empty}}falsy{{/empty}}", object("empty", ""), "falsy", nil, nil, nil},
		{"Issue109 empty string if", "{{#if empty}}truthy{{else}}falsy{{/if}}", object("empty", ""), "falsy", nil, nil, nil},
		{"Issue109 nonempty if", "{{#if nonempty}}truthy{{/if}}", object("nonempty", "xyz"), "truthy", nil, nil, nil},
		{"Issue109 null inverted", "{{^null}}falsy{{/null}}", object(), "falsy", nil, nil, nil},
		{"Issue109 null if", "{{#if null}}truthy{{else}}falsy{{/if}}", object(), "falsy", nil, nil, nil},
		{"Issue123 block trailing space", "{{#if \"stuff\" }}Bingo{{/if}}", object(), "Bingo", nil, nil, nil},
		{"Issue123 block doubled space", "{{#if \"stuff\"  }}Bingo{{/if}}", object(), "Bingo", nil, nil, nil},
		{"Issue123 block name space", "{{# if \"stuff\"}}Bingo{{/if}}", object(), "Bingo", nil, nil, nil},
		{"Issue123 close name space", "{{#if \"stuff\"}}Bingo{{/ if}}", object(), "Bingo", nil, nil, nil},
		{"Issue123 variable spaces", "{{ var }}", object(), "", nil, nil, nil},
		{"Issue133 null helper argument", "{{times nullvalue 3}}", object("nullvalue", nil), "", map[string]handlebars.Helper{"times": repeatHelper}, nil, nil},
		{"Issue133 helper argument", "{{times value 3}}", object("value", "a"), "aaa", map[string]handlebars.Helper{"times": repeatHelper}, nil, nil},
		{"Issue181 block content", "{{#block \"name\"}}block{{/block}}", object(), "block", nil, nil, nil},
		{"Issue181 block no content", "{{block \"name\"}}", object(), "", nil, nil, nil},
		{"Issue181 registered partial", "{{#partial \"name\"}}partial{{/partial}}{{#block \"name\"}}block{{/block}}", object(), "partial", nil, nil, nil},
		{"Issue181 empty partial", "{{partial \"name\"}}{{#block \"name\"}}block{{/block}}", object(), "block", nil, nil, nil},
		{"Issue319 block hash", "{{#partial \"hello\"}}Hello, {{name}}{{/partial}}{{#block \"hello\" name=\"Fred\"}}{{/block}}", object(), "Hello, Fred", nil, nil, nil},
		{"Issue319 fallback hash", "{{#block \"hello\" name=\"Fred\"}}Hello, {{name}}{{/block}}", object(), "Hello, Fred", nil, nil, nil},
		{"Issue319 partial hash", "{{#partial \"hello\" name=\"Edgar\"}}{{/partial}}{{#block \"hello\" name=\"Fred\"}}Hello, {{name}}{{/block}}", object(), "Hello, Edgar", nil, nil, nil},
		{"Issue463 parent scope disabled", "Hello {{#child}}{{value}}{{bestQB}}{{/child}}", object("value", "Brett", "child", object("bestQB", "Favre")), "Hello Favre", nil, nil, func(engine *handlebars.Handlebars) { engine.ParentScopeResolution(false) }},
		{"Issue468 raw helper", "{{{{raw-helper}}}}{{bar}}{{{{/raw-helper}}}}", object(), "{{bar}}", map[string]handlebars.Helper{
			"raw-helper": func(context any, options *handlebars.Options) (any, error) { return options.ApplyFn(context) },
		}, nil, nil},
		{"Issue471 nested implicit collections", "{{#this}}{{val}}{{#subDum}}{{val}}{{/subDum}}{{/this}}", []any{object("val", "d1", "subDum", []any{object("val", "d3")}), object("val", "d2", "subDum", []any{object("val", "d4")})}, "d1d3d2d4", nil, nil, nil},
		{"Issue473 with parent lookup", "{{#with group}}{{needsPano}}{{/with}}", object("needsPano", true, "group", object("title", "test")), "true", nil, nil, nil},
		{"Issue473 implicit parent lookup", "{{#group}}{{needsPano}}{{/group}}", object("needsPano", true, "group", object("title", "test")), "true", nil, nil, nil},
		{"Issue682 trusted formatter", "{{this}}", "as\"df", "as\"df", nil, nil, func(engine *handlebars.Handlebars) {
			engine.WithFormatter(handlebars.FormatterFunc(func(value any) (any, bool) {
				return handlebars.NewSafeString(value), true
			}))
		}},
		{"Issue714 empty cut", "{{cut value \"-\"}}", object("value", ""), "", nil, nil, stringHelpers},
		{"Issue714 date cut", "{{cut value \"-\"}}", object("value", "2019-12-30"), "20191230", nil, nil, stringHelpers},
		{"Issue827 greater mixed numbers", "{{gt 5 2.0}}", object(), "true", nil, nil, conditional},
		{"Issue827 unequal mixed numbers", "{{eq 5 2.0}}", object(), "false", nil, nil, conditional},
		{"Issue827 equal mixed numbers", "{{eq 2 2.0}}", object(), "true", nil, nil, conditional},
		{"Issue191 long comment", "{{!--{{var}}--}}", object(), "", nil, nil, nil},
		{"Issue251 trim variable left", "{{~tilde}}", object(), "", nil, nil, nil},
		{"Issue251 trim variable right", "{{tilde ~}}", object(), "", nil, nil, nil},
		{"Issue251 trim variable both", "{{~tilde ~}}", object(), "", nil, nil, nil},
		{"Issue251 trim ampersand", "{{~&tilde~}}", object(), "", nil, nil, nil},
		{"Issue251 trim triple", "{{~{tilde}~}}", object(), "", nil, nil, nil},
		{"Issue251 trim block", "{{~#tilde~}}{{~/tilde~}}", object(), "", nil, nil, nil},
		{"Issue294 escaped variable", "\\{{foo}}", object(), "{{foo}}", nil, nil, nil},
		{"Issue294 escaped variable in text", "before \\{{foo}} after", object(), "before {{foo}} after", nil, nil, nil},
		{"Issue310 long comment suffix", "{{!-- not a var}} --}}", object(), "", nil, nil, nil},
		{"Issue310 short comment nesting", "{{! {{not}} a var}}", object(), " a var}}", nil, nil, nil},
		{"Issue355 zero number", "{{numberFormat 0}}", object(), "0", nil, nil, stringHelpers},
		{"Issue355 zero currency", "{{numberFormat 0 'currency'}}", object(), "$0.00", nil, nil, stringHelpers},
		{"Issue379 bracketed text", "<div> [a:b] </div>", object(), "<div> [a:b] </div>", nil, nil, nil},
		{"Issue404 escaped variable quoted", "\"\\{{var}}\"", object(), "\"{{var}}\"", nil, nil, nil},
		{"Issue404 escaped variable attribute", "<tag attribute=\"\\{{var}}\"/>", object(), "<tag attribute=\"{{var}}\"/>", nil, nil, nil},
		{"Issue573 decimal 1.5 truthy", "{{#if 1.5}}OK{{/if}}", object(), "OK", nil, nil, nil},
		{"Issue573 decimal 0.5 truthy", "{{#if 0.5}}OK{{/if}}", object(), "OK", nil, nil, nil},
		{"Issue573 leading-dot decimal truthy", "{{#if .6}}OK{{/if}}", object(), "OK", nil, nil, nil},
	})
}

func repeatHelper(context any, options *handlebars.Options) (any, error) {
	value, _ := context.(string)
	count := int32(0)
	switch number := options.Param(0).(type) {
	case int32:
		count = number
	case int:
		count = int32(number)
	}
	return strings.Repeat(value, int(count)), nil
}
