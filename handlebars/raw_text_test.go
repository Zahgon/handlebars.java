// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

// Nashorn precompilation and JavaScript helper loading are omitted because Go has no JavaScript engine counterpart.

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestTemplateText(t *testing.T) {
	inputs := []string{
		"Plain Text!",
		"hello {{var}}!",
		"hello {{& var}}!",
		"hello {{{ var }}}!",
		"hello {{#section}} {{/section}}!",
		"hello {{#section}} hello {{/section}}!",
		"hello {{^section}} {{/section}}!",
		"hello {{^section}} hello {{var}}! {{/section}}!",
		"hello {{>user}}!",
		"hello {{>user context}}!",
		"hello {{with context arg0 hash=hash0}}!",
		"hello {{#with context arg0 hash=hash0}}hah{{/with}}!",
		"{{{{raw}}}} {{test}} {{{{/raw}}}}",
	}
	for _, input := range inputs {
		template, err := handlebars.New().CompileInline(input)
		if err != nil {
			t.Fatalf("CompileInline(%q) error: %v; expected Text %q", input, err, input)
		}
		if actual := template.Text(); actual != input {
			t.Errorf("Text(%q) = %q; expected %q", input, actual, input)
		}
	}
}

func TestRawHelper(t *testing.T) {
	raw := func(context any, options *handlebars.Options) (any, error) { return options.ApplyFn(context) }
	rawWithParameters := func(context any, options *handlebars.Options) (any, error) {
		body, err := options.ApplyFn(context)
		return body + fmt.Sprint(context) + fmt.Sprint(options.Param(0)) + fmt.Sprint(options.Param(1)), err
	}
	runRenderCases(t, []renderCase{
		{"body is raw", "{{{{raw}}}} {{test}} {{{{/raw}}}}", object(), " {{test}} ", map[string]handlebars.Helper{"raw": raw}, nil, nil},
		{"parameters", "{{{{raw 1 2 3}}}} {{test}} {{{{/raw}}}}", object(), " {{test}} 123", map[string]handlebars.Helper{"raw": rawWithParameters}, nil, nil},
	})
}
