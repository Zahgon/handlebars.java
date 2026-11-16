// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jknack/handlebars.java/internal/jvm"
)

type position struct{ line, renderLine, column, offset int }

type parameter interface {
	eval(*renderState) (any, error)
	text() string
}

type literalParam struct {
	value any
	raw   string
}

func (p literalParam) eval(*renderState) (any, error) { return p.value, nil }
func (p literalParam) text() string                   { return p.raw }

type refParam struct{ path string }

func (p refParam) eval(s *renderState) (any, error) {
	return s.context.Resolve(p.path, s.engine.parentScopeResolution)
}
func (p refParam) text() string { return p.path }

type subexprParam struct{ expr expression }

func (p subexprParam) eval(s *renderState) (any, error) { return p.expr.eval(s, TagSubExpression) }
func (p subexprParam) text() string                     { return "(" + p.expr.text() + ")" }

type expression struct {
	name   string
	params []parameter
	hash   map[string]parameter
}

func (e expression) text() string {
	var b strings.Builder
	b.WriteString(e.name)
	for _, p := range e.params {
		b.WriteByte(' ')
		b.WriteString(p.text())
	}
	for k, p := range e.hash {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(p.text())
	}
	return b.String()
}

type astNode interface {
	render(*renderState) error
	source() string
	collect(map[string]struct{}, map[TagType]bool)
	collectRefs(map[string]struct{})
}

type textNode struct{ value, raw string }

func (n *textNode) render(s *renderState) error {
	if writer, ok := s.writer.(interface{ WriteTemplateText(string) error }); ok {
		return writer.WriteTemplateText(n.value)
	}
	_, err := s.writer.Write([]byte(n.value))
	return err
}
func (n *textNode) source() string                                { return n.raw }
func (n *textNode) collect(map[string]struct{}, map[TagType]bool) {}
func (n *textNode) collectRefs(map[string]struct{})               {}

type varNode struct {
	expr  expression
	type_ TagType
	pos   position
	raw   string
}

func (n *varNode) source() string { return n.raw }
func (n *varNode) collect(out map[string]struct{}, types map[TagType]bool) {
	if types[n.type_] {
		out[n.expr.name] = struct{}{}
	}
}
func (n *varNode) collectRefs(out map[string]struct{}) {
	for _, p := range n.expr.params {
		if r, ok := p.(refParam); ok {
			out[r.path] = struct{}{}
		}
	}
}

type blockNode struct {
	expr        expression
	body        []astNode
	inverse     []astNode
	blockParams []string
	inverted    bool
	decorator   bool
	raw         bool
	pos         position
	closeName   string
	closePos    position
	sourceText  string
}

func (n *blockNode) source() string { return n.sourceText }
func (n *blockNode) collect(out map[string]struct{}, types map[TagType]bool) {
	t := TagSection
	if n.decorator {
		t = TagStartSection
	}
	if types[t] {
		out[n.expr.name] = struct{}{}
	}
	for _, child := range append(append([]astNode{}, n.body...), n.inverse...) {
		child.collect(out, types)
	}
}
func (n *blockNode) collectRefs(out map[string]struct{}) {
	for _, p := range n.expr.params {
		if r, ok := p.(refParam); ok {
			out[r.path] = struct{}{}
		}
	}
	for _, child := range append(append([]astNode{}, n.body...), n.inverse...) {
		child.collectRefs(out)
	}
}

type partialNode struct {
	path      parameter
	context   parameter
	hash      map[string]parameter
	body      []astNode
	block     bool
	indent    string
	pos       position
	raw       string
	closeName string
}

func (n *partialNode) source() string { return n.raw }
func (n *partialNode) collect(map[string]struct{}, map[TagType]bool) {
	for _, child := range n.body {
		child.collect(map[string]struct{}{}, map[TagType]bool{})
	}
}
func (n *partialNode) collectRefs(out map[string]struct{}) {
	if r, ok := n.context.(refParam); ok {
		out[r.path] = struct{}{}
	}
}

type commentNode struct{ raw string }

func (n *commentNode) render(*renderState) error                     { return nil }
func (n *commentNode) source() string                                { return n.raw }
func (n *commentNode) collect(map[string]struct{}, map[TagType]bool) {}
func (n *commentNode) collectRefs(map[string]struct{})               {}

type delimiterNode struct{ raw string }

func (n *delimiterNode) render(*renderState) error                     { return nil }
func (n *delimiterNode) source() string                                { return n.raw }
func (n *delimiterNode) collect(map[string]struct{}, map[TagType]bool) {}
func (n *delimiterNode) collectRefs(map[string]struct{})               {}

type parseStop struct {
	kind string
	expr *expression
	pos  position
	raw  string
}

type handwrittenParser struct {
	engine     *Handlebars
	input      string
	filename   string
	start, end string
	offset     int
	line       int
	renderLine int
	column     int
}

// NewParser creates the grammar-equivalent handwritten parser.
func NewParser(engine *Handlebars, startDelimiter, endDelimiter string) Parser {
	return &parserAdapter{engine: engine, start: startDelimiter, end: endDelimiter}
}

type parserAdapter struct {
	engine     *Handlebars
	start, end string
}

func (p *parserAdapter) Parse(source TemplateSource) (Template, error) {
	input, err := source.Content()
	if err != nil {
		return nil, err
	}
	hp := &handwrittenParser{engine: p.engine, input: input, filename: source.Filename(), start: p.start, end: p.end, line: 1, renderLine: 1}
	nodes, stop, err := hp.sequence("")
	if err != nil {
		return nil, err
	}
	if stop != nil {
		found := hp.start
		column := stop.pos.column
		if stop.kind == "close" {
			found = hp.start + "/"
		}
		return nil, hp.parseError(position{line: stop.pos.line, column: column}, found, "<EOF>", false)
	}
	if err := hp.validateClosures(nodes); err != nil {
		return nil, err
	}
	if p.engine.prettyPrint {
		standaloneNodes(nodes, true)
	}
	return newCompiledTemplate(p.engine, source.Filename(), input, nodes), nil
}

func (p *handwrittenParser) sequence(expected string) ([]astNode, *parseStop, error) {
	var nodes []astNode
	for p.offset < len(p.input) {
		idx := strings.Index(p.input[p.offset:], p.start)
		if idx < 0 {
			p.appendText(&nodes, p.input[p.offset:], p.input[p.offset:])
			p.advance(p.input[p.offset:])
			break
		}
		idx += p.offset
		if idx > p.offset {
			text := p.input[p.offset:idx]
			p.appendText(&nodes, text, text)
			p.advance(text)
		}
		if idx > 0 && p.input[idx-1] == '\\' {
			// The slash is already in the preceding text node. Remove it and consume
			// the complete escaped mustache as literal text.
			if len(nodes) != 0 {
				if t, ok := nodes[len(nodes)-1].(*textNode); ok && strings.HasSuffix(t.value, "\\") {
					t.value = strings.TrimSuffix(t.value, "\\")
				}
			}
			close := strings.Index(p.input[idx+len(p.start):], p.end)
			if close < 0 {
				close = len(p.input) - idx - len(p.start)
			} else {
				close += len(p.end)
			}
			raw := p.input[idx : idx+len(p.start)+close]
			p.appendText(&nodes, raw, "\\"+raw)
			p.advance(raw)
			continue
		}
		pos := p.currentPosition()
		if p.start == "{{" && strings.HasPrefix(p.input[p.offset:], "{{{{") {
			n, err := p.rawBlock(pos)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, n)
			continue
		}
		commentOpen := p.start + "!"
		if strings.HasPrefix(p.input[p.offset:], commentOpen) {
			commentClose := p.end
			if strings.HasPrefix(p.input[p.offset:], p.start+"!--") {
				commentClose = "--" + p.end
			}
			closeAt := strings.Index(p.input[p.offset+len(commentOpen):], commentClose)
			if closeAt < 0 {
				return nil, nil, p.lexerErrorAt(p.offset, p.input[p.offset:])
			}
			closeAt += p.offset + len(commentOpen)
			end := closeAt + len(commentClose)
			if end > len(p.input) {
				end = len(p.input)
			}
			raw := p.input[p.offset:end]
			p.advance(raw)
			nodes = append(nodes, &commentNode{raw: raw})
			continue
		}
		if strings.HasPrefix(p.input[p.offset:], p.start+"=") {
			endDelim := "=" + p.end
			if at := strings.Index(p.input[p.offset+len(p.start)+1:], endDelim); at < 0 {
				return nil, nil, p.incompleteDelimiterError(pos, p.input[p.offset:])
			}
		}
		endAt, endLen, lexFailure := p.findTagEnd()
		if lexFailure != nil {
			return nil, nil, p.lexerErrorAt(lexFailure.offset, p.input[lexFailure.offset:])
		}
		if endAt < 0 {
			remainder := p.input[p.offset:]
			if stray := strings.Index(remainder[len(p.start):], "}"); stray >= 0 {
				return nil, nil, p.lexerErrorAt(p.offset+len(p.start)+stray, "}")
			}
			if incomplete := p.incompleteTagError(pos, remainder, expected); incomplete != nil {
				return nil, nil, incomplete
			}
			if !strings.Contains(remainder[len(p.start):], string(p.end[0])) {
				p.advance(remainder)
				return nil, nil, p.parseError(p.currentPosition(), "<EOF>", p.end, false)
			}
			if p.start == "{{" && strings.HasPrefix(p.input[p.offset:], "{{{") {
				if short := strings.Index(p.input[p.offset+3:], "}}"); short >= 0 {
					return nil, nil, p.parseError(position{line: pos.line, column: pos.column + 3 + short}, "}}", "}}}", false)
				}
			}
			found := p.input[p.offset:]
			column := pos.column
			if strings.HasPrefix(found, p.start) {
				column++
			}
			if len(found) > 0 && strings.HasSuffix(found, "}") && !strings.HasSuffix(found, p.end) {
				found = "}"
				column = pos.column + len(p.start) + 1
			}
			return nil, nil, p.parseError(position{line: pos.line, column: column}, found, "", true)
		}
		raw := p.input[p.offset : endAt+endLen]
		triple := (strings.HasPrefix(raw, p.start+"{") && !strings.HasPrefix(raw, p.start+"{{")) || strings.HasPrefix(raw, p.start+"~{")
		expectedEndLen := len(p.end)
		if triple {
			expectedEndLen++
		}
		inner := p.input[p.offset+len(p.start) : endAt]
		if triple {
			markerLen := 1
			if strings.HasPrefix(raw, p.start+"~{") {
				markerLen = 2
			}
			inner = p.input[p.offset+len(p.start)+markerLen : endAt]
		}
		endText := p.input[endAt : endAt+endLen]
		endMatches := endLen == expectedEndLen || (endLen == expectedEndLen+1 && strings.Contains(endText, "~"))
		if !endMatches && startsWithQID(inner) {
			expectedEnd := p.end
			if expectedEndLen > len(p.end) {
				expectedEnd = "}" + p.end
			}
			return nil, nil, p.parseError(p.positionAt(endAt), p.input[endAt:endAt+endLen], expectedEnd, false)
		}
		leftTrim := strings.HasPrefix(raw, p.start+"~") || strings.HasPrefix(inner, "~")
		rightTrim := strings.HasSuffix(inner, "~") || strings.Contains(endText, "~")
		inner = strings.TrimSpace(strings.Trim(inner, "~"))
		p.advance(raw)
		if leftTrim && len(nodes) > 0 {
			if t, ok := nodes[len(nodes)-1].(*textNode); ok {
				t.value = strings.TrimRightFunc(t.value, jvm.IsWhitespace)
			}
		}
		if rightTrim {
			p.trimLeadingWhitespace()
		}
		if !triple && (strings.HasPrefix(inner, "!--") || strings.HasPrefix(inner, "!")) {
			if strings.HasPrefix(inner, "!--") && !strings.HasSuffix(inner, "--") {
				return nil, nil, p.parseError(position{line: pos.line, column: pos.column + 1}, raw, "", false)
			}
			nodes = append(nodes, &commentNode{raw: raw})
			continue
		}
		if !triple && strings.HasPrefix(inner, "=") && strings.HasSuffix(inner, "=") {
			parts := strings.Fields(strings.TrimSpace(strings.Trim(inner, "=")))
			if len(parts) == 2 {
				p.start, p.end = parts[0], parts[1]
				nodes = append(nodes, &delimiterNode{raw: raw})
				continue
			}
			foundAt := strings.LastIndex(raw, "=")
			return nil, nil, p.parseError(position{line: pos.line, column: pos.column + foundAt}, raw[foundAt:len(raw)-len(p.end)]+p.end, "space", false)
		}
		if !triple && strings.HasPrefix(inner, "/") {
			name := strings.TrimSpace(inner[1:])
			return nodes, &parseStop{kind: "close", expr: &expression{name: name}, pos: pos, raw: raw}, nil
		}
		if !triple && strings.HasPrefix(inner, "^") && expected == "" {
			candidate := strings.TrimSpace(inner[1:])
			preview, lexErr := lexExpression(candidate)
			if lexErr == nil && (candidate == "" || len(preview) == 0 || preview[0].kind != tokenWord || !p.hasClosingTag(preview[0].raw)) {
				return nil, nil, p.parseError(pos, p.start+"^", "<EOF>", false)
			}
		}
		if !triple && (inner == "else" || inner == "^" || strings.HasPrefix(inner, "else ")) {
			var ex *expression
			if strings.HasPrefix(inner, "else ") {
				e, eerr := p.parseExpression(strings.TrimSpace(inner[5:]), pos)
				if eerr != nil {
					return nil, nil, eerr
				}
				ex = &e
			}
			return nodes, &parseStop{kind: "else", expr: ex, pos: pos, raw: raw}, nil
		}
		if !triple && strings.HasPrefix(inner, "#>") {
			n, err := p.partialBlock(strings.TrimSpace(inner[2:]), raw, pos)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, n)
			continue
		}
		if !triple && (strings.HasPrefix(inner, "#") || strings.HasPrefix(inner, "^")) {
			marker := inner[0]
			rest := strings.TrimSpace(inner[1:])
			decorator := strings.HasPrefix(rest, "*")
			if decorator {
				rest = strings.TrimSpace(rest[1:])
			}
			exprText, bp, bpErr := p.splitBlockParams(rest, pos)
			if bpErr != nil {
				return nil, nil, bpErr
			}
			if strings.TrimSpace(exprText) == "" {
				return nil, nil, p.parseError(p.positionAt(endAt), p.input[endAt:endAt+endLen], "id", false)
			}
			expr, err := p.parseExpression(exprText, pos)
			if err != nil {
				return nil, nil, err
			}
			if missing, at := missingSubexpression(expr, exprText, p.engine); missing != "" {
				return nil, nil, p.semanticError(position{line: pos.line, column: pos.column + len(p.start) + at}, "could not find helper: '"+missing+"'", true)
			}
			if !decorator && (len(expr.params) > 0 || len(expr.hash) > 0) && p.engine.Helper(expr.name) == nil && p.engine.Helper(HelperMissing) == nil {
				return nil, nil, jvm.Throw("java.lang.IllegalArgumentException", "could not find helper: '"+expr.name+"'")
			}
			if decorator && p.engine.Decorator(expr.name) == nil {
				return nil, nil, p.semanticError(p.positionAt(p.expressionBase(exprText, pos)), "could not find decorator: '"+expr.name+"'", true)
			}
			if marker == '^' && !p.hasClosingTag(expr.name) {
				return nil, nil, p.parseError(pos, p.start+"^", "<EOF>", false)
			}
			body, stop, err := p.sequence(expr.name)
			if err != nil {
				return nil, nil, err
			}
			if stop == nil {
				return nil, nil, p.parseError(p.blockEOFPosition(), "<EOF>", p.start+"/", false)
			}
			var inverse []astNode
			finalStop := stop
			if stop.kind == "else" {
				inverse, finalStop, err = p.elseChain(expr.name, stop)
				if err != nil {
					return nil, nil, err
				}
			}
			closeName := ""
			closePos := position{}
			if finalStop != nil && finalStop.expr != nil {
				closeName, closePos = finalStop.expr.name, finalStop.pos
			}
			nodes = append(nodes, &blockNode{expr: expr, body: body, inverse: inverse, blockParams: bp, inverted: marker == '^', decorator: decorator, pos: pos, closeName: closeName, closePos: closePos, sourceText: raw})
			continue
		}
		if !triple && strings.HasPrefix(inner, ">") {
			n, err := p.partial(strings.TrimSpace(inner[1:]), raw, pos, false)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, n)
			continue
		}
		type_ := TagVar
		if triple {
			type_ = TagTripleVar
		}
		if !triple && strings.HasPrefix(inner, "&") {
			type_, inner = TagAmpVar, strings.TrimSpace(inner[1:])
		}
		if type_ == TagVar && strings.HasPrefix(inner, "*") {
			type_, inner = TagStarVar, strings.TrimSpace(inner[1:])
		}
		if inner == "" {
			if type_ == TagAmpVar || type_ == TagStarVar || triple {
				return nil, nil, p.parseError(p.positionAt(endAt), p.input[endAt:endAt+endLen], "id", false)
			}
			found := p.start
			if strings.HasPrefix(raw, p.start+"~") {
				found += "~"
			}
			return nil, nil, p.parseError(pos, found, "<EOF>", false)
		}
		if at := strings.Index(inner, " as "); at >= 0 {
			return nil, nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start) + at + 1}, "as", p.end, false)
		}
		if at := outsideQuotedIndex(inner, '\\'); at >= 0 && (at+1 >= len(inner) || inner[at+1] != ']') {
			return nil, nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start) + at}, "\\", "", true)
		}
		if at := strings.Index(inner, ".[]"); at >= 0 {
			return nil, nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start) + at + 1}, "[]", "", true)
		}
		if at := strings.IndexByte(inner, '['); at > 0 && !strings.ContainsAny(inner, " \t") && inner[at-1] != '.' && strings.HasSuffix(inner, "]") {
			name := inner[:at]
			if p.engine.Helper(name) == nil && p.engine.Helper(HelperMissing) == nil {
				return nil, nil, p.semanticError(position{line: pos.line, column: pos.column + len(p.start)}, "could not find helper: '"+name+"'", true)
			}
		}
		if preview, lexErr := lexExpression(inner); lexErr == nil && len(preview) > 0 && preview[0].kind != tokenWord && type_ == TagVar {
			if preview[0].kind == tokenBoolean || preview[0].kind == tokenNumber {
				found := preview[0].raw
				return nil, nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start)}, found, "no viable alternative at input '"+p.start+found+"'", false)
			}
			found := p.start
			if strings.HasPrefix(raw, p.start+"~") {
				found += "~"
			}
			return nil, nil, p.parseError(pos, found, "<EOF>", false)
		}
		expr, err := p.parseExpression(inner, pos)
		if err != nil {
			return nil, nil, err
		}
		if type_ == TagStarVar && p.engine.Decorator(expr.name) == nil {
			return nil, nil, p.semanticError(p.positionAt(p.expressionBase(inner, pos)), "could not find decorator: '"+expr.name+"'", true)
		}
		if expr.name == "true" || expr.name == "false" || isNumberExpressionName(expr.name) {
			return nil, nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start)}, expr.name, "no viable alternative at input '"+p.start+expr.name+"'", false)
		}
		if missing, at := missingSubexpression(expr, inner, p.engine); missing != "" {
			return nil, nil, p.semanticError(position{line: pos.line, column: pos.column + len(p.start) + at}, "could not find helper: '"+missing+"'", true)
		}
		if type_ != TagStarVar && (len(expr.params) > 0 || len(expr.hash) > 0) && p.engine.Helper(expr.name) == nil && p.engine.Helper(HelperMissing) == nil {
			return nil, nil, p.semanticError(position{line: pos.line, column: pos.column + len(p.start)}, "could not find helper: '"+expr.name+"'", true)
		}
		nodes = append(nodes, &varNode{expr: expr, type_: type_, pos: pos, raw: raw})
	}
	if expected != "" {
		return nodes, nil, nil
	}
	return nodes, nil, nil
}

func (p *handwrittenParser) elseChain(expected string, stop *parseStop) ([]astNode, *parseStop, error) {
	if stop.expr == nil {
		body, next, err := p.sequence(expected)
		if err != nil {
			return nil, nil, err
		}
		if next == nil {
			return nil, nil, p.parseError(p.blockEOFPosition(), "<EOF>", p.start+"/", false)
		}
		if next.kind == "else" {
			found := p.start
			if strings.Contains(next.raw, "^") {
				found += "^"
			}
			return nil, nil, p.parseError(next.pos, found, p.start+"/", false)
		}
		return body, next, nil
	}
	body, next, err := p.sequence(expected)
	if err != nil {
		return nil, nil, err
	}
	if next == nil {
		return nil, nil, p.parseError(p.blockEOFPosition(), "<EOF>", p.start+"/", false)
	}
	chain := &blockNode{expr: *stop.expr, body: body, pos: stop.pos}
	if next.kind == "else" {
		chain.inverse, next, err = p.elseChain(expected, next)
		if err != nil {
			return nil, nil, err
		}
	}
	return []astNode{chain}, next, nil
}

func (p *handwrittenParser) validateClosures(nodes []astNode) error {
	for _, node := range nodes {
		block, ok := node.(*blockNode)
		if !ok {
			continue
		}
		if block.closeName != "" && block.closeName != block.expr.name {
			namePos := block.closePos
			namePos.offset += len(p.start) + 1
			namePos.column += utf8.RuneCountInString(p.start) + 1
			return p.semanticError(namePos, "found: '"+block.closeName+"', expected: '"+block.expr.name+"'", true)
		}
		if err := p.validateClosures(block.body); err != nil {
			return err
		}
		if err := p.validateClosures(block.inverse); err != nil {
			return err
		}
	}
	return nil
}

func isNumberExpressionName(name string) bool {
	if name == "" {
		return false
	}
	_, ok := parseNumber(name)
	return ok
}

func (p *handwrittenParser) rawBlock(pos position) (astNode, error) {
	openEnd := strings.Index(p.input[p.offset+4:], "}}}}")
	if openEnd < 0 {
		if p.offset+4 < len(p.input) {
			r, width := utf8.DecodeRuneInString(p.input[p.offset+4:])
			if !isIDStart(r) && r != '[' && r != '.' {
				return nil, p.lexerErrorAt(p.offset+4, p.input[p.offset+4:p.offset+4+width])
			}
		}
		return nil, p.parseError(pos, p.input[p.offset:], "}}}}", false)
	}
	openEnd += p.offset + 4
	rawOpen := p.input[p.offset : openEnd+4]
	inner := strings.TrimSpace(p.input[p.offset+4 : openEnd])
	expr, err := p.parseExpression(inner, pos)
	if err != nil {
		return nil, err
	}
	if (len(expr.params) > 0 || len(expr.hash) > 0) && p.engine.Helper(expr.name) == nil && p.engine.Helper(HelperMissing) == nil {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "could not find helper: '"+expr.name+"'")
	}
	closeTag := "{{{{/" + expr.name + "}}}}"
	bodyStart := openEnd + 4
	closeAt := strings.Index(p.input[bodyStart:], closeTag)
	if closeAt < 0 {
		p.advance(p.input[p.offset:])
		return nil, p.parseError(p.currentPosition(), "<EOF>", "{{{{/", false)
	}
	closeAt += bodyStart
	bodyText := p.input[bodyStart:closeAt]
	whole := p.input[p.offset : closeAt+len(closeTag)]
	p.advance(whole)
	return &blockNode{expr: expr, body: []astNode{&textNode{value: bodyText, raw: bodyText}}, raw: true, pos: pos, sourceText: rawOpen + bodyText + closeTag}, nil
}

func (p *handwrittenParser) partialBlock(text, raw string, pos position) (astNode, error) {
	n, err := p.partial(text, raw, pos, true)
	if err != nil {
		return nil, err
	}
	body, stop, err := p.sequence("partial:" + n.closeName)
	if err != nil {
		return nil, err
	}
	if stop == nil {
		return nil, p.parseError(p.currentPosition(), "<EOF>", p.start+"/", false)
	}
	n.body = body
	return n, nil
}

func (p *handwrittenParser) partial(text, raw string, pos position, block bool) (*partialNode, error) {
	tokens, lexErr := lexExpression(text)
	base := p.expressionBase(text, pos)
	if lexErr != nil {
		return nil, p.lexerErrorAt(base+lexErr.column, lexErr.found)
	}
	if len(tokens) == 0 {
		return nil, p.parseError(position{line: pos.line, column: pos.column + len(raw) - len(p.end)}, p.end, "no viable alternative at input '"+p.end+"'", false)
	}
	if tokens[0].kind == tokenLP && (len(tokens) == 1 || tokens[1].kind != tokenWord) {
		endOffset := pos.offset + len(raw) - len(p.end)
		return nil, p.parseError(p.positionAt(endOffset), p.end, "id", false)
	}
	if tokens[0].kind != tokenWord && tokens[0].kind != tokenPath && tokens[0].kind != tokenString && tokens[0].kind != tokenLP {
		token := tokens[0]
		return nil, p.parseError(p.positionAt(base+token.offset), token.raw, "no viable alternative at input '"+token.raw+"'", false)
	}
	var path parameter
	i := 0
	var err error
	if tokens[0].kind == tokenPath {
		path, i = refParam{path: tokens[0].raw}, 1
	} else {
		path, i, err = parseParameter(tokens, 0)
	}
	if err != nil {
		return nil, p.parseError(p.positionAt(base+tokens[0].offset), tokens[0].raw, "no viable alternative at input '"+tokens[0].raw+"'", false)
	}
	if lit, ok := path.(literalParam); ok {
		name := jvm.ToString(lit.value)
		if strings.HasPrefix(name, "/") {
			return nil, p.parseError(position{line: pos.line, column: pos.column + len(p.start) + 1}, "/", "partial shouldn't start with '/'", false)
		}
	}
	n := &partialNode{path: path, hash: make(map[string]parameter), block: block, pos: pos, raw: raw}
	if r, ok := path.(refParam); ok {
		n.closeName = strings.Trim(r.path, "[]'\"")
		if strings.HasPrefix(n.closeName, "/") {
			return nil, p.semanticError(p.positionAt(base+tokens[0].offset), "found: '/', partial shouldn't start with '/'", true)
		}
		n.path = literalParam{value: n.closeName, raw: r.path}
	}
	if sub, ok := path.(subexprParam); ok && p.engine.Helper(sub.expr.name) == nil && p.engine.Helper(HelperMissing) == nil {
		return nil, p.semanticError(position{line: pos.line, column: pos.column + len(p.start) + 3}, "could not find helper: '"+sub.expr.name+"'", true)
	}
	if i < len(tokens) && !isHashStart(tokens, i) {
		n.context, i, err = parseParameter(tokens, i)
		if err != nil {
			return nil, err
		}
	}
	for i < len(tokens) {
		key, value, next, herr := parseHash(tokens, i)
		if herr != nil {
			endOffset := pos.offset + len(raw) - len(p.end)
			return nil, p.parseError(p.positionAt(endOffset), p.end, "no viable alternative at input '"+p.end+"'", false)
		}
		n.hash[key], i = value, next
	}
	return n, nil
}

type tagLexFailure struct{ offset int }

func (p *handwrittenParser) incompleteDelimiterError(pos position, raw string) error {
	content := raw[len(p.start)+1:]
	i := 0
	for i < len(content) && strings.ContainsRune(" \t\r\n", rune(content[i])) {
		i++
	}
	expected := "delim"
	start := i
	for i < len(content) && !strings.ContainsRune(" \t\r\n", rune(content[i])) {
		i++
	}
	if i > start {
		expected = "space"
	}
	spaceStart := i
	for i < len(content) && strings.ContainsRune(" \t\r\n", rune(content[i])) {
		i++
	}
	if i > spaceStart {
		expected = "delim"
		endStart := i
		for i < len(content) && !strings.ContainsRune(" \t\r\n", rune(content[i])) {
			i++
		}
		if i > endStart {
			expected = "=" + p.end
		}
	}
	return p.parseError(p.positionAt(pos.offset+len(raw)), "<EOF>", expected, false)
}

func startsWithQID(inner string) bool {
	inner = strings.TrimSpace(strings.Trim(inner, "~"))
	for _, marker := range []string{"#>", "#", "^", ">", "/", "&", "*"} {
		if strings.HasPrefix(inner, marker) {
			inner = strings.TrimSpace(inner[len(marker):])
			break
		}
	}
	tokens, err := lexExpression(inner)
	return err == nil && len(tokens) > 0 && tokens[0].kind == tokenWord
}

func (p *handwrittenParser) hasClosingTag(name string) bool {
	needle := p.start + "/"
	search := p.input[p.offset:]
	for {
		at := strings.Index(search, needle)
		if at < 0 {
			return false
		}
		contentStart := at + len(needle)
		end := strings.Index(search[contentStart:], p.end)
		if end < 0 {
			return false
		}
		if strings.TrimSpace(search[contentStart:contentStart+end]) == name {
			return true
		}
		search = search[contentStart+end+len(p.end):]
	}
}

func (p *handwrittenParser) incompleteTagError(pos position, raw, enclosingBlock string) error {
	prefixLen := len(p.start)
	content := raw[prefixLen:]
	foundStart := p.start
	if strings.HasPrefix(content, "~") {
		foundStart += "~"
		content = content[1:]
		prefixLen++
	}
	kind := "var"
	close := p.end
	switch {
	case strings.HasPrefix(content, "{"):
		kind, close, content, prefixLen = "triple", "}"+p.end, content[1:], prefixLen+1
	case strings.HasPrefix(content, "&"):
		kind, content, prefixLen = "amp", content[1:], prefixLen+1
	case strings.HasPrefix(content, "#>"):
		kind, content, prefixLen = "partialBlock", content[2:], prefixLen+2
	case strings.HasPrefix(content, "#"):
		kind, content, prefixLen = "block", content[1:], prefixLen+1
	case strings.HasPrefix(content, "^"):
		return p.parseError(pos, foundStart+"^", "<EOF>", false)
	case strings.HasPrefix(content, ">"):
		kind, content, prefixLen = "partial", content[1:], prefixLen+1
	case strings.HasPrefix(content, "/"):
		kind, content, prefixLen = "close", content[1:], prefixLen+1
	}
	base := pos.offset + prefixLen
	tokens, lexErr := lexExpression(content)
	if lexErr != nil {
		return p.lexerErrorAt(base+lexErr.column, lexErr.found)
	}
	eof := p.positionAt(len(p.input))
	if len(tokens) == 0 {
		switch kind {
		case "var":
			if strings.HasPrefix(enclosingBlock, "partial:") {
				return p.parseError(pos, foundStart, p.start+"/", false)
			}
			if enclosingBlock != "" {
				return p.parseError(eof, "<EOF>", "no viable alternative at input '"+foundStart+"'", false)
			}
			return p.parseError(pos, foundStart, "<EOF>", false)
		case "partial", "partialBlock":
			return p.parseError(eof, "<EOF>", "no viable alternative at input '<EOF>'", false)
		default:
			return p.parseError(eof, "<EOF>", "id", false)
		}
	}
	if tokens[0].kind != tokenWord {
		if kind == "var" {
			return p.parseError(pos, foundStart, "<EOF>", false)
		}
		return p.parseError(p.positionAt(base+tokens[0].offset), tokens[0].raw, "id", false)
	}
	if kind == "block" {
		for i, token := range tokens {
			if token.kind != tokenAS {
				continue
			}
			if i+1 >= len(tokens) {
				return p.parseError(eof, "<EOF>", "|", false)
			}
			if tokens[i+1].kind != tokenPipe {
				return p.parseError(p.positionAt(base+tokens[i+1].offset), tokens[i+1].raw, "|", false)
			}
			if i+2 >= len(tokens) {
				return p.parseError(eof, "<EOF>", "id", false)
			}
			if tokens[len(tokens)-1].kind == tokenWord {
				return p.parseError(eof, "<EOF>", "|", false)
			}
		}
	}
	for i := 1; i < len(tokens); i++ {
		if tokens[i].kind == tokenLP {
			if i+1 == len(tokens) {
				return p.parseError(eof, "<EOF>", "id", false)
			}
			depth := 1
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].kind == tokenLP {
					depth++
				} else if tokens[j].kind == tokenRP {
					depth--
				}
			}
			if depth > 0 {
				return p.parseError(eof, "<EOF>", ")", false)
			}
		}
		if tokens[i].kind == tokenEq && i+1 == len(tokens) {
			return p.parseError(eof, "<EOF>", "no viable alternative at input '<EOF>'", false)
		}
	}
	return p.parseError(eof, "<EOF>", close, false)
}

// findTagEnd follows the VAR lexer mode: quoted strings and bracket escapes
// consume delimiter-looking text, while the longest END_RAW/END_T/END token
// wins at a delimiter boundary.
func (p *handwrittenParser) findTagEnd() (int, int, *tagLexFailure) {
	i := p.offset + len(p.start)
	if strings.HasPrefix(p.input[p.offset:], p.start+"{") {
		i++
	}
	quote := byte(0)
	quoteAt := 0
	bracketAt := -1
	for i < len(p.input) {
		if quote != 0 {
			if p.input[i] == '\\' && i+1 < len(p.input) && p.input[i+1] == quote {
				i += 2
				continue
			}
			if p.input[i] == quote {
				quote = 0
			}
			i++
			continue
		}
		if bracketAt >= 0 {
			if p.input[i] == '\\' && i+1 < len(p.input) && p.input[i+1] == ']' {
				i += 2
				continue
			}
			if p.input[i] == ']' {
				bracketAt = -1
			}
			i++
			continue
		}
		if (p.input[i] == '\'' || p.input[i] == '"') && (i == 0 || p.input[i-1] != '\\') &&
			(p.input[i] == '"' || closingQuote(p.input, i, p.input[i]) >= 0) {
			quote, quoteAt = p.input[i], i
			i++
			continue
		}
		if p.input[i] == '[' {
			bracketAt = i
			i++
			continue
		}
		if strings.HasPrefix(p.input[i:], "}}~"+p.end) {
			return i, len(p.end) + 3, nil
		}
		if strings.HasPrefix(p.input[i:], "}}"+p.end) {
			return i, len(p.end) + 2, nil
		}
		if strings.HasPrefix(p.input[i:], "}~"+p.end) {
			return i, len(p.end) + 2, nil
		}
		if strings.HasPrefix(p.input[i:], "}"+p.end) {
			return i, len(p.end) + 1, nil
		}
		if strings.HasPrefix(p.input[i:], "~"+p.end) {
			return i, len(p.end) + 1, nil
		}
		if strings.HasPrefix(p.input[i:], p.end) {
			return i, len(p.end), nil
		}
		_, width := utf8.DecodeRuneInString(p.input[i:])
		i += width
	}
	if quote != 0 {
		return -1, 0, &tagLexFailure{offset: quoteAt}
	}
	if bracketAt >= 0 {
		return -1, 0, &tagLexFailure{offset: bracketAt}
	}
	return -1, 0, nil
}

func (p *handwrittenParser) parseExpression(text string, pos position) (expression, error) {
	base := p.expressionBase(text, pos)
	tokens, err := lexExpression(text)
	if err != nil {
		return expression{}, p.lexerErrorAt(base+err.column, err.found)
	}
	if len(tokens) == 0 {
		return expression{}, nil
	}
	if tokens[0].kind != tokenWord {
		return expression{}, p.parseError(p.positionAt(base+tokens[0].offset), tokens[0].raw, "id", false)
	}
	e := expression{name: tokens[0].raw, hash: make(map[string]parameter)}
	sawHash := false
	for i := 1; i < len(tokens); {
		if isHashStart(tokens, i) {
			sawHash = true
			end := p.expressionEndOffset(base + len(text))
			if i+2 >= len(tokens) {
				return expression{}, p.parseError(p.positionAt(end), p.end, "no viable alternative at input '"+p.end+"'", false)
			}
			valueToken := tokens[i+2]
			if valueToken.kind != tokenString && valueToken.kind != tokenNumber && valueToken.kind != tokenBoolean && valueToken.kind != tokenWord && valueToken.kind != tokenLP {
				return expression{}, p.parseError(p.positionAt(base+valueToken.offset), valueToken.raw, "no viable alternative at input '"+valueToken.raw+"'", false)
			}
			if valueToken.kind == tokenLP && i+3 >= len(tokens) {
				return expression{}, p.parseError(p.positionAt(end), p.end, "id", false)
			}
			key, value, next, herr := parseHash(tokens, i)
			if herr != nil {
				return expression{}, p.parseError(p.positionAt(end), p.end, "no viable alternative at input '"+p.end+"'", false)
			}
			e.hash[key], i = value, next
			continue
		}
		if sawHash {
			return expression{}, p.parseError(p.positionAt(p.expressionEndOffset(base+len(text))), p.end, "=", false)
		}
		if tokens[i].kind == tokenLP {
			if i+1 == len(tokens) {
				end := p.expressionEndOffset(base + len(text))
				return expression{}, p.parseError(p.positionAt(end), p.end, "id", false)
			}
			if tokens[i+1].kind != tokenWord {
				return expression{}, p.parseError(p.positionAt(base+tokens[i+1].offset), tokens[i+1].raw, "id", false)
			}
		}
		param, next, perr := parseParameter(tokens, i)
		if perr != nil {
			if strings.Contains(perr.Error(), "missing ')'") {
				return expression{}, p.parseError(p.positionAt(p.expressionEndOffset(base+len(text))), p.end, ")", false)
			}
			expected := p.end
			if tokens[i].kind == tokenLP {
				expected = "id"
			}
			return expression{}, p.parseError(p.positionAt(base+tokens[i].offset), tokens[i].raw, expected, false)
		}
		e.params, i = append(e.params, param), next
	}
	return e, nil
}

func (p *handwrittenParser) splitBlockParams(text string, pos position) (string, []string, error) {
	tokens, lexErr := lexExpression(text)
	if lexErr != nil {
		base := p.expressionBase(text, pos)
		return "", nil, p.lexerErrorAt(base+lexErr.column, lexErr.found)
	}
	as := -1
	for i, token := range tokens {
		if token.kind == tokenAS {
			as = i
			break
		}
	}
	if as <= 0 {
		return text, nil, nil
	}
	base := p.expressionBase(text, pos)
	exprText := strings.TrimSpace(text[:tokens[as].offset])
	end := p.expressionEndOffset(base + len(text))
	if as+1 >= len(tokens) {
		return "", nil, p.parseError(p.positionAt(end), p.end, "|", false)
	}
	if tokens[as+1].kind != tokenPipe {
		token := tokens[as+1]
		return "", nil, p.parseError(p.positionAt(base+token.offset), token.raw, "|", false)
	}
	i := as + 2
	if i >= len(tokens) {
		return "", nil, p.parseError(p.positionAt(end), p.end, "id", false)
	}
	if tokens[i].kind != tokenWord {
		return "", nil, p.parseError(p.positionAt(base+tokens[i].offset), tokens[i].raw, "id", false)
	}
	var names []string
	for i < len(tokens) && tokens[i].kind == tokenWord {
		names = append(names, tokens[i].raw)
		i++
	}
	if i >= len(tokens) {
		return "", nil, p.parseError(p.positionAt(end), p.end, "|", false)
	}
	if tokens[i].kind != tokenPipe {
		return "", nil, p.parseError(p.positionAt(base+tokens[i].offset), tokens[i].raw, "|", false)
	}
	if i+1 < len(tokens) {
		token := tokens[i+1]
		return "", nil, p.parseError(p.positionAt(base+token.offset), token.raw, p.end, false)
	}
	return exprText, names, nil
}

func (p *handwrittenParser) expressionBase(text string, pos position) int {
	limit := p.offset
	if limit <= pos.offset {
		limit = len(p.input)
	}
	if limit > len(p.input) {
		limit = len(p.input)
	}
	searchStart := pos.offset
	if strings.HasPrefix(p.input[searchStart:limit], p.start) {
		searchStart += len(p.start)
		if searchStart < limit && p.input[searchStart] == '~' {
			searchStart++
		}
		for _, marker := range []string{"#>", "{{", "{", "&", "#", "^", "=", ">", "/"} {
			if strings.HasPrefix(p.input[searchStart:limit], marker) {
				searchStart += len(marker)
				break
			}
		}
	}
	if at := strings.Index(p.input[searchStart:limit], text); at >= 0 {
		return searchStart + at
	}
	return pos.offset + len(p.start)
}

func (p *handwrittenParser) expressionEndOffset(after int) int {
	if after < 0 {
		after = 0
	}
	if after > len(p.input) {
		return len(p.input)
	}
	if at := strings.Index(p.input[after:], p.end); at >= 0 {
		return after + at
	}
	return len(p.input)
}

type expressionTokenKind int

const (
	tokenWord expressionTokenKind = iota
	tokenString
	tokenNumber
	tokenBoolean
	tokenPath
	tokenLP
	tokenRP
	tokenEq
	tokenAS
	tokenPipe
	tokenDecorator
	tokenElse
)

type expressionToken struct {
	kind       expressionTokenKind
	raw, value string
	offset     int
}
type lexError struct {
	found  string
	column int
}

func (e *lexError) Error() string { return e.found }

func lexExpression(input string) ([]expressionToken, *lexError) {
	var out []expressionToken
	for i := 0; i < len(input); {
		for i < len(input) {
			r, width := utf8.DecodeRuneInString(input[i:])
			if !strings.ContainsRune(" \t\r\n", r) {
				break
			}
			i += width
		}
		if i >= len(input) {
			break
		}
		start := i
		switch input[i] {
		case '(', ')':
			kind := tokenLP
			if input[i] == ')' {
				kind = tokenRP
			}
			out = append(out, expressionToken{kind: kind, raw: input[i : i+1], offset: i})
			i++
		case '=':
			out = append(out, expressionToken{kind: tokenEq, raw: "=", offset: i})
			i++
		case '|':
			out = append(out, expressionToken{kind: tokenPipe, raw: "|", offset: i})
			i++
		case '*':
			out = append(out, expressionToken{kind: tokenDecorator, raw: "*", offset: i})
			i++
		case '\'', '"':
			quote := input[i]
			if quote == '\'' && closingQuote(input, i, quote) < 0 {
				end := expressionAtomEnd(input, i)
				word := input[start:end]
				prefix, kind := classifyExpressionAtom(word)
				if prefix == "" {
					return nil, &lexError{found: "'", column: start}
				}
				out = append(out, expressionToken{kind: kind, raw: prefix, offset: start})
				i = start + len(prefix)
				continue
			}
			i++
			var b strings.Builder
			for i < len(input) && input[i] != quote {
				if input[i] == '\\' && i+1 < len(input) && input[i+1] == quote {
					b.WriteByte(quote)
					i += 2
					continue
				}
				b.WriteByte(input[i])
				i++
			}
			if i >= len(input) {
				return nil, &lexError{found: input[start:], column: start}
			}
			i++
			out = append(out, expressionToken{kind: tokenString, raw: input[start:i], value: b.String(), offset: start})
		default:
			end := expressionAtomEnd(input, i)
			word := input[start:end]
			prefix, kind := classifyExpressionAtom(word)
			if prefix == "" {
				if strings.HasPrefix(word, "[]") {
					return nil, &lexError{found: "[]", column: start}
				}
				_, width := utf8.DecodeRuneInString(input[start:])
				return nil, &lexError{found: input[start : start+width], column: start}
			}
			i = start + len(prefix)
			out = append(out, expressionToken{kind: kind, raw: prefix, offset: start})
		}
	}
	return out, nil
}

func expressionAtomEnd(input string, start int) int {
	brackets := 0
	for i := start; i < len(input); {
		r, width := utf8.DecodeRuneInString(input[i:])
		if brackets == 0 && (strings.ContainsRune(" \t\r\n()=|*\"", r)) {
			return i
		}
		if r == '[' {
			brackets++
		} else if r == ']' && brackets > 0 {
			brackets--
		}
		i += width
	}
	return len(input)
}

func closingQuote(input string, start int, quote byte) int {
	for i := start + 1; i < len(input); i++ {
		if input[i] == '\\' && i+1 < len(input) && input[i+1] == quote {
			i++
			continue
		}
		if input[i] == quote {
			return i
		}
	}
	return -1
}

func classifyExpressionAtom(atom string) (string, expressionTokenKind) {
	if atom == "as" {
		return atom, tokenAS
	}
	if atom == "true" || atom == "false" {
		return atom, tokenBoolean
	}
	if atom == "else" {
		return atom, tokenElse
	}
	if isDottedBracketQID(atom) {
		return atom, tokenWord
	}
	for end := len(atom); end > 0; {
		candidate := atom[:end]
		if isNumberToken(candidate) {
			return candidate, tokenNumber
		}
		if isQID(candidate) {
			return candidate, tokenWord
		}
		if isPathToken(candidate) {
			return candidate, tokenPath
		}
		_, width := utf8.DecodeLastRuneInString(atom[:end])
		end -= width
	}
	return "", tokenPath
}

func isDottedBracketQID(text string) bool {
	if !strings.Contains(text, ".[") {
		return false
	}
	var normalized strings.Builder
	for i := 0; i < len(text); {
		if i+1 < len(text) && text[i] == '.' && text[i+1] == '[' {
			end := escapedBracketEnd(text, i+1)
			if end < 0 {
				return false
			}
			normalized.WriteString(".x")
			i = end
			continue
		}
		if text[i] == '[' {
			return false
		}
		normalized.WriteByte(text[i])
		i++
	}
	return isQID(normalized.String())
}

func isNumberToken(text string) bool {
	if text == "" {
		return false
	}
	i := 0
	if text[0] == '-' {
		i++
	}
	digits := 0
	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		i++
		digits++
	}
	if i < len(text) && text[i] == '.' {
		i++
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
			digits++
		}
	}
	return digits > 0 && i == len(text)
}

func isPathToken(text string) bool {
	if len(text) >= 2 && text[0] == '[' && text[len(text)-1] == ']' {
		text = text[1 : len(text)-1]
	}
	if text == "" {
		return false
	}
	for _, r := range text {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("_$'/.:-", r)) {
			return false
		}
	}
	return true
}

func isQID(text string) bool {
	if text == "." || text == ".." {
		return true
	}
	if strings.HasPrefix(text, "../") {
		return isQID(text[3:])
	}
	if strings.HasPrefix(text, "./") {
		return isQID(text[2:])
	}
	if text == "" {
		return false
	}
	if text[0] == '[' {
		close := escapedBracketEnd(text, 0)
		if close < 0 {
			return false
		}
		if close == len(text) {
			return true
		}
		if close >= len(text) || !strings.ContainsRune("./-", rune(text[close])) {
			return false
		}
		return isQID(text[close+1:])
	}
	runes := []rune(text)
	i := 0
	if runes[0] >= '0' && runes[0] <= '9' {
		for i < len(runes) && runes[i] >= '0' && runes[i] <= '9' {
			i++
		}
		if i == len(runes) || !isIDStart(runes[i]) {
			return false
		}
		i++
	} else if isIDStart(runes[0]) {
		i++
	} else {
		return false
	}
	for i < len(runes) {
		r := runes[i]
		if isIDStart(r) || (r >= '0' && r <= '9') || r == '.' || r == '/' || r == '-' {
			i++
			continue
		}
		return false
	}
	return true
}

func isIDStart(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '\u0410' && r <= '\u044f') ||
		strings.ContainsRune("_$@:", r) || (r >= 1 && r <= 8) || r == 11 || r == 12 ||
		(r >= 14 && r <= 30) || (r >= 0xC0 && r <= 0xFF)
}

func escapedBracketEnd(text string, start int) int {
	if start >= len(text) || text[start] != '[' {
		return -1
	}
	hasContent := false
	for i := start + 1; i < len(text); i++ {
		if text[i] == '\\' && i+1 < len(text) && text[i+1] == ']' {
			hasContent = true
			i++
			continue
		}
		if text[i] == ']' {
			if hasContent {
				return i + 1
			}
			return -1
		}
		hasContent = true
	}
	return -1
}

func parseParameter(tokens []expressionToken, i int) (parameter, int, error) {
	if i >= len(tokens) {
		return nil, i, fmt.Errorf("missing parameter")
	}
	t := tokens[i]
	switch t.kind {
	case tokenString:
		return literalParam{value: t.value, raw: t.raw}, i + 1, nil
	case tokenNumber:
		value, ok := parseNumber(t.raw)
		if !ok {
			return nil, i, fmt.Errorf("invalid number")
		}
		return value, i + 1, nil
	case tokenBoolean:
		return literalParam{value: t.raw == "true", raw: t.raw}, i + 1, nil
	case tokenLP:
		depth, end := 1, i+1
		for ; end < len(tokens); end++ {
			if tokens[end].kind == tokenLP {
				depth++
			}
			if tokens[end].kind == tokenRP {
				depth--
				if depth == 0 {
					break
				}
			}
		}
		if depth != 0 {
			return nil, i, fmt.Errorf("missing ')'")
		}
		expr, err := expressionFromTokens(tokens[i+1 : end])
		if err != nil {
			return nil, i, err
		}
		return subexprParam{expr: expr}, end + 1, nil
	case tokenWord:
		return refParam{path: t.raw}, i + 1, nil
	default:
		return nil, i, fmt.Errorf("invalid parameter")
	}
}

func expressionFromTokens(tokens []expressionToken) (expression, error) {
	if len(tokens) == 0 || tokens[0].kind != tokenWord {
		return expression{}, fmt.Errorf("invalid expression")
	}
	e := expression{name: tokens[0].raw, hash: make(map[string]parameter)}
	for i := 1; i < len(tokens); {
		if isHashStart(tokens, i) {
			k, v, n, err := parseHash(tokens, i)
			if err != nil {
				return e, err
			}
			e.hash[k] = v
			i = n
		} else {
			v, n, err := parseParameter(tokens, i)
			if err != nil {
				return e, err
			}
			e.params = append(e.params, v)
			i = n
		}
	}
	return e, nil
}

func missingSubexpression(expr expression, source string, engine *Handlebars) (string, int) {
	for i := len(expr.params) - 1; i >= 0; i-- {
		if sub, ok := expr.params[i].(subexprParam); ok {
			if name, at := missingSubexpression(sub.expr, source, engine); name != "" {
				return name, at
			}
			if engine.Helper(sub.expr.name) == nil && engine.Helper(HelperMissing) == nil {
				at := strings.LastIndex(source, "("+sub.expr.name)
				if at < 0 {
					at = 0
				} else {
					at++
				}
				return sub.expr.name, at
			}
		}
	}
	return "", 0
}

func isHashStart(tokens []expressionToken, i int) bool {
	return i+1 < len(tokens) && tokens[i].kind == tokenWord && tokens[i+1].kind == tokenEq
}

func parseHash(tokens []expressionToken, i int) (string, parameter, int, error) {
	if !isHashStart(tokens, i) || i+2 >= len(tokens) {
		return "", nil, i, fmt.Errorf("invalid hash")
	}
	v, next, err := parseParameter(tokens, i+2)
	return tokens[i].raw, v, next, err
}

func parseNumber(text string) (parameter, bool) {
	if text == "" {
		return nil, false
	}
	for i, r := range text {
		if !(r >= '0' && r <= '9') && r != '.' && !(r == '-' && i == 0) {
			return nil, false
		}
	}
	if v, err := jvm.ParseInt(text); err == nil {
		return literalParam{value: v, raw: text}, true
	}
	if v, err := jvm.ParseDouble(text); err == nil {
		return literalParam{value: v, raw: text}, true
	}
	return nil, false
}

// outsideQuotedIndex returns the byte index of target only when it is not
// inside a Handlebars single- or double-quoted string literal.
func outsideQuotedIndex(text string, target byte) int {
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if c == '\\' && i+1 < len(text) {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == target {
			return i
		}
	}
	return -1
}

func (p *handwrittenParser) appendText(nodes *[]astNode, value, raw string) {
	if value == "" {
		return
	}
	if len(*nodes) > 0 {
		if previous, ok := (*nodes)[len(*nodes)-1].(*textNode); ok {
			previous.value += value
			previous.raw += raw
			return
		}
	}
	*nodes = append(*nodes, &textNode{value: value, raw: raw})
}

func (p *handwrittenParser) trimLeadingWhitespace() {
	start := p.offset
	for p.offset < len(p.input) {
		r, n := utf8.DecodeRuneInString(p.input[p.offset:])
		if !jvm.IsWhitespace(r) {
			break
		}
		p.advance(p.input[p.offset : p.offset+n])
	}
	_ = start
}

func (p *handwrittenParser) currentPosition() position {
	return position{line: p.line, renderLine: p.renderLine, column: p.column, offset: p.offset}
}

func (p *handwrittenParser) positionAt(offset int) position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(p.input) {
		offset = len(p.input)
	}
	pos := position{line: 1, renderLine: 1, offset: offset}
	for i := 0; i < offset; {
		r, width := utf8.DecodeRuneInString(p.input[i:offset])
		i += width
		switch r {
		case '\r':
			pos.line++
			pos.renderLine++
			pos.column = 0
		case '\n':
			precededByCR := i >= 2 && p.input[i-2] == '\r'
			if !precededByCR {
				pos.line++
				pos.renderLine++
			}
			if precededByCR {
				pos.column = 1
			} else {
				pos.column = 0
			}
		default:
			pos.column++
		}
	}
	return pos
}

func (p *handwrittenParser) blockEOFPosition() position {
	pos := p.currentPosition()
	if strings.HasSuffix(p.input, "\r\n") {
		pos.line++
		pos.renderLine++
	}
	return pos
}

func (p *handwrittenParser) lexerErrorAt(offset int, found string) error {
	return p.parseError(p.positionAt(offset), antlrErrorDisplay(found), "", true)
}

func antlrErrorDisplay(text string) string {
	var out strings.Builder
	for _, r := range text {
		switch r {
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		case '\t':
			out.WriteString("\\t")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func (p *handwrittenParser) advance(s string) {
	p.offset += len(s)
	pos := p.positionAt(p.offset)
	p.line, p.renderLine, p.column = pos.line, pos.renderLine, pos.column
}

func (p *handwrittenParser) parseError(pos position, found, expected string, blank bool) error {
	if pos.line < 1 {
		pos.line = 1
	}
	if pos.column < 1 {
		pos.column = 1
	}
	reason := "found: '" + found + "'"
	if expected != "" {
		reason += ", expected: '" + expected + "'"
	}
	lines := javaSplitLines(p.input)
	evidence := lines[min(max(pos.line-1, 0), len(lines)-1)]
	message := fmt.Sprintf("%s:%d:%d: %s\n%s\n%s^", p.filename, pos.line, pos.column, reason, evidence, strings.Repeat(" ", max(0, pos.column)))
	if blank {
		message += "\n"
	}
	return &Error{Filename: p.filename, Line: pos.line, Column: pos.column, Reason: reason, Evidence: evidence, Message: message}
}

func (p *handwrittenParser) semanticError(pos position, reason string, blank bool) error {
	if pos.line < 1 {
		pos.line = 1
	}
	if pos.column < 1 {
		pos.column = 1
	}
	lines := javaSplitLines(p.input)
	evidence := lines[min(max(pos.line-1, 0), len(lines)-1)]
	message := fmt.Sprintf("%s:%d:%d: %s\n%s\n%s^", p.filename, pos.line, pos.column, reason, evidence, strings.Repeat(" ", max(0, pos.column)))
	if blank {
		message += "\n"
	}
	return &Error{Filename: p.filename, Line: pos.line, Column: pos.column, Reason: reason, Evidence: evidence, Message: message}
}

func javaSplitLines(input string) []string {
	lines := strings.Split(input, "\n")
	for len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func standaloneNodes(nodes []astNode, allowStart bool) {
	for i, node := range nodes {
		block, isBlock := node.(*blockNode)
		partial, isPartial := node.(*partialNode)
		_, isComment := node.(*commentNode)
		_, isDelimiter := node.(*delimiterNode)
		if isBlock {
			standaloneNodes(block.body, false)
			standaloneNodes(block.inverse, false)
			trimBlockStandalone(nodes, i, block, allowStart)
			continue
		}
		if !(isPartial || isComment || isDelimiter) {
			continue
		}
		left, indent, leftOK := standaloneLeft(nodes, i, allowStart)
		right, rightOK := standaloneRight(nodes, i)
		if !leftOK || !rightOK {
			continue
		}
		if isPartial {
			partial.indent = indent
		}
		if left != nil {
			left.value = strings.TrimSuffix(left.value, indent)
		}
		if right != nil {
			right.value = removeLeadingNewline(right.value)
		}
	}
}

func trimBlockStandalone(nodes []astNode, index int, block *blockNode, allowStart bool) {
	// Opening tag.
	left, indent, leftOK := standaloneLeft(nodes, index, allowStart)
	if first := firstText(block.body); leftOK && first != nil && hasLeadingNewline(first.value) {
		if left != nil {
			left.value = strings.TrimSuffix(left.value, indent)
		}
		first.value = removeLeadingNewline(first.value)
	}
	// Else tag.
	if len(block.inverse) > 0 {
		last := lastText(block.body)
		first := firstText(block.inverse)
		if last != nil && first != nil {
			if ws, ok := trailingIndent(last.value); ok && hasLeadingNewline(first.value) {
				last.value = strings.TrimSuffix(last.value, ws)
				first.value = removeLeadingNewline(first.value)
			}
		}
	}
	// Closing tag.
	branch := block.body
	if len(block.inverse) > 0 {
		branch = block.inverse
	}
	last := lastText(branch)
	right, rightOK := standaloneRight(nodes, index)
	if last != nil && rightOK {
		if ws, ok := trailingIndent(last.value); ok {
			last.value = strings.TrimSuffix(last.value, ws)
			if right != nil {
				right.value = removeLeadingNewline(right.value)
			}
		}
	}
}

func standaloneLeft(nodes []astNode, index int, allowStart bool) (*textNode, string, bool) {
	if index == 0 {
		return nil, "", true
	}
	left, ok := nodes[index-1].(*textNode)
	if !ok {
		return nil, "", false
	}
	indent, ok := trailingIndent(left.value)
	if !ok {
		return nil, "", false
	}
	if !strings.ContainsAny(left.value, "\r\n") && index > 0 {
		if allowStart && index == 1 {
			return left, indent, true
		}
		// A previous standalone block can consume the newline while leaving
		// this tag's indentation in its adjacent text node.
		if index < 2 {
			return nil, "", false
		}
		switch nodes[index-2].(type) {
		case *blockNode, *partialNode, *commentNode, *delimiterNode:
		default:
			return nil, "", false
		}
	}
	return left, indent, true
}

func standaloneRight(nodes []astNode, index int) (*textNode, bool) {
	if index+1 == len(nodes) {
		return nil, true
	}
	right, ok := nodes[index+1].(*textNode)
	return right, ok && hasLeadingNewline(right.value)
}

func firstText(nodes []astNode) *textNode {
	if len(nodes) == 0 {
		return nil
	}
	t, _ := nodes[0].(*textNode)
	return t
}

func lastText(nodes []astNode) *textNode {
	if len(nodes) == 0 {
		return nil
	}
	t, _ := nodes[len(nodes)-1].(*textNode)
	return t
}
func trailingIndent(s string) (string, bool) {
	i := strings.LastIndexAny(s, "\r\n")
	tail := s[i+1:]
	return tail, strings.Trim(tail, " \t") == ""
}
func hasLeadingNewline(s string) bool {
	return strings.HasPrefix(s, "\n") || strings.HasPrefix(s, "\r\n") || strings.HasPrefix(s, "\r")
}
func removeLeadingNewline(s string) string {
	if strings.HasPrefix(s, "\r\n") {
		return s[2:]
	}
	if len(s) > 0 && (s[0] == '\n' || s[0] == '\r') {
		return s[1:]
	}
	return s
}

// ParseIntegerLiteral exposes the TemplateBuilder integer-then-double rule.
func ParseIntegerLiteral(text string) (any, error) {
	if v, err := jvm.ParseInt(text); err == nil {
		return v, nil
	}
	return jvm.ParseDouble(text)
}

var _ = strconv.IntSize
