package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func TestWindowBounds(t *testing.T) {
	loc := paris(t)
	// Un match exactement à now-7j sort de l'horizon courant et entre dans la période d'avant.
	ms := []Match{
		mk(loc, ago(7, 0)),
		mk(loc, ago(7, time.Second)),
		mk(loc, ago(14, 0)),
		mk(loc, ago(14, time.Second)),
		mk(loc, testNow),
	}
	resp := BuildSolo(ms, opts(loc))
	c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyMatchCount, ""), 7)
	if c.Matches != 2 { // now et now-7j+1s
		t.Errorf("matchs courants = %d, attendu 2", c.Matches)
	}
	if c.PrevMatches != 2 { // now-7j et now-14j+1s ; now-14j exclu
		t.Errorf("matchs d'avant = %d, attendu 2", c.PrevMatches)
	}
}

func TestCompareThreshold(t *testing.T) {
	loc := paris(t)
	tests := []struct {
		name      string
		cur, prev int
		wantPrev  bool
	}{
		{"10 et 10", 10, 10, true},
		{"9 courants", 9, 10, false},
		{"9 avant", 10, 9, false},
		{"0 avant", 10, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ms := many(loc, tc.prev, ago(10, -time.Hour))
			ms = append(ms, many(loc, tc.cur, ago(2, 0))...)
			ind := indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyKDA, "")
			c := horizon(t, ind, 7)
			if c.Value == nil {
				t.Fatalf("valeur courante absente")
			}
			if (c.PrevValue != nil) != tc.wantPrev || (c.Z != nil) != tc.wantPrev {
				t.Errorf("prev=%v z=%v, attendu présence=%v", c.PrevValue, c.Z, tc.wantPrev)
			}
		})
	}
}

func TestHorizonOrderAndMonths(t *testing.T) {
	loc := paris(t)
	resp := BuildSolo(many(loc, 3, ago(1, 0)), opts(loc))
	if len(resp.Months) != 12 || resp.Months[11] != "2026-09" || resp.Months[0] != "2025-10" {
		t.Errorf("mois = %v", resp.Months)
	}
	ind := indicatorOf(t, resp, domain.TrendsKeyWinRate, "")
	var days []int
	for _, c := range ind.Horizons {
		days = append(days, c.Days)
	}
	if len(days) != 4 || days[0] != 365 || days[1] != 90 || days[2] != 30 || days[3] != 7 {
		t.Errorf("ordre des horizons = %v", days)
	}
	if len(ind.Months) != 12 {
		t.Errorf("cellules mensuelles = %d", len(ind.Months))
	}
	if resp.Timezone != "Europe/Paris" || !resp.AsOf.Equal(testNow) {
		t.Errorf("timezone=%s as_of=%v", resp.Timezone, resp.AsOf)
	}
}

func TestMonthMinimumMatches(t *testing.T) {
	loc := paris(t)
	// Août : 4 matchs (sans valeur). Juillet : 5 matchs (avec valeur).
	ms := many(loc, 5, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	ms = append(ms, many(loc, 4, time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))...)
	ind := indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyWinRate, "")
	jul, aug := ind.Months[9], ind.Months[10]
	if aug.Matches != 4 || aug.Value != nil || aug.Z != nil {
		t.Errorf("août = %+v", aug)
	}
	if jul.Matches != 5 || jul.Value == nil {
		t.Errorf("juillet = %+v", jul)
	}
}

func TestZScoreAndZeroStd(t *testing.T) {
	if got := sdOrOne(0); got != 1 {
		t.Errorf("sdOrOne(0) = %v", got)
	}
	if got := sdOrOne(2.5); got != 2.5 {
		t.Errorf("sdOrOne(2.5) = %v", got)
	}
	tests := []struct {
		better int
		diff   float64
		sd     float64
		want   float64
	}{
		{1, 4, 2, 2},
		{-1, 4, 2, -2},
		{0, 4, 2, 0},
		{1, 4, sdOrOne(0), 4},
	}
	for _, tc := range tests {
		if got := zScore(tc.better, tc.diff, tc.sd); !near(got, tc.want) {
			t.Errorf("zScore(%d,%v,%v) = %v, attendu %v", tc.better, tc.diff, tc.sd, got, tc.want)
		}
	}
}

func TestZEndToEndZeroStd(t *testing.T) {
	loc := paris(t)
	// Un seul mois porte une valeur : écart-type nul remplacé par 1, donc z = 1 x (1 - 0).
	ms := many(loc, 10, ago(10, -time.Hour), outcome(canonical.OutcomeLoss))
	ms = append(ms, many(loc, 10, ago(2, 0), outcome(canonical.OutcomeWin))...)
	ind := indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyWinRate, "")
	c := horizon(t, ind, 7)
	if c.Z == nil || !near(*c.Z, 1.0) {
		t.Fatalf("z horizon = %v", c.Z)
	}
	sept := ind.Months[11]
	if sept.Z == nil || !near(*sept.Z, 0) {
		t.Errorf("z du seul mois = %v, attendu 0", sept.Z)
	}
}

func TestBetterZeroGivesZeroZ(t *testing.T) {
	loc := paris(t)
	prev := many(loc, 10, ago(10, -time.Hour), func(m *Match) { m.PowerKills = 1 })
	cur := many(loc, 10, ago(2, 0), func(m *Match) { m.PowerKills = 5 })
	resp := BuildSolo(append(prev, cur...), opts(loc))
	c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyPowerWeaponShare, ""), 7)
	if c.Value == nil || c.PrevValue == nil || near(*c.Value, *c.PrevValue) {
		t.Fatalf("valeurs = %v / %v", c.Value, c.PrevValue)
	}
	if c.Z == nil || *c.Z != 0 {
		t.Errorf("z = %v, attendu 0", c.Z)
	}
}

func TestAggregateKDANotQuotient(t *testing.T) {
	loc := paris(t)
	ms := []Match{
		mk(loc, ago(1, 0), func(m *Match) { m.Kills, m.Assists, m.Deaths = 10, 3, 5 }),
		mk(loc, ago(1, time.Minute), func(m *Match) { m.Kills, m.Assists, m.Deaths = 2, 0, 3 }),
	}
	c := horizon(t, indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyKDA, ""), 7)
	// ((12 + 3/3) - 8) / 2 = 2,5 ; un quotient donnerait 13/8.
	if c.Value == nil || !near(*c.Value, 2.5) {
		t.Errorf("kda = %v, attendu 2,5", c.Value)
	}
}

func TestAccuracyAndShares(t *testing.T) {
	loc := paris(t)
	ms := []Match{
		mk(loc, ago(1, 0), func(m *Match) { m.ShotsFired, m.ShotsHit, m.HeadshotKills, m.Kills = 100, 40, 5, 5 }),
		mk(loc, ago(1, time.Minute), func(m *Match) { m.ShotsFired, m.ShotsHit, m.HeadshotKills, m.Kills = 0, 0, 5, 5 }),
	}
	resp := BuildSolo(ms, opts(loc))
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyAccuracy, ""), 7); c.Value == nil || !near(*c.Value, 0.4) {
		t.Errorf("précision = %v", c.Value)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyHeadshotShare, ""), 7); c.Value == nil || !near(*c.Value, 1.0) {
		t.Errorf("part à la tête = %v", c.Value)
	}
	// Aucun tir, aucun frag : lignes omises.
	none := BuildSolo([]Match{mk(loc, ago(1, 0), func(m *Match) { m.Kills, m.ShotsFired = 0, 0 })}, opts(loc))
	if hasIndicator(none, domain.TrendsKeyAccuracy) || hasIndicator(none, domain.TrendsKeyHeadshotShare) {
		t.Errorf("lignes sans valeur non omises")
	}
}
