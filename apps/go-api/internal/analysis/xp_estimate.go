package analysis

import (
	"math"
	"time"

	"levelup/go-api/internal/games/mappings"
)

// xp_estimate.go — estimation de l'XP de carrière (Career Rank) gagnée par match.
//
// La progression Career Rank d'Halo Infinite crédite le Personal/Applied Score du
// match (343), multiplié par un facteur d'éra. Depuis Operation: Infinite
// (2025-11-18), tous les multiplicateurs d'Applied Score sont doublés (×2 depuis,
// ×1 avant). Les éras vivent en TOML versionné (config/titles/{slug}/constants.toml
// [[career_xp_eras]]) et sont résolues title-agnostic via games.CareerXPErasFor ;
// cette fonction est PURE (aucun accès config/DB) — elle reçoit les éras en entrée.
//
// v1 = multiplicateur UNIFORME par éra : pas de multiplicateur par playlist (BTB /
// bots non calibrables sur nos données — raffinement backlog). Valeur ESTIMÉE :
// l'étiquette produit l'indique honnêtement (« XP de carrière (estimée) »).

// EstimateCareerXP retourne l'XP de carrière estimée d'un match : multiplicateur de
// l'éra couvrant endedAt × personalScore, arrondi à l'entier le plus proche.
//
// Contrat de dégradation (jamais de panic) :
//   - eras vide → 0 (aucun multiplicateur connu) ;
//   - endedAt hors de toute éra (trou de couverture) → 0 ;
//   - personalScore == 0 → 0.
//
// Bornes d'éra : From INCLUSIVE, To EXCLUSIVE ; From.IsZero() = -inf, To.IsZero() =
// +inf (cf. mappings.CareerXPEra). Un match exactement à la borne (2025-11-18
// 00:00:00 UTC) relève de l'éra dont c'est le From (×2).
func EstimateCareerXP(personalScore int, endedAt time.Time, eras []mappings.CareerXPEra) int {
	mult, ok := careerXPMultiplierAt(endedAt, eras)
	if !ok || mult <= 0 {
		return 0
	}
	return int(math.Round(mult * float64(personalScore)))
}

// careerXPMultiplierAt résout le multiplicateur de l'éra couvrant instant t.
// Retourne (0, false) si aucune éra ne couvre t (ou liste vide).
func careerXPMultiplierAt(t time.Time, eras []mappings.CareerXPEra) (float64, bool) {
	for _, era := range eras {
		if !era.From.IsZero() && t.Before(era.From) {
			continue // avant le début (inclusif) de l'éra
		}
		if !era.To.IsZero() && !t.Before(era.To) {
			continue // à ou après la fin (exclusive) de l'éra
		}
		return era.Multiplier, true
	}
	return 0, false
}
