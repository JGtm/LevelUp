// Package duckdb — locale_pick.go : choix d'un nom d'affichage dans la langue de la requête. La
// règle ne code aucune langue : le texte est rassemblé dans un i18n.Text (une entrée par langue lue
// dans les tables par langue) puis lu le long de i18n.Chain, pour qu'une langue de plus soit lue
// sans changer ce code (ADR 0040).
package duckdb

import (
	"strings"

	"levelup/go-api/internal/i18n"
)

// pickAcrossChain retourne le premier texte non vide (espaces exclus) le long de
// i18n.Chain(loc) : la langue demandée, la langue de repli, la langue par défaut. Repli CROISÉ
// des libellés d'affichage (nom de citation, de playlist, de carte) : un nom manquant dans la
// langue demandée est comblé par n'importe quelle autre langue plutôt que laissé vide. Les
// descriptions n'ont PAS ce repli (pickCitationTexts, citation_texts.go). La valeur est rendue telle que stockée.
func pickAcrossChain(loc i18n.Locale, t i18n.Text) string {
	for _, l := range i18n.Chain(loc) {
		if v := t[l]; strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
