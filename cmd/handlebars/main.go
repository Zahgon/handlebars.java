// Handlebars.java: https://github.com/jknack/handlebars.java
// Apache License Version 2.0 http://www.apache.org/licenses/LICENSE-2.0
// Copyright (c) 2012 Edgar Espina

// Command handlebars renders templates through the Go Handlebars.java port.
package main

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"

	hb "github.com/jknack/handlebars.java/handlebars"
	"github.com/jknack/handlebars.java/internal/jtime"
	"github.com/jknack/handlebars.java/internal/jvm"
)

const (
	usageStatus   = 64
	dataStatus    = 65
	compileStatus = 2
	renderStatus  = 3
)

type exitError struct {
	status  int
	message string
}

func (e *exitError) Error() string { return e.message }

func main() {
	out, err := run(os.Args[1:])
	if err != nil {
		writeDiagnostic(err.message + "\n")
		os.Exit(err.status)
	}
	if _, writeErr := os.Stdout.Write(jvm.EncodeUTF8(out)); writeErr != nil {
		writeDiagnostic("handlebars: cannot write standard output: " + writeErr.Error() + "\n")
		os.Exit(renderStatus)
	}
}

// writeDiagnostic writes an error message to stderr and reports its cause via
// stdout if stderr itself is unavailable. The caller retains the reference
// command's exit status when neither stream can accept the diagnostic.
func writeDiagnostic(message string) {
	if _, err := os.Stderr.Write(jvm.EncodeUTF8(message)); err != nil {
		fallback := "handlebars: cannot write standard error: " + err.Error() + "; " + strings.TrimSuffix(message, "\n") + "\n"
		if _, fallbackErr := os.Stdout.Write(jvm.EncodeUTF8(fallback)); fallbackErr != nil {
			// Both observable streams failed. Returning lets main preserve the
			// status selected for the original command failure.
			return
		}
	}
}

func run(args []string) (string, *exitError) {
	opts := make(map[string]string)
	flags := make(map[string]bool)
	var dataOpts [][2]string
	var positional []string
	onlyPos := false
	valueOpts := map[string]bool{"suffix": true, "escape": true, "start-delimiter": true, "end-delimiter": true, "data": true, "locale": true, "timezone": true, "lambda": true, "collect": true}
	flagOpts := map[string]bool{"inline": true, "pretty-print": true, "no-parent-scope": true, "no-pre-evaluate": true, "string-params": true, "infinite-loops": true, "delete-partial-after-merge": true, "hash-maps": true, "big-decimals": true, "text": true, "collect-params": true}
	for _, arg := range args {
		if !onlyPos && arg == "--" {
			onlyPos = true
			continue
		}
		if !onlyPos && strings.HasPrefix(arg, "--") {
			raw := arg[2:]
			name, value, has := strings.Cut(raw, "=")
			if flagOpts[name] && !has {
				flags[name] = true
				continue
			}
			if !valueOpts[name] || !has {
				return "", &exitError{usageStatus, "handlebars: unknown option or missing value: " + arg}
			}
			if name == "data" {
				key, jsonText, ok := strings.Cut(value, "=")
				if !ok || key == "" {
					return "", &exitError{usageStatus, "handlebars: --data needs NAME=JSON: " + arg}
				}
				dataOpts = append(dataOpts, [2]string{key, jsonText})
			} else {
				opts[name] = value
			}
			continue
		}
		positional = append(positional, arg)
	}
	inline := flags["inline"]
	min := 2
	if inline {
		min = 1
	}
	if len(positional) < min || len(positional) > min+1 {
		return "", &exitError{usageStatus, "usage: handlebars [OPTIONS] TEMPLATE_DIR NAME [DATA] | --inline [OPTIONS] TEMPLATE_DIR [DATA]"}
	}
	dir := positional[0]
	name := ""
	if !inline {
		name = positional[1]
	}
	var dataFile string
	if len(positional) > min {
		dataFile = positional[min]
	}
	loc := opts["locale"]
	if loc == "" {
		loc = "en-US"
	}
	jvm.SetDefault(jvm.ForLanguageTag(loc))
	zone := opts["timezone"]
	if zone == "" {
		zone = "UTC"
	}
	jtime.SetSystemDefaultZone(jtime.ZoneFromTimeZoneID(zone))
	var model any
	if dataFile == "" {
		if flags["hash-maps"] {
			model = jvm.NewHashMap()
		} else {
			model = jvm.NewLinkedHashMap()
		}
	} else {
		b, err := os.ReadFile(dataFile)
		if err != nil {
			return "", &exitError{dataStatus, "handlebars: cannot read " + dataFile + ": " + javaFileError(dataFile, err)}
		}
		var perr *exitError
		model, perr = parseJSON(jvm.DecodeUTF8(b), dataFile, flags["hash-maps"], flags["big-decimals"])
		if perr != nil {
			return "", perr
		}
	}
	data := make([][2]any, 0, len(dataOpts))
	for _, d := range dataOpts {
		v, e := parseJSON(d[1], "--data="+d[0], flags["hash-maps"], flags["big-decimals"])
		if e != nil {
			return "", e
		}
		data = append(data, [2]any{d[0], v})
	}
	if number, ok := opts["lambda"]; ok {
		lambda := lambdaOf(number)
		if lambda == nil {
			return "", &exitError{usageStatus, "handlebars: --lambda needs a number from 0 to 9"}
		}
		if m, ok := model.(*jvm.Map); ok {
			m.Put("lambda", lambda)
		}
	}
	inlineText := ""
	if inline {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", &exitError{dataStatus, "handlebars: cannot read standard input: " + err.Error()}
		}
		inlineText = jvm.DecodeUTF8(b)
	}
	var collectTypes []hb.TagType
	if collect, ok := opts["collect"]; ok {
		var e *exitError
		collectTypes, e = parseTags(collect)
		if e != nil {
			return "", e
		}
	}
	suffix := opts["suffix"]
	if _, ok := opts["suffix"]; !ok {
		suffix = ".hbs"
	}
	engine := hb.New(hb.NewFileTemplateLoaderWithSuffix(dir, suffix)).RegisterStringHelpers().RegisterConditionalHelpers()
	if e := configure(engine, opts, flags); e != nil {
		return "", e
	}
	engine.RegisterHelper("date", func(context any, _ *hb.Options) (any, error) { return jtime.ParseISO(jvm.ToString(context)) })
	engine.RegisterHelper("rep", func(context any, o *hb.Options) (any, error) {
		raw := o.Param(0)
		if raw == nil {
			return nil, jvm.Throw("java.lang.IllegalArgumentException", `java.lang.NullPointerException: Cannot invoke "java.lang.Number.intValue()" because the return value of "sun.invoke.util.ValueConversions.primitiveConversion(sun.invoke.util.Wrapper, Object, boolean)" is null`)
		}
		iv, ok := raw.(int32)
		if !ok {
			return nil, jvm.Throw("java.lang.IllegalArgumentException", "argument type mismatch")
		}
		times := int(iv)
		if times < 0 {
			return nil, jvm.Throw("java.lang.IllegalArgumentException", "count is negative: "+strconv.Itoa(times))
		}
		return strings.Repeat(jvm.ToString(context), times), nil
	})
	engine.RegisterHelper("pair", func(context any, o *hb.Options) (any, error) {
		return jvm.ToString(context) + "+" + jvm.ToString(o.Param(0)), nil
	})
	engine.RegisterHelper("describe", func(context any, o *hb.Options) (any, error) {
		typ := "null"
		if context != nil {
			typ = jvm.SimpleName(context)
		}
		return typ + ":" + jvm.ToString(o.HashValue("tag", "none")) + ":" + strconv.Itoa(len(o.Params)), nil
	})
	engine.RegisterDecorator("set", func(t hb.Template, o *hb.Options) (hb.Template, error) {
		key := jvm.ToString(o.Param(0))
		value := o.Param(1)
		if len(o.Params) < 2 && o.Fn != nil {
			body, err := o.ApplyFn(o.Context)
			if err != nil {
				return nil, err
			}
			value = body
		}
		o.SetData(key, value)
		return t, nil
	})
	engine.RegisterDecorator("trace", func(t hb.Template, o *hb.Options) (hb.Template, error) {
		list, ok := o.Data("trace").(*jvm.List)
		if !ok {
			list = &jvm.List{}
			o.SetData("trace", list)
		}
		list.Elems = append(list.Elems, o.Param(0))
		return t, nil
	})
	var template hb.Template
	var err error
	if inline {
		template, err = engine.CompileInline(inlineText)
	} else {
		template, err = engine.Compile(name)
	}
	if err != nil {
		return "", &exitError{compileStatus, message(err)}
	}
	var out strings.Builder
	if flags["text"] {
		out.WriteString(template.Text())
		out.WriteByte('\n')
	}
	if _, ok := opts["collect"]; ok {
		items := template.Collect(collectTypes...)
		out.WriteString("[")
		out.WriteString(strings.Join(items, ", "))
		out.WriteString("]\n")
	}
	if flags["collect-params"] {
		items := template.CollectReferenceParameters()
		out.WriteString("[")
		out.WriteString(strings.Join(items, ", "))
		out.WriteString("]\n")
	}
	context := hb.NewContext(model)
	for _, d := range data {
		context.SetData(d[0].(string), d[1])
	}
	rendered, err := template.ApplyContext(context)
	if err != nil {
		return "", &exitError{renderStatus, message(err)}
	}
	out.WriteString(rendered)
	return out.String(), nil
}

func javaFileError(name string, err error) string {
	if os.IsNotExist(err) {
		return "java.nio.file.NoSuchFileException: " + name
	}
	if os.IsPermission(err) {
		return "java.nio.file.AccessDeniedException: " + name
	}
	return err.Error()
}

func configure(h *hb.Handlebars, opts map[string]string, flags map[string]bool) *exitError {
	if names, ok := opts["escape"]; ok {
		var strategies []hb.EscapingStrategy
		for _, name := range strings.Split(names, ",") {
			switch name {
			case "html", "hbs4":
				strategies = append(strategies, hb.HTMLENTITY)
			case "hbs3":
				strategies = append(strategies, hb.HBS3)
			case "js":
				strategies = append(strategies, hb.JS)
			case "xml":
				strategies = append(strategies, hb.XML)
			case "csv":
				strategies = append(strategies, hb.CSV)
			case "noop":
				strategies = append(strategies, hb.NOOP)
			default:
				return &exitError{usageStatus, "handlebars: unknown escaping strategy: " + name}
			}
		}
		h.WithEscapingStrategy(strategies...)
	}
	h.PrettyPrint(flags["pretty-print"]).ParentScopeResolution(!flags["no-parent-scope"]).PreEvaluatePartialBlocks(!flags["no-pre-evaluate"]).StringParams(flags["string-params"]).InfiniteLoops(flags["infinite-loops"]).DeletePartialAfterMerge(flags["delete-partial-after-merge"])
	if v, ok := opts["start-delimiter"]; ok {
		h.StartDelimiter(v)
	}
	if v, ok := opts["end-delimiter"]; ok {
		h.EndDelimiter(v)
	}
	return nil
}
func message(err error) string { return err.Error() }
func parseTags(input string) ([]hb.TagType, *exitError) {
	names := strings.Split(input, ",")
	out := make([]hb.TagType, len(names))
	values := map[string]hb.TagType{"VAR": hb.TagVar, "STAR_VAR": hb.TagStarVar, "AMP_VAR": hb.TagAmpVar, "TRIPLE_VAR": hb.TagTripleVar, "SUB_EXPRESSION": hb.TagSubExpression, "SECTION": hb.TagSection, "START_SECTION": hb.TagStartSection}
	for i, n := range names {
		v, ok := values[n]
		if !ok {
			return nil, &exitError{usageStatus, "handlebars: unknown tag type: " + n}
		}
		out[i] = v
	}
	return out, nil
}

// lambdaOf returns the Mustache spec lambda that the --lambda option binds to the root key "lambda". The reference
// command's contract fixes these ten lambdas by number: they are the lambdas of the original's
// mustache/specs/LambdasTest.java, each with its own return value and use of the section text (number 3 counts its
// calls). Any other number has no lambda, which the command reports as a usage error.
func lambdaOf(number string) hb.Lambda {
	calls := int32(0)
	switch number {
	case "0":
		return func(any, hb.Template) (any, error) { return "world", nil }
	case "1":
		return func(any, hb.Template) (any, error) { return "{{planet}}", nil }
	case "2":
		return func(any, hb.Template) (any, error) { return "|planet| => {{planet}}", nil }
	case "3":
		return func(any, hb.Template) (any, error) { calls++; return strconv.Itoa(int(calls)), nil }
	case "4":
		return func(any, hb.Template) (any, error) { return ">", nil }
	case "5":
		return func(_ any, t hb.Template) (any, error) {
			if t.Text() == "{{x}}" {
				return "yes", nil
			}
			return "no", nil
		}
	case "6":
		return func(_ any, t hb.Template) (any, error) { return t.Text() + "{{planet}}" + t.Text(), nil }
	case "7":
		return func(_ any, t hb.Template) (any, error) { return t.Text() + "{{planet}} => |planet|" + t.Text(), nil }
	case "8":
		return func(_ any, t hb.Template) (any, error) { return "__" + t.Text() + "__", nil }
	case "9":
		return func(any, hb.Template) (any, error) { return false, nil }
	}
	return nil
}

type jsonReader struct {
	text             string
	pos              int
	hash, bigDecimal bool
}

func parseJSON(text, source string, hash, bigDecimal bool) (any, *exitError) {
	j := &jsonReader{text: text, hash: hash, bigDecimal: bigDecimal}
	j.space()
	v, err := j.value()
	if err == nil {
		j.space()
		if j.pos < len(j.text) {
			err = j.fail("unexpected character")
		}
	}
	if err != nil {
		return nil, &exitError{dataStatus, "handlebars: invalid JSON in " + source + ": " + err.Error()}
	}
	return v, nil
}
func (j *jsonReader) fail(what string) error {
	return fmt.Errorf("%s at offset %d", what, jvm.UTF16Len(j.text[:j.pos]))
}
func (j *jsonReader) space() {
	for j.pos < len(j.text) && strings.ContainsRune(" \t\n\r", rune(j.text[j.pos])) {
		j.pos++
	}
}
func (j *jsonReader) value() (any, error) {
	if j.pos >= len(j.text) {
		return nil, j.fail("unexpected end of input")
	}
	switch j.text[j.pos] {
	case '{':
		return j.object()
	case '[':
		return j.array()
	case '"':
		return j.string()
	case 't':
		return j.literal("true", true)
	case 'f':
		return j.literal("false", false)
	case 'n':
		return j.literal("null", nil)
	default:
		if j.text[j.pos] == '-' || (j.text[j.pos] >= '0' && j.text[j.pos] <= '9') {
			return j.number()
		}
		return nil, j.fail("unexpected character")
	}
}
func (j *jsonReader) literal(word string, v any) (any, error) {
	if !strings.HasPrefix(j.text[j.pos:], word) {
		return nil, j.fail("unexpected character")
	}
	j.pos += len(word)
	return v, nil
}
func (j *jsonReader) object() (any, error) {
	var m *jvm.Map
	if j.hash {
		m = jvm.NewHashMap()
	} else {
		m = jvm.NewLinkedHashMap()
	}
	j.pos++
	j.space()
	if j.pos < len(j.text) && j.text[j.pos] == '}' {
		j.pos++
		return m, nil
	}
	for {
		j.space()
		if j.pos >= len(j.text) || j.text[j.pos] != '"' {
			return nil, j.fail("expected a string key")
		}
		key, err := j.string()
		if err != nil {
			return nil, err
		}
		j.space()
		if err = j.expect(':'); err != nil {
			return nil, err
		}
		j.space()
		v, err := j.value()
		if err != nil {
			return nil, err
		}
		m.Put(key, v)
		j.space()
		if j.pos < len(j.text) && j.text[j.pos] == ',' {
			j.pos++
			continue
		}
		if err = j.expect('}'); err != nil {
			return nil, err
		}
		return m, nil
	}
}
func (j *jsonReader) array() (any, error) {
	l := &jvm.List{}
	j.pos++
	j.space()
	if j.pos < len(j.text) && j.text[j.pos] == ']' {
		j.pos++
		return l, nil
	}
	for {
		j.space()
		v, err := j.value()
		if err != nil {
			return nil, err
		}
		l.Elems = append(l.Elems, v)
		j.space()
		if j.pos < len(j.text) && j.text[j.pos] == ',' {
			j.pos++
			continue
		}
		if err = j.expect(']'); err != nil {
			return nil, err
		}
		return l, nil
	}
}
func (j *jsonReader) expect(c byte) error {
	if j.pos >= len(j.text) || j.text[j.pos] != c {
		return j.fail("expected '" + string(c) + "'")
	}
	j.pos++
	return nil
}
func (j *jsonReader) string() (string, error) {
	var b strings.Builder
	j.pos++
	for {
		if j.pos >= len(j.text) {
			return "", j.fail("unterminated string")
		}
		c := j.text[j.pos]
		j.pos++
		if c == '"' {
			return b.String(), nil
		}
		if c < 0x20 {
			return "", j.fail("control character in string")
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if j.pos >= len(j.text) {
			return "", j.fail("unterminated string")
		}
		e := j.text[j.pos]
		j.pos++
		switch e {
		case '"', '\\', '/':
			b.WriteByte(e)
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'u':
			if j.pos+4 > len(j.text) {
				return "", j.fail("bad unicode escape")
			}
			code := uint16(0)
			for k := 0; k < 4; k++ {
				d := hex(j.text[j.pos])
				j.pos++
				if d < 0 {
					return "", j.fail("bad unicode escape")
				}
				code = code*16 + uint16(d)
			}
			if code >= 0xd800 && code <= 0xdbff && j.pos+6 <= len(j.text) && j.text[j.pos] == '\\' && j.text[j.pos+1] == 'u' {
				low := uint16(0)
				valid := true
				for k := 0; k < 4; k++ {
					d := hex(j.text[j.pos+2+k])
					if d < 0 {
						valid = false
						break
					}
					low = low*16 + uint16(d)
				}
				if valid && low >= 0xdc00 && low <= 0xdfff {
					j.pos += 6
					b.WriteRune(rune(0x10000 + (uint32(code)-0xd800)*0x400 + uint32(low) - 0xdc00))
					break
				}
			}
			writeUTF16Unit(&b, code)
		default:
			return "", j.fail("bad escape")
		}
	}
}
func hex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
func writeUTF16Unit(b *strings.Builder, u uint16) {
	if u < 0x80 {
		b.WriteByte(byte(u))
	} else if u < 0x800 {
		b.WriteByte(0xc0 | byte(u>>6))
		b.WriteByte(0x80 | byte(u&63))
	} else {
		b.WriteByte(0xe0 | byte(u>>12))
		b.WriteByte(0x80 | byte((u>>6)&63))
		b.WriteByte(0x80 | byte(u&63))
	}
}
func (j *jsonReader) number() (any, error) {
	start := j.pos
	if j.text[j.pos] == '-' {
		j.pos++
	}
	if j.pos >= len(j.text) {
		return nil, j.fail("bad number")
	}
	if j.text[j.pos] == '0' {
		j.pos++
	} else if j.text[j.pos] >= '1' && j.text[j.pos] <= '9' {
		j.digits()
	} else {
		return nil, j.fail("bad number")
	}
	whole := true
	if j.pos < len(j.text) && j.text[j.pos] == '.' {
		whole = false
		j.pos++
		if j.digits() == 0 {
			return nil, j.fail("bad number")
		}
	}
	if j.pos < len(j.text) && (j.text[j.pos] == 'e' || j.text[j.pos] == 'E') {
		whole = false
		j.pos++
		if j.pos < len(j.text) && (j.text[j.pos] == '+' || j.text[j.pos] == '-') {
			j.pos++
		}
		if j.digits() == 0 {
			return nil, j.fail("bad number")
		}
	}
	literal := j.text[start:j.pos]
	if !whole {
		if j.bigDecimal {
			v, err := jvm.NewBigDecimal(literal)
			return v, err
		}
		return jvm.ParseDouble(literal)
	}
	if v, err := jvm.ParseInt(literal); err == nil {
		return v, nil
	}
	if v, err := strconv.ParseInt(literal, 10, 64); err == nil {
		return v, nil
	}
	v, ok := new(big.Int).SetString(literal, 10)
	if !ok {
		return nil, j.fail("bad number")
	}
	return v, nil
}
func (j *jsonReader) digits() int {
	start := j.pos
	for j.pos < len(j.text) && j.text[j.pos] >= '0' && j.text[j.pos] <= '9' {
		j.pos++
	}
	return j.pos - start
}
