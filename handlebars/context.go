// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

const (
	// ContextPartials is the execution-context partial registry key.
	ContextPartials = "com.github.jknack.handlebars.Context#partials"
	// ContextInlinePartials is the inline partial stack key.
	ContextInlinePartials = "__inline_partials_"
	// ContextInvocationStack is the partial invocation stack key.
	ContextInvocationStack = "com.github.jknack.handlebars.Context#invocationStack"
	// ContextParamSize is the current helper parameter count key.
	ContextParamSize = "com.github.jknack.handlebars.Context#paramSize"
	// ContextCallee is the partial-block callee key.
	ContextCallee = "com.github.jknack.handlebars.Context#callee"
)

// Context is one scope in the template context stack.
type Context struct {
	model       any
	parent      *Context
	data        map[string]any
	extended    map[string]any
	blockParams map[string]any
	partialHash map[string]any
	resolvers   []ValueResolver
	local       bool
	blockParam  bool
	parentFirst bool
	// shiftedAlias is the parent property whose value became a with scope.
	shiftedAlias string
}

// ContextBuilder incrementally creates a Context.
type ContextBuilder struct{ context *Context }

// NewContext creates a root context.
func NewContext(model any) *Context {
	c := &Context{model: model, data: make(map[string]any), extended: make(map[string]any), resolvers: DefaultValueResolvers()}
	c.data[ContextPartials] = make(map[string]Template)
	c.data[ContextInlinePartials] = []map[string]Template{{}}
	c.data[ContextInvocationStack] = []string{}
	c.data["root"] = model
	return c
}

// NewChildContext creates a child scope.
func NewChildContext(parent *Context, model any) *Context {
	if parent == nil {
		return NewContext(model)
	}
	return &Context{model: model, parent: parent, data: parent.data, extended: make(map[string]any), resolvers: parent.resolvers, parentFirst: parent.blockParam || parent.parentFirst}
}

func newDecoratorContext(context *Context) *Context {
	if context == nil {
		return NewContext(nil)
	}
	return &Context{data: context.data, extended: make(map[string]any), resolvers: context.resolvers}
}

// NewPartialContext creates a partial scope with hash values taking precedence.
func NewPartialContext(parent *Context, model any, hash map[string]any) *Context {
	c := NewChildContext(parent, model)
	c.partialHash = hash
	c.parentFirst = false
	c.blockParam = false
	return c
}

// NewBlockParamContext creates a scope containing named block parameters.
func NewBlockParamContext(parent *Context, model any, names []string, values []any) *Context {
	hash := make(map[string]any)
	javaMap := jvm.NewLinkedHashMap()
	c := NewChildContext(parent, javaMap)
	c.blockParam = true
	c.parentFirst = false
	c.blockParams = hash
	for i, name := range names {
		if i < len(values) {
			c.blockParams[name] = values[i]
			javaMap.Put(name, values[i])
		}
	}
	return c
}

// CopyContext copies a context while sharing execution data.
func CopyContext(context *Context, model any) *Context {
	if context == nil {
		return NewContext(model)
	}
	if model == nil {
		model = context.model
	}
	c := NewChildContext(context.parent, model)
	c.data = context.data
	for k, v := range context.extended {
		c.extended[k] = v
	}
	return c
}

// NewContextBuilder creates a root context builder.
func NewContextBuilder(model any) *ContextBuilder { return &ContextBuilder{context: NewContext(model)} }

// NewChildContextBuilder creates a child context builder.
func NewChildContextBuilder(parent *Context, model any) *ContextBuilder {
	return &ContextBuilder{context: NewChildContext(parent, model)}
}

// Combine adds one extended value.
func (b *ContextBuilder) Combine(name string, value any) *ContextBuilder {
	b.context.extended[name] = value
	return b
}

// CombineMap adds extended values.
func (b *ContextBuilder) CombineMap(values map[string]any) *ContextBuilder {
	for k, v := range values {
		b.context.extended[k] = v
	}
	return b
}

// Resolver replaces the value resolver list.
func (b *ContextBuilder) Resolver(resolvers ...ValueResolver) *ContextBuilder {
	b.context.resolvers = append([]ValueResolver(nil), resolvers...)
	return b
}

// Build returns the context.
func (b *ContextBuilder) Build() *Context { return b.context }

// Model returns the current scope object.
func (c *Context) Model() any { return c.model }

// Parent returns the parent scope.
func (c *Context) Parent() *Context { return c.parent }

// Combine adds one extended value.
func (c *Context) Combine(name string, value any) *Context {
	c.extended[name] = value
	return c
}

// SetData stores an execution-scoped value.
func (c *Context) SetData(name string, value any) *Context {
	c.data[name] = value
	return c
}

// Data returns an execution-scoped value.
func (c *Context) Data(name string) any { return c.data[name] }

// Destroy releases this context's references.
func (c *Context) Destroy() {
	c.model, c.parent, c.data, c.extended, c.blockParams, c.partialHash = nil, nil, nil, nil, nil, nil
}

// Resolve resolves a Handlebars path, walking parent scopes when enabled, and
// returns member-access failures from the configured ValueResolvers.
func (c *Context) Resolve(path string, parentScopeResolution ...bool) (any, error) {
	value := c.lookup(path, parentScopeResolution...)
	if err, ok := value.(error); ok {
		return nil, err
	}
	return value, nil
}

func (c *Context) lookup(path string, parentScopeResolution ...bool) any {
	parentLookup := true
	if len(parentScopeResolution) != 0 {
		parentLookup = parentScopeResolution[0]
	}
	if path == "" {
		return nil
	}
	if c.parentFirst && c.parent != nil && !strings.HasPrefix(path, "../") {
		if value := c.parent.lookup(path, parentLookup); value != nil {
			return value
		}
	}
	cur := c
	for strings.HasPrefix(path, "../") {
		if cur != nil {
			cur = cur.parent
		}
		path = path[3:]
	}
	if path == ".." {
		if cur != nil && cur.parent != nil {
			return cur.parent.model
		}
		return nil
	}
	local := strings.HasPrefix(path, "./") || path == "." || path == "this"
	path = strings.TrimPrefix(path, "./")
	if path == "." || path == "this" || path == "" {
		if cur == nil {
			return nil
		}
		if path == "this" && cur.blockParam && cur.parent != nil {
			return cur.parent.model
		}
		return cur.model
	}
	if strings.HasPrefix(path, "@root") {
		root := c.data["root"]
		rest := strings.TrimPrefix(path, "@root")
		rest = strings.TrimLeft(rest, "./")
		if rest == "" {
			return root
		}
		return resolveSegments(c, root, splitPath(rest))
	}
	if strings.HasPrefix(path, "@") {
		key := strings.TrimPrefix(path, "@")
		for strings.HasPrefix(key, "../") {
			if cur != nil {
				cur = cur.parent
			}
			key = key[3:]
		}
		parts := splitPath(key)
		if len(parts) == 0 {
			return jvm.Throw("java.lang.IndexOutOfBoundsException", "Index 0 out of bounds for length 0")
		}
		for it := cur; it != nil; it = it.parent {
			if value, ok := it.extended["@"+parts[0]]; ok {
				return resolveSegments(c, value, parts[1:])
			}
			if value, ok := it.extended[parts[0]]; ok {
				return resolveSegments(c, value, parts[1:])
			}
			if value, ok := it.data[parts[0]]; ok {
				return resolveSegments(c, value, parts[1:])
			}
		}
		return nil
	}
	segments := splitPath(path)
	for it := cur; it != nil; it = it.parent {
		if value, ok := lookupFirst(it, segments[0]); ok {
			return resolveSegments(c, value, segments[1:])
		}
		if (c.shiftedAlias != "" && segments[0] == c.shiftedAlias && len(segments) > 1) || local || !parentLookup {
			break
		}
	}
	return nil
}

func lookupFirst(c *Context, name string) (any, bool) {
	if c.blockParams != nil {
		if v, ok := c.blockParams[name]; ok {
			return v, true
		}
	}
	if c.partialHash != nil && name != "this" {
		if v, ok := c.partialHash[name]; ok {
			return v, true
		}
	}
	if v, ok := c.extended[name]; ok {
		return v, true
	}
	return resolveProperty(c, c.model, name)
}

func resolveSegments(c *Context, value any, parts []string) any {
	var ok bool
	for _, part := range parts {
		value, ok = resolveProperty(c, value, part)
		if !ok {
			return nil
		}
		if _, failed := value.(error); failed {
			return value
		}
	}
	return value
}

func splitPath(path string) []string {
	var result []string
	for len(path) > 0 {
		if path[0] == '[' {
			end := -1
			for i := 1; i < len(path); i++ {
				if path[i] == '\\' && i+1 < len(path) && path[i+1] == ']' {
					i++
					continue
				}
				if path[i] == ']' {
					end = i
					break
				}
			}
			if end >= 0 {
				// PathCompiler preserves the backslash in an escaped right
				// bracket; it is part of the Java property name (Issue611).
				result = append(result, path[1:end])
				path = strings.TrimLeft(path[end+1:], "./-")
				continue
			}
		}
		i := strings.IndexAny(path, "./")
		if i < 0 {
			result = append(result, path)
			break
		}
		if i > 0 {
			result = append(result, path[:i])
		}
		path = path[i+1:]
	}
	return result
}

func resolveProperty(c *Context, value any, name string) (any, bool) {
	if value == nil {
		return nil, false
	}
	if name == "this" || name == "." || name == "" {
		return value, true
	}
	for _, resolver := range c.resolvers {
		v, ok, err := resolver.Resolve(value, name)
		if err != nil {
			return err, true
		}
		if ok {
			return v, true
		}
	}
	switch x := value.(type) {
	case *jvm.Map:
		return x.Get(name)
	case map[string]any:
		v, ok := x[name]
		return v, ok
	case *jvm.List:
		if name == "length" || name == "size" {
			return int32(len(x.Elems)), true
		}
		i, err := strconv.Atoi(name)
		if err == nil && i >= 0 && i < len(x.Elems) {
			return x.Elems[i], true
		}
	case []any:
		if name == "length" || name == "size" {
			return int32(len(x)), true
		}
		i, err := strconv.Atoi(name)
		if err == nil && i >= 0 && i < len(x) {
			return x[i], true
		}
	case string:
		if name == "length" {
			return int32(jvm.UTF16Len(x)), true
		}
	}
	if v, ok, err := jvm.GetProperty(value, name); err != nil {
		return err, true
	} else if ok {
		return v, true
	}
	if v, ok, err := jvm.InvokeMember(value, name); err != nil {
		return err, true
	} else if ok {
		return v, true
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName(name)
		if !field.IsValid() {
			field = rv.FieldByName(strings.ToUpper(name[:1]) + name[1:])
		}
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), true
		}
	}
	return nil, false
}

// PropertySet lists properties of a value in resolver iteration order: a map's entries, else the first non-empty
// property set of the context's value resolvers. A resolver's error is returned.
func (c *Context) PropertySet(value any) ([]Property, error) {
	if value == nil {
		return nil, nil
	}
	if m, ok := value.(*jvm.Map); ok {
		entries := m.Entries()
		out := make([]Property, 0, len(entries))
		for _, e := range entries {
			out = append(out, Property{Name: jvm.ToString(e.Key), Value: e.Value})
		}
		return out, nil
	}
	for _, resolver := range c.resolvers {
		set, err := resolver.PropertySet(value)
		if err != nil {
			return nil, err
		}
		if len(set) != 0 {
			return set, nil
		}
	}
	if entries, ok := jvm.PropertySet(value); ok {
		out := make([]Property, 0, len(entries))
		for _, e := range entries {
			out = append(out, Property{Name: jvm.ToString(e.Key), Value: e.Value})
		}
		return out, nil
	}
	return nil, nil
}
