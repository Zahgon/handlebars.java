// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jtime"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestDateFormat(t *testing.T) {
	previousLocale := jvm.Default()
	previousZone := jtime.SystemDefaultZone()
	jvm.SetDefault(jvm.ForLanguageTag("en-US"))
	jtime.SetSystemDefaultZone(jtime.ZoneFromTimeZoneID("UTC"))
	defer jvm.SetDefault(previousLocale)
	defer jtime.SetSystemDefaultZone(previousZone)

	date, err := jtime.NewLocalDate(2012, 6, 19)
	if err != nil {
		t.Fatalf("NewLocalDate error: %v; expected nil", err)
	}
	instant, err := jtime.NewInstant(1628779135, 0)
	if err != nil {
		t.Fatalf("NewInstant error: %v; expected nil", err)
	}
	register := func(engine *handlebars.Handlebars) { engine.RegisterStringHelpers() }
	runRenderCases(t, []renderCase{
		{"default", "{{dateFormat this}}", date, "Jun 19, 2012", nil, nil, register},
		{"full", "{{dateFormat this \"full\"}}", date, "Tuesday, June 19, 2012", nil, nil, register},
		{"long", "{{dateFormat this \"long\"}}", date, "June 19, 2012", nil, nil, register},
		{"medium", "{{dateFormat this \"medium\"}}", date, "Jun 19, 2012", nil, nil, register},
		{"short", "{{dateFormat this \"short\"}}", date, "6/19/12", nil, nil, register},
		{"pattern", "{{dateFormat this \"dd/MM/yyyy\"}}", date, "19/06/2012", nil, nil, register},
		{"French", "{{dateFormat this \"short\" \"fr\"}}", date, "19/06/2012", nil, nil, register},
		{"German medium", "{{dateFormat this \"medium\" locale=\"de\"}}", instant, "12.08.2021", nil, nil, register},
		{"English medium positional locale", "{{dateFormat this \"medium\" \"en_EN\"}}", instant, "Aug 12, 2021", nil, nil, register},
		{"day pattern", "{{dateFormat this \"dd\"}}", instant, "12", nil, nil, register},
		{"week minute second pattern", "{{dateFormat this \"'week' w '@' mm:ss\" \"de_DE\"}}", instant, "week 32 @ 38:55", nil, nil, register},
		{"German date time Berlin", "{{dateFormat this time=\"medium\" locale=\"de\" tz=\"Europe/Berlin\"}}", instant, "12.08.2021, 16:38:55", nil, nil, register},
		{"German long date short time GMT minus eight", "{{dateFormat this \"long\" locale=\"de\" time=\"short\" tz=\"GMT-8:00\"}}", instant, "12. August 2021, 06:38", nil, nil, register},
		{"instant default", "{{dateFormat this}}", instant, "Aug 12, 2021", nil, nil, register},
		{"instant default time", "{{dateFormat this time=\"medium\"}}", instant, "Aug 12, 2021, 2:38:55\u202fPM", nil, nil, register},
	})
}
