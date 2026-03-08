package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

type Locale string

const (
	LocaleEN Locale = "en"
	LocaleDE Locale = "de"
	LocaleFR Locale = "fr"
)

var supportedLocales = []Locale{LocaleEN, LocaleDE, LocaleFR}

var (
	acceptMatcher = language.NewMatcher([]language.Tag{
		language.English,
		language.German,
		language.French,
	})
)

func SupportedLocales() []Locale {
	out := make([]Locale, len(supportedLocales))
	copy(out, supportedLocales)
	return out
}

func DefaultLocale() Locale {
	return LocaleEN
}

func ParseLocale(raw string) (Locale, bool) {
	v := Locale(strings.ToLower(strings.TrimSpace(raw)))
	switch v {
	case LocaleEN, LocaleDE, LocaleFR:
		return v, true
	default:
		return "", false
	}
}

func ResolveFromAcceptLanguage(header string) Locale {
	tags, _, err := language.ParseAcceptLanguage(header)
	if err != nil || len(tags) == 0 {
		return DefaultLocale()
	}
	tag, _, _ := acceptMatcher.Match(tags...)
	base, _ := tag.Base()
	switch base.String() {
	case "de":
		return LocaleDE
	case "fr":
		return LocaleFR
	default:
		return LocaleEN
	}
}
