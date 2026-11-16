// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestEmbeddedHelper(t *testing.T) {
	const partial = "<tr>\n  <td>{{firstName}}</td>\n  <td>{{lastName}}</td>\n</tr>"
	runRenderCases(t, []renderCase{
		{"default id", "<html>\n...\n{{embedded \"user\"}}\n...\n</html>", object(), "<html>\n...\n<script id=\"user-hbs\" type=\"text/x-handlebars\">\n" + partial + "\n</script>\n...\n</html>", nil, map[string]string{"user": partial}, nil},
		{"custom id", "<html>\n...\n{{embedded \"user\" \"user-tmpl\"}}\n...\n</html>", object(), "<html>\n...\n<script id=\"user-tmpl\" type=\"text/x-handlebars\">\n" + partial + "\n</script>\n...\n</html>", nil, map[string]string{"user": partial}, nil},
	})
}

func TestPartialRegression(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"Issue1182 parameter propagation", "{{> testParamPage param=\"hello\"}}", object(), "hello", nil, map[string]string{"testParamPage": "{{> parameters}}", "parameters": "{{param}}"}, nil},
		{"Issue1185 iteration data", "{{#each items}}{{> myPartial}}{{/each}}", object("items", []any{"Item 1"}), "@index: 0, @first: first, @last: last", nil, map[string]string{"myPartial": "@index: {{@index}}, @first: {{@first}}, @last: {{@last}}"}, nil},
		{"Issue459 double quoted name", "{{> \"loop\"}}", object("foo", "bar"), "bar", nil, map[string]string{"loop": "{{foo}}"}, nil},
		{"Issue459 single quoted name", "{{> 'loop'}}", object("foo", "bar"), "bar", nil, map[string]string{"loop": "{{foo}}"}, nil},
		{"Issue459 explicit context", "{{> \"loop\" this}}", object("foo", "bar"), "bar", nil, map[string]string{"loop": "{{foo}}"}, nil},
		{"Issue459 hash", "{{> \"loop\" h=1}}", object("foo", "bar"), "bar1", nil, map[string]string{"loop": "{{foo}}{{h}}"}, nil},
		{"Issue475 repeated hash", "{{> p name=\"firstName\" suffix=suffix}}\n{{> p name=\"lastName\" suffix=suffix}}\n{{> p name=\"streetName\" suffix=suffix}}", object("suffix", "foo"), "firstName foo\nlastName foo\nstreetName foo", nil, map[string]string{"p": "{{name}} {{suffix}}"}, nil},
		{"Issue475 nested repeated hash", "{{> p suffix=\"foo\"}}", object(), "firstName foo\nlastName foo\nstreetName foo", nil, map[string]string{"p": "{{> q name=\"firstName\" suffix=suffix}}\n{{> q name=\"lastName\" suffix=suffix}}\n{{> q name=\"streetName\" suffix=suffix}}", "q": "{{name}} {{suffix}}"}, nil},
		{"Issue479 hash override", "{{> dude _greeting=\"Hello\"}}", object("name", "Elliot", "_greeting", "Good Morning"), "Hello Elliot!", nil, map[string]string{"dude": "{{_greeting}} {{name}}!"}, nil},
		{"Issue479 inherited hash", "{{> dude}}", object("name", "Elliot", "_greeting", "Good Morning"), "Good Morning Elliot!", nil, map[string]string{"dude": "{{_greeting}} {{name}}!"}, nil},
		{"Issue250 explicit partial context keeps parent", "{{> share page}}", object("p", "parent", "page", object("name", "share")), "parent", nil, map[string]string{"share": "{{p}}"}, nil},
		{"Issue250 implicit partial context keeps parent", "{{> share}}", object("p", "parent", "page", object("name", "share")), "parent", nil, map[string]string{"share": "{{p}}"}, nil},
		{"Issue552 partial block parameter context", "{{> button size='large'}}", object(), "<button class=\"button-large>\n    Button with size large\n</button>", nil, map[string]string{"button": "<button class=\"button-{{size}}>\n    Button with size {{size}}\n</button>"}, nil},
		{"Issue643 false partial value", "template: {{empty}} {{> partial}}", object("empty", false), "template: false partial: false", nil, map[string]string{"partial": "partial: {{empty}}"}, nil},
		{"Issue660 multiline partial block", "text text text\n{{#>some-partial param1=\"val1\"\n                 param2=\"val2\"}}\n    default content\n{{/some-partial}}", object(), "text text text\n\n    default content\n", nil, nil, nil},
	})
}

func TestIssue588PartialBlocks(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) {
		engine.PreEvaluatePartialBlocks(false).InfiniteLoops(true)
	}
	partial := "<div {{#if x}}x={{x}}{{/if}}>{{> @partial-block}}</div>"
	runRenderCases(t, []renderCase{
		{"inline not defined without call", "{{#> dude}}{{#*inline \"myPartial\"}}success{{/inline}}{{/dude}}", object(), "", nil, map[string]string{"dude": "{{#> myPartial }}{{/myPartial}}"}, configure},
		{"nested block hashes", "{{#> dude x=23}}{{#> dude x=12}}{{/dude}}{{/dude}}", object(), "<div x=23><div x=12></div></div>", nil, map[string]string{"dude": partial}, configure},
		{"false nested block hash", "{{#> dude x=23}}{{#> dude x=false}}{{/dude}}{{/dude}}", object(), "<div x=23><div ></div></div>", nil, map[string]string{"dude": partial}, configure},
	})
}

func TestIssue634PartialBlockErrors(t *testing.T) {
	runRenderCases(t, []renderCase{{"provided", "{{#> my-partial}}634{{/my-partial}}", object(), "Hello 634", nil, map[string]string{"my-partial": "Hello {{> @partial-block}}"}, nil}})
}
