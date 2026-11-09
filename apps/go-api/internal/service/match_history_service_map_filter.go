package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// explorerMapFilterLabel est la CLÉ de carte que le filtre de cartes de l'Explorateur propose et
// compare : le nom, dans la langue par défaut, de la carte de BASE quand la carte jouée en est une
// variante (MapFilterNames), sinon celui de la carte affichée. Une variante est donc regroupée sous sa
// carte de base dans les options. Elle ne dépend pas de la langue de la requête.
func explorerMapFilterLabel(r domain.MatchHistoryRawRow) string {
	if v := r.MapFilterNames[i18n.Default]; v != "" {
		return v
	}
	return r.MapName.Pick(i18n.Default)
}

// explorerMapFilterLabelIn est le libellé AFFICHÉ de l'option de carte de la ligne, dans la langue `loc` :
// le nom de la carte de base quand la carte est une variante, sinon celui de la carte jouée.
func explorerMapFilterLabelIn(r domain.MatchHistoryRawRow, loc i18n.Locale) string {
	if len(r.MapFilterNames) > 0 && r.MapFilterNames[i18n.Default] != "" {
		return r.MapFilterNames.Pick(loc)
	}
	return r.MapName.Pick(loc)
}

// explorerMapSelected dit si la carte d'une ligne est retenue par `maps` : par sa clé de filtre
// (la carte de base, que le filtre propose), ou par son nom dans la langue par défaut. Le second cas
// garde valide le lien « Voir les matchs de cette carte » de la Tactique, dont la vignette d'une
// variante porte son propre nom (map_filter_key) : il ne ramène que les parties de cette variante.
func explorerMapSelected(r domain.MatchHistoryRawRow, maps map[string]struct{}) bool {
	if _, ok := maps[explorerMapFilterLabel(r)]; ok {
		return true
	}
	_, ok := maps[r.MapName.Pick(i18n.Default)]
	return ok
}
