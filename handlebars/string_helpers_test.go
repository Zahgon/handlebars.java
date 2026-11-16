// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

// Mockito interaction counts have no Go counterpart; these cases assert each helper's observable result.

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestStringHelpers(t *testing.T) {
	register := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	runRenderCases(t, []renderCase{
		{"capitalize first", "{{capitalizeFirst this}}", "handlebars.java", "Handlebars.java", nil, nil, register},
		{"center", "{{center this size=19}}", "Handlebars.java", "  Handlebars.java  ", nil, nil, register},
		{"center pad", "{{center this size=19 pad=\"*\"}}", "Handlebars.java", "**Handlebars.java**", nil, nil, register},
		{"cut spaces", "{{cut this}}", "handle bars .  java", "handlebars.java", nil, nil, register},
		{"cut stars", "{{cut this \"*\"}}", "handle*bars*.**java", "handlebars.java", nil, nil, register},
		{"default null", "{{defaultIfEmpty this \"handlebars.java\"}}", nil, "handlebars.java", nil, nil, register},
		{"default false", "{{defaultIfEmpty this \"handlebars.java\"}}", false, "handlebars.java", nil, nil, register},
		{"default value", "{{defaultIfEmpty this \"handlebars.java\"}}", "something", "something", nil, nil, register},
		{"join", "{{{join this \", \"}}}", []any{"6", "7", "8"}, "6, 7, 8", nil, nil, register},
		{"join empty", "{{{join this \", \"}}}", []any{}, "", nil, nil, register},
		{"join values", "{{{join \"6\" 7 n8 \"-\"}}}", object("n8", int32(8)), "6-7-8", nil, nil, register},
		{"join affixes", "{{{join this \", \" prefix=\"<\" suffix=\">\"}}}", []any{"6", "7", "8"}, "<6, 7, 8>", nil, nil, register},
		{"left justify", "{{ljust this size=20}}", "Handlebars.java", "Handlebars.java     ", nil, nil, register},
		{"left justify pad", "{{ljust this size=17 pad=\"+\"}}", "Handlebars.java", "Handlebars.java++", nil, nil, register},
		{"right justify", "{{rjust this size=20}}", "Handlebars.java", "     Handlebars.java", nil, nil, register},
		{"right justify pad", "{{rjust this size=17 pad=\"+\"}}", "Handlebars.java", "++Handlebars.java", nil, nil, register},
		{"substring start", "{{substring this 11}}", "Handlebars.java", "java", nil, nil, register},
		{"substring range", "{{substring this 0 10}}", "Handlebars.java", "Handlebars", nil, nil, register},
		{"lower", "{{lower this}}", "Handlebars.java", "handlebars.java", nil, nil, register},
		{"upper", "{{upper this}}", "Handlebars.java", "HANDLEBARS.JAVA", nil, nil, register},
		{"slugify", "{{slugify this}}", "  Joel is a slug  ", "joel-is-a-slug", nil, nil, register},
		{"replace", "{{replace this \"...\" \"rocks\"}}", "Handlebars ...", "Handlebars rocks", nil, nil, register},
		{"format string", "{{stringFormat this \"handlebars.java\"}}", "Hello %s!", "Hello handlebars.java!", nil, nil, register},
		{"format decimal", "{{{stringFormat format value}}}", object("format", "10 / 3 = %.2f", "value", 10.0/3.0), "10 / 3 = 3.33", nil, nil, register},
		{"strip tags", "{{stripTags this}}", "<p>Handlebars <strong>Java</strong></p>", "Handlebars Java", nil, nil, register},
		{"strip multiline tags", "{{stripTags this}}", "<b>Joel</b>\n<button>is<\n/button> a <span>slug</span>", "Joel\nis a slug", nil, nil, register},
		{"capitalize words", "{{capitalize this}}", "handlebars java", "Handlebars Java", nil, nil, register},
		{"capitalize fully", "{{capitalize this fully=true}}", "HAndleBars JAVA", "Handlebars Java", nil, nil, register},
		{"abbreviate", "{{abbreviate this 13}}", "Handlebars.java", "Handlebars...", nil, nil, register},
		{"word wrap", "{{wordWrap this 5}}", "Joel is a slug", "Joel\nis a\nslug", nil, nil, register},
		{"yes", "{{yesno this}}", true, "yes", nil, nil, register},
		{"no", "{{yesno this}}", false, "no", nil, nil, register},
		{"maybe", "{{yesno this}}", nil, "maybe", nil, nil, register},
		{"custom yes no", "{{yesno this yes=\"aye\" no=\"nay\"}}", false, "nay", nil, nil, register},
		{"custom maybe", "{{yesno this maybe=\"whatever\"}}", nil, "whatever", nil, nil, register},
	})
}
