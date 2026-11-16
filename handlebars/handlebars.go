// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/jknack/handlebars.java/internal/jvm"
)

const (
	// DelimStart is the default opening delimiter.
	DelimStart = "{{"
	// DelimEnd is the default closing delimiter.
	DelimEnd = "}}"
	// HelperMissing is the missing-helper registry key.
	HelperMissing = "helperMissing"
)

// Handlebars configures template compilation and rendering.
type Handlebars struct {
	loader                   TemplateLoader
	cache                    TemplateCache
	stringParams             bool
	prettyPrint              bool
	infiniteLoops            bool
	deletePartialAfterMerge  bool
	startDelimiter           string
	endDelimiter             string
	parentScopeResolution    bool
	preEvaluatePartialBlocks bool
	escaping                 EscapingStrategy
	formatters               []Formatter
	parserFactory            ParserFactory
	helpers                  map[string]Helper
	decorators               map[string]Decorator
	mu                       sync.RWMutex
	charset                  string
}

// New creates a Handlebars engine with the built-in helpers.
func New(loaders ...TemplateLoader) *Handlebars {
	h := &Handlebars{
		cache:                    NullTemplateCache,
		startDelimiter:           DelimStart,
		endDelimiter:             DelimEnd,
		parentScopeResolution:    true,
		preEvaluatePartialBlocks: true,
		escaping:                 HTMLENTITY,
		helpers:                  make(map[string]Helper),
		decorators:               make(map[string]Decorator),
		charset:                  "UTF-8",
	}
	if len(loaders) == 1 {
		h.loader = loaders[0]
	} else if len(loaders) > 1 {
		h.loader = NewCompositeTemplateLoader(loaders...)
	}
	h.registerBuiltins()
	return h
}

// Compile loads and compiles a template by logical name.
func (h *Handlebars) Compile(location string) (Template, error) {
	if h.loader == nil {
		return nil, fmt.Errorf("The template loader is required.")
	}
	source, err := h.loader.SourceAt(location)
	if err != nil {
		return nil, err
	}
	return h.CompileSource(source)
}

// CompileWithDelimiters compiles a named template with explicit delimiters.
func (h *Handlebars) CompileWithDelimiters(location, start, end string) (Template, error) {
	if h.loader == nil {
		return nil, fmt.Errorf("The template loader is required.")
	}
	source, err := h.loader.SourceAt(location)
	if err != nil {
		return nil, err
	}
	return h.compileSource(source, start, end)
}

// CompileInline compiles inline source text.
func (h *Handlebars) CompileInline(input string) (Template, error) {
	name := inlineTemplateName(input)
	return h.compileSource(NewStringTemplateSource(name, input), h.startDelimiter, h.endDelimiter)
}

// CompileInlineWithDelimiters compiles inline text with explicit delimiters.
func (h *Handlebars) CompileInlineWithDelimiters(input, start, end string) (Template, error) {
	name := inlineTemplateName(input)
	return h.compileSource(NewStringTemplateSource(name, input), start, end)
}

func inlineTemplateName(input string) string {
	hash := jvm.StringHashCode(input)
	if hash < 0 {
		hash = -hash
	}
	return "inline@" + fmt.Sprintf("%x", uint32(hash))
}

// CompileSource compiles a TemplateSource.
func (h *Handlebars) CompileSource(source TemplateSource) (Template, error) {
	return h.compileSource(source, h.startDelimiter, h.endDelimiter)
}

func (h *Handlebars) compileSource(source TemplateSource, start, end string) (Template, error) {
	if strings.EqualFold(h.charset, "US-ASCII") {
		content, err := source.Content()
		if err != nil {
			return nil, err
		}
		bytes := []byte(content)
		for i, value := range bytes {
			if value > 0x7f {
				bytes[i] = '?'
			}
		}
		source = NewStringTemplateSource(source.Filename(), string(bytes))
	}
	parser := NewParser(h, start, end)
	if h.parserFactory != nil {
		parser = h.parserFactory.Create(h, start, end)
	}
	return h.cache.Get(source, parser)
}

// Precompile reports the absent JavaScript engine.
func (h *Handlebars) Precompile(path string) (string, error) { return "", missingJSError() }

// PrecompileInline reports the absent JavaScript engine.
func (h *Handlebars) PrecompileInline(template string) (string, error) { return "", missingJSError() }
func missingJSError() error {
	return jvm.Throw("java.lang.IllegalStateException", "No JavaScript engine available")
}

// RegisterHelper adds or replaces a helper.
func (h *Handlebars) RegisterHelper(name string, helper Helper) *Handlebars {
	h.mu.Lock()
	h.helpers[name] = helper
	h.mu.Unlock()
	return h
}

// RegisterHelperMissing sets the fallback helper.
func (h *Handlebars) RegisterHelperMissing(helper Helper) *Handlebars {
	return h.RegisterHelper(HelperMissing, helper)
}

// Helper returns a registered helper.
func (h *Handlebars) Helper(name string) Helper {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.helpers[name]
}

// Helpers returns a copy of the helper registry.
func (h *Handlebars) Helpers() map[string]Helper {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := make(map[string]Helper, len(h.helpers))
	for k, v := range h.helpers {
		m[k] = v
	}
	return m
}

// RegisterDecorator adds or replaces a decorator.
func (h *Handlebars) RegisterDecorator(name string, decorator Decorator) *Handlebars {
	h.mu.Lock()
	h.decorators[name] = decorator
	h.mu.Unlock()
	return h
}

// Decorator returns a decorator.
func (h *Handlebars) Decorator(name string) Decorator {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.decorators[name]
}

// PrettyPrint enables standalone-line and partial indentation handling.
func (h *Handlebars) PrettyPrint(value bool) *Handlebars { h.prettyPrint = value; return h }

// PrettyPrintEnabled reports whether standalone-line processing is enabled.
func (h *Handlebars) PrettyPrintEnabled() bool { return h.prettyPrint }

// StringParams configures unresolved helper parameters to pass their names.
func (h *Handlebars) StringParams(value bool) *Handlebars { h.stringParams = value; return h }

// StringParamsEnabled reports whether unresolved parameters retain their names.
func (h *Handlebars) StringParamsEnabled() bool { return h.stringParams }

// InfiniteLoops configures recursive partial calls.
func (h *Handlebars) InfiniteLoops(value bool) *Handlebars { h.infiniteLoops = value; return h }

// InfiniteLoopsEnabled reports whether recursive partial calls are permitted.
func (h *Handlebars) InfiniteLoopsEnabled() bool { return h.infiniteLoops }

// DeletePartialAfterMerge configures block partial cleanup.
func (h *Handlebars) DeletePartialAfterMerge(value bool) *Handlebars {
	h.deletePartialAfterMerge = value
	return h
}

// StartDelimiter sets the opening delimiter.
func (h *Handlebars) StartDelimiter(value string) *Handlebars { h.startDelimiter = value; return h }

// EndDelimiter sets the closing delimiter.
func (h *Handlebars) EndDelimiter(value string) *Handlebars { h.endDelimiter = value; return h }

// Delimiters returns the configured opening and closing delimiters.
func (h *Handlebars) Delimiters() (start, end string) { return h.startDelimiter, h.endDelimiter }

// ParentScopeResolution configures fallback to parent scopes.
func (h *Handlebars) ParentScopeResolution(value bool) *Handlebars {
	h.parentScopeResolution = value
	return h
}

// ParentScopeResolutionEnabled reports whether path lookup falls back to parents.
func (h *Handlebars) ParentScopeResolutionEnabled() bool { return h.parentScopeResolution }

// PreEvaluatePartialBlocks configures partial block pre-evaluation.
func (h *Handlebars) PreEvaluatePartialBlocks(value bool) *Handlebars {
	h.preEvaluatePartialBlocks = value
	return h
}

// WithLoader replaces the loader.
func (h *Handlebars) WithLoader(loaders ...TemplateLoader) *Handlebars {
	if len(loaders) == 1 {
		h.loader = loaders[0]
	} else {
		h.loader = NewCompositeTemplateLoader(loaders...)
	}
	return h
}

// WithCache replaces the template cache.
func (h *Handlebars) WithCache(cache TemplateCache) *Handlebars { h.cache = cache; return h }

// Loader returns the configured template loader.
func (h *Handlebars) Loader() TemplateLoader { return h.loader }

// Cache returns the configured template cache.
func (h *Handlebars) Cache() TemplateCache { return h.cache }

// WithParserFactory replaces the parser factory.
func (h *Handlebars) WithParserFactory(factory ParserFactory) *Handlebars {
	h.parserFactory = factory
	return h
}

// ParserFactory returns the configured parser factory.
func (h *Handlebars) ParserFactory() ParserFactory { return h.parserFactory }

// WithCharset records the template character set; Go loaders decode UTF-8.
func (h *Handlebars) WithCharset(charset string) *Handlebars { h.charset = charset; return h }

// Charset returns the configured character-set name.
func (h *Handlebars) Charset() string { return h.charset }

// WithEscapingStrategy replaces the escaping strategy or composes a chain.
func (h *Handlebars) WithEscapingStrategy(strategies ...EscapingStrategy) *Handlebars {
	if len(strategies) == 1 {
		h.escaping = strategies[0]
	} else {
		h.escaping = ChainEscaping(strategies...)
	}
	return h
}

// EscapingStrategy returns the current strategy.
func (h *Handlebars) EscapingStrategy() EscapingStrategy { return h.escaping }

// WithFormatter appends a formatter.
func (h *Handlebars) WithFormatter(formatter Formatter) *Handlebars {
	h.formatters = append(h.formatters, formatter)
	return h
}

// IsEmpty implements Handlebars.Utils.isEmpty.
func IsEmpty(value any, includeZero ...bool) bool {
	if value == nil {
		return true
	}
	inc := false
	if len(includeZero) > 0 {
		inc = includeZero[0]
	}
	switch v := value.(type) {
	case bool:
		return !v
	case string:
		return v == ""
	case *jvm.List:
		return v == nil || len(v.Elems) == 0
	case []any:
		return len(v) == 0
	}
	if jvm.IsNumber(value) && !inc {
		return jvm.DoubleValue(value) == 0
	}
	return false
}

// Options contains the arguments and execution state of one helper call.
type Options struct {
	// Handlebars is the engine executing the helper.
	Handlebars *Handlebars
	// Context is the current execution scope.
	Context *Context
	// Fn is the block body, when present.
	Fn Template
	// Inverse is the else body, when present.
	Inverse Template
	// Params contains positional arguments after the helper context.
	Params []any
	// Hash contains named arguments.
	Hash map[string]any
	// TagType identifies the invoking tag form.
	TagType TagType
	// HelperName is the registry name used for this invocation.
	HelperName string
	// BlockParams contains declared block-parameter names.
	BlockParams []string
	// Writer is the active render destination.
	Writer io.Writer
	state  *renderState
}

// Param returns a positional argument or defaultValue.
func (o *Options) Param(index int, defaultValue ...any) any {
	if index >= 0 && index < len(o.Params) && o.Params[index] != nil {
		return o.Params[index]
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// HashValue returns a hash argument or defaultValue.
func (o *Options) HashValue(name string, defaultValue ...any) any {
	if v, ok := o.Hash[name]; ok && v != nil {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// Resolve looks up a value in the current context and preserves resolver errors.
func (o *Options) Resolve(name string) (any, error) {
	return o.Context.Resolve(name, o.Handlebars.parentScopeResolution)
}

// IsFalsy tests a value with the includeZero hash option.
func (o *Options) IsFalsy(value any) bool {
	include, _ := o.Hash["includeZero"].(bool)
	return IsEmpty(value, include)
}

// Data returns context data.
func (o *Options) Data(name string) any { return o.Context.Data(name) }

// SetData writes context data.
func (o *Options) SetData(name string, value any) { o.Context.SetData(name, value) }

// Partial returns a registered partial.
func (o *Options) Partial(path string) Template {
	m, _ := o.Context.Data(ContextPartials).(map[string]Template)
	return m[path]
}

// SetPartial registers a partial.
func (o *Options) SetPartial(path string, t Template) {
	m, _ := o.Context.Data(ContextPartials).(map[string]Template)
	m[path] = t
}

// PropertySet lists a value's properties.
func (o *Options) PropertySet(value any) ([]Property, error) { return o.Context.PropertySet(value) }

// Wrap turns a value into a child context.
func (o *Options) Wrap(value any) *Context {
	if c, ok := value.(*Context); ok {
		return c
	}
	return NewChildContext(o.Context, value)
}

// ApplyFn applies the block body to value.
func (o *Options) ApplyFn(value any) (string, error) {
	if o.Fn == nil {
		return "", nil
	}
	return o.Fn.ApplyContext(o.Wrap(value))
}

// ApplyInverse applies the inverse body to value.
func (o *Options) ApplyInverse(value any) (string, error) {
	if o.Inverse == nil {
		return "", nil
	}
	return o.Inverse.ApplyContext(o.Wrap(value))
}

// String concatenates positional parameters using Java string conversion.
func (o *Options) String() string {
	var b strings.Builder
	for _, v := range o.Params {
		b.WriteString(jvm.ToString(v))
	}
	return b.String()
}
