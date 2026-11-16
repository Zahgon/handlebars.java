// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"errors"
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestCollectionsLength(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"list length", "{{this.length}}", []any{"a"}, "1", nil, nil, nil},
		{"explicit length property", "{{this.length}}", object("length", int32(5), "size", int32(4)), "5", nil, nil, nil},
		{"explicit size property", "{{this.size}}", object("length", int32(5), "size", int32(4)), "4", nil, nil, nil},
	})
}

func TestEachObjectWithLast(t *testing.T) {
	goodbyes := jvm.NewLinkedHashMap()
	goodbyes.Put("foo", object("text", "goodbye"))
	goodbyes.Put("bar", object("text", "Goodbye"))
	runRenderCases(t, []renderCase{{
		"last map entry", "{{#each goodbyes}}{{#if @last}}{{text}}! {{/if}}{{/each}}cruel {{world}}!",
		object("goodbyes", goodbyes, "world", "world"), "Goodbye! cruel world!", nil, nil, nil,
	}})
}

func TestEnumHelpers(t *testing.T) {
	helpers := map[string]handlebars.Helper{}
	for _, name := range []string{"h1", "h2", "h3"} {
		value := name
		helpers[name] = func(any, *handlebars.Options) (any, error) { return value, nil }
	}
	runRenderCases(t, []renderCase{
		{"h1", "{{h1}}", object(), "h1", helpers, nil, nil},
		{"h2", "{{h2}}", object(), "h2", helpers, nil, nil},
		{"h3", "{{h3}}", object(), "h3", helpers, nil, nil},
	})
}

func TestMissingValueResolver(t *testing.T) {
	engine := handlebars.New()
	engine.RegisterHelper("helperMissing", func(any, *handlebars.Options) (any, error) { return "(none)", nil })
	template, err := engine.CompileInline("{{missingVar}}")
	if err != nil {
		t.Fatalf("CompileInline(missingVar) error = %v; expected output %q", err, "(none)")
	}
	actual, applyErr := template.Apply(object())
	if applyErr != nil || actual != "(none)" {
		t.Errorf("Apply(missingVar) = %q, %v; expected %q, nil", actual, applyErr, "(none)")
	}

	engine = handlebars.New()
	engine.RegisterHelper("helperMissing", func(any, *handlebars.Options) (any, error) {
		return nil, errors.New("Missing variable: missingVar")
	})
	template, err = engine.CompileInline("{{missingVar}}")
	if err != nil {
		t.Fatalf("CompileInline(throwing missingVar) error = %v; expected apply error %q", err, "Missing variable: missingVar")
	}
	_, applyErr = template.Apply(object())
	if applyErr == nil || !strings.Contains(applyErr.Error(), "Missing variable: missingVar") {
		t.Errorf("Apply(throwing missingVar) error = %q; expected error containing %q", errorText(applyErr), "Missing variable: missingVar")
	}

	engine = handlebars.New()
	engine.RegisterHelper("helperMissing", func(any, *handlebars.Options) (any, error) { return "nousers", nil })
	template, err = engine.CompileInline("{{#userss}} <tr> <td>{{fullName}}</td> <td>{{jobTitle}}</td> </tr> {{/userss}}")
	if err != nil {
		t.Fatalf("CompileInline(Issue367) error = %v; expected output %q", err, "nousers")
	}
	actual, applyErr = template.Apply(object())
	if applyErr != nil || actual != "nousers" {
		t.Errorf("Apply(Issue367 missing section) = %q, %v; expected %q, nil", actual, applyErr, "nousers")
	}
}

func TestEmptyTemplate(t *testing.T) {
	template, err := handlebars.New().CompileInline("")
	if err != nil {
		t.Fatalf("CompileInline(empty) error = %v; expected nil", err)
	}
	for _, data := range []any{nil, object(), "value"} {
		actual, applyErr := template.Apply(data)
		if applyErr != nil || actual != "" {
			t.Errorf("Apply(%#v) = %q, %v; expected %q, nil", data, actual, applyErr, "")
		}
	}
}

func TestNoFalsyValues(t *testing.T) {
	template, err := handlebars.New().CompileInline("{{#if this}}true{{else}}false{{/if}}")
	if err != nil {
		t.Fatalf("CompileInline(no-falsy) error = %v; expected nil", err)
	}
	values := []any{true, "Hi", int32(1), []any{int32(1)}, object("value", int32(1))}
	for _, value := range values {
		actual, applyErr := template.Apply(value)
		if applyErr != nil || actual != "true" {
			t.Errorf("Apply(%T(%v)) = %q, %v; expected %q, nil", value, value, actual, applyErr, "true")
		}
	}
}

func TestOverrideBuiltinHelper(t *testing.T) {
	engine := handlebars.New()
	engine.RegisterHelper("each", func(any, *handlebars.Options) (any, error) { return "custom", nil })
	template, err := engine.CompileInline("{{#each this}}{{this}}{{/each}}")
	if err != nil {
		t.Fatalf("CompileInline(override each) error = %v; expected output %q", err, "custom")
	}
	actual, applyErr := template.Apply(object("hash", []any{int32(1), int32(2), int32(3)}))
	if applyErr != nil || actual != "custom" {
		t.Errorf("Apply(override each) = %q, %v; expected %q, nil", actual, applyErr, "custom")
	}
}

func TestTemplateNotFound(t *testing.T) {
	_, err := handlebars.New().Compile("template-that-does-not-exist")
	if err == nil {
		t.Errorf("Compile(missing) error = %q; expected a non-nil missing-template error", errorText(err))
	}

	template, compileErr := handlebars.New().CompileInline("{{> text}}")
	if compileErr != nil {
		t.Fatalf("CompileInline(missing partial) error = %v; expected apply-time missing-partial error", compileErr)
	}
	_, err = template.Apply(nil)
	if err == nil {
		t.Errorf("Apply(missing partial) error = %q; expected a non-nil missing-partial error", errorText(err))
	}
}

func TestToStringEscaping(t *testing.T) {
	template, err := handlebars.New().CompileInline("{{this}}")
	if err != nil {
		t.Fatalf("CompileInline(toString) error = %v; expected output %q", err, "&lt;h1&gt;Hello&lt;/h1&gt;")
	}
	actual, applyErr := template.Apply("<h1>Hello</h1>")
	if applyErr != nil || actual != "&lt;h1&gt;Hello&lt;/h1&gt;" {
		t.Errorf("Apply(unsafe string) = %q, %v; expected %q, nil", actual, applyErr, "&lt;h1&gt;Hello&lt;/h1&gt;")
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
