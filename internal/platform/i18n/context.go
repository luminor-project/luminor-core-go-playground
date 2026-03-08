package i18n

import (
	"context"
	"strings"
)

type contextKey string

const (
	localeContextKey     contextKey = "i18n_locale"
	translatorContextKey contextKey = "i18n_translator"
	basePathContextKey   contextKey = "i18n_base_path"
)

func WithLocale(ctx context.Context, locale Locale) context.Context {
	return context.WithValue(ctx, localeContextKey, locale)
}

func LocaleFromContext(ctx context.Context) Locale {
	locale, ok := ctx.Value(localeContextKey).(Locale)
	if !ok {
		return DefaultLocale()
	}
	if _, valid := ParseLocale(string(locale)); !valid {
		return DefaultLocale()
	}
	return locale
}

func WithTranslator(ctx context.Context, translator *Translator) context.Context {
	return context.WithValue(ctx, translatorContextKey, translator)
}

func TranslatorFromContext(ctx context.Context) *Translator {
	translator, _ := ctx.Value(translatorContextKey).(*Translator)
	return translator
}

func WithBasePath(ctx context.Context, path string) context.Context {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return context.WithValue(ctx, basePathContextKey, path)
}

func BasePathFromContext(ctx context.Context) string {
	path, _ := ctx.Value(basePathContextKey).(string)
	if path == "" {
		return "/"
	}
	return path
}

func T(ctx context.Context, key string, args ...any) string {
	translator := TranslatorFromContext(ctx)
	if translator == nil {
		return key
	}
	return translator.Message(LocaleFromContext(ctx), key, args...)
}

func TPlural(ctx context.Context, key string, count int, args ...any) string {
	translator := TranslatorFromContext(ctx)
	if translator == nil {
		return key
	}
	return translator.Plural(LocaleFromContext(ctx), key, count, args...)
}

func LocalizedPath(ctx context.Context, path string) string {
	locale := LocaleFromContext(ctx)
	if path == "" {
		return "/" + string(locale)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) > 0 {
		if _, ok := ParseLocale(parts[0]); ok {
			return path
		}
	}
	if path == "/" {
		return "/" + string(locale)
	}
	return "/" + string(locale) + path
}

func AlternateLocalizedPath(ctx context.Context, locale Locale) string {
	base := BasePathFromContext(ctx)
	if base == "/" {
		return "/" + string(locale)
	}
	return "/" + string(locale) + base
}
