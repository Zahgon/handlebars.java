// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

// TypeSafeTemplateTest and TypeSafeTemplateDefaultMethodTest are omitted because Go has no Java dynamic proxies/default methods.
// ReflectiveHelperTest, PropertyAccessTest, ValueResolverTest, and CustomObjectTest are omitted where they inspect Java reflection/test beans.
// MethodValueResolverTest's checked-vs-runtime reflection exception distinction has no Go reflection counterpart.
// HandlebarsExceptionTest's Mockito constructor identity/cause checks and Java serialization have no Go counterpart; plain error text is checked elsewhere.

import (
	"math/big"
	"reflect"
	"sort"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestPathResolution(t *testing.T) {
	context := object("title", "root", "foo", object("title", "foo", "bar", object("title", "bar")))
	runRenderCases(t, []renderCase{
		{"current", "{{#foo}}{{#bar}}{{title}}{{/bar}}{{/foo}}", context, "bar", nil, nil, nil},
		{"parent", "{{#foo}}{{#bar}}{{../title}}{{/bar}}{{/foo}}", context, "foo", nil, nil, nil},
		{"grandparent", "{{#foo}}{{#bar}}{{../../title}}{{/bar}}{{/foo}}", context, "root", nil, nil, nil},
		{"above root", "{{#foo}}{{#bar}}{{../../../title}}{{/bar}}{{/foo}}", context, "", nil, nil, nil},
	})
}

func TestTagTypesAndCollection(t *testing.T) {
	for tag, expected := range map[handlebars.TagType]bool{
		handlebars.TagVar:           true,
		handlebars.TagAmpVar:        true,
		handlebars.TagTripleVar:     true,
		handlebars.TagSubExpression: true,
		handlebars.TagSection:       false,
		handlebars.TagStartSection:  false,
	} {
		if actual := tag.Inline(); actual != expected {
			t.Errorf("TagType(%d).Inline() = %v; expected %v", tag, actual, expected)
		}
	}
	cases := []struct {
		name     string
		input    string
		tags     []handlebars.TagType
		expected []string
	}{
		{"variables", "{{#hello}}{{a}}{{&b}}{{z}}{{/hello}}{{k}}", []handlebars.TagType{handlebars.TagVar}, []string{"a", "z", "k"}},
		{"ampersand", "{{#hello}}{{a}}{{&b}}{{z}}{{/hello}}{{k}}", []handlebars.TagType{handlebars.TagAmpVar}, []string{"b"}},
		{"triple", "{{{tvar}}}{{#hello}}{{a}}{{/hello}}", []handlebars.TagType{handlebars.TagTripleVar}, []string{"tvar"}},
		{"section", "{{#hello}}{{a}}{{/hello}}", []handlebars.TagType{handlebars.TagSection}, []string{"hello"}},
		{"subexpression", "{{vowels (tag)}}", []handlebars.TagType{handlebars.TagSubExpression}, []string{"tag"}},
	}
	for _, test := range cases {
		engine := handlebars.New()
		if test.name == "subexpression" {
			noop := func(any, *handlebars.Options) (any, error) { return "", nil }
			engine.RegisterHelper("tag", noop).RegisterHelper("vowels", noop)
		}
		template, err := engine.CompileInline(test.input)
		if err != nil {
			t.Fatalf("CompileInline(%q) error: %v; expected collect %v", test.input, err, test.expected)
		}
		actual := template.Collect(test.tags...)
		sort.Strings(actual)
		expected := append([]string(nil), test.expected...)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Collect(%q) = %v; expected %v", test.input, actual, expected)
		}
	}
}

func TestSafeString(t *testing.T) {
	a := handlebars.NewSafeString("hello")
	b := handlebars.NewSafeString("hello")
	c := handlebars.NewSafeString("hello!")
	if a != b {
		t.Errorf("SafeString equality = false for %q and %q; expected true", a, b)
	}
	if a == c {
		t.Errorf("SafeString equality = true for %q and %q; expected false", a, c)
	}
	if actual := a.String(); actual != "hello" {
		t.Errorf("SafeString.String() = %q; expected %q", actual, "hello")
	}
	if actual := a.JavaClass(); actual != "com.github.jknack.handlebars.Handlebars$SafeString" {
		t.Errorf("SafeString.JavaClass() = %q; expected %q", actual, "com.github.jknack.handlebars.Handlebars$SafeString")
	}
}

func TestFalsyValues(t *testing.T) {
	decimal, err := jvm.NewBigDecimal("0")
	if err != nil {
		t.Fatalf("NewBigDecimal(0) error: %v; expected nil", err)
	}
	values := []any{nil, false, "", int8(0), int16(0), int32(0), int64(0), float32(0), float64(0), big.NewInt(0), decimal, &jvm.List{Elems: []any{}}, []any{}}
	for _, value := range values {
		if actual := handlebars.IsEmpty(value); !actual {
			t.Errorf("IsEmpty(%T(%v)) = %v; expected true", value, value, actual)
		}
	}
	truthy := []any{true, "x", int32(1), []any{"x"}, &jvm.List{Elems: []any{"x"}}}
	for _, value := range truthy {
		if actual := handlebars.IsEmpty(value); actual {
			t.Errorf("IsEmpty(%T(%v)) = %v; expected false", value, value, actual)
		}
	}
	if actual := handlebars.IsEmpty(int32(0), true); actual {
		t.Errorf("IsEmpty(0, true) = %v; expected false", actual)
	}
}

func TestContextData(t *testing.T) {
	template, err := newInlineTemplate("{{@hello}}")
	if err != nil {
		t.Fatalf("CompileInline error: %v; expected output %q", err, "hello")
	}
	context := handlebars.NewContext(object())
	context.SetData("hello", "hello")
	actual, err := template.ApplyContext(context)
	if err != nil || actual != "hello" {
		t.Errorf("ApplyContext = %q, %v; expected %q, nil", actual, err, "hello")
	}
}
