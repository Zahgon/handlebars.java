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
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// ZoneOffset is a fixed displacement from UTC in seconds.
type ZoneOffset struct {
	// TotalSeconds is the signed displacement from UTC.
	TotalSeconds int
}

// NewZoneOffset constructs a validated fixed offset.
func NewZoneOffset(totalSeconds int) (ZoneOffset, error) {
	if totalSeconds < -64800 || totalSeconds > 64800 {
		return ZoneOffset{}, dateTimeError("Zone offset not in valid range: -18:00 to +18:00")
	}
	return ZoneOffset{TotalSeconds: totalSeconds}, nil
}

// ParseZoneOffset parses Z or a signed ISO-8601 offset.
func ParseZoneOffset(text string) (ZoneOffset, error) {
	if text == "Z" || text == "+00" || text == "+0000" || text == "+00:00" || text == "-00" || text == "-0000" || text == "-00:00" {
		return ZoneOffset{}, nil
	}
	sec, ok := parseOffsetSeconds(text)
	if !ok {
		return ZoneOffset{}, dateTimeError("Invalid ID for ZoneOffset, invalid format: %s", text)
	}
	return NewZoneOffset(sec)
}

// JavaClass returns the corresponding Java class name.
func (ZoneOffset) JavaClass() string { return "java.time.ZoneOffset" }

// JavaString returns ZoneOffset.toString().
func (z ZoneOffset) JavaString() string { return offsetID(z.TotalSeconds, true, true, true) }

// String returns ZoneOffset.toString().
func (z ZoneOffset) String() string { return z.JavaString() }

// JavaHashCode returns ZoneOffset.hashCode().
func (z ZoneOffset) JavaHashCode() int32 { return int32(z.TotalSeconds) }

// JavaEquals implements ZoneOffset.equals().
func (z ZoneOffset) JavaEquals(other any) bool {
	o, ok := asZoneOffset(other)
	return ok && z == o
}

// JavaCompareTo implements ZoneOffset.compareTo(), whose ordering is descending by total seconds.
func (z ZoneOffset) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot read field "totalSeconds" because "other" is null`)
	}
	o, ok := asZoneOffset(other)
	if !ok {
		return 0, classCast(other, "java.time.ZoneOffset")
	}
	return o.TotalSeconds - z.TotalSeconds, nil
}

func asZoneOffset(v any) (ZoneOffset, bool) {
	switch x := v.(type) {
	case ZoneOffset:
		return x, true
	case *ZoneOffset:
		if x != nil {
			return *x, true
		}
		return ZoneOffset{}, false
	case ZoneID:
		if x.isOffsetID() {
			return ZoneOffset{TotalSeconds: x.fixedOffset}, true
		}
		return ZoneOffset{}, false
	case *ZoneID:
		if x != nil && x.isOffsetID() {
			return ZoneOffset{TotalSeconds: x.fixedOffset}, true
		}
		return ZoneOffset{}, false
	default:
		return ZoneOffset{}, false
	}
}

// ZoneID identifies either a region time-zone or a fixed offset zone.
type ZoneID struct {
	id          string
	location    *time.Location
	fixedOffset int
	isFixed     bool
}

// OfZoneID implements ZoneId.of for region IDs, UTC/GMT IDs and ISO offsets.
func OfZoneID(id string) (ZoneID, error) {
	if id == "Z" || strings.HasPrefix(id, "+") || strings.HasPrefix(id, "-") {
		o, err := ParseZoneOffset(id)
		if err != nil {
			return ZoneID{}, err
		}
		return fixedZone(o.JavaString(), o.TotalSeconds), nil
	}
	if id == "UTC" || id == "GMT" || id == "UT" {
		return fixedZone(id, 0), nil
	}
	for _, prefix := range []string{"UTC", "GMT", "UT"} {
		if strings.HasPrefix(id, prefix+"+") || strings.HasPrefix(id, prefix+"-") {
			o, err := ParseZoneOffset(strings.TrimPrefix(id, prefix))
			if err != nil {
				return ZoneID{}, err
			}
			if o.TotalSeconds == 0 {
				return fixedZone(prefix, 0), nil
			}
			return fixedZone(prefix+o.JavaString(), o.TotalSeconds), nil
		}
	}
	loc, err := time.LoadLocation(id)
	if err != nil {
		return ZoneID{}, jvm.Throw("java.time.zone.ZoneRulesException", "Unknown time-zone ID: "+id)
	}
	return ZoneID{id: id, location: loc}, nil
}

func fixedZone(id string, seconds int) ZoneID {
	return ZoneID{id: id, location: time.FixedZone(id, seconds), fixedOffset: seconds, isFixed: true}
}

// ID returns ZoneId.getId().
func (z ZoneID) ID() string { return z.id }

// JavaClass returns the corresponding Java class name.
func (z ZoneID) JavaClass() string {
	if z.isOffsetID() {
		return "java.time.ZoneOffset"
	}
	return "java.time.ZoneRegion"
}

// JavaString returns ZoneId.toString().
func (z ZoneID) JavaString() string { return z.id }

// String returns ZoneId.toString().
func (z ZoneID) String() string { return z.id }

// JavaHashCode returns ZoneId.hashCode().
func (z ZoneID) JavaHashCode() int32 {
	if z.isOffsetID() {
		return int32(z.fixedOffset)
	}
	return jvm.StringHashCode(z.id)
}

// JavaEquals implements ZoneId.equals().
func (z ZoneID) JavaEquals(other any) bool {
	switch o := other.(type) {
	case ZoneID:
		return z.id == o.id
	case *ZoneID:
		return o != nil && z.id == o.id
	case ZoneOffset:
		return z.isOffsetID() && z.fixedOffset == o.TotalSeconds
	case *ZoneOffset:
		return o != nil && z.isOffsetID() && z.fixedOffset == o.TotalSeconds
	default:
		return false
	}
}

func (z ZoneID) isOffsetID() bool {
	return z.id == "Z" || strings.HasPrefix(z.id, "+") || strings.HasPrefix(z.id, "-")
}

func (z ZoneID) offsetAt(i Instant) int {
	if z.isFixed {
		return z.fixedOffset
	}
	if z.location == nil {
		return 0
	}
	_, off := time.Unix(i.EpochSecond, int64(i.Nano)).In(z.location).Zone()
	return off
}

func (z ZoneID) abbreviationAt(i Instant) string {
	if z.isFixed {
		if z.fixedOffset == 0 {
			return z.id
		}
		return z.id
	}
	name, _ := time.Unix(i.EpochSecond, int64(i.Nano)).In(z.location).Zone()
	return name
}

func (z ZoneID) instantAtLocal(d LocalDate, t LocalTime) Instant {
	gt := time.Date(d.Year, time.Month(d.Month), d.Day, t.Hour, t.Minute, t.Second, t.Nano, z.location)
	return InstantFromTime(gt)
}

var (
	systemZoneMu sync.RWMutex
	systemZone   = fixedZone("UTC", 0)
)

// ZoneFromTimeZoneID implements TimeZone.getTimeZone(id).toZoneId().
func ZoneFromTimeZoneID(id string) ZoneID {
	if id == "UTC" || id == "GMT" {
		z, _ := OfZoneID(id)
		return z
	}
	if strings.HasPrefix(id, "GMT+") || strings.HasPrefix(id, "GMT-") {
		raw := strings.TrimPrefix(id, "GMT")
		if sec, ok := parseFlexibleGMTOffset(raw); ok && sec >= -64800 && sec <= 64800 {
			return fixedZone("GMT"+offsetID(sec, false, true, false), sec)
		}
		return fixedZone("GMT", 0)
	}
	if id == "Europe/Berlin" || id == "America/New_York" {
		z, err := OfZoneID(id)
		if err == nil {
			return z
		}
	}
	if id == "PST" {
		z, err := OfZoneID("America/Los_Angeles")
		if err == nil {
			return z
		}
	}
	// TimeZone accepts the IANA database; verify rather than maintaining aliases.
	if strings.Contains(id, "/") {
		if z, err := OfZoneID(id); err == nil {
			return z
		}
	}
	return fixedZone("GMT", 0)
}

// SystemDefaultZone returns the process-wide ZoneId used by formatter defaults.
func SystemDefaultZone() ZoneID {
	systemZoneMu.RLock()
	defer systemZoneMu.RUnlock()
	return systemZone
}

// SetSystemDefaultZone changes the process-wide ZoneId used by formatter defaults.
func SetSystemDefaultZone(zone ZoneID) {
	systemZoneMu.Lock()
	systemZone = zone
	systemZoneMu.Unlock()
}

// OffsetDateTime combines a LocalDateTime and ZoneOffset.
type OffsetDateTime struct {
	// DateTime is the local date-time component.
	DateTime LocalDateTime
	// Offset is the fixed offset from UTC.
	Offset ZoneOffset
}

// NewOffsetDateTime constructs an OffsetDateTime from local fields and an offset.
func NewOffsetDateTime(dateTime LocalDateTime, offset ZoneOffset) OffsetDateTime {
	return OffsetDateTime{DateTime: dateTime, Offset: offset}
}

// OffsetDateTimeOfInstant constructs an OffsetDateTime from an instant and offset.
func OffsetDateTimeOfInstant(instant Instant, offset ZoneOffset) OffsetDateTime {
	return OffsetDateTime{DateTime: localDateTimeFromEpoch(instant.EpochSecond, instant.Nano, offset.TotalSeconds), Offset: offset}
}

// ToInstant converts the value to its point on the time-line.
func (d OffsetDateTime) ToInstant() Instant {
	return Instant{EpochSecond: d.DateTime.epochSecond(d.Offset.TotalSeconds), Nano: d.DateTime.Time.Nano}
}

// JavaClass returns the corresponding Java class name.
func (OffsetDateTime) JavaClass() string { return "java.time.OffsetDateTime" }

// JavaString returns OffsetDateTime.toString().
func (d OffsetDateTime) JavaString() string { return d.DateTime.JavaString() + d.Offset.JavaString() }

// String returns OffsetDateTime.toString().
func (d OffsetDateTime) String() string { return d.JavaString() }

// JavaHashCode returns OffsetDateTime.hashCode().
func (d OffsetDateTime) JavaHashCode() int32 {
	return d.DateTime.JavaHashCode() ^ d.Offset.JavaHashCode()
}

// JavaEquals implements OffsetDateTime.equals().
func (d OffsetDateTime) JavaEquals(other any) bool {
	o, ok := asOffsetDateTime(other)
	return ok && d == o
}

// JavaCompareTo implements OffsetDateTime.compareTo().
func (d OffsetDateTime) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot invoke "java.time.OffsetDateTime.getOffset()" because "datetime2" is null`)
	}
	o, ok := asOffsetDateTime(other)
	if !ok {
		return 0, classCast(other, "java.time.OffsetDateTime")
	}
	if n, _ := d.ToInstant().JavaCompareTo(o.ToInstant()); n != 0 {
		return n, nil
	}
	return d.DateTime.JavaCompareTo(o.DateTime)
}

func asOffsetDateTime(v any) (OffsetDateTime, bool) {
	switch x := v.(type) {
	case OffsetDateTime:
		return x, true
	case *OffsetDateTime:
		if x != nil {
			return *x, true
		}
		return OffsetDateTime{}, false
	default:
		return OffsetDateTime{}, false
	}
}

// ZonedDateTime combines a LocalDateTime, resolved offset and ZoneId.
type ZonedDateTime struct {
	// DateTime is the resolved local date-time component.
	DateTime LocalDateTime
	// Offset is the offset selected from the zone rules.
	Offset ZoneOffset
	// Zone is the region or fixed-offset zone ID.
	Zone ZoneID
}

// ZonedDateTimeOfInstant constructs a ZonedDateTime using the zone rules at an instant.
func ZonedDateTimeOfInstant(instant Instant, zone ZoneID) ZonedDateTime {
	offset := ZoneOffset{TotalSeconds: zone.offsetAt(instant)}
	return ZonedDateTime{DateTime: localDateTimeFromEpoch(instant.EpochSecond, instant.Nano, offset.TotalSeconds), Offset: offset, Zone: zone}
}

// NewZonedDateTime constructs a ZonedDateTime from already-resolved components.
func NewZonedDateTime(dateTime LocalDateTime, offset ZoneOffset, zone ZoneID) ZonedDateTime {
	return ZonedDateTime{DateTime: dateTime, Offset: offset, Zone: zone}
}

// ToInstant converts the value to its point on the time-line.
func (d ZonedDateTime) ToInstant() Instant {
	return Instant{EpochSecond: d.DateTime.epochSecond(d.Offset.TotalSeconds), Nano: d.DateTime.Time.Nano}
}

// JavaClass returns the corresponding Java class name.
func (ZonedDateTime) JavaClass() string { return "java.time.ZonedDateTime" }

// JavaString returns ZonedDateTime.toString().
func (d ZonedDateTime) JavaString() string {
	s := d.DateTime.JavaString() + d.Offset.JavaString()
	if d.Zone.id != d.Offset.JavaString() {
		s += "[" + d.Zone.id + "]"
	}
	return s
}

// String returns ZonedDateTime.toString().
func (d ZonedDateTime) String() string { return d.JavaString() }

// JavaHashCode returns ZonedDateTime.hashCode().
func (d ZonedDateTime) JavaHashCode() int32 {
	zh := uint32(d.Zone.JavaHashCode())
	rot := int32((zh << 3) | (zh >> 29))
	return d.DateTime.JavaHashCode() ^ d.Offset.JavaHashCode() ^ rot
}

// JavaEquals implements ZonedDateTime.equals().
func (d ZonedDateTime) JavaEquals(other any) bool {
	o, ok := asZonedDateTime(other)
	return ok && d.DateTime == o.DateTime && d.Offset == o.Offset && d.Zone.id == o.Zone.id
}

// JavaCompareTo implements ZonedDateTime.compareTo().
func (d ZonedDateTime) JavaCompareTo(other any) (int, error) {
	if other == nil {
		return 0, jvm.Throw("java.lang.NullPointerException", `Cannot invoke "java.time.chrono.ChronoZonedDateTime.toEpochSecond()" because "other" is null`)
	}
	o, ok := asZonedDateTime(other)
	if !ok {
		return 0, classCast(other, "java.time.chrono.ChronoZonedDateTime")
	}
	if n, _ := d.ToInstant().JavaCompareTo(o.ToInstant()); n != 0 {
		return n, nil
	}
	if n, _ := d.DateTime.JavaCompareTo(o.DateTime); n != 0 {
		return n, nil
	}
	if d.Zone.id < o.Zone.id {
		return -1, nil
	}
	if d.Zone.id > o.Zone.id {
		return 1, nil
	}
	return 0, nil
}

func asZonedDateTime(v any) (ZonedDateTime, bool) {
	switch x := v.(type) {
	case ZonedDateTime:
		return x, true
	case *ZonedDateTime:
		if x != nil {
			return *x, true
		}
		return ZonedDateTime{}, false
	default:
		return ZonedDateTime{}, false
	}
}

func parseOffsetSeconds(s string) (int, bool) {
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return 0, false
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	body := s[1:]
	var hh, mm, ss int
	var err error
	switch {
	case len(body) == 2:
		hh, err = strconv.Atoi(body)
	case len(body) == 4:
		hh, err = strconv.Atoi(body[:2])
		if err == nil {
			mm, err = strconv.Atoi(body[2:])
		}
	case len(body) == 5 && body[2] == ':':
		hh, err = strconv.Atoi(body[:2])
		if err == nil {
			mm, err = strconv.Atoi(body[3:])
		}
	case len(body) == 6:
		hh, err = strconv.Atoi(body[:2])
		if err == nil {
			mm, err = strconv.Atoi(body[2:4])
		}
		if err == nil {
			ss, err = strconv.Atoi(body[4:])
		}
	case len(body) == 8 && body[2] == ':' && body[5] == ':':
		hh, err = strconv.Atoi(body[:2])
		if err == nil {
			mm, err = strconv.Atoi(body[3:5])
		}
		if err == nil {
			ss, err = strconv.Atoi(body[6:])
		}
	default:
		return 0, false
	}
	if err != nil || mm > 59 || ss > 59 {
		return 0, false
	}
	return sign * (hh*3600 + mm*60 + ss), true
}

func parseFlexibleGMTOffset(s string) (int, bool) {
	if len(s) < 2 || (s[0] != '+' && s[0] != '-') {
		return 0, false
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	parts := strings.Split(s[1:], ":")
	if len(parts) > 3 || len(parts[0]) < 1 || len(parts[0]) > 2 {
		return 0, false
	}
	vals := [3]int{}
	for i, p := range parts {
		if p == "" || (i > 0 && len(p) != 2) {
			return 0, false
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return 0, false
		}
		vals[i] = v
	}
	if vals[0] > 23 || vals[1] > 59 || vals[2] > 59 {
		return 0, false
	}
	return sign * (vals[0]*3600 + vals[1]*60 + vals[2]), true
}

func offsetID(seconds int, zeroZ, colon, optionalSeconds bool) string {
	if seconds == 0 && zeroZ {
		return "Z"
	}
	sign := '+'
	if seconds < 0 {
		sign = '-'
		seconds = -seconds
	}
	h, m, s := seconds/3600, seconds/60%60, seconds%60
	sep := ""
	if colon {
		sep = ":"
	}
	result := fmt.Sprintf("%c%02d%s%02d", sign, h, sep, m)
	if s != 0 || !optionalSeconds {
		if s != 0 {
			result += sep + fmt.Sprintf("%02d", s)
		}
	}
	return result
}
