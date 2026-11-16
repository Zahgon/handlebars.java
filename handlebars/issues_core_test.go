// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestIssue259BigDecimal(t *testing.T) {
	for _, test := range []struct {
		input    string
		expected bool
	}{{"0.01", false}, {"0.0000", true}} {
		value, err := jvm.NewBigDecimal(test.input)
		if err != nil {
			t.Errorf("NewBigDecimal(%q) error = %v; expected nil", test.input, err)
			continue
		}
		if actual := handlebars.IsEmpty(value); actual != test.expected {
			t.Errorf("IsEmpty(BigDecimal(%q)) = %v; expected %v", test.input, actual, test.expected)
		}
	}
}

func TestIssue266Configuration(t *testing.T) {
	engine := handlebars.New()
	if actual := engine.PrettyPrint(false); actual != engine {
		t.Errorf("PrettyPrint(false) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.StringParams(false); actual != engine {
		t.Errorf("StringParams(false) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.InfiniteLoops(false); actual != engine {
		t.Errorf("InfiniteLoops(false) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.StartDelimiter("<<"); actual != engine {
		t.Errorf("StartDelimiter(<<) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.EndDelimiter(">>"); actual != engine {
		t.Errorf("EndDelimiter(>>) = %p; expected engine %p", actual, engine)
	}
	start, end := engine.Delimiters()
	if start != "<<" || end != ">>" {
		t.Errorf("Delimiters() = %q, %q; expected %q, %q", start, end, "<<", ">>")
	}
	if actual := engine.WithLoader(handlebars.NewMapTemplateLoader(map[string]string{})); actual != engine {
		t.Errorf("WithLoader(...) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.WithCache(handlebars.NewConcurrentMapTemplateCache()); actual != engine {
		t.Errorf("WithCache(...) = %p; expected engine %p", actual, engine)
	}
	if actual := engine.WithEscapingStrategy(handlebars.HTMLENTITY); actual != engine {
		t.Errorf("WithEscapingStrategy(...) = %p; expected engine %p", actual, engine)
	}
}

func TestJavaScriptHelperAdaptations(t *testing.T) {
	dowork307 := func(context any, options *handlebars.Options) (any, error) {
		for _, item := range context.([]any) {
			if item.(map[string]any)["name"] == "pato" {
				return options.ApplyFn(item)
			}
		}
		return "", nil
	}
	dowork308 := func(context any, options *handlebars.Options) (any, error) {
		result := ""
		for _, item := range context.([]any) {
			row := item.(map[string]any)
			if row["name"] == "pato" {
				row["newval"] = "friend"
			} else {
				row["newval"] = "colleague"
			}
			body, err := options.ApplyFn(row)
			if err != nil {
				return nil, err
			}
			result += body
		}
		return result, nil
	}
	link := func(context any, options *handlebars.Options) (any, error) {
		return handlebars.NewSafeString(fmt.Sprintf("<a href=\"%s\">%s</a>", handlebars.EscapeExpression(options.Param(0)), handlebars.EscapeExpression(context))), nil
	}
	rows := []any{object("name", "edgar", "age", int32(34)), object("name", "pato", "age", int32(34))}
	runRenderCases(t, []renderCase{
		{"Issue307 JS helper", "{{#dowork root/results}}name:{{name}}, age:{{age}}{{/dowork}}", object("root", object("results", rows)), "name:pato, age:34", map[string]handlebars.Helper{"dowork": dowork307}, nil, nil},
		{"Issue308 JS helper mutation", "{{#dowork root/results}}name:{{name}}, age:{{age}}, newval:{{newval}} {{/dowork}}", object("root", object("results", rows)), "name:edgar, age:34, newval:colleague name:pato, age:34, newval:friend ", map[string]handlebars.Helper{"dowork": dowork308}, nil, nil},
		{"Issue327 link", "{{link 'handlebars.java' 'https://github.com/jknack/handlebars.java'}}", object(), "<a href=\"https://github.com/jknack/handlebars.java\">handlebars.java</a>", map[string]handlebars.Helper{"link": link}, nil, nil},
		{"Issue327 call link", "{{call-link 'handlebars.java' 'https://github.com/jknack/handlebars.java'}}", object(), "<a href=\"https://github.com/jknack/handlebars.java\">handlebars.java</a>", map[string]handlebars.Helper{"call-link": link}, nil, nil},
	})
}

type epochMillis struct{ value int64 }

func TestIssue332Formatter(t *testing.T) {
	const now int64 = 1_700_000_000_123
	configure := func(engine *handlebars.Handlebars) {
		engine.WithFormatter(handlebars.FormatterFunc(func(value any) (any, bool) {
			date, ok := value.(epochMillis)
			if !ok {
				return nil, false
			}
			return date.value, true
		}))
	}
	runRenderCases(t, []renderCase{
		{"this", "time is {{this}}", epochMillis{now}, "time is 1700000000123", nil, nil, configure},
		{"named", "time is {{now}}", object("now", epochMillis{now}), "time is 1700000000123", nil, nil, configure},
		{"qualified", "time is {{this.now}}", object("now", epochMillis{now}), "time is 1700000000123", nil, nil, configure},
	})
}

func TestIssue95RecursiveLoader(t *testing.T) {
	engine := handlebars.New(handlebars.NewFileTemplateLoader("testdata/issue95"))
	engine.InfiniteLoops(true)
	if template, err := engine.Compile("hbs/start"); err != nil || template == nil {
		t.Errorf("Compile(%q) = %#v, %v; expected non-nil template, nil", "hbs/start", template, err)
	}
}
