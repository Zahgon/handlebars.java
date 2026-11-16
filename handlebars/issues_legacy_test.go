// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jtime"
)

func TestIssue42And1183(t *testing.T) {
	helpers := map[string]handlebars.Helper{"test.helper": func(context any, _ *handlebars.Options) (any, error) {
		return "Hello " + fmt.Sprint(context), nil
	}}
	runRenderCases(t, []renderCase{
		{"Issue42 null nested title", "{{#foo}}{{title}} {{#bar}}{{title}}{{/bar}}{{/foo}}", object("foo", object("title", "foo", "bar", object("title", nil))), "foo ", nil, nil, nil},
		{"Issue1183 namespaced helper", "\n{{#each names as |name|}}\n  {{test.helper name}}\n{{/each}}", object("names", []any{"John", "Jeff"}), "\n\n  Hello John\n\n  Hello Jeff\n", helpers, nil, nil},
	})
}

func TestIssue229ContextData(t *testing.T) {
	template, err := handlebars.New().CompileInline("{{@data.context}}")
	if err != nil {
		t.Fatalf("CompileInline(Issue229) error = %v; expected %q", err, "Ok!")
	}
	context := handlebars.NewContext(nil).SetData("data", object("context", "Ok!"))
	actual, applyErr := template.ApplyContext(context)
	if applyErr != nil || actual != "Ok!" {
		t.Errorf("ApplyContext(Issue229) = %q, %v; expected %q, nil", actual, applyErr, "Ok!")
	}
}

func TestIssue241DateHash(t *testing.T) {
	date, err := jtime.NewLocalDate(1999, 7, 16)
	if err != nil {
		t.Fatalf("NewLocalDate(1999-07-16) error = %v; expected nil", err)
	}
	configure := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	runRenderCases(t, []renderCase{{"hash format", "{{dateFormat date format=\"dd-MM-yyyy\"}}", object("date", date), "16-07-1999", nil, nil, configure}})
}

func TestIssue247PseudoVariables(t *testing.T) {
	template := "{{#each list1}}before each: {{@index}} {{@first}} {{@last}}\n{{#each list2}}  {{@index}} {{@first}} {{@last}}\n{{/each}}after each: {{@index}} {{@first}} {{@last}}\n{{/each}}"
	expected := "before each: 0 first \n  0 first \n  1  \n  2  last\nafter each: 0 first \nbefore each: 1  \n  0 first \n  1  \n  2  last\nafter each: 1  \nbefore each: 2  last\n  0 first \n  1  \n  2  last\nafter each: 2  last\n"
	runRenderCases(t, []renderCase{{"nested loop data", template, object("list1", []any{"i1", "i2", "i3"}, "list2", []any{"a", "b", "c"}), expected, nil, nil, nil}})
}

func TestIssue287And290Compilation(t *testing.T) {
	loader := handlebars.NewCompositeTemplateLoader(handlebars.NewFileTemplateLoader("."), handlebars.NewFileTemplateLoader("testdata"))
	for _, source := range []string{"{{issue287}}", "{{#each article-1-column}}{{/each}}", "{{#each article-1column}}{{/each}}"} {
		if _, err := handlebars.New(loader).CompileInline(source); err != nil {
			t.Errorf("CompileInline(%q) error = %v; expected nil", source, err)
		}
	}
}

func TestIssue289SubexpressionText(t *testing.T) {
	const source = "{{lowercase (concat \"string1\" \"string2\")}}"
	engine := handlebars.New()
	engine.RegisterHelper("lowercase", func(context any, _ *handlebars.Options) (any, error) { return context, nil })
	engine.RegisterHelper("concat", func(context any, _ *handlebars.Options) (any, error) { return context, nil })
	template, err := engine.CompileInline(source)
	if err != nil {
		t.Fatalf("CompileInline(%q) error = %v; expected text %q", source, err, source)
	}
	if actual := template.Text(); actual != source {
		t.Errorf("Text() = %q; expected %q", actual, source)
	}
}

func TestIssue237EnumMapAdaptation(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"enum map", "{{statuses.NEW}}", object("statuses", object("NEW", int32(10), "DONE", int32(20), "CLOSED", int32(3))), "10", nil, nil, nil},
		{"ordinary non-string key map", "{{statuses.NEW}}", object("statuses", map[int32]any{1: int32(10)}), "", nil, nil, nil},
	})
}
