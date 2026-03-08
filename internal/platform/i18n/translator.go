package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"strings"
)

//go:embed locales/*.json
var localesFS embed.FS

type Translator struct {
	defaultLocale Locale
	catalogs      map[Locale]map[string]string
}

func NewTranslator(defaultLocale Locale, catalogs map[Locale]map[string]string) *Translator {
	cloned := make(map[Locale]map[string]string, len(catalogs))
	for locale, messages := range catalogs {
		cloned[locale] = maps.Clone(messages)
	}
	return &Translator{
		defaultLocale: defaultLocale,
		catalogs:      cloned,
	}
}

func LoadEmbeddedTranslator() (*Translator, error) {
	catalogs := make(map[Locale]map[string]string, len(supportedLocales))
	for _, locale := range supportedLocales {
		path := filepath.Join("locales", string(locale)+".json")
		raw, err := localesFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", locale, err)
		}
		messages := map[string]string{}
		if err := json.Unmarshal(raw, &messages); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", locale, err)
		}
		catalogs[locale] = messages
	}
	return NewTranslator(DefaultLocale(), catalogs), nil
}

func (t *Translator) Message(locale Locale, key string, args ...any) string {
	msg, ok := t.lookup(locale, key)
	if !ok {
		return key
	}
	return interpolate(msg, args...)
}

func (t *Translator) Plural(locale Locale, key string, count int, args ...any) string {
	form := "other"
	if count == 1 {
		form = "one"
	}
	merged := append([]any{"count", count}, args...)
	return t.Message(locale, key+"."+form, merged...)
}

func (t *Translator) lookup(locale Locale, key string) (string, bool) {
	if messages, ok := t.catalogs[locale]; ok {
		if msg, ok := messages[key]; ok {
			return msg, true
		}
	}
	if messages, ok := t.catalogs[t.defaultLocale]; ok {
		msg, ok := messages[key]
		return msg, ok
	}
	return "", false
}

func (t *Translator) MissingKeysFor(locale Locale) []string {
	base := t.catalogs[t.defaultLocale]
	target := t.catalogs[locale]
	var missing []string
	for key := range base {
		if _, ok := target[key]; !ok {
			missing = append(missing, key)
		}
	}
	return missing
}

func interpolate(input string, args ...any) string {
	if len(args) == 0 {
		return input
	}
	values := map[string]string{}
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		values[key] = fmt.Sprint(args[i+1])
	}
	result := input
	for key, value := range values {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
	}
	return result
}
