// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package commons

import (
	"sort"
	"strconv"
	"strings"
)

// EscapeEcmaScript returns StringEscapeUtils.escapeEcmaScript.
func EscapeEcmaScript(s string) string {
	var b strings.Builder
	hex := "0123456789ABCDEF"
	for _, c := range utf16Units(s) {
		switch c {
		case '\'':
			b.WriteString("\\'")
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		case '/':
			b.WriteString("\\/")
		case '\b':
			b.WriteString("\\b")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '\f':
			b.WriteString("\\f")
		case '\r':
			b.WriteString("\\r")
		default:
			if c < 0x20 || c > 0x7f {
				b.WriteString("\\u")
				b.WriteByte(hex[c>>12])
				b.WriteByte(hex[c>>8&15])
				b.WriteByte(hex[c>>4&15])
				b.WriteByte(hex[c&15])
			} else {
				b.WriteByte(byte(c))
			}
		}
	}
	return b.String()
}

// EscapeXml11 returns StringEscapeUtils.escapeXml11.
func EscapeXml11(s string) string {
	u := utf16Units(s)
	var b strings.Builder
	for i := 0; i < len(u); i++ {
		c := u[i]
		switch c {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			if c == 0 || c == 0xfffe || c == 0xffff || (c >= 0xd800 && c <= 0xdfff) {
				if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
					b.WriteString(unitsString(u[i : i+2]))
					i++
				}
				continue
			}
			if (c >= 1 && c <= 8) || c == 0xb || c == 0xc || (c >= 0xe && c <= 0x1f) || (c >= 0x7f && c <= 0x84) || (c >= 0x86 && c <= 0x9f) {
				b.WriteString("&#")
				b.WriteString(strconv.Itoa(int(c)))
				b.WriteByte(';')
			} else {
				b.WriteString(unitsString([]uint16{c}))
			}
		}
	}
	return b.String()
}

// EscapeCsv returns StringEscapeUtils.escapeCsv.
func EscapeCsv(s string) string {
	if !strings.ContainsAny(s, ",\n\r\"") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// EscapeHTML4 returns StringEscapeUtils.escapeHtml4.
func EscapeHTML4(s string) string {
	var b strings.Builder
	u := utf16Units(s)
	for i, c := range u {
		if c >= 0xdc00 && c <= 0xdfff && i > 0 && u[i-1] >= 0xd800 && u[i-1] <= 0xdbff {
			continue
		}
		if c >= 0xd800 && c <= 0xdbff && i+1 < len(u) && u[i+1] >= 0xdc00 && u[i+1] <= 0xdfff {
			b.WriteString(unitsString(u[i : i+2]))
			continue
		}
		j := sort.Search(len(html4Codepoints), func(j int) bool { return html4Codepoints[j] >= c })
		if j < len(html4Codepoints) && html4Codepoints[j] == c {
			b.WriteByte('&')
			b.WriteString(html4Names[j])
			b.WriteByte(';')
		} else {
			b.WriteString(unitsString([]uint16{c}))
		}
	}
	return b.String()
}

var html4Codepoints = []uint16{
	0x22, 0x26, 0x3c, 0x3e, 0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xbd, 0xbe, 0xbf, 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf, 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf, 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef, 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff, 0x152, 0x153, 0x160, 0x161, 0x178, 0x192, 0x2c6, 0x2dc,
	0x391, 0x392, 0x393, 0x394, 0x395, 0x396, 0x397, 0x398, 0x399, 0x39a, 0x39b, 0x39c, 0x39d, 0x39e, 0x39f, 0x3a0, 0x3a1, 0x3a3, 0x3a4, 0x3a5, 0x3a6, 0x3a7, 0x3a8, 0x3a9, 0x3b1, 0x3b2, 0x3b3, 0x3b4, 0x3b5, 0x3b6, 0x3b7, 0x3b8, 0x3b9, 0x3ba, 0x3bb, 0x3bc, 0x3bd, 0x3be, 0x3bf, 0x3c0, 0x3c1, 0x3c2, 0x3c3, 0x3c4, 0x3c5, 0x3c6, 0x3c7, 0x3c8, 0x3c9, 0x3d1, 0x3d2, 0x3d6,
	0x2002, 0x2003, 0x2009, 0x200c, 0x200d, 0x200e, 0x200f, 0x2013, 0x2014, 0x2018, 0x2019, 0x201a, 0x201c, 0x201d, 0x201e, 0x2020, 0x2021, 0x2022, 0x2026, 0x2030, 0x2032, 0x2033, 0x2039, 0x203a, 0x203e, 0x2044, 0x20ac, 0x2111, 0x2118, 0x211c, 0x2122, 0x2135, 0x2190, 0x2191, 0x2192, 0x2193, 0x2194, 0x21b5, 0x21d0, 0x21d1, 0x21d2, 0x21d3, 0x21d4, 0x2200, 0x2202, 0x2203, 0x2205, 0x2207, 0x2208, 0x2209, 0x220b, 0x220f, 0x2211, 0x2212, 0x2217, 0x221a, 0x221d, 0x221e, 0x2220, 0x2227, 0x2228, 0x2229, 0x222a, 0x222b, 0x2234, 0x223c, 0x2245, 0x2248, 0x2260, 0x2261, 0x2264, 0x2265, 0x2282, 0x2283, 0x2284, 0x2286, 0x2287, 0x2295, 0x2297, 0x22a5, 0x22c5, 0x2308, 0x2309, 0x230a, 0x230b, 0x2329, 0x232a, 0x25ca, 0x2660, 0x2663, 0x2665, 0x2666,
}
var html4Names = strings.Fields("quot amp lt gt nbsp iexcl cent pound curren yen brvbar sect uml copy ordf laquo not shy reg macr deg plusmn sup2 sup3 acute micro para middot cedil sup1 ordm raquo frac14 frac12 frac34 iquest Agrave Aacute Acirc Atilde Auml Aring AElig Ccedil Egrave Eacute Ecirc Euml Igrave Iacute Icirc Iuml ETH Ntilde Ograve Oacute Ocirc Otilde Ouml times Oslash Ugrave Uacute Ucirc Uuml Yacute THORN szlig agrave aacute acirc atilde auml aring aelig ccedil egrave eacute ecirc euml igrave iacute icirc iuml eth ntilde ograve oacute ocirc otilde ouml divide oslash ugrave uacute ucirc uuml yacute thorn yuml OElig oelig Scaron scaron Yuml fnof circ tilde Alpha Beta Gamma Delta Epsilon Zeta Eta Theta Iota Kappa Lambda Mu Nu Xi Omicron Pi Rho Sigma Tau Upsilon Phi Chi Psi Omega alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron pi rho sigmaf sigma tau upsilon phi chi psi omega thetasym upsih piv ensp emsp thinsp zwnj zwj lrm rlm ndash mdash lsquo rsquo sbquo ldquo rdquo bdquo dagger Dagger bull hellip permil prime Prime lsaquo rsaquo oline frasl euro image weierp real trade alefsym larr uarr rarr darr harr crarr lArr uArr rArr dArr hArr forall part exist empty nabla isin notin ni prod sum minus lowast radic prop infin ang and or cap cup int there4 sim cong asymp ne equiv le ge sub sup nsub sube supe oplus otimes perp sdot lceil rceil lfloor rfloor lang rang loz spades clubs hearts diams")
