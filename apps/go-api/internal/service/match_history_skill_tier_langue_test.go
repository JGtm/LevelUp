package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// Le libellé de palier servi par l'Explorer se déduit du code (palier anglais + sous-palier) dans la
// langue de la requête : la colonne stockée (« Diamond 4 » pour Halo 5, « Or VI » pour Halo Infinite)
// n'est jamais servie.

func skillTierRaw(id, tier string, sub int, stored string) domain.MatchHistoryRawRow {
	r := briefingRaw(id, 0, domain.OutcomeWin, 10, 5, 2, 60, "map1", "Aquarius", "Slayer", "Arène")
	r.SkillRatingType = strPtr("CSR")
	r.SkillTier = tier
	r.SubTier = intPtr(sub)
	r.SkillStoredLabel = strPtr(stored)
	return r
}

func TestEnrichRow_LibelleDePalierDeduitDuCodeDansLaLangueDeLaRequete(t *testing.T) {
	cases := []struct {
		name   string
		tier   string
		sub    int
		stored string
		fr, en string
	}{
		{"Halo 5 stocke en anglais arabe", "Diamond", 4, "Diamond 4", "Diamant IV", "Diamond IV"},
		{"Halo Infinite stocke en francais", "Gold", 6, "Or VI", "Or VI", "Gold VI"},
		{"Platine stocke en francais", "Platinum", 1, "Platine I", "Platine I", "Platinum I"},
	}
	for _, c := range cases {
		raw := skillTierRaw("m1", c.tier, c.sub, c.stored)
		for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: c.fr, i18n.LocaleEN: c.en} {
			got := enrichRow(raw, nil, rowFormatters{}, loc).SkillTierLabel
			if got == nil || *got != want {
				t.Errorf("%s (%s) : SkillTierLabel = %v, attendu %q", c.name, loc, got, want)
			}
		}
	}
}

func TestEnrichRow_PlacementNeSertAucunLibelleDePalier(t *testing.T) {
	raw := skillTierRaw("m1", "Placement", 0, "Placement (4 restants)")
	for _, loc := range []i18n.Locale{i18n.LocaleFR, i18n.LocaleEN} {
		if got := enrichRow(raw, nil, rowFormatters{}, loc).SkillTierLabel; got != nil {
			t.Errorf("%s : placement servi comme palier %q, attendu nil", loc, *got)
		}
	}
}

func TestScopePeakRanks_LibelleDansLaLangueDeLaRequete(t *testing.T) {
	scope := []domain.MatchHistoryRawRow{
		skillTierRaw("a", "Diamond", 1, "Diamond 1"),
		skillTierRaw("b", "Diamond", 4, "Diamond 4"),
		skillTierRaw("c", "Platinum", 6, "Platinum 6"),
	}
	for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: "Diamant IV", i18n.LocaleEN: "Diamond IV"} {
		peaks := scopePeakRanks(scope, loc)
		if len(peaks) != 1 || peaks[0].TierLabel != want {
			t.Errorf("%s : pic = %+v, attendu %q", loc, peaks, want)
		}
	}
}

func TestBuildBriefingRanked_PaliersDansLaLangueDeLaRequete(t *testing.T) {
	var scope []domain.MatchHistoryRawRow
	for i := 0; i < 12; i++ {
		tier, sub, stored := "Gold", 6, "Or VI"
		if i == 11 { // le plus ancien
			tier, sub, stored = "Platinum", 1, "Platine I"
		}
		r := skillTierRaw("m"+string(rune('a'+i)), tier, sub, stored)
		r.StartTime = briefingRaw("x", i, domain.OutcomeWin, 1, 1, 1, 1, "m", "m", "p", "q").StartTime
		r.PlaylistGroup = strPtr("ranked")
		r.RatingValue = float64Ptr(float64(100 + i))
		scope = append(scope, r)
	}
	for loc, want := range map[i18n.Locale][2]string{
		i18n.LocaleFR: {"Platine I", "Or VI"},
		i18n.LocaleEN: {"Platinum I", "Gold VI"},
	} {
		got := buildBriefingRanked(i18n.WithContext(context.Background(), loc), scope)
		if got == nil || len(got.Kinds) != 1 {
			t.Fatalf("%s : %+v", loc, got)
		}
		k := got.Kinds[0]
		if k.TierStartLabel == nil || *k.TierStartLabel != want[0] || k.TierEndLabel == nil || *k.TierEndLabel != want[1] {
			t.Errorf("%s : paliers %v -> %v, attendu %q -> %q", loc, k.TierStartLabel, k.TierEndLabel, want[0], want[1])
		}
	}
}
