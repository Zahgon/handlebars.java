// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"strings"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

type replaceEscape struct{ old, new string }

func (e replaceEscape) Escape(input string) string { return strings.ReplaceAll(input, e.old, e.new) }

func TestTextRegressions(t *testing.T) {
	sources := []string{
		"{{#if true}}true{{else}}false{{/if}}",
		"{{#if true}}true{{^}}false{{/if}}",
		"{{> partial root=this name=\"Han\"}}",
		"{{#if a}}a{{else if b}}b{{else}}c{{/if}}",
		"{{#if\r\nvalue}}true{{else}}false{{/if}}",
	}
	for _, source := range sources {
		template, err := handlebars.New(handlebars.NewMapTemplateLoader(map[string]string{"partial": ""})).CompileInline(source)
		if err != nil {
			t.Errorf("CompileInline(%q) error = %v; expected nil and text %q", source, err, source)
			continue
		}
		if actual := template.Text(); actual != source {
			t.Errorf("Text(%q) = %q; expected %q", source, actual, source)
		}
	}
}

func TestIssue447Log(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"one argument", "{{log 'message'}}", object(), "", nil, nil, nil},
		{"three arguments", "{{log 'a' 'b' 'c'}}", object(), "", nil, nil, nil},
		{"info", "{{log 'message' level='info'}}", object(), "", nil, nil, nil},
		{"debug", "{{log 'message' level='debug'}}", object(), "", nil, nil, nil},
		{"error", "{{log 'message' level='error'}}", object(), "", nil, nil, nil},
		{"trace", "{{log 'message' level='trace'}}", object(), "", nil, nil, nil},
		{"block", "{{#log}}Name: {{name}}{{/log}}", object("name", "John"), "", nil, nil, nil},
	})
}

func TestIssue465LegacyEscaping(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) { engine.WithEscapingStrategy(handlebars.HBS3) }
	runRenderCases(t, []renderCase{{"equals", "equals is {{eq}}", object("eq", "="), "equals is =", nil, nil, configure}})
}

func TestIssue478EscapingChain(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) {
		engine.WithEscapingStrategy(replaceEscape{"foo", "bar"}, replaceEscape{"bar", "$bar$"})
	}
	runRenderCases(t, []renderCase{{"chain", "{{var}}", object("var", "foo"), "$bar$", nil, nil, configure}})
}

func TestIssue483And500Helpers(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"equal": func(context any, options *handlebars.Options) (any, error) {
			return context == options.Param(0), nil
		},
		"chunk": func(_ any, options *handlebars.Options) (any, error) {
			return []any{"a", "bc", "d", options.Param(0), int32(4)}, nil
		},
	}
	runRenderCases(t, []renderCase{
		{"Issue483 true", "{{#if (equal arg 'foo')}}foo{{/if}}", object("arg", "foo"), "foo", helpers, nil, nil},
		{"Issue483 false", "{{#if (equal arg 'foo')}}foo{{/if}}", object("arg", "bar"), "", helpers, nil, nil},
		{"Issue500 list", "{{#each (chunk array 2)}}{{this}}{{/each}}", object("array", []any{"a", "b", "c", "d"}), "abcd24", helpers, nil, nil},
		{"Issue500 array", "{{#each (chunk array 2)}}{{this}}{{/each}}", object("array", []string{"a", "b", "c", "d"}), "abcd24", helpers, nil, nil},
	})
}

func TestIssue584PrettyPrint(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) { engine.PrettyPrint(true) }
	runRenderCases(t, []renderCase{
		{"else", "A\n{{#if someVariableWhichIsFalse}}\nB\n{{else}}\nC\n{{/if}}\nD", object(), "A\nC\nD", nil, nil, configure},
		{"inverse", "A\n{{#if someVariableWhichIsFalse}}\nB\n{{^}}\nC\n{{/if}}\nD", object(), "A\nC\nD", nil, nil, configure},
	})
}

func TestIssue248EscapingStrategies(t *testing.T) {
	for _, test := range []struct {
		name, input, expected string
		strategy              handlebars.EscapingStrategy
	}{
		{"default", "\"Escaping\"", "&quot;Escaping&quot;", handlebars.HTMLENTITY},
		{"csv", "\"Escaping\"", "\"\"\"Escaping\"\"\"", handlebars.CSV},
		{"xml", "<xml>", "&lt;xml&gt;", handlebars.XML},
		{"javascript", "'javascript'", "\\'javascript\\'", handlebars.JS},
	} {
		configure := func(engine *handlebars.Handlebars) { engine.WithEscapingStrategy(test.strategy) }
		runRenderCases(t, []renderCase{{test.name, "{{this}}", test.input, test.expected, nil, nil, configure}})
	}
}

func TestIssue282Subexpressions(t *testing.T) {
	helpers := map[string]handlebars.Helper{
		"inner-helper": func(context any, _ *handlebars.Options) (any, error) { return "inner-helper-" + context.(string), nil },
		"outer-helper": func(context any, options *handlebars.Options) (any, error) {
			return context.(string) + options.Param(0).(string), nil
		},
		"a":      func(context any, _ *handlebars.Options) (any, error) { return "a" + context.(string), nil },
		"e":      func(context any, _ *handlebars.Options) (any, error) { return "e" + context.(string), nil },
		"i":      func(context any, _ *handlebars.Options) (any, error) { return "i" + context.(string), nil },
		"o":      func(context any, _ *handlebars.Options) (any, error) { return "o" + context.(string), nil },
		"u":      func(any, *handlebars.Options) (any, error) { return "u", nil },
		"vowels": func(context any, _ *handlebars.Options) (any, error) { return context, nil },
	}
	runRenderCases(t, []renderCase{
		{"nested helper", "{{outer-helper (inner-helper 'abc') 'def'}}", object(), "inner-helper-abcdef", helpers, nil, nil},
		{"nested vowels", "{{vowels (a (e (i (o (u)))))}}", object(), "aeiou", helpers, nil, nil},
	})
}

func TestIssue506BlockParameters(t *testing.T) {
	lower := map[string]handlebars.Helper{"lower": func(context any, _ *handlebars.Options) (any, error) {
		value := strings.ToLower(context.(string))
		return value, nil
	}}
	runRenderCases(t, []renderCase{
		{"without as", "{{#each things}}not using \"as\": value is {{this}}, key is {{@index}}, hello is {{lower \"HELLO\"}}{{/each}}", object("things", []any{"foo"}), "not using \"as\": value is foo, key is 0, hello is hello", lower, nil, nil},
		{"with as", "{{#each things as |value key|}}using \"as\": value is {{value}}, key is {{key}}, hello is {{lower \"HELLO\"}}{{/each}}", object("things", []any{"foo"}), "using \"as\": value is foo, key is 0, hello is hello", lower, nil, nil},
		{"block helper with as", "{{#each things as |value key|}}using \"as\" and \"#\": value is {{value}}, key is {{key}}, hello is {{#lower \"HELLO\"}}{{this}}{{/lower}}{{/each}}", object("things", []any{"foo"}), "using \"as\" and \"#\": value is foo, key is 0, hello is hello", lower, nil, nil},
	})
}

func TestIssue1167ChildProperty(t *testing.T) {
	data := object("typeName", "ParentType", "prop", object("arrayItems", object("typeName", "ChildType")))
	runRenderCases(t, []renderCase{
		{"map", "{{typeName}}{{#with prop.arrayItems}} {{typeName}}{{/with}}", data, "ParentType ChildType", nil, nil, nil},
		{"record equivalent", "{{typeName}}{{#with prop.arrayItems}} {{typeName}}{{/with}}", data, "ParentType ChildType", nil, nil, nil},
	})
}

type constantResolver struct{}

func (constantResolver) Resolve(any, string) (any, bool, error) { return int32(1), true, nil }
func (constantResolver) ResolveValue(any) (any, bool, error)    { return int32(1), true, nil }
func (constantResolver) PropertySet(any) ([]handlebars.Property, error) {
	return nil, nil
}

func TestIssue490And654Resolvers(t *testing.T) {
	model := object("foo", "bar")
	combined := handlebars.NewContextBuilder(model).Resolver(handlebars.MapValueResolverInstance, constantResolver{}).Build()
	for path, expected := range map[string]any{"foo": "bar", "missing": int32(1)} {
		actual, err := combined.Resolve(path)
		if err != nil || actual != expected {
			t.Errorf("combined.Resolve(%q) = %#v, %v; expected %#v, nil", path, actual, err, expected)
		}
	}
	for _, resolvers := range [][]handlebars.ValueResolver{{handlebars.JavaBeanValueResolverInstance}, {handlebars.MapValueResolverInstance}} {
		context := handlebars.NewContextBuilder(model).Resolver(resolvers...).Build()
		actual, err := context.Resolve("foo")
		if err != nil || actual != "bar" {
			t.Errorf("Resolve(foo) with %T = %#v, %v; expected %q, nil", resolvers[0], actual, err, "bar")
		}
	}
}
