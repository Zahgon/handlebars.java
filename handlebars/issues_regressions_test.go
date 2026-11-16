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

func TestRemainingIssueRegressions(t *testing.T) {
	linked := jvm.NewLinkedHashMap()
	linked.Put("123", []any{int32(1), int32(2), int32(3)})
	linked.Put("456", []any{int32(4), int32(5), int32(6)})
	one := jvm.NewLinkedHashMap()
	one.Put("africa", &jvm.List{Elems: []any{"egypt", "kenya"}})
	subtract := map[string]handlebars.Helper{
		"subtract": func(context any, options *handlebars.Options) (any, error) {
			return context.(int32) - options.Param(0).(int32), nil
		},
	}
	issue1189 := map[string]handlebars.Helper{
		"contextAndHash": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("context=%v,scope=%v", context, options.HashValue("scope")), nil
		},
		"extraParam": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("context=%v,param0=%v", context, options.Param(0)), nil
		},
	}
	replace := map[string]handlebars.Helper{
		"replaceHelperTest": func(context any, _ *handlebars.Options) (any, error) {
			return strings.ReplaceAll(fmt.Sprint(context), "foo", ""), nil
		},
	}
	negative := map[string]handlebars.Helper{
		"negParam": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v", context, options.Param(0)), nil
		},
		"negHash": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v", context, options.HashValue("h")), nil
		},
	}
	jsEquivalent := map[string]handlebars.Helper{
		"getIndex": func(context any, _ *handlebars.Options) (any, error) { return context, nil },
		"nullHelper": func(context any, _ *handlebars.Options) (any, error) {
			if context == nil {
				return "NULL", nil
			}
			return "NOT_NULL", nil
		},
	}
	modifier := map[string]handlebars.Helper{
		"modifiers": func(_ any, options *handlebars.Options) (any, error) { return options.Param(0), nil },
	}
	length := map[string]handlebars.Helper{
		"length": func(context any, _ *handlebars.Options) (any, error) {
			switch value := context.(type) {
			case []any:
				return int32(len(value)), nil
			case []string:
				return int32(len(value)), nil
			default:
				return int32(0), nil
			}
		},
	}
	raw := map[string]handlebars.Helper{
		"raw-helper": func(context any, options *handlebars.Options) (any, error) { return options.ApplyFn(context) },
	}
	namespaced := map[string]handlebars.Helper{
		"nav.render": func(context any, _ *handlebars.Options) (any, error) { return context, nil },
	}
	ok := map[string]handlebars.Helper{
		"foo": func(any, *handlebars.Options) (any, error) { return "ok", nil },
	}
	overrides := map[string]handlebars.Helper{
		"each":   func(any, *handlebars.Options) (any, error) { return "each", nil },
		"unless": func(any, *handlebars.Options) (any, error) { return "unless", nil },
	}
	messages := map[string]handlebars.Helper{
		"msg": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v%v%v", context, options.Param(0), options.Param(1), options.Param(2)), nil
		},
		"msg1": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprintf("%v%v%v%v", context, options.Param(0), options.Param(1), options.Param(2)), nil
		},
	}
	ifEq := map[string]handlebars.Helper{
		"if_eq": func(context any, options *handlebars.Options) (any, error) {
			if fmt.Sprint(context) == fmt.Sprint(options.Param(0)) {
				return options.ApplyFn(context)
			}
			return options.ApplyInverse(context)
		},
	}
	greeting := map[string]handlebars.Helper{
		"greeting": func(any, *handlebars.Options) (any, error) { return "Hello", nil },
	}
	equal := map[string]handlebars.Helper{
		"equal": func(context any, options *handlebars.Options) (any, error) {
			return fmt.Sprint(context) == fmt.Sprint(options.Param(0)), nil
		},
	}
	mathHelpers := map[string]handlebars.Helper{
		"remainder": func(context any, options *handlebars.Options) (any, error) {
			return context.(int32) % options.Param(0).(int32), nil
		},
		"is": func(context any, options *handlebars.Options) (any, error) {
			if fmt.Sprint(context) == fmt.Sprint(options.Param(0)) {
				return options.ApplyFn(context)
			}
			return options.ApplyInverse(context)
		},
	}
	families := []any{
		[]any{object("age", int32(10), "name", "jimmy"), object("age", int32(15), "name", "rose")},
		[]any{object("age", int32(35), "name", "John"), object("age", int32(32), "name", "Jessy")},
	}
	models := object(
		"model1", object("listOfValues1", []any{"m1-1"}),
		"model2", object("users", []any{"User 1"}),
		"model3", true,
	)
	modelsWithProperties := object(
		"model1", object("listOfValues1", []any{"m1-1"}),
		"model2", object("users", []any{object("name", "User 1")}),
		"model3", true,
	)
	runRenderCases(t, []renderCase{
		{"Issue1189 inverse helper context and hash", "{{^contextAndHash foo scope=\"deps\"}}{{/contextAndHash}}", object("foo", "bar"), "context=bar,scope=deps", issue1189, nil, nil},
		{"Issue1189 inverse helper extra parameter", "{{^extraParam foo bar}}{{/extraParam}}", object("foo", "a", "bar", "b"), "context=a,param0=b", issue1189, nil, nil},
		{"Issue200 dynamic helper equivalent", "hello world: {{replaceHelperTest \"foobar\"}}", object(), "hello world: bar", replace, nil, nil},
		{"Issue149 negative positional", "{{negParam foo -1}}", object("foo", "x"), "x-1", negative, nil, nil},
		{"Issue149 negative hash", "{{negHash foo h=-1}}", object("foo", "x"), "x-1", negative, nil, nil},
		{"Issue150 dotted array index", "{{foo.0.}}", object("foo", []any{"bar"}), "bar", nil, nil, nil},
		{"Issue150 bracketed array index", "{{foo.[0]}}", object("foo", []any{"bar"}), "bar", nil, nil, nil},
		{"Issue177 bracketed model list", "{{model1.listOfValues1.[0]}}", models, "m1-1", nil, nil, nil},
		{"Issue177 two bracketed models", "{{model1.listOfValues1.[0]}}\n{{model2.users.[0]}}", models, "m1-1\nUser 1", nil, nil, nil},
		{"Issue177 bracketed model property", "{{model2.users.[0].name}}", modelsWithProperties, "User 1", nil, nil, nil},
		{"Issue177 bracketed models under if", "{{model1.listOfValues1.[0]}}\n{{#if model3}}\n{{model2.users.[0].name}}\n{{/if}}", modelsWithProperties, "m1-1\n\nUser 1\n", nil, nil, nil},
		{"Issue243 zero JavaScript-helper equivalent", "{{#each item}}{{getIndex @index}} {{/each}}", object("item", []any{int32(10), int32(20), int32(30)}), "0 1 2 ", jsEquivalent, nil, nil},
		{"Issue243 null JavaScript-helper equivalent", "{{nullHelper item}}", object("item", nil), "NULL", jsEquivalent, nil, nil},
		{"Issue243 nonnull JavaScript-helper equivalent", "{{nullHelper item}}", object("item", object()), "NOT_NULL", jsEquivalent, nil, nil},
		{"Issue270 character literal", "{{modifiers this 'clock'}}", object(), "clock", modifier, nil, nil},
		{"Issue311 property containing period", "{{ this.[foo.bar] }}", object("foo.bar", "baz"), "baz", nil, nil, nil},
		{"Issue314 with relative property", "{{#with obj}}{{context.one}}{{/with}}", object("obj", object("context", object("one", int32(1), "two", int32(2)))), "1", nil, nil, nil},
		{"Issue314 with cannot see root child", "{{#with obj}}{{obj.context.one}}{{/with}}", object("obj", object("context", object("one", int32(1), "two", int32(2)))), "", nil, nil, nil},
		{"Issue470 parent index", "{{#each families}} {{#each this}}{{../@index}} :{{name}} {{/each}} {{/each}}", object("families", families), " 0 :jimmy 0 :rose   1 :John 1 :Jessy  ", nil, nil, nil},
		{"Issue470 parent data index", "{{#each families}} {{#each this}}{{@../index}} :{{name}} {{/each}} {{/each}}", object("families", families), " 0 :jimmy 0 :rose   1 :John 1 :Jessy  ", nil, nil, nil},
		{"Issue476 JavaScript length list equivalent", "{{length this}}", []any{"1", "3", "4"}, "3", length, nil, nil},
		{"Issue476 JavaScript length array equivalent", "{{length this}}", []string{"1", "3", "4"}, "3", length, nil, nil},
		{"Hbs519 raw helper in conditional", "{{#unless test}}{{{{raw-helper}}}}{{testing}}{{{{/raw-helper}}}}{{/unless}}", object(), "{{testing}}", raw, nil, nil},
		{"Hbs525 namespaced helper", "{{nav.render 'main'}}", object(), "main", namespaced, nil, nil},
		{"Hbs530 empty block parameters", "{{#each users as |user userId|}}\n  Id: {{userId}} Name: {{user.name}} <BR>\n{{/each}}", object("users", []any{}), "", nil, nil, nil},
		{"Issue1078 digit-leading identifier", "<p>{{24HrServicePh}}</p>", object("24HrServicePh", "1-800-000-0000"), "<p>1-800-000-0000</p>", nil, nil, nil},
		{"Issue1078 digit-leading block identifier", "{{#if 24Hr}}yes{{/if}}", object("24Hr", true), "yes", nil, nil, nil},
		{"Issue1078 numeric literals remain parameters", "{{foo 24}} {{foo 24.5}}", object(), "ok ok", ok, nil, nil},
		{"Issue1163 empty list last", "{{@last}}", []any{}, "", nil, nil, nil},
		{"Issue1163 empty list first", "{{@first}}", []any{}, "", nil, nil, nil},
		{"Issue334 map first marker", "{{#each this}}{{@first}}{{/each}}", object("one", int32(1), "two", int32(2)), "first", nil, nil, nil},
		{"Issue338 override each", "{{each}}", object(), "each", overrides, nil, nil},
		{"Issue338 override unless", "{{unless}}", object(), "unless", overrides, nil, nil},
		{"Issue347 bracketed property with space", "{{ this.[foo bar] }}", object("foo bar", "foo.bar"), "foo.bar", nil, nil, nil},
		{"Issue347 unqualified bracketed property with space", "{{ [foo bar] }}", object("foo bar", "foo.bar"), "foo.bar", nil, nil, nil},
		{"Issue410 options parameters", "{{msg 'p' 1 2 3}}", object(), "p123", messages, nil, nil},
		{"Issue410 explicit and options parameters", "{{msg1 'p' 1 2 3}}", object(), "p123", messages, nil, nil},
		{"Issue374 primitive numeric equality", "{{#if_eq num1 num2}}True{{else}}False{{/if_eq}}", object("num1", int32(5), "num2", int32(5)), "True", ifEq, nil, nil},
		{"Issue374 boxed numeric equality", "{{#if_eq num1 num2}}True{{else}}False{{/if_eq}}", object("num1", int32(5), "num2", int32(5)), "True", ifEq, nil, nil},
		{"Issue374 mixed numeric equality", "{{#if_eq num1 num2}}True{{else}}False{{/if_eq}}", object("num1", int32(5), "num2", int32(5)), "True", ifEq, nil, nil},
		{"Issue374 property numeric equality", "{{#if_eq val1.value val2.value}}True{{else}}False{{/if_eq}}", object("val1", object("value", int32(5)), "val2", object("value", int32(5))), "True", ifEq, nil, nil},
		{"Issue374 boxed property numeric equality", "{{#if_eq val1.value val2.value}}True{{else}}False{{/if_eq}}", object("val1", object("value", int32(5)), "val2", object("value", int32(5))), "True", ifEq, nil, nil},
		{"Issue363 missing bracket paths", "{{model1.listOfValues1.[0]}}{{#if model3}}{{model2.users.[0].name}}{{/if}}", object(), "", nil, nil, nil},
		{"Issue370 single quote escaping", "{{this}}", "'", "&#x27;", nil, nil, nil},
		{"Issue385 root data in each", "{{#each array}}{{@root.foo}}{{/each}}", object("foo", "bar", "array", []any{"foo"}), "bar", nil, nil, nil},
		{"Issue418 lookup helper in each", "{{#each series}} {{lookup ../types this}}{{/each}}", object("series", []any{"Test A", "Test B", "Test C"}, "types", object("Test A", "bar", "Test B", "bar", "Test C", "line")), " bar bar line", nil, nil, nil},
		{"Issue430 equals escaping", "{{this}}", "foo=", "foo&#x3D;", nil, nil, nil},
		{"Issue432 with block parameter", "{{#with this as |foo|}}\n    Foo: {{foo.baz}} \n{{/with}}", object("baz", "foo"), "\n    Foo: foo \n", nil, nil, nil},
		{"Issue820 argumentless helper in each", "{{greeting}}\n{{#each users as |user|}}{{greeting}} {{user}}\n{{/each}}", object("users", []any{"Jack", "John"}), "Hello\nHello Jack\nHello John\n", greeting, nil, nil},
		{"Issue448b subexpression on block parameter", "{{#each this as |c|}} {{c}} - {{#is 0 (remainder c 2)}}even{{else}} odd{{/is}} \n{{/each}}", []any{int32(4), int32(5), int32(6), int32(7), int32(8)}, " 4 - even \n 5 -  odd \n 6 - even \n 7 -  odd \n 8 - even \n", mathHelpers, nil, nil},
		{"Issue483b JavaScript equal equivalent true", "{{#if (equal arg 'foo')}}foo{{/if}}", object("arg", "foo"), "foo", equal, nil, nil},
		{"Issue483b JavaScript equal equivalent false", "{{#if (equal arg 'foo')}}foo{{/if}}", object("arg", "bar"), "", equal, nil, nil},
		{"Issue458 null property", "{{baa}}", object("baa", nil), "", nil, nil, nil},
		{"Issue458 qualified null property", "{{this.baa}}", object("baa", nil), "", nil, nil, nil},
		{"Issue458 null property under with", "{{#with this}}{{baa}}{{/with}}", object("baa", nil), "", nil, nil, nil},
		{"Issue467 map key block parameter", "{{#each this as |eitem key| }}{{this}} || {{key}} | {{eitem}}{{/each}}", one, "[egypt, kenya] || africa | [egypt, kenya]", nil, nil, nil},
		{"Issue502 nested each over map", "{{#each data}}{{ @key }} - {{#each . }}Val:{{.}}{{/each}}{{/each}}", object("data", linked), "123 - Val:1Val:2Val:3456 - Val:4Val:5Val:6", nil, nil, nil},
		{"Issue521 else trim true", "\n{{~#if bold~}}\n<b>\n{{else}}\n<i>\n{{~/if~}}\n", object("bold", true), "<b>\n", nil, nil, nil},
		{"Issue521 else trim false", "\n{{~#if bold~}}\n<b>\n{{else}}\n<i>\n{{~/if~}}\n", object("bold", false), "\n<i>", nil, nil, nil},
		{"Issue521 bilateral else trim true", "\n{{~#if bold~}}\n<b>\n{{~else~}}\n<i>\n{{~/if~}}\n", object("bold", true), "<b>", nil, nil, nil},
		{"Issue521 bilateral else trim false", "\n{{~#if bold~}}\n<b>\n{{~else~}}\n<i>\n{{~/if~}}\n", object("bold", false), "<i>", nil, nil, nil},
		{"Issue521 inverse trim true", "\n{{~#if bold~}}\n<b>\n{{~^~}}\n<i>\n{{~/if~}}\n", object("bold", true), "<b>", nil, nil, nil},
		{"Issue521 inverse trim false", "\n{{~#if bold~}}\n<b>\n{{~^~}}\n<i>\n{{~/if~}}\n", object("bold", false), "<i>", nil, nil, nil},
		{"Issue574 empty list inverse", "{{#each list}}not empty{{else}}empty{{/each}}", object("list", []any{}), "empty", nil, nil, nil},
		{"Issue574 null list inverse", "{{#each list}}not empty{{else}}empty{{/each}}", object("list", nil), "empty", nil, nil, nil},
		{"Issue606 numeric subexpression truthy", "{{#if (subtract value 1)}}OK{{/if}}", object("value", int32(2)), "OK", subtract, nil, nil},
		{"Issue606 numeric subexpression falsy", "{{#if (subtract value 1)}}OK{{/if}}", object("value", int32(1)), "", subtract, nil, nil},
		{"Issue611 escaped bracket path", "<div class=\"entry\">\n  <h1>{{ fields.[Special [case\\]] }}</h1>\n</div>", object("fields", object("Special [case\\]", "yo")), "<div class=\"entry\">\n  <h1>yo</h1>\n</div>", nil, nil, nil},
	})

	const invalidIssue150 = "{{foo[0]}}"
	if _, err := handlebars.New().CompileInline(invalidIssue150); err == nil {
		t.Errorf("CompileInline(%q) error = nil; expected the invalid-bracket syntax error asserted by Issue150", invalidIssue150)
	}
}
