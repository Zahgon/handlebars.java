// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

// NashornTest, PrecompileHelperTest, and JavaScriptHelperTest are omitted because Go has no Nashorn/JS engine.
// Mockito-only helper invocation-count assertions are adapted to rendered-output assertions in this file.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

type formatterValue struct{ text string }

func TestHelperMissing(t *testing.T) {
	runRenderCases(t, []renderCase{
		{
			name:     "fallback helper",
			template: "{{hello}} {{link_to world}}",
			data:     object("hello", "Hello", "world", "world"),
			expected: "Hello <a>world</a>",
			configure: func(engine *handlebars.Handlebars) {
				engine.RegisterHelperMissing(func(context any, _ *handlebars.Options) (any, error) {
					return handlebars.NewSafeString("<a>" + fmt.Sprint(context) + "</a>"), nil
				})
			},
		},
	})
}

func TestGoHelperAdaptations(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"helperA": func(any, *handlebars.Options) (any, error) { return "helperA", nil },
		"helperAWithContext": func(context any, _ *handlebars.Options) (any, error) {
			return "helperAWithContext" + fmt.Sprint(context), nil
		},
		"helperWithParams": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("helperWithParams:%v:%v:%v", context, options.Param(0), options.Param(1)), nil
		},
	}
	runRenderCases(t, []renderCase{
		{"no arguments", "{{helperA}}", object(), "helperA", helpers, nil, nil},
		{"context argument", "{{helperAWithContext value}}", object("value", "!"), "helperAWithContext!", helpers, nil, nil},
		{"typed literals", "{{helperWithParams \"string\" true 4}}", object(), "helperWithParams:string:true:4", helpers, nil, nil},
	})
}

func TestHelperErrorIsWrapped(t *testing.T) {
	engine := handlebars.New().RegisterHelper("fail", func(any, *handlebars.Options) (any, error) {
		return nil, errors.New("helper failed")
	})
	template, err := engine.CompileInline("before {{fail}} after")
	if err != nil {
		t.Fatalf("CompileInline error: %v; expected render error containing %q", err, "helper failed")
	}
	actual, err := template.Apply(nil)
	if err == nil || actual != "before " || !strings.Contains(err.Error(), "helper failed") {
		t.Errorf("Apply = %q, %v; expected %q and error containing %q", actual, err, "before ", "helper failed")
	}
}

func TestFormatters(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) {
		engine.WithFormatter(handlebars.FormatterFunc(func(value any) (any, bool) {
			if marker, ok := value.(formatterValue); ok {
				return marker.text, true
			}
			return nil, false
		}))
		engine.WithFormatter(handlebars.FormatterFunc(func(value any) (any, bool) {
			if integer, ok := value.(int32); ok {
				return fmt.Sprintf("%x", integer), true
			}
			return nil, false
		}))
	}
	runRenderCases(t, []renderCase{
		{"first formatter twice", "value is {{this}}/{{this}}", formatterValue{"formatted"}, "value is formatted/formatted", nil, nil, configure},
		{"formatter miss", "string is {{this}}", "testvalue", "string is testvalue", nil, nil, configure},
		{"second formatter", "Hex-Value is {{this}}", int32(16), "Hex-Value is 10", nil, nil, configure},
	})
}
