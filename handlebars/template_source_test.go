// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"reflect"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestAbstractTemplateSource(t *testing.T) {
	first := handlebars.NewStringTemplateSource("file", "abc")
	second := handlebars.NewStringTemplateSource("file", "abc")
	if !reflect.DeepEqual(first, first) {
		t.Errorf("reflect.DeepEqual(source, source) = false; expected true for %#v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("reflect.DeepEqual(%#v, %#v) = false; expected true", first, second)
	}
	for _, source := range []handlebars.TemplateSource{first, second} {
		actual, err := source.Content()
		if err != nil || actual != "abc" {
			t.Errorf("Content(%#v) = %q, %v; expected %q, nil", source, actual, err, "abc")
		}
		if actual := source.Filename(); actual != "file" {
			t.Errorf("Filename(%#v) = %q; expected %q", source, actual, "file")
		}
	}
}

func TestIssue157WhitespaceTemplateName(t *testing.T) {
	template, err := handlebars.New(handlebars.NewFileTemplateLoader("testdata")).Compile("space between")
	if err != nil {
		t.Fatalf("Compile(%q) error = %v; expected output %q", "space between", err, "works!")
	}
	actual, applyErr := template.Apply(nil)
	if applyErr != nil || actual != "works!" {
		t.Errorf("Apply(%q) = %q, %v; expected %q, nil", "space between", actual, applyErr, "works!")
	}
}
