package service

import "levelup/go-api/internal/domain"

// explorerMapFilterLabel est le libellé de carte que le filtre de cartes de l'Explorateur propose et
// compare : la carte de BASE quand la carte jouée en est une variante (MapFilterLabel), sinon le
// libellé affiché (FR > EN). Une variante est donc regroupée sous sa carte de base dans les options.
func explorerMapFilterLabel(r domain.MatchHistoryRawRow) string {
	if r.MapFilterLabel != nil && *r.MapFilterLabel != "" {
		return *r.MapFilterLabel
	}
	return coalesceStr(r.MapNameFR, r.MapName)
}

// explorerMapSelected dit si la carte d'une ligne est retenue par `maps` : par son libellé de filtre
// (la carte de base, que le filtre propose), ou par son libellé affiché. Le second cas garde valide
// le lien « Voir les matchs de cette carte » de la Tactique, dont la vignette d'une variante porte
// son propre libellé : il ne ramène que les parties de cette variante.
func explorerMapSelected(r domain.MatchHistoryRawRow, maps map[string]struct{}) bool {
	if _, ok := maps[explorerMapFilterLabel(r)]; ok {
		return true
	}
	_, ok := maps[coalesceStr(r.MapNameFR, r.MapName)]
	return ok
}
