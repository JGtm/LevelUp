package skill

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

import "errors"

// lusrChainClassifier est le classifier PAR DÉFAUT (Halo Infinite + titres sans
// classifier dédié). nil tant que SetLUSRChainClassifier n'a pas été appelé
// (→ panic à l'appel de GetLUSRChain).
var lusrChainClassifier func(pairName string) string

// lusrChainClassifiersByTitle : classifiers SPÉCIFIQUES par titre, posés au boot
// pour les titres dont la classification LUSR diffère du pattern pair_name Infinite
// (ex. Halo 5 : pas de pair_name → chaîne dérivée autrement). Vide → tous les titres
// utilisent le classifier par défaut. Key = title slug.
var lusrChainClassifiersByTitle = map[string]func(pairName string) string{}

// SetLUSRChainClassifier enregistre le classifier PAR DÉFAUT (title-owned). À appeler
// au boot, avant tout scoring/backfill LUSR. Idempotent (dernier gagne).
func SetLUSRChainClassifier(f func(pairName string) string) { lusrChainClassifier = f }

// SetLUSRChainClassifierForTitle enregistre un classifier SPÉCIFIQUE à un titre,
// consommé par GetLUSRChainForTitle. Pour les titres dont la partition LUSR ne
// dérive pas du pair_name (Halo 5). Idempotent. N'affecte pas le classifier défaut.
func SetLUSRChainClassifierForTitle(titleSlug string, f func(pairName string) string) {
	lusrChainClassifiersByTitle[titleSlug] = f
}

// GetLUSRChain détermine la chaîne TrueSkill LUSR depuis le pair_name d'un match
// via le classifier PAR DÉFAUT (Infinite). Conservé pour les callers Infinite-only.
// Retourne "" si le match est exclu du LUSR. Panique si non câblé.
func GetLUSRChain(pairName string) string {
	return GetLUSRChainForTitle("", pairName)
}

// GetLUSRChainForTitle détermine la chaîne LUSR de façon TITLE-AWARE : si le titre
// a un classifier dédié (Halo 5+), il prime ; sinon délègue au classifier défaut
// (Infinite). Le seam title-aware permet à la v2 LUSR (calcul sur données basiques)
// de tourner pour tout titre sans collapser les modes h5 dans arena_slayer.
// Retourne "" si le match est exclu (Ranked/Firefight). Panique si aucun classifier.
func GetLUSRChainForTitle(titleSlug, pairName string) string {
	if f, ok := lusrChainClassifiersByTitle[titleSlug]; ok {
		return f(pairName)
	}
	if lusrChainClassifier == nil {
		panic("sync: classifier LUSR non câblé — appeler sync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain) au boot")
	}
	return lusrChainClassifier(pairName)
}

// ValidateLUSRChainClassifierWired vérifie au BOOT que le classifier LUSR par
// défaut a été posé (Lot B, audit robustesse). Transforme le panic tardif au 1er
// match live (GetLUSRChainForTitle) en refus de démarrage propre : le caller boot
// (cmd/server) échoue AVANT de servir. Le panic FAIL-LOUD reste le filet ultime.
func ValidateLUSRChainClassifierWired() error {
	if lusrChainClassifier == nil {
		return errors.New("sync: classifier LUSR par défaut non câblé — appeler SetLUSRChainClassifier(skillchain.ClassifyLUSRChain) au boot")
	}
	return nil
}

// ── Seam famille objectif (chaîne de performance classée) ───────────────────
//
// MIROIR du seam LUSR ci-dessus, pour un besoin distinct : savoir si le sous-mode
// d'un match CLASSÉ relève de la famille objectif (→ PerfChainRankedObjectif) ou
// de la famille slayer (→ PerfChainRankedSlayer). Le classifier LUSR ne peut pas
// répondre : sur un pair_name Ranked il retourne "" (exclu, le classé va au CSR).
// La logique de liste est Halo-spécifique et vit dans le package de titre
// (skillchain.IsObjectiveSubMode) — sync/skill ne peut pas l'importer (cycle).
//
// PAS DE FAIL-LOUD ici, contrairement au seam LUSR — différence ASSUMÉE :
//   - la sortie ne partitionne pas un état TrueSkill cumulatif, elle nomme la
//     population de référence d'une note recalculable ;
//   - un binaire non câblé produit ranked_slayer (fallback), et le mécanisme de
//     skip par chaîne stockée (sync/performance.go — recompute dès que la chaîne
//     recalculée diffère de la chaîne stockée) rattrape la classification au
//     premier batch d'un binaire câblé. Un panic coûterait plus (refus de scorer)
//     qu'il ne protège.
//
// Câblage : partout où SetLUSRChainClassifier est posé (cmd/server, cmd LUSR/h5,
// TestMain des packages dont les tests atteignent GetPerformanceChain).

// objectiveFamilyClassifier est le classifier de famille PAR DÉFAUT (Halo Infinite
// et titres sans classifier dédié). nil tant qu'il n'est pas câblé → famille
// slayer (fallback documenté ci-dessus).
var objectiveFamilyClassifier func(pairName string) bool

// objectiveFamilyClassifiersByTitle : classifiers de famille SPÉCIFIQUES par titre
// (key = title slug), pour les titres dont la famille ne se lit pas dans un
// pair_name au format Infinite. Vide → classifier par défaut.
var objectiveFamilyClassifiersByTitle = map[string]func(pairName string) bool{}

// SetObjectiveFamilyClassifier enregistre le classifier de famille PAR DÉFAUT
// (title-owned). À appeler au boot, avant tout scoring de performance.
// Idempotent (dernier gagne).
func SetObjectiveFamilyClassifier(f func(pairName string) bool) { objectiveFamilyClassifier = f }

// SetObjectiveFamilyClassifierForTitle enregistre un classifier de famille
// SPÉCIFIQUE à un titre, consommé par IsObjectiveFamilyForTitle. Idempotent.
// N'affecte pas le classifier par défaut.
func SetObjectiveFamilyClassifierForTitle(titleSlug string, f func(pairName string) bool) {
	objectiveFamilyClassifiersByTitle[titleSlug] = f
}

// IsObjectiveFamilyForTitle indique, de façon TITLE-AWARE, si le sous-mode du
// pair_name relève de la famille objectif. Classifier dédié du titre s'il existe,
// sinon classifier par défaut. Aucun classifier → false (famille slayer).
func IsObjectiveFamilyForTitle(titleSlug, pairName string) bool {
	if f, ok := objectiveFamilyClassifiersByTitle[titleSlug]; ok {
		return f(pairName)
	}
	if objectiveFamilyClassifier == nil {
		return false
	}
	return objectiveFamilyClassifier(pairName)
}

// ValidateObjectiveFamilyClassifierWired vérifie au BOOT que le classifier de
// famille par défaut a été posé. Miroir de ValidateLUSRChainClassifierWired : le
// boot refuse de démarrer plutôt que de scorer tous les classés en ranked_slayer
// silencieusement (le fallback existe pour les binaires hors chemin de scoring,
// pas pour le serveur).
func ValidateObjectiveFamilyClassifierWired() error {
	if objectiveFamilyClassifier == nil {
		return errors.New("sync: classifier de famille objectif non câblé — appeler SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) au boot")
	}
	return nil
}
