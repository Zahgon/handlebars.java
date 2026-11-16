// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import "testing"

func TestPartials(t *testing.T) {
	dudes := []any{object("name", "Yehuda", "url", "http://yehuda"), object("name", "Alan", "url", "http://alan")}
	runRenderCases(t, []renderCase{
		{"basic", "Dudes: {{#dudes}}{{> dude}}{{/dudes}}", object("dudes", dudes), "Dudes: Yehuda (http://yehuda) Alan (http://alan) ", nil, map[string]string{"dude": "{{name}} ({{url}}) "}, nil},
		{"explicit context", "Dudes: {{>dude dudes}}", object("dudes", dudes), "Dudes: Yehuda (http://yehuda) Alan (http://alan) ", nil, map[string]string{"dude": "{{#this}}{{name}} ({{url}}) {{/this}}"}, nil},
		{"nested", "Dudes: {{#dudes}}{{>dude}}{{/dudes}}", object("dudes", dudes), "Dudes: Yehuda <a href='http://yehuda'>http://yehuda</a> Alan <a href='http://alan'>http://alan</a> ", nil, map[string]string{"dude": "{{name}} {{> url}} ", "url": "<a href='{{url}}'>{{url}}</a>"}, nil},
		{"preceding selector", "Dudes: {{>dude}} {{another_dude}}", object("name", "Jeepers", "another_dude", "Creepers"), "Dudes: Jeepers Creepers", nil, map[string]string{"dude": "{{name}}"}, nil},
		{"literal name", "Dudes: {{> [dude]}}", object("name", "Jeepers"), "Dudes: Jeepers", nil, map[string]string{"dude": "{{name}}"}, nil},
		{"inline", "{{#*inline \"myPartial\"}}success{{/inline}}{{> myPartial}}", object(), "success", nil, nil, nil},
		{"inline overwrite", "{{#*inline \"myPartial\"}}fail{{/inline}}{{#*inline \"myPartial\"}}success{{/inline}}{{> myPartial}}", object(), "success", nil, nil, nil},
		{"inline overrides global", "{{#*inline \"myPartial\"}}success{{/inline}}{{> myPartial}}", object(), "success", nil, map[string]string{"myPartial": "fail"}, nil},
		{"inline reaches partial", "{{#*inline \"myPartial\"}}success{{/inline}}{{> dude}}", object(), "success", nil, map[string]string{"dude": "{{> myPartial}}"}, nil},
		{"block default", "{{#> dude}}success{{/dude}}", object(), "success", nil, nil, nil},
		{"block existing", "{{#> dude}}fail{{/dude}}", object(), "success", nil, map[string]string{"dude": "success"}, nil},
		{"partial block", "{{#> dude}}success{{/dude}}", object(), "success", nil, map[string]string{"dude": "{{> @partial-block }}"}, nil},
	})
}

func TestPartialAndInlineTemplateText(t *testing.T) {
	cases := []string{
		"{{#*inline \"myPartial\"}}success{{/inline}}{{> myPartial}}",
		"{{#> dude}}{{#*inline \"myPartial\"}}success{{/inline}}{{/dude}}",
		"{{#> dude}}success{{/dude}}",
	}
	for _, input := range cases {
		template, err := newInlineTemplate(input)
		if err != nil {
			t.Fatalf("CompileInline(%q) error: %v; expected text %q", input, err, input)
		}
		if actual := template.Text(); actual != input {
			t.Errorf("Text() = %q; expected %q", actual, input)
		}
	}
}
