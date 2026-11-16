// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestParsingErrors(t *testing.T) {
	cases := []struct {
		name, input, expected string
		partials              map[string]string
	}{
		{"unclosed variable", "{{value", "inline@432f1e11:1:7: found: '<EOF>', expected: '}}'\n{{value\n       ^", nil},
		{"wrong closing block", "{{#block}}{{/nan}}", "inline@4e2b08aa:1:13: found: 'nan', expected: 'block'\n{{#block}}{{/nan}}\n             ^\n", nil},
		{"unclosed block", "{{#block}}invalid block", "inline@4f26a966:1:23: found: '<EOF>', expected: '{{/'\n{{#block}}invalid block\n                       ^", nil},
		{"delimiter variable unclosed", "{{=<% %>=}} <%Hello", "inline@4ad1f18d:1:19: found: '<EOF>', expected: '%>'\n{{=<% %>=}} <%Hello\n                   ^", nil},
		{"missing partial", "{{> missingPartial}}", "inline@3a6836e3:1:4: The partial '/missingPartial.hbs' at '/missingPartial.hbs' could not be found", nil},
		{"unclosed section", "{{#value}}", "inline@223e2552:1:10: found: '<EOF>', expected: '{{/'\n{{#value}}\n          ^", nil},
		{"invalid helper", "\n{{tag message.from \"user\"}}\n", "inline@51a47cf6:2:2: could not find helper: 'tag'\n{{tag message.from \"user\"}}\n  ^\n", nil},
		{"unbalanced delimiter", "{{=<%%>=}}", "inline@4121bfe:1:7: found: '=}}', expected: 'space'\n{{=<%%>=}}\n       ^", nil},
		{"hash before positional", "{{f param hashx=1 param}}", "inline@653c9fde:1:23: found: '}}', expected: '='\n{{f param hashx=1 param}}\n                       ^", nil},
		{"unclosed triple", "{{{tvar", "inline@437e59ce:1:7: found: '<EOF>', expected: '}}}'\n{{{tvar\n       ^", nil},
		{"custom unclosed triple", "{{=** **=}}**{tvar", "inline@6ed0e9ac:1:18: found: '<EOF>', expected: '}**'\n{{=** **=}}**{tvar\n                  ^", nil},
		{"unclosed ampersand", "{{&tvar", "inline@3ed08bf9:1:7: found: '<EOF>', expected: '}}'\n{{&tvar\n       ^", nil},
		{"custom unclosed ampersand", "{{=** **=}}**&tvar", "inline@737eb781:1:18: found: '<EOF>', expected: '**'\n{{=** **=}}**&tvar\n                  ^", nil},
		{"missing identifier close", "{{is", "inline@39c48a:1:4: found: '<EOF>', expected: '}}'\n{{is\n    ^", nil},
		{"absolute partial", "{{> /user}}", "inline@5253d058:1:4: found: '/', partial shouldn't start with '/'\n{{> /user}}\n    ^\n", nil},
		{"absolute nested partial", "{{> /layout/base}}", "inline@69ef1f97:1:4: found: '/', partial shouldn't start with '/'\n{{> /layout/base}}\n    ^\n", nil},
		{"index without separator", "{{list[0]}}", "inline@602227d6:1:2: could not find helper: 'list'\n{{list[0]}}\n  ^\n", nil},
		{"bad index", "{{list.[0}}", "inline@6289667b:1:7: found: '[0}}'\n{{list.[0}}\n       ^\n", nil},
		{"empty index", "{{list.[]}}", "inline@6288bd8e:1:7: found: '[]'\n{{list.[]}}\n       ^\n", nil},
		{"open index", "{{list.[}}", "inline@13b1d0b5:1:7: found: '[}}'\n{{list.[}}\n       ^\n", nil},
		{"multiple else", "{{#if true}} b1 {{else}} b2 {{else}} b3 {{/if}}", "inline@39492724:1:28: found: '{{', expected: '{{/'\n{{#if true}} b1 {{else}} b2 {{else}} b3 {{/if}}\n                            ^", nil},
		{"partial level one", "{{> p1}}", "/p1.hbs:1:7: found: '<EOF>', expected: '}}'\n{{value\n       ^", map[string]string{"p1": "{{value"}},
		{"partial level two", "{{> deep1}}", "/deep1.hbs:1:10: found: '<EOF>', expected: '}}'\n {{> deep2\n          ^", map[string]string{"deep1": " {{> deep2"}},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			engine := handlebars.New(handlebars.NewMapTemplateLoader(test.partials))
			template, err := engine.CompileInline(test.input)
			if err == nil {
				_, err = template.Apply(object())
			}
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if actual != test.expected {
				t.Errorf("Compile/apply(%q) error = %q; expected %q", test.input, actual, test.expected)
			}
		})
	}
}
