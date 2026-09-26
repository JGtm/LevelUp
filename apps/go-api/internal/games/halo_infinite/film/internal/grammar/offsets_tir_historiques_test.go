package grammar

// offsets_tir_historiques_test.go — LES OFFSETS FIXES DU RECORD DE TIR D AVANT LE LOT M4b, pour les
// SEULS instruments historiques qui les MESURENT (`lot1_visee_compare_research_test.go`,
// `lot1_visee_ghidra_research_test.go` : l ancre de visee au bit 113 du « record vide » contre le
// chemin modal).
//
// La production ne les porte plus : depuis le lot M4b (2026-09-24), la tete du record est lue par
// sa grammaire (`fire_events.go`, [lireEnteteTir36]), et ces offsets ne valaient que pour la
// disposition canonique. Ils vivent ici, sous `_test.go`, et le ratchet
// `archlint/record36_grammaire_test.go` interdit qu ils reviennent en production.
const (
	// fireFlagsBit : le premier des cinq « drapeaux » 108..112 de la disposition canonique (en
	// grammaire : `i`, `j`, la porte des comptes, les portes des deux composites).
	fireFlagsBit = 108
	// fireAimBit : la visee du « record vide » canonique, au bit 113 (post-comptes + 2).
	fireAimBit = 113
)
