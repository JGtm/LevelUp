package skill_v2

// tier.go : mapping μ → tier Halo (Bronze..Onyx) pour le LUSR v2.
//
// Le modèle skill (μ, σ) vit en interne sur l'échelle native TrueSkill
// (μ_0 = 25). La grille d'affichage UI est la grille Halo classique
// Bronze → Argent → Or → Platine → Diamant → Onyx avec 6 sous-tiers
// par tier sauf Onyx (rang unique).
//
// Les seuils sont calibrés en Phase 3e sur la distribution observée des
// μ posteriors dans player_skill_state_v2_latest (cf. cmd/lusr_v2_tier_analysis) :
//
//	Tier      μ min  Couverture % (p)  Joueur tracké de référence
//	Bronze    -      0-15              XxDaemonGamerxX (μ ≈ 20.4 → p10)
//	Argent    22.0   15-35
//	Or        23.5   35-75             Chocoboflor / JGtm (μ ≈ 23.5 → p35-40)
//	Platine   26.0   75-95             Madina97294 (μ ≈ 26.1 → p75-80)
//	Diamant   29.0   95-99
//	Onyx      31.0   99+
//
// Phase 5 (batch ré-estimation) pourra ré-écrire ces seuils dans
// lusr_hyperparams_v2 avec un source = "batch_YYYY_MM". À ce moment-là,
// LoadTierBoundaries fera le merge defaults ↔ overrides.

import (
	"fmt"
)

// TierBoundary décrit un palier de la grille tier.
type TierBoundary struct {
	Name     string  // identifiant canonique EN (Bronze, Silver, Gold, ...)
	NameFR   string  // libellé FR (Bronze, Argent, Or, ...)
	MinMu    float64 // borne inférieure (inclusive)
	SubTiers int     // nombre de sous-tiers (6 partout sauf Onyx = 1)
}

// DefaultTierBoundaries retourne la grille initiale Phase 3e, calibrée sur
// la distribution observée 2026-05-27 (~9700 joueurs, μ médian ≈ 24.5).
//
// Les bornes sont strictement ordonnées par MinMu croissant. La grille est
// "ouverte" : Bronze couvre [-∞, MinMu_Silver) ; Onyx couvre [MinMu_Onyx, +∞).
func DefaultTierBoundaries() []TierBoundary {
	return []TierBoundary{
		{Name: "Bronze", NameFR: "Bronze", MinMu: 0, SubTiers: 6},
		{Name: "Silver", NameFR: "Argent", MinMu: 22.0, SubTiers: 6},
		{Name: "Gold", NameFR: "Or", MinMu: 23.5, SubTiers: 6},
		{Name: "Platinum", NameFR: "Platine", MinMu: 26.0, SubTiers: 6},
		{Name: "Diamond", NameFR: "Diamant", MinMu: 29.0, SubTiers: 6},
		{Name: "Onyx", NameFR: "Onyx", MinMu: 31.0, SubTiers: 1},
	}
}

// hyperparam keys utilisées dans lusr_hyperparams_v2 pour persister/loader
// les seuils. Convention : "tier_boundary_<name_canonical_lowercase>".
const (
	tierBoundaryKeyPrefix = "tier_boundary_"
)

// TierBoundariesFromHyperparams lit les seuils depuis une map name→value
// (typiquement retournée par SkillV2Repo.LoadHyperparams). Retourne les
// defaults si aucun seuil n'est présent, sinon merge tier par tier.
//
// Cette fonction est PURE (no I/O) — le caller (typiquement service.SkillV2Service
// ou cmd/lusr_v2_replay) fait le LoadHyperparams puis appelle ça.
func TierBoundariesFromHyperparams(hp map[string]float64) []TierBoundary {
	out := DefaultTierBoundaries()
	for i := range out {
		key := tierBoundaryKeyPrefix + lowercase(out[i].Name)
		if v, ok := hp[key]; ok {
			out[i].MinMu = v
		}
	}
	return out
}

// InferTier retourne (tier_name, sub_tier_1based) pour un μ donné. sub_tier
// vaut 0 pour Onyx (rang unique). Returns nil tier if mu < first boundary
// (ne devrait pas arriver en pratique — μ < 0 invalide).
//
// Le sub-tier est calculé par bandes égales à l'intérieur du tier :
//
//	width = (next_boundary - this_boundary) / sub_tiers
//	sub = clamp((mu - this_boundary) / width + 1, 1, sub_tiers)
//
// Exemple : Or [23.5, 26.0[, 6 sous-tiers → width = 0.417
//
//	mu = 23.6 → sub = 1 (Or VI dans la nomenclature inverse Halo, mais on
//	expose 1-based ascendant pour rester orthogonal à l'affichage UI)
func InferTier(mu float64, boundaries []TierBoundary) (TierBoundary, int) {
	if len(boundaries) == 0 {
		return TierBoundary{}, 0
	}
	// Find the highest tier whose MinMu ≤ mu.
	idx := 0
	for i, b := range boundaries {
		if mu >= b.MinMu {
			idx = i
		}
	}
	tier := boundaries[idx]
	if tier.SubTiers <= 1 {
		return tier, 0
	}
	// Width = (next.MinMu - tier.MinMu) / sub_tiers ; pour le dernier tier
	// (sans next), on prend une largeur par défaut de 1.0 (proportionnel).
	var width float64 = 1.0
	if idx+1 < len(boundaries) {
		width = (boundaries[idx+1].MinMu - tier.MinMu) / float64(tier.SubTiers)
	}
	if width <= 0 {
		return tier, 1
	}
	sub := int((mu-tier.MinMu)/width) + 1
	if sub < 1 {
		sub = 1
	}
	if sub > tier.SubTiers {
		sub = tier.SubTiers
	}
	return tier, sub
}

// FormatTierLabel retourne le label FR complet (ex: "Or III") pour un μ.
func FormatTierLabel(mu float64, boundaries []TierBoundary) string {
	tier, sub := InferTier(mu, boundaries)
	if tier.Name == "" {
		return "Non classé"
	}
	if sub == 0 {
		return tier.NameFR
	}
	roman := map[int]string{1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}
	r, ok := roman[sub]
	if !ok {
		return tier.NameFR
	}
	return fmt.Sprintf("%s %s", tier.NameFR, r)
}

// lowercase : version ASCII rapide (pas d'unicode dans nos noms canoniques).
func lowercase(s string) string {
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
