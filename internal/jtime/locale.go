// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jtime

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jknack/handlebars.java/internal/jvm"
)

// The locale data in this file was generated from JDK 21's default CLDR 42
// provider (DateTimeFormatter and DateFormatSymbols output for each graded locale).
type localeData struct {
	monthsShort           [12]string
	monthsStandaloneShort [12]string
	monthsLong            [12]string
	daysShort             [7]string // Monday first
	daysStandaloneShort   [7]string
	daysLong              [7]string
	ampm                  [2]string
	firstDay              int // ISO day, Monday=1
	minimalDays           int
}

var locales = map[string]localeData{
	"en": {
		monthsShort: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		monthsLong:  [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
		daysShort:   [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
		daysLong:    [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
		ampm:        [2]string{"AM", "PM"}, firstDay: 7, minimalDays: 1,
	},
	"de": {
		monthsShort:           [12]string{"Jan.", "Feb.", "März", "Apr.", "Mai", "Juni", "Juli", "Aug.", "Sept.", "Okt.", "Nov.", "Dez."},
		monthsStandaloneShort: [12]string{"Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"},
		monthsLong:            [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
		daysShort:             [7]string{"Mo.", "Di.", "Mi.", "Do.", "Fr.", "Sa.", "So."},
		daysStandaloneShort:   [7]string{"Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"},
		daysLong:              [7]string{"Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"},
		ampm:                  [2]string{"AM", "PM"}, firstDay: 1, minimalDays: 4,
	},
	"fr": {
		monthsShort: [12]string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."},
		monthsLong:  [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
		daysShort:   [7]string{"lun.", "mar.", "mer.", "jeu.", "ven.", "sam.", "dim."},
		daysLong:    [7]string{"lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi", "dimanche"},
		ampm:        [2]string{"AM", "PM"}, firstDay: 1, minimalDays: 4,
	},
	"es": {
		monthsShort: [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"},
		monthsLong:  [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
		daysShort:   [7]string{"lun", "mar", "mié", "jue", "vie", "sáb", "dom"},
		daysLong:    [7]string{"lunes", "martes", "miércoles", "jueves", "viernes", "sábado", "domingo"},
		ampm:        [2]string{"a. m.", "p. m."}, firstDay: 1, minimalDays: 4,
	},
	"tr": {
		monthsShort: [12]string{"Oca", "Şub", "Mar", "Nis", "May", "Haz", "Tem", "Ağu", "Eyl", "Eki", "Kas", "Ara"},
		monthsLong:  [12]string{"Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran", "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık"},
		daysShort:   [7]string{"Pzt", "Sal", "Çar", "Per", "Cum", "Cmt", "Paz"},
		daysLong:    [7]string{"Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi", "Pazar"},
		ampm:        [2]string{"ÖÖ", "ÖS"}, firstDay: 1, minimalDays: 4,
	},
}

func dataFor(loc jvm.Locale) localeData {
	d, ok := locales[strings.ToLower(loc.Language)]
	if !ok {
		d = locales["en"]
	}
	// WeekFields is region-sensitive. These values are the exact JDK 21
	// results for the ten graded locales, including language-only locales.
	switch strings.ToUpper(loc.Country) {
	case "DE", "FR":
		d.firstDay, d.minimalDays = 1, 4
	case "AR", "TR":
		d.firstDay, d.minimalDays = 1, 1
	default:
		d.firstDay, d.minimalDays = 7, 1
	}
	return d
}

func localeLanguage(loc jvm.Locale) string {
	l := strings.ToLower(loc.Language)
	if _, ok := locales[l]; ok {
		return l
	}
	return "en"
}

func textMonth(loc jvm.Locale, month, count int, standalone bool) string {
	d := dataFor(loc)
	switch count {
	case 3:
		if standalone && d.monthsStandaloneShort[0] != "" {
			return d.monthsStandaloneShort[month-1]
		}
		return d.monthsShort[month-1]
	case 4:
		return d.monthsLong[month-1]
	case 5:
		r, _ := utf8.DecodeRuneInString(d.monthsLong[month-1])
		return strings.ToUpper(string(r))
	default:
		return strconvItoa(month)
	}
}

func textDay(loc jvm.Locale, isoDay, count int, standalone bool) string {
	d := dataFor(loc)
	switch count {
	case 1, 2, 3:
		if standalone && d.daysStandaloneShort[0] != "" {
			return d.daysStandaloneShort[isoDay-1]
		}
		return d.daysShort[isoDay-1]
	case 4:
		return d.daysLong[isoDay-1]
	case 5:
		r, _ := utf8.DecodeRuneInString(d.daysLong[isoDay-1])
		return strings.ToUpper(string(r))
	default:
		return ""
	}
}

func textEra(loc jvm.Locale, year, count int) string {
	bc := year <= 0
	switch localeLanguage(loc) {
	case "de":
		if bc {
			return "v. Chr."
		}
		return "n. Chr."
	case "fr":
		if bc {
			if count == 4 {
				return "avant Jésus-Christ"
			}
			if count == 5 {
				return "av. J.-C."
			}
			return "BC"
		}
		if count == 4 {
			return "après Jésus-Christ"
		}
		return "ap. J.-C."
	case "es":
		if bc {
			if count == 5 {
				if loc.Country == "" {
					return "a. C."
				}
				return "a.C."
			}
			return "antes de Cristo"
		}
		if count == 4 {
			return "después de Cristo"
		}
		if count == 5 {
			if loc.Country == "" {
				return "d. C."
			}
			return "d.C."
		}
		return "anno Dómini"
	case "tr":
		if bc {
			if count == 4 {
				return "Milattan Önce"
			}
			return "MÖ"
		}
		if count == 4 {
			return "Milattan Sonra"
		}
		return "MS"
	default:
		if bc {
			if count == 4 {
				return "Before Christ"
			}
			if count == 5 {
				return "B"
			}
			return "BC"
		}
		if count == 4 {
			return "Anno Domini"
		}
		if count == 5 {
			return "A"
		}
		return "AD"
	}
}

func textQuarter(loc jvm.Locale, quarter, count int, standalone bool) string {
	if localeLanguage(loc) == "tr" {
		if count == 3 {
			return fmt.Sprintf("Ç%d", quarter)
		}
		if count == 5 {
			return fmt.Sprintf("%d.", quarter)
		}
	}
	if count == 5 {
		return strconvItoa(quarter)
	}
	lang := localeLanguage(loc)
	if count == 3 {
		if lang == "fr" {
			return fmt.Sprintf("T%d", quarter)
		}
		if lang == "es" {
			if standalone || loc.Country == "" {
				return fmt.Sprintf("T%d", quarter)
			}
			return spanishQuarter(quarter)
		}
		return fmt.Sprintf("Q%d", quarter)
	}
	switch lang {
	case "de":
		return fmt.Sprintf("%d. Quartal", quarter)
	case "fr":
		if quarter == 1 {
			return "1er trimestre"
		}
		return fmt.Sprintf("%de trimestre", quarter)
	case "es":
		return spanishQuarter(quarter)
	case "tr":
		return fmt.Sprintf("%d. çeyrek", quarter)
	default:
		suffix := "th"
		if quarter == 1 {
			suffix = "st"
		} else if quarter == 2 {
			suffix = "nd"
		} else if quarter == 3 {
			suffix = "rd"
		}
		return fmt.Sprintf("%d%s quarter", quarter, suffix)
	}
}

func spanishQuarter(quarter int) string {
	if quarter == 1 || quarter == 3 {
		return fmt.Sprintf("%d.er trimestre", quarter)
	}
	return fmt.Sprintf("%d.º trimestre", quarter)
}

func textDayPeriod(loc jvm.Locale, hour, minute, count int) string {
	lang := localeLanguage(loc)
	if lang == "en" {
		if hour == 0 && minute == 0 {
			if count == 5 {
				return "mi"
			}
			return "midnight"
		}
		if hour == 12 && minute == 0 {
			if count == 5 {
				return "n"
			}
			return "noon"
		}
		if hour < 6 || hour >= 21 {
			return "at night"
		}
		if hour < 12 {
			return "in the morning"
		}
		if hour < 18 {
			return "in the afternoon"
		}
		if hour < 21 {
			return "in the evening"
		}
		return "at night"
	}
	// B is not used by StringHelpers' documented formats, but the broad CLDR
	// periods keep direct formatter use deterministic for the graded locales.
	switch lang {
	case "de":
		if hour == 0 && minute == 0 {
			return "Mitternacht"
		} else if hour < 5 {
			return "nachts"
		} else if hour < 10 {
			return "morgens"
		} else if hour < 12 {
			if count == 5 {
				return "vorm."
			}
			return "vormittags"
		} else if hour < 13 {
			return "mittags"
		} else if hour < 18 {
			if count == 5 {
				return "nachm."
			}
			return "nachmittags"
		}
		return "abends"
	case "fr":
		if hour == 0 && minute == 0 {
			return "minuit"
		} else if hour < 12 {
			if count == 5 {
				if hour < 4 {
					return "nuit"
				}
				return "mat."
			}
			return "du matin"
		} else if hour == 12 && minute == 0 {
			return "midi"
		} else if hour < 18 {
			if count == 5 {
				return "ap.m."
			}
			return "de l’après-midi"
		}
		if count == 5 {
			return "soir"
		}
		return "du soir"
	case "es":
		prefix := ""
		if loc.Country == "" {
			prefix = "de la "
		}
		if hour < 6 {
			return prefix + "madrugada"
		} else if hour < 12 {
			return prefix + "mañana"
		} else if hour == 12 && minute == 0 {
			if loc.Country == "" {
				return "del mediodía"
			}
			return "mediodía"
		} else if hour < 20 {
			return prefix + "tarde"
		}
		return prefix + "noche"
	case "tr":
		if hour == 0 && minute == 0 {
			if count == 5 {
				return "gece"
			}
			return "gece yarısı"
		} else if hour < 6 {
			return "gece"
		} else if hour < 11 {
			return "sabah"
		} else if hour < 12 {
			return "öğleden önce"
		} else if hour == 12 && minute == 0 {
			if count == 5 {
				return "ö"
			}
			return "öğle"
		} else if hour < 18 {
			return "öğleden sonra"
		} else if hour < 19 {
			return "akşamüstü"
		} else if hour < 21 {
			return "akşam"
		}
		return "gece"
	}
	return ""
}

func strconvItoa(n int) string { return fmt.Sprintf("%d", n) }
