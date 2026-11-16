// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestSourceText(t *testing.T) {
	sources := []string{
		" {{foo}} ", "{{foo}}", "{{ foo }}", "{{  foo  }}",
		"{{&var}}", "{{& var }}", "{{&  var  }}",
		"{{{var}}}", "{{{ var }}}", "{{{  var  }}}",
		"{{#section}}content{{/section}}", "{{# section }}content{{/ section }}",
		"{{^section}}content{{/section}}", "{{^ section }}content{{/ section }}",
		"{{>user}}", "{{> user }}", "{{>user context}}", "{{> user context }}",
		"{{with context arg0 hash=hash0}}", "{{ with context arg0 hash=hash0 }}",
		"{{#with context}}content{{/with}}", "{{# with context }}content{{/ with }}",
		"{{\tfoo\t}}", "{{ foo }}  {{  bar  }}", "{{  \tfoo\t  }}",
		"Hello {{ name }}, welcome!",
	}
	partials := handlebars.NewMapTemplateLoader(map[string]string{"user": "{{user}}"})
	for _, source := range sources {
		template, err := handlebars.New(partials).CompileInline(source)
		if err != nil {
			t.Errorf("CompileInline(%q) error = %v; expected nil and Text() %q", source, err, source)
			continue
		}
		if actual := template.Text(); actual != source {
			t.Errorf("CompileInline(%q).Text() = %q; expected %q", source, actual, source)
		}
	}
}

func TestWhitespaceChannel(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"compact variable", "{{foo}}", object("foo", "A"), "A", nil, nil, nil},
		{"spaced variable", "{{ foo }}", object("foo", "A"), "A", nil, nil, nil},
		{"double spaced variable", "{{  foo  }}", object("foo", "A"), "A", nil, nil, nil},
		{"triple spaced variable", "{{   foo   }}", object("foo", "A"), "A", nil, nil, nil},
		{"whitespace trim", "{{~foo~}}  {{~bar~}}", object("foo", "A", "bar", "B"), "AB", nil, nil, nil},
		{"whitespace retained", "{{foo}}  {{bar}}", object("foo", "A", "bar", "B"), "A  B", nil, nil, nil},
		{"compact block", "{{#if condition}}yes{{/if}}", object("condition", true), "yes", nil, nil, nil},
		{"spaced block", "{{# if condition }}yes{{/ if }}", object("condition", true), "yes", nil, nil, nil},
		{"double spaced block", "{{#  if  condition  }}yes{{/  if  }}", object("condition", true), "yes", nil, nil, nil},
	})
	compileOnly := []string{
		"{{foo}}  {{bar}}", "{{ foo }}  {{ bar }}",
		"{{#each items}}{{name}}{{/each}}", "{{# each items }}{{ name }}{{/ each }}",
		"{{# each items }}\n  {{ name }}  :  {{ value }}\n{{/ each }}", "{{~foo~}}",
	}
	for _, source := range compileOnly {
		if _, err := handlebars.New().CompileInline(source); err != nil {
			t.Errorf("CompileInline(%q) error = %v; expected nil", source, err)
		}
	}
}
