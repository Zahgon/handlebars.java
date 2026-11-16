// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import "testing"

func TestLookupHelper(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"dynamic property", "{{lookup object key}}", object("object", object("name", "found"), "key", "name"), "found", nil, nil, nil},
		{"nested dynamic property", "{{lookup object key}}", object("object", object("person", object("name", "found")), "key", "person.name"), "found", nil, nil, nil},
		{"missing dynamic key fallback", "{{#with (lookup object person.id)}}{{name}}{{/with}}", object("object", object("name", "fallback"), "person", object()), "fallback", nil, nil, nil},
		{"null dynamic key fallback", "{{#with (lookup object key)}}{{name}}{{/with}}", object("object", object("name", "fallback"), "key", nil), "fallback", nil, nil, nil},
		{"null parent key in each", "{{#each objects}}{{#with (lookup this ../key)}}{{name}};{{/with}}{{/each}}", object("objects", []any{object("name", "A"), object("name", "B")}, "key", nil), "A;B;", nil, nil, nil},
		{"false property", "{{lookup object key}}", object("object", object("value", false), "key", "value"), "false", nil, nil, nil},
		{"zero property", "{{lookup object key}}", object("object", object("value", int32(0)), "key", "value"), "0", nil, nil, nil},
		{"empty property", "{{lookup object key}}", object("object", object("value", ""), "key", "value"), "", nil, nil, nil},
	})
}
