package skill_v2

// legacy_mapping.go : mapping μ (échelle TrueSkill v2) → rating_value
// legacy (échelle LUSR v1 [1000..2000]) pour la Stratégie C (write-through
// aliasing).
//
// Quand LUSR v2 devient canonical (LEVELUP_LUSR_CANONICAL=LUSR_V2), le
// shadow runner écrit dans `match_skill_rank` avec `rating_type='LUSR'`
// (slot historique) en utilisant cette fonction de mapping. Les readers
// UI continuent de lire `rating_type='LUSR'` sans aucune modification.
//
// Principe : préserver les tiers Bronze..Onyx exactement. Le tier v2
// (e.g., "Or IV") est calculé par InferTier(mu, v2_boundaries), puis on
// génère un rating_value v1 dans la plage correspondante du tier v1.

// LegacyTierRange retourne les bornes [min, max[ du tier dans la grille
// legacy LUSR v1 (cf. internal/sync/skill_config.go SkillTiers).
//
// Dupliqué ici plutôt qu'importé pour éviter une dépendance cyclique
// vers internal/sync. Les valeurs legacy sont stables (figées depuis
// LUSR v1 production) — pas de risque de drift.
func LegacyTierRange(tierName string) (min, max float64) {
	switch tierName {
	case "Bronze":
		return 1000, 1200
	case "Silver":
		return 1200, 1400
	case "Gold":
		return 1400, 1600
	case "Platinum":
		return 1600, 1800
	case "Diamond":
		return 1800, 2000
	case "Onyx":
		// Onyx est ouvert en haut — on retourne 2000 comme point d'ancrage,
		// les rares joueurs Onyx auront des rating_value au-dessus selon
		// leur μ. Cf. MapMuToLegacyRating.
		return 2000, 2200
	default:
		return 1000, 1200
	}
}

// MapMuToLegacyRating : μ v2 → rating_value legacy v1.
//
// Algorithme :
//   1. Trouve le tier v2 (e.g., "Or IV") via InferTier
//   2. Récupère la plage [min, max[ du tier dans la grille v1
//   3. rating = min + (sub-1)/N · (max - min) avec N = nombre de sous-tiers
//
// Pour Onyx (sub=0), retourne min ; le caller pourrait ajouter un bonus
// proportionnel à (μ - boundary_onyx) si on veut différencier les Onyx.
//
// Exemple Madina Diamant II (μ=26.17) :
//   v2 tier = Diamond, sub = 2
//   v1 range = [1800, 2000]
//   rating = 1800 + 1/6 × 200 = 1833
func MapMuToLegacyRating(mu float64, v2Boundaries []TierBoundary) float64 {
	tier, sub := InferTier(mu, v2Boundaries)
	if tier.Name == "" {
		// Cas pathologique (μ très bas) — retourne Bronze entrée.
		return 1000
	}
	minR, maxR := LegacyTierRange(tier.Name)
	if tier.SubTiers <= 1 {
		// Onyx ou tier dégradé — pas de sub-tier, retourne min.
		// Pour Onyx, on pourrait ajouter (μ - tier.MinMu) · k pour le
		// bonus open-ended, mais pour Phase 3e MVP on reste sur l'ancre.
		return minR
	}
	// Sub est 1-based. sub=1 → début du tier (rating = minR).
	// sub=SubTiers → haut du tier (rating ≈ maxR - 1 sub-band).
	band := (maxR - minR) / float64(tier.SubTiers)
	return minR + float64(sub-1)*band
}

// MapSigmaToLegacyDeviation : σ v2 → rating_deviation legacy v1.
//
// Le LUSR v1 utilise σ ∈ [60, 350] (cf. skill_config.go MinSigma/MaxSigma).
// Le LUSR v2 utilise σ ∈ [τ, σ_0] ≈ [0.08, 8.33]. Facteur d'échelle :
// (350-60)/(8.33-0.08) ≈ 35. Mais en pratique nos σ posteriors convergent
// vers 0.7 typiquement, ce qui mapperait à v1 σ ≈ 85 (très "confiant").
//
// Mapping pragmatique : σ_v1 = clamp(σ_v2 × 40, 60, 350). Le 40 est
// calibré pour qu'un σ_v2 de 0.7 (skill bien établi) donne σ_v1 ≈ 90.
func MapSigmaToLegacyDeviation(sigma float64) float64 {
	v1 := sigma * 40
	if v1 < 60 {
		return 60
	}
	if v1 > 350 {
		return 350
	}
	return v1
}
