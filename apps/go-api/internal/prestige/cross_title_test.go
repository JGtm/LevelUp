package prestige

import "testing"

// TestCreditTitlesFor_MonoTitleToday : le point d'extension cross-titre est
// strictement mono-titre aujourd'hui (titre primaire uniquement). Garde-fou : si
// quelqu'un élargit creditTitlesFor sans décider la politique PP produit+UX, ce
// test échoue et force la discussion (cf. PLAN_CROSS_TITLE_ARCS_BACKEND Phase 3).
func TestCreditTitlesFor_MonoTitleToday(t *testing.T) {
	c := Challenge{TitleSlug: "halo_infinite"}
	got := creditTitlesFor(c)
	if len(got) != 1 || got[0] != "halo_infinite" {
		t.Errorf("creditTitlesFor = %v, want [halo_infinite] (comportement mono-titre)", got)
	}
}
