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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

var (
	isoDatePattern = regexp.MustCompile(`^([+-]?\d{4,10})-(\d{2})-(\d{2})$`)
	isoTimePattern = regexp.MustCompile(`^(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{0,9}))?)?$`)
)

// ParseISO chooses the same java.time type as the reference command's date helper.
func ParseISO(text string) (any, error) {
	if strings.Contains(text, "[") {
		return parseZonedDateTime(text)
	}
	if strings.HasSuffix(text, "Z") {
		return parseInstant(text)
	}
	if hasFinalOffset(text) {
		return parseOffsetDateTime(text)
	}
	if strings.Contains(text, "T") {
		return parseLocalDateTime(text)
	}
	if strings.Contains(text, ":") {
		return parseLocalTime(text)
	}
	return parseLocalDate(text)
}

func parseLocalDate(text string) (LocalDate, error) {
	m := isoDatePattern.FindStringSubmatch(text)
	if m == nil {
		return LocalDate{}, parseAt(text, firstDateError(text))
	}
	yearText := m[1]
	digits := strings.TrimLeft(yearText, "+-")
	if (yearText[0] != '+' && yearText[0] != '-' && len(digits) != 4) || yearText[0] == '+' && len(digits) <= 4 {
		return LocalDate{}, parseAt(text, 0)
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	if year < -999999999 || year > 999999999 {
		return LocalDate{}, parsedDetail(text, fmt.Sprintf("Invalid value for Year (valid values -999999999 - 999999999): %d", year))
	}
	if month < 1 || month > 12 {
		return LocalDate{}, parsedDetail(text, fmt.Sprintf("Invalid value for MonthOfYear (valid values 1 - 12): %d", month))
	}
	if day < 1 || day > 31 {
		return LocalDate{}, parsedDetail(text, fmt.Sprintf("Invalid value for DayOfMonth (valid values 1 - 28/31): %d", day))
	}
	if day > lengthOfMonth(year, month) {
		return LocalDate{}, parsedDetail(text, fmt.Sprintf("Invalid date '%s %d'", strings.ToUpper(monthEnglish[month]), day))
	}
	return LocalDate{Year: year, Month: month, Day: day}, nil
}

func parseLocalTime(text string) (LocalTime, error) {
	m := isoTimePattern.FindStringSubmatch(text)
	if m == nil {
		if dot := strings.IndexByte(text, '.'); dot >= 0 && len(text)-dot-1 > 9 {
			return LocalTime{}, jvm.Throw("java.time.format.DateTimeParseException", fmt.Sprintf("Text '%s' could not be parsed, unparsed text found at index %d", text, dot+10))
		}
		return LocalTime{}, parseAt(text, firstTimeError(text))
	}
	hour, _ := strconv.Atoi(m[1])
	minute, _ := strconv.Atoi(m[2])
	second := 0
	if m[3] != "" {
		second, _ = strconv.Atoi(m[3])
	}
	nano := 0
	if m[4] != "" {
		fraction := m[4] + strings.Repeat("0", 9-len(m[4]))
		nano, _ = strconv.Atoi(fraction)
	}
	if hour > 23 {
		return LocalTime{}, parsedDetail(text, fmt.Sprintf("Invalid value for HourOfDay (valid values 0 - 23): %d", hour))
	}
	if minute > 59 {
		return LocalTime{}, parsedDetail(text, fmt.Sprintf("Invalid value for MinuteOfHour (valid values 0 - 59): %d", minute))
	}
	if second > 59 {
		return LocalTime{}, parsedDetail(text, fmt.Sprintf("Invalid value for SecondOfMinute (valid values 0 - 59): %d", second))
	}
	return LocalTime{Hour: hour, Minute: minute, Second: second, Nano: nano}, nil
}

func parseLocalDateTime(text string) (LocalDateTime, error) {
	i := strings.IndexByte(text, 'T')
	if i < 0 {
		return LocalDateTime{}, parseAt(text, firstDateError(text))
	}
	d, err := parseLocalDate(text[:i])
	if err != nil {
		return LocalDateTime{}, rewrapParse(err, text)
	}
	t, err := parseLocalTime(text[i+1:])
	if err != nil {
		return LocalDateTime{}, shiftParse(err, text, i+1)
	}
	return LocalDateTime{Date: d, Time: t}, nil
}

func parseOffsetDateTime(text string) (OffsetDateTime, error) {
	pos := finalOffsetPosition(text)
	if pos < 0 {
		return OffsetDateTime{}, parseAt(text, len(text))
	}
	dt, err := parseLocalDateTime(text[:pos])
	if err != nil {
		return OffsetDateTime{}, rewrapParse(err, text)
	}
	offset, err := ParseZoneOffset(text[pos:])
	if err != nil {
		if e, ok := err.(*jvm.Exception); ok {
			return OffsetDateTime{}, parsedDetail(text, e.Message)
		}
		return OffsetDateTime{}, err
	}
	return OffsetDateTime{DateTime: dt, Offset: offset}, nil
}

func parseInstant(text string) (Instant, error) {
	if !strings.HasSuffix(text, "Z") {
		return Instant{}, parseAt(text, len(text))
	}
	withoutZone := strings.TrimSuffix(text, "Z")
	if t := strings.LastIndexByte(withoutZone, 'T'); t >= 0 && strings.Count(withoutZone[t+1:], ":") < 2 {
		return Instant{}, parseAt(text, len(text)-1)
	}
	dt, err := parseLocalDateTime(withoutZone)
	if err != nil {
		return Instant{}, rewrapParse(err, text)
	}
	return NewInstant(dt.epochSecond(0), dt.Time.Nano)
}

func parseZonedDateTime(text string) (ZonedDateTime, error) {
	open := strings.LastIndexByte(text, '[')
	if open < 0 || !strings.HasSuffix(text, "]") {
		return ZonedDateTime{}, parseAt(text, len(text))
	}
	zone, err := OfZoneID(text[open+1 : len(text)-1])
	if err != nil {
		return ZonedDateTime{}, err
	}
	prefix := text[:open]
	if strings.HasSuffix(prefix, "Z") {
		dt, parseErr := parseLocalDateTime(strings.TrimSuffix(prefix, "Z"))
		if parseErr != nil {
			return ZonedDateTime{}, rewrapParse(parseErr, text)
		}
		o := OffsetDateTime{DateTime: dt, Offset: ZoneOffset{}}
		z := ZonedDateTimeOfInstant(o.ToInstant(), zone)
		if z.Offset.TotalSeconds == 0 {
			z.DateTime = dt
		}
		return z, nil
	}
	if hasFinalOffset(prefix) {
		o, err := parseOffsetDateTime(prefix)
		if err != nil {
			return ZonedDateTime{}, rewrapParse(err, text)
		}
		// ZonedDateTime.parse resolves by the parsed instant if the stated offset is
		// not valid in the named zone.
		z := ZonedDateTimeOfInstant(o.ToInstant(), zone)
		if z.Offset == o.Offset {
			z.DateTime = o.DateTime
		}
		return z, nil
	}
	return ZonedDateTime{}, parseAt(text, open)
}

func hasFinalOffset(text string) bool { return finalOffsetPosition(text) >= 0 }

func finalOffsetPosition(text string) int {
	t := strings.LastIndexByte(text, 'T')
	if t < 0 {
		return -1
	}
	for i := len(text) - 1; i > t; i-- {
		if (text[i] == '+' || text[i] == '-') && i+6 == len(text) && text[i+3] == ':' {
			return i
		}
	}
	return -1
}

func firstDateError(text string) int {
	if text == "" {
		return 0
	}
	if len(text) < 4 || !allDigits(text[:min(4, len(text))]) {
		return 0
	}
	if len(text) < 5 || text[4] != '-' {
		return 4
	}
	if len(text) < 7 || !allDigits(text[5:min(7, len(text))]) {
		return 5
	}
	if len(text) < 8 || text[7] != '-' {
		return 7
	}
	if len(text) < 10 || !allDigits(text[8:min(10, len(text))]) {
		return 8
	}
	return 0
}

func firstTimeError(text string) int {
	if text == "" {
		return 0
	}
	if len(text) < 2 || text[0] < '0' || text[0] > '9' || text[1] < '0' || text[1] > '9' {
		return 0
	}
	if len(text) < 3 || text[2] != ':' {
		return 2
	}
	if len(text) < 5 || !allDigits(text[3:min(5, len(text))]) {
		return 3
	}
	return len(text)
}

func allDigits(s string) bool {
	for i := range s {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func parseAt(text string, index int) error {
	return jvm.Throw("java.time.format.DateTimeParseException", fmt.Sprintf("Text '%s' could not be parsed at index %d", text, index))
}

func parsedDetail(text, detail string) error {
	return jvm.Throw("java.time.format.DateTimeParseException", fmt.Sprintf("Text '%s' could not be parsed: %s", text, detail))
}

func rewrapParse(err error, full string) error {
	e, ok := err.(*jvm.Exception)
	if !ok {
		return err
	}
	msg := e.Message
	if i := strings.Index(msg, " could not be parsed"); i >= 0 {
		if q := strings.Index(msg, "Text '"); q == 0 {
			msg = "Text '" + full + "'" + msg[i:]
		}
	}
	return jvm.Throw(e.Class, msg)
}

func shiftParse(err error, full string, shift int) error {
	e, ok := err.(*jvm.Exception)
	if !ok {
		return err
	}
	msg := e.Message
	if marker := " could not be parsed at index "; strings.Contains(msg, marker) {
		parts := strings.Split(msg, marker)
		n, _ := strconv.Atoi(parts[len(parts)-1])
		return parseAt(full, shift+n)
	}
	return rewrapParse(err, full)
}
