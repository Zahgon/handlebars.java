// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jtime

import (
	"testing"

	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestParseISOValues(t *testing.T) {
	tests := []struct {
		input, class, text string
		hash               int32
	}{
		{"2024-03-05", "java.time.LocalDate", "2024-03-05", 4145349},
		{"00:00", "java.time.LocalTime", "00:00", 0},
		{"16:07", "java.time.LocalTime", "16:07", -713204540},
		{"2024-03-05T16:07:09.123456789", "java.time.LocalDateTime", "2024-03-05T16:07:09.123456789", -176866538},
		{"1966-02-02T02:26:51.123Z", "java.time.Instant", "1966-02-02T02:26:51.123Z", 2101489492},
		{"2024-03-05T16:07:09.123456789-05:00", "java.time.OffsetDateTime", "2024-03-05T16:07:09.123456789-05:00", 176849574},
		{"2024-03-05T16:07:09.123456789-05:00[America/New_York]", "java.time.ZonedDateTime", "2024-03-05T16:07:09.123456789-05:00[America/New_York]", -1514925861},
	}
	for _, tt := range tests {
		value, err := ParseISO(tt.input)
		if err != nil {
			t.Fatalf("ParseISO(%q): %v", tt.input, err)
		}
		object := value.(jvm.Object)
		if object.JavaClass() != tt.class || object.JavaString() != tt.text {
			t.Errorf("ParseISO(%q) = %s %q, want %s %q", tt.input, object.JavaClass(), object.JavaString(), tt.class, tt.text)
		}
		if got := value.(jvm.Hasher).JavaHashCode(); got != tt.hash {
			t.Errorf("ParseISO(%q).hashCode = %d, want %d", tt.input, got, tt.hash)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct{ input, message string }{
		{"x", "Text 'x' could not be parsed at index 0"},
		{"2024-13-01", "Text '2024-13-01' could not be parsed: Invalid value for MonthOfYear (valid values 1 - 12): 13"},
		{"2024-02-30", "Text '2024-02-30' could not be parsed: Invalid date 'FEBRUARY 30'"},
		{"25:00", "Text '25:00' could not be parsed: Invalid value for HourOfDay (valid values 0 - 23): 25"},
		{"2024-03-05T25:00", "Text '2024-03-05T25:00' could not be parsed: Invalid value for HourOfDay (valid values 0 - 23): 25"},
	}
	for _, tt := range tests {
		_, err := ParseISO(tt.input)
		assertJavaError(t, err, "java.time.format.DateTimeParseException", tt.message)
	}
}

func TestPatternAndLocalizedFormatting(t *testing.T) {
	value, err := ParseISO("2024-03-05T16:07:09.123456789-05:00[America/New_York]")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		pattern, locale, want string
	}{
		{"yyyy-MM-dd HH:mm:ss.SSS XXX VV zzzz OOOO", "en-US", "2024-03-05 16:07:09.123 -05:00 America/New_York Eastern Standard Time GMT-05:00"},
		{"YYYY-'W'ww-e", "en-US", "2024-W10-3"},
		{"EEEE, d. MMMM y", "de-DE", "Dienstag, 5. März 2024"},
		{"ppH pppd", "en-US", "16   5"},
	}
	for _, tt := range tests {
		formatter, e := OfPattern(tt.pattern)
		if e != nil {
			t.Fatal(e)
		}
		got, e := formatter.WithLocale(jvm.ForLanguageTag(tt.locale)).Format(value)
		if e != nil || got != tt.want {
			t.Errorf("pattern %q = %q, %v; want %q", tt.pattern, got, e, tt.want)
		}
	}

	formatter, _ := OfLocalizedDateTime(FullStyle, FullStyle)
	got, err := formatter.WithLocale(jvm.ForLanguageTag("en-US")).Format(value)
	if err != nil || got != "Tuesday, March 5, 2024, 4:07:09\u202fPM Eastern Standard Time" {
		t.Errorf("localized = %q, %v", got, err)
	}
}

func TestFormatterErrorsAndZones(t *testing.T) {
	for pattern, message := range map[string]string{
		"b":  "Unknown pattern letter: b",
		"aa": "Too many pattern letters: a",
		"V":  "Pattern letter count must be 2: V",
		"#":  "Pattern includes reserved character: '#'",
	} {
		_, err := OfPattern(pattern)
		assertJavaError(t, err, "java.lang.IllegalArgumentException", message)
	}

	date, _ := ParseISO("2024-03-05")
	timePattern, _ := OfPattern("HH")
	_, err := timePattern.Format(date)
	assertJavaError(t, err, "java.time.temporal.UnsupportedTemporalTypeException", "Unsupported field: HourOfDay")
	zonePattern, _ := OfPattern("VV")
	_, err = zonePattern.Format(date)
	assertJavaError(t, err, "java.time.DateTimeException", "Unable to extract ZoneId from temporal 2024-03-05")

	if got := ZoneFromTimeZoneID("not-a-zone").ID(); got != "GMT" {
		t.Errorf("unknown TimeZone = %q", got)
	}
	if got := ZoneFromTimeZoneID("GMT+2").ID(); got != "GMT+02:00" {
		t.Errorf("GMT+2 = %q", got)
	}
}

func TestZoneOverrideOnLocalDateTime(t *testing.T) {
	local, _ := ParseISO("2024-06-15T23:59:59.123456789")
	zone, _ := OfZoneID("Asia/Kolkata")
	formatter, _ := OfLocalizedDateTime(MediumStyle, FullStyle)
	got, err := formatter.WithLocale(jvm.ForLanguageTag("es")).WithZone(zone).Format(local)
	if err != nil || got != "15 jun 2024, 23:59:59 (hora estándar de la India)" {
		t.Errorf("localized local date-time with zone = %q, %v", got, err)
	}
	formatter, _ = OfPattern("VV z XXX")
	_, err = formatter.WithZone(zone).Format(local)
	assertJavaError(t, err, "java.time.temporal.UnsupportedTemporalTypeException", "Unsupported field: OffsetSeconds")
	formatter, _ = OfPattern("VV z")
	got, err = formatter.WithZone(zone).Format(local)
	if err != nil || got != "Asia/Kolkata IST" {
		t.Errorf("local zone query = %q, %v", got, err)
	}
	if got := ZoneFromTimeZoneID("PST").ID(); got != "America/Los_Angeles" {
		t.Errorf("PST TimeZone = %q", got)
	}
}

func TestISOFormattersAndInstantFields(t *testing.T) {
	local, _ := ParseISO("2024-03-05T16:07")
	if got, err := ISOLocalDateTime.Format(local); err != nil || got != "2024-03-05T16:07:00" {
		t.Errorf("ISO_LOCAL_DATE_TIME = %q, %v", got, err)
	}
	offset, _ := ParseISO("2024-03-05T16:07:09.100+02:00")
	if got, err := ISOOffsetDateTime.Format(offset); err != nil || got != "2024-03-05T16:07:09.1+02:00" {
		t.Errorf("ISO_OFFSET_DATE_TIME = %q, %v", got, err)
	}
	instant, _ := ParseISO("1970-01-01T00:00:01.123456789Z")
	formatter, _ := OfPattern("S n [XXX]")
	if got, err := formatter.Format(instant); err != nil || got != "1 123456789 " {
		t.Errorf("instant nano fields = %q, %v", got, err)
	}
	zoneOffset, _ := ParseZoneOffset("+01:02:03")
	formatter, _ = OfPattern("X XX XXX XXXX XXXXX O OOOO")
	if got, err := formatter.Format(zoneOffset); err != nil || got != "+0102 +0102 +01:02 +010203 +01:02:03 GMT+1:02:03 GMT+01:02:03" {
		t.Errorf("ZoneOffset formatting = %q, %v", got, err)
	}

	zoned, _ := ParseISO("2024-03-05T16:07:09.123456789-05:00[America/New_York]")
	isoTests := []struct {
		formatter *Formatter
		want      string
	}{
		{BasicISODate, "20240305-0500"},
		{ISOLocalDate, "2024-03-05"},
		{ISOOffsetDate, "2024-03-05-05:00"},
		{ISODate, "2024-03-05-05:00"},
		{ISOLocalTime, "16:07:09.123456789"},
		{ISOOffsetTime, "16:07:09.123456789-05:00"},
		{ISOTime, "16:07:09.123456789-05:00"},
		{ISOLocalDateTime, "2024-03-05T16:07:09.123456789"},
		{ISOOffsetDateTime, "2024-03-05T16:07:09.123456789-05:00"},
		{ISOZonedDateTime, "2024-03-05T16:07:09.123456789-05:00[America/New_York]"},
		{ISODateTime, "2024-03-05T16:07:09.123456789-05:00[America/New_York]"},
		{ISOOrdinalDate, "2024-065-05:00"},
		{ISOWeekDate, "2024-W10-2-05:00"},
		{ISOInstant, "2024-03-05T21:07:09.123456789Z"},
		{RFC1123DateTime, "Tue, 5 Mar 2024 16:07:09 -0500"},
	}
	for _, tt := range isoTests {
		if got, err := tt.formatter.Format(zoned); err != nil || got != tt.want {
			t.Errorf("ISO formatter = %q, %v; want %q", got, err, tt.want)
		}
	}
}

func assertJavaError(t *testing.T, err error, class, message string) {
	t.Helper()
	e, ok := err.(*jvm.Exception)
	if !ok || e.Class != class || e.Message != message {
		t.Errorf("error = %#v, want %s: %s", err, class, message)
	}
}
