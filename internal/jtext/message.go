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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// MessageFormat formats arguments using java.text.MessageFormat pattern
// quoting, number subformats, and ChoiceFormat.
func MessageFormat(pattern string, loc jvm.Locale, args []any) (string, error) {
	var out strings.Builder
	quoted := false
	for i := 0; i < len(pattern); {
		if pattern[i] == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				out.WriteByte('\'')
				i += 2
				continue
			}
			quoted = !quoted
			i++
			continue
		}
		if pattern[i] == '{' && !quoted {
			end, err := messageElementEnd(pattern, i)
			if err != nil {
				return "", err
			}
			elem := pattern[i+1 : end]
			value, err := formatMessageElement(elem, loc, args)
			if err != nil {
				return "", err
			}
			out.WriteString(value)
			i = end + 1
			continue
		}
		out.WriteByte(pattern[i])
		i++
	}
	return out.String(), nil
}

func messageElementEnd(pattern string, start int) (int, error) {
	depth := 1
	quoted := false
	for i := start + 1; i < len(pattern); i++ {
		if pattern[i] == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				i++
				continue
			}
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		if pattern[i] == '{' {
			depth++
		} else if pattern[i] == '}' {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, jvm.Throw("java.lang.IllegalArgumentException", "Unmatched braces in the pattern.")
}

func splitMessageElement(s string) []string {
	var result []string
	start, depth := 0, 0
	quoted := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				i++
				continue
			}
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			depth--
		} else if s[i] == ',' && depth == 0 && len(result) < 2 {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	return append(result, s[start:])
}

func formatMessageElement(element string, loc jvm.Locale, args []any) (string, error) {
	parts := splitMessageElement(element)
	indexText := strings.TrimSpace(parts[0])
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		return "", jvm.Throw("java.lang.IllegalArgumentException", "can't parse argument number: "+indexText)
	}
	if index >= len(args) {
		return "{" + element + "}", nil
	}
	arg := args[index]
	if len(parts) == 1 {
		if arg == nil {
			return "null", nil
		}
		if jvm.IsNumber(arg) {
			f := NewNumberInstance(loc)
			return f.Format(arg)
		}
		if t, ok := arg.(time.Time); ok {
			return formatDate(t, loc, "date", "short"), nil
		}
		return jvm.ToString(arg), nil
	}
	typ := strings.ToLower(strings.TrimSpace(parts[1]))
	style := ""
	if len(parts) > 2 {
		style = strings.TrimSpace(parts[2])
	}
	if typ != "number" && typ != "choice" && typ != "date" && typ != "time" {
		return "", jvm.Throw("java.lang.IllegalArgumentException", "unknown format type: "+typ)
	}
	if arg == nil {
		return "null", nil
	}
	switch typ {
	case "number":
		if !jvm.IsNumber(arg) {
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Cannot format given Object as a Number")
		}
		var f *NumberFormat
		switch strings.ToLower(style) {
		case "":
			f = NewNumberInstance(loc)
		case "integer":
			f = NewIntegerInstance(loc)
		case "currency":
			f = NewCurrencyInstance(loc)
		case "percent":
			f = NewPercentInstance(loc)
		default:
			f, err = NewDecimalFormat(style, loc)
			if err != nil {
				return "", err
			}
		}
		return f.Format(arg)
	case "choice":
		if !jvm.IsNumber(arg) {
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Cannot format given Object as a Number")
		}
		selected, e := formatChoice(style, jvm.DoubleValue(arg))
		if e != nil {
			return "", e
		}
		if strings.Contains(selected, "{") {
			return MessageFormat(selected, loc, args)
		}
		return selected, nil
	case "date", "time":
		t, ok := arg.(time.Time)
		if !ok && jvm.IsNumber(arg) {
			t = time.UnixMilli(jvm.LongValue(arg)).UTC()
			ok = true
		}
		if !ok {
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Cannot format given Object as a Date")
		}
		if style == "" {
			style = "medium"
		}
		return formatDate(t, loc, typ, style), nil
	}
	return "", nil
}

func formatChoice(pattern string, value float64) (string, error) {
	choices := splitUnquoted(pattern, '|')
	selected := ""
	valid := false
	for _, choice := range choices {
		op := -1
		for i, r := range choice {
			if r == '#' || r == '<' || r == '≤' {
				op = i
				break
			}
		}
		if op < 0 {
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Each interval must contain a number before a format")
		}
		limitText := strings.TrimSpace(choice[:op])
		limit := 0.0
		var err error
		if limitText == "∞" || limitText == "+∞" {
			limit = math.Inf(1)
		} else if limitText == "-∞" {
			limit = math.Inf(-1)
		} else {
			limit, err = strconv.ParseFloat(limitText, 64)
		}
		if err != nil {
			return "", jvm.Throw("java.lang.IllegalArgumentException", "Choice Pattern incorrect: "+pattern)
		}
		r := utf8RuneAt(choice, op)
		matches := value > limit
		if r == '#' || r == '≤' {
			matches = value >= limit
		}
		if matches {
			selected = unquoteMessage(choice[op+len(string(r)):])
			valid = true
		}
	}
	if !valid && len(choices) > 0 {
		c := choices[0]
		for i, r := range c {
			if r == '#' || r == '<' || r == '≤' {
				selected = unquoteMessage(c[i+len(string(r)):])
				break
			}
		}
	}
	return selected, nil
}

func utf8RuneAt(s string, i int) rune {
	r := rune(s[i])
	if r < 128 {
		return r
	}
	for _, x := range s[i:] {
		return x
	}
	return 0
}

func unquoteMessage(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			if i+1 < len(s) && s[i+1] == '\'' {
				b.WriteByte('\'')
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func formatDate(t time.Time, loc jvm.Locale, kind, style string) string {
	lang := strings.ToLower(loc.Language)
	if kind == "time" {
		st := strings.ToLower(style)
		if lang == "en" {
			base := t.Format("3:04")
			if st != "short" {
				base = t.Format("3:04:05")
			}
			base += "\u202f" + t.Format("PM")
			if st == "long" {
				base += " UTC"
			}
			if st == "full" {
				base += " Coordinated Universal Time"
			}
			return base
		}
		base := t.Format("15:04")
		if st != "short" {
			base = t.Format("15:04:05")
		}
		if st == "long" {
			base += " UTC"
		}
		if st == "full" {
			zones := map[string]string{"de": "Koordinierte Weltzeit", "fr": "temps universel coordonné", "es": "(tiempo universal coordinado)", "tr": "Eş Güdümlü Evrensel Zaman", "": "Coordinated Universal Time"}
			z := zones[lang]
			if lang == "es" && strings.EqualFold(loc.Country, "AR") {
				z = "hora universal coordinada"
			}
			base += " " + z
		}
		return base
	}
	st := strings.ToLower(style)
	d, m, y := t.Day(), int(t.Month()), t.Year()
	monthsLong := map[string][]string{"de": {"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}, "fr": {"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"}, "es": {"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}, "tr": {"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"}}
	monthsShort := map[string][]string{"fr": {"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."}, "es": {"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"}, "tr": {"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"}}
	weekdays := map[string][]string{"de": {"Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"}, "fr": {"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}, "es": {"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}, "tr": {"Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi"}}
	if lang == "en" {
		switch st {
		case "short":
			return t.Format("1/2/06")
		case "medium":
			return t.Format("Jan 2, 2006")
		case "long":
			return t.Format("January 2, 2006")
		case "full":
			return t.Format("Monday, January 2, 2006")
		}
	}
	if lang == "" {
		switch st {
		case "short":
			return t.Format("2006-01-02")
		case "medium", "long":
			return strconv.Itoa(y) + " " + t.Format("Jan") + " " + strconv.Itoa(d)
		case "full":
			return strconv.Itoa(y) + " " + t.Format("Jan") + " " + strconv.Itoa(d) + ", " + t.Format("Mon")
		}
	}
	if st == "short" {
		switch lang {
		case "de":
			return t.Format("02.01.06")
		case "fr":
			return t.Format("02/01/2006")
		case "es":
			return t.Format("2/1/06")
		case "tr":
			return t.Format("02.01.2006")
		}
	}
	if st == "medium" {
		if lang == "de" {
			return t.Format("02.01.2006")
		}
		return strconv.Itoa(d) + " " + monthsShort[lang][m-1] + " " + strconv.Itoa(y)
	}
	month := monthsLong[lang][m-1]
	base := strconv.Itoa(d) + " " + month + " " + strconv.Itoa(y)
	if lang == "de" {
		base = strconv.Itoa(d) + ". " + month + " " + strconv.Itoa(y)
	}
	if lang == "es" {
		base = strconv.Itoa(d) + " de " + month + " de " + strconv.Itoa(y)
	}
	if st == "full" {
		day := weekdays[lang][int(t.Weekday())]
		if lang == "tr" {
			return base + " " + day
		}
		if lang == "fr" {
			return day + " " + base
		}
		return day + ", " + base
	}
	if st == "long" {
		return base
	}
	return javaDatePattern(t, style, loc)
}

func javaDatePattern(t time.Time, pattern string, _ jvm.Locale) string {
	r := strings.NewReplacer("yyyy", "2006", "yy", "06", "MMMM", "January", "MMM", "Jan", "MM", "01", "dd", "02", "d", "2", "HH", "15", "hh", "03", "mm", "04", "ss", "05", "a", "PM")
	return t.Format(r.Replace(pattern))
}
