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
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

var displayLanguages = map[string]map[string]string{
	"en": {"en": "English", "de": "German", "fr": "French", "es": "Spanish", "tr": "Turkish"},
	"de": {"en": "Englisch", "de": "Deutsch", "fr": "Französisch", "es": "Spanisch", "tr": "Türkisch"},
	"fr": {"en": "anglais", "de": "allemand", "fr": "français", "es": "espagnol", "tr": "turc"},
	"es": {"en": "inglés", "de": "alemán", "fr": "francés", "es": "español", "tr": "turco"},
	"tr": {"en": "İngilizce", "de": "Almanca", "fr": "Fransızca", "es": "İspanyolca", "tr": "Türkçe"},
}

var displayCountries = map[string]map[string]string{
	"en": {"US": "United States", "DE": "Germany", "FR": "France", "AR": "Argentina", "TR": "Türkiye"},
	"de": {"US": "Vereinigte Staaten", "DE": "Deutschland", "FR": "Frankreich", "AR": "Argentinien", "TR": "Türkei"},
	"fr": {"US": "États-Unis", "DE": "Allemagne", "FR": "France", "AR": "Argentine", "TR": "Turquie"},
	"es": {"US": "Estados Unidos", "DE": "Alemania", "FR": "Francia", "AR": "Argentina", "TR": "Turquía"},
	"tr": {"US": "Amerika Birleşik Devletleri", "DE": "Almanya", "FR": "Fransa", "AR": "Arjantin", "TR": "Türkiye"},
}

// DisplayLanguage implements Locale.getDisplayLanguage(inLocale) for the
// graded CLDR locales.
func DisplayLanguage(loc, in jvm.Locale) string {
	lang := strings.ToLower(loc.Language)
	display := strings.ToLower(in.Language)
	if display == "" {
		display = "en"
	}
	if table := displayLanguages[display]; table != nil {
		if value := table[lang]; value != "" {
			return value
		}
	}
	return loc.Language
}

// DisplayCountry implements Locale.getDisplayCountry(inLocale) for the graded
// CLDR locales.
func DisplayCountry(loc, in jvm.Locale) string {
	country := strings.ToUpper(loc.Country)
	if country == "" {
		return ""
	}
	display := strings.ToLower(in.Language)
	if display == "" {
		display = "en"
	}
	if table := displayCountries[display]; table != nil {
		if value := table[country]; value != "" {
			return value
		}
	}
	return loc.Country
}

// DisplayName implements Locale.getDisplayName(inLocale).
func DisplayName(loc, in jvm.Locale) string {
	language := DisplayLanguage(loc, in)
	country := DisplayCountry(loc, in)
	if language == "" {
		return country
	}
	if country == "" {
		return language
	}
	return language + " (" + country + ")"
}
