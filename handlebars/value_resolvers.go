// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"reflect"
	"strings"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// MapValueResolver resolves entries in Java and Go maps.
type MapValueResolver struct{}

// Resolve resolves a named map entry.
func (MapValueResolver) Resolve(context any, name string) (any, bool, error) {
	switch m := context.(type) {
	case *jvm.Map:
		v, ok := m.Get(name)
		return v, ok, nil
	case map[string]any:
		v, ok := m[name]
		return v, ok, nil
	}
	return nil, false, nil
}

// ResolveValue accepts map contexts unchanged.
func (MapValueResolver) ResolveValue(context any) (any, bool, error) {
	switch context.(type) {
	case *jvm.Map, map[string]any:
		return context, true, nil
	}
	return nil, false, nil
}

// PropertySet returns map entries.
func (MapValueResolver) PropertySet(context any) ([]Property, error) {
	switch m := context.(type) {
	case *jvm.Map:
		out := make([]Property, 0, m.Len())
		for _, e := range m.Entries() {
			out = append(out, Property{Name: jvm.ToString(e.Key), Value: e.Value})
		}
		return out, nil
	case map[string]any:
		out := make([]Property, 0, len(m))
		for k, v := range m {
			out = append(out, Property{Name: k, Value: v})
		}
		return out, nil
	}
	return nil, nil
}

// MethodValueResolver invokes public zero-argument members.
type MethodValueResolver struct{}

// Resolve invokes a named method.
func (MethodValueResolver) Resolve(context any, name string) (any, bool, error) {
	return jvm.InvokeMember(context, name)
}

// ResolveValue does not transform contexts.
func (MethodValueResolver) ResolveValue(any) (any, bool, error) { return nil, false, nil }

// PropertySet lists deterministic public properties supplied by the JVM model.
func (MethodValueResolver) PropertySet(context any) ([]Property, error) {
	entries, ok := jvm.PropertySet(context)
	if !ok {
		return nil, nil
	}
	out := make([]Property, 0, len(entries))
	for _, e := range entries {
		out = append(out, Property{Name: jvm.ToString(e.Key), Value: e.Value})
	}
	return out, nil
}

// JavaBeanValueResolver resolves getX and isX properties.
type JavaBeanValueResolver struct{}

// Resolve invokes a JavaBean getter.
func (JavaBeanValueResolver) Resolve(context any, name string) (any, bool, error) {
	return jvm.GetProperty(context, name)
}

// ResolveValue does not transform contexts.
func (JavaBeanValueResolver) ResolveValue(any) (any, bool, error) { return nil, false, nil }

// PropertySet lists modeled bean properties.
func (JavaBeanValueResolver) PropertySet(context any) ([]Property, error) {
	return (MethodValueResolver{}).PropertySet(context)
}

// FieldValueResolver resolves exported Go fields and array length.
type FieldValueResolver struct{}

// Resolve resolves a field.
func (FieldValueResolver) Resolve(context any, name string) (any, bool, error) {
	rv := reflect.ValueOf(context)
	if !rv.IsValid() {
		return nil, false, nil
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false, nil
		}
		rv = rv.Elem()
	}
	if (rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice) && name == "length" {
		return int32(rv.Len()), true, nil
	}
	if rv.Kind() != reflect.Struct {
		return nil, false, nil
	}
	field := rv.FieldByName(name)
	if !field.IsValid() && name != "" {
		field = rv.FieldByName(strings.ToUpper(name[:1]) + name[1:])
	}
	if field.IsValid() && field.CanInterface() {
		return field.Interface(), true, nil
	}
	return nil, false, nil
}

// ResolveValue does not transform contexts.
func (FieldValueResolver) ResolveValue(any) (any, bool, error) { return nil, false, nil }

// PropertySet lists exported fields.
func (FieldValueResolver) PropertySet(context any) ([]Property, error) {
	rv := reflect.ValueOf(context)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, nil
	}
	typ := rv.Type()
	var out []Property
	for i := 0; i < rv.NumField(); i++ {
		if typ.Field(i).IsExported() && rv.Field(i).CanInterface() {
			out = append(out, Property{Name: typ.Field(i).Name, Value: rv.Field(i).Interface()})
		}
	}
	return out, nil
}

// MemberValueResolver composes field, bean and method resolution.
type MemberValueResolver struct {
	// Resolvers are tried in order until one resolves or fails.
	Resolvers []ValueResolver
}

// Resolve delegates until one resolver succeeds.
func (r MemberValueResolver) Resolve(context any, name string) (any, bool, error) {
	for _, resolver := range r.Resolvers {
		v, ok, err := resolver.Resolve(context, name)
		if err != nil || ok {
			return v, ok, err
		}
	}
	return nil, false, nil
}

// ResolveValue delegates context conversion.
func (r MemberValueResolver) ResolveValue(context any) (any, bool, error) {
	for _, resolver := range r.Resolvers {
		v, ok, err := resolver.ResolveValue(context)
		if err != nil || ok {
			return v, ok, err
		}
	}
	return nil, false, nil
}

// PropertySet combines resolver property sets.
func (r MemberValueResolver) PropertySet(context any) ([]Property, error) {
	var out []Property
	seen := make(map[string]bool)
	for _, resolver := range r.Resolvers {
		set, err := resolver.PropertySet(context)
		if err != nil {
			return nil, err
		}
		for _, p := range set {
			if !seen[p.Name] {
				seen[p.Name] = true
				out = append(out, p)
			}
		}
	}
	return out, nil
}

var (
	// MapValueResolverInstance is the shared map resolver.
	MapValueResolverInstance ValueResolver = MapValueResolver{}
	// JavaBeanValueResolverInstance is the shared bean resolver.
	JavaBeanValueResolverInstance ValueResolver = JavaBeanValueResolver{}
	// MethodValueResolverInstance is the shared method resolver.
	MethodValueResolverInstance ValueResolver = MethodValueResolver{}
	// FieldValueResolverInstance is the shared field resolver.
	FieldValueResolverInstance ValueResolver = FieldValueResolver{}
)

// DefaultValueResolvers returns the Java default resolver order.
func DefaultValueResolvers() []ValueResolver {
	return []ValueResolver{MapValueResolverInstance, JavaBeanValueResolverInstance, MethodValueResolverInstance}
}
