package relations

import "testing"

func TestCrossGameBadge(t *testing.T) {
	cases := []struct {
		name      string
		game      string
		matches   int
		wantBadge bool
	}{
		{"below_threshold", "Halo 5", CrossGameMinMatchesTogether - 1, false},
		{"at_threshold", "Halo 5", CrossGameMinMatchesTogether, true},
		{"above_threshold", "Halo 5", CrossGameMinMatchesTogether + 10, true},
		{"empty_game_name", "", 99, false},
		{"zero_matches", "Halo 5", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := CrossGameBadge(tc.game, tc.matches)
			if tc.wantBadge != (b != nil) {
				t.Fatalf("game=%q matches=%d → badge=%v want=%v", tc.game, tc.matches, b != nil, tc.wantBadge)
			}
			if b == nil {
				return
			}
			if b.LabelKey != "narrative.encounter.cross_game" {
				t.Errorf("LabelKey=%q", b.LabelKey)
			}
			if b.Style != BadgeStyleSolid {
				t.Errorf("Style=%q want solid", b.Style)
			}
			if got := b.Detail["game"]; got != tc.game {
				t.Errorf("Detail[game]=%v want %q", got, tc.game)
			}
			if got := b.Detail["matches_together"]; got != tc.matches {
				t.Errorf("Detail[matches_together]=%v want %d", got, tc.matches)
			}
		})
	}
}
