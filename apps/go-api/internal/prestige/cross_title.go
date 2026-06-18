package prestige

// cross_title.go — point d'extension UNIQUE pour le crédit PP cross-titre.
//
// Socle backend-ready posé par PLAN_CROSS_TITLE_ARCS_BACKEND (Phase 3). Un arc
// peut, à terme, couvrir plusieurs titres (table de jointure arc_titles) ; la
// répartition des PP d'un défi réellement cross-titre — chaque titre, le titre
// primaire seul, ou au prorata — est une décision PRODUIT + UX non tranchée tant
// que le 2e titre n'est pas validé.
//
// En attendant, ce module expose le seul point surchargeable, au comportement
// strictement mono-titre : aucun changement observable aujourd'hui.

// creditTitlesFor retourne les title_slug auxquels créditer les PP d'un défi
// complété. AUJOURD'HUI : le titre primaire uniquement → comportement identique
// à l'historique (un seul PrestigeEvent, sur c.TitleSlug).
//
// Point d'extension : le jour où un défi cross-titre doit créditer plusieurs
// titres, c'est ICI qu'on décide la politique (et nulle part ailleurs). Le
// crédit reste donc traçable à une fonction unique et testable.
func creditTitlesFor(c Challenge) []string {
	return []string{c.TitleSlug}
}
