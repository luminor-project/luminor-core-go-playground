package i18n_test

import (
	"testing"
	"time"

	"github.com/luminor-project/luminor-core-go-playground/internal/platform/i18n"
)

func TestLoadEmbeddedTranslator_AllLocalesPresent(t *testing.T) {
	translator, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, locale := range []i18n.Locale{i18n.LocaleDE, i18n.LocaleFR} {
		missing := translator.MissingKeysFor(locale)
		if len(missing) > 0 {
			t.Fatalf("missing keys for %s: %d", locale, len(missing))
		}
	}
}

func TestResolveFromAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   i18n.Locale
	}{
		{header: "de-DE,de;q=0.9,en;q=0.8", want: i18n.LocaleDE},
		{header: "fr-CH,fr;q=0.9,en;q=0.8", want: i18n.LocaleFR},
		{header: "es-ES,es;q=0.9", want: i18n.LocaleEN},
		{header: "", want: i18n.LocaleEN},
	}
	for _, tc := range tests {
		got := i18n.ResolveFromAcceptLanguage(tc.header)
		if got != tc.want {
			t.Fatalf("header=%q want=%q got=%q", tc.header, tc.want, got)
		}
	}
}

func TestPluralAndDateFormatting(t *testing.T) {
	translator, err := i18n.LoadEmbeddedTranslator()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	baseCtx := i18n.WithTranslator(i18n.WithLocale(t.Context(), i18n.LocaleFR), translator)
	if got := i18n.TPlural(baseCtx, "organization.members.count", 1); got != "1 membre" {
		t.Fatalf("unexpected singular: %q", got)
	}
	if got := i18n.TPlural(baseCtx, "organization.members.count", 3); got != "3 membres" {
		t.Fatalf("unexpected plural: %q", got)
	}
	value := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	if got := i18n.FormatDateShort(baseCtx, value); got == "" {
		t.Fatal("expected non-empty short date")
	}
}
