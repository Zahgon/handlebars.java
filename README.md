Handlebars.java for Go
======================
## Logic-less and semantic Mustache templates, the Handlebars.java way, in Go

```go
package main

import (
	"fmt"

	"github.com/jknack/handlebars.java/handlebars"
)

func main() {
	hbs := handlebars.New()
	template, err := hbs.CompileInline("Hello {{this}}!")
	if err != nil {
		panic(err)
	}
	out, err := template.Apply("Handlebars.java")
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
```

Output:
```
Hello Handlebars.java!
```

This module is the Go port of [Handlebars.java](https://github.com/jknack/handlebars.java) 4.5.8, itself a port of
[handlebars](https://handlebarsjs.com/). It keeps the Java engine's behaviour: the same template language, the same
helpers and the same output, including the way the Java runtime prints numbers, collections and dates and the error
messages of the Java parser.

[Mustache](https://mustache.github.io/mustache.5.html) templates are compatible with Handlebars, so you can take a
Mustache template, import it into Handlebars, and start taking advantage of the extra Handlebars features.

# Requirements

- Go 1.23 or newer. The module uses the standard library only.

# Getting Started

```
go get github.com/jknack/handlebars.java
```

The packages are:

- `github.com/jknack/handlebars.java/handlebars`: the engine (compiler, templates, context, helpers, escaping
  strategies, template loaders and caches).
- `github.com/jknack/handlebars.java/cmd/handlebars`: a command that renders a template directory with JSON data.
- `internal/jvm`, `internal/commons`, `internal/jtext` and `internal/jtime`: the Java runtime semantics the engine's
  output depends on (Java value text, `java.util.Formatter`, `java.text` number and message formats, `java.time`
  formatting with CLDR data, commons-lang3 and commons-text helpers).

### Loading templates

Templates are loaded with a `TemplateLoader`. The default loader reads files from the working directory with the
`.hbs` suffix:

```go
hbs := handlebars.New(handlebars.NewFileTemplateLoaderWithSuffix("templates", ".html"))
template, err := hbs.Compile("mytemplate") // templates/mytemplate.html
```

Other loaders: `NewMapTemplateLoader` (templates held in memory), `NewFSTemplateLoader` (any `fs.FS`, for example an
embedded file system) and `NewCompositeTemplateLoader` (the first loader that finds the template wins). A partial
`{{> name}}` is resolved through the same loader, relative to its prefix.

### Data

A template renders any Go value. Maps, slices, strings, numbers, booleans and nil behave like their Java
counterparts (`java.util.Map`, `java.util.List`, `String`, `Integer`, `Long`, `BigInteger`, `Double`, `BigDecimal`,
`Boolean`, `null`); `internal/jvm` models the Java collection classes, including the iteration order of
`java.util.HashMap` and `java.util.LinkedHashMap`. Members are resolved the way the Java value resolvers do: map
keys, JavaBean properties (`{{list.empty}}`) and public no-argument methods (`{{name.length}}`).

## Helpers

### Built-in helpers

- `with`, `each`, `if`, `unless`: the block helpers of Handlebars.js. `each` exposes `@index`, `@first`, `@last`,
  `@odd`, `@even`, `@key` and `@index_1`, accepts `base=` and block parameters (`as |value key|`).
- `block` and `partial`: template inheritance. A layout declares `{{#block "name"}}default{{/block}}`; a page
  overrides it with `{{#partial "name"}}content{{/partial}}` and then renders the layout with `{{> layout}}`.
- `embedded`: embeds a template's source in a `<script type="text/x-handlebars">` element.
- `lookup`, `log` and the `inline` decorator.
- `i18n` and `i18nJs`: messages from properties bundles (`ResourceBundle` rules: locale fallback, `MessageFormat`
  arguments).
- `precompile`: compiling templates to JavaScript needs a JavaScript engine, which Go does not have; the helper and
  `Handlebars.Precompile` return an error saying so.

### String helpers

`RegisterStringHelpers` registers the Handlebars.java `StringHelpers`: `capitalizeFirst`, `center`, `cut`,
`defaultIfEmpty`, `join`, `ljust`, `rjust`, `substring`, `lower`, `upper`, `slugify`, `stringFormat`, `stripTags`,
`capitalize`, `abbreviate`, `wordWrap`, `replace`, `yesno`, `dateFormat`, `numberFormat` and `now`.

### Conditional helpers

`RegisterConditionalHelpers` registers `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `and`, `or` and `not`, with the
`yes=` and `no=` hash and block forms (`{{#eq a b}}...{{else}}...{{/eq}}`).

### Registering helpers

A helper is a function of the context and the call's options:

```go
hbs.RegisterHelper("blog", func(context any, options *handlebars.Options) (any, error) {
	return fmt.Sprintf("%v by %v", context, options.HashValue("author", "anonymous")), nil
})
```

`options.Param(i, def)`, `options.HashValue(name, def)`, `options.Data(name)`, `options.ApplyFn(ctx)` and
`options.ApplyInverse(ctx)` give access to parameters, the hash, `@data` variables and the block bodies. Return a
`handlebars.SafeString` to skip escaping.

## Error reporting

### Syntax errors

```
file:line:column: found: '<EOF>', expected: '{{/'
{{#if a}}x
          ^
```

Compile errors carry the file name, the position, the parser's reason and the evidence line with a caret, exactly as
Handlebars.java prints them.

### Helper and runtime errors

```
file:line:column: java.lang.IllegalArgumentException: Minimum abbreviation width is 4
    file:line:column
```

A failure while rendering names the tag that failed and the Java exception the original reports.

## Advanced Usage

- Escaping strategies: `HTMLENTITY` (the default), `HBS3`, `JS`, `XML`, `CSV`, `NOOP` and chains of them
  (`WithEscapingStrategy`).
- `PrettyPrint(true)` removes standalone tag lines as the Mustache spec requires; without it the template's
  whitespace is kept.
- `ParentScopeResolution(false)` stops the lookup of missing names in the parent contexts.
- `StringParams(true)` passes unresolved parameters as their text.
- `InfiniteLoops(true)` allows recursive partials; `DeletePartialAfterMerge(true)` removes a `partial` override once a
  `block` used it; `PreEvaluatePartialBlocks(false)` evaluates partial blocks lazily.
- `StartDelimiter` and `EndDelimiter` change the default `{{` and `}}`.
- Template caches: `NullTemplateCache`, `NewConcurrentMapTemplateCache()` (with reloading of changed sources).
- `TypedTemplate[T]` gives a typed wrapper over a compiled template.

## The handlebars command

```
go build -o /tmp/handlebars ./cmd/handlebars
/tmp/handlebars [OPTIONS] TEMPLATE_DIR NAME [DATA]
/tmp/handlebars --inline [OPTIONS] TEMPLATE_DIR [DATA]
```

The command compiles `TEMPLATE_DIR/NAME.hbs` (or the template read from standard input with `--inline`), renders it
with the JSON file `DATA` and prints the result. JSON objects become ordered maps (`--hash-maps` uses
`java.util.HashMap` order), whole numbers Integer, Long or BigInteger by range, other numbers Double
(`--big-decimals`: BigDecimal). It registers the string and conditional helpers, a `date` helper parsing ISO-8601
text into `java.time` values, three helper-source methods (`rep`, `pair`, `describe`) and the decorators `set` and
`trace`. Options: `--suffix`, `--escape`, `--pretty-print`, `--start-delimiter`, `--end-delimiter`,
`--no-parent-scope`, `--no-pre-evaluate`, `--string-params`, `--infinite-loops`, `--delete-partial-after-merge`,
`--data=NAME=JSON`, `--locale`, `--timezone`, `--lambda`, `--text`, `--collect` and `--collect-params`. Exit statuses:
0 on success, 2 when compiling fails, 3 when rendering fails, 64 for a usage error and 65 for unreadable input.

## Differences between Handlebars.java and Handlebars.js

Handlebars.java scope resolution follows the Mustache Spec. For example:

Given:

```json
{
  "value": "parent",
  "child": {
  }
}
```
and

```html
Hello {{#child}}{{value}}{{/child}}
```

will be:

```html
Hello parent
```

Now, the same model and template with Handlebars.js is:

```html
Hello 
```
That is because Handlebars.js doesn't look in the context stack for missing attributes in the current scope (this is
consistent with the Mustache Spec).

You can turn off the context stack lookup by qualifying the attribute with ```this.```:

```html
Hello {{#child}}{{this.value}}{{/child}}
```

Other differences kept from Handlebars.java: `@first` and `@last` are the strings `first` and `last` (empty
otherwise), standalone lines are kept unless pretty printing is on, and `=` and the backtick are HTML-escaped.

## Differences between Handlebars.java and Mustache.js
 * A partial that cannot be loaded is an error.

## Status
### Mustache 1.0 Compliant
 * Passes the tests of the [Mustache Spec](https://github.com/mustache/spec) (`go test ./handlebars -run Spec`; the
   spec files and their licence are in `handlebars/testdata/mustache`).

### Handlebars.js Compliant
  * The Handlebars.js tests that Handlebars.java ports run as Go tests in `handlebars/`.

## Development

```
gofmt -l .
go vet ./...
go test ./...
```

`src/etc/formatter.sh` is a pre-commit hook that runs `gofmt` on the staged Go files.

## Help and Support
 [Bugs, Issues and Features](https://github.com/jknack/handlebars.java/issues)

## Related Projects
 * [Handlebars.js](http://handlebarsjs.com/)
 * [Mustache](http://mustache.github.io/)

## Author
 [Edgar Espina](https://twitter.com/edgarespina)

## License
[Apache License 2](http://www.apache.org/licenses/LICENSE-2.0.html)
