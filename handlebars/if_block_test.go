// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"
)

func TestIfBlockTruthiness(t *testing.T) {
	templates := []struct {
		name, input, truthy, falsy string
	}{
		{"if", "{{#if value}}true{{else}}false{{/if}}", "true", "false"},
		{"section inverse", "{{#value}}true{{^}}false{{/value}}", "true", "false"},
		{"inverted", "{{^value}}false{{/value}}", "", "false"},
	}
	values := []struct {
		name   string
		value  any
		truthy bool
	}{
		{"string", "x", true},
		{"object", object(), true},
		{"true", true, true},
		{"nonempty list", []any{int32(0)}, true},
		{"empty string", "", false},
		{"null", nil, false},
		{"false", false, false},
		{"empty list", []any{}, false},
		{"zero", int32(0), false},
	}
	var cases []renderCase
	for _, value := range values {
		for _, template := range templates {
			expected := template.falsy
			if value.truthy {
				expected = template.truthy
			}
			cases = append(cases, renderCase{
				name:     fmt.Sprintf("%s %s", value.name, template.name),
				template: template.input,
				data:     object("value", value.value),
				expected: expected,
			})
		}
	}
	runRenderCases(t, cases)
}

func TestIfIncludeZero(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"zero default", "{{#if value}}true{{else}}false{{/if}}", object("value", int32(0)), "false", nil, nil, nil},
		{"zero included", "{{#if value includeZero=true}}true{{else}}false{{/if}}", object("value", int32(0)), "true", nil, nil, nil},
	})
}
