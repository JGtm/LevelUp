// Package domain — engagement_modes.go : catégories de mode du recompute
// d'engagement (source unique). K1n (2026-07-06) : la liste était copiée dans
// sync.engagementCoefModes ET service.engagementCoefModesService ; centralisée ici
// pour que les deux couches s'alignent sans divergence possible.
package domain

// EngagementCoefModes retourne les catégories de mode supportées par le recompute
// des coefficients d'engagement personnels. Nouvelle slice à chaque appel (pas de
// global mutable partagé entre les couches).
func EngagementCoefModes() []string {
	return []string{"PvP_ranked", "PvP_unranked"}
}
