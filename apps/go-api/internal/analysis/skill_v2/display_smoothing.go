package skill_v2

// display_smoothing.go — lissage de l'AFFICHAGE du palier LUSR v2.
//
// Sépare la vérité interne (μ honnête, jamais bridé) de la présentation : le
// palier affiché monte immédiatement mais ne descend que d'un sous-palier par
// match (hystérésis / demotion protection façon CSR). C'est le standard pour
// éviter les "grosses chutes en un match" sans corrompre la convergence du
// modèle bayésien.
//
// Justification chiffrée : étude .ai/thought_log.md [2026-05-31]. Sur les 3
// joueurs trackés (full history, σ convergé ~0.67), la volatilité résiduelle
// est un problème de QUEUE (matchs rares à fort Δμ : btb max 2.13 μ, quits).
// L'hystérésis élimine 100 % des chutes abruptes (−≥2 sous-paliers) avec un lag
// final de 0–1 sous-palier.
//
// Toutes les fonctions sont pures (no I/O). Le câblage vit dans
// internal/sync/skill_v2_canonical.go (writeCanonicalLUSRRow).

// PlacementMatches : nombre de matchs d'un groupe pendant lesquels le rang
// affiché n'est PAS lissé. σ est encore élevé (l'étude montre σ < 1.0 seulement
// après ~31 matchs, < 1.5 après 13) : laisser le rang converger vite est
// légitime — l'incertitude est réelle, pas du bruit à amortir.
const PlacementMatches = 10

// romanSubTier mappe un sous-palier 1-based vers son chiffre romain. Partagé
// par FormatTierLabel et FormatTierSubLabel.
var romanSubTier = map[int]string{1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}

// TierOrdinal projette un (tierName, sub 1-based) sur un ordinal global croissant
// (0-based) le long de la grille. Onyx (sub 0) → base du tier. Retourne -1 si le
// tier est inconnu de la grille. Inverse de TierSubFromOrdinal.
func TierOrdinal(boundaries []TierBoundary, tierName string, sub int) int {
	base := 0
	for _, b := range boundaries {
		if b.Name == tierName {
			if sub <= 0 {
				return base
			}
			return base + (sub - 1)
		}
		base += subTierCount(b)
	}
	return -1
}

// TierSubFromOrdinal est l'inverse de TierOrdinal : retourne le (tier, sub 1-based)
// correspondant à un ordinal global. Onyx → sub 0. Clamp aux extrêmes.
func TierSubFromOrdinal(boundaries []TierBoundary, ord int) (TierBoundary, int) {
	if len(boundaries) == 0 {
		return TierBoundary{}, 0
	}
	if ord < 0 {
		return boundaries[0], firstSub(boundaries[0])
	}
	base := 0
	for _, b := range boundaries {
		n := subTierCount(b)
		if ord < base+n {
			if b.SubTiers <= 1 {
				return b, 0
			}
			sub := ord - base + 1
			if sub < 1 {
				sub = 1
			}
			if sub > n {
				sub = n
			}
			return b, sub
		}
		base += n
	}
	// au-delà du sommet → haut du dernier tier.
	last := boundaries[len(boundaries)-1]
	if last.SubTiers <= 1 {
		return last, 0
	}
	return last, last.SubTiers
}

// SmoothDisplayedOrdinal applique l'hystérésis sur l'ordinal affiché :
//   - pas de précédent (prevOrd < 0) ou phase de placement (experience ≤
//     PlacementMatches) → pas de lissage, on affiche la cible.
//   - cible ≥ précédent → promotion immédiate.
//   - cible < précédent → descente bridée à 1 sous-palier par match.
//
// experience = nombre de matchs LUSR-éligibles joués dans le groupe, ce match
// inclus (cf. domain.SkillV2State.Experience).
func SmoothDisplayedOrdinal(prevOrd, targetOrd, experience int) int {
	if prevOrd < 0 || experience <= PlacementMatches {
		return targetOrd
	}
	if targetOrd >= prevOrd {
		return targetOrd
	}
	return prevOrd - 1
}

// FormatTierSubLabel retourne le libellé FR ("Or III", "Onyx", "Non classé")
// pour un (tier, sub) déjà résolu — utilisé par le chemin lissé qui dispose
// directement de la cible affichée, sans repasser par μ.
func FormatTierSubLabel(tier TierBoundary, sub int) string {
	if tier.Name == "" {
		return "Non classé"
	}
	if sub == 0 || tier.SubTiers <= 1 {
		return tier.NameFR
	}
	r, ok := romanSubTier[sub]
	if !ok {
		return tier.NameFR
	}
	return tier.NameFR + " " + r
}

// mapTierSubToLegacyRating mappe un (tier, sub) résolu vers le rating_value
// legacy [1000..2000] — variante de mapMuToLegacyRating pour le chemin lissé
// (le rating doit refléter le palier AFFICHÉ, pas μ brut, sinon incohérence
// label/valeur quand l'hystérésis bride la descente).
func mapTierSubToLegacyRating(tier TierBoundary, sub int) float64 {
	if tier.Name == "" {
		return 1000
	}
	minR, maxR := LegacyTierRange(tier.Name)
	if tier.SubTiers <= 1 {
		return minR
	}
	band := (maxR - minR) / float64(tier.SubTiers)
	s := sub
	if s < 1 {
		s = 1
	}
	return minR + float64(s-1)*band
}

// subTierCount retourne le nombre de sous-paliers d'un tier (≥1).
func subTierCount(b TierBoundary) int {
	if b.SubTiers < 1 {
		return 1
	}
	return b.SubTiers
}

// firstSub retourne le sous-palier de départ d'un tier (0 pour Onyx, sinon 1).
func firstSub(b TierBoundary) int {
	if b.SubTiers <= 1 {
		return 0
	}
	return 1
}
