package profile

import (
	"fmt"

	"levelup/go-api/internal/sync"
)

// tier.go — conversion μ → TierState + calcul du prochain sub-tier.
//
// Pas de duplication des constantes : utilise sync.SkillTiers et
// sync.GetTierForRating comme source de vérité.

// romanI retourne le chiffre romain I à VI pour les sub-tiers.
var romanI = map[int]string{1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}

// TierFromMu construit le TierState pour un μ donné, ou empty si hors plage.
func TierFromMu(mu float64) TierState {
	tier, sub := sync.GetTierForRating(mu)
	if tier == nil {
		return TierState{}
	}
	lower, upper := subTierBounds(tier, sub)
	return TierState{
		Name:    tier.Name,
		NameFR:  tier.NameFR,
		SubTier: sub,
		Label:   formatLabel(tier.Name, sub),
		LowerMu: lower,
		UpperMu: upper,
	}
}

// NextTierFromMu retourne l'état du prochain sub-tier (ou tier suivant si on
// est sur le dernier sub-tier). TierState vide si déjà au max (Onyx).
//
// La sortie sert pour l'alerte LUSRTierApproach (gap = NextTier.LowerMu - μ).
func NextTierFromMu(mu float64) TierState {
	current := TierFromMu(mu)
	if current.IsEmpty() {
		// Hors plage → on prend le 1er sub-tier comme « cible » si possible.
		if len(sync.SkillTiers) > 0 {
			first := &sync.SkillTiers[0]
			return TierState{
				Name: first.Name, NameFR: first.NameFR, SubTier: 1,
				Label:   formatLabel(first.Name, 1),
				LowerMu: first.MinRating, UpperMu: subTierUpper(first, 1),
			}
		}
		return TierState{}
	}

	// Trouver le tier dans la slice.
	var tierIdx int = -1
	for i := range sync.SkillTiers {
		if sync.SkillTiers[i].Name == current.Name {
			tierIdx = i
			break
		}
	}
	if tierIdx < 0 {
		return TierState{}
	}
	tier := &sync.SkillTiers[tierIdx]

	// Si on n'est pas sur le dernier sub-tier → passer au sub suivant.
	if current.SubTier > 0 && current.SubTier < tier.SubTiers {
		return TierState{
			Name: tier.Name, NameFR: tier.NameFR, SubTier: current.SubTier + 1,
			Label:   formatLabel(tier.Name, current.SubTier+1),
			LowerMu: subTierLower(tier, current.SubTier+1),
			UpperMu: subTierUpper(tier, current.SubTier+1),
		}
	}

	// Sinon, passer au tier suivant (sub-tier 1).
	if tierIdx+1 >= len(sync.SkillTiers) {
		return TierState{} // déjà au max (Onyx)
	}
	next := &sync.SkillTiers[tierIdx+1]
	return TierState{
		Name: next.Name, NameFR: next.NameFR, SubTier: 1,
		Label:   formatLabel(next.Name, 1),
		LowerMu: next.MinRating,
		UpperMu: subTierUpper(next, 1),
	}
}

// subTierBounds retourne (lower, upper) pour (tier, sub).
// Sub=0 (tier sans sous-paliers comme Onyx) → bornes du tier complet.
func subTierBounds(tier *sync.SkillTier, sub int) (float64, float64) {
	if sub <= 0 || tier.SubTiers <= 1 {
		return tier.MinRating, tier.MaxRating
	}
	return subTierLower(tier, sub), subTierUpper(tier, sub)
}

func subTierLower(tier *sync.SkillTier, sub int) float64 {
	rangePerSub := (tier.MaxRating - tier.MinRating) / float64(tier.SubTiers)
	return tier.MinRating + rangePerSub*float64(sub-1)
}

func subTierUpper(tier *sync.SkillTier, sub int) float64 {
	rangePerSub := (tier.MaxRating - tier.MinRating) / float64(tier.SubTiers)
	return tier.MinRating + rangePerSub*float64(sub)
}

// formatLabel produit "Diamond III" / "Onyx" (sans sub si 0/1).
func formatLabel(tierName string, sub int) string {
	if sub <= 0 {
		return tierName
	}
	r, ok := romanI[sub]
	if !ok {
		return tierName
	}
	return fmt.Sprintf("%s %s", tierName, r)
}
