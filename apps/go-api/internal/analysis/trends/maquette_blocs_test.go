package trends

import (
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// compareWinLoss compare les blocs « Victoires et défaites » par horizon : nombre
// de matchs, ensemble des clés de lignes, puis chaque valeur.
func compareWinLoss(t *testing.T, want map[string]refWinLoss, resp domain.TrendsPageResponse, c *refCounts) {
	t.Helper()
	for daysKey, w := range want {
		days, _ := strconv.Atoi(daysKey)
		var blk *domain.TrendsWinLossBlock
		for i := range resp.WinLoss {
			if resp.WinLoss[i].Days == days {
				blk = &resp.WinLoss[i]
			}
		}
		if blk == nil {
			t.Errorf("victoires/défaites %d j : bloc absent", days)
			continue
		}
		if blk.Matches != w.Matches {
			t.Errorf("victoires/défaites %d j : matchs Go %d, attendu %d", days, blk.Matches, w.Matches)
		}
		gotRows := map[string]domain.TrendsWinLossRow{}
		for _, r := range blk.Rows {
			gotRows[r.Key] = r
		}
		if len(gotRows) != len(w.Rows) {
			t.Errorf("victoires/défaites %d j : %d lignes Go, attendu %d", days, len(gotRows), len(w.Rows))
		}
		for _, wr := range w.Rows {
			g, ok := gotRows[wr.Key]
			if !ok {
				t.Errorf("victoires/défaites %d j %s : ligne absente de Go", days, wr.Key)
				continue
			}
			c.winLossRows++
			compareWinLossRow(t, days, wr, g)
		}
	}
}

func compareWinLossRow(t *testing.T, days int, w refWinLossRow, g domain.TrendsWinLossRow) {
	t.Helper()
	where := "victoires/défaites " + strconv.Itoa(days) + " j " + w.Key
	check := func(name string, got, want float64) {
		if !refNear(got, want) {
			t.Errorf("%s : %s Go %v, attendu %v", where, name, got, want)
		}
	}
	if g.Matches != w.N {
		t.Errorf("%s : n Go %d, attendu %d", where, g.Matches, w.N)
	}
	check("win", g.WinMean, w.Win)
	check("loss", g.LossMean, w.Loss)
	check("zw", g.ZWin, w.ZW)
	check("zl", g.ZLoss, w.ZL)
	check("r", g.R, w.R)
}

func seriesPoints(s domain.TrendsSeries, step string) ([]domain.TrendsPoint, bool) {
	switch step {
	case "match":
		return s.Match, true
	case "day":
		return s.Day, true
	case "week":
		return s.Week, true
	case "month":
		return s.Month, true
	}
	return nil, false
}

// compareSeries exige les mêmes points (début d'intervalle, valeur, matchs).
func compareSeries(t *testing.T, want []refSeries, resp domain.TrendsPageResponse, c *refCounts) {
	t.Helper()
	for _, w := range want {
		where := "série " + w.Key + "/" + w.Variant + " pas " + w.Step
		ind := findIndicator(resp, w.Key, w.Variant)
		if ind == nil {
			t.Errorf("%s : indicateur absent", where)
			continue
		}
		pts, ok := seriesPoints(ind.Series, w.Step)
		if !ok {
			t.Errorf("%s : pas inconnu", where)
			continue
		}
		if len(pts) != len(w.Points) {
			t.Errorf("%s : %d points Go, attendu %d", where, len(pts), len(w.Points))
			continue
		}
		for i, wp := range w.Points {
			c.points++
			wt, _ := wp[0].Int64()
			wv, _ := wp[1].Float64()
			wn, _ := wp[2].Int64()
			g := pts[i]
			if g.T.Unix() != wt || !refNear(g.Value, wv) || int64(g.Matches) != wn {
				t.Errorf("%s point %d : Go (t=%d v=%v n=%d), attendu (t=%d v=%v n=%d)",
					where, i, g.T.Unix(), g.Value, g.Matches, wt, wv, wn)
			}
		}
	}
}

func asFloat(v any) *float64 {
	if v == nil {
		return nil
	}
	f := v.(float64)
	return &f
}

// compareCalendar exige les mêmes jours, dans le même ordre, avec les mêmes comptes.
func compareCalendar(t *testing.T, want [][]any, resp domain.TrendsPageResponse, c *refCounts) {
	t.Helper()
	if len(resp.Calendar) != len(want) {
		t.Errorf("calendrier : %d jours Go, attendu %d", len(resp.Calendar), len(want))
		return
	}
	for i, w := range want {
		g := resp.Calendar[i]
		c.days++
		date := w[0].(string)
		if g.Date != date || g.Matches != int(w[1].(float64)) || g.Wins != int(w[2].(float64)) || g.Losses != int(w[3].(float64)) {
			t.Errorf("calendrier %s : Go (%s, %d matchs, %d v, %d d), attendu %v", date, g.Date, g.Matches, g.Wins, g.Losses, w[:4])
			continue
		}
		if wr := w[4].(float64); g.WinRate == nil || !refNear(*g.WinRate, wr) {
			t.Errorf("calendrier %s : taux de victoire Go %s, attendu %v", date, fmtOpt(g.WinRate), wr)
		}
		if ws := asFloat(w[5]); !optNear(ws, g.PerformanceScore) {
			t.Errorf("calendrier %s : score Go %s, attendu %s", date, fmtOpt(g.PerformanceScore), fmtOpt(ws))
		}
	}
}

func compareGameTypes(t *testing.T, want map[string]int, resp domain.TrendsPageResponse) {
	t.Helper()
	got := map[string]int{}
	for _, g := range resp.GameTypes {
		got[g.Key] = g.Matches
	}
	if len(got) != len(want) {
		t.Errorf("types de partie : %v Go, attendu %v", got, want)
		return
	}
	for k, n := range want {
		if got[k] != n {
			t.Errorf("types de partie %s : Go %d, attendu %d", k, got[k], n)
		}
	}
}

// TestReferenceMaquette rejoue les 1 158 matchs de la maquette et compare chaque
// bloc de la réponse aux valeurs attendues.
func TestReferenceMaquette(t *testing.T) {
	matches := loadRefMatches(t)
	want := loadRefExpected(t)
	now := time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)
	opts := Options{Now: now, Loc: time.UTC, HpToKill: refHpToKill}
	resp := BuildSolo(matches, opts)
	var c refCounts

	if len(resp.Months) != len(want.Months) {
		t.Fatalf("mois : Go %v, attendu %v", resp.Months, want.Months)
	}
	for i := range want.Months {
		if resp.Months[i] != want.Months[i] {
			t.Errorf("mois %d : Go %s, attendu %s", i, resp.Months[i], want.Months[i])
		}
	}
	compareMatrix(t, "matrice", want.Matrix, resp, &c)
	opts.GameType = "chaos"
	compareMatrix(t, "matrice chaos", want.MatrixChaos, BuildSolo(matches, opts), &c)
	compareWinLoss(t, want.WinLoss, resp, &c)
	compareSeries(t, want.Series, resp, &c)
	compareCalendar(t, want.Calendar, resp, &c)
	compareGameTypes(t, want.GameTypes, resp)
	t.Logf("%d matchs ; comparaisons : %+v", len(matches), c)
}
