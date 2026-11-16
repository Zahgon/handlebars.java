// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package commons

import (
	"regexp"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// WordWrap returns WordUtils.wrap with the library's default space pattern.
func WordWrap(s string, width int, newline string, wrapLong bool) string {
	return WordWrapPattern(s, width, newline, wrapLong, " ")
}

// WordWrapPattern returns WordUtils.wrap with an explicit regular expression.
func WordWrapPattern(s string, width int, newline string, wrapLong bool, pattern string) string {
	if newline == "" {
		newline = "\n"
	}
	if width < 1 {
		width = 1
	}
	if pattern == " " {
		s = strings.TrimLeft(s, " ")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		re = regexp.MustCompile(" ")
	}
	offset := 0
	var out strings.Builder
	for jvm.UTF16Len(s)-offset > width {
		rest := Substring(s, offset)
		locs := re.FindAllStringIndex(rest, -1)
		space := -1
		next := -1
		for _, p := range locs {
			idx := jvm.UTF16Len(rest[:p[0]])
			if idx <= width {
				space = idx
			} else {
				next = idx
				break
			}
		}
		if space >= 0 {
			out.WriteString(Substring(rest, 0, space))
			out.WriteString(newline)
			offset += space + 1
		} else if wrapLong {
			out.WriteString(Substring(rest, 0, width))
			out.WriteString(newline)
			offset += width
		} else if next >= 0 {
			out.WriteString(Substring(rest, 0, next))
			out.WriteString(newline)
			offset += next + 1
		} else {
			break
		}
	}
	out.WriteString(Substring(s, offset))
	return out.String()
}

// WordCapitalize returns WordUtils.capitalize with whitespace delimiters.
func WordCapitalize(s string, delimiters ...rune) string { return wordCap(s, false, delimiters) }

// WordCapitalizeFully lowercases then capitalizes words.
func WordCapitalizeFully(s string, delimiters ...rune) string { return wordCap(s, true, delimiters) }
func wordCap(s string, fully bool, delims []rune) string {
	if fully {
		s = jvm.ToLowerCase(s, jvm.Default())
	}
	set := map[rune]bool{}
	if len(delims) > 0 {
		for _, d := range delims {
			set[d] = true
		}
	}
	word := true
	var b strings.Builder
	for _, r := range s {
		delimiter := false
		if len(delims) == 0 {
			delimiter = jvm.IsWhitespace(r)
		} else {
			delimiter = set[r]
		}
		if delimiter {
			word = true
			b.WriteRune(r)
		} else if word {
			b.WriteRune(jvm.CharacterToTitleCase(r))
			word = false
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
