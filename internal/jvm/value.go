// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// ClassName returns getClass().getName() for a modeled value.
func ClassName(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return "java.lang.Boolean"
	case int8:
		return "java.lang.Byte"
	case int16:
		return "java.lang.Short"
	case int32:
		return "java.lang.Integer"
	case int64:
		return "java.lang.Long"
	case float32:
		return "java.lang.Float"
	case float64:
		return "java.lang.Double"
	case Char:
		return "java.lang.Character"
	case string:
		return "java.lang.String"
	case *big.Int:
		return "java.math.BigInteger"
	case BigDecimal:
		return "java.math.BigDecimal"
	case Locale:
		return "java.util.Locale"
	case *Map:
		if x != nil && x.Kind == HashMap {
			return "java.util.HashMap"
		}
		return "java.util.LinkedHashMap"
	case *List:
		return "java.util.ArrayList"
	case Object:
		return x.JavaClass()
	default:
		return fmt.Sprintf("%T", v)
	}
}

// SimpleName returns Class.getSimpleName() for a modeled value.
func SimpleName(v any) string {
	n := ClassName(v)
	if i := strings.LastIndexByte(n, '.'); i >= 0 {
		return n[i+1:]
	}
	return n
}

// ToString implements String.valueOf(Object) for modeled values.
func ToString(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return FloatToString(x)
	case float64:
		return DoubleToString(x)
	case Char:
		return fromUnits([]uint16{uint16(x)})
	case *big.Int:
		if x == nil {
			return "null"
		}
		return x.String()
	case BigDecimal:
		return x.String()
	case *Map:
		return mapString(x)
	case *List:
		return listString(x)
	case Entry:
		return ToString(x.Key) + "=" + ToString(x.Value)
	case Object:
		return x.JavaString()
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}

func mapString(m *Map) string {
	if m == nil {
		return "null"
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, e := range m.Entries() {
		if i > 0 {
			b.WriteString(", ")
		}
		if e.Key == m {
			b.WriteString("(this Map)")
		} else {
			b.WriteString(ToString(e.Key))
		}
		b.WriteByte('=')
		if e.Value == m {
			b.WriteString("(this Map)")
		} else {
			b.WriteString(ToString(e.Value))
		}
	}
	b.WriteByte('}')
	return b.String()
}
func listString(l *List) string {
	if l == nil {
		return "null"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, v := range l.Elems {
		if i > 0 {
			b.WriteString(", ")
		}
		if v == l {
			b.WriteString("(this Collection)")
		} else {
			b.WriteString(ToString(v))
		}
	}
	b.WriteByte(']')
	return b.String()
}

// Equals implements equals for modeled JDK values.
func Equals(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if e, ok := a.(Equaler); ok {
		return e.JavaEquals(b)
	}
	switch x := a.(type) {
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case int8:
		y, ok := b.(int8)
		return ok && x == y
	case int16:
		y, ok := b.(int16)
		return ok && x == y
	case int32:
		y, ok := b.(int32)
		return ok && x == y
	case int64:
		y, ok := b.(int64)
		return ok && x == y
	case float32:
		y, ok := b.(float32)
		return ok && float32BitsCanonical(x) == float32BitsCanonical(y)
	case float64:
		y, ok := b.(float64)
		return ok && float64BitsCanonical(x) == float64BitsCanonical(y)
	case Char:
		y, ok := b.(Char)
		return ok && x == y
	case string:
		y, ok := b.(string)
		return ok && x == y
	case *big.Int:
		y, ok := b.(*big.Int)
		return ok && x != nil && y != nil && x.Cmp(y) == 0
	case BigDecimal:
		y, ok := b.(BigDecimal)
		return ok && x.Scale == y.Scale && x.Unscaled.Cmp(y.Unscaled) == 0
	case Locale:
		y, ok := b.(Locale)
		return ok && x == y
	case Entry:
		y, ok := b.(Entry)
		return ok && Equals(x.Key, y.Key) && Equals(x.Value, y.Value)
	case *List:
		y, ok := b.(*List)
		if !ok || x == nil || y == nil || len(x.Elems) != len(y.Elems) {
			return x == nil && y == nil
		}
		if x == y {
			return true
		}
		for i := range x.Elems {
			if !Equals(x.Elems[i], y.Elems[i]) {
				return false
			}
		}
		return true
	case *Map:
		y, ok := b.(*Map)
		if !ok || x == nil || y == nil || x.Len() != y.Len() {
			return x == nil && y == nil
		}
		if x == y {
			return true
		}
		for _, e := range x.Entries() {
			v, found := y.Get(e.Key)
			if !found || !Equals(e.Value, v) {
				return false
			}
		}
		return true
	}
	return false
}

func float32BitsCanonical(v float32) uint32 {
	if math.IsNaN(float64(v)) {
		return 0x7fc00000
	}
	return math.Float32bits(v)
}
func float64BitsCanonical(v float64) uint64 {
	if math.IsNaN(v) {
		return 0x7ff8000000000000
	}
	return math.Float64bits(v)
}

// HashCode implements hashCode for modeled JDK values.
func HashCode(v any) int32 {
	if v == nil {
		return 0
	}
	if h, ok := v.(Hasher); ok {
		return h.JavaHashCode()
	}
	switch x := v.(type) {
	case bool:
		if x {
			return 1231
		}
		return 1237
	case int8:
		return int32(x)
	case int16:
		return int32(x)
	case int32:
		return x
	case int64:
		return int32(uint64(x) ^ uint64(x)>>32)
	case float32:
		return int32(float32BitsCanonical(x))
	case float64:
		bits := float64BitsCanonical(x)
		return int32(bits ^ bits>>32)
	case Char:
		return int32(x)
	case string:
		return StringHashCode(x)
	case *big.Int:
		return bigIntegerHash(x)
	case BigDecimal:
		return 31*bigIntegerHash(x.Unscaled) + x.Scale
	case Locale:
		h := StringHashCode(x.Language)
		h = 31*h + StringHashCode(x.Script)
		h = 31*h + StringHashCode(x.Country)
		return 31*h + StringHashCode(x.Variant)
	case *List:
		h := int32(1)
		if x != nil {
			for _, e := range x.Elems {
				h = 31*h + HashCode(e)
			}
		}
		return h
	case *Map:
		h := int32(0)
		if x != nil {
			for _, e := range x.Entries() {
				h += HashCode(e.Key) ^ HashCode(e.Value)
			}
		}
		return h
	case Entry:
		return HashCode(x.Key) ^ HashCode(x.Value)
	default:
		return 0
	}
}

func bigIntegerHash(x *big.Int) int32 {
	if x == nil {
		return 0
	}
	a := new(big.Int).Abs(x)
	words := a.Bits()
	h := int32(0)
	for i := len(words) - 1; i >= 0; i-- {
		w := uint64(words[i])
		if strconv.IntSize == 64 {
			h = 31*h + int32(w>>32)
			h = 31*h + int32(w)
		} else {
			h = 31*h + int32(w)
		}
	}
	if x.Sign() < 0 {
		return -h
	}
	return h
}

// CompareTo implements Comparable.compareTo for modeled JDK values.
func CompareTo(a, b any) (int, error) {
	if c, ok := a.(Comparer); ok {
		return c.JavaCompareTo(b)
	}
	if a == nil {
		return 0, ThrowNoMessage("java.lang.NullPointerException")
	}
	bad := func(expected string) (int, error) { return 0, classCast(b, expected) }
	switch x := a.(type) {
	case string:
		y, ok := b.(string)
		if !ok {
			return bad("java.lang.String")
		}
		return CompareStrings(x, y), nil
	case bool:
		y, ok := b.(bool)
		if !ok {
			return bad("java.lang.Boolean")
		}
		if x == y {
			return 0, nil
		}
		if x {
			return 1, nil
		}
		return -1, nil
	case int8:
		y, ok := b.(int8)
		if !ok {
			return bad("java.lang.Byte")
		}
		return cmp64(int64(x), int64(y)), nil
	case int16:
		y, ok := b.(int16)
		if !ok {
			return bad("java.lang.Short")
		}
		return cmp64(int64(x), int64(y)), nil
	case int32:
		y, ok := b.(int32)
		if !ok {
			return bad("java.lang.Integer")
		}
		return cmp64(int64(x), int64(y)), nil
	case int64:
		y, ok := b.(int64)
		if !ok {
			return bad("java.lang.Long")
		}
		return cmp64(x, y), nil
	case float32:
		y, ok := b.(float32)
		if !ok {
			return bad("java.lang.Float")
		}
		return compareFloat32(x, y), nil
	case float64:
		y, ok := b.(float64)
		if !ok {
			return bad("java.lang.Double")
		}
		return compareFloat64(x, y, float64BitsCanonical(x), float64BitsCanonical(y)), nil
	case Char:
		y, ok := b.(Char)
		if !ok {
			return bad("java.lang.Character")
		}
		return cmp64(int64(x), int64(y)), nil
	case *big.Int:
		y, ok := b.(*big.Int)
		if !ok {
			return bad("java.math.BigInteger")
		}
		return x.Cmp(y), nil
	case BigDecimal:
		y, ok := b.(BigDecimal)
		if !ok {
			return bad("java.math.BigDecimal")
		}
		return compareBigDecimal(x, y), nil
	default:
		return bad(ClassName(a))
	}
}
func cmp64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
func compareFloat64(a, b float64, ab, bb uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	if ab == bb {
		return 0
	}
	if int64(ab) < int64(bb) {
		return -1
	}
	return 1
}
func compareFloat32(a, b float32) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	ab, bb := int32(float32BitsCanonical(a)), int32(float32BitsCanonical(b))
	if ab == bb {
		return 0
	}
	if ab < bb {
		return -1
	}
	return 1
}
func classCast(v any, target string) error {
	src := ClassName(v)
	return Throw("java.lang.ClassCastException", "class "+src+" cannot be cast to class "+target+" ("+src+" and "+target+" are in module java.base of loader 'bootstrap')")
}

// IsNumber reports whether a value is a modeled java.lang.Number.
func IsNumber(v any) bool {
	switch v.(type) {
	case int8, int16, int32, int64, float32, float64, *big.Int, BigDecimal:
		return true
	}
	return false
}

// DoubleValue returns Number.doubleValue().
func DoubleValue(v any) float64 {
	switch x := v.(type) {
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	case *big.Int:
		f, _ := new(big.Float).SetInt(x).Float64()
		return f
	case BigDecimal:
		return x.doubleValue()
	}
	return 0
}

// IntValue returns Number.intValue().
func IntValue(v any) int32 {
	switch x := v.(type) {
	case int8:
		return int32(x)
	case int16:
		return int32(x)
	case int32:
		return x
	case int64:
		return int32(x)
	case float32:
		return javaFloatToInt(float64(x))
	case float64:
		return javaFloatToInt(x)
	case *big.Int:
		return bigIntIntValue(x)
	case BigDecimal:
		return bigIntIntValue(x.integerValue())
	}
	return 0
}

// LongValue returns Number.longValue().
func LongValue(v any) int64 {
	switch x := v.(type) {
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case float32:
		return javaFloatToLong(float64(x))
	case float64:
		return javaFloatToLong(x)
	case *big.Int:
		return bigIntLongValue(x)
	case BigDecimal:
		return bigIntLongValue(x.integerValue())
	}
	return 0
}

func bigIntIntValue(x *big.Int) int32 {
	if x == nil {
		return 0
	}
	v := uint32(new(big.Int).Abs(x).Uint64())
	if x.Sign() < 0 {
		v = -v
	}
	return int32(v)
}

func bigIntLongValue(x *big.Int) int64 {
	if x == nil {
		return 0
	}
	v := new(big.Int).Abs(x).Uint64()
	if x.Sign() < 0 {
		v = -v
	}
	return int64(v)
}
func javaFloatToInt(x float64) int32 {
	if math.IsNaN(x) {
		return 0
	}
	if x >= math.MaxInt32 {
		return math.MaxInt32
	}
	if x <= math.MinInt32 {
		return math.MinInt32
	}
	return int32(x)
}
func javaFloatToLong(x float64) int64 {
	if math.IsNaN(x) {
		return 0
	}
	if x >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if x <= float64(math.MinInt64) {
		return math.MinInt64
	}
	return int64(x)
}
