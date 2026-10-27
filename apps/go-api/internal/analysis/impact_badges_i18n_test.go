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
