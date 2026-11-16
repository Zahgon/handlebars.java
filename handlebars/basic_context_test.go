// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestBasicContext(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"most basic", "{{foo}}", object("foo", "foo"), "foo", nil, nil, nil},
		{"basic context", "Goodbye\n{{cruel}}\n{{world}}!", object("cruel", "cruel", "world", "world"), "Goodbye\ncruel\nworld!", nil, nil, nil},
		{"accented colon path", "Goodbye\n{{x:cruél}}\n{{world}}!", object("x:cruél", "cruel", "world", "world"), "Goodbye\ncruel\nworld!", nil, nil, nil},
		{"comment", "{{! Goodbye}}Goodbye\n{{cruel}}\n{{world}}!", object("cruel", "cruel", "world", "world"), "Goodbye\ncruel\nworld!", nil, nil, nil},
		{"true section", "{{#goodbye}}GOODBYE {{/goodbye}}cruel {{world}}!", object("goodbye", true, "world", "world"), "GOODBYE cruel world!", nil, nil, nil},
		{"false section", "{{#goodbye}}GOODBYE {{/goodbye}}cruel {{world}}!", object("goodbye", false, "world", "world"), "cruel world!", nil, nil, nil},
		{"zero properties", "num1: {{num1}}, num2: {{num2}}", object("num1", int32(42), "num2", int32(0)), "num1: 42, num2: 0", nil, nil, nil},
		{"zero scalar", "num: {{.}}", int32(0), "num: 0", nil, nil, nil},
		{"zero nested", "num: {{num1/num2}}", object("num1", object("num2", int32(0))), "num: 0", nil, nil, nil},
		{"LF text", "Alan's\nTest", object(), "Alan's\nTest", nil, nil, nil},
		{"CR text", "Alan's\rTest", object(), "Alan's\rTest", nil, nil, nil},
		{"triple unescaped", "{{{awesome}}}", object("awesome", "&\"\\<>"), "&\"\\<>", nil, nil, nil},
		{"ampersand unescaped", "{{&awesome}}", object("awesome", "&\"\\<>"), "&\"\\<>", nil, nil, nil},
		{"escaped expression", "{{awesome}}", object("awesome", "&\"'`\\<>"), "&amp;&quot;&#x27;&#x60;\\&lt;&gt;", nil, nil, nil},
		{"safe string", "{{awesome}}", object("awesome", handlebars.NewSafeString("&\"\\<>")), "&\"\\<>", nil, nil, nil},
		{"hyphen path", "{{foo-bar}}", object("foo-bar", "baz"), "baz", nil, nil, nil},
		{"nested dotted hyphen", "{{foo.foo-bar}}", object("foo", object("foo-bar", "baz")), "baz", nil, nil, nil},
		{"nested slash hyphen", "{{foo/foo-bar}}", object("foo", object("foo-bar", "baz")), "baz", nil, nil, nil},
		{"nested path", "Goodbye {{alan/expression}} world!", object("alan", object("expression", "beautiful")), "Goodbye beautiful world!", nil, nil, nil},
		{"empty nested path", "Goodbye {{alan/expression}} world!", object("alan", object("expression", "")), "Goodbye  world!", nil, nil, nil},
		{"literal path", "Goodbye {{[@alan]/expression}} world!", object("@alan", object("expression", "beautiful")), "Goodbye beautiful world!", nil, nil, nil},
		{"missing complex path", "{{person/name}}", object("person", object()), "", nil, nil, nil},
		{"this list", "{{#goodbyes}}{{this}}{{/goodbyes}}", object("goodbyes", []any{"goodbye", "Goodbye", "GOODBYE"}), "goodbyeGoodbyeGOODBYE", nil, nil, nil},
		{"this nested", "{{#hellos}}{{this/text}}{{/hellos}}", object("hellos", []any{object("text", "hello"), object("text", "Hello"), object("text", "HELLO")}), "helloHelloHELLO", nil, nil, nil},
	})
}
