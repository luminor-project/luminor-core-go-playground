package i18n

import (
	"context"
	"strconv"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	printerEN = message.NewPrinter(language.English)
	printerDE = message.NewPrinter(language.German)
	printerFR = message.NewPrinter(language.French)
)

var monthsLong = map[Locale][]string{
	LocaleEN: {"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	LocaleDE: {"Januar", "Februar", "Marz", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"},
	LocaleFR: {"janvier", "fevrier", "mars", "avril", "mai", "juin", "juillet", "aout", "septembre", "octobre", "novembre", "decembre"},
}

var monthsShort = map[Locale][]string{
	LocaleEN: {"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	LocaleDE: {"Jan", "Feb", "Mar", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"},
	LocaleFR: {"janv.", "fevr.", "mars", "avr.", "mai", "juin", "juil.", "aout", "sept.", "oct.", "nov.", "dec."},
}

func FormatDateLong(ctx context.Context, value time.Time) string {
	locale := LocaleFromContext(ctx)
	monthName := monthName(monthsLong[locale], value.Month())
	switch locale {
	case LocaleDE:
		return strconv.Itoa(value.Day()) + ". " + monthName + " " + strconv.Itoa(value.Year())
	case LocaleFR:
		return strconv.Itoa(value.Day()) + " " + monthName + " " + strconv.Itoa(value.Year())
	default:
		return monthName + " " + strconv.Itoa(value.Day()) + ", " + strconv.Itoa(value.Year())
	}
}

func FormatDateShort(ctx context.Context, value time.Time) string {
	locale := LocaleFromContext(ctx)
	monthName := monthName(monthsShort[locale], value.Month())
	switch locale {
	case LocaleDE:
		return value.Format("02.01.")
	case LocaleFR:
		return strconv.Itoa(value.Day()) + " " + monthName
	default:
		return monthName + " " + strconv.Itoa(value.Day())
	}
}

func FormatNumber(ctx context.Context, value int) string {
	switch LocaleFromContext(ctx) {
	case LocaleDE:
		return printerDE.Sprintf("%d", value)
	case LocaleFR:
		return printerFR.Sprintf("%d", value)
	default:
		return printerEN.Sprintf("%d", value)
	}
}

func monthName(months []string, month time.Month) string {
	if len(months) < int(month) || month <= 0 {
		return ""
	}
	return months[month-1]
}
