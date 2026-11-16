// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestGlobalDelimiters(t *testing.T) {
	cases := []struct {
		name, start, end, input, expected string
	}{
		{"angle", "<%", "%>", "Hello <%name%>!", "Hello Handlebars!"},
		{"square", "[[", "]]", "Hello [[name]]!", "Hello Handlebars!"},
		{"mixed literal defaults", "[[", "]]", "{{name}} [[name]]", "{{name}} Handlebars"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			engine := handlebars.New().StartDelimiter(test.start).EndDelimiter(test.end)
			template, err := engine.CompileInline(test.input)
			if err != nil {
				t.Fatalf("CompileInline(%q) error: %v; expected %q", test.input, err, test.expected)
			}
			actual, err := template.Apply(object("name", "Handlebars"))
			if err != nil || actual != test.expected {
				t.Errorf("Apply(%q) = %q, %v; expected %q, nil", test.input, actual, err, test.expected)
			}
		})
	}
}

func TestSetDelimiterTags(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"switch", "{{=<% %>=}}<%name%>", object("name", "Handlebars"), "Handlebars", nil, nil, nil},
		{"switch back", "{{=<% %>=}}<%name%><%={{ }}=%>{{name}}", object("name", "H"), "HH", nil, nil, nil},
	})
}
