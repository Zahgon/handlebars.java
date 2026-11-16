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
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jknack/handlebars.java/internal/jvm"
)

type formatSpec struct {
	raw       string
	index     int
	reuse     bool
	flags     string
	width     int
	precision int
	conv      byte
}

// Format implements String.format(loc, format, args).
func Format(loc jvm.Locale, format string, args []any) (string, error) {
	// java.util.Formatter parses and validates every specifier when the format
	// string is constructed, before it reads or converts any argument. A later
	// syntactic error therefore wins over an earlier argument type/missing-arg
	// error (for example %E%.1d reports IllegalFormatPrecisionException).
	if err := prevalidateFormat(format); err != nil {
		return "", err
	}
	var out strings.Builder
	ordinaryIndex, lastIndex := 0, -1
	for i := 0; i < len(format); {
		if format[i] != '%' {
			out.WriteByte(format[i])
			i++
			continue
		}
		spec, next, err := parseFormatSpec(format, i)
		if err != nil {
			return "", err
		}
		i = next
		if spec.conv == '%' {
			if spec.precision >= 0 {
				return "", illegalFormatPrecision(spec.precision)
			}
			if strings.Contains(spec.flags, "-") && strings.Contains(spec.flags, "0") {
				return "", illegalFormatFlags("-0")
			}
			bad := invalidFlags(spec.flags, "-")
			if spec.reuse {
				bad += "<"
			}
			if bad != "" {
				return "", illegalFormatFlags(bad)
			}
			if strings.Contains(spec.flags, "-") && spec.width < 0 {
				return "", jvm.Throw("java.util.MissingFormatWidthException", spec.raw)
			}
			out.WriteString(padFormat("%", spec, ' '))
			continue
		}
		if spec.conv == 'n' {
			if spec.precision >= 0 {
				return "", illegalFormatPrecision(spec.precision)
			}
			if spec.width >= 0 {
				return "", illegalFormatWidth(spec.width)
			}
			bad := invalidFlags(spec.flags, "")
			if spec.reuse {
				bad += "<"
			}
			if bad != "" {
				return "", illegalFormatFlags(bad)
			}
			out.WriteByte('\n')
			continue
		}
		if err := prevalidateFormatSpec(spec); err != nil {
			return "", err
		}
		argIndex := ordinaryIndex
		if spec.reuse {
			argIndex = lastIndex
		} else if spec.index >= 0 {
			argIndex = spec.index
		} else {
			ordinaryIndex++
		}
		if argIndex < 0 || argIndex >= len(args) {
			raw := spec.raw
			if spec.reuse && spec.index >= 0 {
				raw = "%<" + raw[strings.LastIndexByte(raw, '<')+1:]
			}
			return "", jvm.Throw("java.util.MissingFormatArgumentException", "Format specifier '"+raw+"'")
		}
		lastIndex = argIndex
		piece, err := formatArgument(loc, spec, args[argIndex])
		if err != nil {
			return "", err
		}
		out.WriteString(piece)
	}
	return out.String(), nil
}

func prevalidateFormat(format string) error {
	for i := 0; i < len(format); {
		if format[i] != '%' {
			i++
			continue
		}
		spec, next, err := parseFormatSpec(format, i)
		if err != nil {
			return err
		}
		i = next
		switch spec.conv {
		case '%':
			if spec.precision >= 0 {
				return illegalFormatPrecision(spec.precision)
			}
			if strings.Contains(spec.flags, "-") && strings.Contains(spec.flags, "0") {
				return illegalFormatFlags("-0")
			}
			bad := invalidFlags(spec.flags, "-")
			if spec.reuse {
				bad += "<"
			}
			if bad != "" {
				return illegalFormatFlags(bad)
			}
			if strings.Contains(spec.flags, "-") && spec.width < 0 {
				return jvm.Throw("java.util.MissingFormatWidthException", spec.raw)
			}
		case 'n':
			if spec.precision >= 0 {
				return illegalFormatPrecision(spec.precision)
			}
			if spec.width >= 0 {
				return illegalFormatWidth(spec.width)
			}
			bad := invalidFlags(spec.flags, "")
			if spec.reuse {
				bad += "<"
			}
			if bad != "" {
				return illegalFormatFlags(bad)
			}
		default:
			if err := prevalidateFormatSpec(spec); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseFormatSpec(format string, start int) (formatSpec, int, error) {
	s := formatSpec{index: -1, width: -1, precision: -1}
	i := start + 1
	if i >= len(format) {
		return s, i, jvm.Throw("java.util.UnknownFormatConversionException", "Conversion = '%'")
	}
	numStart := i
	for i < len(format) && format[i] >= '0' && format[i] <= '9' {
		i++
	}
	if i < len(format) && i > numStart && format[i] == '$' {
		n, parseErr := strconv.ParseInt(format[numStart:i], 10, 32)
		if parseErr != nil {
			return s, i, jvm.Throw("java.util.IllegalFormatArgumentIndexException", "Format argument index: (not representable as int)")
		}
		if n == 0 {
			return s, i, jvm.Throw("java.util.IllegalFormatArgumentIndexException", "Illegal format argument index = 0")
		}
		s.index = int(n) - 1
		i++
	} else {
		i = numStart
	}
	seen := map[byte]bool{}
	for i < len(format) && strings.ContainsRune("-#+ 0,(<", rune(format[i])) {
		ch := format[i]
		if seen[ch] {
			return s, i, jvm.Throw("java.util.DuplicateFormatFlagsException", "Flags = '"+string(ch)+"'")
		}
		seen[ch] = true
		if ch == '<' {
			s.reuse = true
		} else {
			s.flags += string(ch)
		}
		i++
	}
	widthStart := i
	for i < len(format) && format[i] >= '0' && format[i] <= '9' {
		i++
	}
	if i > widthStart {
		n, parseErr := strconv.ParseInt(format[widthStart:i], 10, 32)
		if parseErr != nil {
			return s, i, illegalFormatWidth(math.MinInt32)
		}
		s.width = int(n)
	}
	if i < len(format) && format[i] == '.' {
		i++
		pStart := i
		for i < len(format) && format[i] >= '0' && format[i] <= '9' {
			i++
		}
		if pStart == i {
			return s, i, jvm.Throw("java.util.UnknownFormatConversionException", "Conversion = '.'")
		}
		n, parseErr := strconv.ParseInt(format[pStart:i], 10, 32)
		if parseErr != nil {
			return s, i, illegalFormatPrecision(math.MinInt32)
		}
		s.precision = int(n)
	}
	if i >= len(format) {
		return s, i, jvm.Throw("java.util.UnknownFormatConversionException", "Conversion = '"+format[start+1:]+"'")
	}
	s.conv = format[i]
	if !strings.ContainsRune("bBhHsScCdoxXeEfgGaA%n", rune(s.conv)) {
		return s, i + 1, jvm.Throw("java.util.UnknownFormatConversionException", "Conversion = '"+string(s.conv)+"'")
	}
	s.raw = format[start : i+1]
	return s, i + 1, nil
}

func formatArgument(loc jvm.Locale, s formatSpec, arg any) (string, error) {
	upper := s.conv >= 'A' && s.conv <= 'Z'
	conv := s.conv
	if upper {
		conv += 'a' - 'A'
	}
	if strings.ContainsRune("bhs", rune(conv)) {
		allowed := "-"
		if conv == 's' {
			allowed = "-#"
		}
		if conv == 's' && strings.Contains(s.flags, "0") {
			return "", flagsMismatch("0", conv)
		}
		if strings.Contains(s.flags, "#") {
			return "", flagsMismatch("#", conv)
		}
		if bad := invalidFlags(s.flags, allowed); bad != "" {
			return "", flagsMismatch(bad, conv)
		}
		if strings.Contains(s.flags, "#") {
			return "", flagsMismatch("#", conv)
		}
		var v string
		switch conv {
		case 's':
			v = jvm.ToString(arg)
		case 'b':
			if arg == nil {
				v = "false"
			} else if b, ok := arg.(bool); ok {
				v = strconv.FormatBool(b)
			} else {
				v = "true"
			}
		case 'h':
			if arg == nil {
				v = "null"
			} else {
				v = strconv.FormatUint(uint64(uint32(jvm.HashCode(arg))), 16)
			}
		}
		if s.precision >= 0 {
			v = truncateRunes(v, s.precision)
		}
		if upper {
			v = localeUpper(v, loc)
		}
		return padFormat(v, s, ' '), nil
	}
	if conv == 'c' {
		if bad := invalidFlags(s.flags, "-"); bad != "" {
			return "", flagsMismatch(bad, conv)
		}
		if arg == nil {
			v := "null"
			if upper {
				v = localeUpper(v, loc)
			}
			return padFormat(v, s, ' '), nil
		}
		cp, ok := charCode(arg)
		if !ok {
			return "", illegalConversion(s.conv, arg)
		}
		if cp < 0 || cp > utf8.MaxRune || cp >= 0xD800 && cp <= 0xDFFF {
			return "", jvm.Throw("java.util.IllegalFormatCodePointException", "Code point = 0x"+strconv.FormatUint(uint64(uint32(cp)), 16))
		}
		v := string(rune(cp))
		if upper {
			v = localeUpper(v, loc)
		}
		return padFormat(v, s, ' '), nil
	}
	if strings.ContainsRune("dox", rune(conv)) {
		return formatInteger(loc, s, arg, conv, upper)
	}
	return formatFloat(loc, s, arg, conv, upper)
}

func formatInteger(loc jvm.Locale, s formatSpec, arg any, conv byte, upper bool) (string, error) {
	allowed := "-0"
	_, isBigInteger := arg.(*big.Int)
	if conv == 'd' {
		allowed += "+ ,("
	} else {
		allowed += "#"
		if isBigInteger {
			allowed += "+ ("
		}
	}
	if err := validateCommonFlags(s); err != nil {
		return "", err
	}
	if arg == nil {
		v := "null"
		if upper {
			v = localeUpper(v, loc)
		}
		return padFormat(v, s, ' '), nil
	}
	n, bits, ok := integerValue(arg)
	if !ok {
		return "", illegalConversion(s.conv, arg)
	}
	if bad := firstNumericInvalid(s.flags, allowed); bad != "" {
		return "", flagsMismatch(bad, conv)
	}
	neg := n.Sign() < 0
	digits := ""
	if conv == 'd' {
		digits = new(big.Int).Abs(n).Text(10)
		if strings.Contains(s.flags, ",") {
			digits = groupDigits(digits, 3, localeSymbols(loc).grouping)
		}
	} else {
		base := 8
		if conv == 'x' {
			base = 16
		}
		if neg && bits > 0 {
			mod := new(big.Int).Lsh(big.NewInt(1), uint(bits))
			digits = new(big.Int).Add(n, mod).Text(base)
		} else {
			digits = new(big.Int).Abs(n).Text(base)
		}
		if strings.Contains(s.flags, "#") {
			if conv == 'o' {
				digits = "0" + digits
			} else {
				digits = "0x" + digits
			}
		}
	}
	if upper {
		digits = strings.ToUpper(digits)
	}
	prefix, suffix := "", ""
	if neg && (conv == 'd' || isBigInteger) {
		if strings.Contains(s.flags, "(") {
			prefix, suffix = "(", ")"
		} else {
			prefix = "-"
		}
	} else if conv == 'd' || isBigInteger {
		if strings.Contains(s.flags, "+") {
			prefix = "+"
		} else if strings.Contains(s.flags, " ") {
			prefix = " "
		}
	}
	if conv != 'd' && (strings.HasPrefix(digits, "0x") || strings.HasPrefix(digits, "0X")) {
		prefix += digits[:2]
		digits = digits[2:]
	}
	v := prefix + digits + suffix
	return numericPad(v, prefix, digits, suffix, s), nil
}

func formatFloat(loc jvm.Locale, s formatSpec, arg any, conv byte, upper bool) (string, error) {
	allowed := "-+ 0("
	if conv == 'f' || conv == 'g' {
		allowed += ","
	}
	if conv == 'e' || conv == 'f' || conv == 'a' {
		allowed += "#"
	}
	if conv == 'a' {
		allowed = "-#+ 0"
	}
	badFlags := firstNumericInvalid(s.flags, allowed)
	if conv == 'a' {
		badFlags = invalidFlags(s.flags, allowed)
	}
	if badFlags != "" {
		return "", flagsMismatch(badFlags, conv)
	}
	if err := validateCommonFlags(s); err != nil {
		return "", err
	}
	if arg == nil {
		v := "null"
		if s.precision >= 0 {
			v = truncateRunes(v, s.precision)
		}
		if upper {
			v = localeUpper(v, loc)
		}
		return padFormat(v, s, ' '), nil
	}
	var exactDecimal *big.Rat
	var decimalScale *int32
	if decimal, ok := arg.(jvm.BigDecimal); ok {
		if conv == 'a' {
			return "", illegalConversion(s.conv, arg)
		}
		// A modeled BigDecimal always has a numeric rational representation.
		exactDecimal, _ = numberRat(decimal)
		decimalScale = &decimal.Scale
	}
	n, ok := floatingValue(arg)
	if !ok {
		return "", illegalConversion(s.conv, arg)
	}
	if exactDecimal != nil {
		n = 0
	}
	neg := math.Signbit(n)
	if exactDecimal != nil {
		neg = exactDecimal.Sign() < 0
		if neg {
			exactDecimal.Neg(exactDecimal)
		}
	}
	abs := math.Abs(n)
	precision := s.precision
	if precision < 0 && conv != 'a' {
		precision = 6
	}
	if conv == 'g' && precision == 0 {
		precision = 1
	}
	var digits string
	if math.IsNaN(n) {
		digits, neg = "NaN", false
	} else if math.IsInf(n, 0) {
		digits = "Infinity"
	} else {
		switch conv {
		case 'f':
			if exactDecimal != nil {
				digits = decimalRatFixed(exactDecimal, precision)
			} else {
				digits = decimalFloat(abs, precision, false)
			}
			if strings.Contains(s.flags, ",") {
				if dot := strings.IndexByte(digits, '.'); dot >= 0 {
					digits = groupDigits(digits[:dot], 3, "\x00") + digits[dot:]
				} else {
					digits = groupDigits(digits, 3, "\x00")
				}
			}
		case 'e':
			if exactDecimal != nil {
				digits = decimalRatScientific(exactDecimal, precision, upper)
			} else {
				digits = decimalScientific(abs, precision, upper)
			}
		case 'g':
			if exactDecimal != nil {
				if exactDecimal.Sign() == 0 && decimalScale != nil && *decimalScale != 0 {
					digits = decimalZeroScientific(*decimalScale, precision, upper)
				} else {
					digits = generalRat(exactDecimal, precision, upper)
				}
			} else {
				digits = generalFloat(abs, precision, upper)
			}
		case 'a':
			hexPrecision := precision
			if hexPrecision == 0 {
				hexPrecision = 1
			}
			digits = strconv.FormatFloat(abs, 'x', hexPrecision, 64)
			// Go pads the binary exponent to two digits; Formatter does not.
			if i := strings.LastIndexByte(digits, 'p'); i >= 0 {
				head, exponent := digits[:i+1], digits[i+1:]
				sign := ""
				if strings.HasPrefix(exponent, "-") {
					sign, exponent = exponent[:1], exponent[1:]
				} else if strings.HasPrefix(exponent, "+") {
					exponent = exponent[1:]
				}
				exponent = strings.TrimLeft(exponent, "0")
				if exponent == "" {
					exponent = "0"
				}
				digits = head + sign + exponent
			}
			if p := strings.LastIndexByte(digits, 'p'); p >= 0 && !strings.Contains(digits[:p], ".") {
				digits = digits[:p] + ".0" + digits[p:]
			}
		}
	}
	if conv != 'a' && !math.IsNaN(n) {
		digits = localizeDecimal(digits, loc)
		digits = strings.ReplaceAll(digits, "\x00", localeSymbols(loc).grouping)
	}
	if upper {
		digits = localeUpper(digits, loc)
	}
	prefix, suffix := "", ""
	if neg {
		if strings.Contains(s.flags, "(") {
			prefix, suffix = "(", ")"
		} else {
			prefix = "-"
		}
	} else if strings.Contains(s.flags, "+") {
		prefix = "+"
	} else if strings.Contains(s.flags, " ") {
		prefix = " "
	}
	if strings.Contains(s.flags, "#") && !strings.ContainsAny(digits, ".,") && !math.IsInf(n, 0) && !math.IsNaN(n) {
		if i := strings.IndexAny(digits, "eE"); i >= 0 {
			digits = digits[:i] + localeSymbols(loc).decimal + digits[i:]
		} else {
			digits += localeSymbols(loc).decimal
		}
	}
	if conv == 'a' && (strings.HasPrefix(digits, "0x") || strings.HasPrefix(digits, "0X")) {
		prefix += digits[:2]
		digits = digits[2:]
	}
	return numericPad(prefix+digits+suffix, prefix, digits, suffix, s), nil
}

func decimalZeroScientific(scale int32, precision int, upper bool) string {
	letter := "e"
	if upper {
		letter = "E"
	}
	exp := int64(-scale)
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	e := strconv.FormatInt(exp, 10)
	if len(e) < 2 {
		e = "0" + e
	}
	return decimalRatFixed(new(big.Rat), precision-1) + letter + sign + e
}

func decimalFloat(v float64, precision int, _ bool) string {
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(v, 'g', -1, 64))
	if !ok {
		return "0"
	}
	// HALF_UP never produces the UNNECESSARY rounding error.
	u, _ := roundRat(r, precision, "HALF_UP", false)
	p := pow10(precision)
	i, frac := new(big.Int).QuoRem(u, p, new(big.Int))
	if precision == 0 {
		return i.String()
	}
	f := frac.String()
	f = strings.Repeat("0", precision-len(f)) + f
	return i.String() + "." + f
}

func decimalRatFixed(r *big.Rat, precision int) string {
	// HALF_UP never produces the UNNECESSARY rounding error.
	u, _ := roundRat(r, precision, "HALF_UP", false)
	p := pow10(precision)
	i, frac := new(big.Int).QuoRem(u, p, new(big.Int))
	if precision == 0 {
		return i.String()
	}
	f := frac.String()
	f = strings.Repeat("0", precision-len(f)) + f
	return i.String() + "." + f
}

func ratDecimalExponent(r *big.Rat) int {
	if r.Sign() == 0 {
		return 0
	}
	exp := len(r.Num().String()) - len(r.Denom().String())
	p := new(big.Rat).SetInt(pow10(maxInt(exp, 0)))
	if exp < 0 {
		p.Inv(new(big.Rat).SetInt(pow10(-exp)))
	}
	if r.Cmp(p) < 0 {
		exp--
	} else {
		next := new(big.Rat).Mul(p, big.NewRat(10, 1))
		if r.Cmp(next) >= 0 {
			exp++
		}
	}
	return exp
}

func decimalRatScientific(r *big.Rat, precision int, upper bool) string {
	if r.Sign() == 0 {
		letter := "e"
		if upper {
			letter = "E"
		}
		return decimalRatFixed(r, precision) + letter + "+00"
	}
	exp := ratDecimalExponent(r)
	scaled := new(big.Rat).Set(r)
	if exp >= 0 {
		scaled.Quo(scaled, new(big.Rat).SetInt(pow10(exp)))
	} else {
		scaled.Mul(scaled, new(big.Rat).SetInt(pow10(-exp)))
	}
	text := decimalRatFixed(scaled, precision)
	if strings.HasPrefix(text, "10") {
		exp++
		text = decimalRatFixed(big.NewRat(1, 1), precision)
	}
	letter := "e"
	if upper {
		letter = "E"
	}
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	e := strconv.Itoa(exp)
	if len(e) < 2 {
		e = "0" + e
	}
	return text + letter + sign + e
}

func generalRat(r *big.Rat, precision int, upper bool) string {
	if r.Sign() == 0 {
		return decimalRatFixed(r, precision-1)
	}
	exp := ratDecimalExponent(r)
	if exp < -4 || exp >= precision {
		return decimalRatScientific(r, precision-1, upper)
	}
	return decimalRatFixed(r, maxInt(precision-exp-1, 0))
}

func decimalScientific(v float64, precision int, upper bool) string {
	if v == 0 {
		e := "e+00"
		if upper {
			e = "E+00"
		}
		return decimalFloat(0, precision, false) + e
	}
	if v < math.SmallestNonzeroFloat64*0x1p52 && shortestDecimalDigits(v, 64) < 2 {
		return decimalRatScientific(roundRatToSignificant(new(big.Rat).SetFloat64(v), 2), precision, upper)
	}
	exp := int(math.Floor(math.Log10(v)))
	m := v / math.Pow10(exp)
	text := decimalFloat(m, precision, false)
	if strings.HasPrefix(text, "10") {
		exp++
		text = decimalFloat(1, precision, false)
	}
	letter := "e"
	if upper {
		letter = "E"
	}
	sign := "+"
	if exp < 0 {
		sign = "-"
		exp = -exp
	}
	e := strconv.Itoa(exp)
	if len(e) < 2 {
		e = "0" + e
	}
	return text + letter + sign + e
}

func generalFloat(v float64, precision int, upper bool) string {
	if v == 0 {
		return decimalFloat(0, precision-1, false)
	}
	exp := int(math.Floor(math.Log10(v)))
	var s string
	if exp < -4 || exp >= precision {
		s = decimalScientific(v, precision-1, upper)
	} else {
		s = decimalFloat(v, maxInt(precision-exp-1, 0), false)
	}
	return s
}

func localizeDecimal(s string, loc jvm.Locale) string {
	if localeSymbols(loc).decimal != "." {
		if i := strings.IndexByte(s, '.'); i >= 0 {
			s = s[:i] + localeSymbols(loc).decimal + s[i+1:]
		}
	}
	return s
}

func integerValue(v any) (*big.Int, int, bool) {
	switch n := v.(type) {
	case int8:
		return big.NewInt(int64(n)), 8, true
	case int16:
		return big.NewInt(int64(n)), 16, true
	case int32:
		return big.NewInt(int64(n)), 32, true
	case int64:
		return big.NewInt(n), 64, true
	case int:
		return big.NewInt(int64(n)), strconv.IntSize, true
	case *big.Int:
		if n != nil {
			return new(big.Int).Set(n), 0, true
		}
	}
	return nil, 0, false
}

func floatingValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case jvm.BigDecimal:
		r, ok := numberRat(n)
		if ok {
			// The accuracy result is immaterial to Number.doubleValue semantics.
			f, _ := r.Float64()
			return f, true
		}
	}
	return 0, false
}

func charCode(v any) (int64, bool) {
	switch n := v.(type) {
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case jvm.Char:
		return int64(n), true
	}
	return 0, false
}

func validateCommonFlags(s formatSpec) error {
	if strings.Contains(s.flags, "+") && strings.Contains(s.flags, " ") {
		return illegalFormatFlags("+ ")
	}
	if strings.Contains(s.flags, "-") && strings.Contains(s.flags, "0") {
		return illegalFormatFlags("-0")
	}
	if (strings.Contains(s.flags, "-") || strings.Contains(s.flags, "0")) && s.width < 0 {
		return jvm.Throw("java.util.MissingFormatWidthException", s.raw)
	}
	return nil
}

func prevalidateFormatSpec(s formatSpec) error {
	conv := s.conv
	if conv >= 'A' && conv <= 'Z' {
		conv += 'a' - 'A'
	}
	if conv == 'c' && s.precision >= 0 {
		return illegalFormatPrecision(s.precision)
	}
	if conv == 'c' {
		if bad := invalidFlags(s.flags, "-"); bad != "" {
			return flagsMismatch(bad, conv)
		}
	}
	if strings.Contains(s.flags, "-") && s.width < 0 {
		return jvm.Throw("java.util.MissingFormatWidthException", s.raw)
	}
	if strings.ContainsRune("doxefga", rune(conv)) && strings.Contains(s.flags, "0") && s.width < 0 {
		return jvm.Throw("java.util.MissingFormatWidthException", s.raw)
	}
	if strings.ContainsRune("doxefga", rune(conv)) {
		if strings.Contains(s.flags, "+") && strings.Contains(s.flags, " ") {
			return illegalFormatFlags(withUpperFlag("+ ", s.conv))
		}
		if strings.Contains(s.flags, "-") && strings.Contains(s.flags, "0") {
			return illegalFormatFlags(withUpperFlag("-0", s.conv))
		}
	}
	if strings.ContainsRune("cdox", rune(conv)) && s.precision >= 0 {
		return illegalFormatPrecision(s.precision)
	}
	if conv == 'd' && strings.Contains(s.flags, "#") {
		return flagsMismatch("#", conv)
	}
	if (conv == 'o' || conv == 'x') && strings.Contains(s.flags, ",") {
		return flagsMismatch(",", conv)
	}
	allowed := ""
	switch conv {
	case 'b', 'h':
		allowed = "-"
	case 's':
		// '#' is legal for Formattable and therefore cannot be rejected while
		// parsing the format string. Plain strings reject it when this
		// conversion is actually reached.
		allowed = "-#"
	case 'c':
		allowed = "-"
	case 'd':
		allowed = "-+ 0,("
	case 'o', 'x':
		allowed = "-#0"
	case 'e':
		allowed = "-#+ 0("
	case 'f':
		allowed = "-#+ 0,("
	case 'g':
		allowed = "-+ 0,("
	case 'a':
		allowed = "-#+ 0"
	}
	if conv == 's' && strings.Contains(s.flags, "0") {
		return flagsMismatch("0", conv)
	}
	if (conv == 'b' || conv == 'h') && strings.Contains(s.flags, "#") {
		return flagsMismatch("#", conv)
	}
	if strings.ContainsRune("doxefga", rune(conv)) {
		return nil
	}
	if bad := invalidFlags(s.flags, allowed); bad != "" {
		return flagsMismatch(bad, conv)
	}
	return nil
}

func numericPad(v, prefix, digits, suffix string, s formatSpec) string {
	if s.width <= jvm.UTF16Len(v) {
		return v
	}
	n := s.width - jvm.UTF16Len(v)
	if strings.Contains(s.flags, "-") {
		return v + strings.Repeat(" ", n)
	}
	if strings.Contains(s.flags, "0") {
		return prefix + strings.Repeat("0", n) + digits + suffix
	}
	return strings.Repeat(" ", n) + v
}

func padFormat(v string, s formatSpec, pad byte) string {
	n := s.width - jvm.UTF16Len(v)
	if n <= 0 {
		return v
	}
	p := strings.Repeat(string(pad), n)
	if strings.Contains(s.flags, "-") {
		return v + p
	}
	return p + v
}

func invalidFlags(flags, allowed string) string {
	var b strings.Builder
	for _, flag := range "-#+ 0,(" {
		if strings.ContainsRune(flags, flag) && !strings.ContainsRune(allowed, flag) {
			b.WriteRune(flag)
		}
	}
	return b.String()
}
func firstNumericInvalid(flags, allowed string) string {
	for _, flag := range ",#+ (0-" {
		if strings.ContainsRune(flags, flag) && !strings.ContainsRune(allowed, flag) {
			return string(flag)
		}
	}
	return ""
}
func withUpperFlag(flags string, conv byte) string {
	if conv < 'A' || conv > 'Z' {
		return flags
	}
	if strings.HasPrefix(flags, "-") {
		return "-^" + flags[1:]
	}
	return "^" + flags
}
func truncateRunes(s string, n int) string {
	if n < 0 {
		return s
	}
	if jvm.UTF16Len(s) > n {
		// Bounds were established from the same UTF-16 length.
		result, _ := jvm.Substring(s, 0, n)
		return result
	}
	return s
}
func localeUpper(s string, loc jvm.Locale) string {
	if strings.EqualFold(loc.Language, "tr") {
		s = strings.ReplaceAll(s, "i", "İ")
		s = strings.ReplaceAll(s, "ı", "I")
	}
	s = strings.ReplaceAll(s, "ß", "SS")
	return strings.Map(unicode.ToUpper, s)
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func illegalConversion(conv byte, arg any) error {
	if conv >= 'A' && conv <= 'Z' {
		conv += 'a' - 'A'
	}
	return jvm.Throw("java.util.IllegalFormatConversionException", string(conv)+" != "+jvm.ClassName(arg))
}
func illegalFormatPrecision(n int) error {
	return jvm.Throw("java.util.IllegalFormatPrecisionException", strconv.Itoa(n))
}
func illegalFormatWidth(n int) error {
	return jvm.Throw("java.util.IllegalFormatWidthException", strconv.Itoa(n))
}
func illegalFormatFlags(flags string) error {
	return jvm.Throw("java.util.IllegalFormatFlagsException", "Flags = '"+flags+"'")
}
func flagsMismatch(flags string, conv byte) error {
	return jvm.Throw("java.util.FormatFlagsConversionMismatchException", "Conversion = "+string(conv)+", Flags = "+flags)
}
