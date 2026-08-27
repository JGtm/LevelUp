package skillchain

// objective_family.go — SOURCE UNIQUE de la liste des sous-modes de la famille
// OBJECTIF (Halo Infinite).
//
// Trois classifications partagent cette liste et DOIVENT rester d'accord :
//   - chaîne LUSR sociale, catégorie Assassin : lusrChainForAssassin ;
//   - chaîne LUSR sociale, catégorie Other : lusrChainForOther (après les règles
//     chaos) — toutes deux → arena_objectif / arena_slayer ;
//   - chaîne du score de performance en classé : skill.GetPerformanceChain →
//     ranked_objectif / ranked_slayer, atteinte via le seam title-aware
//     sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) posé au boot.
//
// Une 2e copie de la liste est interdite par le ratchet
// internal/archlint/no_objective_submode_list_test.go : deux copies divergent
// (un ajout fait d'un seul côté classerait le même match en famille objectif pour
// le LUSR et en famille slayer pour la performance).
//
// LACUNES D-I CORRIGÉES le 2026-08-27 (lot 1bis du plan
// .ai/PLAN_PERF_NOTE_OBJECTIFS.md, décision utilisateur) — 26 matchs du corpus des
// 4 joueurs suivis tombaient en famille slayer alors qu'ils sont des matchs
// d'objectif :
//   - 5 sous-modes ajoutés à la liste : `vip`, `neutral bomb`, `one bomb`,
//     `neutral bomb squad` (Assaut), `ctf 3 captures` ;
//   - RÈGLE DU PRÉFIXE : certains pair_name arrivent INVERSÉS de l'API
//     (`Strongholds:Arena on Behemoth` — le mode est à GAUCHE du deux-points, le
//     conteneur de playlist à droite), 14 matchs du corpus. NormalizeModeLabel
//     retient la partie DROITE (« Arena »), qui n'est pas un mode : la famille se
//     lit alors dans le préfixe. InferModeCategoryFromPairName gère déjà cette
//     inversion pour la CATÉGORIE (mode_category.go, « Format inversé »), la
//     famille lui emboîte le pas.
//
// `arena` n'entre dans AUCUNE des deux lectures : c'est un conteneur de playlist,
// jamais un mode. Conséquence assumée de la correction : ~25 matchs sociaux
// changent de chaîne LUSR → recompute LUSR complet des joueurs suivis (lot 4 du
// plan). Périmètre NON touché : NormalizeModeLabel et
// InferModeCategoryFromPairName (catégories UI) sont inchangés.

import (
	"strings"

	"levelup/go-api/internal/analysis"
)

// IsObjectiveSubMode indique si le pair_name Halo Infinite porte un sous-mode de
// la famille OBJECTIF (par opposition à la famille slayer/combat).
//
// Deux lectures, dans cet ordre :
//  1. sous-mode = partie DROITE du pair_name (NormalizeModeLabel : strip du
//     préfixe de playlist et du suffixe de carte), comparé en ASCII minuscule ;
//  2. à défaut, PRÉFIXE = partie gauche du premier `:`, même normalisation —
//     couvre les pair_name inversés (`Strongholds:Arena on Behemoth`).
//
// Le nom parle bien du SOUS-MODE : sur un pair_name inversé, le sous-mode est la
// partie gauche. Les deux positions sont donc examinées, jamais autre chose.
//
// pair_name vide, sous-mode inconnu ou non listé → false (famille slayer) : c'est
// le fallback assumé des trois consommateurs.
func IsObjectiveSubMode(pairName string) bool {
	if isObjectiveModeLabel(toLowerASCII(analysis.NormalizeModeLabel(pairName))) {
		return true
	}
	prefix, _, hasSeparator := strings.Cut(pairName, ":")
	if !hasSeparator {
		return false
	}
	return isObjectiveModeLabel(toLowerASCII(analysis.NormalizeModeLabel(prefix)))
}

// isObjectiveModeLabel — LA liste (17 entrées), sur un label DÉJÀ normalisé
// (NormalizeModeLabel puis minuscule ASCII). Ne pas appeler ailleurs qu'ici :
// IsObjectiveSubMode est le point d'entrée de la classification.
func isObjectiveModeLabel(label string) bool {
	switch label {
	case "ctf", "capture the flag", "neutral flag ctf", "one flag ctf", "covert one flag",
		"ctf 3 captures",
		"strongholds", "oddball", "king of the hill",
		"total control", "land grab", "extraction", "stockpile",
		"vip", "neutral bomb", "one bomb", "neutral bomb squad":
		return true
	default:
		return false
	}
}
