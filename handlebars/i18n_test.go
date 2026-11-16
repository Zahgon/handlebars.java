// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"os"
	"testing"

	"github.com/jknack/handlebars.java/internal/jvm"
)

func TestI18NHelpers(t *testing.T) {
	previousLocale := jvm.Default()
	jvm.SetDefault(jvm.ForLanguageTag("en-US"))
	defer jvm.SetDefault(previousLocale)

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v; expected nil", err)
	}
	if err := os.Chdir("testdata"); err != nil {
		t.Fatalf("Chdir(testdata) error: %v; expected nil", err)
	}
	defer func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory error: %v; expected nil", err)
		}
	}()

	runRenderCases(t, []renderCase{
		{"default", "{{i18n \"hello\"}} Handlebars.java!", object(), "Hi Handlebars.java!", nil, nil, nil},
		{"Argentine Spanish", "{{i18n \"hello\" locale=\"es_AR\"}} Handlebars.java!", object(), "Hola Handlebars.java!", nil, nil, nil},
		{"formatted", "{{i18n \"formatted\" \"Handlebars.java\"}}!", nil, "Hi Handlebars.java!", nil, nil, nil},
		{"escaped", "{{i18n \"escaped\" \"Handlebars.java\"}}", nil, "Hi, &quot;Handlebars.java&quot;, a &lt;tag&gt; &#x60;in backticks&#x60; &amp; other entities", nil, nil, nil},
		{"custom bundle", "{{i18n \"hello\" bundle=\"myMessages\" locale=\"es_AR\"}}", nil, "Hola", nil, nil, nil},
		{"Issue488 UTF-8 bundle", "{{i18n \"utf8\" bundle=\"i488\" locale=\"en\"}}", nil, "Bonjour à tous.", nil, nil, nil},
		{"Issue293 custom default equivalent", "{{i18n \"hello\" bundle=\"i293\" locale=\"es_AR\"}}", nil, "i293 AR", nil, nil, nil},
	})

	for _, input := range []string{`{{i18n "missing"}}`, `{{i18n "key" bundle="missing"}}!`} {
		template, compileErr := newInlineTemplate(input)
		var applyErr error
		if compileErr == nil {
			_, applyErr = template.Apply(nil)
		}
		if compileErr == nil && applyErr == nil {
			t.Errorf("Compile/apply(%q) error = nil; expected missing resource error", input)
		}
	}

	t.Run("i18n JavaScript", func(t *testing.T) {
		const expectedJava17 = "<script type='text/javascript'>\n" +
			"  /* Spanish (Argentina) */\n" +
			"  I18n.translations = I18n.translations || {};\n" +
			"  I18n.translations['es_AR'] = {\n" +
			"    \"hello\": \"Hola\",\n" +
			"    \"formatted\": \"Hi {{arg0}}\",\n" +
			"    \"escaped\": \"Hi, &quot;{{arg0}}&quot;, a &lt;tag&gt; &#x60;in backticks&#x60; &amp; other &#x27;entities&#x27;\"\n" +
			"  };\n" +
			"</script>\n"
		const expectedJava18 = "<script type='text/javascript'>\n" +
			"  /* Spanish (Argentina) */\n" +
			"  I18n.translations = I18n.translations || {};\n" +
			"  I18n.translations['es_AR'] = {\n" +
			"    \"hello\": \"Hola\",\n" +
			"    \"escaped\": \"Hi, &quot;{{arg0}}&quot;, a &lt;tag&gt; &#x60;in backticks&#x60; &amp; other &#x27;entities&#x27;\",\n" +
			"    \"formatted\": \"Hi {{arg0}}\"\n" +
			"  };\n" +
			"</script>\n"
		template, compileErr := newInlineTemplate(`{{i18nJs "es_AR"}}`)
		if compileErr != nil {
			t.Fatalf("CompileInline(i18nJs) error = %v; expected %q or %q", compileErr, expectedJava17, expectedJava18)
		}
		actual, applyErr := template.Apply(nil)
		if applyErr != nil || (actual != expectedJava17 && actual != expectedJava18) {
			t.Errorf("Apply(i18nJs) = %q, %v; expected %q or %q, nil", actual, applyErr, expectedJava17, expectedJava18)
		}
	})
}
