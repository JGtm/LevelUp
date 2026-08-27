// Package skillchain — classification title-owned d'un pair_name Halo Infinite
// vers une chaîne TrueSkill LUSR (MT-15, déplacé verbatim depuis
// internal/sync/skill_config.go).
//
// La logique de catégorisation des modes est Halo-spécifique (cf. skill halo-modes) :
// elle vit donc dans le package de titre et remonte au moteur LUSR title-agnostique
// (internal/sync) via le seam sync.SetLUSRChainClassifier(ClassifyLUSRChain), posé
// au boot. Aucun import de internal/sync ici (sync importe halo_infinite → un import
// inverse créerait un cycle) : les 4 valeurs de chaîne sont dupliquées localement et
// vérouillées byte-identiques par un cross-check test (package sync_test).
package skillchain

import (
	"levelup/go-api/internal/games/halo_infinite"
)

// Valeurs canoniques des chaînes LUSR — DUPLIQUÉES de sync.LUSRChain* (valeurs
// persistées dans match_skill_rank.playlist_group). Un cross-check test garantit
// l'égalité stricte avec les constantes sync (interdiction d'import sync = cycle).
const (
	chainArenaSlayer   = "arena_slayer"
	chainArenaObjectif = "arena_objectif"
	chainBTB           = "btb"
	chainChaos         = "chaos"
)

// ClassifyLUSRChain détermine la chaîne TrueSkill LUSR depuis le pair_name d'un
// match. Retourne "" si le match est exclu du LUSR (Ranked → CSR, Firefight → PvE).
//
// Classification :
//   - Ranked, Firefight                          → exclu ("")
//   - BTB, BTB Heavies                           → btb
//   - Fiesta, Super Fiesta, Husky Raid           → chaos
//   - Other : Infection/Griffball/Rocket Hog/Action Sack/Event → chaos
//     Rumble Pit + préfixes inconnus     → arena_slayer (fallback)
//   - Assassin (Arena/Tactical/Assault/Community) :
//     sous-mode objectif (CTF, Strongholds…)  → arena_objectif
//     tout le reste                            → arena_slayer
func ClassifyLUSRChain(pairName string) string {
	category := halo_infinite.InferModeCategoryFromPairName(pairName)
	switch category {
	case halo_infinite.ModeCategoryRanked, halo_infinite.ModeCategoryFirefight:
		return ""
	case halo_infinite.ModeCategoryBTB:
		if containsI(pairName, "rocket hog") {
			return chainChaos
		}
		return chainBTB
	case halo_infinite.ModeCategoryFiesta, halo_infinite.ModeCategorySuperFiesta, halo_infinite.ModeCategoryHuskyRaid:
		return chainChaos
	case halo_infinite.ModeCategoryOther:
		return lusrChainForOther(pairName)
	default: // ModeCategoryAssassin
		return lusrChainForAssassin(pairName)
	}
}

// lusrChainForOther classe les modes de catégorie Other.
// Chaos : Infection, Griffball, Rocket Hog Race, Action Sack, Event.
// Fallback arena_slayer : Rumble Pit et tout préfixe inconnu.
func lusrChainForOther(pairName string) string {
	if containsI(pairName, "infection") || containsI(pairName, "griffball") ||
		containsI(pairName, "rocket hog") || containsI(pairName, "action sack") ||
		containsI(pairName, "event") {
		return chainChaos
	}
	return chainArenaSlayer
}

// lusrChainForAssassin classe les sous-modes Arena/Tactical/Assault/Community.
// La liste des sous-modes objectif reconnus vit dans IsObjectiveSubMode
// (objective_family.go) — SOURCE UNIQUE partagée avec la chaîne de performance
// classée (ranked_objectif). Tout le reste (Slayer, Attrition, Elimination,
// inconnu) → arena_slayer.
func lusrChainForAssassin(pairName string) string {
	if IsObjectiveSubMode(pairName) {
		return chainArenaObjectif
	}
	return chainArenaSlayer
}

// containsI est un contains case-insensitive simplifié.
func containsI(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	s = toLowerASCII(s)
	substr = toLowerASCII(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
