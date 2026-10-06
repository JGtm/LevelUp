package trends

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func TestSeriesMinimumPerPoint(t *testing.T) {
	loc := paris(t)
	at := func(m time.Month, d, h int) time.Time { return time.Date(2026, m, d, h, 0, 0, 0, time.UTC) }
	var ms []Match
	// Jour : le 22/09 compte 2 matchs (point), le 23/09 en compte 1 (pas de point).
	ms = append(ms, mk(loc, at(9, 22, 10)), mk(loc, at(9, 22, 11)), mk(loc, at(9, 23, 10)))
	// Semaine du 07/09 : 3 matchs sur 3 jours (point) ; semaine du 14/09 : 2 matchs (pas de point).
	ms = append(ms, mk(loc, at(9, 8, 10)), mk(loc, at(9, 9, 10)), mk(loc, at(9, 10, 10)))
	ms = append(ms, mk(loc, at(9, 15, 10)), mk(loc, at(9, 16, 10)))
	// Mois d'août : 5 matchs sur 5 jours (point) ; juillet : 4 matchs (pas de point).
	for d := 1; d <= 5; d++ {
		ms = append(ms, mk(loc, at(8, d, 10)))
	}
	for d := 1; d <= 4; d++ {
		ms = append(ms, mk(loc, at(7, d+10, 10)))
	}
	ind := indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyKDA, "")
	if len(ind.Series.Day) != 1 || ind.Series.Day[0].Matches != 2 {
		t.Errorf("points par jour = %+v, attendu un seul point de 2 matchs", ind.Series.Day)
	}
	var weeks []time.Time
	for _, p := range ind.Series.Week {
		weeks = append(weeks, p.T)
	}
	wantWeek := time.Date(2026, 9, 7, 0, 0, 0, 0, loc)
	hasWeek := false
	for _, w := range weeks {
		if w.Equal(wantWeek) {
			hasWeek = true
		}
		if w.Equal(time.Date(2026, 9, 14, 0, 0, 0, 0, loc)) {
			t.Errorf("la semaine du 14/09 (2 matchs) ne doit pas porter de point")
		}
	}
	if !hasWeek {
		t.Errorf("semaine du 07/09 absente : %v", weeks)
	}
	// Août (5 matchs) et septembre (8 matchs) portent un point ; juillet (4 matchs) n'en porte pas.
	if len(ind.Series.Month) != 2 || !ind.Series.Month[0].T.Equal(time.Date(2026, 8, 1, 0, 0, 0, 0, loc)) {
		t.Errorf("points par mois = %+v, attendu août et septembre", ind.Series.Month)
	}
}

func TestMatchSeriesLimitedTo30Days(t *testing.T) {
	loc := paris(t)
	ms := []Match{mk(loc, ago(29, 0)), mk(loc, ago(31, 0))}
	resp := BuildSolo(ms, opts(loc))
	kda := indicatorOf(t, resp, domain.TrendsKeyKDA, "")
	if len(kda.Series.Match) != 1 {
		t.Errorf("série par match = %d points, attendu 1", len(kda.Series.Match))
	}
	for _, key := range []string{domain.TrendsKeyWinRate, domain.TrendsKeyMatchCount, domain.TrendsKeyHoursPlayed, domain.TrendsKeyDaysPlayed, domain.TrendsKeyDNFRate} {
		ind := indicatorOf(t, resp, key, "")
		if ind.Series.Match == nil || len(ind.Series.Match) != 0 {
			t.Errorf("%s : série par match = %v, attendu vide non nil", key, ind.Series.Match)
		}
	}
}

func TestLocalDayChangeAroundMidnight(t *testing.T) {
	loc := paris(t)
	// 21:30Z = 23:30 à Paris le 26/09 ; 22:30Z = 00:30 le 27/09.
	ms := []Match{
		mk(loc, time.Date(2026, 9, 26, 21, 30, 0, 0, time.UTC)),
		mk(loc, time.Date(2026, 9, 26, 22, 30, 0, 0, time.UTC)),
	}
	resp := BuildSolo(ms, opts(loc))
	if len(resp.Calendar) != 2 || resp.Calendar[0].Date != "2026-09-26" || resp.Calendar[1].Date != "2026-09-27" {
		t.Fatalf("calendrier = %+v", resp.Calendar)
	}
	c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyDaysPlayed, ""), 7)
	if c.Value == nil || *c.Value != 2 {
		t.Errorf("jours joués = %v, attendu 2", c.Value)
	}
}

func TestCalendarDayContent(t *testing.T) {
	loc := paris(t)
	ms := []Match{
		mk(loc, ago(1, 0), func(m *Match) { m.PerformanceScore = f64(60) }),
		mk(loc, ago(1, time.Minute), outcome(canonical.OutcomeLoss), func(m *Match) { m.PerformanceScore = f64(40) }),
		mk(loc, ago(1, 2*time.Minute), outcome(canonical.OutcomeTie)),
	}
	d := BuildSolo(ms, opts(loc)).Calendar
	if len(d) != 1 {
		t.Fatalf("calendrier = %+v", d)
	}
	if d[0].Matches != 3 || d[0].Wins != 1 || d[0].Losses != 1 {
		t.Errorf("jour = %+v", d[0])
	}
	if d[0].WinRate == nil || !near(*d[0].WinRate, 1.0/3.0) {
		t.Errorf("taux de victoire = %v, attendu 1/3 (victoires / tous les matchs)", d[0].WinRate)
	}
	if d[0].PerformanceScore == nil || !near(*d[0].PerformanceScore, 50) {
		t.Errorf("score de performance = %v", d[0].PerformanceScore)
	}
}

func TestEmptyInput(t *testing.T) {
	loc := paris(t)
	resp := BuildSolo(nil, opts(loc))
	if len(resp.Months) != 12 || len(resp.WinLoss) != 4 {
		t.Errorf("mois=%d blocs=%d", len(resp.Months), len(resp.WinLoss))
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, bad := range []string{`"game_types":null`, `"indicators":null`, `"calendar":null`, `"win_loss":null`, `"medals":null`,
		`"day":null`, `"week":null`, `"month":null`, `"rows":null`} {
		if strings.Contains(s, bad) {
			t.Errorf("réponse vide contient %s", bad)
		}
	}
	for _, b := range resp.WinLoss {
		if b.Rows == nil || b.Required != 30 {
			t.Errorf("bloc %+v", b)
		}
	}
}

func TestGameTypeFilter(t *testing.T) {
	loc := paris(t)
	ms := many(loc, 3, ago(2, 0))
	ms = append(ms, many(loc, 2, ago(2, time.Hour), func(m *Match) { m.Chain = "btb" })...)
	ms = append(ms, many(loc, 1, ago(2, 2*time.Hour), func(m *Match) { m.Chain = "" })...)
	o := opts(loc)
	o.GameType = "btb"
	resp := BuildSolo(ms, o)
	if resp.GameType != "btb" {
		t.Errorf("game_type = %q", resp.GameType)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyMatchCount, ""), 365); c.Matches != 2 {
		t.Errorf("matchs filtrés = %d, attendu 2", c.Matches)
	}
	want := []domain.TrendsGameType{{Key: "arena", Matches: 3}, {Key: "btb", Matches: 2}, {Key: "other", Matches: 1}}
	if len(resp.GameTypes) != 3 {
		t.Fatalf("types = %+v", resp.GameTypes)
	}
	for i, w := range want {
		if resp.GameTypes[i] != w {
			t.Errorf("types[%d] = %+v, attendu %+v", i, resp.GameTypes[i], w)
		}
	}
	total := 0
	for _, b := range resp.Mix.Month {
		for _, n := range b.Counts {
			total += n
		}
	}
	if total != 6 {
		t.Errorf("mix non filtré : %d matchs, attendu 6", total)
	}
	o.GameType = "other"
	if c := horizon(t, indicatorOf(t, BuildSolo(ms, o), domain.TrendsKeyMatchCount, ""), 365); c.Matches != 1 {
		t.Errorf("filtre other = %d matchs, attendu 1", c.Matches)
	}
}

func TestFutureMatchesIgnored(t *testing.T) {
	loc := paris(t)
	ms := []Match{mk(loc, ago(1, 0)), mk(loc, testNow.Add(time.Hour))}
	c := horizon(t, indicatorOf(t, BuildSolo(ms, opts(loc)), domain.TrendsKeyMatchCount, ""), 365)
	if c.Matches != 1 {
		t.Errorf("matchs = %d, attendu 1", c.Matches)
	}
}

func TestRatingVariants(t *testing.T) {
	loc := paris(t)
	rate := func(typ canonical.RatingType, group string, v *float64) func(*Match) {
		return func(m *Match) { m.RatingType, m.RatingGroup, m.RatingValue = typ, group, v }
	}
	ms := []Match{
		mk(loc, ago(3, 0), rate(canonical.RatingTypeCSR, "ranked-arena", f64(1000))),
		mk(loc, ago(2, 0), rate(canonical.RatingTypeCSR, "ranked-arena", f64(1100))),
		mk(loc, ago(1, 0), rate(canonical.RatingTypeCSR, "ranked-arena", nil)), // placement
		mk(loc, ago(1, time.Minute), rate(canonical.RatingTypeLUSR, "", f64(50))),
		mk(loc, ago(1, 2*time.Minute), rate(canonical.RatingTypeLUSR, "btb", f64(60))),
	}
	resp := BuildSolo(ms, opts(loc))
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyCSRValue, "ranked-arena"), 7); c.Value == nil || *c.Value != 1100 {
		t.Errorf("csr = %v, attendu la dernière valeur connue 1100", c.Value)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyLUSRValue, ""), 7); c.Value == nil || *c.Value != 50 {
		t.Errorf("lusr sans groupe = %v", c.Value)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyLUSRValue, "btb"), 7); c.Value == nil || *c.Value != 60 {
		t.Errorf("lusr btb = %v", c.Value)
	}
}

func TestYieldAndMissingDamage(t *testing.T) {
	loc := paris(t)
	ms := []Match{mk(loc, ago(1, 0), func(m *Match) {
		m.Kills, m.Assists, m.Deaths = 9, 0, 2
		m.DamageDealt, m.DamageTaken = f64(2250), f64(900)
	})}
	resp := BuildSolo(ms, opts(loc))
	// conversion = 225 x 9 / 2250 = 0,9 ; résistance = 900 / (225 x 2) = 2.
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyOffensiveConversion, ""), 7); c.Value == nil || !near(*c.Value, 0.9) {
		t.Errorf("conversion = %v", c.Value)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyDefensiveResistance, ""), 7); c.Value == nil || !near(*c.Value, 2) {
		t.Errorf("résistance = %v", c.Value)
	}
	if c := horizon(t, indicatorOf(t, resp, domain.TrendsKeyDamageBalance, ""), 7); c.Value == nil || !near(*c.Value, 1350) {
		t.Errorf("bilan de dégâts = %v", c.Value)
	}
	nodmg := BuildSolo([]Match{mk(loc, ago(1, 0))}, opts(loc))
	if hasIndicator(nodmg, domain.TrendsKeyOffensiveConversion) || hasIndicator(nodmg, domain.TrendsKeyDamageBalance) {
		t.Errorf("lignes sans dégâts non omises")
	}
}
