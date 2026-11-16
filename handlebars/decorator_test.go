// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

import (
	"fmt"
	"testing"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestDecorators(t *testing.T) {
	configureInline := func(engine *handlebars.Handlebars) {
		engine.RegisterHelper("helper", func(_ any, options *handlebars.Options) (any, error) {
			return options.Data("run"), nil
		})
		engine.RegisterDecorator("decorator", func(template handlebars.Template, options *handlebars.Options) (handlebars.Template, error) {
			options.SetData("run", "success")
			return template, nil
		})
	}
	configureBlock := func(engine *handlebars.Handlebars) {
		engine.RegisterHelper("helper", func(_ any, options *handlebars.Options) (any, error) {
			return options.Data("run"), nil
		})
		engine.RegisterDecorator("decorator", func(template handlebars.Template, options *handlebars.Options) (handlebars.Template, error) {
			body, err := options.Fn.ApplyContext(options.Context)
			if err == nil {
				options.SetData("run", body)
			}
			return template, err
		})
	}
	runRenderCases(t, []renderCase{
		{"inline decorator", "{{#helper}}{{*decorator}}{{/helper}}", object(), "success", nil, nil, configureInline},
		{"block decorator", "{{#helper}}{{#*decorator}}success{{/decorator}}{{/helper}}", object(), "success", nil, nil, configureBlock},
	})
}

func TestRootDecoratorExecutionCount(t *testing.T) {
	for count := 1; count <= 3; count++ {
		t.Run(fmt.Sprintf("%d decorators", count), func(t *testing.T) {
			actualCalls := 0
			engine := handlebars.New().RegisterDecorator("decorator", func(template handlebars.Template, _ *handlebars.Options) (handlebars.Template, error) {
				actualCalls++
				return template, nil
			})
			input := ""
			for index := 0; index < count; index++ {
				input += "{{*decorator}}"
			}
			template, err := engine.CompileInline(input)
			if err != nil {
				t.Fatalf("CompileInline(%q) error: %v; expected %d calls", input, err, count)
			}
			actualOutput, err := template.Apply(object())
			if err != nil || actualOutput != "" || actualCalls != count {
				t.Errorf("Apply = %q, %v, calls %d; expected %q, nil, calls %d", actualOutput, err, actualCalls, "", count)
			}
		})
	}
}
