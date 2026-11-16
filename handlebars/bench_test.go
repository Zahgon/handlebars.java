// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func BenchmarkCompileInline(b *testing.B) {
	engine := handlebars.New()
	const input = "Hello {{name}}! {{#each values}}{{@index}}={{this}};{{/each}}"
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := engine.CompileInline(input); err != nil {
			b.Fatalf("CompileInline error: %v; expected nil", err)
		}
	}
}

func BenchmarkApply(b *testing.B) {
	template, err := handlebars.New().CompileInline("Hello {{name}}! {{#each values}}{{@index}}={{this}};{{/each}}")
	if err != nil {
		b.Fatalf("CompileInline error: %v; expected nil", err)
	}
	data := object("name", "Handlebars.java", "values", []any{"a", "b", "c"})
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := template.Apply(data); err != nil {
			b.Fatalf("Apply error: %v; expected nil", err)
		}
	}
}
