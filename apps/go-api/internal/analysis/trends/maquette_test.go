// Test de référence : compare les résultats de BuildSolo à ceux de la maquette
// de la page Tendances. Les données (testdata/maquette_matchs.tsv) et les valeurs
// attendues (testdata/maquette_attendu.json) sont tirées de
// .ai/MAQUETTE_TENDANCES_2026-10-04.html (maquette v15) : les calculs de la
// maquette, rejoués à now fixé (2026-09-27T02:00:00Z) avec une découpe en UTC,
// puis convertis dans les unités de l'API (taux et parts en 0..1).
package trends

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

const (
	refTolerance = 1e-9
	refHpToKill  = 225
)

// refCounts totalise les comparaisons faites, par bloc.
type refCounts struct{ matrixRows, cells, winLossRows, points, days int }

type refMonth struct {
	V *float64 `json:"v"`
	N int      `json:"n"`
	Z *float64 `json:"z"`
}

type refHorizon struct {
	Days int      `json:"days"`
	V    *float64 `json:"v"`
	N    int      `json:"n"`
	P    *float64 `json:"p"`
	NP   int      `json:"np"`
	Z    *float64 `json:"z"`
}

type refRow struct {
	Key      string       `json:"key"`
	Variant  string       `json:"variant"`
	Months   []*refMonth  `json:"months"`
	Horizons []refHorizon `json:"horizons"`
}

type refWinLossRow struct {
	Key  string  `json:"key"`
	N    int     `json:"n"`
	Win  float64 `json:"win"`
	Loss float64 `json:"loss"`
	ZW   float64 `json:"zw"`
	ZL   float64 `json:"zl"`
	R    float64 `json:"r"`
}

type refWinLoss struct {
	Matches int             `json:"matches"`
	Rows    []refWinLossRow `json:"rows"`
}

type refSeries struct {
	Key     string          `json:"key"`
	Variant string          `json:"variant"`
	Step    string          `json:"step"`
	Points  [][]json.Number `json:"points"`
}

type refExpected struct {
	Now         string                `json:"now"`
	Months      []string              `json:"months"`
	Matrix      []refRow              `json:"matrix"`
	MatrixChaos []refRow              `json:"matrix_chaos"`
	WinLoss     map[string]refWinLoss `json:"winloss"`
	Series      []refSeries           `json:"series"`
	Calendar    [][]any               `json:"calendar"`
	GameTypes   map[string]int        `json:"game_types"`
}

func refNear(a, b float64) bool {
	d := math.Abs(a - b)
	return d <= refTolerance || d <= refTolerance*math.Max(math.Abs(a), math.Abs(b))
}

// optNear compare une valeur attendue optionnelle à une valeur Go optionnelle.
func optNear(want, got *float64) bool {
	if want == nil || got == nil {
		return want == nil && got == nil
	}
	return refNear(*want, *got)
}

func fmtOpt(p *float64) string {
	if p == nil {
		return "absent"
	}
	return strconv.FormatFloat(*p, 'g', -1, 64)
}

func tsvFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return &v
}

func tsvInt(s string) int {
	v := tsvFloat(s)
	if v == nil {
		return 0
	}
	return int(*v)
}

var refOutcomes = map[int]canonical.Outcome{
	1: canonical.OutcomeTie, 2: canonical.OutcomeWin, 3: canonical.OutcomeLoss, 4: canonical.OutcomeDNF,
}

func loadRefMatches(t *testing.T) []Match {
	t.Helper()
	raw, err := os.ReadFile("testdata/maquette_matchs.tsv")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\r\n"), "\n")
	col := map[string]int{}
	for i, name := range strings.Split(strings.TrimRight(lines[0], "\r"), "\t") {
		col[name] = i
	}
	out := make([]Match, 0, len(lines)-1)
	for _, ln := range lines[1:] {
		f := strings.Split(strings.TrimRight(ln, "\r"), "\t")
		for len(f) < len(col) {
			f = append(f, "")
		}
		out = append(out, refMatch(func(name string) string { return f[col[name]] }))
	}
	return out
}

// refMatch construit un Match depuis les colonnes d'une ligne du TSV.
func refMatch(g func(string) string) Match {
	m := Match{
		Start:            time.Unix(int64(tsvInt(g("t"))), 0).UTC(),
		Outcome:          refOutcomes[tsvInt(g("o"))],
		Kills:            tsvInt(g("k")),
		Deaths:           tsvInt(g("d")),
		Assists:          tsvInt(g("a")),
		KDA:              tsvFloat(g("kda")),
		ShotsFired:       tsvInt(g("sf")),
		ShotsHit:         tsvInt(g("sh")),
		DamageDealt:      tsvFloat(g("dd")),
		DamageTaken:      tsvFloat(g("dt")),
		AvgLife:          tsvFloat(g("life")),
		MaxSpree:         tsvFloat(g("sp")),
		HeadshotKills:    tsvInt(g("hs")),
		PowerKills:       tsvInt(g("pw")),
		TeamMMR:          tsvFloat(g("tm")),
		EnemyMMR:         tsvFloat(g("em")),
		PerformanceScore: tsvFloat(g("perf")),
		Seconds:          float64(tsvInt(g("dur"))),
		IsWithFriends:    tsvInt(g("mask")) != 0,
		Chain:            g("lg"),
	}
	if g("pt") != "" {
		m.Objective = &ObjectiveSample{
			Take: *tsvFloat(g("pt")), Defend: *tsvFloat(g("pd")), HoldSeconds: *tsvFloat(g("ph")),
			TeamTake: *tsvFloat(g("tt")), TeamDefend: *tsvFloat(g("td")), TeamHoldSeconds: *tsvFloat(g("th")),
			TeamSize: tsvInt(g("osz")),
		}
	}
	if g("e_t") != "" {
		used := tsvInt(g("e_u"))
		m.Equipment = &EquipmentSample{Used: used, Total: used + tsvInt(g("e_k")) + tsvInt(g("e_d"))}
	}
	if csr := tsvFloat(g("csr")); csr != nil {
		m.RatingType, m.RatingValue, m.RatingGroup = canonical.RatingTypeCSR, csr, ""
	} else if lu := tsvFloat(g("lu")); lu != nil {
		m.RatingType, m.RatingValue, m.RatingGroup = canonical.RatingTypeLUSR, lu, m.Chain
	}
	return m
}

func loadRefExpected(t *testing.T) refExpected {
	t.Helper()
	raw, err := os.ReadFile("testdata/maquette_attendu.json")
	if err != nil {
		t.Fatal(err)
	}
	var e refExpected
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func findIndicator(resp domain.TrendsPageResponse, key, variant string) *domain.TrendsIndicator {
	for i := range resp.Indicators {
		if resp.Indicators[i].Key == key && resp.Indicators[i].Variant == variant {
			return &resp.Indicators[i]
		}
	}
	return nil
}

// compareMatrix compare les lignes de matrice attendues (mois et horizons).
func compareMatrix(t *testing.T, label string, want []refRow, resp domain.TrendsPageResponse, c *refCounts) {
	t.Helper()
	for _, w := range want {
		got := findIndicator(resp, w.Key, w.Variant)
		if got == nil {
			t.Errorf("%s %s/%q : ligne absente de Indicators", label, w.Key, w.Variant)
			continue
		}
		c.matrixRows++
		if len(got.Months) != len(w.Months) {
			t.Errorf("%s %s/%q : %d mois, attendu %d", label, w.Key, w.Variant, len(got.Months), len(w.Months))
			continue
		}
		for i, wm := range w.Months {
			compareMonth(t, label+" "+w.Key+"/"+w.Variant+" mois "+strconv.Itoa(i), wm, got.Months[i])
			c.cells++
		}
		for _, wh := range w.Horizons {
			compareHorizon(t, label+" "+w.Key+"/"+w.Variant, wh, got.Horizons)
			c.cells++
		}
	}
}

func compareMonth(t *testing.T, where string, wm *refMonth, gm domain.TrendsMonthCell) {
	t.Helper()
	if wm == nil {
		if gm.Value != nil {
			t.Errorf("%s : attendu absent, Go %s (n=%d)", where, fmtOpt(gm.Value), gm.Matches)
		}
		return
	}
	if !optNear(wm.V, gm.Value) {
		t.Errorf("%s : valeur Go %s, attendu %s", where, fmtOpt(gm.Value), fmtOpt(wm.V))
	}
	if gm.Matches != wm.N {
		t.Errorf("%s : matchs Go %d, attendu %d", where, gm.Matches, wm.N)
	}
	if !optNear(wm.Z, gm.Z) {
		t.Errorf("%s : z Go %s, attendu %s", where, fmtOpt(gm.Z), fmtOpt(wm.Z))
	}
}

func compareHorizon(t *testing.T, prefix string, wh refHorizon, gots []domain.TrendsHorizonCell) {
	t.Helper()
	where := prefix + " horizon " + strconv.Itoa(wh.Days) + " j"
	var gh *domain.TrendsHorizonCell
	for i := range gots {
		if gots[i].Days == wh.Days {
			gh = &gots[i]
		}
	}
	if gh == nil {
		t.Errorf("%s : horizon absent", where)
		return
	}
	if !optNear(wh.V, gh.Value) {
		t.Errorf("%s : valeur Go %s, attendu %s", where, fmtOpt(gh.Value), fmtOpt(wh.V))
	}
	if gh.Matches != wh.N {
		t.Errorf("%s : matchs Go %d, attendu %d", where, gh.Matches, wh.N)
	}
	if !optNear(wh.P, gh.PrevValue) {
		t.Errorf("%s : valeur d'avant Go %s, attendu %s", where, fmtOpt(gh.PrevValue), fmtOpt(wh.P))
	}
	if gh.PrevMatches != wh.NP {
		t.Errorf("%s : matchs d'avant Go %d, attendu %d", where, gh.PrevMatches, wh.NP)
	}
	if !optNear(wh.Z, gh.Z) {
		t.Errorf("%s : z Go %s, attendu %s", where, fmtOpt(gh.Z), fmtOpt(wh.Z))
	}
}
