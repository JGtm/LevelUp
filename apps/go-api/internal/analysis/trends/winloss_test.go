package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func TestPearsonByHand(t *testing.T) {
	// x = 1..5, y = 2,1,4,3,5 : sxy = 8, sxx = 10, syy = 10 -> r = 0,8.
	r, ok := pearson([]float64{1, 2, 3, 4, 5}, []float64{2, 1, 4, 3, 5})
	if !ok || !near(r, 0.8) {
		t.Errorf("r = %v (%v), attendu 0,8", r, ok)
	}
	if _, ok := pearson([]float64{1, 1, 1}, []float64{0, 1, 0}); ok {
		t.Errorf("série constante : r doit être indéfini")
	}
	if _, ok := pearson([]float64{1}, []float64{1}); ok {
		t.Errorf("un point : r doit être indéfini")
	}
}

func TestWinLossBelowThreshold(t *testing.T) {
	loc := paris(t)
	ms := many(loc, 15, ago(1, 0), outcome(canonical.OutcomeWin))
	ms = append(ms, many(loc, 14, ago(1, time.Hour), outcome(canonical.OutcomeLoss))...)
	ms = append(ms, many(loc, 5, ago(1, 2*time.Hour), outcome(canonical.OutcomeTie))...)
	for _, b := range BuildSolo(ms, opts(loc)).WinLoss {
		if b.Days == 7 && (b.Matches != 29 || b.Required != 30 || len(b.Rows) != 0) {
			t.Errorf("bloc 7 j = %+v, attendu 29 matchs sans ligne", b)
		}
	}
}

func TestWinLossAboveThreshold(t *testing.T) {
	loc := paris(t)
	wins := many(loc, 15, ago(1, 0), outcome(canonical.OutcomeWin), func(m *Match) { m.Kills = 10 })
	losses := many(loc, 15, ago(1, time.Hour), outcome(canonical.OutcomeLoss), func(m *Match) { m.Kills = 4 })
	resp := BuildSolo(append(wins, losses...), opts(loc))
	var blk domain.TrendsWinLossBlock
	for _, b := range resp.WinLoss {
		if b.Days == 7 {
			blk = b
		}
	}
	if blk.Matches != 30 || len(blk.Rows) != 1 {
		t.Fatalf("bloc 7 j = %+v", blk)
	}
	// frags : moyenne 7, écart-type 3 -> z_win = 1, z_loss = -1, r = 1 ;
	// morts et assistances constantes : corrélation indéfinie, lignes omises.
	row := blk.Rows[0]
	if row.Key != domain.TrendsKeyKills || row.Group != domain.TrendsWinLossStats {
		t.Fatalf("ligne = %+v", row)
	}
	if !near(row.WinMean, 10) || !near(row.LossMean, 4) || !near(row.ZWin, 1) || !near(row.ZLoss, -1) || !near(row.R, 1) {
		t.Errorf("ligne = %+v", row)
	}
	if row.Matches != 30 {
		t.Errorf("valeurs = %d", row.Matches)
	}
}

func TestWinLossSkipsMatchesWithoutValue(t *testing.T) {
	loc := paris(t)
	// 30 décidés mais 29 portent un score de performance : la ligne est omise.
	ms := many(loc, 15, ago(1, 0), outcome(canonical.OutcomeWin), func(m *Match) { m.PerformanceScore = f64(70) })
	ms = append(ms, many(loc, 15, ago(1, time.Hour), outcome(canonical.OutcomeLoss), func(m *Match) { m.PerformanceScore = f64(30) })...)
	ms[0].PerformanceScore = nil
	for _, b := range BuildSolo(ms, opts(loc)).WinLoss {
		for _, r := range b.Rows {
			if r.Key == domain.TrendsKeyPerformanceScore {
				t.Errorf("ligne avec 29 valeurs non omise : %+v", r)
			}
		}
	}
}

func TestSortWinLoss(t *testing.T) {
	rows := []domain.TrendsWinLossRow{
		{Key: "mmr_gap", Group: domain.TrendsWinLossContext, R: 0.9},
		{Key: "kda", Group: domain.TrendsWinLossComposite, R: 0.2},
		{Key: "deaths", Group: domain.TrendsWinLossStats, R: -0.6},
		{Key: "kills", Group: domain.TrendsWinLossStats, R: 0.3},
		{Key: "performance_score", Group: domain.TrendsWinLossComposite, R: -0.5},
	}
	sortWinLoss(rows)
	want := []string{"deaths", "kills", "performance_score", "kda", "mmr_gap"}
	for i, k := range want {
		if rows[i].Key != k {
			t.Errorf("rang %d = %s, attendu %s", i, rows[i].Key, k)
		}
	}
}

func TestFromCanonical(t *testing.T) {
	loc := paris(t)
	truth, falsity := true, false
	sec := 120
	dur := 600
	placement := 2
	grp := "ranked-arena"
	val := 1200.0
	pair := "Arena:Slayer"
	base := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	row := func(at time.Time, mod func(*canonical.PlayerMatchRow)) canonical.PlayerMatchRow {
		r := canonical.PlayerMatchRow{}
		r.Summary.StartedAtUTC = at
		r.Summary.IsPvE = &falsity
		r.Summary.IsRanked = &truth
		r.Summary.DurationSeconds = &dur
		r.Enrichment.PairName = &pair
		if mod != nil {
			mod(&r)
		}
		return r
	}
	rows := []canonical.PlayerMatchRow{
		row(base.Add(2*time.Hour), func(r *canonical.PlayerMatchRow) { r.Self.TimePlayed = &sec }),
		row(base.Add(time.Hour), func(r *canonical.PlayerMatchRow) { r.Summary.IsPvE = &truth }),
		row(base, func(r *canonical.PlayerMatchRow) {
			r.Enrichment.SkillSnapshot = &canonical.SkillSnapshot{
				RatingType: canonical.RatingTypeCSR, RatingValue: &val, PlaylistGroup: &grp, MeasurementRemaining: &placement,
			}
		}),
	}
	var gotPair string
	var gotRanked bool
	got := FromCanonical(rows, FromOptions{Loc: loc, ChainOf: func(p string, ranked, pve bool) string {
		gotPair, gotRanked = p, ranked
		return "arena"
	}})
	if len(got) != 2 {
		t.Fatalf("PvE non écarté : %d lignes", len(got))
	}
	if !got[0].Start.Before(got[1].Start) {
		t.Errorf("tri chronologique non respecté")
	}
	if got[0].Start.Location() != loc {
		t.Errorf("heure non convertie dans le fuseau")
	}
	if got[0].Seconds != 600 { // repli sur la durée du match
		t.Errorf("secondes (repli) = %v", got[0].Seconds)
	}
	if got[1].Seconds != 120 { // temps joué du joueur
		t.Errorf("secondes = %v", got[1].Seconds)
	}
	if got[0].RatingValue != nil || got[0].RatingType != canonical.RatingTypeCSR || got[0].RatingGroup != grp {
		t.Errorf("placement : %+v", got[0])
	}
	if gotPair != pair || !gotRanked || got[0].Chain != "arena" {
		t.Errorf("chaîne : pair=%q ranked=%v chain=%q", gotPair, gotRanked, got[0].Chain)
	}
	if ms := FromCanonical(nil, FromOptions{}); ms == nil || len(ms) != 0 {
		t.Errorf("entrée vide : %v", ms)
	}
}
