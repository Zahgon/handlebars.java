// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func toUnits(s string) []uint16 {
	u := make([]uint16, 0, len(s))
	for i := 0; i < len(s); {
		if i+2 < len(s) && s[i] == 0xed && s[i+1]&0xe0 == 0xa0 && s[i+2]&0xc0 == 0x80 {
			u = append(u, uint16(s[i]&15)<<12|uint16(s[i+1]&63)<<6|uint16(s[i+2]&63))
			i += 3
			continue
		}
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && n == 1 {
			u = append(u, 0xfffd)
			i++
			continue
		}
		i += n
		if r <= 0xffff {
			u = append(u, uint16(r))
		} else {
			r -= 0x10000
			u = append(u, uint16(0xd800+r>>10), uint16(0xdc00+r&0x3ff))
		}
	}
	return u
}
func fromUnits(u []uint16) string {
	var b strings.Builder
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
			r := rune(c-0xd800)<<10 + rune(u[i+1]-0xdc00) + 0x10000
			b.WriteRune(r)
			i++
		} else if c >= 0xd800 && c <= 0xdfff {
			b.WriteByte(byte(0xe0 | c>>12))
			b.WriteByte(byte(0x80 | c>>6&0x3f))
			b.WriteByte(byte(0x80 | c&0x3f))
		} else {
			b.WriteRune(rune(c))
		}
	}
	return b.String()
}

// UTF16Len returns String.length().
func UTF16Len(s string) int { return len(toUnits(s)) }

// CharAt returns String.charAt or its Java exception.
func CharAt(s string, i int) (Char, error) {
	u := toUnits(s)
	if i < 0 || i >= len(u) {
		return 0, Throw("java.lang.StringIndexOutOfBoundsException", "String index out of range: "+itoa(i))
	}
	return Char(u[i]), nil
}

// CodePointAt returns String.codePointAt or its Java exception.
func CodePointAt(s string, i int) (rune, error) {
	u := toUnits(s)
	if i < 0 || i >= len(u) {
		return 0, Throw("java.lang.StringIndexOutOfBoundsException", "index "+itoa(i)+", length "+itoa(len(u)))
	}
	c := u[i]
	if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
		return rune(c-0xd800)<<10 + rune(u[i+1]-0xdc00) + 0x10000, nil
	}
	return rune(c), nil
}

// Substring returns String.substring(begin,end) or its Java exception.
func Substring(s string, begin, end int) (string, error) {
	u := toUnits(s)
	if begin < 0 || end > len(u) || begin > end {
		return "", Throw("java.lang.StringIndexOutOfBoundsException", "begin "+itoa(begin)+", end "+itoa(end)+", length "+itoa(len(u)))
	}
	return fromUnits(u[begin:end]), nil
}

// SubSequence returns String.subSequence with its range exception text.
func SubSequence(s string, begin, end int) (string, error) {
	u := toUnits(s)
	if begin < 0 || end > len(u) || begin > end {
		return "", Throw("java.lang.IndexOutOfBoundsException", "Range ["+itoa(begin)+", "+itoa(end)+") out of bounds for length "+itoa(len(u)))
	}
	return fromUnits(u[begin:end]), nil
}

// IndexOf returns String.indexOf(String,int) in UTF-16 indices.
func IndexOf(s, sub string, from int) int {
	u, q := toUnits(s), toUnits(sub)
	if from < 0 {
		from = 0
	}
	if len(q) == 0 {
		if from > len(u) {
			return len(u)
		}
		return from
	}
	for i := from; i+len(q) <= len(u); i++ {
		ok := true
		for j := range q {
			if u[i+j] != q[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var a [24]byte
	n := len(a)
	for i > 0 {
		n--
		a[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		a[n] = '-'
	}
	return string(a[n:])
}

// CompareStrings implements String.compareTo over UTF-16 code units.
func CompareStrings(a, b string) int {
	x, y := toUnits(a), toUnits(b)
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	for i := 0; i < n; i++ {
		if x[i] != y[i] {
			return int(x[i]) - int(y[i])
		}
	}
	return len(x) - len(y)
}

// CompareStringsIgnoreCase returns String.compareToIgnoreCase.
func CompareStringsIgnoreCase(a, b string) int {
	x, y := toUnits(a), toUnits(b)
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	for i := 0; i < n; i++ {
		c1, c2 := rune(x[i]), rune(y[i])
		if c1 == c2 {
			continue
		}
		u1, u2 := CharacterToUpperCase(c1), CharacterToUpperCase(c2)
		if u1 == u2 {
			continue
		}
		l1, l2 := CharacterToLowerCase(u1), CharacterToLowerCase(u2)
		if l1 != l2 {
			return int(l1 - l2)
		}
	}
	return len(x) - len(y)
}

// EqualsIgnoreCase returns String.equalsIgnoreCase.
func EqualsIgnoreCase(a, b string) bool {
	return UTF16Len(a) == UTF16Len(b) && CompareStringsIgnoreCase(a, b) == 0
}

// Repeat returns String.repeat or its negative-count exception.
func Repeat(s string, count int) (string, error) {
	if count < 0 {
		return "", Throw("java.lang.IllegalArgumentException", "count is negative: "+itoa(count))
	}
	if count == 0 || s == "" {
		return "", nil
	}
	if count > int(^uint(0)>>1)/len(s) {
		return "", ThrowNoMessage("java.lang.OutOfMemoryError")
	}
	return strings.Repeat(s, count), nil
}

// Replace returns String.replace(CharSequence,CharSequence).
func Replace(s, old, new string) string {
	u, o, n := toUnits(s), toUnits(old), toUnits(new)
	if len(o) == 0 {
		out := make([]uint16, 0, len(u)+(len(u)+1)*len(n))
		out = append(out, n...)
		for _, c := range u {
			out = append(out, c)
			out = append(out, n...)
		}
		return fromUnits(out)
	}
	var out []uint16
	for i := 0; i < len(u); {
		match := i+len(o) <= len(u)
		if match {
			for j := range o {
				if u[i+j] != o[j] {
					match = false
					break
				}
			}
		}
		if match {
			out = append(out, n...)
			i += len(o)
		} else {
			out = append(out, u[i])
			i++
		}
	}
	return fromUnits(out)
}

// RegexSplit returns String.split(regex,limit) for the regular-expression subset used by the core.
func RegexSplit(s, pattern string, limit int) ([]string, error) {
	re, e := regexp.Compile(pattern)
	if e != nil {
		return nil, Throw("java.util.regex.PatternSyntaxException", e.Error())
	}
	n := -1
	if limit > 0 {
		n = limit
	}
	parts := re.Split(s, n)
	if limit == 0 {
		for len(parts) > 0 && parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
		}
	}
	return parts, nil
}

// StringHashCode returns String.hashCode over UTF-16 code units.
func StringHashCode(s string) int32 {
	h := int32(0)
	for _, c := range toUnits(s) {
		h = 31*h + int32(c)
	}
	return h
}

// Trim implements String.trim, removing code units no greater than U+0020.
func Trim(s string) string {
	u := toUnits(s)
	a, b := 0, len(u)
	for a < b && u[a] <= 0x20 {
		a++
	}
	for a < b && u[b-1] <= 0x20 {
		b--
	}
	return fromUnits(u[a:b])
}

// Strip implements String.strip using Character.isWhitespace.
func Strip(s string) string {
	u := toUnits(s)
	a, b := 0, len(u)
	for a < b && IsWhitespace(rune(u[a])) {
		a++
	}
	for a < b && IsWhitespace(rune(u[b-1])) {
		b--
	}
	return fromUnits(u[a:b])
}

// IsBlank implements String.isBlank.
func IsBlank(s string) bool {
	for _, c := range toUnits(s) {
		if !IsWhitespace(rune(c)) {
			return false
		}
	}
	return true
}

// IsWhitespace implements Character.isWhitespace for Unicode 15.
func IsWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', 0x0b, '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2008, 0x2009, 0x200a, 0x2028, 0x2029, 0x205f, 0x3000:
		return true
	}
	return false
}

// IsSpaceChar implements Character.isSpaceChar.
func IsSpaceChar(r rune) bool {
	return unicode.Is(unicode.Zs, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}

// IsLetter implements Character.isLetter.
func IsLetter(r rune) bool { return unicode.IsLetter(r) }

// IsDigit implements Character.isDigit.
func IsDigit(r rune) bool { return unicode.Is(unicode.Nd, r) }

// IsLetterOrDigit implements Character.isLetterOrDigit.
func IsLetterOrDigit(r rune) bool { return IsLetter(r) || IsDigit(r) }

// IsUpperCase implements Character.isUpperCase.
func IsUpperCase(r rune) bool { return unicode.IsUpper(r) || unicode.Is(unicode.Other_Uppercase, r) }

// IsLowerCase implements Character.isLowerCase.
func IsLowerCase(r rune) bool { return unicode.IsLower(r) || unicode.Is(unicode.Other_Lowercase, r) }

// CharacterType returns Character.getType's numeric category.
func CharacterType(r rune) int {
	tables := []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Mn, unicode.Me, unicode.Mc, unicode.Nd, unicode.Nl, unicode.No, unicode.Zs, unicode.Zl, unicode.Zp, unicode.Cc, unicode.Cf, unicode.Co, unicode.Cs, unicode.Pd, unicode.Ps, unicode.Pe, unicode.Pc, unicode.Po, unicode.Sm, unicode.Sc, unicode.Sk, unicode.So, unicode.Pi, unicode.Pf}
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30}
	for i, t := range tables {
		if unicode.Is(t, r) {
			return nums[i]
		}
	}
	return 0
}

// CharacterToUpperCase implements Character.toUpperCase(int), which is one code point.
func CharacterToUpperCase(r rune) rune {
	if r == 0x00df {
		return r
	}
	return unicode.ToUpper(r)
}

// CharacterToLowerCase implements Character.toLowerCase(int), which is one code point.
func CharacterToLowerCase(r rune) rune { return unicode.ToLower(r) }

// CharacterToTitleCase implements Character.toTitleCase(int).
func CharacterToTitleCase(r rune) rune { return unicode.ToTitle(r) }

// ToUpperCase implements String.toUpperCase(Locale), including full mappings.
func ToUpperCase(s string, loc Locale) string { return mapCase(s, loc, true) }

// ToLowerCase implements String.toLowerCase(Locale), including contextual mappings.
func ToLowerCase(s string, loc Locale) string { return mapCase(s, loc, false) }

func mapCase(s string, loc Locale, upper bool) string {
	r := javaCodePoints(s)
	var b strings.Builder
	lang := strings.ToLower(loc.Language)
	for i, c := range r {
		if upper {
			if lang == "lt" && c == 0x0307 && afterSoftDotted(r, i) {
				continue
			}
			if mapped, ok := fullUpper[c]; ok {
				for _, x := range mapped {
					b.WriteRune(x)
				}
				continue
			}
			switch c {
			case 'i':
				if lang == "tr" || lang == "az" {
					b.WriteRune('İ')
					continue
				}
			case 0x0131:
				if lang == "tr" || lang == "az" {
					b.WriteRune('I')
					continue
				}
			}
			writeJavaRune(&b, unicode.ToUpper(c))
			continue
		}
		if lang == "lt" {
			switch c {
			case 'I', 'J', 0x012e:
				writeJavaRune(&b, unicode.ToLower(c))
				if hasMoreAbove(r, i) {
					b.WriteRune(0x0307)
				}
				continue
			case 0x00cc:
				b.WriteString("i\u0307\u0300")
				continue
			case 0x00cd:
				b.WriteString("i\u0307\u0301")
				continue
			case 0x0128:
				b.WriteString("i\u0307\u0303")
				continue
			}
		}
		switch c {
		case 'I':
			if lang == "tr" || lang == "az" {
				if hasDotAboveAfter(r, i) {
					b.WriteRune('i')
				} else {
					b.WriteRune('ı')
				}
				continue
			}
		case 'İ':
			if lang == "tr" || lang == "az" {
				b.WriteRune('i')
			} else {
				b.WriteString("i̇")
			}
			continue
		case 0x0307:
			if (lang == "tr" || lang == "az") && afterCapitalI(r, i) {
				continue
			}
		case 'Σ':
			if finalSigma(r, i) {
				b.WriteRune('ς')
			} else {
				b.WriteRune('σ')
			}
			continue
		}
		writeJavaRune(&b, unicode.ToLower(c))
	}
	return b.String()
}

func isCombining(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r)
}
func isAboveMark(r rune) bool {
	return (r >= 0x0300 && r <= 0x0314) || r == 0x033d || r == 0x0342 || r == 0x0343 || r == 0x0344 || r == 0x0346 || r == 0x034a || r == 0x034b || r == 0x034c || r == 0x0350 || r == 0x0351 || r == 0x0352 || r == 0x0357 || r == 0x035b || r == 0x0363 || r == 0x0364 || r == 0x0365 || r == 0x0366 || r == 0x0367 || r == 0x0368 || r == 0x0369 || r == 0x036a || r == 0x036b || r == 0x036c || r == 0x036d || r == 0x036e || r == 0x036f
}
func hasMoreAbove(r []rune, i int) bool {
	for j := i + 1; j < len(r) && isCombining(r[j]); j++ {
		if isAboveMark(r[j]) {
			return true
		}
	}
	return false
}
func hasDotAboveAfter(r []rune, i int) bool {
	for j := i + 1; j < len(r) && isCombining(r[j]); j++ {
		if r[j] == 0x0307 {
			return true
		}
		if isAboveMark(r[j]) {
			return false
		}
	}
	return false
}
func afterCapitalI(r []rune, i int) bool {
	for j := i - 1; j >= 0; j-- {
		if isCombining(r[j]) {
			if isAboveMark(r[j]) {
				return false
			}
			continue
		}
		return r[j] == 'I'
	}
	return false
}
func afterSoftDotted(r []rune, i int) bool {
	for j := i - 1; j >= 0; j-- {
		if isCombining(r[j]) {
			if isAboveMark(r[j]) {
				return false
			}
			continue
		}
		return r[j] == 'i' || r[j] == 'j' || r[j] == 0x012f
	}
	return false
}

func javaCodePoints(s string) []rune {
	u := toUnits(s)
	out := make([]rune, 0, len(u))
	for i := 0; i < len(u); i++ {
		if u[i] >= 0xd800 && u[i] <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
			out = append(out, rune(u[i]-0xd800)<<10+rune(u[i+1]-0xdc00)+0x10000)
			i++
		} else {
			out = append(out, rune(u[i]))
		}
	}
	return out
}
func writeJavaRune(b *strings.Builder, r rune) {
	if r >= 0xd800 && r <= 0xdfff {
		b.WriteString(fromUnits([]uint16{uint16(r)}))
	} else {
		b.WriteRune(r)
	}
}
func finalSigma(r []rune, i int) bool {
	before := false
	for j := i - 1; j >= 0; j-- {
		if isCaseIgnorable(r[j]) {
			continue
		}
		before = isCased(r[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(r); j++ {
		if isCaseIgnorable(r[j]) {
			continue
		}
		return !isCased(r[j])
	}
	return true
}
func isCased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.Is(unicode.Lt, r)
}
func isCaseIgnorable(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Lm, r) || unicode.Is(unicode.Sk, r) || r == 0x27
}

// EncodeUTF8 encodes a Java string, replacing each unpaired surrogate with '?'.
func EncodeUTF8(s string) []byte {
	u := toUnits(s)
	var b strings.Builder
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
			r := rune(c-0xd800)<<10 + rune(u[i+1]-0xdc00) + 0x10000
			b.WriteRune(r)
			i++
		} else if c >= 0xd800 && c <= 0xdfff {
			b.WriteByte('?')
		} else {
			b.WriteRune(rune(c))
		}
	}
	return []byte(b.String())
}

// DecodeUTF8 decodes with Java's replacement behavior for malformed input.
func DecodeUTF8(src []byte) string {
	var b strings.Builder
	cont := func(x byte) bool { return x&0xc0 == 0x80 }
	for i := 0; i < len(src); {
		b0 := src[i]
		if b0 < 0x80 {
			b.WriteByte(b0)
			i++
			continue
		}
		if b0 >= 0xc2 && b0 <= 0xdf {
			if i+1 < len(src) && cont(src[i+1]) {
				b.WriteRune(rune(b0&0x1f)<<6 | rune(src[i+1]&0x3f))
				i += 2
			} else {
				b.WriteRune(utf8.RuneError)
				i++
			}
			continue
		}
		if b0 >= 0xe0 && b0 <= 0xef {
			if i+1 >= len(src) || !cont(src[i+1]) {
				b.WriteRune(utf8.RuneError)
				i++
				continue
			}
			if b0 == 0xe0 && src[i+1] < 0xa0 {
				b.WriteRune(utf8.RuneError)
				i++
				continue
			}
			if b0 == 0xed && src[i+1] >= 0xa0 {
				n := 2
				if i+2 < len(src) && cont(src[i+2]) {
					n = 3
				}
				b.WriteRune(utf8.RuneError)
				i += n
				continue
			}
			if i+2 >= len(src) || !cont(src[i+2]) {
				b.WriteRune(utf8.RuneError)
				i += 2
				continue
			}
			b.WriteRune(rune(b0&15)<<12 | rune(src[i+1]&63)<<6 | rune(src[i+2]&63))
			i += 3
			continue
		}
		if b0 >= 0xf0 && b0 <= 0xf4 {
			if i+1 >= len(src) || !cont(src[i+1]) {
				b.WriteRune(utf8.RuneError)
				i++
				continue
			}
			if (b0 == 0xf0 && src[i+1] < 0x90) || (b0 == 0xf4 && src[i+1] >= 0x90) {
				b.WriteRune(utf8.RuneError)
				i++
				continue
			}
			if i+2 >= len(src) || !cont(src[i+2]) {
				b.WriteRune(utf8.RuneError)
				i += 2
				continue
			}
			if i+3 >= len(src) || !cont(src[i+3]) {
				b.WriteRune(utf8.RuneError)
				i += 3
				continue
			}
			b.WriteRune(rune(b0&7)<<18 | rune(src[i+1]&63)<<12 | rune(src[i+2]&63)<<6 | rune(src[i+3]&63))
			i += 4
			continue
		}
		b.WriteRune(utf8.RuneError)
		i++
	}
	return b.String()
}
