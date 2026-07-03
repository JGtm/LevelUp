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
		// leur μ. Cf. mapMuToLegacyRating.
		return 2000, 2200
	default:
		return 1000, 1200
	}
}

// mapMuToLegacyRating : μ v2 → rating_value legacy v1.
//
// Algorithme :
//  1. Trouve le tier v2 (e.g., "Or IV") via InferTier
//  2. Récupère la plage [min, max[ du tier dans la grille v1
//  3. rating = min + (sub-1)/N · (max - min) avec N = nombre de sous-tiers
//
// Pour Onyx (sub=0), retourne min ; le caller pourrait ajouter un bonus
// proportionnel à (μ - boundary_onyx) si on veut différencier les Onyx.
//
// Exemple Madina Diamant II (μ=26.17) :
//
//	v2 tier = Diamond, sub = 2
//	v1 range = [1800, 2000]
//	rating = 1800 + 1/6 × 200 = 1833
func mapMuToLegacyRating(mu float64, v2Boundaries []TierBoundary) float64 {
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

// LegacySubTierRange retourne la plage legacy [min, max[ du SOUS-PALIER (tier, sub).
// Sert à CLAMPER la valeur continue dans le sous-palier AFFICHÉ (lissé) pour
// garantir la cohérence badge ↔ valeur ↔ barre de progression quand l'hystérésis
// bride une descente. Pour Onyx (sous-palier unique), retourne la plage complète
// du tier ([2000, 2200[) — ce qui donne enfin une granularité continue au sommet.
func LegacySubTierRange(tier TierBoundary, sub int) (min, max float64) {
	minR, maxR := LegacyTierRange(tier.Name)
	if tier.SubTiers <= 1 {
		return minR, maxR
	}
	band := (maxR - minR) / float64(tier.SubTiers)
	s := sub
	if s < 1 {
		s = 1
	}
	lo := minR + float64(s-1)*band
	return lo, lo + band
}

// MapMuToContinuousRating : μ v2 → rating_value legacy CONTINU (non quantifié).
//
// Variante de mapMuToLegacyRating qui, au lieu de renvoyer le BAS du sous-palier,
// interpole linéairement la position de μ DANS son tier sur la plage legacy
// [min, max[ du tier. Le résultat bouge donc à chaque variation de μ — c'est lui
// qui alimente rating_value pour LUSR v2, de sorte que rating_delta devienne un
// vrai « gain de skill par match » (cf. .ai/thought_log.md [2026-06-10]).
//
// Continue par morceaux (une pente par tier) ET continue aux frontières de tier,
// car LegacyTierRange(T).max == LegacyTierRange(T+1).min sur toute la grille.
//
// Onyx (palier ouvert en μ, pas de borne haute) : prolongement linéaire avec la
// pente du tier précédent (continuité C0 à μ=27), borné à LegacyTierRange("Onyx").max
// pour rester sur une échelle finie.
//
// Le palier/badge reste calculé séparément (InferTier + lissage d'affichage) ; le
// caller CLAMPE cette valeur via LegacySubTierRange(tier_affiché, sub_affiché) pour
// préserver la cohérence badge↔valeur quand l'hystérésis bride une descente.
func MapMuToContinuousRating(mu float64, boundaries []TierBoundary) float64 {
	if len(boundaries) == 0 {
		return 1000
	}
	// Tier de μ = plus haut palier dont MinMu ≤ μ (même logique qu'InferTier).
	idx := 0
	for i, b := range boundaries {
		if mu >= b.MinMu {
			idx = i
		}
	}
	tier := boundaries[idx]
	legacyMin, legacyMax := LegacyTierRange(tier.Name)

	// Dernier tier (Onyx, ouvert) : prolongement linéaire avec la pente du tier
	// précédent, borné à legacyMax.
	if idx+1 >= len(boundaries) {
		if idx == 0 {
			return legacyMin
		}
		prev := boundaries[idx-1]
		prevMuWidth := tier.MinMu - prev.MinMu
		pMin, pMax := LegacyTierRange(prev.Name)
		if prevMuWidth <= 0 || pMax <= pMin {
			return legacyMin
		}
		slope := (pMax - pMin) / prevMuWidth // legacy par unité de μ
		r := legacyMin + (mu-tier.MinMu)*slope
		if r < legacyMin {
			return legacyMin
		}
		if r > legacyMax {
			return legacyMax
		}
		return r
	}

	// Tier borné : interpolation linéaire de μ dans [MinMu, next.MinMu[.
	muWidth := boundaries[idx+1].MinMu - tier.MinMu
	if muWidth <= 0 {
		return legacyMin
	}
	pos := (mu - tier.MinMu) / muWidth
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	return legacyMin + pos*(legacyMax-legacyMin)
}

// LegacyContinuousSubTierProgress : position 0..1 d'un rating_value LUSR CONTINU
// dans son sous-palier d'affichage, déduite de la SEULE valeur (la grille legacy
// 1000..2200 est fixe). Remplace l'ancien `(rating mod 50)/50` — faux pour LUSR
// dont les sous-paliers ont des largeurs variables (33/67/100). Réservé au LUSR :
// pour le CSR (échelle propre, sous-paliers de 50 pts), garder le modulo.
//
//   - tiers bornés : (rating − sous_palier_min) / largeur_sous_palier.
//   - Onyx [2000,2200[ : position sur la bande Onyx (granularité continue au sommet).
//   - hors grille (< 1000 ou ≥ 2200) → ok=false (caller décide : nil/plein).
func LegacyContinuousSubTierProgress(ratingValue float64) (pct float64, ok bool) {
	for _, b := range DefaultTierBoundaries() {
		lo, hi := LegacyTierRange(b.Name)
		if ratingValue < lo || ratingValue >= hi {
			continue
		}
		sub := 1
		if b.SubTiers > 1 {
			band := (hi - lo) / float64(b.SubTiers)
			sub = int((ratingValue-lo)/band) + 1
			if sub > b.SubTiers {
				sub = b.SubTiers
			}
		}
		sLo, sHi := LegacySubTierRange(b, sub)
		if sHi <= sLo {
			return 0, true
		}
		p := (ratingValue - sLo) / (sHi - sLo)
		if p < 0 {
			p = 0
		} else if p > 1 {
			p = 1
		}
		return p, true
	}
	if ratingValue >= 2200 {
		return 1, true // au-delà du sommet Onyx → barre pleine
	}
	return 0, false
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
