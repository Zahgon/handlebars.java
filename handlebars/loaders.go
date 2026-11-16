// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

const (
	// DefaultTemplatePrefix is the default loader prefix.
	DefaultTemplatePrefix = "/"
	// DefaultTemplateSuffix is the default template suffix.
	DefaultTemplateSuffix = ".hbs"
)

type stringTemplateSource struct {
	filename, content string
	modified          int64
}

// StringTemplateSource is an immutable in-memory source.
type StringTemplateSource = stringTemplateSource

// NewStringTemplateSource creates an immutable in-memory source.
func NewStringTemplateSource(filename, content string) TemplateSource {
	return &stringTemplateSource{filename: filename, content: content, modified: int64(jvm.StringHashCode(content))}
}

// Content returns the source text.
func (s *stringTemplateSource) Content() (string, error) { return s.content, nil }

// Filename returns the source's logical name.
func (s *stringTemplateSource) Filename() string { return s.filename }

// LastModified returns the stable content-derived version.
func (s *stringTemplateSource) LastModified() (int64, error) { return s.modified, nil }

type fileTemplateSource struct{ filename, pathname string }

// URLTemplateSource is the Go resource-backed template source.
type URLTemplateSource = fileTemplateSource

// Content reads the current file contents.
func (s *fileTemplateSource) Content() (string, error) {
	b, err := os.ReadFile(s.pathname)
	if err != nil {
		return "", err
	}
	return jvm.DecodeUTF8(b), nil
}

// Filename returns the logical loader path.
func (s *fileTemplateSource) Filename() string { return s.filename }

// LastModified returns the file modification time and reports stat failures.
func (s *fileTemplateSource) LastModified() (int64, error) {
	st, err := os.Stat(s.pathname)
	if err != nil {
		return 0, err
	}
	return st.ModTime().UnixMilli(), nil
}

// FileTemplateLoader loads templates beneath a filesystem directory.
type FileTemplateLoader struct {
	base, display, prefix, suffix string
	initErr                       error
}

// NewFileTemplateLoader creates a file loader with the .hbs suffix.
func NewFileTemplateLoader(base string) *FileTemplateLoader {
	return NewFileTemplateLoaderWithSuffix(base, DefaultTemplateSuffix)
}

// NewFileTemplateLoaderWithSuffix creates a file loader.
func NewFileTemplateLoaderWithSuffix(base, suffix string) *FileTemplateLoader {
	clean, err := filepath.Abs(base)
	return &FileTemplateLoader{base: clean, display: base, prefix: base + string(os.PathSeparator), suffix: suffix, initErr: err}
}

// Resolve converts a logical template location into its display path.
func (l *FileTemplateLoader) Resolve(location string) string {
	location = strings.TrimPrefix(location, "/")
	base := l.display
	if !strings.HasSuffix(base, string(os.PathSeparator)) {
		base += string(os.PathSeparator)
	}
	return base + filepath.FromSlash(location) + l.suffix
}

// Prefix returns the configured filesystem prefix.
func (l *FileTemplateLoader) Prefix() string { return l.prefix }

// Suffix returns the configured template suffix.
func (l *FileTemplateLoader) Suffix() string { return l.suffix }

// SourceAt resolves location while rejecting paths outside the base directory.
func (l *FileTemplateLoader) SourceAt(location string) (TemplateSource, error) {
	if l.initErr != nil {
		return nil, fmt.Errorf("resolve template base %q: %w", l.display, l.initErr)
	}
	resolved := l.Resolve(location)
	canonical, err := filepath.Abs(resolved)
	if err != nil {
		return nil, err
	}
	base := l.base + string(os.PathSeparator)
	if !strings.HasPrefix(canonical, base) {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "Path traversal attempt detected. Resolved path escapes base directory: "+resolved)
	}
	if _, err := os.Stat(canonical); err != nil {
		if os.IsNotExist(err) {
			return nil, jvm.Throw("java.io.FileNotFoundException", filepath.ToSlash(resolved))
		}
		return nil, err
	}
	return &fileTemplateSource{filename: filepath.ToSlash(resolved), pathname: canonical}, nil
}

type fsTemplateSource struct {
	filesystem fs.FS
	name       string
	modified   int64
}

// Content reads the source through its fs.FS.
func (s *fsTemplateSource) Content() (string, error) {
	b, err := fs.ReadFile(s.filesystem, s.name)
	if err != nil {
		return "", err
	}
	return jvm.DecodeUTF8(b), nil
}

// Filename returns the fs-relative name.
func (s *fsTemplateSource) Filename() string { return s.name }

// LastModified returns the modification time captured by the loader.
func (s *fsTemplateSource) LastModified() (int64, error) { return s.modified, nil }

// FSTemplateLoader loads templates from an fs.FS.
type FSTemplateLoader struct {
	// FS is the filesystem searched by the loader.
	FS             fs.FS
	prefix, suffix string
}

// NewFSTemplateLoader creates an fs.FS loader.
func NewFSTemplateLoader(fsys fs.FS, prefix, suffix string) *FSTemplateLoader {
	prefix = strings.Trim(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	return &FSTemplateLoader{FS: fsys, prefix: prefix, suffix: suffix}
}

// Resolve combines the loader prefix, location, and suffix.
func (l *FSTemplateLoader) Resolve(location string) string {
	return l.prefix + strings.TrimPrefix(location, "/") + l.suffix
}

// Prefix returns the normalized fs prefix.
func (l *FSTemplateLoader) Prefix() string { return l.prefix }

// Suffix returns the configured template suffix.
func (l *FSTemplateLoader) Suffix() string { return l.suffix }

// SourceAt resolves and stats a source in the configured fs.FS.
func (l *FSTemplateLoader) SourceAt(location string) (TemplateSource, error) {
	name := path.Clean(l.Resolve(location))
	if strings.HasPrefix(name, "../") || name == ".." {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "Path traversal attempt detected. Resolved path escapes base prefix: "+name)
	}
	st, err := fs.Stat(l.FS, name)
	if err != nil {
		return nil, err
	}
	return &fsTemplateSource{filesystem: l.FS, name: name, modified: st.ModTime().UnixMilli()}, nil
}

// ClassPathTemplateLoader is the Go fs.FS equivalent of the Java classpath loader.
type ClassPathTemplateLoader = FSTemplateLoader

// ServletContextTemplateLoader is the Go fs.FS equivalent of the servlet loader.
type ServletContextTemplateLoader = FSTemplateLoader

// URLTemplateLoader is represented by FSTemplateLoader in Go.
type URLTemplateLoader = FSTemplateLoader

// MapTemplateLoader loads templates from a string map.
type MapTemplateLoader struct {
	// Templates maps logical or resolved names to source text.
	Templates      map[string]string
	prefix, suffix string
}

// NewMapTemplateLoader creates a map loader.
func NewMapTemplateLoader(values map[string]string) *MapTemplateLoader {
	return &MapTemplateLoader{Templates: values, prefix: DefaultTemplatePrefix, suffix: DefaultTemplateSuffix}
}

// Resolve combines the loader prefix, location, and suffix.
func (l *MapTemplateLoader) Resolve(location string) string {
	return l.prefix + strings.TrimPrefix(location, "/") + l.suffix
}

// Prefix returns the map-key prefix.
func (l *MapTemplateLoader) Prefix() string { return l.prefix }

// Suffix returns the map-key suffix.
func (l *MapTemplateLoader) Suffix() string { return l.suffix }

// SourceAt returns the mapped source or fs.ErrNotExist.
func (l *MapTemplateLoader) SourceAt(location string) (TemplateSource, error) {
	name := l.Resolve(location)
	text, ok := l.Templates[name]
	if !ok {
		text, ok = l.Templates[location]
	}
	if !ok {
		return nil, fs.ErrNotExist
	}
	return NewStringTemplateSource(name, text), nil
}

// CompositeTemplateLoader tries loaders in order.
type CompositeTemplateLoader struct{ delegates []TemplateLoader }

// NewCompositeTemplateLoader creates a composite loader.
func NewCompositeTemplateLoader(loaders ...TemplateLoader) *CompositeTemplateLoader {
	return &CompositeTemplateLoader{delegates: append([]TemplateLoader(nil), loaders...)}
}

// SourceAt tries delegates in order, continuing only after not-found errors.
func (l *CompositeTemplateLoader) SourceAt(location string) (TemplateSource, error) {
	var notFound error
	for _, d := range l.delegates {
		s, err := d.SourceAt(location)
		if err == nil {
			return s, nil
		}
		if !templateNotFound(err) {
			return nil, err
		}
		if notFound == nil {
			notFound = err
		}
	}
	if notFound != nil {
		return nil, notFound
	}
	return nil, fmt.Errorf("%s: %w", location, fs.ErrNotExist)
}
func templateNotFound(err error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	var ex *jvm.Exception
	return errors.As(err, &ex) && ex.Class == "java.io.FileNotFoundException"
}

// Resolve returns the first delegate path whose source exists.
func (l *CompositeTemplateLoader) Resolve(location string) string {
	if len(l.delegates) != 0 {
		return l.delegates[0].Resolve(location)
	}
	return ""
}

// Prefix returns an empty string because delegates own their prefixes.
func (l *CompositeTemplateLoader) Prefix() string { return "" }

// Suffix returns an empty string because delegates own their suffixes.
func (l *CompositeTemplateLoader) Suffix() string { return "" }

// Delegates returns the loader chain.
func (l *CompositeTemplateLoader) Delegates() []TemplateLoader {
	return append([]TemplateLoader(nil), l.delegates...)
}

// ReloadableTemplateSource decorates a source for modification-time caches.
type ReloadableTemplateSource struct{ TemplateSource }

// NewReloadableTemplateSource creates a reloadable source wrapper.
func NewReloadableTemplateSource(source TemplateSource) *ReloadableTemplateSource {
	return &ReloadableTemplateSource{TemplateSource: source}
}

// ForwardingTemplateSource delegates every operation to another source.
type ForwardingTemplateSource struct{ TemplateSource }

// NewForwardingTemplateSource creates a source delegate.
func NewForwardingTemplateSource(source TemplateSource) *ForwardingTemplateSource {
	return &ForwardingTemplateSource{TemplateSource: source}
}

// AbstractTemplateLoader contains the common prefix/suffix resolution policy.
type AbstractTemplateLoader struct {
	// ViewPrefix is prepended to logical locations.
	ViewPrefix string
	// ViewSuffix is appended to logical locations.
	ViewSuffix string
}

// NewAbstractTemplateLoader creates the default prefix/suffix policy.
func NewAbstractTemplateLoader() *AbstractTemplateLoader {
	return &AbstractTemplateLoader{ViewPrefix: DefaultTemplatePrefix, ViewSuffix: DefaultTemplateSuffix}
}

// Resolve combines prefix, normalized location and suffix.
func (l *AbstractTemplateLoader) Resolve(location string) string {
	return l.ViewPrefix + strings.TrimPrefix(location, "/") + l.ViewSuffix
}

// Prefix returns the prefix.
func (l *AbstractTemplateLoader) Prefix() string { return l.ViewPrefix }

// Suffix returns the suffix.
func (l *AbstractTemplateLoader) Suffix() string { return l.ViewSuffix }
