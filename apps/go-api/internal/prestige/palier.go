package prestige

// palier.go — calcul du palier de difficulté d'un défi.
//
// Référence : Annexe B du plan conceptuel.
// Combine deux signaux :
//   1. Stretch personnel : à quel point la cible dépasse la baseline du joueur
//   2. Plancher population : où se situe la cible dans la population active
//
// Le palier final = min(palier_personnel, cap_population).
// Si la population active est inférieure à AntiSmurf.PopulationMinThreshold,
// le cap population est désactivé (palier purement personnel).

// CalculatePalierInput regroupe les entrées du calcul de palier.
type CalculatePalierInput struct {
	Stretch              float64  // ratio cible/baseline (voir stretch.go)
	PopulationPercentile float64  // [0, 1] position de la cible dans la population
	PopulationSize       int      // nb de joueurs ayant cette métrique sur la fenêtre
	DataTier             DataTier // qualité des données du joueur
}

// CalculatePalier détermine le palier d'un défi à sa création.
//
// Retourne (TierNormal..TierMythic, RejectNone) en cas de succès.
// Retourne (TierNormal, RejectXxx) en cas de rejet — la valeur de Tier
// n'est alors pas significative.
//
// Règles :
//   - DataTier=tracking → rejet "insufficient_data" (le défi peut quand même
//     être créé mais sans bonus PP)
//   - Stretch < tuning.AntiSmurf.MinStretch + 1 → rejet "too_easy"
//   - Sinon : palier personnel selon Stretch.Normal/Heroic/Legendary/Mythic
//   - Cap population appliqué SI PopulationSize ≥ tuning.AntiSmurf.PopulationMinThreshold
//   - Palier final = min(personnel, cap)
func CalculatePalier(t Tuning, in CalculatePalierInput) (Tier, RejectReason) {
	// Données insuffisantes : refus formel pour le bonus PP, mais
	// la création peut quand même se faire (gestion en amont).
	if in.DataTier == DataTracking {
		return TierNormal, RejectInsufficientData
	}

	minStretch := 1.0 + t.AntiSmurf.MinStretch
	if in.Stretch < minStretch {
		return TierNormal, RejectTooEasy
	}

	personal := personalTier(t.Stretch, in.Stretch)

	// Cap population uniquement si la pop est suffisante pour être significative.
	if in.PopulationSize >= t.AntiSmurf.PopulationMinThreshold {
		cap := t.PopulationCapTier(in.PopulationPercentile)
		personal = capTier(personal, cap)
	}

	return personal, RejectNone
}

// personalTier retourne le palier personnel selon le stretch ratio.
func personalTier(thresholds StretchThresholds, stretch float64) Tier {
	switch {
	case stretch >= thresholds.Mythic:
		return TierMythic
	case stretch >= thresholds.Legendary:
		return TierLegendary
	case stretch >= thresholds.Heroic:
		return TierHeroic
	default:
		return TierNormal
	}
}

// capTier retourne le minimum de deux paliers selon l'ordre Normal < Heroic < Legendary < Mythic.
func capTier(a, b Tier) Tier {
	if tierRank(a) <= tierRank(b) {
		return a
	}
	return b
}

func tierRank(t Tier) int {
	switch t {
	case TierNormal:
		return 0
	case TierHeroic:
		return 1
	case TierLegendary:
		return 2
	case TierMythic:
		return 3
	}
	return -1
}
