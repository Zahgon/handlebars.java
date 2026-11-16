// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"strings"

	"github.com/jknack/handlebars.java/internal/commons"
)

// EscapingStrategy escapes rendered variable text.
type EscapingStrategy interface{ Escape(input string) string }

type escapingFunc func(string) string

func (f escapingFunc) Escape(input string) string { return f(input) }

var (
	// HTMLENTITY escapes the seven Handlebars HTML characters.
	HTMLENTITY EscapingStrategy = escapingFunc(escapeHTML)
	// HBS4 is the Handlebars 4 escaping strategy.
	HBS4 EscapingStrategy = HTMLENTITY
	// DEF is the default escaping strategy.
	DEF EscapingStrategy = HTMLENTITY
	// HBS3 is the legacy Handlebars 3 strategy.
	HBS3 EscapingStrategy = escapingFunc(escapeHBS3)
	// JS escapes ECMAScript strings.
	JS EscapingStrategy = escapingFunc(func(s string) string { return commons.EscapeEcmaScript(s) })
	// XML escapes XML 1.1 text.
	XML EscapingStrategy = escapingFunc(func(s string) string { return commons.EscapeXml11(s) })
	// CSV escapes RFC-style CSV fields.
	CSV EscapingStrategy = escapingFunc(func(s string) string { return commons.EscapeCsv(s) })
	// NOOP does not escape text.
	NOOP EscapingStrategy = escapingFunc(func(s string) string { return s })
)

func escapeHTML(input string) string {
	var b strings.Builder
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#x27;")
		case '`':
			b.WriteString("&#x60;")
		case '=':
			b.WriteString("&#x3D;")
		default:
			b.WriteByte(input[i])
		}
	}
	return b.String()
}

func escapeHBS3(input string) string {
	var b strings.Builder
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#x27;")
		case '`':
			b.WriteString("&#x60;")
		default:
			b.WriteByte(input[i])
		}
	}
	return b.String()
}

// ChainEscaping applies strategies from left to right.
func ChainEscaping(strategies ...EscapingStrategy) EscapingStrategy {
	copyOf := append([]EscapingStrategy(nil), strategies...)
	return escapingFunc(func(s string) string {
		for _, strategy := range copyOf {
			if strategy != nil {
				s = strategy.Escape(s)
			}
		}
		return s
	})
}

// EscapeExpression applies the default Handlebars escape operation.
func EscapeExpression(input any) string {
	if s, ok := input.(SafeString); ok {
		return s.Content
	}
	return HTMLENTITY.Escape(toJavaString(input))
}
