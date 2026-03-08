package i18n

import (
	"context"
	"net/http"
	"path"
	"strings"
)

func Middleware(translator *Translator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isUnlocalizedPassThrough(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			locale, basePath, ok := extractLocaleFromPath(r.URL.Path)
			if !ok {
				targetLocale := ResolveFromAcceptLanguage(r.Header.Get("Accept-Language"))
				redirectToLocale(w, r, targetLocale)
				return
			}

			addVary(w.Header(), "Accept-Language")
			w.Header().Set("Content-Language", string(locale))

			ctx := r.Context()
			ctx = WithLocale(ctx, locale)
			ctx = WithBasePath(ctx, basePath)
			ctx = WithTranslator(ctx, translator)

			r2 := cloneRequestWithPath(r, ctx, basePath)
			next.ServeHTTP(w, r2)
		})
	}
}

func extractLocaleFromPath(rawPath string) (Locale, string, bool) {
	cleaned := path.Clean(rawPath)
	if cleaned == "." {
		cleaned = "/"
	}
	trimmed := strings.TrimPrefix(cleaned, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", false
	}
	locale, ok := ParseLocale(parts[0])
	if !ok {
		return "", "", false
	}
	base := "/"
	if len(parts) > 1 {
		base = "/" + strings.Join(parts[1:], "/")
	}
	return locale, base, true
}

func isUnlocalizedPassThrough(requestPath string) bool {
	if strings.HasPrefix(requestPath, "/static/") {
		return true
	}
	ext := path.Ext(requestPath)
	return ext != ""
}

func redirectToLocale(w http.ResponseWriter, r *http.Request, locale Locale) {
	targetPath := "/" + string(locale)
	if r.URL.Path != "/" {
		targetPath += r.URL.Path
	}
	if r.URL.RawQuery != "" {
		targetPath += "?" + r.URL.RawQuery
	}
	addVary(w.Header(), "Accept-Language")
	http.Redirect(w, r, targetPath, http.StatusPermanentRedirect)
}

func cloneRequestWithPath(r *http.Request, ctx context.Context, newPath string) *http.Request {
	r2 := r.Clone(ctx)
	r2.URL.Path = newPath
	r2.URL.RawPath = ""
	return r2
}

func addVary(header http.Header, value string) {
	current := header.Values("Vary")
	for _, item := range current {
		for _, token := range strings.Split(item, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}
