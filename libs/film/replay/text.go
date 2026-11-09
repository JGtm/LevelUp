package replay

import (
	"slices"
	"strings"
)

// text.go — LE LIBELLÉ PAR LANGUE QUE LE DOCUMENT TRANSPORTE (plan D2, item 2.9).
//
// Text associe un code de langue (`fr`, `en`, ... : la clé de `config/locales.toml`) à un libellé.
// La brique ne connaît AUCUNE liste de langues et n'en choisit aucune : elle reçoit des textes du
// titre (`replaylabels.TexteVersFilm`, qui pose chaque langue de l'application, vide comprise),
// les copie dans le document et les écrit à plat (`film/internal/flatjson`). Le choix d'une langue
// — repli compris — appartient à LevelUp, après conversion (`replaylabels.TexteDepuisFilm`).
type Text map[string]string

// Clone retourne une copie indépendante (nil reste nil).
func (t Text) Clone() Text {
	if t == nil {
		return nil
	}
	out := make(Text, len(t))
	for code, v := range t {
		out[code] = v
	}
	return out
}

// Empty indique si le texte ne porte aucune valeur non vide, dans aucune langue : c'est la
// question « ce libellé existe-t-il ? ».
func (t Text) Empty() bool {
	for _, v := range t {
		if v != "" {
			return false
		}
	}
	return true
}

// Compare ordonne deux textes langue par langue, dans l'ordre alphabétique des codes que portent
// l'un ou l'autre, sur leurs valeurs (une langue absente vaut ""). Ordre TOTAL : deux textes qu'il
// ne sépare pas portent les mêmes valeurs dans chaque langue. Il sert aux tris déterministes du
// document ; sur des textes dont les codes sont ceux de l'application, il rend l'ordre de
// `i18n.Text.Compare`.
func (t Text) Compare(o Text) int {
	codes := make([]string, 0, len(t)+len(o))
	for code := range t {
		codes = append(codes, code)
	}
	for code := range o {
		codes = append(codes, code)
	}
	slices.Sort(codes)
	for _, code := range slices.Compact(codes) {
		if c := strings.Compare(t[code], o[code]); c != 0 {
			return c
		}
	}
	return 0
}
