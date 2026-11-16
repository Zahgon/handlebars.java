// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestEscaping(t *testing.T) {
	cases := []struct {
		name, input, expected string
	}{
		{"ampersand", "&", "&amp;"},
		{"less than", "<", "&lt;"},
		{"greater than", ">", "&gt;"},
		{"double quote", "\"", "&quot;"},
		{"apostrophe", "'", "&#x27;"},
		{"backtick", "`", "&#x60;"},
		{"equals", "=", "&#x3D;"},
		{"plain", "Handlebars.java", "Handlebars.java"},
	}
	for _, test := range cases {
		if actual := handlebars.EscapeExpression(test.input); actual != test.expected {
			t.Errorf("%s: EscapeExpression(%q) = %q; expected %q", test.name, test.input, actual, test.expected)
		}
	}
	runRenderCases(t, []renderCase{
		{"double escape", "{{value}}", object("value", "Escaped, <b> looks like: &lt;b&gt;"), "Escaped, &lt;b&gt; looks like: &amp;lt;b&amp;gt;", nil, nil, nil},
		{"safe string", "{{value}}", object("value", handlebars.NewSafeString("<b>safe</b>")), "<b>safe</b>", nil, nil, nil},
		{"triple", "{{{value}}}", object("value", "<b>raw</b>"), "<b>raw</b>", nil, nil, nil},
	})
}
