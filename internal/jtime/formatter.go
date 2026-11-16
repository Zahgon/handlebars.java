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
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// FormatStyle names a JDK localized formatter style.
type FormatStyle string

const (
	// FullStyle is FormatStyle.FULL.
	FullStyle FormatStyle = "full"
	// LongStyle is FormatStyle.LONG.
	LongStyle FormatStyle = "long"
	// MediumStyle is FormatStyle.MEDIUM.
	MediumStyle FormatStyle = "medium"
	// ShortStyle is FormatStyle.SHORT.
	ShortStyle FormatStyle = "short"
)

type token struct {
	kind     byte
	count    int
	literal  string
	optional []token
	pad      int
}

// Formatter is an immutable JDK-compatible DateTimeFormatter.
type Formatter struct {
	pattern   string
	tokens    []token
	locale    jvm.Locale
	zone      ZoneID
	hasZone   bool
	dateStyle FormatStyle
	timeStyle FormatStyle
	localized bool
	iso       isoFormatter
}

type isoFormatter uint8

const (
	isoNone isoFormatter = iota
	isoBasicDate
	isoLocalDate
	isoOffsetDate
	isoDate
	isoLocalTime
	isoOffsetTime
	isoTime
	isoLocalDateTime
	isoOffsetDateTime
	isoZonedDateTime
	isoDateTime
	isoOrdinalDate
	isoWeekDate
	isoInstant
	isoRFC1123
)

// BasicISODate is DateTimeFormatter.BASIC_ISO_DATE.
var BasicISODate = &Formatter{locale: jvm.Default(), iso: isoBasicDate}

// ISOLocalDate is DateTimeFormatter.ISO_LOCAL_DATE.
var ISOLocalDate = &Formatter{locale: jvm.Default(), iso: isoLocalDate}

// ISOOffsetDate is DateTimeFormatter.ISO_OFFSET_DATE.
var ISOOffsetDate = &Formatter{locale: jvm.Default(), iso: isoOffsetDate}

// ISODate is DateTimeFormatter.ISO_DATE.
var ISODate = &Formatter{locale: jvm.Default(), iso: isoDate}

// ISOLocalTime is DateTimeFormatter.ISO_LOCAL_TIME.
var ISOLocalTime = &Formatter{locale: jvm.Default(), iso: isoLocalTime}

// ISOOffsetTime is DateTimeFormatter.ISO_OFFSET_TIME.
var ISOOffsetTime = &Formatter{locale: jvm.Default(), iso: isoOffsetTime}

// ISOTime is DateTimeFormatter.ISO_TIME.
var ISOTime = &Formatter{locale: jvm.Default(), iso: isoTime}

// ISOLocalDateTime is DateTimeFormatter.ISO_LOCAL_DATE_TIME.
var ISOLocalDateTime = &Formatter{locale: jvm.Default(), iso: isoLocalDateTime}

// ISOOffsetDateTime is DateTimeFormatter.ISO_OFFSET_DATE_TIME.
var ISOOffsetDateTime = &Formatter{locale: jvm.Default(), iso: isoOffsetDateTime}

// ISOZonedDateTime is DateTimeFormatter.ISO_ZONED_DATE_TIME.
var ISOZonedDateTime = &Formatter{locale: jvm.Default(), iso: isoZonedDateTime}

// ISODateTime is DateTimeFormatter.ISO_DATE_TIME.
var ISODateTime = &Formatter{locale: jvm.Default(), iso: isoDateTime}

// ISOOrdinalDate is DateTimeFormatter.ISO_ORDINAL_DATE.
var ISOOrdinalDate = &Formatter{locale: jvm.Default(), iso: isoOrdinalDate}

// ISOWeekDate is DateTimeFormatter.ISO_WEEK_DATE.
var ISOWeekDate = &Formatter{locale: jvm.Default(), iso: isoWeekDate}

// ISOInstant is DateTimeFormatter.ISO_INSTANT.
var ISOInstant = &Formatter{locale: jvm.Default(), iso: isoInstant}

// RFC1123DateTime is DateTimeFormatter.RFC_1123_DATE_TIME.
var RFC1123DateTime = &Formatter{locale: jvm.Default(), iso: isoRFC1123}

// OfPattern creates a formatter from a JDK DateTimeFormatter pattern.
func OfPattern(pattern string) (*Formatter, error) {
	tokens, pos, err := parseTokens(pattern, 0, false)
	if err != nil {
		return nil, err
	}
	if pos != len(pattern) {
		return nil, illegalArgument("Pattern invalid as it contains ] without previous [")
	}
	return &Formatter{pattern: pattern, tokens: tokens, locale: jvm.Default()}, nil
}

// OfLocalizedDate creates a localized date formatter.
func OfLocalizedDate(style FormatStyle) (*Formatter, error) {
	if !validStyle(style) {
		return nil, illegalArgument("Unknown FormatStyle: " + string(style))
	}
	return &Formatter{localized: true, dateStyle: style, locale: jvm.Default()}, nil
}

// OfLocalizedDateTime creates a localized date-time formatter.
func OfLocalizedDateTime(dateStyle, timeStyle FormatStyle) (*Formatter, error) {
	if !validStyle(dateStyle) {
		return nil, illegalArgument("Unknown FormatStyle: " + string(dateStyle))
	}
	if !validStyle(timeStyle) {
		return nil, illegalArgument("Unknown FormatStyle: " + string(timeStyle))
	}
	return &Formatter{localized: true, dateStyle: dateStyle, timeStyle: timeStyle, locale: jvm.Default()}, nil
}

func validStyle(s FormatStyle) bool {
	return s == FullStyle || s == LongStyle || s == MediumStyle || s == ShortStyle
}

// WithLocale returns a copy with the requested locale.
func (f *Formatter) WithLocale(locale jvm.Locale) *Formatter {
	c := *f
	c.locale = locale
	return &c
}

// WithZone returns a copy with an override zone.
func (f *Formatter) WithZone(zone ZoneID) *Formatter {
	c := *f
	c.zone = zone
	c.hasZone = true
	return &c
}

// Format formats a supported java.time value.
func (f *Formatter) Format(value any) (string, error) {
	fields, err := fieldsOf(value, f)
	if err != nil {
		return "", err
	}
	if f.localized {
		return f.formatLocalized(fields)
	}
	if f.iso != isoNone {
		return f.formatISO(fields)
	}
	return formatTokens(f.tokens, fields, f.locale)
}

func (f *Formatter) formatISO(x temporalFields) (string, error) {
	date := func() (string, error) {
		if x.date == nil {
			return "", unsupported("Year")
		}
		return x.date.JavaString(), nil
	}
	tm := func() (string, error) {
		if x.time == nil {
			return "", unsupported("HourOfDay")
		}
		return isoTimeString(*x.time), nil
	}
	offset := func() (string, error) {
		if x.offset == nil {
			return "", unsupported("OffsetSeconds")
		}
		return ZoneOffset{TotalSeconds: *x.offset}.JavaString(), nil
	}
	switch f.iso {
	case isoBasicDate:
		if x.date == nil {
			return "", unsupported("Year")
		}
		s := fmt.Sprintf("%s%02d%02d", formatYear(x.date.Year, 4), x.date.Month, x.date.Day)
		if x.offset != nil {
			s += formatOffsetPattern(*x.offset, 'X', 2, f.locale)
		}
		return s, nil
	case isoLocalDate:
		return date()
	case isoOffsetDate:
		d, err := date()
		if err != nil {
			return "", err
		}
		o, err := offset()
		return d + o, err
	case isoDate:
		s, err := date()
		if err == nil && x.offset != nil {
			o, _ := offset()
			s += o
		}
		return s, err
	case isoLocalTime:
		return tm()
	case isoOffsetTime:
		t, err := tm()
		if err != nil {
			return "", err
		}
		o, err := offset()
		return t + o, err
	case isoTime:
		s, err := tm()
		if err == nil && x.offset != nil {
			o, _ := offset()
			s += o
		}
		return s, err
	case isoLocalDateTime:
		d, err := date()
		if err != nil {
			return "", err
		}
		t, err := tm()
		return d + "T" + t, err
	case isoOffsetDateTime:
		dt, err := (&Formatter{iso: isoLocalDateTime}).formatISO(x)
		if err != nil {
			return "", err
		}
		o, err := offset()
		return dt + o, err
	case isoZonedDateTime:
		if x.zone == nil {
			return "", cannotExtractZone(x.display)
		}
		dt, err := (&Formatter{iso: isoOffsetDateTime}).formatISO(x)
		if err != nil {
			return "", err
		}
		if x.zone.id != (ZoneOffset{TotalSeconds: *x.offset}).JavaString() {
			dt += "[" + x.zone.id + "]"
		}
		return dt, nil
	case isoDateTime:
		dt, err := (&Formatter{iso: isoLocalDateTime}).formatISO(x)
		if err != nil {
			return "", err
		}
		if x.offset != nil {
			o, _ := offset()
			dt += o
		}
		if x.zone != nil && x.zone.id != (ZoneOffset{TotalSeconds: *x.offset}).JavaString() {
			dt += "[" + x.zone.id + "]"
		}
		return dt, nil
	case isoOrdinalDate:
		if x.date == nil {
			return "", unsupported("Year")
		}
		s := fmt.Sprintf("%s-%03d", formatYear(x.date.Year, 4), dayOfYear(*x.date))
		if x.offset != nil {
			o, _ := offset()
			s += o
		}
		return s, nil
	case isoWeekDate:
		if x.date == nil {
			return "", unsupported("WeekBasedYear")
		}
		y, w := localizedWeek(*x.date, localeData{firstDay: 1, minimalDays: 4})
		s := fmt.Sprintf("%s-W%02d-%d", formatYear(y, 4), w, isoDayOfWeek(*x.date))
		if x.offset != nil {
			o, _ := offset()
			s += o
		}
		return s, nil
	case isoInstant:
		if x.instant == nil {
			return "", unsupported("InstantSeconds")
		}
		return x.instant.JavaString(), nil
	case isoRFC1123:
		if x.date == nil {
			return "", unsupported("DayOfMonth")
		}
		if x.time == nil {
			return "", unsupported("HourOfDay")
		}
		if x.offset == nil {
			return "", unsupported("OffsetSeconds")
		}
		zone := "GMT"
		if *x.offset != 0 {
			zone = formatOffsetPattern(*x.offset, 'Z', 1, jvm.Locale{Language: "en"})
		}
		en := locales["en"]
		return fmt.Sprintf("%s, %d %s %04d %02d:%02d:%02d %s", en.daysShort[isoDayOfWeek(*x.date)-1], x.date.Day, en.monthsShort[x.date.Month-1], x.date.Year, x.time.Hour, x.time.Minute, x.time.Second, zone), nil
	}
	return "", nil
}

func isoTimeString(t LocalTime) string {
	s := fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, t.Second)
	if t.Nano != 0 {
		fraction := strings.TrimRight(fmt.Sprintf("%09d", t.Nano), "0")
		s += "." + fraction
	}
	return s
}

func parseTokens(pattern string, pos int, nested bool) ([]token, int, error) {
	result := make([]token, 0, len(pattern)/2+1)
	for pos < len(pattern) {
		c := pattern[pos]
		if c == ']' {
			if !nested {
				return nil, pos, illegalArgument("Pattern invalid as it contains ] without previous [")
			}
			return result, pos + 1, nil
		}
		if c == '[' {
			children, next, err := parseTokens(pattern, pos+1, true)
			if err != nil {
				return nil, pos, err
			}
			result = append(result, token{kind: '[', optional: children})
			pos = next
			continue
		}
		if c == '\'' {
			start := pos
			if pos+1 < len(pattern) && pattern[pos+1] == '\'' {
				result = append(result, token{literal: "'"})
				pos += 2
				continue
			}
			pos++
			var b strings.Builder
			closed := false
			for pos < len(pattern) {
				if pattern[pos] == '\'' {
					if pos+1 < len(pattern) && pattern[pos+1] == '\'' {
						b.WriteByte('\'')
						pos += 2
						continue
					}
					pos++
					closed = true
					break
				}
				b.WriteByte(pattern[pos])
				pos++
			}
			if !closed {
				return nil, start, illegalArgument("Pattern ends with an incomplete string literal: " + pattern)
			}
			result = append(result, token{literal: b.String()})
			continue
		}
		if c == '#' || c == '{' || c == '}' {
			return nil, pos, illegalArgument(fmt.Sprintf("Pattern includes reserved character: '%c'", c))
		}
		if isASCIILetter(c) {
			start := pos
			for pos < len(pattern) && pattern[pos] == c {
				pos++
			}
			count := pos - start
			if c == 'p' {
				if pos >= len(pattern) || !isASCIILetter(pattern[pos]) || pattern[pos] == 'p' {
					return nil, start, illegalArgument("Pad letter 'p' must be followed by valid pad pattern: " + pattern)
				}
				nextStart := pos
				nextChar := pattern[pos]
				for pos < len(pattern) && pattern[pos] == nextChar {
					pos++
				}
				nextCount := pos - nextStart
				if err := validatePatternLetter(nextChar, nextCount); err != nil {
					return nil, nextStart, err
				}
				result = append(result, token{kind: nextChar, count: nextCount, pad: count})
				continue
			}
			if err := validatePatternLetter(c, count); err != nil {
				return nil, start, err
			}
			result = append(result, token{kind: c, count: count})
			continue
		}
		start := pos
		for pos < len(pattern) && pattern[pos] != '[' && pattern[pos] != ']' && pattern[pos] != '\'' && !isASCIILetter(pattern[pos]) && pattern[pos] != '#' && pattern[pos] != '{' && pattern[pos] != '}' {
			pos++
		}
		result = append(result, token{literal: pattern[start:pos]})
	}
	return result, pos, nil
}

func validatePatternLetter(c byte, count int) error {
	tooMany := func(max int) error {
		if count > max {
			return illegalArgument(fmt.Sprintf("Too many pattern letters: %c", c))
		}
		return nil
	}
	switch c {
	case 'G', 'M', 'L', 'Q', 'q', 'E', 'e':
		return tooMany(5)
	case 'c':
		if count == 2 {
			return illegalArgument(`Invalid pattern "cc"`)
		}
		return tooMany(5)
	case 'D':
		return tooMany(3)
	case 'd', 'h', 'H', 'K', 'k', 'm', 's', 'w':
		return tooMany(2)
	case 'W', 'F', 'a':
		return tooMany(1)
	case 'S':
		if count > 9 {
			return illegalArgument(fmt.Sprintf("Minimum width must be from 0 to 9 inclusive but was %d", count))
		}
	case 'V':
		if count != 2 {
			return illegalArgument("Pattern letter count must be 2: V")
		}
	case 'v':
		if count != 1 && count != 4 {
			return illegalArgument("Wrong number of pattern letters: v")
		}
	case 'z':
		return tooMany(4)
	case 'O':
		if count != 1 && count != 4 {
			return illegalArgument("Pattern letter count must be 1 or 4: O")
		}
	case 'X', 'x', 'Z':
		return tooMany(5)
	case 'B':
		if count != 1 && count != 4 && count != 5 {
			return illegalArgument("Wrong number of pattern letters: B")
		}
	case 'u', 'y', 'Y', 'A', 'n', 'N', 'g':
		// JDK accepts arbitrary positive widths (subject to builder limits far
		// above the pattern sizes accepted by the CLI).
	default:
		return illegalArgument(fmt.Sprintf("Unknown pattern letter: %c", c))
	}
	return nil
}

func isASCIILetter(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

type temporalFields struct {
	date    *LocalDate
	time    *LocalTime
	offset  *int
	zone    *ZoneID
	instant *Instant
	display string
}

func fieldsOf(v any, f *Formatter) (temporalFields, error) {
	var x temporalFields
	switch value := v.(type) {
	case LocalDate:
		x.date, x.display = &value, value.JavaString()
	case *LocalDate:
		if value != nil {
			x.date, x.display = value, value.JavaString()
		}
	case LocalTime:
		x.time, x.display = &value, value.JavaString()
	case *LocalTime:
		if value != nil {
			x.time, x.display = value, value.JavaString()
		}
	case LocalDateTime:
		d, t := value.Date, value.Time
		x.date, x.time, x.display = &d, &t, value.JavaString()
	case *LocalDateTime:
		if value != nil {
			d, t := value.Date, value.Time
			x.date, x.time, x.display = &d, &t, value.JavaString()
		}
	case OffsetDateTime:
		x = fieldsFromOffset(value)
	case *OffsetDateTime:
		if value != nil {
			x = fieldsFromOffset(*value)
		}
	case ZonedDateTime:
		x = fieldsFromZoned(value)
	case *ZonedDateTime:
		if value != nil {
			x = fieldsFromZoned(*value)
		}
	case Instant:
		i := value
		x.instant, x.display = &i, value.JavaString()
	case *Instant:
		if value != nil {
			x.instant, x.display = value, value.JavaString()
		}
	case ZoneOffset:
		off := value.TotalSeconds
		x.offset, x.display = &off, value.JavaString()
	case *ZoneOffset:
		if value != nil {
			off := value.TotalSeconds
			x.offset, x.display = &off, value.JavaString()
		}
	default:
		return temporalFields{}, jvm.Throw("java.time.DateTimeException", "Unable to obtain TemporalAccessor from TemporalAccessor: "+jvm.ToString(v)+" of type "+jvm.ClassName(v))
	}
	if x.display == "" {
		return temporalFields{}, jvm.Throw("java.lang.NullPointerException", "temporal")
	}
	if f.hasZone && x.instant != nil {
		z := ZonedDateTimeOfInstant(*x.instant, f.zone)
		d, t, off, zone := z.DateTime.Date, z.DateTime.Time, z.Offset.TotalSeconds, z.Zone
		x.date, x.time, x.offset, x.zone = &d, &t, &off, &zone
	} else if f.hasZone {
		// An override ZoneId is visible through temporal queries even when a
		// local value cannot supply OffsetSeconds or an instant.
		zone := f.zone
		x.zone = &zone
	} else if x.instant != nil && x.date == nil {
		// An Instant alone has only INSTANT_SECONDS and NANO_OF_SECOND.
	}
	return x, nil
}

func fieldsFromOffset(v OffsetDateTime) temporalFields {
	d, t, off, i := v.DateTime.Date, v.DateTime.Time, v.Offset.TotalSeconds, v.ToInstant()
	return temporalFields{date: &d, time: &t, offset: &off, instant: &i, display: v.JavaString()}
}

func fieldsFromZoned(v ZonedDateTime) temporalFields {
	d, t, off, z, i := v.DateTime.Date, v.DateTime.Time, v.Offset.TotalSeconds, v.Zone, v.ToInstant()
	return temporalFields{date: &d, time: &t, offset: &off, zone: &z, instant: &i, display: v.JavaString()}
}

func formatTokens(tokens []token, x temporalFields, loc jvm.Locale) (string, error) {
	var b strings.Builder
	for _, tok := range tokens {
		if tok.kind == 0 {
			b.WriteString(tok.literal)
			continue
		}
		if tok.kind == '[' {
			s, err := formatTokens(tok.optional, x, loc)
			if err == nil {
				b.WriteString(s)
			} else if !isMissingField(err) {
				return "", err
			}
			continue
		}
		s, err := formatField(tok, x, loc)
		if err != nil {
			return "", err
		}
		if tok.pad > 0 {
			width := utf8.RuneCountInString(s)
			if width > tok.pad {
				return "", jvm.Throw("java.time.DateTimeException", fmt.Sprintf("Cannot print as output of %d characters exceeds pad width of %d", width, tok.pad))
			}
			s = strings.Repeat(" ", tok.pad-width) + s
		}
		b.WriteString(s)
	}
	return b.String(), nil
}

func formatField(tok token, x temporalFields, loc jvm.Locale) (string, error) {
	c, n := tok.kind, tok.count
	requireDate := func(field string) (*LocalDate, error) {
		if x.date == nil {
			return nil, unsupported(field)
		}
		return x.date, nil
	}
	requireTime := func(field string) (*LocalTime, error) {
		if x.time == nil {
			return nil, unsupported(field)
		}
		return x.time, nil
	}
	switch c {
	case 'G':
		d, err := requireDate("Era")
		if err != nil {
			return "", err
		}
		return textEra(loc, d.Year, n), nil
	case 'u', 'y':
		d, err := requireDate(map[byte]string{'u': "Year", 'y': "YearOfEra"}[c])
		if err != nil {
			return "", err
		}
		year := d.Year
		if c == 'y' && year <= 0 {
			year = 1 - year
		}
		return formatYear(year, n), nil
	case 'D':
		d, err := requireDate("DayOfYear")
		if err != nil {
			return "", err
		}
		return number(dayOfYear(*d), n, 0), nil
	case 'M', 'L':
		d, err := requireDate("MonthOfYear")
		if err != nil {
			return "", err
		}
		if n >= 3 {
			return textMonth(loc, d.Month, n, c == 'L'), nil
		}
		return number(d.Month, n, 0), nil
	case 'd':
		d, err := requireDate("DayOfMonth")
		if err != nil {
			return "", err
		}
		return number(d.Day, n, 0), nil
	case 'Q', 'q':
		d, err := requireDate("QuarterOfYear")
		if err != nil {
			return "", err
		}
		q := (d.Month-1)/3 + 1
		if n >= 3 {
			return textQuarter(loc, q, n, c == 'q'), nil
		}
		return number(q, n, 0), nil
	case 'Y':
		d, err := requireDate("DayOfWeek")
		if err != nil {
			return "", err
		}
		y, _ := localizedWeek(*d, dataFor(loc))
		return formatYear(y, n), nil
	case 'w':
		d, err := requireDate("DayOfWeek")
		if err != nil {
			return "", err
		}
		_, w := localizedWeek(*d, dataFor(loc))
		return number(w, n, 0), nil
	case 'W':
		d, err := requireDate("DayOfWeek")
		if err != nil {
			return "", err
		}
		return strconv.Itoa(weekOfMonth(*d, dataFor(loc))), nil
	case 'E':
		d, err := requireDate("DayOfWeek")
		if err != nil {
			return "", err
		}
		return textDay(loc, isoDayOfWeek(*d), n, false), nil
	case 'e', 'c':
		d, err := requireDate("DayOfWeek")
		if err != nil {
			return "", err
		}
		if n >= 3 {
			return textDay(loc, isoDayOfWeek(*d), n, c == 'c'), nil
		}
		ld := localizedDay(isoDayOfWeek(*d), dataFor(loc).firstDay)
		return number(ld, n, 0), nil
	case 'F':
		d, err := requireDate("AlignedWeekOfMonth")
		if err != nil {
			return "", err
		}
		return strconv.Itoa((d.Day-1)/7 + 1), nil
	case 'a':
		t, err := requireTime("AmPmOfDay")
		if err != nil {
			return "", err
		}
		return dataFor(loc).ampm[t.Hour/12], nil
	case 'B':
		t, err := requireTime("HourOfDay")
		if err != nil {
			return "", err
		}
		return textDayPeriod(loc, t.Hour, t.Minute, n), nil
	case 'h', 'K', 'k', 'H':
		t, err := requireTime(map[byte]string{'h': "ClockHourOfAmPm", 'K': "HourOfAmPm", 'k': "ClockHourOfDay", 'H': "HourOfDay"}[c])
		if err != nil {
			return "", err
		}
		v := t.Hour
		switch c {
		case 'h':
			v %= 12
			if v == 0 {
				v = 12
			}
		case 'K':
			v %= 12
		case 'k':
			if v == 0 {
				v = 24
			}
		}
		return number(v, n, 0), nil
	case 'm':
		t, err := requireTime("MinuteOfHour")
		if err != nil {
			return "", err
		}
		return number(t.Minute, n, 0), nil
	case 's':
		t, err := requireTime("SecondOfMinute")
		if err != nil {
			return "", err
		}
		return number(t.Second, n, 0), nil
	case 'S':
		nano := 0
		if x.time != nil {
			nano = x.time.Nano
		} else if x.instant != nil {
			nano = x.instant.Nano
		} else {
			return "", unsupported("NanoOfSecond")
		}
		return fmt.Sprintf("%09d", nano)[:n], nil
	case 'A':
		t, err := requireTime("MilliOfDay")
		if err != nil {
			return "", err
		}
		return number64(t.nanoOfDay()/1000000, n), nil
	case 'n':
		nano := 0
		if x.time != nil {
			nano = x.time.Nano
		} else if x.instant != nil {
			nano = x.instant.Nano
		} else {
			return "", unsupported("NanoOfSecond")
		}
		return number(nano, n, 0), nil
	case 'N':
		t, err := requireTime("NanoOfDay")
		if err != nil {
			return "", err
		}
		return number64(t.nanoOfDay(), n), nil
	case 'g':
		d, err := requireDate("ModifiedJulianDay")
		if err != nil {
			return "", err
		}
		return number64(d.toEpochDay()+40587, n), nil
	case 'V':
		if x.zone == nil {
			return "", cannotExtractZone(x.display)
		}
		return x.zone.id, nil
	case 'z', 'v':
		if x.zone == nil {
			return "", cannotExtractZone(x.display)
		}
		return zoneName(*x.zone, zoneNameInstant(x), c, n, loc), nil
	case 'O':
		if x.offset == nil {
			return "", unsupported("OffsetSeconds")
		}
		return localizedOffset(*x.offset, n, loc), nil
	case 'X', 'x', 'Z':
		if x.offset == nil {
			return "", unsupported("OffsetSeconds")
		}
		return formatOffsetPattern(*x.offset, c, n, loc), nil
	}
	return "", nil
}

func zoneNameInstant(x temporalFields) *Instant {
	if x.instant != nil {
		return x.instant
	}
	if x.date == nil || x.time == nil || x.zone == nil || x.zone.location == nil {
		return nil
	}
	i := x.zone.instantAtLocal(*x.date, *x.time)
	return &i
}

func number(v, min, _ int) string { return fmt.Sprintf("%0*d", min, v) }

func number64(v int64, min int) string { return fmt.Sprintf("%0*d", min, v) }

func formatYear(year, count int) string {
	if count == 2 {
		return fmt.Sprintf("%02d", floorMod(year, 100))
	}
	abs := year
	if abs < 0 {
		abs = -abs
	}
	digits := strconv.Itoa(abs)
	if len(digits) < count {
		digits = strings.Repeat("0", count-len(digits)) + digits
	}
	if year < 0 {
		return "-" + digits
	}
	if count >= 4 && len(digits) > count {
		return "+" + digits
	}
	return digits
}

func dayOfYear(d LocalDate) int {
	n := d.Day
	for m := 1; m < d.Month; m++ {
		n += lengthOfMonth(d.Year, m)
	}
	return n
}

func isoDayOfWeek(d LocalDate) int { return int(floorMod64(d.toEpochDay()+3, 7)) + 1 }

func localizedDay(iso, first int) int { return floorMod(iso-first, 7) + 1 }

func firstWeekStart(year int, data localeData) int64 {
	jan1 := LocalDate{Year: year, Month: 1, Day: 1}.toEpochDay()
	ld := localizedDay(isoDayOfWeek(LocalDate{Year: year, Month: 1, Day: 1}), data.firstDay)
	start := jan1 - int64(ld-1)
	if 8-ld < data.minimalDays {
		start += 7
	}
	return start
}

func localizedWeek(d LocalDate, data localeData) (int, int) {
	epoch := d.toEpochDay()
	ld := localizedDay(isoDayOfWeek(d), data.firstDay)
	start := epoch - int64(ld-1)
	anchor := dateFromEpochDay(start + int64(7-data.minimalDays))
	year := anchor.Year
	return year, int((start-firstWeekStart(year, data))/7) + 1
}

func weekOfMonth(d LocalDate, data localeData) int {
	ld := localizedDay(isoDayOfWeek(d), data.firstDay)
	weekStart := floorMod(d.Day-ld, 7)
	offset := -weekStart
	if weekStart+1 > data.minimalDays {
		offset = 7 - weekStart
	}
	return (7 + offset + d.Day - 1) / 7
}

func formatOffsetPattern(offset int, kind byte, count int, loc jvm.Locale) string {
	if kind == 'Z' && count == 4 {
		return localizedOffset(offset, 4, loc)
	}
	zeroZ := kind == 'X' || kind == 'Z' && count == 5
	if offset == 0 && zeroZ {
		return "Z"
	}
	sign := '+'
	value := offset
	if value < 0 {
		sign, value = '-', -value
	}
	h, m, s := value/3600, value/60%60, value%60
	switch {
	case kind == 'Z' && count <= 3:
		return fmt.Sprintf("%c%02d%02d", sign, h, m)
	case count == 1:
		if m == 0 && s == 0 {
			return fmt.Sprintf("%c%02d", sign, h)
		}
		return fmt.Sprintf("%c%02d%02d", sign, h, m)
	case count == 2:
		return fmt.Sprintf("%c%02d%02d", sign, h, m)
	case count == 3:
		return fmt.Sprintf("%c%02d:%02d", sign, h, m)
	case count == 4:
		if s != 0 {
			return fmt.Sprintf("%c%02d%02d%02d", sign, h, m, s)
		}
		return fmt.Sprintf("%c%02d%02d", sign, h, m)
	default:
		if s != 0 {
			return fmt.Sprintf("%c%02d:%02d:%02d", sign, h, m, s)
		}
		return fmt.Sprintf("%c%02d:%02d", sign, h, m)
	}
}

func localizedOffset(offset, count int, loc jvm.Locale) string {
	prefix := "GMT"
	if localeLanguage(loc) == "fr" {
		prefix = "UTC"
	}
	if offset == 0 {
		return prefix
	}
	sign := '+'
	value := offset
	if value < 0 {
		sign, value = '-', -value
	}
	h, m, s := value/3600, value/60%60, value%60
	if count == 1 {
		result := fmt.Sprintf("%s%c%d", prefix, sign, h)
		if m != 0 || s != 0 {
			result += fmt.Sprintf(":%02d", m)
		}
		if s != 0 {
			result += fmt.Sprintf(":%02d", s)
		}
		return result
	}
	result := fmt.Sprintf("%s%c%02d:%02d", prefix, sign, h, m)
	if s != 0 {
		result += fmt.Sprintf(":%02d", s)
	}
	return result
}

func zoneName(zone ZoneID, instant *Instant, kind byte, count int, loc jvm.Locale) string {
	if zone.isFixed {
		if zone.fixedOffset == 0 && zone.id == "UTC" {
			if count == 4 {
				switch localeLanguage(loc) {
				case "de":
					return "Koordinierte Weltzeit"
				case "fr":
					return "temps universel coordonné"
				case "es":
					if strings.EqualFold(loc.Country, "AR") {
						return "hora universal coordinada"
					}
					return "tiempo universal coordinado"
				case "tr":
					return "Eş Güdümlü Evrensel Zaman"
				default:
					return "Coordinated Universal Time"
				}
			}
			return zone.id
		}
		if zone.id == "GMT" {
			if count < 4 {
				return "GMT"
			}
			switch localeLanguage(loc) {
			case "de":
				return "Mittlere Greenwich-Zeit"
			case "fr":
				return "heure moyenne de Greenwich"
			case "es":
				if strings.EqualFold(loc.Country, "AR") {
					return "hora del meridiano de Greenwich"
				}
				return "hora del meridiano de Greenwich"
			case "tr":
				return "Greenwich Ortalama Saati"
			default:
				return "Greenwich Mean Time"
			}
		}
		return zone.id
	}
	dst := false
	if instant != nil {
		abbr := zone.abbreviationAt(*instant)
		dst = abbr == "CEST" || abbr == "EDT" || abbr == "PDT"
	}
	lang := localeLanguage(loc)
	if kind == 'v' {
		dst = false
	}
	if count < 4 {
		if zone.id == "America/New_York" && (kind == 'v' || instant == nil) {
			return "ET"
		}
		if zone.id == "Asia/Kolkata" && (kind == 'v' || instant == nil) {
			return "IT"
		}
		if zone.id == "America/Los_Angeles" && (kind == 'v' || instant == nil) {
			return "PT"
		}
		if zone.id == "Europe/Berlin" && (kind == 'v' || instant == nil) {
			if lang == "de" {
				return "MEZ"
			}
			if lang == "es" && strings.EqualFold(loc.Country, "AR") {
				return "GMT+01:00"
			}
			return "CET"
		}
		if instant != nil {
			abbr := zone.abbreviationAt(*instant)
			if lang == "de" && zone.id == "Europe/Berlin" {
				if dst {
					return "MESZ"
				}
				return "MEZ"
			}
			if lang == "es" && strings.EqualFold(loc.Country, "AR") && zone.id == "Europe/Berlin" {
				return localizedOffset(zone.offsetAt(*instant), 4, loc)
			}
			return abbr
		}
		return zone.id
	}
	return longZoneName(zone.id, dst, kind == 'v' || instant == nil, loc)
}

func longZoneName(id string, dst, generic bool, loc jvm.Locale) string {
	lang := localeLanguage(loc)
	if id == "Europe/Berlin" {
		switch lang {
		case "de":
			if generic {
				return "Mitteleuropäische Zeit"
			}
			if dst {
				return "Mitteleuropäische Sommerzeit"
			}
			return "Mitteleuropäische Normalzeit"
		case "fr":
			if generic {
				return "heure d’Europe centrale"
			}
			if dst {
				return "heure d’été d’Europe centrale"
			}
			return "heure normale d’Europe centrale"
		case "es":
			if generic {
				return "hora de Europa central"
			}
			if dst {
				return "hora de verano de Europa central"
			}
			return "hora estándar de Europa central"
		case "tr":
			if generic {
				return "Orta Avrupa Saati"
			}
			if dst {
				return "Orta Avrupa Yaz Saati"
			}
			return "Orta Avrupa Standart Saati"
		default:
			if generic {
				return "Central European Time"
			}
			if dst {
				return "Central European Summer Time"
			}
			return "Central European Standard Time"
		}
	}
	if id == "America/New_York" {
		switch lang {
		case "de":
			if generic {
				return "Nordamerikanische Ostküstenzeit"
			}
			if dst {
				return "Nordamerikanische Ostküsten-Sommerzeit"
			}
			return "Nordamerikanische Ostküsten-Normalzeit"
		case "fr":
			if generic {
				return "heure de l’Est nord-américain"
			}
			if dst {
				return "heure d’été de l’Est nord-américain"
			}
			return "heure normale de l’Est nord-américain"
		case "es":
			if generic {
				return "hora oriental"
			}
			if dst {
				return "hora de verano oriental"
			}
			return "hora estándar oriental"
		case "tr":
			if generic {
				return "Kuzey Amerika Doğu Saati"
			}
			if dst {
				return "Kuzey Amerika Doğu Yaz Saati"
			}
			return "Kuzey Amerika Doğu Standart Saati"
		default:
			if generic {
				return "Eastern Time"
			}
			if dst {
				return "Eastern Daylight Time"
			}
			return "Eastern Standard Time"
		}
	}
	if id == "Asia/Kolkata" {
		switch lang {
		case "de":
			if generic {
				return "Zeitzone für Indien"
			}
			return "Indische Normalzeit"
		case "fr":
			if generic {
				return "Inde"
			}
			return "heure de l’Inde"
		case "es":
			if generic {
				return "Hora de India"
			}
			if strings.EqualFold(loc.Country, "AR") {
				return "hora de India"
			}
			return "hora estándar de la India"
		case "tr":
			if generic {
				return "India Time"
			}
			return "Hindistan Standart Saati"
		default:
			if generic {
				return "India Time"
			}
			return "India Standard Time"
		}
	}
	if id == "America/Los_Angeles" {
		switch lang {
		case "de":
			if generic {
				return "Nordamerikanische Westküstenzeit"
			}
			if dst {
				return "Nordamerikanische Westküsten-Sommerzeit"
			}
			return "Nordamerikanische Westküsten-Normalzeit"
		case "fr":
			if generic {
				return "heure du Pacifique nord-américain"
			}
			if dst {
				return "heure d’été du Pacifique nord-américain"
			}
			return "heure normale du Pacifique nord-américain"
		case "es":
			if generic {
				return "hora del Pacífico"
			}
			if dst {
				return "hora de verano del Pacífico"
			}
			return "hora estándar del Pacífico"
		case "tr":
			if generic {
				return "Kuzey Amerika Pasifik Saati"
			}
			if dst {
				return "Kuzey Amerika Pasifik Yaz Saati"
			}
			return "Kuzey Amerika Pasifik Standart Saati"
		default:
			if generic {
				return "Pacific Time"
			}
			if dst {
				return "Pacific Daylight Time"
			}
			return "Pacific Standard Time"
		}
	}
	return id
}

func unsupported(field string) error {
	return jvm.Throw("java.time.temporal.UnsupportedTemporalTypeException", "Unsupported field: "+field)
}

func cannotExtractZone(display string) error {
	return jvm.Throw("java.time.DateTimeException", "Unable to extract ZoneId from temporal "+display)
}

func isMissingField(err error) bool {
	e, ok := err.(*jvm.Exception)
	return ok && (e.Class == "java.time.temporal.UnsupportedTemporalTypeException" || strings.HasPrefix(e.Message, "Unable to extract ZoneId"))
}

func illegalArgument(message string) error {
	return jvm.Throw("java.lang.IllegalArgumentException", message)
}

func (f *Formatter) formatLocalized(x temporalFields) (string, error) {
	datePattern := localizedDatePattern(f.locale, f.dateStyle)
	df, err := OfPattern(datePattern)
	if err != nil {
		return "", err
	}
	df.locale = f.locale
	date, err := formatTokens(df.tokens, x, f.locale)
	if err != nil {
		return "", err
	}
	if f.timeStyle == "" {
		return date, nil
	}
	timePattern := localizedTimePattern(f.locale, f.timeStyle)
	tf, err := OfPattern(timePattern)
	if err != nil {
		return "", err
	}
	tf.locale = f.locale
	tm, err := formatTokens(tf.tokens, x, f.locale)
	if err != nil {
		return "", err
	}
	return localizedJoin(f.locale, f.dateStyle, f.timeStyle, date, tm), nil
}

func localizedDatePattern(loc jvm.Locale, style FormatStyle) string {
	switch localeLanguage(loc) {
	case "de":
		switch style {
		case FullStyle:
			return "EEEE, d. MMMM y"
		case LongStyle:
			return "d. MMMM y"
		case MediumStyle:
			return "dd.MM.y"
		default:
			return "dd.MM.yy"
		}
	case "fr":
		switch style {
		case FullStyle:
			return "EEEE d MMMM y"
		case LongStyle:
			return "d MMMM y"
		case MediumStyle:
			return "d MMM y"
		default:
			return "dd/MM/y"
		}
	case "es":
		switch style {
		case FullStyle:
			return "EEEE, d 'de' MMMM 'de' y"
		case LongStyle:
			return "d 'de' MMMM 'de' y"
		case MediumStyle:
			return "d MMM y"
		default:
			return "d/M/yy"
		}
	case "tr":
		switch style {
		case FullStyle:
			return "d MMMM y EEEE"
		case LongStyle:
			return "d MMMM y"
		case MediumStyle:
			return "d MMM y"
		default:
			return "d.MM.y"
		}
	default:
		switch style {
		case FullStyle:
			return "EEEE, MMMM d, y"
		case LongStyle:
			return "MMMM d, y"
		case MediumStyle:
			return "MMM d, y"
		default:
			return "M/d/yy"
		}
	}
}

func localizedTimePattern(loc jvm.Locale, style FormatStyle) string {
	if localeLanguage(loc) == "en" {
		switch style {
		case FullStyle:
			return "h:mm:ss\u202fa zzzz"
		case LongStyle:
			return "h:mm:ss\u202fa z"
		case MediumStyle:
			return "h:mm:ss\u202fa"
		default:
			return "h:mm\u202fa"
		}
	}
	if localeLanguage(loc) == "es" && loc.Country == "" && style == FullStyle {
		return "H:mm:ss '('zzzz')'"
	}
	if localeLanguage(loc) == "es" && loc.Country == "" {
		switch style {
		case LongStyle:
			return "H:mm:ss z"
		case MediumStyle:
			return "H:mm:ss"
		default:
			return "H:mm"
		}
	}
	switch style {
	case FullStyle:
		return "HH:mm:ss zzzz"
	case LongStyle:
		return "HH:mm:ss z"
	case MediumStyle:
		return "HH:mm:ss"
	default:
		return "HH:mm"
	}
}

func localizedJoin(loc jvm.Locale, ds, ts FormatStyle, date, tm string) string {
	switch localeLanguage(loc) {
	case "tr":
		return date + " " + tm
	case "fr":
		if ts == ShortStyle || ds == ShortStyle {
			return date + " " + tm
		}
		return date + ", " + tm
	case "es":
		if strings.EqualFold(loc.Country, "AR") {
			if (ds == FullStyle || ds == LongStyle) && ts == MediumStyle || ds == MediumStyle && ts != ShortStyle {
				return date + " " + tm
			}
			return date + ", " + tm
		}
		return date + ", " + tm
	default:
		return date + ", " + tm
	}
}
