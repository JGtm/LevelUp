package relations

import (
	"testing"
	"time"
)

func ratePtr(v float64) *float64 { return &v }

func findBadge(badges []Badge, labelKey string) *Badge {
	for i := range badges {
		if badges[i].LabelKey == labelKey {
			return &badges[i]
		}
	}
	return nil
}

func TestCategorize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		s    RelationStats
		want string
	}{
		{"only ally", RelationStats{TeammateMatches: 5}, CategoryAlly},
		{"only enemy", RelationStats{EnemyMatches: 5}, CategoryEnemy},
		{"mixed", RelationStats{TeammateMatches: 3, EnemyMatches: 2}, CategoryMixed},
		{"tie zero", RelationStats{}, CategoryAlly},
		{"enemy dominates but no teammate", RelationStats{EnemyMatches: 9}, CategoryEnemy},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Categorize(tc.s); got != tc.want {
				t.Fatalf("Categorize=%q want %q", got, tc.want)
			}
		})
	}
}

func TestDuoGagnantBadge_Thresholds(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cases := []struct {
		name string
		s    RelationStats
		want bool
	}{
		{"exactly at thresholds", RelationStats{TotalMatches: 10, TeammateMatches: 10, TeammateWinRate: ratePtr(0.60)}, true},
		{"win rate just below", RelationStats{TotalMatches: 10, TeammateMatches: 10, TeammateWinRate: ratePtr(0.59)}, false},
		{"matches just below", RelationStats{TotalMatches: 9, TeammateMatches: 9, TeammateWinRate: ratePtr(0.9)}, false},
		{"nil win rate", RelationStats{TotalMatches: 12, TeammateMatches: 12}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := findBadge(ComputeBadges(tc.s, now), "narrative.encounter.duo_gagnant")
			if (b != nil) != tc.want {
				t.Fatalf("duo_gagnant present=%v want %v", b != nil, tc.want)
			}
			if b != nil && b.Style != BadgeStyleSolid {
				t.Fatalf("duo_gagnant style=%q want solid", b.Style)
			}
		})
	}
}

func TestCameleonBadge_Thresholds(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cases := []struct {
		name string
		s    RelationStats
		want bool
	}{
		// min(4,6)/10 = 0.40 → qualifie
		{"at ratio threshold", RelationStats{TotalMatches: 10, TeammateMatches: 4, EnemyMatches: 6}, true},
		// min(3,7)/10 = 0.30 → non
		{"below ratio", RelationStats{TotalMatches: 10, TeammateMatches: 3, EnemyMatches: 7}, false},
		// total below 10
		{"below total", RelationStats{TotalMatches: 9, TeammateMatches: 4, EnemyMatches: 5}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := findBadge(ComputeBadges(tc.s, now), "narrative.encounter.cameleon")
			if (b != nil) != tc.want {
				t.Fatalf("cameleon present=%v want %v", b != nil, tc.want)
			}
		})
	}
}

func TestDeLongueDateBadge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, -7, 0) // 7 mois
	recent := now.AddDate(0, -2, 0)
	cases := []struct {
		name string
		s    RelationStats
		want bool
	}{
		{"old first seen", RelationStats{TotalMatches: 5, FirstSeen: &old}, true},
		{"recent but many matches", RelationStats{TotalMatches: 80, FirstSeen: &recent}, true},
		{"recent few matches", RelationStats{TotalMatches: 5, FirstSeen: &recent}, false},
		{"no first seen few matches", RelationStats{TotalMatches: 5}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := findBadge(ComputeBadges(tc.s, now), "narrative.encounter.de_longue_date")
			if (b != nil) != tc.want {
				t.Fatalf("de_longue_date present=%v want %v", b != nil, tc.want)
			}
		})
	}
}

func TestRecrueBadge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	veryRecent := now.AddDate(0, 0, -10)
	tooOld := now.AddDate(0, 0, -40)
	cases := []struct {
		name string
		s    RelationStats
		want bool
	}{
		{"recent enough matches", RelationStats{TotalMatches: 4, FirstSeen: &veryRecent}, true},
		{"recent too few matches", RelationStats{TotalMatches: 3, FirstSeen: &veryRecent}, false},
		{"old", RelationStats{TotalMatches: 10, FirstSeen: &tooOld}, false},
		{"no first seen", RelationStats{TotalMatches: 10}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := findBadge(ComputeBadges(tc.s, now), "narrative.encounter.recrue")
			if (b != nil) != tc.want {
				t.Fatalf("recrue present=%v want %v", b != nil, tc.want)
			}
		})
	}
}

func TestProieFavoriteBadge(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cases := []struct {
		name string
		s    RelationStats
		want bool
	}{
		{"above ratio enough matches", RelationStats{EnemyMatches: 6, DuelRatio: ratePtr(1.6)}, true},
		{"at ratio (strict >) fails", RelationStats{EnemyMatches: 6, DuelRatio: ratePtr(1.5)}, false},
		{"too few enemy matches", RelationStats{EnemyMatches: 5, DuelRatio: ratePtr(3.0)}, false},
		{"nil duel ratio", RelationStats{EnemyMatches: 10}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := findBadge(ComputeBadges(tc.s, now), "narrative.encounter.proie_favorite")
			if (b != nil) != tc.want {
				t.Fatalf("proie_favorite present=%v want %v", b != nil, tc.want)
			}
		})
	}
}

func TestComputeCounts(t *testing.T) {
	t.Parallel()
	rels := []RelationStats{
		{TotalMatches: 25, TeammateMatches: 12, EnemyMatches: 13}, // core
		{TotalMatches: 22, TeammateMatches: 20, EnemyMatches: 2},  // core : duo-partenaire (enemy<3 OK depuis retrait du seuil)
		{TotalMatches: 5, TeammateMatches: 5},                     // ally only, pas core (total<20)
		{TotalMatches: 5, EnemyMatches: 5},                        // enemy only
		{TotalMatches: 20, TeammateMatches: 2, EnemyMatches: 18},  // pas core (teammate<3)
	}
	c := ComputeCounts(rels)
	if c.DistinctPlayers != 5 {
		t.Fatalf("distinct=%d want 5", c.DistinctPlayers)
	}
	if c.AlliesCount != 4 {
		t.Fatalf("allies=%d want 4", c.AlliesCount)
	}
	if c.RivalsCount != 4 {
		t.Fatalf("rivals=%d want 4", c.RivalsCount)
	}
	if c.CoreCount != 2 {
		t.Fatalf("core=%d want 2 (inclut le duo-partenaire enemy<3)", c.CoreCount)
	}
}

func TestSelectTopAlly(t *testing.T) {
	t.Parallel()
	rels := []RelationStats{
		{Gamertag: "Below8", TeammateMatches: 7, TeammateWinRate: ratePtr(0.99)}, // ineligible
		{Gamertag: "A", TeammateMatches: 8, TeammateWinRate: ratePtr(0.70)},
		{Gamertag: "B", TeammateMatches: 12, TeammateWinRate: ratePtr(0.70)}, // tiebreak: more matches
		{Gamertag: "C", TeammateMatches: 10, TeammateWinRate: ratePtr(0.65)},
	}
	got := SelectTopAlly(rels)
	if got == nil || got.Gamertag != "B" {
		t.Fatalf("top ally=%v want B", got)
	}
}

func TestSelectTopAlly_NoCandidate(t *testing.T) {
	t.Parallel()
	rels := []RelationStats{{Gamertag: "X", TeammateMatches: 3, TeammateWinRate: ratePtr(1.0)}}
	if got := SelectTopAlly(rels); got != nil {
		t.Fatalf("expected nil top ally, got %v", got)
	}
}

func TestSelectTopNemesis(t *testing.T) {
	t.Parallel()
	rels := []RelationStats{
		{Gamertag: "Below8", EnemyMatches: 7, EnemyWinRate: ratePtr(0.01)}, // ineligible
		{Gamertag: "A", EnemyMatches: 8, EnemyWinRate: ratePtr(0.30)},
		{Gamertag: "B", EnemyMatches: 10, EnemyWinRate: ratePtr(0.20)}, // worst win rate
		{Gamertag: "C", EnemyMatches: 9, EnemyWinRate: ratePtr(0.40)},
	}
	got := SelectTopNemesis(rels)
	if got == nil || got.Gamertag != "B" {
		t.Fatalf("top nemesis=%v want B", got)
	}
}

func TestSelectTopNemesis_TiebreakDuelRatio(t *testing.T) {
	t.Parallel()
	rels := []RelationStats{
		{Gamertag: "Hi", EnemyMatches: 8, EnemyWinRate: ratePtr(0.25), DuelRatio: ratePtr(2.0)},
		{Gamertag: "Lo", EnemyMatches: 8, EnemyWinRate: ratePtr(0.25), DuelRatio: ratePtr(0.5)}, // worse duel
	}
	got := SelectTopNemesis(rels)
	if got == nil || got.Gamertag != "Lo" {
		t.Fatalf("top nemesis=%v want Lo (worse duel ratio)", got)
	}
}

func TestEncounterBadgesReused(t *testing.T) {
	t.Parallel()
	now := time.Now()
	// ally_plus : win rate >= 0.65, ally >= 2. Badge de rencontre → désormais solid.
	s := RelationStats{TotalMatches: 5, TeammateMatches: 5, TeammateWinRate: ratePtr(0.8)}
	b := findBadge(ComputeBadges(s, now), "narrative.encounter.ally_plus")
	if b == nil {
		t.Fatal("expected ally_plus encounter badge")
	}
	if b.Style != BadgeStyleSolid {
		t.Fatalf("ally_plus style=%q want solid", b.Style)
	}
	if b.ColorToken != "narrative-encounter-ally-plus" {
		t.Fatalf("ally_plus color_token=%q", b.ColorToken)
	}
}
