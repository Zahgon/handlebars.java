// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"math"
	"math/big"
)

type opaque struct{ class, text string }

func (o opaque) JavaClass() string  { return o.class }
func (o opaque) JavaString() string { return o.text }

type optionalValue struct{ value any }

func (o optionalValue) JavaClass() string   { return "java.util.Optional" }
func (o optionalValue) JavaString() string  { return "Optional[" + ToString(o.value) + "]" }
func (o optionalValue) JavaHashCode() int32 { return HashCode(o.value) }
func (o optionalValue) JavaEquals(v any) bool {
	p, ok := v.(optionalValue)
	return ok && Equals(o.value, p.value)
}

// InvokeMember invokes a public no-argument instance method by exact name.
func InvokeMember(v any, method string) (any, bool, error) {
	if v == nil {
		return nil, false, nil
	}
	if x, ok := v.(Invoker); ok {
		return x.JavaInvoke(method)
	}
	common := func() (any, bool, error) {
		switch method {
		case "toString":
			return ToString(v), true, nil
		case "hashCode":
			return HashCode(v), true, nil
		}
		return nil, false, nil
	}
	switch x := v.(type) {
	case string:
		switch method {
		case "length":
			return int32(UTF16Len(x)), true, nil
		case "isEmpty":
			return len(x) == 0, true, nil
		case "isBlank":
			return IsBlank(x), true, nil
		case "toUpperCase":
			return ToUpperCase(x, Default()), true, nil
		case "toLowerCase":
			return ToLowerCase(x, Default()), true, nil
		case "trim":
			return Trim(x), true, nil
		case "strip":
			return Strip(x), true, nil
		case "stripLeading":
			return stripLeading(x), true, nil
		case "stripTrailing":
			return stripTrailing(x), true, nil
		case "stripIndent":
			return stripIndent(x), true, nil
		case "intern":
			return x, true, nil
		case "describeConstable":
			return optionalValue{x}, true, nil
		case "getBytes":
			return opaque{"[B", "[B@0"}, true, nil
		case "toCharArray":
			return opaque{"[C", "[C@0"}, true, nil
		case "translateEscapes":
			return translateEscapes(x)
		case "chars":
			return opaque{"java.util.stream.IntStream", "java.util.stream.IntPipeline$Head"}, true, nil
		case "codePoints":
			return opaque{"java.util.stream.IntStream", "java.lang.StringUTF16$CodePointsSpliterator"}, true, nil
		case "lines":
			return opaque{"java.util.stream.Stream", "java.util.stream.ReferencePipeline$Head"}, true, nil
		}
		return common()
	case bool:
		switch method {
		case "booleanValue":
			return x, true, nil
		case "describeConstable":
			return optionalValue{x}, true, nil
		}
		return common()
	case Char:
		switch method {
		case "charValue":
			return x, true, nil
		case "describeConstable":
			return optionalValue{x}, true, nil
		}
		return common()
	case int8, int16, int32, int64, float32, float64:
		switch method {
		case "byteValue":
			return int8(IntValue(x)), true, nil
		case "shortValue":
			return int16(IntValue(x)), true, nil
		case "intValue":
			return IntValue(x), true, nil
		case "longValue":
			return LongValue(x), true, nil
		case "floatValue":
			return float32(DoubleValue(x)), true, nil
		case "doubleValue":
			return DoubleValue(x), true, nil
		case "describeConstable":
			return optionalValue{x}, true, nil
		case "isNaN":
			switch n := x.(type) {
			case float32:
				return math.IsNaN(float64(n)), true, nil
			case float64:
				return math.IsNaN(n), true, nil
			}
		case "isInfinite":
			switch n := x.(type) {
			case float32:
				return math.IsInf(float64(n), 0), true, nil
			case float64:
				return math.IsInf(n, 0), true, nil
			}
		}
		return common()
	case *big.Int:
		switch method {
		case "signum":
			return int32(x.Sign()), true, nil
		case "abs":
			return new(big.Int).Abs(x), true, nil
		case "negate":
			return new(big.Int).Neg(x), true, nil
		case "bitLength":
			return int32(x.BitLen()), true, nil
		case "bitCount":
			return int32(bigIntegerBitCount(x)), true, nil
		case "getLowestSetBit":
			return int32(bigIntegerLowestSetBit(x)), true, nil
		case "not":
			return new(big.Int).Not(x), true, nil
		case "nextProbablePrime":
			if x.Sign() < 0 {
				return nil, true, Throw("java.lang.ArithmeticException", "start < 0: "+x.String())
			}
			return nextProbablePrime(x), true, nil
		case "sqrt":
			if x.Sign() < 0 {
				return nil, true, Throw("java.lang.ArithmeticException", "Negative BigInteger")
			}
			return new(big.Int).Sqrt(x), true, nil
		case "sqrtAndRemainder":
			if x.Sign() < 0 {
				return nil, true, Throw("java.lang.ArithmeticException", "Negative BigInteger")
			}
			return opaque{"[Ljava.math.BigInteger;", "[Ljava.math.BigInteger;@0"}, true, nil
		case "toByteArray":
			return opaque{"[B", "[B@0"}, true, nil
		case "byteValueExact":
			return exactBigInteger(x, 8, "byte")
		case "shortValueExact":
			return exactBigInteger(x, 16, "short")
		case "intValueExact":
			return exactBigInteger(x, 32, "int")
		case "longValueExact":
			return exactBigInteger(x, 64, "long")
		case "intValue":
			return IntValue(x), true, nil
		case "longValue":
			return LongValue(x), true, nil
		case "floatValue":
			return float32(DoubleValue(x)), true, nil
		case "doubleValue":
			return DoubleValue(x), true, nil
		}
		return common()
	case BigDecimal:
		switch method {
		case "signum":
			return int32(x.Unscaled.Sign()), true, nil
		case "scale":
			return x.Scale, true, nil
		case "precision":
			return int32(len(new(big.Int).Abs(x.Unscaled).String())), true, nil
		case "unscaledValue":
			return new(big.Int).Set(x.Unscaled), true, nil
		case "toPlainString":
			return x.ToPlainString(), true, nil
		case "toEngineeringString":
			return x.ToEngineeringString(), true, nil
		case "stripTrailingZeros":
			return x.StripTrailingZeros(), true, nil
		case "ulp":
			return BigDecimal{Unscaled: big.NewInt(1), Scale: x.Scale}, true, nil
		case "plus":
			return x, true, nil
		case "toBigInteger":
			return x.integerValue(), true, nil
		case "toBigIntegerExact":
			return exactBigDecimalInteger(x, 0, "BigInteger")
		case "byteValueExact":
			return exactBigDecimalInteger(x, 8, "byte")
		case "shortValueExact":
			return exactBigDecimalInteger(x, 16, "short")
		case "intValueExact":
			return exactBigDecimalInteger(x, 32, "int")
		case "longValueExact":
			return exactBigDecimalInteger(x, 64, "long")
		case "abs":
			if x.Unscaled.Sign() < 0 {
				x.Unscaled = new(big.Int).Neg(x.Unscaled)
			}
			return x, true, nil
		case "negate":
			x.Unscaled = new(big.Int).Neg(x.Unscaled)
			return x, true, nil
		case "intValue":
			return IntValue(x), true, nil
		case "longValue":
			return LongValue(x), true, nil
		case "floatValue":
			return float32(DoubleValue(x)), true, nil
		case "doubleValue":
			return DoubleValue(x), true, nil
		}
		return common()
	case *List:
		switch method {
		case "size":
			return int32(x.Len()), true, nil
		case "isEmpty":
			return x.Len() == 0, true, nil
		case "getFirst":
			if x.Len() == 0 {
				return nil, true, ThrowNoMessage("java.util.NoSuchElementException")
			}
			return x.Elems[0], true, nil
		case "getLast":
			if x.Len() == 0 {
				return nil, true, ThrowNoMessage("java.util.NoSuchElementException")
			}
			return x.Elems[len(x.Elems)-1], true, nil
		case "reversed":
			e := make([]any, len(x.Elems))
			for i := range x.Elems {
				e[len(e)-1-i] = x.Elems[i]
			}
			return &List{Elems: e}, true, nil
		case "clone":
			return &List{Elems: append([]any(nil), x.Elems...)}, true, nil
		case "toArray":
			return opaque{"[Ljava.lang.Object;", "[Ljava.lang.Object;@0"}, true, nil
		case "iterator", "listIterator", "spliterator":
			return opaque{"java.util.Iterator", "java.util.ArrayList$Itr@0"}, true, nil
		case "removeFirst":
			if x.Len() == 0 {
				return nil, true, ThrowNoMessage("java.util.NoSuchElementException")
			}
			v := x.Elems[0]
			x.Elems = x.Elems[1:]
			return v, true, nil
		case "removeLast":
			if x.Len() == 0 {
				return nil, true, ThrowNoMessage("java.util.NoSuchElementException")
			}
			i := len(x.Elems) - 1
			v := x.Elems[i]
			x.Elems = x.Elems[:i]
			return v, true, nil
		case "clear":
			x.Elems = nil
			return nil, true, nil
		case "trimToSize":
			return nil, true, nil
		}
		return common()
	case *Map:
		switch method {
		case "size":
			return int32(x.Len()), true, nil
		case "isEmpty":
			return x.Len() == 0, true, nil
		case "keySet":
			return &List{Elems: x.Keys()}, true, nil
		case "values":
			es := x.Entries()
			a := make([]any, len(es))
			for i := range es {
				a[i] = es[i].Value
			}
			return &List{Elems: a}, true, nil
		case "entrySet":
			es := x.Entries()
			a := make([]any, len(es))
			for i := range es {
				a[i] = es[i]
			}
			return &List{Elems: a}, true, nil
		case "sequencedKeySet":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			return &List{Elems: x.Keys()}, true, nil
		case "sequencedValues":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			a := make([]any, len(es))
			for i := range es {
				a[i] = es[i].Value
			}
			return &List{Elems: a}, true, nil
		case "sequencedEntrySet":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			a := make([]any, len(es))
			for i := range es {
				a[i] = es[i]
			}
			return &List{Elems: a}, true, nil
		case "firstEntry":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			if len(es) == 0 {
				return nil, true, nil
			}
			return es[0], true, nil
		case "lastEntry":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			if len(es) == 0 {
				return nil, true, nil
			}
			return es[len(es)-1], true, nil
		case "pollFirstEntry":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			if len(es) == 0 {
				return nil, true, nil
			}
			x.Remove(es[0].Key)
			return es[0], true, nil
		case "pollLastEntry":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			es := x.Entries()
			if len(es) == 0 {
				return nil, true, nil
			}
			x.Remove(es[len(es)-1].Key)
			return es[len(es)-1], true, nil
		case "reversed":
			if x.Kind == HashMap {
				return nil, false, nil
			}
			r := NewLinkedHashMap()
			es := x.Entries()
			for i := len(es) - 1; i >= 0; i-- {
				r.Put(es[i].Key, es[i].Value)
			}
			return r, true, nil
		case "clone":
			r := &Map{Kind: x.Kind}
			for _, e := range x.Entries() {
				r.Put(e.Key, e.Value)
			}
			return r, true, nil
		case "clear":
			for i := range x.entries {
				x.entries[i].alive = false
			}
			x.size = 0
			for i := range x.table {
				x.table[i] = nil
				x.tree[i] = false
				x.roots[i] = nil
			}
			return nil, true, nil
		}
		return common()
	case Entry:
		switch method {
		case "getKey":
			return x.Key, true, nil
		case "getValue":
			return x.Value, true, nil
		}
		return common()
	default:
		return common()
	}
}

func translateEscapes(s string) (any, bool, error) {
	var b []byte
	u := toUnits(s)
	for i := 0; i < len(u); i++ {
		c := u[i]
		if c != '\\' {
			b = append(b, []byte(fromUnits([]uint16{c}))...)
			continue
		}
		i++
		if i == len(u) {
			return nil, true, Throw("java.lang.IllegalArgumentException", "Invalid escape sequence: \\ at end of string")
		}
		switch u[i] {
		case 'b':
			b = append(b, '\b')
		case 't':
			b = append(b, '\t')
		case 'n':
			b = append(b, '\n')
		case 'f':
			b = append(b, '\f')
		case 'r':
			b = append(b, '\r')
		case 's':
			b = append(b, ' ')
		case '\'':
			b = append(b, '\'')
		case '"':
			b = append(b, '"')
		case '\\':
			b = append(b, '\\')
		case '\n':
		case '\r':
			if i+1 < len(u) && u[i+1] == '\n' {
				i++
			}
		default:
			if u[i] >= '0' && u[i] <= '7' {
				v := int(u[i] - '0')
				max := 2
				if u[i] <= '3' {
					max = 3
				}
				for n := 1; n < max && i+1 < len(u) && u[i+1] >= '0' && u[i+1] <= '7'; n++ {
					i++
					v = v*8 + int(u[i]-'0')
				}
				b = append(b, byte(v))
			} else {
				return nil, true, Throw("java.lang.IllegalArgumentException", "Invalid escape sequence: \\"+fromUnits([]uint16{u[i]}))
			}
		}
	}
	return string(b), true, nil
}

func bigIntegerBitCount(x *big.Int) int {
	if x.Sign() >= 0 {
		return popcountBig(x)
	}
	z := new(big.Int).Not(x)
	return popcountBig(z)
}
func popcountBig(x *big.Int) int {
	n := 0
	for _, w := range x.Bits() {
		v := uint64(w)
		for v != 0 {
			v &= v - 1
			n++
		}
	}
	return n
}
func bigIntegerLowestSetBit(x *big.Int) int {
	if x.Sign() == 0 {
		return -1
	}
	z := new(big.Int).Abs(x)
	return int(z.TrailingZeroBits())
}
func nextProbablePrime(x *big.Int) *big.Int {
	z := new(big.Int).Add(x, big.NewInt(1))
	if z.Cmp(big.NewInt(2)) <= 0 {
		return big.NewInt(2)
	}
	if z.Bit(0) == 0 {
		z.Add(z, big.NewInt(1))
	}
	for !z.ProbablyPrime(100) {
		z.Add(z, big.NewInt(2))
	}
	return z
}

func exactBigInteger(x *big.Int, bits int, name string) (any, bool, error) {
	if bits < 64 && !x.IsInt64() || bits == 64 && !x.IsInt64() {
		return nil, true, Throw("java.lang.ArithmeticException", "BigInteger out of "+name+" range")
	}
	v := x.Int64()
	if bits < 64 {
		min := -(int64(1) << (bits - 1))
		max := (int64(1) << (bits - 1)) - 1
		if v < min || v > max {
			return nil, true, Throw("java.lang.ArithmeticException", "BigInteger out of "+name+" range")
		}
	}
	switch bits {
	case 8:
		return int8(v), true, nil
	case 16:
		return int16(v), true, nil
	case 32:
		return int32(v), true, nil
	default:
		return v, true, nil
	}
}

func exactBigDecimalInteger(x BigDecimal, bits int, _ string) (any, bool, error) {
	z := new(big.Int)
	if x.Unscaled != nil {
		z.Set(x.Unscaled)
	}
	if x.Scale > 0 {
		q, r := new(big.Int), new(big.Int)
		q.QuoRem(z, pow10(int64(x.Scale)), r)
		if r.Sign() != 0 {
			return nil, true, Throw("java.lang.ArithmeticException", "Rounding necessary")
		}
		z = q
	} else if x.Scale < 0 {
		z.Mul(z, pow10(-int64(x.Scale)))
	}
	if bits == 0 {
		return z, true, nil
	}
	if !z.IsInt64() {
		return nil, true, Throw("java.lang.ArithmeticException", "Overflow")
	}
	v := z.Int64()
	if bits < 64 {
		min := -(int64(1) << (bits - 1))
		max := (int64(1) << (bits - 1)) - 1
		if v < min || v > max {
			return nil, true, Throw("java.lang.ArithmeticException", "Overflow")
		}
	}
	switch bits {
	case 8:
		return int8(v), true, nil
	case 16:
		return int16(v), true, nil
	case 32:
		return int32(v), true, nil
	default:
		return v, true, nil
	}
}

// GetProperty invokes a JavaBean getX/isX property, including collection length.
func GetProperty(v any, name string) (any, bool, error) {
	if v == nil || name == "" {
		return nil, false, nil
	}
	if name == "length" {
		switch x := v.(type) {
		case *List:
			return int32(x.Len()), true, nil
		}
	}
	u := toUnits(name)
	u[0] = uint16(CharacterToUpperCase(rune(u[0])))
	tail := fromUnits(u)
	if r, f, e := InvokeMember(v, "get"+tail); f {
		return r, true, e
	}
	if r, f, e := InvokeMember(v, "is"+tail); f {
		return r, true, e
	}
	return nil, false, nil
}

// PropertySet returns the deterministic public no-argument method view.
func PropertySet(v any) ([]Entry, bool) {
	if v == nil {
		return nil, false
	}
	names := memberNames(v)
	out := make([]Entry, 0, len(names))
	for _, n := range names {
		if r, f, e := InvokeMember(v, n); f && e == nil {
			out = append(out, Entry{n, r})
		}
	}
	return out, true
}
func memberNames(v any) []string {
	switch v.(type) {
	case string:
		return []string{"length", "isEmpty", "isBlank", "toUpperCase", "toLowerCase", "trim", "strip", "stripLeading", "stripTrailing", "hashCode", "toString"}
	case *List:
		return []string{"size", "isEmpty", "getFirst", "getLast", "reversed", "hashCode", "toString"}
	case *Map:
		return []string{"size", "isEmpty", "keySet", "values", "entrySet", "firstEntry", "lastEntry", "reversed", "hashCode", "toString"}
	case *big.Int:
		return []string{"signum", "abs", "negate", "bitLength", "intValue", "longValue", "floatValue", "doubleValue", "hashCode", "toString"}
	case BigDecimal:
		return []string{"signum", "scale", "precision", "unscaledValue", "toPlainString", "abs", "negate", "intValue", "longValue", "floatValue", "doubleValue", "hashCode", "toString"}
	default:
		return []string{"hashCode", "toString"}
	}
}
func stripLeading(s string) string {
	u := toUnits(s)
	i := 0
	for i < len(u) && IsWhitespace(rune(u[i])) {
		i++
	}
	return fromUnits(u[i:])
}
func stripTrailing(s string) string {
	u := toUnits(s)
	i := len(u)
	for i > 0 && IsWhitespace(rune(u[i-1])) {
		i--
	}
	return fromUnits(u[:i])
}

// stripIndent implements String.stripIndent, including String.lines line
// terminators and the trailing-line opt-out rule.
func stripIndent(s string) string {
	u := toUnits(s)
	if len(u) == 0 {
		return s
	}
	optOut := u[len(u)-1] == '\n' || u[len(u)-1] == '\r'
	lines := make([][]uint16, 0, 8)
	for start, i := 0, 0; i < len(u); {
		if u[i] != '\n' && u[i] != '\r' {
			i++
			continue
		}
		lines = append(lines, u[start:i])
		if u[i] == '\r' && i+1 < len(u) && u[i+1] == '\n' {
			i++
		}
		i++
		start = i
		if start == len(u) {
			break
		}
	}
	lastTerm := len(u) > 0 && (u[len(u)-1] == '\n' || u[len(u)-1] == '\r')
	if !lastTerm {
		start := len(u)
		for start > 0 && u[start-1] != '\n' && u[start-1] != '\r' {
			start--
		}
		lines = append(lines, u[start:])
	}
	outdent := 0
	if !optOut {
		outdent = int(^uint(0) >> 1)
		for _, line := range lines {
			i := 0
			for i < len(line) && IsWhitespace(rune(line[i])) {
				i++
			}
			if i < len(line) && i < outdent {
				outdent = i
			}
		}
		if outdent == int(^uint(0)>>1) {
			outdent = 0
		}
	}
	var out []uint16
	for n, line := range lines {
		first := 0
		for first < len(line) && IsWhitespace(rune(line[first])) {
			first++
		}
		last := len(line)
		for last > first && IsWhitespace(rune(line[last-1])) {
			last--
		}
		if n > 0 {
			out = append(out, '\n')
		}
		if first < last {
			cut := outdent
			if cut > first {
				cut = first
			}
			out = append(out, line[cut:last]...)
		}
	}
	if optOut {
		out = append(out, '\n')
	}
	return fromUnits(out)
}

// Len returns the list size.
func (l *List) Len() int {
	if l == nil {
		return 0
	}
	return len(l.Elems)
}
