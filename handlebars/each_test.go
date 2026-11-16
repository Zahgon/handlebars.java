// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import "testing"

func TestEachAndPseudoVariables(t *testing.T) {
	pseudo := "{{#list}}i={{@index}}\neven={{@even}}\nodd={{@odd}}\nfirst={{@first}}\nlast={{@last}}\ni+1={{@index_1}}\n{{/list}}"
	runRenderCases(t, []renderCase{
		{"implicit array", "Hello {{#list}}{{this}}{{/list}}!", object("list", []any{"w", "o", "r", "l", "d"}), "Hello world!", nil, nil, nil},
		{"each array", "Hello {{#each list}}{{this}}{{/each}}!", object("list", []any{"w", "o", "r", "l", "d"}), "Hello world!", nil, nil, nil},
		{"each inverse", "{{#each list}}bad{{else}}empty{{/each}}", object("list", []any{}), "empty", nil, nil, nil},
		{"metadata", pseudo, object("list", []any{"a", "b", "c"}), "i=0\neven=even\nodd=\nfirst=first\nlast=\ni+1=1\ni=1\neven=\nodd=odd\nfirst=\nlast=\ni+1=2\ni=2\neven=even\nodd=\nfirst=\nlast=last\ni+1=3\n", nil, nil, nil},
		{"base index", "{{#each list base=1}}{{@index}}:{{@odd}}:{{@even}};{{/each}}", object("list", []any{"a", "b"}), "1:odd:;2::even;", nil, nil, nil},
		{"block params", "{{#each list as |value index|}}{{index}}={{value}};{{/each}}", object("list", []any{"a", "b"}), "0=a;1=b;", nil, nil, nil},
		{"parent survives", "{{#parent}}{{#list}}{{@index}}. {{name}} {{/list}}{{/parent}}", object("parent", object("name", "px", "list", []any{"a", "b"})), "0. px 1. px ", nil, nil, nil},
	})
}
