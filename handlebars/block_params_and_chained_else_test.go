// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import "testing"

func TestBlockParameters(t *testing.T) {
	input := "{{#each users as |user userId|}}\n  Id: {{userId}} Name: {{user.name}}\n{{/each}}"
	runRenderCases(t, []renderCase{
		{"each named index", input, object("users", []any{object("name", "Pedro"), object("name", "Pablo")}), "\n  Id: 0 Name: Pedro\n\n  Id: 1 Name: Pablo\n", nil, nil, nil},
		{"each named key", input, object("users", []any{object("name", "Pedro")}), "\n  Id: 0 Name: Pedro\n", nil, nil, nil},
		{"with", "{{#with title as |value|}}{{value}}{{/with}}", object("title", "Block Param"), "Block Param", nil, nil, nil},
	})
}

func TestChainedElseBlocks(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"else if", "{{#people}}{{name}}{{else if none}}{{none}}{{/people}}", object("none", "No people"), "No people", nil, nil, nil},
		{"else if unless", "{{#people}}{{name}}{{else if nothere}}fail{{else unless nothere}}{{none}}{{/people}}", object("none", "No people"), "No people", nil, nil, nil},
		{"else if else", "{{#people}}{{name}}{{else if none}}{{none}}{{else}}fail{{/people}}", object("none", "No people"), "No people", nil, nil, nil},
		{"standalone inverse", "{{#people}}{{name}}{{else if none}}{{none}}{{^}}{{/people}}", object("none", "No people"), "No people", nil, nil, nil},
	})
}
