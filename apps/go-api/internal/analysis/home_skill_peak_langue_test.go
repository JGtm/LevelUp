package analysis

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// Le libellé du palier du « Pic rang » de l'Accueil se déduit du code (Tier + SubTier) dans la langue de
// la requête, pour les deux titres : « Diamond 4 » (Halo 5) et « Or VI » (Halo Infinite) stockés en base
// ne sont jamais servis.
func TestBuildHomeSkillPeak_LibelleDeduitDuCodeDansLaLangueDeLaRequete(t *testing.T) {
	zero := 0
	badge := "/x.png"
	cases := []struct {
		name   string
		raw    domain.HomeSkillPeakRow
		fr, en string
	}{
		{"Halo 5 Diamant 4", domain.HomeSkillPeakRow{RatingValue: 1, Tier: "Diamond", SubTier: 4, MeasurementMatchesRemaining: &zero}, "Diamant IV", "Diamond IV"},
		{"Infinite Or 6", domain.HomeSkillPeakRow{RatingValue: 1700, Tier: "Gold", SubTier: 6, MeasurementMatchesRemaining: &zero}, "Or VI", "Gold VI"},
		{"Onyx", domain.HomeSkillPeakRow{RatingValue: 1850, Tier: "Onyx", MeasurementMatchesRemaining: &zero}, "Onyx", "Onyx"},
		{"Champion (Halo 5)", domain.HomeSkillPeakRow{RatingValue: 1579, Tier: "Champion", MeasurementMatchesRemaining: &zero}, "Champion", "Champion"},
	}
	for _, c := range cases {
		for locale, want := range map[string]string{"fr": c.fr, "en": c.en} {
			got := buildHomeSkillPeak(&c.raw, locale)
			if got == nil || got.TierLabel == nil || *got.TierLabel != want {
				t.Errorf("%s (%s) : %v, attendu %q", c.name, locale, got, want)
			}
		}
	}
	// Placement : aucun libellé de palier, le badge « non classé » suffit à garder le résumé.
	placement := domain.HomeSkillPeakRow{BadgeImageURL: &badge}
	got := buildHomeSkillPeak(&placement, "en")
	if got == nil || got.TierLabel != nil {
		t.Errorf("placement : %+v, attendu un résumé sans libellé de palier", got)
	}
}
