// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

type mustacheSpecFile struct {
	Tests []mustacheSpec `json:"tests"`
}

type mustacheSpec struct {
	Name     string            `json:"name"`
	Desc     string            `json:"desc"`
	Template string            `json:"template"`
	Expected string            `json:"expected"`
	Data     any               `json:"data"`
	Partials map[string]string `json:"partials"`
}

func TestMustacheSpec(t *testing.T) {
	files := []string{
		"comments.json",
		"delimiters.json",
		"interpolation.json",
		"inverted.json",
		"partials.json",
		"sections.json",
		"~lambdas.json",
	}
	for _, filename := range files {
		filename := filename
		t.Run(filename, func(t *testing.T) {
			runMustacheSpecFile(t, filename)
		})
	}
}

func runMustacheSpecFile(t *testing.T, filename string) {
	t.Helper()
	pathname := filepath.Join("testdata", "mustache", "specs", filename)
	contents, err := os.ReadFile(pathname)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", pathname, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.UseNumber()
	var suite mustacheSpecFile
	if err := decoder.Decode(&suite); err != nil {
		t.Fatalf("Decode(%q) error: %v", pathname, err)
	}
	for number := range suite.Tests {
		number, spec := number, suite.Tests[number]
		t.Run(fmt.Sprintf("%02d_%s", number, spec.Name), func(t *testing.T) {
			// PartialsTest.java skips exactly spec number 1: Handlebars.java
			// reports a missing partial while the Mustache spec renders empty text.
			if filename == "partials.json" && number == 1 {
				t.Skip("original PartialsTest skips failed partial lookup")
			}
			spec.Data = normalizeJSONNumbers(spec.Data)
			if filename == "~lambdas.json" {
				installSpecLambda(t, number, spec.Data)
			}
			templates := make(map[string]string, len(spec.Partials)+1)
			for name, text := range spec.Partials {
				templates[name] = text
			}
			templates["template"] = spec.Template
			engine := handlebars.New(handlebars.NewMapTemplateLoader(templates)).PrettyPrint(true)
			if filename == "partials.json" {
				engine.InfiniteLoops(true)
			}
			template, err := engine.Compile("template")
			if err != nil {
				t.Fatalf("Compile(%q) error: %v; expected %q", spec.Template, err, spec.Expected)
			}
			actual, err := template.Apply(spec.Data)
			if err != nil {
				t.Fatalf("Apply(%q) error: %v; expected %q", spec.Template, err, spec.Expected)
			}
			if actual != spec.Expected {
				t.Errorf("%s: got %q; expected %q", spec.Desc, actual, spec.Expected)
			}
		})
	}
}

func normalizeJSONNumbers(value any) any {
	switch value := value.(type) {
	case json.Number:
		if integer, err := strconv.ParseInt(string(value), 10, 32); err == nil {
			return int32(integer)
		}
		if integer, err := strconv.ParseInt(string(value), 10, 64); err == nil {
			return integer
		}
		floating, _ := strconv.ParseFloat(string(value), 64)
		return floating
	case []any:
		for index := range value {
			value[index] = normalizeJSONNumbers(value[index])
		}
	case map[string]any:
		for name := range value {
			value[name] = normalizeJSONNumbers(value[name])
		}
	}
	return value
}

func installSpecLambda(t *testing.T, number int, data any) {
	t.Helper()
	values, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("lambda spec data = %T; expected map[string]any", data)
	}
	calls := 0
	values["lambda"] = handlebars.Lambda(func(_ any, template handlebars.Template) (any, error) {
		switch number {
		case 0:
			return "world", nil
		case 1:
			return "{{planet}}", nil
		case 2:
			return "|planet| => {{planet}}", nil
		case 3:
			calls++
			return strconv.Itoa(calls), nil
		case 4:
			return ">", nil
		case 5:
			return map[bool]string{true: "yes", false: "no"}[template.Text() == "{{x}}"], nil
		case 6:
			text := template.Text()
			return text + "{{planet}}" + text, nil
		case 7:
			text := template.Text()
			return text + "{{planet}} => |planet|" + text, nil
		case 8:
			return "__" + template.Text() + "__", nil
		case 9:
			return false, nil
		default:
			return nil, fmt.Errorf("unsupported lambda spec number %d", number)
		}
	})
}
