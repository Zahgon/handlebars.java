// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

type renderCase struct {
	name      string
	template  string
	data      any
	expected  string
	helpers   map[string]handlebars.Helper
	partials  map[string]string
	configure func(*handlebars.Handlebars)
}

func runRenderCases(t *testing.T, cases []renderCase) {
	t.Helper()
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			engine := handlebars.New(handlebars.NewMapTemplateLoader(test.partials))
			if test.configure != nil {
				test.configure(engine)
			}
			for name, helper := range test.helpers {
				engine.RegisterHelper(name, helper)
			}
			template, err := engine.CompileInline(test.template)
			if err != nil {
				t.Fatalf("CompileInline(%q) error: %v; expected output %q", test.template, err, test.expected)
			}
			actual, err := template.Apply(test.data)
			if err != nil {
				t.Fatalf("Apply(%q) error: %v; expected output %q", test.template, err, test.expected)
			}
			if actual != test.expected {
				t.Errorf("Apply(%q) = %q; expected %q", test.template, actual, test.expected)
			}
		})
	}
}

func object(values ...any) map[string]any {
	result := make(map[string]any, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		result[values[i].(string)] = values[i+1]
	}
	return result
}

func newInlineTemplate(input string) (handlebars.Template, error) {
	return handlebars.New().CompileInline(input)
}
