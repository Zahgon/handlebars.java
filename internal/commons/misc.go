// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package commons

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/jknack/handlebars.java/internal/jvm"
)

var creatablePattern = regexp.MustCompile(`^[+-]?(?:(?:0[xX][0-9a-fA-F]+)|(?:#[0-9a-fA-F]+)|(?:0[0-7]+)|(?:(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?))[lLfFdD]?$`)

// IsCreatable returns NumberUtils.isCreatable.
func IsCreatable(s string) bool {
	if strings.TrimRightFunc(s, unicode.IsSpace) != s {
		return false
	}
	original := s
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if s == "" {
		return false
	}
	if s != original {
		q := s
		if q[0] == '+' || q[0] == '-' {
			q = q[1:]
		}
		exp := strings.IndexAny(q, "eE")
		if strings.HasPrefix(q, "#") || strings.HasPrefix(q, "0x") || strings.HasPrefix(q, "0X") || !strings.ContainsAny(q, ".eEfFdD") || (exp == 1 && q[0] == '0') || (exp >= 0 && exp+1 < len(q) && q[exp+1] == '-') {
			return false
		}
	}
	if !creatablePattern.MatchString(s) {
		return false
	}
	t := s
	if t[0] == '+' || t[0] == '-' {
		t = t[1:]
	}
	suffix := t[len(t)-1]
	floatSuffix := strings.ContainsRune("fFdD", rune(suffix))
	if strings.ContainsRune("lLfFdD", rune(suffix)) {
		t = t[:len(t)-1]
	}
	hex := strings.HasPrefix(t, "#") || strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "0X")
	if !floatSuffix && !hex && len(t) > 1 && t[0] == '0' && !strings.ContainsAny(t, ".eE") {
		for _, c := range t[1:] {
			if c < '0' || c > '7' {
				return false
			}
		}
	}
	last := s[len(s)-1]
	if last == 'l' || last == 'L' {
		if hex {
			return true
		}
		return !strings.ContainsAny(s, ".eEfFdD")
	}
	return true
}

// LocaleToLocale returns LocaleUtils.toLocale.
func LocaleToLocale(s string) (jvm.Locale, error) {
	if s == "" {
		return jvm.Root, nil
	}
	if s[0] == '_' {
		if len(s) < 3 || s[1] < 'A' || s[1] > 'Z' || s[2] < 'A' || s[2] > 'Z' || (len(s) > 3 && (len(s) < 5 || s[3] != '_')) {
			return jvm.Locale{}, invalidLocale(s)
		}
		l := jvm.Locale{Country: s[1:3]}
		if len(s) > 4 {
			l.Variant = s[4:]
		}
		return l, nil
	}
	p := strings.Split(s, "_")
	validLang := func(x string) bool {
		if len(x) != 2 && len(x) != 3 {
			return false
		}
		for _, c := range x {
			if c < 'a' || c > 'z' {
				return false
			}
		}
		return true
	}
	if !validLang(p[0]) {
		return jvm.Locale{}, invalidLocale(s)
	}
	l := jvm.Locale{Language: p[0]}
	switch len(p) {
	case 1:
		return l, nil
	case 2:
		if len(p[1]) != 2 {
			return jvm.Locale{}, invalidLocale(s)
		}
		for _, c := range p[1] {
			if c < 'A' || c > 'Z' {
				return jvm.Locale{}, invalidLocale(s)
			}
		}
		l.Country = p[1]
		return l, nil
	case 3:
		if p[1] == "" && p[2] != "" {
			l.Variant = p[2]
			return l, nil
		}
		if len(p[1]) != 2 || p[2] == "" {
			return jvm.Locale{}, invalidLocale(s)
		}
		for _, c := range p[1] {
			if c < 'A' || c > 'Z' {
				return jvm.Locale{}, invalidLocale(s)
			}
		}
		l.Country = p[1]
		l.Variant = p[2]
		return l, nil
	}
	return jvm.Locale{}, invalidLocale(s)
}
func invalidLocale(s string) error {
	return jvm.Throw("java.lang.IllegalArgumentException", "Invalid locale format: "+s)
}

// ValidateNotNull implements Validate.notNull and returns the input.
func ValidateNotNull(v any, message string, args ...any) (any, error) {
	if v == nil {
		return nil, jvm.Throw("java.lang.NullPointerException", formatMessage(message, args))
	}
	return v, nil
}

// ValidateNotEmpty implements Validate.notEmpty for strings.
func ValidateNotEmpty(v string, message string, args ...any) (string, error) {
	if v == "" {
		return v, jvm.Throw("java.lang.IllegalArgumentException", formatMessage(message, args))
	}
	return v, nil
}

// ValidateIsTrue implements Validate.isTrue.
func ValidateIsTrue(ok bool, message string, args ...any) error {
	if !ok {
		return jvm.Throw("java.lang.IllegalArgumentException", formatMessage(message, args))
	}
	return nil
}

// ValidateValidIndex implements Validate.validIndex for indexed collections.
func ValidateValidIndex(index, size int, message string, args ...any) error {
	if index < 0 || index >= size {
		return jvm.Throw("java.lang.IndexOutOfBoundsException", formatMessage(message, args))
	}
	return nil
}

func formatMessage(s string, args []any) string {
	if len(args) == 0 {
		return s
	}
	converted := make([]any, len(args))
	copy(converted, args)
	for i, v := range converted {
		switch x := v.(type) {
		case int8:
			converted[i] = int64(x)
		case int16:
			converted[i] = int64(x)
		case int32:
			converted[i] = int64(x)
		}
	}
	return javaSimpleFormat(s, converted)
}
func javaSimpleFormat(s string, args []any) string {
	var b strings.Builder
	argi := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+1 < len(s) {
			if s[i+1] == '%' {
				b.WriteByte('%')
				i++
				continue
			}
			j := i + 1
			for j < len(s) && strings.ContainsRune("-#+ 0,(<", rune(s[j])) {
				j++
			}
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j < len(s) && s[j] == 's' && argi < len(args) {
				b.WriteString(jvm.ToString(args[argi]))
				argi++
				i = j
				continue
			}
			if j < len(s) && (s[j] == 'd' || s[j] == 'x' || s[j] == 'X') && argi < len(args) {
				n, _ := strconv.ParseInt(fmt.Sprint(args[argi]), 10, 64)
				base := 10
				if s[j] == 'x' || s[j] == 'X' {
					base = 16
				}
				z := strconv.FormatInt(n, base)
				if s[j] == 'X' {
					z = strings.ToUpper(z)
				}
				b.WriteString(z)
				argi++
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
