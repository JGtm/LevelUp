package prestige

import "math"

// pp_amounts.go — calcul des PP attribués à un événement Prestige.
//
// Référence : Annexe B (multiplicateur XP par palier) + Axe 6 (sources de PP).

// PPForCompletion retourne le nombre de PP gagnés à la complétion d'un défi.
//
// Formule :
//
//	base = tuning.PPAmounts[tier]
//	multiplicateur_data = tuning.DataTier[dataTier]      (1.0 / 0.5 / 0.0)
//	multiplicateur_squad = 1.0 ou tuning.SquadMultiplier (si défi escouade)
//
//	PP = arrondi(base * multiplicateur_data * multiplicateur_squad)
//
// Si dataTier=tracking → PP = 0 (multiplicateur=0). Le défi est validé,
// l'historique le note, mais aucun PP n'est crédité — cohérent avec la
// règle "tracking pur" en données insuffisantes.
func PPForCompletion(t Tuning, tier Tier, isSquad bool, dataTier DataTier) int {
	base := float64(t.PPForTier(tier))
	if base == 0 {
		return 0
	}
	dataMul := t.DataTierMultiplierFor(dataTier)
	squadMul := 1.0
	if isSquad {
		squadMul = t.PPAmounts.SquadMultiplier
	}
	return int(math.Round(base * dataMul * squadMul))
}

// PPForArcCompletion retourne le bonus PP attribué à la complétion d'un arc.
//
// Pas de multiplicateur data_tier sur le bonus arc (l'arc requiert déjà
// que les défis successifs aient été validés, donc data_tier est implicite).
func PPForArcCompletion(t Tuning) int {
	return t.PPAmounts.ArcCompletionBonus
}

// PPForMatch retourne les PP gagnés pour un match joué (avec bonus victoire).
func PPForMatch(t Tuning, won bool) int {
	pp := t.PPAmounts.MatchPlayed
	if won {
		pp += t.PPAmounts.MatchWon
	}
	return pp
}

// PPForStreak retourne le bonus de streak pour 3 sessions consécutives.
func PPForStreak(t Tuning) int {
	return t.PPAmounts.Streak3Sessions
}

// PPForMedal retourne les PP pour une médaille rare obtenue.
//
// raretyRatio dans [0, 1] où 0 = la plus commune des rares (Heroic+), 1 = la plus rare.
// Interpolation linéaire entre PPAmounts.MedalMin et PPAmounts.MedalMax.
func PPForMedal(t Tuning, raretyRatio float64) int {
	if raretyRatio < 0 {
		raretyRatio = 0
	}
	if raretyRatio > 1 {
		raretyRatio = 1
	}
	low := float64(t.PPAmounts.MedalMin)
	high := float64(t.PPAmounts.MedalMax)
	return int(math.Round(low + (high-low)*raretyRatio))
}
