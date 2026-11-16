// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

// Package jvm implements the Java value model and the java.lang and java.util
// behavior used by handlebars.java.
package jvm

import (
	"math/big"
)

// Char is one UTF-16 code unit, corresponding to java.lang.Character.
type Char uint16

// BigDecimal is a Java BigDecimal represented by an unscaled integer and scale.
type BigDecimal struct {
	// Unscaled is the arbitrary-precision integer coefficient.
	Unscaled *big.Int
	// Scale is the number of digits to the right of the decimal point.
	Scale int32
}

// Object is implemented by modeled Java objects owned by other packages.
type Object interface {
	JavaClass() string
	JavaString() string
}

// Hasher supplies Java's hashCode operation.
type Hasher interface{ JavaHashCode() int32 }

// Equaler supplies Java's equals operation.
type Equaler interface{ JavaEquals(any) bool }

// Comparer supplies Java's Comparable.compareTo operation.
type Comparer interface{ JavaCompareTo(any) (int, error) }

// Invoker supplies a public, no-argument Java instance method.
type Invoker interface {
	JavaInvoke(string) (any, bool, error)
}

// Exception models a Java throwable used by the port.
type Exception struct {
	// Class is the fully qualified Java throwable class name.
	Class string
	// Message is the Java detail message when HasMessage is true.
	Message string
	// HasMessage distinguishes an empty message from a null message.
	HasMessage bool
	// Cause is the modeled throwable cause, when present.
	Cause *Exception
}

// Error returns Throwable.toString().
func (e *Exception) Error() string {
	if e == nil {
		return "<nil>"
	}
	if !e.HasMessage {
		return e.Class
	}
	return e.Class + ": " + e.Message
}

// Throw creates a Java exception with a non-null message.
func Throw(class, msg string) *Exception {
	return &Exception{Class: class, Message: msg, HasMessage: true}
}

// ThrowNoMessage creates a Java exception with a null message.
func ThrowNoMessage(class string) *Exception { return &Exception{Class: class} }

// Entry is one java.util.Map.Entry value.
type Entry struct {
	// Key is the entry key.
	Key any
	// Value is the entry value.
	Value any
}

// MapKind identifies the iteration semantics of a modeled map.
type MapKind uint8

const (
	// LinkedHashMap preserves insertion order.
	LinkedHashMap MapKind = iota
	// HashMap follows java.util.HashMap table order.
	HashMap
)

// Map models LinkedHashMap and HashMap.
type Map struct {
	// Kind selects LinkedHashMap or HashMap behavior.
	Kind    MapKind
	entries []mapEntry
	table   [][]int
	tree    []bool
	roots   []*rbNode
	size    int
	limit   int
}

type mapEntry struct {
	key, value any
	hash       int32
	alive      bool
}

// List models java.util.ArrayList.
type List struct {
	// Elems holds the list elements in encounter order.
	Elems []any
}
