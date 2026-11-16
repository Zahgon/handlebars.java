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

// Package jtime reproduces the subset of JDK 21 java.time used by handlebars.java.
package jtime

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jknack/handlebars.java/internal/jvm"
)

const (
	secondsPerDay = int64(86400)
	nanosPerDay   = int64(86400000000000)
)

// LocalDate is an ISO-8601 date without a time or zone.
type LocalDate struct {
	Year int // Year is the proleptic ISO year.
	// Month is the month-of-year from 1 through 12.
	Month int
	// Day is the day-of-month.
	Day int
}

// NewLocalDate constructs a validated LocalDate.
func NewLocalDate(year, month, day int) (LocalDate, error) {
	d := LocalDate{Year: year, Month: month, Day: day}
	if err := d.validate(); err != nil {
		return LocalDate{}, err
	}
	return d, nil
}

func (d LocalDate) validate() error {
	if d.Year < -999999999 || d.Year > 999999999 {
		return dateTimeError("Invalid value for Year (valid values -999999999 - 999999999): %d", d.Year)
	}
	if d.Month < 1 || d.Month > 12 {
		return dateTimeError("Invalid value for MonthOfYear (valid values 1 - 12): %d", d.Month)
	}
	max := lengthOfMonth(d.Year, d.Month)
	if d.Day < 1 || d.Day > max {
		if d.Day >= 1 && d.Day <= 31 {
			return dateTimeError("Invalid date '%s %d'", monthEnglish[d.Month], d.Day)
		}
		return dateTimeError("Invalid value for DayOfMonth (valid values 1 - 28/31): %d", d.Day)
	}
	return nil
}

// JavaClass returns the corresponding Java class name.
func (LocalDate) JavaClass() string { return "java.time.LocalDate" }

// JavaString returns LocalDate.toString().
func (d LocalDate) JavaString() string {
	y := formatISOYear(d.Year)
	return fmt.Sprintf("%s-%02d-%02d", y, d.Month, d.Day)
}

// String returns LocalDate.toString().
func (d LocalDate) String() string { return d.JavaString() }

// JavaHashCode returns LocalDate.hashCode().
func (d LocalDate) JavaHashCode() int32 {
	y := int32(d.Year)
	return (y & -2048) ^ ((y << 11) + int32(d.Month<<6) + int32(d.Day))
}

// JavaEquals implements LocalDate.equals().
func (d LocalDate) JavaEquals(other any) bool {
	o, ok := asLocalDate(other)
	return ok && d == o
}

// JavaCompareTo implements LocalDate.compareTo().
func (d LocalDate) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot invoke "java.time.chrono.ChronoLocalDate.toEpochDay()" because "other" is null`)
	}
	o, ok := asLocalDate(other)
	if !ok {
		return 0, classCast(other, "java.time.chrono.ChronoLocalDate")
	}
	if d.Year != o.Year {
		return d.Year - o.Year, nil
	}
	if d.Month != o.Month {
		return d.Month - o.Month, nil
	}
	return d.Day - o.Day, nil
}

func asLocalDate(v any) (LocalDate, bool) {
	switch x := v.(type) {
	case LocalDate:
		return x, true
	case *LocalDate:
		if x != nil {
			return *x, true
		}
		return LocalDate{}, false
	default:
		return LocalDate{}, false
	}
}

// LocalTime is a time of day with nanosecond precision.
type LocalTime struct {
	// Hour is the hour-of-day from 0 through 23.
	Hour int
	// Minute is the minute-of-hour.
	Minute int
	// Second is the second-of-minute.
	Second int
	// Nano is the nanosecond-of-second.
	Nano int
}

// NewLocalTime constructs a validated LocalTime.
func NewLocalTime(hour, minute, second, nano int) (LocalTime, error) {
	t := LocalTime{Hour: hour, Minute: minute, Second: second, Nano: nano}
	if err := t.validate(); err != nil {
		return LocalTime{}, err
	}
	return t, nil
}

func (t LocalTime) validate() error {
	checks := []struct {
		name       string
		value, max int
	}{{"HourOfDay", t.Hour, 23}, {"MinuteOfHour", t.Minute, 59}, {"SecondOfMinute", t.Second, 59}, {"NanoOfSecond", t.Nano, 999999999}}
	for _, c := range checks {
		if c.value < 0 || c.value > c.max {
			return dateTimeError("Invalid value for %s (valid values 0 - %d): %d", c.name, c.max, c.value)
		}
	}
	return nil
}

// JavaClass returns the corresponding Java class name.
func (LocalTime) JavaClass() string { return "java.time.LocalTime" }

// JavaString returns LocalTime.toString().
func (t LocalTime) JavaString() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "%02d:%02d", t.Hour, t.Minute)
	if t.Second != 0 || t.Nano != 0 {
		fmt.Fprintf(b, ":%02d", t.Second)
		if t.Nano != 0 {
			frac := fmt.Sprintf("%09d", t.Nano)
			if t.Nano%1000000 == 0 {
				frac = frac[:3]
			} else if t.Nano%1000 == 0 {
				frac = frac[:6]
			}
			b.WriteByte('.')
			b.WriteString(frac)
		}
	}
	return b.String()
}

// String returns LocalTime.toString().
func (t LocalTime) String() string { return t.JavaString() }

func (t LocalTime) nanoOfDay() int64 {
	return int64(t.Hour)*3600000000000 + int64(t.Minute)*60000000000 + int64(t.Second)*1000000000 + int64(t.Nano)
}

// JavaHashCode returns LocalTime.hashCode().
func (t LocalTime) JavaHashCode() int32 {
	n := t.nanoOfDay()
	return int32(n ^ int64(uint64(n)>>32))
}

// JavaEquals implements LocalTime.equals().
func (t LocalTime) JavaEquals(other any) bool {
	o, ok := asLocalTime(other)
	return ok && t == o
}

// JavaCompareTo implements LocalTime.compareTo().
func (t LocalTime) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot read field "hour" because "other" is null`)
	}
	o, ok := asLocalTime(other)
	if !ok {
		return 0, classCast(other, "java.time.LocalTime")
	}
	if t.Hour != o.Hour {
		return t.Hour - o.Hour, nil
	}
	if t.Minute != o.Minute {
		return t.Minute - o.Minute, nil
	}
	if t.Second != o.Second {
		return t.Second - o.Second, nil
	}
	return t.Nano - o.Nano, nil
}

func asLocalTime(v any) (LocalTime, bool) {
	switch x := v.(type) {
	case LocalTime:
		return x, true
	case *LocalTime:
		if x != nil {
			return *x, true
		}
		return LocalTime{}, false
	default:
		return LocalTime{}, false
	}
}

// LocalDateTime is an ISO-8601 date and time without an offset or zone.
type LocalDateTime struct {
	// Date is the local date component.
	Date LocalDate
	// Time is the local time component.
	Time LocalTime
}

// NewLocalDateTime constructs a validated LocalDateTime.
func NewLocalDateTime(year, month, day, hour, minute, second, nano int) (LocalDateTime, error) {
	d, err := NewLocalDate(year, month, day)
	if err != nil {
		return LocalDateTime{}, err
	}
	t, err := NewLocalTime(hour, minute, second, nano)
	if err != nil {
		return LocalDateTime{}, err
	}
	return LocalDateTime{Date: d, Time: t}, nil
}

// JavaClass returns the corresponding Java class name.
func (LocalDateTime) JavaClass() string { return "java.time.LocalDateTime" }

// JavaString returns LocalDateTime.toString().
func (d LocalDateTime) JavaString() string { return d.Date.JavaString() + "T" + d.Time.JavaString() }

// String returns LocalDateTime.toString().
func (d LocalDateTime) String() string { return d.JavaString() }

// JavaHashCode returns LocalDateTime.hashCode().
func (d LocalDateTime) JavaHashCode() int32 { return d.Date.JavaHashCode() ^ d.Time.JavaHashCode() }

// JavaEquals implements LocalDateTime.equals().
func (d LocalDateTime) JavaEquals(other any) bool {
	o, ok := asLocalDateTime(other)
	return ok && d == o
}

// JavaCompareTo implements LocalDateTime.compareTo().
func (d LocalDateTime) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot invoke "java.time.chrono.ChronoLocalDateTime.toLocalDate()" because "other" is null`)
	}
	o, ok := asLocalDateTime(other)
	if !ok {
		return 0, classCast(other, "java.time.chrono.ChronoLocalDateTime")
	}
	if n, _ := d.Date.JavaCompareTo(o.Date); n != 0 {
		return n, nil
	}
	return d.Time.JavaCompareTo(o.Time)
}

func asLocalDateTime(v any) (LocalDateTime, bool) {
	switch x := v.(type) {
	case LocalDateTime:
		return x, true
	case *LocalDateTime:
		if x != nil {
			return *x, true
		}
		return LocalDateTime{}, false
	default:
		return LocalDateTime{}, false
	}
}

// Instant is a point on the time-line, stored as epoch seconds and nanoseconds.
type Instant struct {
	// EpochSecond is the number of seconds since 1970-01-01T00:00:00Z.
	EpochSecond int64
	// Nano is the nanosecond adjustment within the second.
	Nano int
}

// NewInstant constructs a normalized Instant.
func NewInstant(epochSecond int64, nano int) (Instant, error) {
	epochSecond += int64(floorDiv(nano, 1000000000))
	nano = floorMod(nano, 1000000000)
	if epochSecond < -31557014167219200 || epochSecond > 31556889864403199 {
		return Instant{}, dateTimeError("Instant exceeds minimum or maximum instant")
	}
	return Instant{EpochSecond: epochSecond, Nano: nano}, nil
}

// InstantFromTime converts a Go time.Time to an Instant.
func InstantFromTime(t time.Time) Instant {
	return Instant{EpochSecond: t.Unix(), Nano: t.Nanosecond()}
}

// JavaClass returns the corresponding Java class name.
func (Instant) JavaClass() string { return "java.time.Instant" }

// JavaString returns Instant.toString().
func (i Instant) JavaString() string {
	dt := localDateTimeFromEpoch(i.EpochSecond, i.Nano, 0)
	y := formatISOYear(dt.Date.Year)
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s-%02d-%02dT%02d:%02d:%02d", y, dt.Date.Month, dt.Date.Day, dt.Time.Hour, dt.Time.Minute, dt.Time.Second)
	if i.Nano != 0 {
		frac := fmt.Sprintf("%09d", i.Nano)
		if i.Nano%1000000 == 0 {
			frac = frac[:3]
		} else if i.Nano%1000 == 0 {
			frac = frac[:6]
		}
		b.WriteByte('.')
		b.WriteString(frac)
	}
	b.WriteByte('Z')
	return b.String()
}

// String returns Instant.toString().
func (i Instant) String() string { return i.JavaString() }

// JavaHashCode returns Instant.hashCode().
func (i Instant) JavaHashCode() int32 {
	return int32(i.EpochSecond^int64(uint64(i.EpochSecond)>>32)) + 51*int32(i.Nano)
}

// JavaEquals implements Instant.equals().
func (i Instant) JavaEquals(other any) bool {
	o, ok := asInstant(other)
	return ok && i == o
}

// JavaCompareTo implements Instant.compareTo().
func (i Instant) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot read field "seconds" because "otherInstant" is null`)
	}
	o, ok := asInstant(other)
	if !ok {
		return 0, classCast(other, "java.time.Instant")
	}
	if i.EpochSecond < o.EpochSecond {
		return -1, nil
	}
	if i.EpochSecond > o.EpochSecond {
		return 1, nil
	}
	return i.Nano - o.Nano, nil
}

func asInstant(v any) (Instant, bool) {
	switch x := v.(type) {
	case Instant:
		return x, true
	case *Instant:
		if x != nil {
			return *x, true
		}
		return Instant{}, false
	default:
		return Instant{}, false
	}
}

func formatISOYear(year int) string {
	if year >= 0 && year <= 9999 {
		return fmt.Sprintf("%04d", year)
	}
	if year > 9999 {
		return "+" + strconv.Itoa(year)
	}
	if year >= -9999 {
		return fmt.Sprintf("-%04d", -year)
	}
	return strconv.Itoa(year)
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

func lengthOfMonth(year, month int) int {
	switch month {
	case 2:
		if isLeap(year) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

var monthEnglish = [...]string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

// toEpochDay follows LocalDate.toEpochDay and works over Java's full year range.
func (d LocalDate) toEpochDay() int64 {
	y, m := int64(d.Year), int64(d.Month)
	var total int64
	if y >= 0 {
		total = 365*y + (y+3)/4 - (y+99)/100 + (y+399)/400
	} else {
		total = 365*y - y/-4 + y/-100 - y/-400
	}
	total += (367*m - 362) / 12
	total += int64(d.Day - 1)
	if m > 2 {
		total--
		if !isLeap(d.Year) {
			total--
		}
	}
	return total - 719528
}

func dateFromEpochDay(epochDay int64) LocalDate {
	zeroDay := epochDay + 719528 - 60
	adjust := int64(0)
	if zeroDay < 0 {
		adjustCycles := (zeroDay+1)/146097 - 1
		adjust = adjustCycles * 400
		zeroDay += -adjustCycles * 146097
	}
	yearEst := (400*zeroDay + 591) / 146097
	doyEst := zeroDay - (365*yearEst + yearEst/4 - yearEst/100 + yearEst/400)
	if doyEst < 0 {
		yearEst--
		doyEst = zeroDay - (365*yearEst + yearEst/4 - yearEst/100 + yearEst/400)
	}
	yearEst += adjust
	marchDoy0 := int(doyEst)
	marchMonth0 := (marchDoy0*5 + 2) / 153
	month := (marchMonth0+2)%12 + 1
	day := marchDoy0 - (marchMonth0*306+5)/10 + 1
	yearEst += int64(marchMonth0 / 10)
	return LocalDate{Year: int(yearEst), Month: month, Day: day}
}

func localDateTimeFromEpoch(epochSecond int64, nano, offsetSeconds int) LocalDateTime {
	localSecond := epochSecond + int64(offsetSeconds)
	epochDay := floorDiv64(localSecond, secondsPerDay)
	secs := floorMod64(localSecond, secondsPerDay)
	return LocalDateTime{
		Date: dateFromEpochDay(epochDay),
		Time: LocalTime{Hour: int(secs / 3600), Minute: int(secs/60) % 60, Second: int(secs % 60), Nano: nano},
	}
}

func (d LocalDateTime) epochSecond(offsetSeconds int) int64 {
	return d.Date.toEpochDay()*secondsPerDay + int64(d.Time.Hour*3600+d.Time.Minute*60+d.Time.Second-offsetSeconds)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func floorMod(a, b int) int { return a - floorDiv(a, b)*b }

func floorDiv64(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func floorMod64(a, b int64) int64 { return a - floorDiv64(a, b)*b }

func dateTimeError(format string, args ...any) error {
	return jvm.Throw("java.time.DateTimeException", fmt.Sprintf(format, args...))
}

func classCast(v any, target string) error {
	name := "null"
	if v != nil {
		name = jvm.ClassName(v)
	}
	return jvm.Throw("java.lang.ClassCastException", fmt.Sprintf("class %s cannot be cast to class %s (%s and %s are in module java.base of loader 'bootstrap')", name, target, name, target))
}
