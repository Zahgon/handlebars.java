// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars_test

// ClassPathTemplateLoaderTest is omitted because Go has no Java class-path resource loader; fs.FS behavior is checked below.
// ServletContextTemplateLoaderTest is omitted because Go has no ServletContext; fs.FS behavior is checked below.
// MultipleClassLoadersMethodValueResolverTest is omitted because Go has no Java ClassLoader graph.
// ForwardingTemplateSourceTest's Mockito delegation checks are adapted to observable loader/source behavior below.
// URLTemplateSourceTest's Mockito URLConnection checks are adapted to file/fs source content behavior below.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	handlebars "github.com/jknack/handlebars.java/handlebars"
)

func TestFileTemplateLoader(t *testing.T) {
	loader := handlebars.NewFileTemplateLoader("testdata")
	source, err := loader.SourceAt("template")
	if err != nil {
		t.Fatalf("SourceAt(template) error: %v; expected a source", err)
	}
	content, err := source.Content()
	if err != nil {
		t.Fatalf("Content() error: %v; expected template.hbs content", err)
	}
	if content == "" {
		t.Errorf("Content() = %q; expected non-empty template.hbs content", content)
	}

	yaml := handlebars.NewFileTemplateLoaderWithSuffix("testdata", ".yml")
	if _, err := yaml.SourceAt("/mustache/specs/comments"); err != nil {
		t.Errorf("SourceAt(/mustache/specs/comments) error: %v; expected success", err)
	}

	withoutSuffix := handlebars.NewFileTemplateLoaderWithSuffix("testdata", "")
	plain, err := withoutSuffix.SourceAt("noextension")
	if err != nil {
		t.Fatalf("SourceAt(noextension) error: %v; expected success", err)
	}
	actual, err := plain.Content()
	if err != nil || actual != "suffix should be optional" {
		t.Errorf("noextension content = %q, %v; expected %q, nil", actual, err, "suffix should be optional")
	}
}

func TestTemplateLoaderPathTraversal(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "valid.hbs"), []byte("Hello {{name}}"), 0o600); err != nil {
		t.Fatalf("WriteFile(valid.hbs) error: %v; expected nil", err)
	}
	loader := handlebars.NewFileTemplateLoader(base)
	source, err := loader.SourceAt("valid")
	if err != nil {
		t.Fatalf("SourceAt(valid) error: %v; expected nil", err)
	}
	content, err := source.Content()
	if err != nil || content != "Hello {{name}}" {
		t.Errorf("valid content = %q, %v; expected %q, nil", content, err, "Hello {{name}}")
	}
	_, err = loader.SourceAt("../secret")
	if err == nil || !strings.Contains(err.Error(), "escapes base directory") {
		t.Errorf("SourceAt(../secret) error = %v; expected error containing %q", err, "escapes base directory")
	}

	fsLoader := handlebars.NewFSTemplateLoader(fstest.MapFS{"templates/ok.hbs": {Data: []byte("ok")}}, "templates", ".hbs")
	if _, err := fsLoader.SourceAt("ok"); err != nil {
		t.Errorf("FSTemplateLoader.SourceAt(ok) error: %v; expected nil", err)
	}
	_, err = fsLoader.SourceAt("../../outside")
	if err == nil || !strings.Contains(err.Error(), "escapes base prefix") {
		t.Errorf("FSTemplateLoader traversal error = %v; expected error containing %q", err, "escapes base prefix")
	}
}

func TestCompositeTemplateLoader(t *testing.T) {
	first := handlebars.NewMapTemplateLoader(map[string]string{"first": "one"})
	second := handlebars.NewMapTemplateLoader(map[string]string{"second": "two"})
	engine := handlebars.New(handlebars.NewCompositeTemplateLoader(first, second))
	for name, expected := range map[string]string{"first": "one", "second": "two"} {
		template, err := engine.Compile(name)
		if err != nil {
			t.Fatalf("Compile(%q) error: %v; expected output %q", name, err, expected)
		}
		actual, err := template.Apply(nil)
		if err != nil || actual != expected {
			t.Errorf("Apply(%q) = %q, %v; expected %q, nil", name, actual, err, expected)
		}
	}
}

func TestTemplateCaches(t *testing.T) {
	values := map[string]string{"template": "one"}
	loader := handlebars.NewMapTemplateLoader(values)
	cache := handlebars.NewConcurrentMapTemplateCache()
	engine := handlebars.New(loader).WithCache(cache)
	first, err := engine.Compile("template")
	if err != nil {
		t.Fatalf("first Compile error: %v; expected nil", err)
	}
	second, err := engine.Compile("template")
	if err != nil || first != second {
		t.Errorf("cached templates equal = %v, error = %v; expected true, nil", first == second, err)
	}
	cache.Evict(mustSource(t, loader, "template"))
	third, err := engine.Compile("template")
	if err != nil || first == third {
		t.Errorf("template after eviction differs = %v, error = %v; expected true, nil", first != third, err)
	}
	cache.Clear()

	nullEngine := handlebars.New(loader).WithCache(handlebars.NullTemplateCache)
	nullFirst, _ := nullEngine.Compile("template")
	nullSecond, err := nullEngine.Compile("template")
	if err != nil || nullFirst == nullSecond {
		t.Errorf("NullTemplateCache templates differ = %v, error = %v; expected true, nil", nullFirst != nullSecond, err)
	}
}

func mustSource(t *testing.T, loader handlebars.TemplateLoader, name string) handlebars.TemplateSource {
	t.Helper()
	source, err := loader.SourceAt(name)
	if err != nil {
		t.Fatalf("SourceAt(%q) error: %v; expected nil", name, err)
	}
	return source
}
