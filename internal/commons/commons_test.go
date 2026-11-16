// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package commons

import "testing"

func TestStringUtils(t *testing.T) {
	if got := Center("a", 4, "ab"); got != "aaab" {
		t.Errorf("center=%q", got)
	}
	if got := LeftPad("x", 5, "ab"); got != "ababx" {
		t.Errorf("left=%q", got)
	}
	if got := RightPad("x", 5, "ab"); got != "xabab" {
		t.Errorf("right=%q", got)
	}
	if got, e := Abbreviate("abcdef", 4); e != nil || got != "a..." {
		t.Errorf("abbreviate=%q %v", got, e)
	}
	if got := Capitalize("ǆemal"); got != "ǅemal" {
		t.Errorf("capitalize=%q", got)
	}
	if got := Substring("abcdef", -4, -1); got != "cde" {
		t.Errorf("substring=%q", got)
	}
}

func TestWordAndEscapeUtils(t *testing.T) {
	if got := WordWrap("The quick brown fox", 10, "|", false); got != "The quick|brown fox" {
		t.Errorf("wrap=%q", got)
	}
	if got := WordCapitalizeFully("hELLO wORLD"); got != "Hello World" {
		t.Errorf("capitalize fully=%q", got)
	}
	if got := EscapeEcmaScript("'\"\\/\u0001"); got != "\\'\\\"\\\\\\/\\u0001" {
		t.Errorf("ecma=%q", got)
	}
	if got := EscapeXml11("<&'\u0001\u0085\ufffe"); got != "&lt;&amp;&apos;&#1;\u0085" {
		t.Errorf("xml=%q", got)
	}
	if got := EscapeHTML4("© α\u2003"); got != "&copy; &alpha;&emsp;" {
		t.Errorf("html=%q", got)
	}
	if got := EscapeCsv("a,b\""); got != "\"a,b\"\"\"" {
		t.Errorf("csv=%q", got)
	}
}

func TestNumberAndLocaleUtils(t *testing.T) {
	for _, s := range []string{"0", "0x1f", "1.", ".1", "1e2", "-09d"} {
		if !IsCreatable(s) {
			t.Errorf("%q not creatable", s)
		}
	}
	for _, s := range []string{"", "09", "1e", "1_0"} {
		if IsCreatable(s) {
			t.Errorf("%q creatable", s)
		}
	}
	loc, e := LocaleToLocale("en__WIN")
	if e != nil || loc.String() != "en__WIN" {
		t.Errorf("locale=%v %v", loc, e)
	}
	if _, e := LocaleToLocale("xx_"); e == nil || e.Error() != "java.lang.IllegalArgumentException: Invalid locale format: xx_" {
		t.Errorf("bad locale=%v", e)
	}
}
