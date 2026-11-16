// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

package jtext

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// Bundle is a PropertyResourceBundle and its parent chain.
type Bundle struct {
	values map[string]string
	order  []string
	parent *Bundle
	locale jvm.Locale
}

// LoadBundle implements ResourceBundle.getBundle candidate and default-locale
// fallback lookup, reading properties through I18nHelper.UTF8Control.
func LoadBundle(classpathDir, baseName string, loc jvm.Locale, defaultLoc jvm.Locale) (*Bundle, error) {
	root, err := loadBundleFile(classpathDir, baseName, jvm.Root, nil)
	if err != nil {
		return nil, err
	}
	if loc == jvm.Root && root != nil {
		return root, nil
	}
	if b, err := loadCandidateChain(classpathDir, baseName, loc, root); err != nil {
		return nil, err
	} else if b != nil {
		return b, nil
	}
	if defaultLoc != loc {
		if b, err := loadCandidateChain(classpathDir, baseName, defaultLoc, root); err != nil {
			return nil, err
		} else if b != nil {
			return b, nil
		}
	}
	if root != nil {
		return root, nil
	}
	return nil, jvm.Throw("java.util.MissingResourceException", "Can't find bundle for base name "+baseName+", locale "+loc.String())
}

func loadCandidateChain(dir, base string, loc jvm.Locale, root *Bundle) (*Bundle, error) {
	candidates := bundleCandidates(loc)
	parent := root
	found := false
	for i := len(candidates) - 2; i >= 0; i-- {
		candidate := candidates[i]
		bundle, err := loadBundleFile(dir, base, candidate, parent)
		if err != nil {
			return nil, err
		}
		if bundle != nil {
			parent = bundle
			found = true
		}
	}
	if !found {
		return nil, nil
	}
	return parent, nil
}

func loadBundleFile(dir, base string, loc jvm.Locale, parent *Bundle) (*Bundle, error) {
	name := strings.ReplaceAll(base, ".", string(filepath.Separator))
	if suffix := loc.String(); suffix != "" {
		name += "_" + suffix
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".properties"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	values, order, parseErr := loadProperties(jvm.DecodeUTF8(data))
	if parseErr != nil {
		return nil, parseErr
	}
	return &Bundle{values: values, order: order, parent: parent, locale: loc}, nil
}

func bundleCandidates(loc jvm.Locale) []jvm.Locale {
	var out []jvm.Locale
	add := func(l jvm.Locale) {
		for _, old := range out {
			if old == l {
				return
			}
		}
		out = append(out, l)
	}
	language := loc.Language
	if language != "" {
		variants := []string{}
		if loc.Variant != "" {
			parts := strings.Split(loc.Variant, "_")
			for i := len(parts); i > 0; i-- {
				variants = append(variants, strings.Join(parts[:i], "_"))
			}
		}
		for _, variant := range variants {
			add(jvm.Locale{Language: language, Script: loc.Script, Country: loc.Country, Variant: variant})
		}
		if loc.Country != "" {
			add(jvm.Locale{Language: language, Script: loc.Script, Country: loc.Country})
		}
		if loc.Script != "" {
			add(jvm.Locale{Language: language, Script: loc.Script})
			for _, variant := range variants {
				add(jvm.Locale{Language: language, Country: loc.Country, Variant: variant})
			}
			if loc.Country != "" {
				add(jvm.Locale{Language: language, Country: loc.Country})
			}
		}
		add(jvm.Locale{Language: language})
	}
	add(jvm.Root)
	return out
}

// GetString looks up key through this bundle and its parents.
func (b *Bundle) GetString(key string) (string, error) {
	for p := b; p != nil; p = p.parent {
		if v, ok := p.values[key]; ok {
			return v, nil
		}
	}
	return "", jvm.Throw("java.util.MissingResourceException", "Can't find resource for bundle java.util.PropertyResourceBundle, key "+key)
}

// Keys returns ResourceBundle.getKeys enumeration order. Each bundle emits its
// local HashMap keys first, followed by unseen keys from its parent.
func (b *Bundle) Keys() []string {
	set := map[string]bool{}
	var result []string
	for p := b; p != nil; p = p.parent {
		for _, k := range hashMapStringOrder(p.order) {
			if !set[k] {
				set[k] = true
				result = append(result, k)
			}
		}
	}
	return result
}

func hashMapStringOrder(keys []string) []string {
	ordered := append([]string(nil), keys...)
	// PropertyResourceBundle constructs its lookup HashMap from the complete
	// Properties map and pre-sizes to the next power of two at least as large
	// as ceil(size/0.75) (unlike a default, incrementally populated map, whose first
	// table has 16 slots). Six properties therefore use 8 buckets; seven use
	// 16. ResourceBundle.getKeys exposes that distinction.
	needed := (len(ordered)*4 + 2) / 3
	capacity := 1
	for capacity < needed {
		capacity *= 2
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return javaHashBucket(ordered[i], capacity) < javaHashBucket(ordered[j], capacity)
	})
	return ordered
}

func javaHashBucket(s string, capacity int) int {
	h := uint32(jvm.StringHashCode(s))
	h ^= h >> 16
	return int(h & uint32(capacity-1))
}

// Locale returns the locale of the most-specific loaded bundle.
func (b *Bundle) Locale() jvm.Locale { return b.locale }

func loadProperties(text string) (map[string]string, []string, error) {
	physical := splitPropertyLines(text)
	values := make(map[string]string)
	var order []string
	for i := 0; i < len(physical); i++ {
		line := physical[i]
		for hasOddTrailingBackslashes(line) && i+1 < len(physical) {
			line = line[:len(line)-1]
			i++
			line += strings.TrimLeft(physical[i], " \t\f")
		}
		if hasOddTrailingBackslashes(line) {
			line = line[:len(line)-1]
		}
		line = strings.TrimLeft(line, " \t\f")
		if line == "" || line[0] == '#' || line[0] == '!' {
			continue
		}
		keyEnd, valueStart := propertySeparator(line)
		key, err := decodePropertyEscapes(line[:keyEnd])
		if err != nil {
			return nil, nil, err
		}
		value := ""
		if valueStart < len(line) {
			value, err = decodePropertyEscapes(line[valueStart:])
			if err != nil {
				return nil, nil, err
			}
		}
		if _, exists := values[key]; !exists {
			order = append(order, key)
		}
		values[key] = value
	}
	return values, order, nil
}

func splitPropertyLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func hasOddTrailingBackslashes(s string) bool {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n%2 == 1
}

func propertySeparator(s string) (int, int) {
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '=' || c == ':' || c == ' ' || c == '\t' || c == '\f' {
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\f') {
				j++
			}
			if j < len(s) && (s[j] == '=' || s[j] == ':') {
				j++
			}
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\f') {
				j++
			}
			return i, j
		}
	}
	return len(s), len(s)
}

func decodePropertyEscapes(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			b.WriteByte('\\')
			continue
		}
		i++
		switch s[i] {
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 'f':
			b.WriteByte('\f')
		case 'u':
			if i+4 < len(s) {
				if n, err := strconv.ParseUint(s[i+1:i+5], 16, 16); err == nil {
					unit := uint16(n)
					if unit >= 0xD800 && unit <= 0xDBFF && i+10 < len(s) && s[i+5] == '\\' && s[i+6] == 'u' {
						if low, parseErr := strconv.ParseUint(s[i+7:i+11], 16, 16); parseErr == nil && low >= 0xDC00 && low <= 0xDFFF {
							b.WriteRune(rune(0x10000 + (uint32(unit)-0xD800)*0x400 + (uint32(low) - 0xDC00)))
							i += 10
							continue
						}
					}
					writeUTF16Unit(&b, unit)
					i += 4
					continue
				}
			}
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Malformed \\uxxxx encoding.")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String(), nil
}

func writeUTF16Unit(b *strings.Builder, unit uint16) {
	if unit < 0xD800 || unit > 0xDFFF {
		b.WriteRune(rune(unit))
		return
	}
	// WTF-8 preserves an unpaired Java UTF-16 surrogate as its three-byte
	// UTF-8-shaped encoding rather than replacing it with U+FFFD.
	b.WriteByte(byte(0xE0 | unit>>12))
	b.WriteByte(byte(0x80 | (unit>>6)&0x3F))
	b.WriteByte(byte(0x80 | unit&0x3F))
}
