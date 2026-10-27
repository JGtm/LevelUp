package replay

import "levelup/go-api/internal/i18n"

// enFr fabrique le libellé d'un test dans les deux langues `required`.
func enFr(en, fr string) i18n.Text {
	return i18n.Text{i18n.LocaleEN: en, i18n.LocaleFR: fr}
}

// enOf / frOf lisent la langue d'un libellé sans repli : une langue absente rend "".
func enOf(t i18n.Text) string { return t[i18n.LocaleEN] }
func frOf(t i18n.Text) string { return t[i18n.LocaleFR] }
