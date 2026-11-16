// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

// PathExpression is one compiled path operation.
type PathExpression interface {
	String() string
	Local() bool
}

type propertyPath struct {
	text  string
	local bool
}

func (p propertyPath) String() string { return p.text }
func (p propertyPath) Local() bool    { return p.local }

// CompilePath splits a Handlebars path while preserving bracket-escaped segments.
func CompilePath(key string, parentScopeResolution ...bool) []PathExpression {
	local := false
	if len(parentScopeResolution) > 0 {
		local = !parentScopeResolution[0]
	}
	parts := splitPath(key)
	out := make([]PathExpression, 0, len(parts))
	for _, part := range parts {
		out = append(out, propertyPath{text: part, local: local || key == "this" || key == "." || len(key) >= 2 && key[:2] == "./"})
	}
	if len(out) == 0 {
		out = append(out, propertyPath{text: key, local: local})
	}
	return out
}

// PathCompiler exposes the Java PathCompiler operation as a stateless Go value.
type PathCompiler struct{}

// Compile compiles a path.
func (PathCompiler) Compile(key string, parentScopeResolution ...bool) []PathExpression {
	return CompilePath(key, parentScopeResolution...)
}
