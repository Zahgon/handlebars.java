// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

// Package handlebars implements the Handlebars.java template engine.
package handlebars

import (
	"fmt"
	"io"
	"sync"
)

// TagType identifies a parsed Handlebars tag.
type TagType int

const (
	// TagVar is an escaped variable.
	TagVar TagType = iota
	// TagStarVar is an inline decorator.
	TagStarVar
	// TagAmpVar is an ampersand, unescaped variable.
	TagAmpVar
	// TagTripleVar is a triple-mustache variable.
	TagTripleVar
	// TagSubExpression is a parenthesized expression.
	TagSubExpression
	// TagSection is a block section.
	TagSection
	// TagStartSection is a block decorator section.
	TagStartSection
)

// Inline reports whether the tag is inline.
func (t TagType) Inline() bool { return t != TagSection && t != TagStartSection }

// Template is a compiled Handlebars template.
type Template interface {
	// Apply renders against a model and returns the complete output.
	Apply(context any) (string, error)
	// ApplyContext renders against an existing execution Context.
	ApplyContext(context *Context) (string, error)
	// ApplyWriter streams a model's output to writer.
	ApplyWriter(context any, writer io.Writer) error
	// ApplyContextWriter streams an execution Context's output to writer.
	ApplyContextWriter(context *Context, writer io.Writer) error
	// Text returns the original template source.
	Text() string
	// ToJavaScript returns precompiled JavaScript or a missing-engine error.
	ToJavaScript() (string, error)
	// Filename returns the logical source name.
	Filename() string
	// Position returns the template's one-based source position.
	Position() (line, column int)
	// Collect lists expressions having one of the requested tag types.
	Collect(types ...TagType) []string
	// CollectReferenceParameters lists referenced paths in source order.
	CollectReferenceParameters() []string
}

// TypedTemplate is a generic type-safe view of a Template.
type TypedTemplate[T any] struct {
	// Template is the compiled template wrapped by this typed view.
	Template Template
}

// Apply renders the typed template.
func (t TypedTemplate[T]) Apply(context T) (string, error) { return t.Template.Apply(context) }

// ApplyWriter renders the typed template into writer.
func (t TypedTemplate[T]) ApplyWriter(context T, writer io.Writer) error {
	return t.Template.ApplyWriter(context, writer)
}

// Helper is a callable Handlebars helper.
type Helper func(context any, options *Options) (any, error)

// Decorator is a callable Handlebars decorator.
type Decorator func(template Template, options *Options) (Template, error)

// Lambda is a template lambda.
type Lambda func(context any, template Template) (any, error)

// Formatter converts a value before escaping and output.
type Formatter interface {
	Format(value any) (formatted any, accepted bool)
}

// FormatterFunc adapts a function to Formatter.
type FormatterFunc func(value any) (any, bool)

// Format calls f.
func (f FormatterFunc) Format(value any) (any, bool) { return f(value) }

// Error is a source-positioned parse or render error. Cause is available via
// errors.Unwrap when the diagnostic originated from another error.
type Error struct {
	// Filename is the logical template name.
	Filename string
	// Line is the one-based template line.
	Line int
	// Column is the one-based template column used by the Java API.
	Column int
	// Reason is the unadorned failure description.
	Reason string
	// Evidence is the source excerpt or render position shown to the user.
	Evidence string
	// Message is the complete formatted diagnostic.
	Message string
	// Cause is the underlying resolver, helper, loader, or I/O error.
	Cause error
}

// Error returns the complete diagnostic.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap returns the underlying failure.
func (e *Error) Unwrap() error { return e.Cause }

// SafeString marks text that must not be escaped.
type SafeString struct {
	// Content is emitted without applying the configured escaping strategy.
	Content string
}

// NewSafeString creates unescaped text.
func NewSafeString(content any) SafeString { return SafeString{Content: fmt.Sprint(content)} }

// String returns the safe content.
func (s SafeString) String() string { return s.Content }

// JavaClass implements the shared Java object model.
func (s SafeString) JavaClass() string { return "com.github.jknack.handlebars.Handlebars$SafeString" }

// JavaString implements the shared Java object model.
func (s SafeString) JavaString() string { return s.Content }

// ValueResolver resolves named values from an object.
type ValueResolver interface {
	// Resolve looks up a named member and returns lookup failures unchanged.
	Resolve(context any, name string) (value any, resolved bool, err error)
	// ResolveValue optionally converts a context value.
	ResolveValue(context any) (value any, resolved bool, err error)
	// PropertySet enumerates visible properties in resolver order.
	PropertySet(context any) ([]Property, error)
}

// Property is one resolved object property.
type Property struct {
	// Name is the visible property name.
	Name string
	// Value is the resolved property value.
	Value any
}

// TemplateSource provides named template text.
type TemplateSource interface {
	// Content returns template text or its read error.
	Content() (string, error)
	// Filename returns the logical source name used in diagnostics and caches.
	Filename() string
	// LastModified returns the source version or its metadata error.
	LastModified() (int64, error)
}

// Parser parses TemplateSource objects.
type Parser interface {
	// Parse compiles one source.
	Parse(source TemplateSource) (Template, error)
}

// ParserFactory creates parsers owned by a Handlebars instance.
type ParserFactory interface {
	// Create constructs a parser for the supplied delimiters.
	Create(handlebars *Handlebars, startDelimiter, endDelimiter string) Parser
}

// TemplateLoader locates templates.
type TemplateLoader interface {
	// SourceAt locates a source or returns the underlying loader error.
	SourceAt(location string) (TemplateSource, error)
	// Resolve returns the loader-specific display path.
	Resolve(location string) string
	// Prefix returns the configured logical prefix.
	Prefix() string
	// Suffix returns the configured template suffix.
	Suffix() string
}

// TemplateCache caches parsed templates.
type TemplateCache interface {
	// Get returns a cached template or parses and caches the source.
	Get(source TemplateSource, parser Parser) (Template, error)
	// Evict removes one source.
	Evict(source TemplateSource)
	// Clear removes all cached sources.
	Clear()
	// WithReload controls source modification-time validation.
	WithReload(reload bool) TemplateCache
}

type templateCacheEntry struct {
	template Template
	modified int64
}

// ConcurrentMapTemplateCache is the default thread-safe template cache.
type ConcurrentMapTemplateCache struct {
	mu     sync.RWMutex
	values map[string]templateCacheEntry
	reload bool
}

// NewConcurrentMapTemplateCache creates an empty cache.
func NewConcurrentMapTemplateCache() *ConcurrentMapTemplateCache {
	return &ConcurrentMapTemplateCache{values: make(map[string]templateCacheEntry)}
}

// Get returns or parses a template.
func (c *ConcurrentMapTemplateCache) Get(source TemplateSource, parser Parser) (Template, error) {
	name := source.Filename()
	modified, err := source.LastModified()
	if err != nil {
		return nil, fmt.Errorf("last modified time for %s: %w", name, err)
	}
	c.mu.RLock()
	entry, ok := c.values[name]
	c.mu.RUnlock()
	if ok && (!c.reload || entry.modified == modified) {
		return entry.template, nil
	}
	t, err := parser.Parse(source)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.values[name] = templateCacheEntry{template: t, modified: modified}
	c.mu.Unlock()
	return t, nil
}

// Evict removes one source.
func (c *ConcurrentMapTemplateCache) Evict(source TemplateSource) {
	c.mu.Lock()
	delete(c.values, source.Filename())
	c.mu.Unlock()
}

// Clear removes all entries.
func (c *ConcurrentMapTemplateCache) Clear() {
	c.mu.Lock()
	c.values = make(map[string]templateCacheEntry)
	c.mu.Unlock()
}

// WithReload enables or disables modification-time validation.
func (c *ConcurrentMapTemplateCache) WithReload(reload bool) TemplateCache {
	c.reload = reload
	return c
}

// HighConcurrencyTemplateCache is the reload-aware concurrent cache.
type HighConcurrencyTemplateCache = ConcurrentMapTemplateCache

type nullTemplateCache struct{}

// NullTemplateCache never caches a parsed template.
var NullTemplateCache TemplateCache = nullTemplateCache{}

func (nullTemplateCache) Get(s TemplateSource, p Parser) (Template, error) { return p.Parse(s) }
func (nullTemplateCache) Evict(TemplateSource)                             {}
func (nullTemplateCache) Clear()                                           {}
func (n nullTemplateCache) WithReload(bool) TemplateCache                  { return n }
