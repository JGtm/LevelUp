package analysis

import (
	"testing"

	"levelup/go-api/internal/i18n"
)

// Les badges d'impact portent leur nom dans toutes les langues de l'application (catalogue des
// messages) : ils étaient français seuls.
func TestComputeMatchImpactFull_BadgesDansToutesLesLangues(t *testing.T) {
	badges := ComputeMatchImpactFull(MatchImpactInput{
		Events: []ImpactEvent{
			{TimeMS: 1000, EventType: "kill", ActorXUID: "a"},
			{TimeMS: 2000, EventType: "death", ActorXUID: "b"},
		},
		Participants: []ParticipantSnap{
			{XUID: "a", Outcome: 2, Kills: 12, Deaths: 1, Assists: 2},
			{XUID: "b", Outcome: 2, Kills: 2, Deaths: 3, Assists: 6},
		},
	})
	if len(badges) == 0 {
		t.Fatal("aucun badge produit")
	}
	for _, b := range badges {
		for _, l := range i18n.Locales() {
			if b.Badge[l] == "" {
				t.Errorf("badge %s : nom absent en %q", b.BadgeKey, l)
			}
		}
	}
	for _, b := range badges {
		if b.BadgeKey == "first_blood" {
			if b.Badge.Pick(i18n.LocaleFR) != "Premier sang" || b.Badge.Pick(i18n.LocaleEN) != "First blood" {
				t.Errorf("first_blood : %v", b.Badge)
			}
			return
		}
	}
	t.Error("first_blood absent")
}

func TestBuildSkillTierLabel_DansLaLangueDemandee(t *testing.T) {
	tier := "Gold"
	sub := 3
	cases := map[i18n.Locale]string{i18n.LocaleFR: "Or III", i18n.LocaleEN: "Gold III"}
	for loc, want := range cases {
		got := BuildSkillTierLabel(&tier, &sub, loc)
		if got == nil || *got != want {
			t.Errorf("langue %s : %v, attendu %q", loc, got, want)
		}
	}
}

// TestSkillTierDisplayLabel_DeduitDuCodeDansLaLangueDemandee : le libellé d'un palier lu en base se
// déduit du code (tier anglais + sous-palier), jamais du libellé stocké ; un palier sans sous-palier
// valide garde son nom seul, le placement et le vide ne donnent aucun libellé.
func TestSkillTierDisplayLabel_DeduitDuCodeDansLaLangueDemandee(t *testing.T) {
	sub := func(n int) *int { return &n }
	cases := []struct {
		name   string
		tier   string
		sub    *int
		fr, en string
	}{
		{"diamant stocke en anglais arabe", "Diamond", sub(4), "Diamant IV", "Diamond IV"},
		{"or stocke en francais", "Gold", sub(6), "Or VI", "Gold VI"},
		{"casse libre", "PLATINUM", sub(1), "Platine I", "Platinum I"},
		{"onyx sans sous-palier", "Onyx", sub(0), "Onyx", "Onyx"},
		{"palier a sous-palier nul (CSR Diamant 0)", "Diamond", sub(0), "Diamant", "Diamond"},
		{"sous-palier absent", "Gold", nil, "Or", "Gold"},
		{"palier inconnu du catalogue", "Champion", sub(0), "Champion", "Champion"},
	}
	for _, c := range cases {
		for loc, want := range map[i18n.Locale]string{i18n.LocaleFR: c.fr, i18n.LocaleEN: c.en} {
			got := SkillTierDisplayLabel(c.tier, c.sub, loc)
			if got == nil || *got != want {
				t.Errorf("%s (%s) : %v, attendu %q", c.name, loc, got, want)
			}
		}
	}
	for _, tier := range []string{"", "  ", "Placement", "placement"} {
		if got := SkillTierDisplayLabel(tier, sub(0), i18n.LocaleFR); got != nil {
			t.Errorf("palier %q : libellé %q, attendu nil", tier, *got)
		}
	}
	if got := SkillTierDisplayLabelPtr(nil, sub(1), i18n.LocaleFR); got != nil {
		t.Errorf("palier nil : %q, attendu nil", *got)
	}
}
