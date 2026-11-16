// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestConditionalHelpers(t *testing.T) {
	register := func(engine *handlebars.Handlebars) { engine.RegisterConditionalHelpers() }
	runRenderCases(t, []renderCase{
		{"large longs", "{{eq a b}}/{{neq a b}}", object("a", int64(9007199254740992), "b", int64(9007199254740993)), "false/true", nil, nil, register},
		{"equal mixed numbers", "{{eq a b}}", object("a", int32(2), "b", float64(2)), "true", nil, nil, register},
		{"not equal", "{{neq a b}}", object("a", "2", "b", int32(2)), "true", nil, nil, register},
		{"greater", "{{gt a b}}", object("a", int32(3), "b", int32(2)), "true", nil, nil, register},
		{"greater equal", "{{gte a b}}", object("a", int32(2), "b", int32(2)), "true", nil, nil, register},
		{"less", "{{lt a b}}", object("a", int32(1), "b", int32(2)), "true", nil, nil, register},
		{"less equal", "{{lte a b}}", object("a", int32(2), "b", int32(2)), "true", nil, nil, register},
		{"and", "{{and a b c}}", object("a", true, "b", "x", "c", int32(1)), "true", nil, nil, register},
		{"or", "{{or a b c}}", object("a", false, "b", "", "c", "x"), "true", nil, nil, register},
		{"not", "{{not a}}", object("a", false), "true", nil, nil, register},
		{"section", "{{#eq a b}}same{{else}}different{{/eq}}", object("a", int32(2), "b", int32(2)), "same", nil, nil, register},
		{"custom output", "{{eq a b yes=\"same\" no=\"different\"}}", object("a", int32(1), "b", int32(2)), "different", nil, nil, register},
	})
}

func TestIssue627ConditionalHelpers(t *testing.T) {
	configure := func(engine *handlebars.Handlebars) { engine.RegisterConditionalHelpers() }
	pairs := [][2]string{
		{"{{#eq 1 1}}yes{{/eq}}", "yes"},
		{"{{#eq 1 2}}yes{{else}}no{{/eq}}", "no"},
		{"{{eq 1 1}}", "true"},
		{"{{eq 1 0}}", "false"},
		{"{{eq 1 1 yes='yes' no='no'}}", "yes"},
		{"{{eq 1 0 yes='yes' no='no'}}", "no"},
		{"{{#if (eq 1 1)}}yes{{/if}}", "yes"},
		{"{{#neq 1 1}}yes{{/neq}}", ""},
		{"{{#neq 1 2}}yes{{else}}no{{/neq}}", "yes"},
		{"{{neq 1 1}}", "false"},
		{"{{neq 1 0}}", "true"},
		{"{{neq 1 1 yes='yes' no='no'}}", "no"},
		{"{{neq 1 0 yes='yes' no='no'}}", "yes"},
		{"{{#if (neq 1 2)}}yes{{/if}}", "yes"},
		{"{{#gt 2 1}}yes{{else}}no{{/gt}}", "yes"},
		{"{{#gt 2 3}}yes{{else}}no{{/gt}}", "no"},
		{"{{#gt 2 2}}yes{{else}}no{{/gt}}", "no"},
		{"{{#gte 2 2}}yes{{else}}no{{/gte}}", "yes"},
		{"{{gt 2 1}}", "true"},
		{"{{gt 1 12}}", "false"},
		{"{{gt 2 1 yes='y' no='n'}}", "y"},
		{"{{gt 1 12 yes='y' no='n'}}", "n"},
		{"{{gte 2 1 yes='y' no='n'}}", "y"},
		{"{{gte 2 2 yes='y' no='n'}}", "y"},
		{"{{#if (gte 2 1)}}yes{{/if}}", "yes"},
		{"{{#if (gt 2 2)}}yes{{/if}}", ""},
		{"{{#if (gte 2 2)}}yes{{/if}}", "yes"},
		{"{{#lt 2 1}}yes{{else}}no{{/lt}}", "no"},
		{"{{#lt 2 3}}yes{{else}}no{{/lt}}", "yes"},
		{"{{#lt 2 2}}yes{{else}}no{{/lt}}", "no"},
		{"{{#lte 2 2}}yes{{else}}no{{/lte}}", "yes"},
		{"{{lt 2 1}}", "false"},
		{"{{lt 1 12}}", "true"},
		{"{{lt 1 12 yes='y' no='n'}}", "y"},
		{"{{lt 14 12 yes='y' no='n'}}", "n"},
		{"{{lte 14 14 yes='y' no='n'}}", "y"},
		{"{{lte 14 12 yes='y' no='n'}}", "n"},
		{"{{#if (lte 2 1)}}yes{{/if}}", ""},
		{"{{#if (lt 2 2)}}yes{{/if}}", ""},
		{"{{#if (lte 2 2)}}yes{{/if}}", "yes"},
		{"{{#and 2 1}}yes{{else}}no{{/and}}", "yes"},
		{"{{#and 0 1}}yes{{else}}no{{/and}}", "no"},
		{"{{#and '' ''}}yes{{else}}no{{/and}}", "no"},
		{"{{#and 'a' 'b'}}yes{{else}}no{{/and}}", "yes"},
		{"{{#and 'a' ''}}yes{{else}}no{{/and}}", "no"},
		{"{{#and 2 1 4 5 6}}yes{{else}}no{{/and}}", "yes"},
		{"{{#and 0 1 4 5 6}}yes{{else}}no{{/and}}", "no"},
		{"{{#and 1 0 4 5 6}}yes{{else}}no{{/and}}", "no"},
		{"{{and 2 1}}", "true"},
		{"{{and 2 1 yes='y'}}", "y"},
		{"{{and 0 1 no='n'}}", "n"},
		{"{{#if (and 2 1)}}yes{{/if}}", "yes"},
		{"{{#or 2 1}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 0 1}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or '' ''}}yes{{else}}no{{/or}}", "no"},
		{"{{#or 'a' 'b'}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 'a' ''}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 0 1 0}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 0 0 1}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 0 0 0 1}}yes{{else}}no{{/or}}", "yes"},
		{"{{#or 0 0 0 0}}yes{{else}}no{{/or}}", "no"},
		{"{{or 2 1}}", "true"},
		{"{{or 0 1}}", "true"},
		{"{{or 2 1 yes='y'}}", "y"},
		{"{{or 0 0 no='n'}}", "n"},
		{"{{#if (or 2 1)}}yes{{/if}}", "yes"},
		{"{{#not true}}yes{{else}}no{{/not}}", "no"},
		{"{{#not 1}}yes{{else}}no{{/not}}", "no"},
		{"{{#not false}}yes{{else}}no{{/not}}", "yes"},
		{"{{#not 0}}yes{{else}}no{{/not}}", "yes"},
		{"{{not false}}", "true"},
		{"{{not false yes='y'}}", "y"},
		{"{{not true no='n'}}", "n"},
		{"{{#if (not false)}}yes{{/if}}", "yes"},
	}
	cases := make([]renderCase, 0, len(pairs))
	for _, pair := range pairs {
		cases = append(cases, renderCase{name: pair[0], template: pair[0], data: object(), expected: pair[1], configure: configure})
	}
	runRenderCases(t, cases)
}
