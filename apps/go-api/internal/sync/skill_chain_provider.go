package sync

// skill_chain_provider.go — seam title-aware pour la classification LUSR (MT-15).
//
// La logique pair_name → chaîne LUSR est Halo-spécifique (cf. skill halo-modes) et
// vit désormais dans internal/games/halo_infinite/skillchain. Le moteur LUSR
// (title-agnostique) la consomme via ce provider, posé au boot par le câblage qui
// importe le package de titre : sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain).
//
// FAIL-LOUD (décision workflow + revue adversariale) : GetLUSRChain panique si le
// classifier n'est PAS posé. Sa sortie est PERSISTÉE dans match_skill_rank.playlist_group
// et partitionne l'état TrueSkill ; un fallback silencieux (ex: tout en arena_slayer)
// ferait entrer les matchs Ranked/Firefight dans le LUSR → corruption indétectable.
// Un binaire non câblé doit crasher au premier appel, pas fabriquer une chaîne
// plausible mais fausse. Le fallback légitime pour pair_name inconnu (arena_slayer)
// reste À L'INTÉRIEUR de ClassifyLUSRChain.
//
// Câblage requis : cmd/server + tout cmd LUSR (lusr_v2_canonical_backfill,
// diag_lusr_player, lusr_v2_phase0, lusr_v2_squad_estimate, lusr_v2_ttt_batch) +
// les TestMain des packages dont les tests atteignent GetLUSRChain (sync, service).

// lusrChainClassifier est la source title-owned de classification LUSR. nil tant
// que SetLUSRChainClassifier n'a pas été appelé (→ panic à l'appel de GetLUSRChain).
var lusrChainClassifier func(pairName string) string

// SetLUSRChainClassifier enregistre le classifier title-owned. À appeler au boot,
// avant tout scoring/backfill LUSR. Idempotent (dernier gagne).
func SetLUSRChainClassifier(f func(pairName string) string) { lusrChainClassifier = f }

// GetLUSRChain détermine la chaîne TrueSkill LUSR depuis le pair_name d'un match
// (délègue au classifier title-owned). Retourne "" si le match est exclu du LUSR
// (Ranked → CSR, Firefight → PvE). Panique si le classifier n'est pas câblé.
func GetLUSRChain(pairName string) string {
	if lusrChainClassifier == nil {
		panic("sync: classifier LUSR non câblé — appeler sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain) au boot")
	}
	return lusrChainClassifier(pairName)
}
