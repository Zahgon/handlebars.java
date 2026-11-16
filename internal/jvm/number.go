// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// DoubleToString returns JDK 21 Double.toString formatting.
func DoubleToString(v float64) string { return javaFloatString(v, 64) }

// FloatToString returns JDK 21 Float.toString formatting.
func FloatToString(v float32) string { return javaFloatString(float64(v), 32) }

func javaFloatString(v float64, bits int) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "Infinity"
	}
	if math.IsInf(v, -1) {
		return "-Infinity"
	}
	if v == 0 {
		if math.Signbit(v) {
			return "-0.0"
		}
		return "0.0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	e := strconv.FormatFloat(v, 'e', -1, bits)
	p := strings.IndexByte(e, 'e')
	digits := strings.ReplaceAll(e[:p], ".", "")
	exp, _ := strconv.Atoi(e[p+1:])
	if len(strings.TrimRight(digits, "0")) == 1 {
		// The JDK specification chooses the closest member of the shortest
		// decimal family after accounting for the mandatory digit following
		// the decimal point. Go may stop at one significant digit; ask for the
		// nearest two-digit member of that same family.
		candidate := strconv.FormatFloat(v, 'e', 1, bits)
		cp := strings.IndexByte(candidate, 'e')
		parsed, err := strconv.ParseFloat(candidate, bits)
		if err == nil && ((bits == 64 && math.Float64bits(parsed) == math.Float64bits(v)) ||
			(bits == 32 && math.Float32bits(float32(parsed)) == math.Float32bits(float32(v)))) {
			e = candidate
			p = cp
			digits = strings.ReplaceAll(e[:p], ".", "")
			exp, _ = strconv.Atoi(e[p+1:])
		}
	}
	decExp := exp + 1
	for len(digits) > 1 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
	}
	var out string
	if decExp > 0 && decExp <= 7 {
		if len(digits) <= decExp {
			out = digits + strings.Repeat("0", decExp-len(digits)) + ".0"
		} else {
			out = digits[:decExp] + "." + digits[decExp:]
		}
	} else if decExp <= 0 && decExp > -3 {
		out = "0." + strings.Repeat("0", -decExp) + digits
	} else {
		out = digits[:1] + "."
		if len(digits) == 1 {
			out += "0"
		} else {
			out += digits[1:]
		}
		out += "E" + strconv.Itoa(decExp-1)
	}
	if neg {
		return "-" + out
	}
	return out
}

// ParseInt implements Integer.parseInt(String).
func ParseInt(s string) (int32, error) {
	if s == "" {
		return 0, nfe(`For input string: ""`)
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, nfe(`For input string: "` + s + `"`)
	}
	return int32(v), nil
}

// ParseIntRadix implements Integer.parseInt(String,int).
func ParseIntRadix(s string, radix int) (int32, error) {
	if radix < 2 {
		return 0, nfe("radix " + strconv.Itoa(radix) + " less than Character.MIN_RADIX")
	}
	if radix > 36 {
		return 0, nfe("radix " + strconv.Itoa(radix) + " greater than Character.MAX_RADIX")
	}
	v, err := strconv.ParseInt(s, radix, 32)
	if err != nil {
		msg := `For input string: "` + s + `"`
		if radix != 10 {
			msg += ` under radix ` + strconv.Itoa(radix)
		}
		return 0, nfe(msg)
	}
	return int32(v), nil
}
func nfe(s string) error { return Throw("java.lang.NumberFormatException", s) }

// ParseDouble implements Double.parseDouble(String).
func ParseDouble(s string) (float64, error) {
	t := Trim(s)
	if t == "" {
		return 0, nfe(`empty String`)
	}
	if t == "NaN" || t == "+NaN" || t == "-NaN" {
		return math.NaN(), nil
	}
	if t == "Infinity" || t == "+Infinity" {
		return math.Inf(1), nil
	}
	if t == "-Infinity" {
		return math.Inf(-1), nil
	}
	display := t
	if hasMultiplePoints(display) {
		return 0, nfe("multiple points")
	}
	if n := len(t); n > 0 && (t[n-1] == 'd' || t[n-1] == 'D' || t[n-1] == 'f' || t[n-1] == 'F') {
		t = t[:n-1]
	}
	if strings.EqualFold(t, "inf") || strings.EqualFold(t, "infinity") {
		return 0, nfe(`For input string: "` + display + `"`)
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, nfe(`For input string: "` + display + `"`)
	}
	return v, nil
}

func hasMultiplePoints(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	hex := i+2 <= len(s) && s[i] == '0' && (s[i+1] == 'x' || s[i+1] == 'X')
	if hex {
		return false
	}
	seen := false
	for ; i < len(s); i++ {
		c := s[i]
		digit := c >= '0' && c <= '9'
		if hex {
			digit = digit || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		}
		if digit {
			continue
		}
		if c == '.' {
			if seen {
				return true
			}
			seen = true
			continue
		}
		break
	}
	return false
}

// NewBigDecimal parses the BigDecimal(String) grammar.
func NewBigDecimal(s string) (BigDecimal, error) {
	if s == "" {
		return BigDecimal{}, nfe("")
	}
	i := 0
	neg := false
	if s[0] == '+' || s[0] == '-' {
		neg = s[0] == '-'
		i++
	}
	if i == len(s) {
		return BigDecimal{}, nfe("Character " + string(s[0]) + " is neither a decimal digit number, decimal point, nor \"e\" notation exponential mark.")
	}
	var digits strings.Builder
	frac := int64(0)
	dot := false
	any := false
	for i < len(s) {
		c := s[i]
		if c >= '0' && c <= '9' {
			digits.WriteByte(c)
			any = true
			if dot {
				frac++
			}
			i++
			continue
		}
		if c == '.' && !dot {
			dot = true
			i++
			continue
		}
		break
	}
	if !any {
		return BigDecimal{}, nfe("Character array is missing \"e\" notation exponential mark.")
	}
	exp := int64(0)
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i == len(s) {
			return BigDecimal{}, nfe("")
		}
		start := i
		if s[i] == '+' || s[i] == '-' {
			i++
		}
		if i == len(s) {
			return BigDecimal{}, nfe("")
		}
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == start || ((s[start] == '+' || s[start] == '-') && i == start+1) {
			return BigDecimal{}, nfe("")
		}
		x, e := strconv.ParseInt(s[start:i], 10, 64)
		if e != nil {
			return BigDecimal{}, nfe("Too many nonzero exponent digits.")
		}
		exp = x
	}
	if i != len(s) {
		return BigDecimal{}, nfe("Character " + string(s[i]) + " is neither a decimal digit number, decimal point, nor \"e\" notation exponential mark.")
	}
	scale := frac - exp
	if scale < math.MinInt32 || scale > math.MaxInt32 {
		return BigDecimal{}, nfe("Scale out of range.")
	}
	z := new(big.Int)
	z.SetString(digits.String(), 10)
	if neg {
		z.Neg(z)
	}
	return BigDecimal{Unscaled: z, Scale: int32(scale)}, nil
}

// String returns BigDecimal.toString().
func (d BigDecimal) String() string {
	if d.Unscaled == nil {
		return "0"
	}
	neg := d.Unscaled.Sign() < 0
	a := new(big.Int).Abs(d.Unscaled).String()
	precision := len(a)
	adjusted := int64(precision) - int64(d.Scale) - 1
	var out string
	if d.Scale >= 0 && adjusted >= -6 {
		point := precision - int(d.Scale)
		if point > 0 {
			if d.Scale == 0 {
				out = a
			} else {
				out = a[:point] + "." + a[point:]
			}
		} else {
			out = "0." + strings.Repeat("0", -point) + a
		}
	} else {
		out = a[:1]
		if precision > 1 {
			out += "." + a[1:]
		}
		if adjusted != 0 {
			out += "E"
			if adjusted > 0 {
				out += "+"
			}
			out += strconv.FormatInt(adjusted, 10)
		}
	}
	if neg {
		return "-" + out
	}
	return out
}

// ToPlainString returns BigDecimal.toPlainString().
func (d BigDecimal) ToPlainString() string {
	if d.Unscaled == nil {
		return "0"
	}
	if d.Unscaled.Sign() == 0 && d.Scale < 0 {
		return "0"
	}
	neg := d.Unscaled.Sign() < 0
	a := new(big.Int).Abs(d.Unscaled).String()
	var out string
	if d.Scale <= 0 {
		out = a + strings.Repeat("0", int(-int64(d.Scale)))
	} else {
		p := len(a) - int(d.Scale)
		if p > 0 {
			out = a[:p] + "." + a[p:]
		} else {
			out = "0." + strings.Repeat("0", -p) + a
		}
	}
	if neg {
		return "-" + out
	}
	return out
}

// StripTrailingZeros returns BigDecimal.stripTrailingZeros().
func (d BigDecimal) StripTrailingZeros() BigDecimal {
	if d.Unscaled == nil || d.Unscaled.Sign() == 0 {
		return BigDecimal{Unscaled: new(big.Int)}
	}
	z := new(big.Int).Set(d.Unscaled)
	scale := d.Scale
	ten := big.NewInt(10)
	q, r := new(big.Int), new(big.Int)
	for scale > math.MinInt32 {
		q.QuoRem(z, ten, r)
		if r.Sign() != 0 {
			break
		}
		z.Set(q)
		scale--
	}
	return BigDecimal{Unscaled: z, Scale: scale}
}

// ToEngineeringString returns BigDecimal.toEngineeringString().
func (d BigDecimal) ToEngineeringString() string {
	if d.Unscaled == nil {
		return "0"
	}
	a := new(big.Int).Abs(d.Unscaled).String()
	adjusted := int64(len(a)) - int64(d.Scale) - 1
	if d.Scale >= 0 && adjusted >= -6 {
		return d.String()
	}
	eng := adjusted % 3
	if eng < 0 {
		eng += 3
	}
	engExp := adjusted - eng
	before := int(eng + 1)
	digits := a
	if len(digits) < before {
		digits += strings.Repeat("0", before-len(digits))
	}
	out := digits[:before]
	if len(digits) > before {
		out += "." + digits[before:]
	}
	if engExp != 0 {
		out += "E"
		if engExp > 0 {
			out += "+"
		}
		out += strconv.FormatInt(engExp, 10)
	}
	if d.Unscaled.Sign() < 0 {
		return "-" + out
	}
	return out
}

func (d BigDecimal) integerValue() *big.Int {
	if d.Unscaled == nil {
		return new(big.Int)
	}
	z := new(big.Int).Set(d.Unscaled)
	if d.Scale > 0 {
		z.Quo(z, pow10(int64(d.Scale)))
	} else if d.Scale < 0 {
		z.Mul(z, pow10(-int64(d.Scale)))
	}
	return z
}
func (d BigDecimal) doubleValue() float64 {
	if d.Unscaled == nil {
		return 0
	}
	r := new(big.Rat).SetInt(d.Unscaled)
	if d.Scale > 0 {
		r.Quo(r, new(big.Rat).SetInt(pow10(int64(d.Scale))))
	} else if d.Scale < 0 {
		r.Mul(r, new(big.Rat).SetInt(pow10(-int64(d.Scale))))
	}
	f, _ := r.Float64()
	return f
}
func pow10(n int64) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(n), nil) }
func compareBigDecimal(a, b BigDecimal) int {
	if a.Scale == b.Scale {
		return a.Unscaled.Cmp(b.Unscaled)
	}
	s := a.Scale
	if b.Scale > s {
		s = b.Scale
	}
	x := new(big.Int).Set(a.Unscaled)
	y := new(big.Int).Set(b.Unscaled)
	if s > a.Scale {
		x.Mul(x, pow10(int64(s-a.Scale)))
	}
	if s > b.Scale {
		y.Mul(y, pow10(int64(s-b.Scale)))
	}
	return x.Cmp(y)
}
