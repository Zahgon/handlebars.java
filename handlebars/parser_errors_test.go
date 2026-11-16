// Handlebars.java: https://github.com/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import "testing"

func TestParserErrorMessages(t *testing.T) {
	tests := []struct {
		name, source, message string
	}{
		{"unterminated variable", "{{value", "test.hbs:1:7: found: '<EOF>', expected: '}}'\n{{value\n       ^"},
		{"hash after hash", "{{f param hashx=1 param}}", "test.hbs:1:23: found: '}}', expected: '='\n{{f param hashx=1 param}}\n                       ^"},
		{"unterminated string", "{{a \"unterminated}}\nmore {{x}}", "test.hbs:1:4: found: '\"unterminated}}\\nmore {{x}}'\n{{a \"unterminated}}\n    ^\n"},
		{"code point column", "😀 {{λ}}", "test.hbs:1:4: found: 'λ'\n😀 {{λ}}\n    ^\n"},
		{"crlf column", "a\r\nb {{a b) }}", "test.hbs:2:8: found: ')', expected: '}}'\nb {{a b) }}\n        ^"},
		{"set delimiter eof", "{{=<% %>}}", "test.hbs:1:10: found: '<EOF>', expected: '=}}'\n{{=<% %>}}\n          ^"},
		{"outer block checked first", "{{#a}}{{#b}}{{/a}}{{/b}}", "test.hbs:1:21: found: 'b', expected: 'a'\n{{#a}}{{#b}}{{/a}}{{/b}}\n                     ^\n"},
		{"empty block parameter", "{{#each a as | |}}{{/each}}", "test.hbs:1:15: found: '|', expected: 'id'\n{{#each a as | |}}{{/each}}\n               ^"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().CompileSource(NewStringTemplateSource("test.hbs", test.source))
			if err == nil {
				t.Fatal("expected a parser error")
			}
			if got := err.Error(); got != test.message {
				t.Fatalf("message mismatch\nwant: %q\n got: %q", test.message, got)
			}
		})
	}
}
