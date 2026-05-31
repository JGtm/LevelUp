package skill_v2

// tier.go : mapping μ → tier Halo (Bronze..Onyx) pour le LUSR v2.
//
// Le modèle skill (μ, σ) vit en interne sur l'échelle native TrueSkill
// (μ_0 = 25). La grille d'affichage UI est la grille Halo classique
// Bronze → Argent → Or → Platine → Diamant → Onyx avec 6 sous-tiers
// par tier sauf Onyx (rang unique).
//
// Grille Phase 3e v2 (post-feedback utilisateur) — calibrée pour que les
// tiers v2 ressemblent aux ranks CSR observés en jeu. Largeurs des tiers
// NON uniformes : Bronze large (absorbe les outliers bas, pop peu dense),
// Or large (le gros bucket de la population), Platine étroit (zone de
// promotion), Diamant + Onyx larges (pour pousser les top players plus haut
// même si leur μ relatif est modeste).
//
//	Tier     μ range      Largeur   Couvre approx. (population)
//	Bronze   [0, 21[      21        bottom ~10% (queue large, peu dense)
//	Argent   [21, 22[     1         10-15% (promotion étroite)
//	Or       [22, 25[     3         15-65% (LE gros bucket)
//	Platine  [25, 25.8[   0.8       65-70% (promotion étroite)
//	Diamant  [25.8, 27[   1.2       70-90%
//	Onyx     [27, ∞[      ouvert    top ~10%
//
// Justification : LUSR v2 mesure le skill social (matches non classés),
// dont la pop est plus large que le ranked CSR. Onyx à top 10% (vs top 1 %
// classique CSR) est défendable parce qu'on inclut beaucoup de joueurs
// occasionnels dans la population.
//
// Phase 5 (batch ré-estimation) pourra ré-écrire ces seuils dans
// lusr_hyperparams_v2 avec un source = "batch_YYYY_MM". TierBoundariesFromHyperparams
// fera le merge defaults ↔ overrides.

// TierBoundary décrit un palier de la grille tier.
type TierBoundary struct {
	Name     string  // identifiant canonique EN (Bronze, Silver, Gold, ...)
	NameFR   string  // libellé FR (Bronze, Argent, Or, ...)
	MinMu    float64 // borne inférieure (inclusive)
	SubTiers int     // nombre de sous-tiers (6 partout sauf Onyx = 1)
}

// DefaultTierBoundaries retourne la grille Phase 3e v2, calibrée sur la
// distribution observée + cross-référence avec les ranks CSR connus des
// joueurs trackés. Voir l'en-tête du fichier pour le rationale.
//
// Bornes strictement ordonnées par MinMu croissant. Grille ouverte :
// Bronze couvre [-∞, MinMu_Silver) ; Onyx couvre [MinMu_Onyx, +∞).
func DefaultTierBoundaries() []TierBoundary {
	// SubTiers calibrés (étude volatilité .ai/thought_log.md [2026-05-31]) pour
	// UNIFORMISER la largeur d'un sous-palier (~0.33–0.5 μ) SANS toucher les
	// bornes (la calibration tier↔CSR validée — Madina Diamant, etc. — est
	// préservée). Les tiers étroits (Argent 1.0 μ, Platine 0.8 μ, Diamant 1.2 μ)
	// avaient des sous-paliers ridiculement fins (0.13–0.2 μ) qu'un seul match
	// traversait en rafale ; on réduit leur nombre de sous-paliers pour élargir
	// chaque bande et limiter le lag sous l'hystérésis d'affichage.
	//
	//	Tier     bornes        sous-paliers   largeur sous-palier
	//	Bronze   [0, 21[       6              3.500 μ
	//	Argent   [21, 22[      3              0.333 μ
	//	Or       [22, 25[      6              0.500 μ
	//	Platine  [25, 25.8[    2              0.400 μ
	//	Diamant  [25.8, 27[    3              0.400 μ
	//	Onyx     [27, ∞[       1              ouvert
	return []TierBoundary{
		{Name: "Bronze", NameFR: "Bronze", MinMu: 0, SubTiers: 6},
		{Name: "Silver", NameFR: "Argent", MinMu: 21.0, SubTiers: 3},
		{Name: "Gold", NameFR: "Or", MinMu: 22.0, SubTiers: 6},
		{Name: "Platinum", NameFR: "Platine", MinMu: 25.0, SubTiers: 2},
		{Name: "Diamond", NameFR: "Diamant", MinMu: 25.8, SubTiers: 3},
		{Name: "Onyx", NameFR: "Onyx", MinMu: 27.0, SubTiers: 1},
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
// Délègue à FormatTierSubLabel (display_smoothing.go) après résolution du tier.
func FormatTierLabel(mu float64, boundaries []TierBoundary) string {
	tier, sub := InferTier(mu, boundaries)
	return FormatTierSubLabel(tier, sub)
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
