// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

import (
	"math"
	"testing"
)

func TestFloatingStrings(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{{1, "1.0"}, {1e8, "1.0E8"}, {1e-4, "1.0E-4"}, {math.Copysign(0, -1), "-0.0"}, {math.Inf(1), "Infinity"}, {math.NaN(), "NaN"}, {1e23, "1.0E23"}, {math.SmallestNonzeroFloat64, "4.9E-324"}}
	for _, tc := range cases {
		if got := DoubleToString(tc.v); got != tc.want {
			t.Errorf("DoubleToString(%g)=%q want %q", tc.v, got, tc.want)
		}
	}
	if got := FloatToString(math.SmallestNonzeroFloat32); got != "1.4E-45" {
		t.Errorf("smallest float=%q", got)
	}
}

func TestBigDecimal(t *testing.T) {
	cases := []struct{ in, str, plain string }{{"1E+3", "1E+3", "1000"}, {"1.23E-7", "1.23E-7", "0.000000123"}, {"1.23E-6", "0.00000123", "0.00000123"}, {"-0.00", "0.00", "0.00"}}
	for _, tc := range cases {
		x, e := NewBigDecimal(tc.in)
		if e != nil {
			t.Fatal(e)
		}
		if x.String() != tc.str || x.ToPlainString() != tc.plain {
			t.Errorf("%s => %s/%s", tc.in, x.String(), x.ToPlainString())
		}
	}
}

func TestUTF16CaseAndCodec(t *testing.T) {
	surrogate := string([]byte{0xed, 0xa0, 0x80})
	s := "A😀" + surrogate
	if UTF16Len(s) != 4 {
		t.Fatalf("length=%d", UTF16Len(s))
	}
	c, e := CharAt(s, 3)
	if e != nil || c != 0xd800 {
		t.Fatalf("char=%x err=%v", c, e)
	}
	if got := string(EncodeUTF8(s)); got != "A😀?" {
		t.Errorf("encoded=%q", got)
	}
	if got := DecodeUTF8([]byte{0xe2, 0x82, 'A'}); got != "�A" {
		t.Errorf("decoded=%q", got)
	}
	if got := ToUpperCase("ß ŉ ﬁ ǰ", Root); got != "SS ʼN FI J̌" {
		t.Errorf("upper=%q", got)
	}
	if got := ToLowerCase("ΟΔΟΣ ΣΑΣ", Root); got != "οδος σας" {
		t.Errorf("lower=%q", got)
	}
	if got := ToLowerCase("I\u0307", ForLanguageTag("tr")); got != "i" {
		t.Errorf("tr lower=%q", got)
	}
}

func TestHashMapTreeOrder(t *testing.T) {
	m := NewHashMap()
	keys := make([]string, 11)
	for mask := 0; mask < 11; mask++ {
		for bit := 0; bit < 6; bit++ {
			if mask&(1<<bit) == 0 {
				keys[mask] += "Aa"
			} else {
				keys[mask] += "BB"
			}
		}
		m.Put(keys[mask], int32(mask))
	}
	want := []int{1, 0, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := m.Keys()
	for i, index := range want {
		if got[i] != keys[index] {
			t.Fatalf("key %d=%q want %q", i, got[i], keys[index])
		}
	}
}

func TestEntryAndSelfCollectionEquality(t *testing.T) {
	if !Equals(Entry{Key: "x", Value: int32(1)}, Entry{Key: "x", Value: int32(1)}) {
		t.Error("equal Map.Entry values did not compare equal")
	}
	l := &List{}
	l.Elems = []any{l}
	if !Equals(l, l) {
		t.Error("self-referential list did not compare equal to itself")
	}
	us := Locale{Language: "en", Country: "US"}
	if ClassName(us) != "java.util.Locale" || !Equals(us, us) || HashCode(us) != 96636889 {
		t.Errorf("Locale identity/hash: class=%s equal=%v hash=%d", ClassName(us), Equals(us, us), HashCode(us))
	}
}

func TestParseErrors(t *testing.T) {
	if _, e := ParseInt("x"); e == nil || e.Error() != `java.lang.NumberFormatException: For input string: "x"` {
		t.Errorf("parse int: %v", e)
	}
	if _, e := ParseIntRadix("", 16); e == nil || e.Error() != `java.lang.NumberFormatException: For input string: "" under radix 16` {
		t.Errorf("parse radix: %v", e)
	}
	if _, e := ParseDouble(" .. "); e == nil || e.Error() != "java.lang.NumberFormatException: multiple points" {
		t.Errorf("parse double: %v", e)
	}
}

func TestDeterministicMembers(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"ab", "ab"},
		{"  a\n  b", "a\nb"},
		{"  a\n  b\n", "  a\n  b\n"},
		{"   \n  a  \n  b  ", "\na\nb"},
		{"\r\n  a\r  b", "\na\nb"},
	} {
		got, found, err := InvokeMember(tc.in, "stripIndent")
		if !found || err != nil || got != tc.want {
			t.Errorf("stripIndent(%q)=%q,%v,%v want %q", tc.in, got, found, err, tc.want)
		}
	}
	if got, found, err := InvokeMember(math.Inf(1), "isInfinite"); !found || err != nil || got != true {
		t.Errorf("isInfinite=%v,%v,%v", got, found, err)
	}
	if got, found, err := InvokeMember(math.NaN(), "isNaN"); !found || err != nil || got != true {
		t.Errorf("isNaN=%v,%v,%v", got, found, err)
	}
	m := NewLinkedHashMap()
	m.Put("x", int32(1))
	m.Put("y", int32(2))
	if got, found, err := InvokeMember(m, "pollFirstEntry"); !found || err != nil || ToString(got) != "x=1" || ToString(m) != "{y=2}" {
		t.Errorf("pollFirstEntry=%v,%v,%v map=%s", got, found, err, ToString(m))
	}
	if got, found, err := InvokeMember(m, "sequencedKeySet"); !found || err != nil || ToString(got) != "[y]" {
		t.Errorf("sequencedKeySet=%v,%v,%v", got, found, err)
	}
}
