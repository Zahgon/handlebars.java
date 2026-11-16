// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestHandlebarsJSBuiltinsAndHelpers(t *testing.T) {
	goodbyes := jvm.NewLinkedHashMap()
	goodbyes.Put("<b>#1</b>", object("text", "goodbye"))
	goodbyes.Put("2", object("text", "GOODBYE"))
	helpers := map[string]handlebars.Helper{
		"goodbye": func(context any, options *handlebars.Options) (any, error) {
			resolved, resolveErr := options.Resolve("goodbye")
			if resolveErr != nil {
				return nil, resolveErr
			}
			value, _ := resolved.(string)
			if value == "" {
				value, _ = context.(map[string]any)["goodbye"].(string)
			}
			body, err := options.ApplyFn(context)
			return strings.ToUpper(value) + body, err
		},
		"cruel": func(context any, _ *handlebars.Options) (any, error) {
			return "cruel " + strings.ToUpper(fmt.Sprint(context)), nil
		},
	}
	runRenderCases(t, []renderCase{
		{"each map", "{{#each goodbyes}}{{@key}}. {{text}}! {{/each}}cruel {{world}}!", object("goodbyes", goodbyes, "world", "world"), "&lt;b&gt;#1&lt;/b&gt;. goodbye! 2. GOODBYE! cruel world!", nil, nil, nil},
		{"with", "{{#with person}}{{first}} {{last}}{{/with}}", object("person", object("first", "Alan", "last", "Johnson")), "Alan Johnson", nil, nil, nil},
		{"if true", "{{#if goodbye}}GOODBYE {{/if}}cruel {{world}}!", object("goodbye", true, "world", "world"), "GOODBYE cruel world!", nil, nil, nil},
		{"if string", "{{#if goodbye}}GOODBYE {{/if}}cruel {{world}}!", object("goodbye", "dummy", "world", "world"), "GOODBYE cruel world!", nil, nil, nil},
		{"if empty list", "{{#if goodbye}}GOODBYE {{/if}}cruel {{world}}!", object("goodbye", []any{}, "world", "world"), "cruel world!", nil, nil, nil},
		{"helper precedence", "{{goodbye}} {{cruel world}}", object("goodbye", "goodbye", "world", "world"), "GOODBYE cruel WORLD", helpers, nil, nil},
		{"scoped bypass", "{{this.goodbye}} {{cruel world}} {{cruel this.goodbye}}", object("goodbye", "goodbye", "world", "world"), "goodbye cruel WORLD cruel GOODBYE", helpers, nil, nil},
		{"helper hash", "{{goodbye cruel=\"CRUEL\" world=\"WORLD\" times=12}}", object(), "GOODBYE CRUEL WORLD 12 TIMES", map[string]handlebars.Helper{
			"goodbye": func(_ any, options *handlebars.Options) (any, error) {
				return fmt.Sprintf("GOODBYE %v %v %v TIMES", options.HashValue("cruel"), options.HashValue("world"), options.HashValue("times")), nil
			},
		}, nil, nil},
		{"boolean hash true", "{{goodbye cruel=\"CRUEL\" world=\"WORLD\" print=true}}", object(), "GOODBYE CRUEL WORLD", map[string]handlebars.Helper{"goodbye": printGoodbye}, nil, nil},
		{"boolean hash false", "{{goodbye cruel=\"CRUEL\" world=\"WORLD\" print=false}}", object(), "NOT PRINTING", map[string]handlebars.Helper{"goodbye": printGoodbye}, nil, nil},
		{"inverted shorthand", "{{#people}}{{name}}{{^}}{{none}}{{/people}}", object("none", "No people"), "No people", nil, nil, nil},
		{"inverted empty list", "{{#people}}{{name}}{{^}}{{none}}{{/people}}", object("none", "No people", "people", []any{}), "No people", nil, nil, nil},
	})
}

func printGoodbye(_ any, options *handlebars.Options) (any, error) {
	if print, _ := options.HashValue("print").(bool); !print {
		return "NOT PRINTING", nil
	}
	return fmt.Sprintf("GOODBYE %v %v", options.HashValue("cruel"), options.HashValue("world")), nil
}

func TestHandlebarsJSHelperHash(t *testing.T) {
	constantWorld := func(any, *handlebars.Options) (any, error) { return "world", nil }
	runRenderCases(t, []renderCase{
		{"root helper hash", "Goodbye {{cruel}} {{world}}!", object("cruel", "cruel"), "Goodbye cruel world!", map[string]handlebars.Helper{"world": constantWorld}, nil, nil},
		{"nested helper hash", "Goodbye {{#iter}}{{cruel}} {{world}}{{/iter}}!", object("iter", []any{object("cruel", "cruel")}), "Goodbye cruel world!", map[string]handlebars.Helper{"world": constantWorld}, nil, nil},
	})
}

func TestHandlebarsJSStringParameters(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) { engine.StringParams(true) }
	helpers := map[string]handlebars.Helper{
		"wycats": func(_ any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("HELP ME MY BOSS %v %v", options.Param(0), options.Param(1)), nil
		},
		"tomdale": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("STOP ME FROM READING HACKER NEWS I %v %v", context, options.Param(0)), nil
		},
	}
	runRenderCases(t, []renderCase{
		{"inline unresolved paths", "{{wycats this is.a slave.driver}}", object(), "HELP ME MY BOSS is.a slave.driver", helpers, nil, configure},
		{"nested context", "{{#with dale}}{{tomdale ../need dad.joke}}{{/with}}", object("dale", object(), "need", "need-a"), "STOP ME FROM READING HACKER NEWS I need-a dad.joke", helpers, nil, configure},
	})
}

func TestHandlebarsJSStringLiteralParameters(t *testing.T) {
	hello := func(context any, options *handlebars.Options) (any, error) {
		return fmt.Sprintf("Hello %v %v times: %v %v", context, options.Param(0), options.Param(1), options.Param(2)), nil
	}
	quotedHello := func(context any, _ *handlebars.Options) (any, error) {
		return "Hello " + fmt.Sprint(context), nil
	}
	multilineHello := func(context any, options *handlebars.Options) (any, error) {
		return fmt.Sprintf("Hello %v %v %v", context, options.Param(0), options.Param(1)), nil
	}
	goodbye := func(context any, options *handlebars.Options) (any, error) {
		world, err := options.Resolve("world")
		return fmt.Sprintf("Goodbye %v %v", context, world), err
	}
	blockGoodbye := func(_ any, options *handlebars.Options) (any, error) {
		return options.ApplyFn(object("greeting", "Goodbye", "adj", "cruel", "noun", "world"))
	}
	runRenderCases(t, []renderCase{
		{"simple literals", `Message: {{hello "world" 12 true false}}`, object(), "Message: Hello world 12 times: true false", map[string]handlebars.Helper{"hello": hello}, nil, nil},
		{"escaped quote", `Message: {{{hello "\"world\""}}}`, object(), `Message: Hello "world"`, map[string]handlebars.Helper{"hello": quotedHello}, nil, nil},
		{"apostrophe", `Message: {{{hello "Alan's world"}}}`, object(), "Message: Hello Alan's world", map[string]handlebars.Helper{"hello": quotedHello}, nil, nil},
		{"inline multiple parameters", "Message: {{goodbye cruel world}}", object("cruel", "cruel", "world", "world"), "Message: Goodbye cruel world", map[string]handlebars.Helper{"goodbye": goodbye}, nil, nil},
		{"block multiple parameters", "Message: {{#goodbye cruel world}}{{greeting}} {{adj}} {{noun}}{{/goodbye}}", object("cruel", "cruel", "world", "world"), "Message: Goodbye cruel world", map[string]handlebars.Helper{"goodbye": blockGoodbye}, nil, nil},
		{"newline in double quoted literal", "Message: {{{hello \"before\" \"multi\nline\" \"after\"}}}", object(), "Message: Hello before multi\nline after", map[string]handlebars.Helper{"hello": multilineHello}, nil, nil},
		{"newline in single quoted literal", "Message: {{{hello 'before' 'multi\nline' 'after' }}}", object(), "Message: Hello before multi\nline after", map[string]handlebars.Helper{"hello": multilineHello}, nil, nil},
	})

	const invalid = `Message: {{hello wo"rld"}}`
	if _, err := newInlineTemplate(invalid); err == nil {
		t.Errorf("CompileInline(%q) error = nil; expected a syntax error", invalid)
	}
}

func TestHandlebarsJSBlockHelperMissing(t *testing.T) {
	t.Run("helper missing fallback", func(t *testing.T) {
		engine := handlebars.New().RegisterHelperMissing(func(context any, _ *handlebars.Options) (any, error) {
			return handlebars.NewSafeString("<a>" + fmt.Sprint(context) + "</a>"), nil
		})
		template, err := engine.CompileInline("{{hello}} {{link_to world}}")
		if err != nil {
			t.Fatalf("CompileInline error = %v; expected output %q", err, "Hello <a>world</a>")
		}
		actual, err := template.Apply(object("hello", "Hello", "world", "world"))
		if err != nil || actual != "Hello <a>world</a>" {
			t.Errorf("Apply = %q, %v; expected %q, nil", actual, err, "Hello <a>world</a>")
		}
	})

	t.Run("data hash parameter", func(t *testing.T) {
		engine := handlebars.New().RegisterHelper("hello", func(_ any, options *handlebars.Options) (any, error) {
			return "Hello " + fmt.Sprint(options.HashValue("noun")), nil
		})
		template, err := engine.CompileInline("{{hello noun=@world}}")
		if err != nil {
			t.Fatalf("CompileInline error = %v; expected output %q", err, "Hello world")
		}
		context := handlebars.NewContext(object())
		context.SetData("world", "world")
		actual, err := template.ApplyContext(context)
		if err != nil || actual != "Hello world" {
			t.Errorf("ApplyContext = %q, %v; expected %q, nil", actual, err, "Hello world")
		}
	})

	t.Run("data inherited downstream", func(t *testing.T) {
		engine := handlebars.New().RegisterHelper("let", func(context any, options *handlebars.Options) (any, error) {
			for key, value := range options.Hash {
				options.SetData(key, value)
			}
			return options.ApplyFn(context)
		})
		template, err := engine.CompileInline("{{#let foo=bar.baz}}{{@foo}}{{/let}}")
		if err != nil {
			t.Fatalf("CompileInline error = %v; expected output %q", err, "hello world")
		}
		actual, err := template.Apply(object("bar", object("baz", "hello world")))
		if err != nil || actual != "hello world" {
			t.Errorf("Apply = %q, %v; expected %q, nil", actual, err, "hello world")
		}
	})
}
