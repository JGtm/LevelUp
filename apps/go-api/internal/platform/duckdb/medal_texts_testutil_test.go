package duckdb

import (
	"levelup/go-api/internal/i18n"
)

// frEn assemble un texte français et anglais (les deux langues obligatoires).
func frEn(fr, en string) i18n.Text {
	return i18n.Text{i18n.LocaleFR: fr, i18n.LocaleEN: en}
}
