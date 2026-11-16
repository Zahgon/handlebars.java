// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

package handlebars

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jknack/handlebars.java/internal/commons"
	"github.com/jknack/handlebars.java/internal/jtext"
	"github.com/jknack/handlebars.java/internal/jtime"
	"github.com/jknack/handlebars.java/internal/jvm"
)

func (h *Handlebars) registerBuiltins() {
	h.RegisterHelper("if", ifHelper(false)).RegisterHelper("unless", ifHelper(true)).RegisterHelper("with", withHelper).RegisterHelper("each", eachHelper)
	h.RegisterHelper("lookup", lookupHelper).RegisterHelper("embedded", embeddedHelper).RegisterHelper("block", blockHelper).RegisterHelper("partial", partialHelper)
	h.RegisterHelper("precompile", func(any, *Options) (any, error) { return nil, missingJSError() }).RegisterHelper("log", logHelper)
	h.RegisterHelper("i18n", i18nHelper).RegisterHelper("i18nJs", i18nJSHelper)
	h.RegisterDecorator("inline", func(t Template, _ *Options) (Template, error) { return t, nil })
}

func ifHelper(invert bool) Helper {
	return func(context any, o *Options) (any, error) {
		truth := !o.IsFalsy(context)
		if invert {
			truth = !truth
		}
		if truth {
			return renderOptionTemplate(o, o.Fn, o.Context, nil)
		}
		return renderOptionTemplate(o, o.Inverse, o.Context, nil)
	}
}
func withHelper(context any, o *Options) (any, error) {
	if o.IsFalsy(context) {
		return renderOptionTemplate(o, o.Inverse, o.Context, nil)
	}
	child := NewChildContext(o.Context, context)
	props, err := o.Context.PropertySet(o.Context.Model())
	if err != nil {
		return nil, err
	}
	for _, property := range props {
		if reflect.DeepEqual(property.Value, context) {
			child.shiftedAlias = property.Name
			break
		}
	}
	if len(o.BlockParams) > 0 {
		child = NewBlockParamContext(child, context, o.BlockParams, []any{context})
	}
	return renderOptionTemplate(o, o.Fn, child, []any{context})
}
func eachHelper(context any, o *Options) (any, error) {
	var b strings.Builder
	base := 0
	if v, ok := o.Hash["base"]; ok {
		base = int(jvm.IntValue(v))
	}
	if values, ok := elements(context); ok {
		for i, v := range values {
			idx := base + i
			c := NewChildContext(o.Context, v)
			c.Combine("@key", int32(idx)).Combine("@index", int32(idx)).Combine("@first", boolWord(i == 0, "first")).Combine("@last", boolWord(i == len(values)-1, "last")).Combine("@odd", boolWord(idx%2 != 0, "odd")).Combine("@even", boolWord(idx%2 == 0, "even")).Combine("@index_1", int32(idx+1))
			if len(o.BlockParams) > 0 {
				c = NewBlockParamContext(c, v, o.BlockParams, []any{v, int32(idx)})
			}
			text, err := renderOptionTemplate(o, o.Fn, c, nil)
			if err != nil {
				return nil, err
			}
			b.WriteString(text)
		}
		if len(values) == 0 {
			text, err := renderOptionTemplate(o, o.Inverse, o.Context, nil)
			return text, err
		}
		return b.String(), nil
	}
	props, err := o.PropertySet(context)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return renderOptionTemplate(o, o.Inverse, o.Context, nil)
	}
	for i, p := range props {
		c := NewChildContext(o.Context, p.Value)
		c.Combine("@key", p.Name).Combine("@index", int32(i)).Combine("@first", boolWord(i == 0, "first")).Combine("@last", boolWord(i == len(props)-1, "last"))
		if len(o.BlockParams) > 0 {
			c = NewBlockParamContext(c, p.Value, o.BlockParams, []any{p.Value, p.Name})
		}
		text, err := renderOptionTemplate(o, o.Fn, c, nil)
		if err != nil {
			return nil, err
		}
		b.WriteString(text)
	}
	return b.String(), nil
}
func boolWord(v bool, s string) string {
	if v {
		return s
	}
	return ""
}
func blockParamValues(names []string, values []any) map[string]any {
	result := make(map[string]any, len(names))
	for i, name := range names {
		if i < len(values) {
			result[name] = values[i]
		}
	}
	return result
}
func renderOptionTemplate(o *Options, t Template, c *Context, _ []any) (string, error) {
	if t == nil {
		return "", nil
	}
	return t.ApplyContext(c)
}
func lookupHelper(context any, o *Options) (any, error) {
	if context == nil {
		return nil, nil
	}
	if len(o.Params) == 0 {
		return context, nil
	}
	name := toJavaString(o.Param(0))
	v, err := NewChildContext(o.Context, context).Resolve(name, o.Handlebars.parentScopeResolution)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return context, nil
	}
	return v, nil
}
func embeddedHelper(context any, o *Options) (any, error) {
	name := toJavaString(context)
	t, err := o.Handlebars.Compile(name)
	if err != nil {
		return nil, err
	}
	var id string
	if len(o.Params) > 0 {
		id = toJavaString(o.Param(0))
	} else {
		id = strings.NewReplacer("/", "-", ".", "-").Replace(name + o.Handlebars.loader.Suffix())
	}
	return SafeString{Content: "<script id=\"" + id + "\" type=\"text/x-handlebars\">\n" + t.Text() + "\n</script>"}, nil
}

// isIOError reports whether err is an I/O failure of a template loader (a missing file or a read error), the
// failures the original reports as java.io exceptions, as opposed to syntax or argument errors.
func isIOError(err error) bool {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) || errors.Is(err, fs.ErrNotExist) {
		return true
	}
	var ex *jvm.Exception
	return errors.As(err, &ex) && strings.HasPrefix(ex.Class, "java.io.")
}

func blockHelper(context any, o *Options) (any, error) {
	if _, ok := context.(string); !ok {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "found '"+toJavaString(context)+"', expected 'partial's name'")
	}
	name := toJavaString(context)
	template := o.Partial(name)
	if template == nil {
		compiled, err := o.Handlebars.Compile(name)
		switch {
		case err == nil:
			template = compiled
		case isIOError(err):
			// BlockHelper.java treats an IOException while loading the block's template as a missing
			// partial (it only logs it at debug level) and renders the block's own body instead.
			template = o.Fn
		default:
			return nil, err
		}
	}
	if tag, ok := o.Data(ContextPartials + "#" + name + "#type").(TagType); ok && tag.Inline() {
		template = o.Fn
	}
	if template == nil {
		template = o.Fn
	}
	if nodes, ok := template.(*nodeTemplate); ok && len(nodes.nodes) == 0 {
		template = o.Fn
	}
	if template == nil {
		return "", nil
	}
	hash := make(map[string]any, len(o.Hash))
	for k, v := range o.Hash {
		hash[k] = v
	}
	if registered, ok := o.Data(ContextPartials + "#" + name + "#hash").(map[string]any); ok {
		for k, v := range registered {
			hash[k] = v
		}
	}
	result, err := template.ApplyContext(NewPartialContext(o.Context, o.Context.Model(), hash))
	deleteAfter, _ := o.HashValue("delete-after-merge", o.Handlebars.deletePartialAfterMerge).(bool)
	if deleteAfter {
		o.SetPartial(name, nil)
		o.SetData(ContextPartials+"#"+name+"#hash", nil)
	}
	return result, err
}
func partialHelper(context any, o *Options) (any, error) {
	if _, ok := context.(string); !ok {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "found '"+toJavaString(context)+"', expected 'partial's name'")
	}
	name := toJavaString(context)
	template := o.Fn
	if template == nil {
		filename := ""
		if o.state != nil {
			filename = o.state.filename
		}
		template = &nodeTemplate{engine: o.Handlebars, filename: filename}
	}
	o.SetPartial(name, template)
	o.SetData(ContextPartials+"#"+name+"#hash", o.Hash)
	o.SetData(ContextPartials+"#"+name+"#type", o.TagType)
	return "", nil
}
func logHelper(context any, o *Options) (any, error) {
	if o.Fn != nil {
		body, err := renderOptionTemplate(o, o.Fn, o.Context, nil)
		if err != nil {
			return nil, err
		}
		level := strings.ToUpper(toJavaString(o.HashValue("level", "info")))
		if level == "TRACE" || level == "DEBUG" {
			return "", nil
		}
		if _, err := os.Stderr.Write(jvm.EncodeUTF8(fmt.Sprintf("%s com.github.jknack.handlebars.helper.LogHelper - %s\n", level, strings.TrimSpace(body)))); err != nil {
			return nil, fmt.Errorf("could not write log message: %w", err)
		}
		return "", nil
	}
	args := append([]any{context}, o.Params...)
	parts := make([]string, len(args))
	for i, v := range args {
		parts[i] = toJavaString(v)
	}
	level := strings.ToUpper(toJavaString(o.HashValue("level", "info")))
	if level == "TRACE" || level == "DEBUG" {
		return "", nil
	}
	if level == "" {
		level = "INFO"
	}
	if _, err := os.Stderr.Write(jvm.EncodeUTF8(fmt.Sprintf("%s com.github.jknack.handlebars.helper.LogHelper - %s\n", level, strings.Join(parts, " ")))); err != nil {
		return nil, fmt.Errorf("could not write log message: %w", err)
	}
	return "", nil
}

func i18nHelper(context any, o *Options) (any, error) {
	loc := jvm.Default()
	if raw, ok := o.Hash["locale"]; ok {
		localeName, ok := raw.(string)
		if !ok {
			return nil, stringClassCast(raw)
		}
		parsed, err := commons.LocaleToLocale(strings.ReplaceAll(localeName, "-", "_"))
		if err != nil {
			return nil, err
		}
		loc = parsed
	}
	bundle, err := jtext.LoadBundle("i18n", toJavaString(o.HashValue("bundle", "messages")), loc, jvm.Default())
	if err != nil {
		return nil, err
	}
	pattern, err := bundle.GetString(toJavaString(context))
	if err != nil {
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "no message found: '"+toJavaString(context)+"' for locale '"+loc.String()+"'.")
	}
	return jtext.MessageFormat(pattern, loc, o.Params)
}

func i18nJSHelper(context any, o *Options) (any, error) {
	loc := jvm.Default()
	name := toJavaString(context)
	if name != "" {
		if _, ok := context.(string); !ok {
			return nil, stringClassCast(context)
		}
		parsed, err := commons.LocaleToLocale(strings.ReplaceAll(name, "-", "_"))
		if err != nil {
			return nil, err
		}
		loc = parsed
	}
	bundle, err := jtext.LoadBundle("i18n", toJavaString(o.HashValue("bundle", "messages")), loc, jvm.Default())
	if err != nil {
		return nil, err
	}
	wrap, _ := o.HashValue("wrap", true).(bool)
	var b strings.Builder
	if wrap {
		b.WriteString("<script type='text/javascript'>\n")
	}
	b.WriteString("  /* " + jtext.DisplayName(loc, jvm.Default()) + " */\n  I18n.translations = I18n.translations || {};\n  I18n.translations['" + loc.String() + "'] = {\n")
	keys := bundle.Keys()
	re := regexp.MustCompile(`\{(\d+)\}`)
	for i, key := range keys {
		msg, e := bundle.GetString(key)
		if e != nil {
			return nil, e
		}
		msg = re.ReplaceAllString(HTMLENTITY.Escape(msg), "{{arg$1}}")
		b.WriteString("    \"" + key + "\": \"" + msg + "\"")
		if i+1 < len(keys) {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  };\n")
	if wrap {
		b.WriteString("</script>\n")
	}
	return SafeString{Content: b.String()}, nil
}

func stringClassCast(value any) error {
	class := jvm.ClassName(value)
	return jvm.Throw("java.lang.ClassCastException", "class "+class+" cannot be cast to class java.lang.String ("+class+" and java.lang.String are in module java.base of loader 'bootstrap')")
}

// RegisterConditionalHelpers registers eq, neq, gt, gte, lt, lte, and, or and not.
func (h *Handlebars) RegisterConditionalHelpers() *Handlebars {
	for _, name := range []string{"eq", "neq", "gt", "gte", "lt", "lte", "and", "or", "not"} {
		op := name
		h.RegisterHelper(op, func(context any, o *Options) (any, error) {
			result, err := conditional(op, context, o)
			if err != nil {
				return nil, err
			}
			if o.TagType == TagSection {
				if result {
					return renderOptionTemplate(o, o.Fn, o.Context, nil)
				}
				return renderOptionTemplate(o, o.Inverse, o.Context, nil)
			}
			if result {
				return o.HashValue("yes", true), nil
			}
			return o.HashValue("no", false), nil
		})
	}
	return h
}
func conditional(op string, a any, o *Options) (bool, error) {
	switch op {
	case "eq", "neq":
		b := o.Param(0)
		equal := jvm.Equals(a, b)
		if jvm.IsNumber(a) && jvm.IsNumber(b) && reflect.TypeOf(a) != reflect.TypeOf(b) {
			equal = jvm.DoubleValue(a) == jvm.DoubleValue(b)
		}
		if op == "neq" {
			equal = !equal
		}
		return equal, nil
	case "gt", "gte", "lt", "lte":
		b := o.Param(0)
		if !conditionalComparable(a) {
			return false, jvm.Throw("java.lang.IllegalArgumentException", "Not a comparable: "+toJavaString(a))
		}
		if !conditionalComparable(b) {
			return false, jvm.Throw("java.lang.IllegalArgumentException", "Not a comparable: "+toJavaString(b))
		}
		cmp, err := jvm.CompareTo(a, b)
		if err != nil {
			if jvm.IsNumber(a) && jvm.IsNumber(b) {
				av, bv := jvm.DoubleValue(a), jvm.DoubleValue(b)
				cmp = 0
				if av < bv {
					cmp = -1
				} else if av > bv {
					cmp = 1
				}
				err = nil
			}
		}
		if err != nil {
			return false, err
		}
		switch op {
		case "gt":
			return cmp > 0, nil
		case "gte":
			return cmp >= 0, nil
		case "lt":
			return cmp < 0, nil
		default:
			return cmp <= 0, nil
		}
	case "and":
		if IsEmpty(a) {
			return false, nil
		}
		for _, v := range o.Params {
			if IsEmpty(v) {
				return false, nil
			}
		}
		return true, nil
	case "or":
		if !IsEmpty(a) {
			return true, nil
		}
		for _, v := range o.Params {
			if !IsEmpty(v) {
				return true, nil
			}
		}
		return false, nil
	case "not":
		return IsEmpty(a), nil
	}
	return false, nil
}

func conditionalComparable(value any) bool {
	if jvm.IsNumber(value) {
		return true
	}
	switch value.(type) {
	case string, bool, jvm.Char:
		return true
	}
	return false
}

// RegisterStringHelpers registers every StringHelpers enum constant.
func (h *Handlebars) RegisterStringHelpers() *Handlebars {
	for _, name := range []string{"capitalizeFirst", "center", "cut", "defaultIfEmpty", "join", "ljust", "rjust", "substring", "lower", "upper", "slugify", "stringFormat", "stripTags", "capitalize", "abbreviate", "wordWrap", "replace", "yesno", "dateFormat", "numberFormat", "now"} {
		n := name
		h.RegisterHelper(n, stringHelper(n))
	}
	return h
}
func stringHelper(name string) Helper {
	return func(context any, o *Options) (any, error) {
		if name == "yesno" {
			if context == nil {
				return o.HashValue("maybe", "maybe"), nil
			}
			value, ok := context.(bool)
			if !ok {
				return nil, jvm.Throw("java.lang.IllegalArgumentException", "found '"+toJavaString(context)+"', expected 'boolean'")
			}
			if value {
				return o.HashValue("yes", "yes"), nil
			}
			return o.HashValue("no", "no"), nil
		}
		// Zero is empty to section helpers but remains a perfectly valid number
		// to NumberFormat, just as java.text.NumberFormat accepts it.
		if name == "numberFormat" && jvm.IsNumber(context) {
			return numberFormatHelper(context, o)
		}
		if IsEmpty(context) {
			switch name {
			case "cut", "join":
				return "", nil
			case "defaultIfEmpty":
				return o.Param(0, ""), nil
			case "numberFormat":
				if v := o.Param(0); v != nil {
					return toJavaString(v), nil
				}
				return nil, nil
			}
			if v := o.Param(0); v != nil {
				return toJavaString(v), nil
			}
			return nil, nil
		}
		s := toJavaString(context)
		switch name {
		case "capitalizeFirst":
			return commons.Capitalize(s), nil
		case "center":
			return commons.Center(s, int(jvm.IntValue(o.HashValue("size"))), toJavaString(o.HashValue("pad", " "))), nil
		case "cut":
			return strings.ReplaceAll(s, toJavaString(o.Param(0, " ")), ""), nil
		case "defaultIfEmpty":
			if IsEmpty(context) {
				return o.Param(0, ""), nil
			}
			return s, nil
		case "join":
			return joinHelper(context, o)
		case "ljust":
			return commons.RightPad(s, int(jvm.IntValue(o.HashValue("size"))), toJavaString(o.HashValue("pad", " "))), nil
		case "rjust":
			return commons.LeftPad(s, int(jvm.IntValue(o.HashValue("size"))), toJavaString(o.HashValue("pad", " "))), nil
		case "substring":
			start := int(jvm.IntValue(o.Param(0)))
			length := jvm.UTF16Len(s)
			end := length
			if len(o.Params) > 1 {
				end = int(jvm.IntValue(o.Param(1)))
			}
			if start < 0 || end < start || end > length {
				return nil, jvm.Throw("java.lang.StringIndexOutOfBoundsException", fmt.Sprintf("Range [%d, %d) out of bounds for length %d", start, end, length))
			}
			return jvm.Substring(s, start, end)
		case "lower":
			return jvm.ToLowerCase(s, jvm.Default()), nil
		case "upper":
			return jvm.ToUpperCase(s, jvm.Default()), nil
		case "slugify":
			return slugify(s), nil
		case "stringFormat":
			args := append([]any(nil), o.Params...)
			return jtext.Format(jvm.Default(), s, args)
		case "stripTags":
			return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, ""), nil
		case "capitalize":
			if full, _ := o.Hash["fully"].(bool); full {
				return commons.WordCapitalizeFully(s), nil
			}
			return commons.WordCapitalize(s), nil
		case "abbreviate":
			return commons.Abbreviate(s, int(jvm.IntValue(o.Param(0))))
		case "wordWrap":
			width := int(jvm.IntValue(o.Param(0, int32(20))))
			return commons.WordWrap(s, width, "\n", false), nil
		case "replace":
			return strings.ReplaceAll(s, toJavaString(o.Param(0)), toJavaString(o.Param(1))), nil
		case "dateFormat":
			return dateFormatHelper(context, o)
		case "numberFormat":
			return numberFormatHelper(context, o)
		case "now":
			return dateFormatHelper(jtime.InstantFromTime(time.Now()), o)
		default:
			return s, nil
		}
	}
}

func dateFormatHelper(value any, o *Options) (any, error) {
	if _, ok := value.(jvm.Object); !ok {
		class := "null"
		if value != nil {
			class = jvm.SimpleName(value)
		}
		return nil, jvm.Throw("java.lang.IllegalArgumentException", "found instance of "+class+" with value '"+toJavaString(value)+"', but expected instance of TemporalAccessor")
	}
	pattern := toJavaString(o.Param(0, o.HashValue("format", "medium")))
	style := jtime.FormatStyle(strings.ToLower(pattern))
	timeName := toJavaString(o.HashValue("time", ""))
	var formatter *jtime.Formatter
	var err error
	switch style {
	case jtime.FullStyle, jtime.LongStyle, jtime.MediumStyle, jtime.ShortStyle:
		timeStyle := jtime.FormatStyle(strings.ToLower(timeName))
		if timeStyle == jtime.FullStyle || timeStyle == jtime.LongStyle || timeStyle == jtime.MediumStyle || timeStyle == jtime.ShortStyle {
			formatter, err = jtime.OfLocalizedDateTime(style, timeStyle)
		} else {
			formatter, err = jtime.OfLocalizedDate(style)
		}
	default:
		formatter, err = jtime.OfPattern(pattern)
	}
	if err != nil {
		return nil, err
	}
	loc := jvm.Default()
	localeText := toJavaString(o.Param(1, o.HashValue("locale", "")))
	if localeText != "" {
		localeText = strings.ReplaceAll(localeText, "-", "_")
		loc, err = commons.LocaleToLocale(localeText)
		if err != nil {
			return nil, err
		}
	}
	formatter = formatter.WithLocale(loc)
	zone := jtime.SystemDefaultZone()
	if raw, ok := o.Hash["tz"]; ok {
		zone = jtime.ZoneFromTimeZoneID(toJavaString(raw))
	}
	formatter = formatter.WithZone(zone)
	return formatter.Format(value)
}
func joinHelper(context any, o *Options) (any, error) {
	values, ok := elements(context)
	params := o.Params
	if !ok {
		values = append([]any{context}, params[:max(0, len(params)-1)]...)
	}
	sep := ""
	if len(params) > 0 {
		sep = toJavaString(params[len(params)-1])
	}
	return toJavaString(o.HashValue("prefix", "")) + commons.Join(values, sep) + toJavaString(o.HashValue("suffix", "")), nil
}
func slugify(s string) string {
	var out strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsLetter(r) {
			out.WriteRune(unicode.ToLower(r))
		}
		if unicode.IsSpace(r) {
			out.WriteByte('-')
		}
	}
	return out.String()
}
func numberFormatHelper(value any, o *Options) (any, error) {
	if !jvm.IsNumber(value) {
		if v := o.Param(0); v != nil {
			return toJavaString(v), nil
		}
		return nil, nil
	}
	rawStyle := strings.TrimSpace(toJavaString(o.Param(0, o.HashValue("format", "default"))))
	style := jvm.ToUpperCase(rawStyle, jvm.Default())
	loc := jvm.Default()
	if len(o.Params) > 1 {
		parsed, err := commons.LocaleToLocale(strings.ReplaceAll(toJavaString(o.Param(1)), "-", "_"))
		if err != nil {
			return nil, err
		}
		loc = parsed
	}
	var f *jtext.NumberFormat
	var err error
	switch style {
	case "DEFAULT", "":
		f = jtext.NewNumberInstance(loc)
	case "INTEGER":
		f = jtext.NewIntegerInstance(loc)
	case "CURRENCY":
		f = jtext.NewCurrencyInstance(loc)
	case "PERCENT":
		f = jtext.NewPercentInstance(loc)
	default:
		f, err = jtext.NewDecimalFormat(style, loc)
	}
	if err != nil {
		return nil, err
	}
	if v, ok := o.Hash["groupingUsed"].(bool); ok {
		f.SetGroupingUsed(v)
	}
	if v, ok := intHash(o.Hash, "maximumFractionDigits"); ok {
		f.SetMaximumFractionDigits(v)
	}
	if v, ok := intHash(o.Hash, "minimumFractionDigits"); ok {
		f.SetMinimumFractionDigits(v)
	}
	if v, ok := intHash(o.Hash, "maximumIntegerDigits"); ok {
		f.SetMaximumIntegerDigits(v)
	}
	if v, ok := intHash(o.Hash, "minimumIntegerDigits"); ok {
		f.SetMinimumIntegerDigits(v)
	}
	if v, ok := o.Hash["parseIntegerOnly"].(bool); ok {
		f.SetParseIntegerOnly(v)
	}
	if mode, ok := o.Hash["roundingMode"]; ok {
		if err := f.SetRoundingMode(strings.ToUpper(toJavaString(mode))); err != nil {
			return nil, err
		}
	}
	return f.Format(value)
}
func intHash(hash map[string]any, key string) (int, bool) {
	v, ok := hash[key]
	if !ok {
		return 0, false
	}
	return int(jvm.IntValue(v)), true
}

// ParseNumberStyle recognizes StringHelpers.NumberStyle names.
func ParseNumberStyle(s string) string {
	return jvm.ToUpperCase(strings.TrimSpace(s), jvm.Default())
}

var _ = strconv.IntSize
