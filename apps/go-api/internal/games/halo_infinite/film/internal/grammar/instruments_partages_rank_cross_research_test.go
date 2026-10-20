//go:build research

package grammar

// instruments_partages_rank_cross_research_test.go — rangs et predicat de rank_cross_shared_test.go dont
// les seuls utilisateurs sont des instruments `research` (J12.7 lint). Deplacement pur.

// LES RANGS DES CAPACITÉS DU PÉRIMÈTRE, source unique. Ils apparaissaient dans deux
// instruments (`i56_rank_cross`, `i59_rank_cross`) et un troisième les demandait : à la
// troisième copie, la règle du dépôt est de centraliser. Famille A : 4 grappin, 5
// propulseur, 6 répulseur (RECETTE_LOADOUT §13). Famille B : 20 grappin, 21 propulseur
// (`ability_evade`, nommé par murmur3 le 2026-08-18). Aucun rang famille B n'est établi pour
// le répulseur — un film de famille B ne le contredit pas, il l'ignore.
var (
	xrRepulsorRanks = []int{6}
	xrThrusterRanks = []int{5, 21}
	xrGrappleRanks  = []int{4, 20}
)

// xrHasAnyRank dit si la vie a transmis au moins un des rangs demandés.
func xrHasAnyRank(ranks []int, want []int) bool {
	for _, w := range want {
		if eaHasRank(ranks, w) {
			return true
		}
	}
	return false
}
