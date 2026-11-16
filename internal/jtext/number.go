// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

// Package jtext implements the subset of java.text and java.util text
// facilities used by handlebars.java. Locale tables were generated from the
// CLDR provider in JDK 21 (CLDR 42).
package jtext

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jknack/handlebars.java/internal/jvm"
)

type symbols struct {
	decimal, grouping      string
	currency, currencyCode string
}

func localeSymbols(loc jvm.Locale) symbols {
	lang := strings.ToLower(loc.Language)
	s := symbols{decimal: ".", grouping: ",", currency: "¤", currencyCode: "XXX"}
	switch lang {
	case "de", "es", "tr":
		s.decimal, s.grouping = ",", "."
	case "fr":
		s.decimal, s.grouping = ",", "\u202f"
	}
	switch lang + "-" + strings.ToUpper(loc.Country) {
	case "en-US":
		s.currency, s.currencyCode = "$", "USD"
	case "de-DE", "fr-FR":
		s.currency, s.currencyCode = "€", "EUR"
	case "es-AR":
		s.currency, s.currencyCode = "$", "ARS"
	case "tr-TR":
		s.currency, s.currencyCode = "₺", "TRY"
	}
	return s
}

type decimalPattern struct {
	positivePrefix, positiveSuffix string
	negativePrefix, negativeSuffix string
	minInteger, maxInteger         int
	minFraction, maxFraction       int
	grouping                       bool
	groupSize                      int
	multiplier                     int64
	exponent                       bool
	minExponent                    int
}

// NumberFormat is a port of java.text.DecimalFormat/NumberFormat for the
// numeric types in the shared Java value model.
type NumberFormat struct {
	loc              jvm.Locale
	pattern          decimalPattern
	groupingUsed     bool
	minInteger       int
	maxInteger       int
	minFraction      int
	maxFraction      int
	parseIntegerOnly bool
	roundingMode     string
}

// NewNumberInstance returns NumberFormat.getInstance(loc).
func NewNumberInstance(loc jvm.Locale) *NumberFormat {
	// The built-in pattern is a compile-time-valid DecimalFormat pattern.
	f, _ := NewDecimalFormat("#,##0.###", loc)
	return f
}

// NewIntegerInstance returns NumberFormat.getIntegerInstance(loc).
func NewIntegerInstance(loc jvm.Locale) *NumberFormat {
	// The built-in pattern is a compile-time-valid DecimalFormat pattern.
	f, _ := NewDecimalFormat("#,##0", loc)
	return f
}

// NewCurrencyInstance returns NumberFormat.getCurrencyInstance(loc).
func NewCurrencyInstance(loc jvm.Locale) *NumberFormat {
	pattern := "¤#,##0.00"
	if loc.Language == "" {
		pattern = "¤\u00a0#,##0.00"
	}
	switch strings.ToLower(loc.Language) {
	case "de", "fr", "es":
		pattern = "#,##0.00\u00a0¤"
	}
	if strings.EqualFold(loc.Language, "es") && strings.EqualFold(loc.Country, "AR") {
		pattern = "¤\u00a0#,##0.00"
	}
	// Every branch above is a compile-time-valid DecimalFormat pattern.
	f, _ := NewDecimalFormat(pattern, loc)
	return f
}

// NewPercentInstance returns NumberFormat.getPercentInstance(loc).
func NewPercentInstance(loc jvm.Locale) *NumberFormat {
	pattern := "#,##0%"
	switch strings.ToLower(loc.Language) {
	case "de", "fr", "es":
		pattern = "#,##0\u00a0%"
	case "tr":
		pattern = "%#,##0"
	}
	// Every branch above is a compile-time-valid DecimalFormat pattern.
	f, _ := NewDecimalFormat(pattern, loc)
	return f
}

// NewDecimalFormat constructs a DecimalFormat using DecimalFormatSymbols for
// loc and applies the Java non-localized pattern language.
func NewDecimalFormat(pattern string, loc jvm.Locale) (*NumberFormat, error) {
	p, err := parseDecimalPattern(pattern, loc)
	if err != nil {
		return nil, err
	}
	return &NumberFormat{
		loc:          loc,
		pattern:      p,
		groupingUsed: p.grouping,
		minInteger:   p.minInteger,
		maxInteger:   p.maxInteger,
		minFraction:  p.minFraction,
		maxFraction:  p.maxFraction,
		roundingMode: "HALF_EVEN",
	}, nil
}

// SetGroupingUsed sets NumberFormat's grouping flag.
func (f *NumberFormat) SetGroupingUsed(v bool) { f.groupingUsed = v }

// SetMaximumFractionDigits sets the maximum fraction digits, applying the JDK
// invariant that the minimum cannot exceed the maximum.
func (f *NumberFormat) SetMaximumFractionDigits(v int) {
	if v < 0 {
		v = 0
	}
	f.maxFraction = v
	if f.minFraction > v {
		f.minFraction = v
	}
}

// SetMaximumIntegerDigits sets the maximum integer digits.
func (f *NumberFormat) SetMaximumIntegerDigits(v int) {
	if v < 0 {
		v = 0
	}
	f.maxInteger = v
	if f.minInteger > v {
		f.minInteger = v
	}
}

// SetMinimumFractionDigits sets the minimum fraction digits.
func (f *NumberFormat) SetMinimumFractionDigits(v int) {
	if v < 0 {
		v = 0
	}
	f.minFraction = v
	if f.maxFraction < v {
		f.maxFraction = v
	}
}

// SetMinimumIntegerDigits sets the minimum integer digits.
func (f *NumberFormat) SetMinimumIntegerDigits(v int) {
	if v < 0 {
		v = 0
	}
	f.minInteger = v
	if f.maxInteger < v {
		f.maxInteger = v
	}
}

// SetParseIntegerOnly sets the parsing-only flag retained by NumberFormat.
func (f *NumberFormat) SetParseIntegerOnly(v bool) { f.parseIntegerOnly = v }

// SetRoundingMode applies java.math.RoundingMode.valueOf(name).
func (f *NumberFormat) SetRoundingMode(name string) error {
	name = strings.TrimSpace(name)
	switch name {
	case "UP", "DOWN", "CEILING", "FLOOR", "HALF_UP", "HALF_DOWN", "HALF_EVEN", "UNNECESSARY":
		f.roundingMode = name
		return nil
	default:
		return jvm.Throw("java.lang.IllegalArgumentException", "No enum constant java.math.RoundingMode."+name)
	}
}

// Format formats a Java numeric value.
func (f *NumberFormat) Format(number any) (string, error) {
	negativeZero := false
	switch n := number.(type) {
	case float64:
		if math.IsNaN(n) {
			return "NaN", nil
		}
		negativeZero = math.Signbit(n)
		if math.IsInf(n, 0) {
			return f.special("∞", math.Signbit(n)), nil
		}
	case float32:
		if math.IsNaN(float64(n)) {
			return "NaN", nil
		}
		negativeZero = math.Signbit(float64(n))
		if math.IsInf(float64(n), 0) {
			return f.special("∞", math.Signbit(float64(n))), nil
		}
	}
	r, ok := numberRat(number)
	if !ok {
		return "", jvm.Throw("java.lang.IllegalArgumentException", "Cannot format given Object as a Number")
	}
	if f.pattern.multiplier != 1 {
		r.Mul(r, new(big.Rat).SetInt64(f.pattern.multiplier))
	}
	// DecimalFormat's double path uses the exact binary value when discarded
	// digits decide rounding, but does not manufacture digits beyond the
	// shortest round-trippable decimal. This is the behavior of DigitList's
	// FloatingDecimal converter in JDK 21.
	switch n := number.(type) {
	case float64:
		scaled := n * float64(f.pattern.multiplier)
		if math.IsInf(scaled, 0) {
			return f.special("∞", math.Signbit(scaled)), nil
		}
		r = decimalFormatFloatRat(scaled, f.maxFraction, 64)
	case float32:
		scaled := float64(n) * float64(f.pattern.multiplier)
		if math.IsInf(scaled, 0) {
			return f.special("∞", math.Signbit(scaled)), nil
		}
		r = decimalFormatFloatRat(scaled, f.maxFraction, 32)
	}
	neg := r.Sign() < 0 || (r.Sign() == 0 && negativeZero)
	if r.Sign() < 0 {
		r.Neg(r)
	}
	if f.pattern.exponent {
		return f.formatExponent(r, neg)
	}
	units, err := roundRat(r, f.maxFraction, f.roundingMode, neg)
	if err != nil {
		return "", err
	}
	pow := pow10(f.maxFraction)
	integer, fraction := new(big.Int).QuoRem(units, pow, new(big.Int))
	intText := integer.String()
	if integer.Sign() == 0 && f.minInteger == 0 && f.minFraction > 0 {
		intText = ""
	}
	if f.maxInteger >= 0 && len(intText) > f.maxInteger {
		intText = intText[len(intText)-f.maxInteger:]
	}
	if len(intText) < f.minInteger {
		intText = strings.Repeat("0", f.minInteger-len(intText)) + intText
	}
	if f.groupingUsed && f.pattern.groupSize > 0 {
		intText = groupDigits(intText, f.pattern.groupSize, localeSymbols(f.loc).grouping)
	}
	fracText := ""
	if f.maxFraction > 0 {
		fracText = fraction.Text(10)
		fracText = strings.Repeat("0", f.maxFraction-len(fracText)) + fracText
		for len(fracText) > f.minFraction && strings.HasSuffix(fracText, "0") {
			fracText = fracText[:len(fracText)-1]
		}
	}
	numberText := intText
	if fracText != "" {
		numberText += localeSymbols(f.loc).decimal + fracText
	}
	prefix, suffix := f.pattern.positivePrefix, f.pattern.positiveSuffix
	if neg {
		prefix, suffix = f.pattern.negativePrefix, f.pattern.negativeSuffix
	}
	return expandAffix(prefix, f.loc) + numberText + expandAffix(suffix, f.loc), nil
}

func (f *NumberFormat) special(s string, neg bool) string {
	prefix, suffix := f.pattern.positivePrefix, f.pattern.positiveSuffix
	if neg {
		prefix, suffix = f.pattern.negativePrefix, f.pattern.negativeSuffix
	}
	return expandAffix(prefix, f.loc) + s + expandAffix(suffix, f.loc)
}

func (f *NumberFormat) formatExponent(r *big.Rat, neg bool) (string, error) {
	prefix, suffix := f.pattern.positivePrefix, f.pattern.positiveSuffix
	if neg {
		prefix, suffix = f.pattern.negativePrefix, f.pattern.negativeSuffix
	}
	if r.Sign() == 0 {
		minFraction := maxInt(0, f.minInteger+f.minFraction-1)
		zero := "0"
		if minFraction > 0 {
			zero += localeSymbols(f.loc).decimal + strings.Repeat("0", minFraction)
		}
		return expandAffix(prefix, f.loc) + zero + "E" + strings.Repeat("0", f.pattern.minExponent) + expandAffix(suffix, f.loc), nil
	}
	// DecimalFormat engineering notation uses the number of integer pattern
	// positions to choose an exponent divisible by that count.
	exp := ratDecimalExponent(r)
	intDigits := f.maxInteger
	if intDigits < 1 {
		intDigits = 1
	}
	if intDigits > f.minInteger {
		exp -= ((exp % intDigits) + intDigits) % intDigits
	} else {
		exp -= maxInt(f.minInteger-1, 0)
	}
	scaled := new(big.Rat).Set(r)
	if exp >= 0 {
		scaled.Quo(scaled, new(big.Rat).SetInt(pow10(exp)))
	} else {
		scaled.Mul(scaled, new(big.Rat).SetInt(pow10(-exp)))
	}
	clone := *f
	clone.pattern.exponent = false
	clone.pattern.positivePrefix, clone.pattern.positiveSuffix = "", ""
	clone.pattern.negativePrefix, clone.pattern.negativeSuffix = "-", ""
	clone.groupingUsed = false
	// In exponential mode DecimalFormat applies the integer and fraction
	// limits as one significant-digit budget.  Thus ##0.##E0 with a maximum
	// fraction setting of one still prints four significant digits when the
	// engineering mantissa has two integer digits (65.54E3), while a minimum
	// fraction setting is already satisfied by those integer digits.
	mantissaIntegerDigits := ratDecimalExponent(scaled) + 1
	clone.minInteger, clone.maxInteger = 1, maxInt(f.maxInteger, 1)
	clone.maxFraction = maxInt(0, f.maxInteger+f.maxFraction-mantissaIntegerDigits)
	clone.minFraction = maxInt(0, f.minInteger+f.minFraction-mantissaIntegerDigits)
	mantissa, err := clone.Format(scaled)
	if err != nil {
		return "", err
	}
	sign := ""
	if exp < 0 {
		sign, exp = "-", -exp
	}
	e := strconv.Itoa(exp)
	if len(e) < f.pattern.minExponent {
		e = strings.Repeat("0", f.pattern.minExponent-len(e)) + e
	}
	return expandAffix(prefix, f.loc) + mantissa + "E" + sign + e + expandAffix(suffix, f.loc), nil
}

func numberRat(v any) (*big.Rat, bool) {
	switch n := v.(type) {
	case int:
		return new(big.Rat).SetInt64(int64(n)), true
	case int8:
		return new(big.Rat).SetInt64(int64(n)), true
	case int16:
		return new(big.Rat).SetInt64(int64(n)), true
	case int32:
		return new(big.Rat).SetInt64(int64(n)), true
	case int64:
		return new(big.Rat).SetInt64(n), true
	case uint:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint8:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint16:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint32:
		return new(big.Rat).SetUint64(uint64(n)), true
	case uint64:
		return new(big.Rat).SetUint64(n), true
	case float32:
		return new(big.Rat).SetFloat64(float64(n)), true
	case float64:
		return new(big.Rat).SetFloat64(n), true
	case *big.Int:
		if n == nil {
			return nil, false
		}
		return new(big.Rat).SetInt(n), true
	case *big.Rat:
		if n == nil {
			return nil, false
		}
		return new(big.Rat).Set(n), true
	case jvm.BigDecimal:
		if n.Unscaled == nil {
			return nil, false
		}
		den := pow10(int(n.Scale))
		if n.Scale < 0 {
			return new(big.Rat).SetInt(new(big.Int).Mul(n.Unscaled, pow10(int(-n.Scale)))), true
		}
		return new(big.Rat).SetFrac(new(big.Int).Set(n.Unscaled), den), true
	default:
		return nil, false
	}
}

func decimalFormatFloatRat(v float64, fractionDigits, bitSize int) *big.Rat {
	exact := new(big.Rat).SetFloat64(v)
	if v == 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return exact
	}
	abs := math.Abs(v)
	// FloatingDecimal emits exact integral digits while the binary exponent is
	// at most 57. Above that boundary DigitList retains 17 significant digits
	// through the next decimal decade.
	if bitSize == 64 && math.Trunc(v) == v {
		_, binaryExponent := math.Frexp(abs)
		if binaryExponent-1 <= 57 {
			return exact
		}
	}
	if bitSize == 64 && math.Trunc(v) == v && abs < 1e19 {
		exponent := int(math.Floor(math.Log10(abs)))
		drop := exponent - 16
		if drop > 0 {
			factor := pow10(drop)
			scaled := new(big.Rat).Quo(exact, new(big.Rat).SetInt(factor))
			negative := scaled.Sign() < 0
			if negative {
				scaled.Neg(scaled)
			}
			// HALF_EVEN never produces the UNNECESSARY rounding error.
			rounded, _ := roundRat(scaled, 0, "HALF_EVEN", negative)
			if negative {
				rounded.Neg(rounded)
			}
			return new(big.Rat).SetInt(new(big.Int).Mul(rounded, factor))
		}
		return exact
	}
	short := strconv.FormatFloat(v, 'g', -1, bitSize)
	mantissa := short
	if i := strings.IndexAny(short, "eE"); i >= 0 {
		mantissa = short[:i]
	}
	mantissa = strings.ReplaceAll(strings.TrimLeft(mantissa, "+-"), ".", "")
	digits := len(strings.TrimLeft(mantissa, "0"))
	if bitSize == 64 && abs < math.SmallestNonzeroFloat64*0x1p52 && digits < 2 {
		return roundRatToSignificant(exact, 2)
	}
	if bitSize == 64 && math.Trunc(v) == v && abs >= 1e19 && digits == 1 {
		return roundRatToSignificant(exact, 16)
	}
	exponent := int(math.Floor(math.Log10(math.Abs(v))))
	targetSignificant := exponent + 1 + fractionDigits
	if targetSignificant > digits {
		if decimal, ok := new(big.Rat).SetString(short); ok {
			return decimal
		}
	}
	return exact
}

func roundRatToSignificant(exact *big.Rat, digits int) *big.Rat {
	exponent := ratDecimalExponent(new(big.Rat).Abs(exact))
	drop := exponent - digits + 1
	if drop < 0 {
		scale := -drop
		magnitude := new(big.Rat).Abs(exact)
		rounded, _ := roundRat(magnitude, scale, "HALF_EVEN", exact.Sign() < 0)
		if exact.Sign() < 0 {
			rounded.Neg(rounded)
		}
		return new(big.Rat).SetFrac(rounded, pow10(scale))
	}
	if drop == 0 {
		return exact
	}
	factor := pow10(drop)
	scaled := new(big.Rat).Quo(exact, new(big.Rat).SetInt(factor))
	negative := scaled.Sign() < 0
	if negative {
		scaled.Neg(scaled)
	}
	rounded, _ := roundRat(scaled, 0, "HALF_EVEN", negative)
	if negative {
		rounded.Neg(rounded)
	}
	return new(big.Rat).SetInt(new(big.Int).Mul(rounded, factor))
}

func shortestDecimalDigits(v float64, bitSize int) int {
	short := strconv.FormatFloat(v, 'g', -1, bitSize)
	if i := strings.IndexAny(short, "eE"); i >= 0 {
		short = short[:i]
	}
	short = strings.ReplaceAll(strings.TrimLeft(short, "+-"), ".", "")
	return len(strings.TrimLeft(short, "0"))
}

func pow10(n int) *big.Int {
	if n <= 0 {
		return big.NewInt(1)
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

func roundRat(r *big.Rat, scale int, mode string, negative bool) (*big.Int, error) {
	if scale < 0 {
		scale = 0
	}
	n := new(big.Int).Mul(r.Num(), pow10(scale))
	q, rem := new(big.Int).QuoRem(n, r.Denom(), new(big.Int))
	if rem.Sign() == 0 {
		return q, nil
	}
	inc := false
	switch mode {
	case "UNNECESSARY":
		return nil, jvm.Throw("java.lang.ArithmeticException", "Rounding needed with the rounding mode being set to RoundingMode.UNNECESSARY")
	case "UP":
		inc = true
	case "DOWN":
	case "CEILING":
		inc = !negative
	case "FLOOR":
		inc = negative
	case "HALF_UP", "HALF_DOWN", "HALF_EVEN":
		cmp := new(big.Int).Lsh(rem, 1).Cmp(r.Denom())
		inc = cmp > 0 || (cmp == 0 && (mode == "HALF_UP" || (mode == "HALF_EVEN" && q.Bit(0) == 1)))
	}
	if inc {
		q.Add(q, big.NewInt(1))
	}
	return q, nil
}

func groupDigits(s string, size int, sep string) string {
	if size <= 0 || len(s) <= size {
		return s
	}
	first := len(s) % size
	if first == 0 {
		first = size
	}
	var b strings.Builder
	b.WriteString(s[:first])
	for i := first; i < len(s); i += size {
		b.WriteString(sep)
		b.WriteString(s[i : i+size])
	}
	return b.String()
}

func expandAffix(s string, loc jvm.Locale) string {
	sy := localeSymbols(loc)
	s = strings.ReplaceAll(s, "¤¤", sy.currencyCode)
	s = strings.ReplaceAll(s, "¤", sy.currency)
	return s
}

func splitUnquoted(s string, sep rune) []string {
	quoted := false
	start := 0
	var out []string
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == '\'' {
			if i+n < len(s) && s[i+n] == '\'' {
				i += n + 1
				continue
			}
			quoted = !quoted
		} else if r == sep && !quoted {
			out = append(out, s[start:i])
			start = i + n
		}
		i += n
	}
	return append(out, s[start:])
}

func parseDecimalPattern(pattern string, loc jvm.Locale) (decimalPattern, error) {
	parts := splitUnquoted(pattern, ';')
	if len(parts) > 2 {
		return decimalPattern{}, patternError("Unquoted special character ';'", pattern)
	}
	if len(parts) == 2 && parts[0] == "" {
		return decimalPattern{}, patternError("Unquoted special character ';'", pattern)
	}
	pos, err := parseSubpattern(parts[0], pattern)
	if err != nil {
		return decimalPattern{}, err
	}
	pos.negativePrefix, pos.negativeSuffix = "-"+pos.positivePrefix, pos.positiveSuffix
	if len(parts) == 2 && parts[1] != "" {
		neg, e := parseSubpattern(parts[1], pattern)
		if e != nil {
			return decimalPattern{}, e
		}
		pos.negativePrefix, pos.negativeSuffix = neg.positivePrefix, neg.positiveSuffix
	}
	return pos, nil
}

func parseSubpattern(s, whole string) (decimalPattern, error) {
	p := decimalPattern{multiplier: 1, maxInteger: 309}
	var prefix, suffix, number strings.Builder
	quoted := false
	numberStarted := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\'' {
			if i+size < len(s) && s[i+size] == '\'' {
				if numberStarted {
					suffix.WriteByte('\'')
				} else {
					prefix.WriteByte('\'')
				}
				i += size + 1
				continue
			}
			quoted = !quoted
			i += size
			continue
		}
		isNumber := !quoted && strings.ContainsRune("0#,.", r)
		if !quoted && r == 'E' && numberStarted {
			isNumber = true
		}
		if isNumber {
			numberStarted = true
			number.WriteRune(r)
		} else if numberStarted {
			suffix.WriteRune(r)
		} else {
			prefix.WriteRune(r)
		}
		i += size
	}
	if quoted {
		return p, patternError("Malformed pattern", whole)
	}
	if !numberStarted {
		// With no digit section DecimalFormat treats the complete pattern as a
		// literal positive prefix and supplies an implicit optional digit.
		number.WriteByte('#')
	}
	p.positivePrefix = prefix.String()
	p.positiveSuffix = suffix.String()
	for _, aff := range []string{p.positivePrefix, p.positiveSuffix} {
		if containsUnquoted(aff, '%') {
			p.multiplier = 100
		}
		if containsUnquoted(aff, '‰') {
			p.multiplier = 1000
		}
	}
	if countUnquoted(s, '%')+countUnquoted(s, '‰') > 1 {
		return p, patternError("Too many percent/per mille characters in pattern", whole)
	}
	num := number.String()
	if strings.Count(num, ".") > 1 {
		return p, patternError("Multiple decimal separators in pattern", whole)
	}
	if strings.Count(num, "E") > 1 {
		return p, patternError("Malformed exponential pattern", whole)
	}
	base := num
	if at := strings.Index(base, "E"); at >= 0 {
		p.exponent = true
		exp := base[at+1:]
		if exp == "" || strings.Trim(exp, "0") != "" {
			return p, patternError("Malformed exponential pattern", whole)
		}
		p.minExponent = len(exp)
		base = base[:at]
	}
	dot := strings.IndexByte(base, '.')
	integer, fraction := base, ""
	if dot >= 0 {
		integer, fraction = base[:dot], base[dot+1:]
	}
	plainInteger := strings.ReplaceAll(integer, ",", "")
	if strings.HasSuffix(plainInteger, "#") && strings.Contains(plainInteger, "0") {
		return p, patternError("Malformed pattern", whole)
	}
	if strings.Contains(plainInteger, "0#") {
		return p, patternError("Unexpected '0' in pattern", whole)
	}
	if strings.Trim(plainInteger, "0#") != "" || strings.Trim(fraction, "0#") != "" {
		return p, patternError("Malformed pattern", whole)
	}
	p.minInteger = strings.Count(plainInteger, "0")
	if p.exponent {
		p.maxInteger = len(plainInteger)
	}
	p.minFraction = strings.Count(fraction, "0")
	p.maxFraction = len(fraction)
	if strings.Contains(fraction, "#0") {
		return p, patternError("Malformed pattern", whole)
	}
	if comma := strings.LastIndexByte(integer, ','); comma >= 0 {
		p.grouping = true
		p.groupSize = len(integer) - comma - 1
		if p.groupSize == 0 {
			return p, patternError("Malformed pattern", whole)
		}
	}
	return p, nil
}

func countUnquoted(s string, target rune) int {
	count := 0
	quoted := false
	for _, r := range s {
		if r == '\'' {
			quoted = !quoted
		} else if !quoted && r == target {
			count++
		}
	}
	return count
}

func containsUnquoted(s string, target rune) bool {
	quoted := false
	for _, r := range s {
		if r == '\'' {
			quoted = !quoted
		} else if !quoted && r == target {
			return true
		}
	}
	return false
}

func patternError(reason, pattern string) error {
	return jvm.Throw("java.lang.IllegalArgumentException", reason+" \""+pattern+"\"")
}
