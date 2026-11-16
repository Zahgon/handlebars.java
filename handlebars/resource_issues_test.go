// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"bytes"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestIssue835EmptyTemplates(t *testing.T) {
	runRenderCases(t, []renderCase{
		{"zero-size partial", "-{{> i835}}-", object(), "--", nil, map[string]string{"i835": ""}, nil},
		{"empty inline", "", object(), "", nil, nil, nil},
	})
}

func TestIssue585Charset(t *testing.T) {
	for _, test := range []struct {
		charset  string
		expected []byte
	}{
		{"UTF-8", []byte{0xec, 0x84, 0xb8, 0xea, 0xb3, 0x84, 0xec, 0x95, 0xbc, 44, 32, 0xec, 0x95, 0x88, 0xeb, 0x85, 0x95}},
	} {
		engine := handlebars.New(handlebars.NewFileTemplateLoader("testdata")).WithCharset(test.charset)
		template, err := engine.Compile("issue585")
		if err != nil {
			t.Errorf("Compile(issue585, %s) error = %v; expected bytes %v", test.charset, err, test.expected)
			continue
		}
		actual, applyErr := template.Apply(nil)
		actualBytes := []byte(actual)
		if applyErr != nil || !bytes.Equal(actualBytes, test.expected) {
			t.Errorf("Apply(issue585, %s) bytes = %v, %v; expected %v, nil", test.charset, actualBytes, applyErr, test.expected)
		}
	}
}

func TestIssue1083EnumMapObservableBehavior(t *testing.T) {
	statuses := object("NEW", int32(10), "DONE", int32(20))
	data := object("statuses", statuses, "first", "NEW", "second", "DONE")
	runRenderCases(t, []renderCase{
		{"dot", "{{statuses.NEW}} {{statuses.DONE}}", data, "10 20", nil, nil, nil},
		{"bracket", "{{statuses.[NEW]}} {{statuses.[DONE]}}", data, "10 20", nil, nil, nil},
		{"literal lookup", "{{lookup statuses 'NEW'}} {{lookup statuses 'DONE'}}", data, "10 20", nil, nil, nil},
		{"resolved lookup", "{{lookup statuses first}} {{lookup statuses second}}", data, "10 20", nil, nil, nil},
		{"missing", "{{statuses.[DONE]}}", object("statuses", object("NEW", int32(10))), "", nil, nil, nil},
		{"null", "{{statuses.[NEW]}}", object("statuses", object("NEW", nil)), "", nil, nil, nil},
		{"empty", "{{statuses.[NEW]}}", object("statuses", object()), "", nil, nil, nil},
	})
}

func TestIssue275TemporalPartials(t *testing.T) {
	for _, test := range []struct {
		deleteAfter bool
		expected    string
	}{
		{true, "Items:\n\nItem: Custom\n...\nItem: 2\n...\n"},
		{false, "Items:\n\nItem: Custom\n...\nItem: Custom\n...\n"},
	} {
		engine := handlebars.New(handlebars.NewFileTemplateLoader("testdata/issue275"))
		engine.PrettyPrint(true).DeletePartialAfterMerge(test.deleteAfter)
		engine.RegisterHelper("item", func(context any, options *handlebars.Options) (any, error) {
			name := context.(map[string]any)["name"].(string)
			template, err := options.Handlebars.Compile("item" + name)
			if err != nil {
				return nil, err
			}
			actual, err := template.ApplyContext(options.Context)
			return handlebars.NewSafeString(actual), err
		})
		template, err := engine.Compile("temporal-partials")
		if err != nil {
			t.Errorf("Compile(temporal-partials, delete=%v) error = %v; expected output %q", test.deleteAfter, err, test.expected)
			continue
		}
		actual, applyErr := template.Apply([]any{object("name", "1"), object("name", "2")})
		if applyErr != nil || actual != test.expected {
			t.Errorf("Apply(temporal-partials, delete=%v) = %q, %v; expected %q, nil", test.deleteAfter, actual, applyErr, test.expected)
		}
	}
}
