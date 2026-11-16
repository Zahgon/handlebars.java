// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"strings"
	"sync"
)

// Locale models java.util.Locale's language, script, country, and variant.
type Locale struct {
	// Language is the lowercase ISO language code.
	Language string
	// Script is the title-case ISO 15924 script code.
	Script string
	// Country is the uppercase region code.
	Country string
	// Variant is the locale variant string.
	Variant string
}

// Root is Locale.ROOT.
var Root = Locale{}

var (
	defaultLocale = Locale{Language: "en", Country: "US"}
	localeMu      sync.RWMutex
)

// Default returns the process default locale.
func Default() Locale { localeMu.RLock(); defer localeMu.RUnlock(); return defaultLocale }

// SetDefault changes the process default locale.
func SetDefault(loc Locale) { localeMu.Lock(); defaultLocale = loc; localeMu.Unlock() }

// ForLanguageTag implements Locale.forLanguageTag for ordinary BCP 47 tags.
func ForLanguageTag(tag string) Locale {
	tag = strings.ReplaceAll(tag, "_", "-")
	parts := strings.Split(tag, "-")
	var loc Locale
	if len(parts) == 0 || parts[0] == "" || strings.EqualFold(parts[0], "und") {
		return loc
	}
	loc.Language = strings.ToLower(parts[0])
	for _, p := range parts[1:] {
		switch {
		case len(p) == 4 && loc.Script == "":
			loc.Script = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		case (len(p) == 2 || len(p) == 3) && loc.Country == "":
			loc.Country = strings.ToUpper(p)
		case len(p) >= 1 && len(p) <= 8:
			if loc.Variant == "" {
				loc.Variant = p
			} else {
				loc.Variant += "_" + p
			}
		}
	}
	return loc
}

// String returns Locale.toString().
func (l Locale) String() string {
	var b strings.Builder
	b.WriteString(l.Language)
	if l.Country != "" || l.Variant != "" {
		b.WriteByte('_')
		b.WriteString(l.Country)
	}
	if l.Variant != "" {
		b.WriteByte('_')
		b.WriteString(l.Variant)
	}
	if l.Script != "" {
		if b.Len() != 0 {
			b.WriteString("_#")
		} else {
			b.WriteByte('#')
		}
		b.WriteString(l.Script)
	}
	return b.String()
}

// ToLanguageTag returns Locale.toLanguageTag().
func (l Locale) ToLanguageTag() string {
	var p []string
	if l.Language != "" {
		p = append(p, strings.ToLower(l.Language))
	} else {
		p = append(p, "und")
	}
	if l.Script != "" {
		p = append(p, strings.ToUpper(l.Script[:1])+strings.ToLower(l.Script[1:]))
	}
	if l.Country != "" {
		p = append(p, strings.ToUpper(l.Country))
	}
	if l.Variant != "" {
		p = append(p, strings.Split(strings.ReplaceAll(l.Variant, "_", "-"), "-")...)
	}
	return strings.Join(p, "-")
}
