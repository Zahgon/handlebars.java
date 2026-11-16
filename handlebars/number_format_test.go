// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"math"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestNumberFormat(t *testing.T) {
	previous := jvm.Default()
	jvm.SetDefault(jvm.ForLanguageTag("en-US"))
	defer jvm.SetDefault(previous)
	register := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	runRenderCases(t, []renderCase{
		{"default", "{{numberFormat this}}", math.Pi, "3.142", nil, nil, register},
		{"currency", "{{numberFormat this \"currency\"}}", math.Pi, "$3.14", nil, nil, register},
		{"percent", "{{numberFormat this \"percent\"}}", math.Pi, "314%", nil, nil, register},
		{"integer", "{{numberFormat this \"integer\"}}", math.Pi, "3", nil, nil, register},
		{"pattern", "{{numberFormat this \"#,###,##0.000\"}}", math.Pi, "3.142", nil, nil, register},
		{"French", "{{numberFormat this \"#,###,##0.000\" \"fr\"}}", math.Pi, "3,142", nil, nil, register},
		{"named format", "{{numberFormat this format=\"percent\"}}", math.Pi, "314%", nil, nil, register},
		{"grouping disabled", "{{numberFormat this groupingUsed=false}}", float64(12345.25), "12345.25", nil, nil, register},
		{"fraction bounds", "{{numberFormat this minimumFractionDigits=2 maximumFractionDigits=2}}", float64(2), "2.00", nil, nil, register},
	})
}
