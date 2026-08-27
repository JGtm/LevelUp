package skillchain

// objective_family.go — SOURCE UNIQUE de la liste des sous-modes de la famille
// OBJECTIF (Halo Infinite).
//
// Deux classifications partagent cette liste et DOIVENT rester d'accord :
//   - chaîne LUSR sociale : lusrChainForAssassin → arena_objectif / arena_slayer ;
//   - chaîne du score de performance en classé : skill.GetPerformanceChain →
//     ranked_objectif / ranked_slayer, atteinte via le seam title-aware
//     sync.SetObjectiveFamilyClassifier(skillchain.IsObjectiveSubMode) posé au boot.
//
// Une 2e copie de la liste est interdite par le ratchet
// internal/archlint/no_objective_submode_list_test.go : deux copies divergent
// (un ajout fait d'un seul côté classerait le même match en famille objectif pour
// le LUSR et en famille slayer pour la performance).
//
// LACUNES CONNUES, NON CORRIGÉES ICI (décision D-I du plan
// .ai/PLAN_PERF_NOTE_OBJECTIFS.md, 2026-08-27) : l'Assaut (`neutral bomb`,
// `one bomb`), `vip`, `ctf 3 captures` et les pair_names inversés
// (`Strongholds:Arena on Behemoth`) tombent en famille slayer. Élargir la liste
// changerait AUSSI les chaînes LUSR déjà persistées dans
// match_skill_rank.playlist_group (recompute LUSR hors périmètre) : la correction
// se fera avec ce recompute, pas ici.

import "levelup/go-api/internal/analysis"

// IsObjectiveSubMode indique si le sous-mode porté par un pair_name Halo Infinite
// appartient à la famille OBJECTIF (par opposition à la famille slayer/combat).
// Le pair_name est normalisé (NormalizeModeLabel : strip du préfixe de playlist et
// du suffixe de carte) puis comparé en ASCII minuscule.
//
// pair_name vide, sous-mode inconnu ou non listé → false (famille slayer) : c'est
// le fallback assumé des deux consommateurs.
func IsObjectiveSubMode(pairName string) bool {
	switch toLowerASCII(analysis.NormalizeModeLabel(pairName)) {
	case "ctf", "capture the flag", "neutral flag ctf", "one flag ctf", "covert one flag",
		"strongholds", "oddball", "king of the hill",
		"total control", "land grab", "extraction", "stockpile":
		return true
	default:
		return false
	}
}
