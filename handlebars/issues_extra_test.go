// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestIssue288NestedIndex(t *testing.T) {
	runRenderCases(t, []renderCase{{
		"index restored", "{{#each array1}}index_before - {{@index}}\n{{#each array2}}{{/each}}index_after - {{@index}}\n{{/each}}",
		object("array1", []any{object("array2", []any{int32(0), int32(1), int32(2)})}), "index_before - 0\nindex_after - 0\n", nil, nil, nil,
	}})
}

func TestIssue397SubexpressionText(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"concat": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v+%v", context, options.Param(0)), nil
		},
		"helper": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v", context, options.HashValue("param1")), nil
		},
	}
	runRenderCases(t, []renderCase{{"render", "{{helper context_var param1=(concat \"a\" \"b\")}}", object("context_var", "!"), "!a+b", helpers, nil, nil}})
	for _, source := range []string{"{{helper context_var param1=(concat \"a\" \"b\")}}", "{{helper (concat \"a\" \"b\")}}"} {
		engine := handlebars.New()
		for name, helper := range helpers {
			engine.RegisterHelper(name, helper)
		}
		template, err := engine.CompileInline(source)
		if err != nil {
			t.Errorf("CompileInline(%q) error = %v; expected text %q", source, err, source)
		} else if actual := template.Text(); actual != source {
			t.Errorf("Text(%q) = %q; expected %q", source, actual, source)
		}
	}
}

func TestIssue407JavaScriptHelperAdaptation(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"uppercase": func(context any, _ *handlebars.Options) (any, error) {
			return strings.ToUpper(fmt.Sprint(context)), nil
		},
		"compare": func(context any, options *handlebars.Options) (any, error) {
			if context == options.Param(0) {
				return options.ApplyFn(context)
			}
			return options.ApplyInverse(context)
		},
	}
	runRenderCases(t, []renderCase{
		{"simple", "{{#compare true true}}\n    {{uppercase \"aaa\"}}\n{{else}}\n    {{uppercase \"bbb\"}}\n{{/compare}}", object(), "\n    AAA\n", helpers, nil, nil},
		{"nested", "{{#compare true true}}\n    {{#compare true true}}\n        kkk\n    {{/compare}}\n{{/compare}}", object(), "\n    \n        kkk\n    \n", helpers, nil, nil},
	})
}

func TestIssue323HashSubexpression(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"someTemplate": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v%v", context, options.Param(0), options.HashValue("hashArg")), nil
		},
		"myHelper": func(context any, _ *handlebars.Options) (any, error) { return context, nil },
	}
	runRenderCases(t, []renderCase{{"hash", "{{someTemplate param1 param2 hashArg=(myHelper param3)}}", object("param1", "a", "param2", "b", "param3", "c"), "abc", helpers, nil, nil}})
}

func TestIssue463bParentScope(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) { engine.ParentScopeResolution(true) }
	runRenderCases(t, []renderCase{{"parent fallback", "Hello {{#child}}{{value}}{{bestQB}}{{/child}}", object("value", "Brett", "child", object("bestQB", "Favre")), "Hello BrettFavre", nil, nil, configure}})
}

func TestIssue489LookupObject(t *testing.T) {
	book := object("title", "Effective Java (2nd Edition)", "author", "Joshua Bloch", "isbn", "0321356683")
	data := object("booksByIsbn", object("0321356683", book), "myLibrary", []any{"0321356683"})
	runRenderCases(t, []renderCase{{"lookup", "{{#each myLibrary}}{{#with (lookup @root.booksByIsbn .) as |book|}}{{book.title}} by {{book.author}} ({{book.isbn}})\n{{/with}}{{/each}}", data, "Effective Java (2nd Edition) by Joshua Bloch (0321356683)\n", nil, nil, nil}})
}

func TestIssues350356And402(t *testing.T) {
	pretty := func(engine *handlebars.Handlebars) { engine.PrettyPrint(true) }
	runRenderCases(t, []renderCase{
		{
			"Issue350 partial hash parameters",
			"<ul>\n{{#each dudes}}\n  {{> dude title=../title class=\"list-item\"}}\n{{/each}}\n</ul>",
			object("title", "profile", "dudes", []any{object("name", "Yehuda", "url", "http://yehuda"), object("name", "Alan", "url", "http://alan")}),
			"<ul>\n  <li class=\"list-item\">\n    profile: <a href=\"http://yehuda\">Yehuda</a>\n  </li>\n  <li class=\"list-item\">\n    profile: <a href=\"http://alan\">Alan</a>\n  </li>\n</ul>",
			nil,
			map[string]string{"dude": "<li class=\"{{class}}\">\n  {{title}}: <a href=\"{{url}}\">{{name}}</a>\n</li>\n"},
			pretty,
		},
		{
			"Issue356 ancestor traversal",
			"{{#data}}{{#each items}}\n   1111:  {{this.trayType}} \n   2222:  {{../this.trayType}}\n   3333:  {{../../trayType}}\n    {{#if imageOverridden ~}}\n        image-overridden\n    {{else ~}}\n        {{#if ../../trayType ~}}\n            size-{{../../trayType}}\n        {{~/if}}\n    {{~/if}}    \n{{/each}}{{/data}}",
			object("trayType", "video", "data", object("items", []any{object("id", "id-1", "name", "name-1")}, "config", object())),
			"\n   1111:   \n   2222:  \n   3333:  video\n    size-video    \n",
			nil, nil, nil,
		},
		{
			"Issue402 partial hash isolation",
			"{{> user name=\"Bob\" age=\"31\"}}\n{{> user age=\"29\"}}",
			object(),
			"<div><b>Bob</b> 31</div>\n<div><b></b> 29</div>",
			nil,
			map[string]string{"user": "<div><b>{{name}}</b> {{age}}</div>"},
			nil,
		},
	})
}

func TestIssues376And657(t *testing.T) {
	numberHelpers := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	andHelper := map[string]handlebars.Helper{
		"and": func(context any, options *handlebars.Options) (any, error) {
			if context == true && options.Param(0) == true {
				return options.ApplyFn(context)
			}
			return options.ApplyInverse(context)
		},
	}
	runRenderCases(t, []renderCase{
		{"Issue376 implicit this", "{{#each foo}}{{#if this.price}}{{numberFormat price 'currency'}}\n{{/if}}{{/each}}", object("foo", []any{object("price", int32(5)), object("price", int32(7))}), "$5.00\n$7.00\n", nil, nil, numberHelpers},
		{"Issue376 explicit this", "{{#each foo}}{{#if this.price}}{{numberFormat this.price 'currency'}}\n{{/if}}{{/each}}", object("foo", []any{object("price", int32(5)), object("price", int32(7))}), "$5.00\n$7.00\n", nil, nil, numberHelpers},
		{"Issue657 JavaScript helper adaptation", "{{#and great magnificent}}Hello 657{{/and}}", object("great", true, "magnificent", true), "Hello 657", andHelper, nil, nil},
	})
}

func TestIssue375DynamicPartials(t *testing.T) {
	dudes := []any{object("name", "Yehuda", "url", "http://yehuda"), object("name", "Alan", "url", "http://alan")}
	dynamic := map[string]handlebars.Helper{"dude": func(any, *handlebars.Options) (any, error) { return "dyndude", nil }}
	person := "{{name}} ({{url}}) "
	want := "Dudes: Yehuda (http://yehuda) Alan (http://alan) "
	runRenderCases(t, []renderCase{
		{"helper", "Dudes: {{#dudes}}{{> (dude)}}{{/dudes}}", object("dudes", dudes), want, dynamic, map[string]string{"dyndude": person}, nil},
		{"helper hash", "Dudes: {{#dudes}}{{> (dude) p1=1}}{{/dudes}}", object("dudes", dudes), "Dudes: Yehuda (http://yehuda) p1 Alan (http://alan) p1 ", dynamic, map[string]string{"dyndude": "{{name}} ({{url}}) p{{p1}} "}, nil},
		{"lookup dot", "Dudes: {{#dudes}}{{> (lookup . 'dude')}}{{/dudes}}", object("dudes", dudes, "dude", "dyndude"), want, nil, map[string]string{"dyndude": person}, nil},
		{"lookup value", "Dudes: {{#dudes}}{{> (lookup dude)}}{{/dudes}}", object("dudes", dudes, "dude", "dyndude"), want, nil, map[string]string{"dyndude": person}, nil},
		{"lookup parent", "Dudes: {{#dudes}}{{> (lookup ../. 'dude')}}{{/dudes}}", object("dudes", dudes, "dude", "dyndude"), want, nil, map[string]string{"dyndude": person}, nil},
		{"lookup this name", "Dudes: {{#dudes}}{{> (lookup this 'name')}}{{/dudes}}", object("dudes", dudes), want, nil, map[string]string{"Yehuda": person, "Alan": person}, nil},
		{"lookup name", "Dudes: {{#dudes}}{{> (lookup name)}}{{/dudes}}", object("dudes", dudes), want, nil, map[string]string{"Yehuda": person, "Alan": person}, nil},
	})
	for _, source := range []string{
		"Dudes: {{#dudes}}{{> (lookup missing 'name')}}{{/dudes}}",
		"Dudes: {{#dudes}}{{> (lookup this 'missing')}}{{/dudes}}",
	} {
		template, err := handlebars.New(handlebars.NewMapTemplateLoader(map[string]string{"Yehuda": person, "Alan": person})).CompileInline(source)
		if err == nil {
			_, err = template.Apply(object("dudes", dudes))
		}
		if err == nil {
			t.Errorf("Apply(%q) error = nil; expected a dynamic-partial lookup error", source)
		}
	}
}

func TestIssues477And477bPartialHash(t *testing.T) {
	for _, parentScope := range []bool{true, false} {
		parentScope := parentScope
		configure := func(engine *handlebars.Handlebars) { engine.ParentScopeResolution(parentScope) }
		runRenderCases(t, []renderCase{{
			fmt.Sprintf("parent scope %t", parentScope),
			"{{> dude _greeting=\"Hello\"}}",
			object("name", "Elliot"),
			"Hello Elliot!",
			nil,
			map[string]string{"dude": "{{_greeting}} {{name}}!"},
			configure,
		}})
	}
}

func TestIssue448HelperAdaptation(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"is": func(context any, options *handlebars.Options) (any, error) {
			current, err := options.Resolve(".")
			if err != nil {
				return nil, err
			}
			if fmt.Sprint(context) == fmt.Sprint(options.Param(0)) {
				return options.ApplyFn(current)
			}
			return options.ApplyInverse(current)
		},
		"remainder": func(context any, options *handlebars.Options) (any, error) {
			return context.(int32) % options.Param(0).(int32), nil
		},
	}
	runRenderCases(t, []renderCase{
		{"context", "{{#each letter}}{{#if this}}{{@index}}={{this}}{{/if}}\n{{#is 0 0}}{{@index}}={{this}}{{/is}}\n{{/each}}", object("letter", []any{"a", "b", "c", "d"}), "0=a\n0=a\n1=b\n1=b\n2=c\n2=c\n3=d\n3=d\n", helpers, nil, nil},
		{"block parameter", "{{#each this as |c|}} {{c}} - {{#is 0 (remainder c 2)}}even{{else}} odd{{/is}} \n{{/each}}", []any{int32(4), int32(5), int32(6), int32(7), int32(8)}, " 4 - even \n 5 -  odd \n 6 - even \n 7 -  odd \n 8 - even \n", helpers, nil, nil},
		{"this", "{{#each this}} {{this}} - {{#is 0 (remainder this 2)}}even{{else}} odd{{/is}} \n{{/each}}", []any{int32(4), int32(5), int32(6), int32(7), int32(8)}, " 4 - even \n 5 -  odd \n 6 - even \n 7 -  odd \n 8 - even \n", helpers, nil, nil},
	})
}

func TestHbs531DynamicPartialLookup(t *testing.T) {
	source := "{{#each this}} {{> (lookup profession 'name') this.person}}<br /> {{/each}}"
	data := []any{
		object("firstName", "John", "lastName", "Foe", "profession", "developer"),
		object("firstName", "James", "lastName", "Doe", "profession", "analyst"),
	}
	runRenderCases(t, []renderCase{{"render", source, data, " Developer - John Foe<br />  Analyst - James Doe<br /> ", nil, map[string]string{
		"analyst": "Analyst - {{firstName}} {{lastName}}", "developer": "Developer - {{firstName}} {{lastName}}",
	}, nil}})
	template, err := handlebars.New().CompileInline(source)
	if err != nil {
		t.Errorf("CompileInline(%q) error = %v; expected text %q", source, err, source)
	} else if actual := template.Text(); actual != source {
		t.Errorf("Text() = %q; expected %q", actual, source)
	}
}
