// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

// Package commons implements the Apache Commons Lang and Commons Text
// operations used by handlebars.java.
package commons

import (
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// Center returns StringUtils.center.
func Center(s string, size int, pad string) string {
	n := jvm.UTF16Len(s)
	p := size - n
	if p <= 0 {
		return s
	}
	if pad == "" {
		pad = " "
	}
	left := p / 2
	return padUnits(pad, left) + s + padUnits(pad, p-left)
}

// LeftPad returns StringUtils.leftPad.
func LeftPad(s string, size int, pad string) string {
	n := size - jvm.UTF16Len(s)
	if n <= 0 {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return padUnits(pad, n) + s
}

// RightPad returns StringUtils.rightPad.
func RightPad(s string, size int, pad string) string {
	n := size - jvm.UTF16Len(s)
	if n <= 0 {
		return s
	}
	if pad == "" {
		pad = " "
	}
	return s + padUnits(pad, n)
}

func padUnits(p string, n int) string {
	if n <= 0 {
		return ""
	}
	u := utf16Units(p)
	out := make([]uint16, n)
	for i := range out {
		out[i] = u[i%len(u)]
	}
	return unitsString(out)
}

// Abbreviate returns StringUtils.abbreviate(String,int).
func Abbreviate(s string, maxWidth int) (string, error) {
	if s == "" {
		return s, nil
	}
	if maxWidth < 4 {
		return "", jvm.Throw("java.lang.IllegalArgumentException", "Minimum abbreviation width is 4")
	}
	if jvm.UTF16Len(s) <= maxWidth {
		return s, nil
	}
	x, _ := jvm.Substring(s, 0, maxWidth-3)
	return x + "...", nil
}

// Capitalize returns StringUtils.capitalize.
func Capitalize(s string) string {
	u := utf16Units(s)
	if len(u) == 0 {
		return s
	}
	r := jvm.CharacterToTitleCase(rune(u[0]))
	if r == rune(u[0]) {
		return s
	}
	u[0] = uint16(r)
	return unitsString(u)
}

// Uncapitalize returns StringUtils.uncapitalize.
func Uncapitalize(s string) string {
	u := utf16Units(s)
	if len(u) == 0 {
		return s
	}
	r := jvm.CharacterToLowerCase(rune(u[0]))
	if r == rune(u[0]) {
		return s
	}
	u[0] = uint16(r)
	return unitsString(u)
}

// Join returns StringUtils.join for object values.
func Join(elems []any, sep string) string {
	var b strings.Builder
	for i, v := range elems {
		if i > 0 {
			b.WriteString(sep)
		}
		if v != nil {
			b.WriteString(jvm.ToString(v))
		}
	}
	return b.String()
}

// Split returns StringUtils.split using separator characters.
func Split(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	if sep == "" {
		return []string{s}
	}
	set := make(map[uint16]bool)
	for _, c := range utf16Units(sep) {
		set[c] = true
	}
	u := utf16Units(s)
	var out []string
	for i := 0; i < len(u); {
		for i < len(u) && set[u[i]] {
			i++
		}
		start := i
		for i < len(u) && !set[u[i]] {
			i++
		}
		if start < i {
			out = append(out, unitsString(u[start:i]))
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// SplitByWholeSeparator returns StringUtils.splitByWholeSeparator.
func SplitByWholeSeparator(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	if sep == "" {
		return fieldsJavaWhitespace(s)
	}
	var out []string
	for {
		p := strings.Index(s, sep)
		if p < 0 {
			if s != "" {
				out = append(out, s)
			}
			break
		}
		if p > 0 {
			out = append(out, s[:p])
		}
		s = s[p+len(sep):]
	}
	if out == nil {
		return []string{}
	}
	return out
}

func fieldsJavaWhitespace(s string) []string {
	u := utf16Units(s)
	var out []string
	for i := 0; i < len(u); {
		for i < len(u) && jvm.IsWhitespace(rune(u[i])) {
			i++
		}
		a := i
		for i < len(u) && !jvm.IsWhitespace(rune(u[i])) {
			i++
		}
		if a < i {
			out = append(out, unitsString(u[a:i]))
		}
	}
	return out
}

// Strip returns StringUtils.strip.
func Strip(s string) string { return jvm.Strip(s) }

// StripStart returns StringUtils.stripStart.
func StripStart(s, stripChars string) string {
	u := utf16Units(s)
	i := 0
	if stripChars == "" {
		for i < len(u) && jvm.IsWhitespace(rune(u[i])) {
			i++
		}
	} else {
		set := unitSet(stripChars)
		for i < len(u) && set[u[i]] {
			i++
		}
	}
	return unitsString(u[i:])
}

// StripEnd returns StringUtils.stripEnd.
func StripEnd(s, stripChars string) string {
	u := utf16Units(s)
	i := len(u)
	if stripChars == "" {
		for i > 0 && jvm.IsWhitespace(rune(u[i-1])) {
			i--
		}
	} else {
		set := unitSet(stripChars)
		for i > 0 && set[u[i-1]] {
			i--
		}
	}
	return unitsString(u[:i])
}
func unitSet(s string) map[uint16]bool {
	m := map[uint16]bool{}
	for _, c := range utf16Units(s) {
		m[c] = true
	}
	return m
}

// IsWhitespace returns StringUtils.isWhitespace.
func IsWhitespace(s string) bool {
	for _, c := range utf16Units(s) {
		if !jvm.IsWhitespace(rune(c)) {
			return false
		}
	}
	return true
}

// LastIndexOfAny returns StringUtils.lastIndexOfAny.
func LastIndexOfAny(s string, search ...string) int {
	best := -1
	su := utf16Units(s)
	for _, q := range search {
		qu := utf16Units(q)
		if len(qu) == 0 {
			if len(su) > best {
				best = len(su)
			}
			continue
		}
		for i := len(su) - len(qu); i >= 0; i-- {
			ok := true
			for j := range qu {
				if su[i+j] != qu[j] {
					ok = false
					break
				}
			}
			if ok {
				if i > best {
					best = i
				}
				break
			}
		}
	}
	return best
}

// Substring returns StringUtils.substring with negative indexes.
func Substring(s string, start int, end ...int) string {
	n := jvm.UTF16Len(s)
	e := n
	if len(end) > 0 {
		e = end[0]
	}
	if e < 0 {
		e = n + e
	}
	if start < 0 {
		start = n + start
	}
	if e > n {
		e = n
	}
	if start < 0 {
		start = 0
	}
	if e < 0 || start > e {
		return ""
	}
	x, _ := jvm.Substring(s, start, e)
	return x
}

// DefaultIfEmpty returns the default when a string is empty.
func DefaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// IsEmpty reports whether the string is empty.
func IsEmpty(s string) bool { return s == "" }

func utf16Units(s string) []uint16 {
	out := make([]uint16, 0, len(s))
	for len(s) > 0 {
		r, n := decodeRune(s)
		s = s[n:]
		if r <= 0xffff {
			out = append(out, uint16(r))
		} else {
			r -= 0x10000
			out = append(out, uint16(0xd800+r>>10), uint16(0xdc00+r&0x3ff))
		}
	}
	return out
}
func decodeRune(s string) (rune, int) {
	if len(s) >= 3 && s[0] == 0xed && s[1]&0xe0 == 0xa0 && s[2]&0xc0 == 0x80 {
		return rune(s[0]&15)<<12 | rune(s[1]&63)<<6 | rune(s[2]&63), 3
	}
	r, n := rangeFirst(s)
	return r, n
}
func rangeFirst(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}
func unitsString(u []uint16) string {
	var b strings.Builder
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
			b.WriteRune(rune(c-0xd800)<<10 + rune(u[i+1]-0xdc00) + 0x10000)
			i++
		} else if c >= 0xd800 && c <= 0xdfff {
			b.Write([]byte{byte(0xe0 | c>>12), byte(0x80 | c>>6&63), byte(0x80 | c&63)})
		} else {
			b.WriteRune(rune(c))
		}
	}
	return b.String()
}
