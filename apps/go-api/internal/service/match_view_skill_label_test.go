package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

func skillRankRawFor(tier string, sub int, stored string) *domain.SkillRankRaw {
	return &domain.SkillRankRaw{RatingType: "CSR", Tier: &tier, SubTier: &sub, TierLabel: &stored}
}

// Le libellé de palier de la vue du match se déduit du code dans la langue de la requête : le libellé
// stocké (« Diamond 4 », « Or VI ») n'est jamais servi.
func TestLabelSkillRank_DansLaLangueDeLaRequete(t *testing.T) {
	cases := []struct {
		name   string
		tier   string
		sub    int
		stored string
		fr, en string
	}{
		{name: "Halo 5 stocke en anglais arabe", tier: "Diamond", sub: 4, stored: "Diamond 4", fr: "Diamant IV", en: "Diamond IV"},
		{name: "Halo Infinite stocke en francais", tier: "Gold", sub: 6, stored: "Or VI", fr: "Or VI", en: "Gold VI"},
		{name: "Onyx", tier: "Onyx", sub: 0, stored: "Onyx 1850", fr: "Onyx", en: "Onyx"},
		{name: "placement", tier: "Placement", sub: 0, stored: "Placement (4 restants)", fr: "Placement", en: "Placement"},
	}
	for _, c := range cases {
		for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: c.fr, i18n.LocaleEN: c.en} {
			r := skillRankRawFor(c.tier, c.sub, c.stored)
			labelSkillRank(i18n.WithContext(context.Background(), loc), r)
			if r.TierLabel == nil || *r.TierLabel != want {
				t.Errorf("%s (%s) : TierLabel = %v, attendu %q", c.name, loc, r.TierLabel, want)
			}
		}
	}
	labelSkillRank(context.Background(), nil) // sans effet
	// Table par joueur (CSR de tous les participants).
	m := map[string]*domain.SkillRankRaw{"a": skillRankRawFor("Platinum", 5, "Platinum 5"), "b": nil}
	labelSkillRanks(i18n.WithContext(context.Background(), i18n.LocaleFR), m)
	if got := m["a"].TierLabel; got == nil || *got != "Platine V" {
		t.Errorf("participant a : %v, attendu Platine V", got)
	}
}

func TestBuildRankBlock_ServiLeLibelleDeduitDuCode(t *testing.T) {
	r := skillRankRawFor("Diamond", 4, "")
	labelSkillRank(i18n.WithContext(context.Background(), i18n.LocaleEN), r)
	rank := buildRankBlock(r, nil)
	if rank.TierLabel == nil || *rank.TierLabel != "Diamond IV" {
		t.Errorf("rang du header en : %v, attendu Diamond IV", rank.TierLabel)
	}
}

func TestLoadOneFriendExtras_LibelleDansLaLangueDeLaRequete(t *testing.T) {
	t.Parallel()
	for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: "Diamant IV", i18n.LocaleEN: "Diamond IV"} {
		tier, sub, stored := "Diamond", 4, "Diamond 4"
		repo := &mockFriendRepo{rankResult: &domain.SkillRankRaw{RatingType: "CSR", Tier: &tier, SubTier: &sub, TierLabel: &stored}}
		res, _ := loadOneFriendExtras(i18n.WithContext(context.Background(), loc), repo, "match-1", "", "xuid-1", nil)
		if res == nil || res.SkillRank == nil || res.SkillRank.TierLabel == nil || *res.SkillRank.TierLabel != want {
			t.Errorf("%s : %+v, attendu %q", loc, res, want)
		}
	}
}
