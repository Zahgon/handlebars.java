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
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jknack/handlebars.java/internal/jvm"
)

func locale(tag string) jvm.Locale { return jvm.ForLanguageTag(tag) }

func TestNumberFormatsFromJDK21(t *testing.T) {
	tests := []struct{ tag, kind, want string }{
		{"en-US", "number", "1,234,567.89"},
		{"de-DE", "number", "1.234.567,89"},
		{"fr-FR", "number", "1\u202f234\u202f567,89"},
		{"es-AR", "currency", "$\u00a01.234.567,89"},
		{"tr-TR", "currency", "₺1.234.567,89"},
		{"de-DE", "percent", "12\u00a0%"},
		{"tr-TR", "percent", "%12"},
	}
	for _, tt := range tests {
		var f *NumberFormat
		switch tt.kind {
		case "number":
			f = NewNumberInstance(locale(tt.tag))
		case "currency":
			f = NewCurrencyInstance(locale(tt.tag))
		case "percent":
			f = NewPercentInstance(locale(tt.tag))
		}
		got, err := f.Format(1234567.89)
		if tt.kind == "percent" {
			got, err = f.Format(.12)
		}
		if err != nil || got != tt.want {
			t.Errorf("%s/%s = %q, %v; want %q", tt.tag, tt.kind, got, err, tt.want)
		}
	}
	f := NewIntegerInstance(locale("en-US"))
	got, _ := f.Format(2.5)
	if got != "2" {
		t.Errorf("HALF_EVEN: %q", got)
	}
	f = NewCurrencyInstance(locale("en-US"))
	got, _ = f.Format(.125)
	if got != "$0.12" {
		t.Errorf("HALF_EVEN currency: %q", got)
	}
	got, _ = f.Format(math.Copysign(0, -1))
	if got != "-$0.00" {
		t.Errorf("negative zero: %q", got)
	}
	f, err := NewDecimalFormat("#.00", locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	got, err = f.Format(.45)
	if err != nil || got != ".45" {
		t.Errorf("optional integer = %q, %v", got, err)
	}
	_, err = NewDecimalFormat("0%‰", locale("en-US"))
	if err == nil || err.Error() != "java.lang.IllegalArgumentException: Too many percent/per mille characters in pattern \"0%‰\"" {
		t.Errorf("pattern error = %v", err)
	}
	for _, tc := range []struct{ pattern, want string }{
		{`\U00A4#,##0.00`, `java.lang.IllegalArgumentException: Unexpected '0' in pattern "\U00A4#,##0.00"`},
		{`\U2030#`, `java.lang.IllegalArgumentException: Malformed pattern "\U2030#"`},
		{`0.0.0`, `java.lang.IllegalArgumentException: Multiple decimal separators in pattern "0.0.0"`},
		{`#.#0`, `java.lang.IllegalArgumentException: Malformed pattern "#.#0"`},
	} {
		_, err := NewDecimalFormat(tc.pattern, locale("en-US"))
		if err == nil || err.Error() != tc.want {
			t.Errorf("pattern %q error = %v; want %q", tc.pattern, err, tc.want)
		}
	}
	f, err = NewDecimalFormat("#", locale("tr-TR"))
	if err != nil {
		t.Fatal(err)
	}
	f.SetMinimumFractionDigits(4)
	got, err = f.Format(.30000000000000004)
	if err != nil || got != ",3000" {
		t.Errorf("zero minimum integer digits = %q, %v", got, err)
	}
	f, err = NewDecimalFormat("foo", locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	got, err = f.Format(0)
	if err != nil || got != "foo0" {
		t.Errorf("literal-only pattern = %q, %v", got, err)
	}
	if err := f.SetRoundingMode(" HALF_UP "); err != nil {
		t.Errorf("trimmed rounding mode: %v", err)
	}
	for _, tc := range []struct {
		pattern string
		set     func(*NumberFormat)
		value   any
		want    string
	}{
		{"##0.##E0", func(f *NumberFormat) { f.SetMaximumFractionDigits(1) }, int32(65536), "65.54E3"},
		{"##0.##E0", func(f *NumberFormat) { f.SetMinimumFractionDigits(1) }, .0001, "100E-6"},
		{"0.###E0", func(f *NumberFormat) { f.SetMinimumIntegerDigits(5) }, int32(10000000), "10000E3"},
		{"#,##0", func(*NumberFormat) {}, 1e23, "99,999,999,999,999,990,000,000"},
	} {
		f, err := NewDecimalFormat(tc.pattern, locale("en-US"))
		if err != nil {
			t.Fatal(err)
		}
		tc.set(f)
		got, err := f.Format(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("%s = %q, %v; want %q", tc.pattern, got, err, tc.want)
		}
	}
}

func TestFormatterFromJDK21(t *testing.T) {
	tests := []struct {
		format string
		args   []any
		want   string
	}{
		{"%s %<s", []any{"x"}, "x x"},
		{"%05d", []any{int32(-12)}, "-0012"},
		{"%(,10d", []any{int32(-12)}, "      (12)"},
		{"%#.0f", []any{2.675}, "3."},
		{"%.2g", []any{1.25}, "1.3"},
		{"%a", []any{1.25}, "0x1.4p0"},
		{"%.2g", []any{nil}, "nu"},
	}
	for _, tt := range tests {
		got, err := Format(locale("en-US"), tt.format, tt.args)
		if err != nil || got != tt.want {
			t.Errorf("%s = %q, %v; want %q", tt.format, got, err, tt.want)
		}
	}
	_, err := Format(locale("en-US"), "%d", []any{1.25})
	if err == nil || err.Error() != "java.util.IllegalFormatConversionException: d != java.lang.Double" {
		t.Errorf("conversion error: %v", err)
	}
	_, err = Format(locale("en-US"), "%E%.1d", []any{int64(-2147483649), int64(-2147483649)})
	if err == nil || err.Error() != "java.util.IllegalFormatPrecisionException: 1" {
		t.Errorf("whole-format validation precedence: %v", err)
	}
	_, err = Format(locale("en-US"), "%-10.1f|%#s", []any{"text", "text"})
	if err == nil || err.Error() != "java.util.IllegalFormatConversionException: f != java.lang.String" {
		t.Errorf("Formattable flag validation timing: %v", err)
	}
	decimal, err := jvm.NewBigDecimal("12345678901234567890.125")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Format(locale("en-US"), "%.2f", []any{decimal})
	if err != nil || got != "12345678901234567890.13" {
		t.Errorf("BigDecimal formatter = %q, %v", got, err)
	}
	zeroDecimal, _ := jvm.NewBigDecimal("0.0")
	got, err = Format(locale("en-US"), "%.3g", []any{zeroDecimal})
	if err != nil || got != "0.00e-01" {
		t.Errorf("scaled zero BigDecimal = %q, %v", got, err)
	}
	bigInteger := new(big.Int)
	bigInteger.SetString("-12", 10)
	got, err = Format(locale("en-US"), "%#010x", []any{bigInteger})
	if err != nil || got != "-0x000000c" {
		t.Errorf("BigInteger formatter = %q, %v", got, err)
	}
}

func TestMessageAndDisplayNames(t *testing.T) {
	got, err := MessageFormat("{0,choice,0#no files|1#one file|1<{0,number,integer} files}", locale("en-US"), []any{int32(3)})
	if err != nil || got != "3 files" {
		t.Fatalf("choice = %q, %v", got, err)
	}
	if got := DisplayName(locale("es-AR"), locale("de-DE")); got != "Spanisch (Argentinien)" {
		t.Errorf("display = %q", got)
	}
	date := time.Date(2024, time.March, 20, 12, 34, 56, 0, time.UTC)
	got, err = MessageFormat("{0,date,full} {0,time,full}", locale("de-DE"), []any{date})
	if err != nil || got != "Mittwoch, 20. März 2024 12:34:56 Koordinierte Weltzeit" {
		t.Errorf("localized date = %q, %v", got, err)
	}
	got, err = MessageFormat("{0,date,short}", locale("en-US"), []any{int64(3000000000)})
	if err != nil || got != "2/4/70" {
		t.Errorf("numeric date = %q, %v", got, err)
	}
}

func TestPropertiesBundle(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "messages.properties"), []byte("root = r\nline=a\\\n  b\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "messages_de.properties"), []byte("root=de\nutf=Grüße\n"), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBundle(dir, "messages", locale("de-DE"), locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := b.GetString("utf"); got != "Grüße" {
		t.Errorf("utf = %q", got)
	}
	if got, _ := b.GetString("line"); got != "ab" {
		t.Errorf("continuation = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.properties"), []byte("x=\\u12\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = LoadBundle(dir, "bad", locale("en-US"), locale("en-US"))
	if err == nil || err.Error() != "java.lang.IllegalArgumentException: Malformed \\uxxxx encoding." {
		t.Errorf("malformed unicode error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fallback.properties"), []byte("where=root\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fallback_en_US.properties"), []byte("where=default\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fallback, err := LoadBundle(dir, "fallback", locale("fr-FR"), locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fallback.GetString("where"); got != "default" {
		t.Errorf("default-locale fallback = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "order.properties"), []byte("hello=r\ngreet=g\nquote=q\nch=c\nzeta=z\nalpha=a\nb\\ c=b\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "order_de.properties"), []byte("hello=h\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ordered, err := LoadBundle(dir, "order", locale("de-DE"), locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"hello", "zeta", "quote", "b c", "ch", "alpha", "greet"}
	if got := ordered.Keys(); !slices.Equal(got, wantKeys) {
		t.Errorf("bundle keys = %q; want %q", got, wantKeys)
	}
	if err := os.WriteFile(filepath.Join(dir, "six.properties"), []byte("hello=h\ngreet=g\nch=c\nb\\ c=b\nmulti=m\ndate=d\n"), 0600); err != nil {
		t.Fatal(err)
	}
	six, err := LoadBundle(dir, "six", locale("es-AR"), locale("en-US"))
	if err != nil {
		t.Fatal(err)
	}
	wantSix := []string{"date", "greet", "hello", "b c", "ch", "multi"}
	if got := six.Keys(); !slices.Equal(got, wantSix) {
		t.Errorf("six-key bundle = %q; want %q", got, wantSix)
	}
}
