// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

type compiledTemplate struct {
	engine   *Handlebars
	filename string
	input    string
	nodes    []astNode
	line     int
	column   int
}

// indentedTextWriter indents literal template lines while leaving newlines
// produced inside dynamic values untouched, matching Java's partial writer.
type indentedTextWriter struct {
	writer    io.Writer
	indent    string
	lineStart bool
}

func (w *indentedTextWriter) Write(p []byte) (int, error) {
	if len(p) > 0 && w.lineStart && p[0] != '\n' && p[0] != '\r' {
		if _, err := io.WriteString(w.writer, w.indent); err != nil {
			return 0, err
		}
		w.lineStart = false
	}
	n, err := w.writer.Write(p)
	if n > 0 {
		last := p[n-1]
		w.lineStart = last == '\n'
	}
	return n, err
}

func (w *indentedTextWriter) WriteTemplateText(text string) error {
	for len(text) > 0 {
		if w.lineStart && text[0] != '\n' {
			if _, err := io.WriteString(w.writer, w.indent); err != nil {
				return err
			}
			w.lineStart = false
		}
		end := strings.IndexByte(text, '\n')
		if end < 0 {
			_, err := io.WriteString(w.writer, text)
			w.lineStart = false
			return err
		}
		end++
		if _, err := io.WriteString(w.writer, text[:end]); err != nil {
			return err
		}
		w.lineStart = true
		text = text[end:]
	}
	return nil
}

func newCompiledTemplate(engine *Handlebars, filename, input string, nodes []astNode) Template {
	return &compiledTemplate{engine: engine, filename: filename, input: input, nodes: nodes, line: 1}
}

type renderState struct {
	engine    *Handlebars
	context   *Context
	writer    io.Writer
	filename  string
	line      int
	column    int
	decorated map[astNode]bool
}

type positionedError struct {
	line, column int
	err          error
}

func renderPositionLine(pos position) int {
	if pos.renderLine > 0 {
		return pos.renderLine
	}
	return pos.line
}

func (e *positionedError) Error() string { return e.err.Error() }
func (e *positionedError) Unwrap() error { return e.err }

func (t *compiledTemplate) Apply(context any) (string, error) {
	if c, ok := context.(*Context); ok {
		return t.ApplyContext(c)
	}
	return t.ApplyContext(NewContext(context))
}
func (t *compiledTemplate) ApplyContext(context *Context) (string, error) {
	var b strings.Builder
	err := t.ApplyContextWriter(context, &b)
	return b.String(), err
}
func (t *compiledTemplate) ApplyWriter(context any, writer io.Writer) error {
	if c, ok := context.(*Context); ok {
		return t.ApplyContextWriter(c, writer)
	}
	return t.ApplyContextWriter(NewContext(context), writer)
}
func (t *compiledTemplate) ApplyContextWriter(context *Context, writer io.Writer) error {
	if context == nil {
		context = NewContext(nil)
	}
	s := &renderState{engine: t.engine, context: context, writer: writer, filename: t.filename, line: t.line, column: t.column, decorated: make(map[astNode]bool)}
	stack, _ := context.Data(ContextInvocationStack).([]string)
	stack = append(stack, t.filename)
	context.SetData(ContextInvocationStack, stack)
	defer func() { context.SetData(ContextInvocationStack, stack[:len(stack)-1]) }()
	if err := applyAllDecorators(s, t.nodes); err != nil {
		return t.wrapRenderError(s, err)
	}
	if err := renderNodes(s, t.nodes); err != nil {
		return t.wrapRenderError(s, err)
	}
	return nil
}
func (t *compiledTemplate) wrapRenderError(s *renderState, err error) error {
	if positioned, ok := err.(*positionedError); ok {
		s.line, s.column, err = positioned.line, positioned.column, positioned.err
	}
	if _, ok := err.(*Error); ok {
		return err
	}
	if ex, ok := err.(*jvm.Exception); ok && ex.Class == "java.lang.StackOverflowError" {
		return ex
	}
	reason := err.Error()
	evidence := fmt.Sprintf("%s:%d:%d", t.filename, s.line, max(1, s.column))
	message := fmt.Sprintf("%s:%d:%d: %s\n    %s", t.filename, s.line, max(1, s.column), reason, evidence)
	return &Error{Filename: t.filename, Line: s.line, Column: max(1, s.column), Reason: reason, Evidence: evidence, Message: message, Cause: err}
}
func (t *compiledTemplate) Text() string {
	return t.input
}
func (t *compiledTemplate) ToJavaScript() (string, error) { return t.engine.PrecompileInline(t.input) }
func (t *compiledTemplate) Filename() string              { return t.filename }
func (t *compiledTemplate) Position() (int, int)          { return t.line, t.column }
func (t *compiledTemplate) Collect(types ...TagType) []string {
	wanted := make(map[TagType]bool)
	for _, v := range types {
		wanted[v] = true
	}
	seen := make(map[string]bool)
	var out []string
	collectOrdered(t.nodes, wanted, seen, &out)
	return out
}
func (t *compiledTemplate) CollectReferenceParameters() []string {
	seen := make(map[string]bool)
	var out []string
	collectRefsOrdered(t.nodes, seen, &out)
	return out
}

func addOrdered(name string, seen map[string]bool, out *[]string) {
	if name != "" && !seen[name] {
		seen[name] = true
		*out = append(*out, name)
	}
}
func collectOrdered(nodes []astNode, wanted map[TagType]bool, seen map[string]bool, out *[]string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *varNode:
			for _, p := range n.expr.params {
				if sub, ok := p.(subexprParam); ok && wanted[TagSubExpression] {
					addOrdered(sub.expr.name, seen, out)
				}
			}
			if wanted[n.type_] {
				addOrdered(n.expr.name, seen, out)
			}
		case *blockNode:
			collectOrdered(n.body, wanted, seen, out)
			collectOrdered(n.inverse, wanted, seen, out)
			tag := TagSection
			if n.decorator {
				tag = TagStartSection
			}
			if wanted[tag] {
				addOrdered(n.expr.name, seen, out)
			}
		case *partialNode:
			// Partial-block fallback bodies are not part of Template.collect.
		}
	}
}
func collectRefsOrdered(nodes []astNode, seen map[string]bool, out *[]string) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *varNode:
			for _, p := range n.expr.params {
				if r, ok := p.(refParam); ok {
					addOrdered(r.path, seen, out)
				} else if sub, ok := p.(subexprParam); ok {
					collectExprRefs(sub.expr, seen, out)
				}
			}
			for _, p := range n.expr.hash {
				if r, ok := p.(refParam); ok {
					addOrdered(r.path, seen, out)
				}
			}
		case *blockNode:
			collectRefsOrdered(n.body, seen, out)
			collectRefsOrdered(n.inverse, seen, out)
			for _, p := range n.expr.params {
				if r, ok := p.(refParam); ok {
					addOrdered(r.path, seen, out)
				}
				if sub, ok := p.(subexprParam); ok {
					collectExprRefs(sub.expr, seen, out)
				}
			}
			for _, p := range n.expr.hash {
				if r, ok := p.(refParam); ok {
					addOrdered(r.path, seen, out)
				}
			}
		case *partialNode:
			if r, ok := n.context.(refParam); ok {
				addOrdered(r.path, seen, out)
			}
			for _, p := range n.hash {
				if r, ok := p.(refParam); ok {
					addOrdered(r.path, seen, out)
				} else if sub, ok := p.(subexprParam); ok {
					collectExprRefs(sub.expr, seen, out)
				}
			}
		}
	}
}
func collectExprRefs(e expression, seen map[string]bool, out *[]string) {
	for _, p := range e.params {
		if r, ok := p.(refParam); ok {
			addOrdered(r.path, seen, out)
		} else if sub, ok := p.(subexprParam); ok {
			collectExprRefs(sub.expr, seen, out)
		}
	}
	for _, p := range e.hash {
		if r, ok := p.(refParam); ok {
			addOrdered(r.path, seen, out)
		}
	}
}

type nodeTemplate struct {
	engine   *Handlebars
	filename string
	nodes    []astNode
}

type boundTemplate struct {
	Template
	context       *Context
	parentPartial Template
}

func (t *boundTemplate) Apply(any) (string, error) { return t.Template.ApplyContext(t.context) }
func (t *boundTemplate) ApplyContext(*Context) (string, error) {
	return t.Template.ApplyContext(t.context)
}
func (t *boundTemplate) ApplyWriter(_ any, w io.Writer) error {
	return t.Template.ApplyContextWriter(t.context, w)
}
func (t *boundTemplate) ApplyContextWriter(_ *Context, w io.Writer) error {
	partials, _ := t.context.Data(ContextPartials).(map[string]Template)
	current, had := partials["@partial-block"]
	if t.parentPartial == nil {
		delete(partials, "@partial-block")
	} else {
		partials["@partial-block"] = t.parentPartial
	}
	defer func() {
		if had {
			partials["@partial-block"] = current
		} else {
			delete(partials, "@partial-block")
		}
	}()
	if node, ok := t.Template.(*nodeTemplate); ok {
		return node.applyContextWriter(t.context, w, true, true)
	}
	return t.Template.ApplyContextWriter(t.context, w)
}

func (t *nodeTemplate) Apply(v any) (string, error) {
	if c, ok := v.(*Context); ok {
		return t.ApplyContext(c)
	}
	return t.ApplyContext(NewContext(v))
}
func (t *nodeTemplate) ApplyContext(c *Context) (string, error) {
	var b strings.Builder
	err := t.ApplyContextWriter(c, &b)
	return b.String(), err
}
func (t *nodeTemplate) ApplyWriter(v any, w io.Writer) error {
	if c, ok := v.(*Context); ok {
		return t.ApplyContextWriter(c, w)
	}
	return t.ApplyContextWriter(NewContext(v), w)
}
func (t *nodeTemplate) ApplyContextWriter(c *Context, w io.Writer) error {
	return t.applyContextWriter(c, w, true, false)
}

func (t *nodeTemplate) applyContextWriter(c *Context, w io.Writer, pushInlineScope, exportInline bool) error {
	if pushInlineScope {
		stack, _ := c.Data(ContextInlinePartials).([]map[string]Template)
		local := map[string]Template{}
		c.SetData(ContextInlinePartials, append(stack, local))
		defer func() {
			if exportInline && len(stack) != 0 {
				for name, partial := range local {
					stack[len(stack)-1][name] = partial
				}
			}
			c.SetData(ContextInlinePartials, stack)
		}()
	}
	s := &renderState{engine: t.engine, context: c, writer: w, filename: t.filename, line: 1, column: 1, decorated: make(map[astNode]bool)}
	if err := applyAllDecorators(s, t.nodes); err != nil {
		return err
	}
	if err := renderNodes(s, t.nodes); err != nil {
		if _, positioned := err.(*positionedError); !positioned {
			if _, wrapped := err.(*Error); !wrapped {
				return &positionedError{line: s.line, column: s.column, err: err}
			}
		}
		return err
	}
	return nil
}
func (t *nodeTemplate) Text() string {
	var b strings.Builder
	for _, n := range t.nodes {
		b.WriteString(n.source())
	}
	return b.String()
}
func (t *nodeTemplate) ToJavaScript() (string, error) { return t.engine.PrecompileInline(t.Text()) }
func (t *nodeTemplate) Filename() string              { return t.filename }
func (t *nodeTemplate) Position() (int, int)          { return 0, 0 }
func (t *nodeTemplate) Collect(types ...TagType) []string {
	return newCompiledTemplate(t.engine, t.filename, t.Text(), t.nodes).Collect(types...)
}
func (t *nodeTemplate) CollectReferenceParameters() []string {
	return newCompiledTemplate(t.engine, t.filename, t.Text(), t.nodes).CollectReferenceParameters()
}

func renderNodes(s *renderState, nodes []astNode) error {
	for _, n := range nodes {
		if err := n.render(s); err != nil {
			return err
		}
	}
	return nil
}

func (n *varNode) render(s *renderState) error {
	s.line, s.column = renderPositionLine(n.pos), n.pos.column+len(s.engine.startDelimiter)
	if n.type_ == TagStarVar {
		if s.decorated[n] {
			return nil
		}
		s.decorated[n] = true
		decorator := s.engine.Decorator(n.expr.name)
		if decorator == nil {
			return nil
		}
		decoratorState := *s
		decoratorState.context = newDecoratorContext(s.context)
		params, hash, err := n.expr.arguments(&decoratorState, true)
		if err != nil {
			return err
		}
		_, err = decorator(&nodeTemplate{engine: s.engine, filename: s.filename}, &Options{Handlebars: s.engine, Context: decoratorState.context, Params: params, Hash: hash, TagType: n.type_, HelperName: n.expr.name, Writer: s.writer, state: &decoratorState})
		return err
	}
	value, err := n.expr.eval(s, n.type_)
	if err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	if lambda, ok := value.(Lambda); ok {
		done, depthErr := enterLambda(s.context)
		if depthErr != nil {
			return depthErr
		}
		defer done()
		result, callErr := lambda(s.context, &nodeTemplate{engine: s.engine, filename: s.filename, nodes: []astNode{n}})
		if callErr != nil {
			return callErr
		}
		if text, ok := result.(string); ok {
			compiled, compileErr := s.engine.CompileInline(text)
			if compileErr != nil {
				return compileErr
			}
			value, callErr = compiled.ApplyContext(s.context)
			if callErr != nil {
				return callErr
			}
		} else {
			value = result
		}
	}
	if _, ok := value.(SafeString); ok {
		_, err = io.WriteString(s.writer, toJavaString(value))
		return err
	}
	formatted := value
	for _, formatter := range s.engine.formatters {
		if v, accepted := formatter.Format(value); accepted {
			formatted = v
			break
		}
	}
	text := toJavaString(formatted)
	_, formattedSafe := formatted.(SafeString)
	if n.type_ == TagVar && !formattedSafe {
		text = s.engine.escaping.Escape(text)
	}
	_, err = io.WriteString(s.writer, text)
	return err
}

func (e expression) arguments(s *renderState, decorator bool) ([]any, map[string]any, error) {
	values := make([]any, len(e.params))
	for i, p := range e.params {
		v, err := p.eval(s)
		if err != nil {
			return nil, nil, err
		}
		// HelperResolver applies stringParams only to the Options parameter
		// array. The first parameter determines the helper context and is not
		// substituted. Decorators, on the other hand, substitute every param.
		if v == nil && s.engine.stringParams && (decorator || i > 0) {
			if ref, ok := p.(refParam); ok {
				v = ref.path
			}
		}
		values[i] = v
	}
	hash := make(map[string]any, len(e.hash))
	for k, p := range e.hash {
		v, err := p.eval(s)
		if err != nil {
			return nil, nil, err
		}
		hash[k] = v
	}
	return values, hash, nil
}

func (e expression) eval(s *renderState, tag TagType) (any, error) {
	params, hash, err := e.arguments(s, false)
	if err != nil {
		return nil, err
	}
	helper := s.engine.Helper(e.name)
	if helper != nil {
		context := s.context.Model()
		rest := params
		if len(params) > 0 {
			context = params[0]
			rest = params[1:]
		}
		return helper(context, &Options{Handlebars: s.engine, Context: s.context, Params: rest, Hash: hash, TagType: tag, HelperName: e.name, Writer: s.writer, state: s})
	}
	value, resolveErr := s.context.Resolve(e.name, s.engine.parentScopeResolution)
	if resolveErr != nil {
		return nil, resolveErr
	}
	if value == nil {
		if missing := s.engine.Helper(HelperMissing); missing != nil {
			context := s.context.Model()
			rest := params
			if len(params) > 0 {
				context, rest = params[0], params[1:]
			}
			return missing(context, &Options{Handlebars: s.engine, Context: s.context, Params: rest, Hash: hash, TagType: tag, HelperName: e.name, Writer: s.writer, state: s})
		}
		if len(params) > 0 || len(hash) > 0 {
			return nil, jvm.Throw("java.lang.IllegalArgumentException", "could not find helper: '"+e.name+"'")
		}
	}
	return value, nil
}

func (n *blockNode) render(s *renderState) error {
	s.line, s.column = renderPositionLine(n.pos), n.pos.column+len(s.engine.startDelimiter)+1
	if n.decorator && n.expr.name == "inline" {
		if s.decorated[n] {
			return nil
		}
		s.decorated[n] = true
		decoratorState := *s
		decoratorState.context = newDecoratorContext(s.context)
		params, _, err := n.expr.arguments(&decoratorState, true)
		if err != nil {
			return err
		}
		if len(params) == 0 {
			return nil
		}
		name := toJavaString(params[0])
		stack, _ := s.context.Data(ContextInlinePartials).([]map[string]Template)
		if len(stack) == 0 {
			stack = []map[string]Template{{}}
		}
		stack[len(stack)-1][name] = &nodeTemplate{engine: s.engine, filename: s.filename, nodes: n.body}
		s.context.SetData(ContextInlinePartials, stack)
		return nil
	}
	params, hash, err := n.expr.arguments(s, n.decorator)
	if err != nil {
		return err
	}
	bodyT := &nodeTemplate{engine: s.engine, filename: s.filename, nodes: n.body}
	inverseT := &nodeTemplate{engine: s.engine, filename: s.filename, nodes: n.inverse}
	if n.decorator {
		if s.decorated[n] {
			return nil
		}
		s.decorated[n] = true
		decorator := s.engine.Decorator(n.expr.name)
		if decorator == nil {
			return nil
		}
		decoratorState := *s
		decoratorState.context = newDecoratorContext(s.context)
		params, hash, err = n.expr.arguments(&decoratorState, true)
		if err != nil {
			return err
		}
		_, err := decorator(bodyT, &Options{Handlebars: s.engine, Context: decoratorState.context, Fn: bodyT, Inverse: inverseT, Params: params, Hash: hash, TagType: TagStartSection, HelperName: n.expr.name, BlockParams: n.blockParams, Writer: s.writer, state: &decoratorState})
		return err
	}
	helper := s.engine.Helper(n.expr.name)
	if helper != nil {
		if err := applyImmediateDecorators(s, n.body); err != nil {
			return err
		}
		ctx := s.context.Model()
		rest := params
		if len(params) > 0 {
			ctx = params[0]
			rest = params[1:]
		}
		result, callErr := helper(ctx, &Options{Handlebars: s.engine, Context: s.context, Fn: bodyT, Inverse: inverseT, Params: rest, Hash: hash, TagType: TagSection, HelperName: n.expr.name, BlockParams: n.blockParams, Writer: s.writer, state: s})
		if callErr != nil {
			return callErr
		}
		if result != nil {
			_, callErr = io.WriteString(s.writer, toJavaString(result))
		}
		return callErr
	}
	value, resolveErr := s.context.Resolve(n.expr.name, s.engine.parentScopeResolution)
	if resolveErr != nil {
		return resolveErr
	}
	if value == nil {
		if missing := s.engine.Helper(HelperMissing); missing != nil {
			result, callErr := missing(s.context.Model(), &Options{Handlebars: s.engine, Context: s.context, Fn: bodyT, Inverse: inverseT, Params: params, Hash: hash, TagType: TagSection, HelperName: n.expr.name, BlockParams: n.blockParams, Writer: s.writer, state: s})
			if callErr != nil {
				return callErr
			}
			if result != nil {
				_, callErr = io.WriteString(s.writer, toJavaString(result))
			}
			return callErr
		}
	}
	if n.inverted {
		if _, lambda := value.(Lambda); lambda {
			return renderNodes(s, n.inverse)
		}
	}
	if lambda, ok := value.(Lambda); ok {
		done, depthErr := enterLambda(s.context)
		if depthErr != nil {
			return depthErr
		}
		defer done()
		result, callErr := lambda(s.context, bodyT)
		if callErr != nil {
			return callErr
		}
		if text, ok := result.(string); ok {
			start, end := blockDelimiters(n, s.engine.startDelimiter, s.engine.endDelimiter)
			compiled, compileErr := s.engine.CompileInlineWithDelimiters(text, start, end)
			if compileErr != nil {
				return compileErr
			}
			return compiled.ApplyContextWriter(s.context, s.writer)
		}
		_, callErr = io.WriteString(s.writer, toJavaString(result))
		return callErr
	}
	if n.inverted {
		if IsEmpty(value) {
			return renderNodes(s, n.body)
		}
		return renderNodes(s, n.inverse)
	}
	if list, ok := elements(value); ok {
		if len(list) == 0 {
			return renderNodes(s, n.inverse)
		}
		for i, item := range list {
			child := NewChildContext(s.context, item)
			child.Combine("@key", int32(i)).Combine("@index", int32(i)).Combine("@first", map[bool]string{true: "first"}[i == 0]).Combine("@last", map[bool]string{true: "last"}[i == len(list)-1]).Combine("@odd", map[bool]string{true: "odd"}[i%2 == 1]).Combine("@even", map[bool]string{true: "even"}[i%2 == 0]).Combine("@index_1", int32(i+1))
			if len(n.blockParams) > 0 {
				child = NewBlockParamContext(child, item, n.blockParams, []any{item, int32(i)})
			}
			nested := *s
			nested.context = child
			if err := renderNodes(&nested, n.body); err != nil {
				s.line, s.column = nested.line, nested.column
				return err
			}
		}
		return nil
	}
	if IsEmpty(value) {
		return renderNodes(s, n.inverse)
	}
	if _, boolean := value.(bool); boolean {
		return renderNodes(s, n.body)
	}
	child := NewChildContext(s.context, value)
	if len(n.blockParams) > 0 {
		child = NewBlockParamContext(child, value, n.blockParams, []any{value})
	}
	nested := *s
	nested.context = child
	if err := renderNodes(&nested, n.body); err != nil {
		s.line, s.column = nested.line, nested.column
		return err
	}
	return nil
}

func blockDelimiters(n *blockNode, fallbackStart, fallbackEnd string) (string, string) {
	raw := strings.TrimSpace(n.sourceText)
	hash := strings.IndexByte(raw, '#')
	if hash <= 0 {
		return fallbackStart, fallbackEnd
	}
	start := strings.TrimSuffix(raw[:hash], "~")
	rest := raw[hash+1:]
	if n.decorator {
		rest = strings.TrimPrefix(rest, "*")
	}
	if !strings.HasPrefix(strings.TrimSpace(rest), n.expr.name) {
		return fallbackStart, fallbackEnd
	}
	nameAt := strings.Index(rest, n.expr.name)
	end := strings.TrimSpace(rest[nameAt+len(n.expr.name):])
	end = strings.TrimPrefix(end, "~")
	if start == "" || end == "" || strings.ContainsAny(end, " \t\r\n") {
		return fallbackStart, fallbackEnd
	}
	return start, end
}

func applyImmediateDecorators(s *renderState, nodes []astNode) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *varNode:
			if n.type_ == TagStarVar {
				if err := n.render(s); err != nil {
					return err
				}
			}
		case *blockNode:
			if n.decorator && n.expr.name != "inline" {
				if err := n.render(s); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func applyAllDecorators(s *renderState, nodes []astNode) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *varNode:
			if n.type_ == TagStarVar {
				if err := n.render(s); err != nil {
					return err
				}
			}
		case *blockNode:
			if n.decorator && n.expr.name != "inline" {
				if err := n.render(s); err != nil {
					return err
				}
			} else if !n.decorator {
				if err := applyAllDecorators(s, n.body); err != nil {
					return err
				}
				if err := applyAllDecorators(s, n.inverse); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func elements(value any) ([]any, bool) {
	switch v := value.(type) {
	case *jvm.List:
		return v.Elems, true
	case []any:
		return v, true
	}
	return nil, false
}

func (n *partialNode) render(s *renderState) error {
	columnOffset := len(s.engine.startDelimiter) + 1
	if at := strings.Index(n.raw, n.path.text()); at >= 0 {
		columnOffset = at
	}
	s.line, s.column = renderPositionLine(n.pos), n.pos.column+columnOffset
	pathValue, err := n.path.eval(s)
	if err != nil {
		return err
	}
	if pathValue == nil {
		if _, dynamic := n.path.(subexprParam); dynamic {
			s.column++
		}
		return jvm.Throw("java.lang.IllegalArgumentException", "The uri is required.")
	}
	path := toJavaString(pathValue)
	var partial Template
	external := false
	if stack, ok := s.context.Data(ContextInlinePartials).([]map[string]Template); ok {
		for i := len(stack) - 1; i >= 0 && partial == nil; i-- {
			partial = stack[i][path]
		}
	}
	if partial == nil {
		if m, ok := s.context.Data(ContextPartials).(map[string]Template); ok {
			partial = m[path]
		}
	}
	if partial == nil {
		if path == "@partial-block" {
			return jvm.Throw("java.lang.IllegalArgumentException", s.filename+" does not provide a @partial-block for "+n.raw)
		}
		partial, err = s.engine.Compile(path)
		if err != nil {
			fileNotFound := errors.Is(err, fs.ErrNotExist)
			if ex, ok := err.(*jvm.Exception); ok {
				fileNotFound = ex.Class == "java.io.FileNotFoundException"
			}
			if !fileNotFound {
				return err
			}
			if n.block {
				nested := *s
				if renderErr := renderNodes(&nested, n.body); renderErr != nil {
					s.line, s.column = nested.line, nested.column
					return renderErr
				}
				return nil
			}
			resolved := path
			if s.engine.loader != nil {
				resolved = s.engine.loader.Resolve(path)
			}
			reason := "The partial '" + resolved + "' at '" + resolved + "' could not be found"
			line := s.line
			message := fmt.Sprintf("%s:%d:%d: %s", s.filename, line, s.column, reason)
			return &Error{Filename: s.filename, Line: line, Column: s.column, Reason: reason, Message: message, Cause: err}
		}
		external = true
	}
	if external && !s.engine.infiniteLoops {
		for _, loaded := range s.context.Data(ContextInvocationStack).([]string) {
			if loaded == partial.Filename() {
				stack, _ := s.context.Data(ContextInvocationStack).([]string)
				var trace strings.Builder
				for _, item := range stack[:max(0, len(stack)-1)] {
					trace.WriteString("\nat ")
					trace.WriteString(item)
				}
				reason := fmt.Sprintf("infinite loop detected, partial '%s' was previously loaded%s", partial.Filename(), trace.String())
				message := fmt.Sprintf("%s:%d:%d: %s", s.filename, s.line, s.column, reason)
				return &Error{Filename: s.filename, Line: s.line, Column: s.column, Reason: reason, Message: message}
			}
		}
	}
	model, err := s.context.Resolve("this", s.engine.parentScopeResolution)
	if err != nil {
		return err
	}
	if n.context != nil {
		v, e := n.context.eval(s)
		if e != nil {
			return e
		}
		model = v
	}
	hash := make(map[string]any, len(n.hash))
	for k, p := range n.hash {
		v, e := p.eval(s)
		if e != nil {
			return e
		}
		hash[k] = v
	}
	context := NewPartialContext(s.context, model, hash)
	if n.block {
		m, _ := context.Data(ContextPartials).(map[string]Template)
		old, had := m["@partial-block"]
		m["@partial-block"] = &boundTemplate{Template: &nodeTemplate{engine: s.engine, filename: s.filename, nodes: n.body}, context: context, parentPartial: old}
		defer func() {
			if had {
				m["@partial-block"] = old
			} else {
				delete(m, "@partial-block")
			}
		}()
	}
	var b bytes.Buffer
	var writer io.Writer = &b
	if s.engine.prettyPrint && n.indent != "" {
		writer = &indentedTextWriter{writer: &b, indent: n.indent, lineStart: true}
	}
	if err := partial.ApplyContextWriter(context, writer); err != nil {
		return err
	}
	text := b.String()
	_, err = io.WriteString(s.writer, text)
	return err
}

func toJavaString(value any) string {
	switch v := value.(type) {
	case SafeString:
		return v.Content
	case *SafeString:
		if v == nil {
			return "null"
		}
		return v.Content
	}
	return jvm.ToString(value)
}

func visibleTemplateText(input, start, end string) string {
	var out strings.Builder
	for len(input) > 0 {
		at := strings.Index(input, start)
		if at < 0 {
			out.WriteString(input)
			break
		}
		out.WriteString(input[:at])
		input = input[at:]
		if strings.HasPrefix(input, start+"!--") {
			close := "--" + end
			if n := strings.Index(input, close); n >= 0 {
				input = input[n+len(close):]
				continue
			}
		}
		closeAt := strings.Index(input[len(start):], end)
		if closeAt < 0 {
			out.WriteString(input)
			break
		}
		closeAt += len(start)
		raw := input[:closeAt+len(end)]
		inner := strings.TrimSpace(input[len(start):closeAt])
		input = input[len(raw):]
		if strings.HasPrefix(inner, "!") {
			continue
		}
		if strings.HasPrefix(inner, "=") && strings.HasSuffix(inner, "=") {
			parts := strings.Fields(strings.TrimSpace(strings.Trim(inner, "=")))
			if len(parts) == 2 {
				start, end = parts[0], parts[1]
				continue
			}
		}
		out.WriteString(raw)
	}
	return out.String()
}

func enterLambda(context *Context) (func(), error) {
	const key = "com.github.jknack.handlebars.Lambda#depth"
	depth, _ := context.Data(key).(int)
	if depth >= 64 {
		return nil, jvm.ThrowNoMessage("java.lang.StackOverflowError")
	}
	context.SetData(key, depth+1)
	return func() { context.SetData(key, depth) }, nil
}
