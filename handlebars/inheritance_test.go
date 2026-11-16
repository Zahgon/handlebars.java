// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"os"
	"path/filepath"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestTemplateInheritance(t *testing.T) {
	engine := handlebars.New(handlebars.NewFileTemplateLoader(filepath.Join("testdata", "inheritance"))).PrettyPrint(true)
	for _, name := range []string{"home", "about", "base"} {
		t.Run(name, func(t *testing.T) {
			expectedBytes, err := os.ReadFile(filepath.Join("testdata", "inheritance", name+".expected"))
			if err != nil {
				t.Fatalf("ReadFile(%s.expected) error: %v; expected fixture contents", name, err)
			}
			expected := string(expectedBytes)
			template, err := engine.Compile(name)
			if err != nil {
				t.Fatalf("Compile(%q) error: %v; expected %q", name, err, expected)
			}
			actual, err := template.Apply(object())
			if err != nil || actual != expected {
				t.Errorf("Apply(%q) = %q, %v; expected %q, nil", name, actual, err, expected)
			}
		})
	}
}
