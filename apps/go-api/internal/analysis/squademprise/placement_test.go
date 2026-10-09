package squademprise

// placement_test.go — une règle par test (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3.2).
// Chaque test a été vu ROUGE sous la mutation nommée au journal V3 du plan.

import (
	"encoding/json"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
)

func pf(v float64) *float64 { return &v }
func pi(v int64) *int64     { return &v }

// vieMesuree : une vie mesurée à la portée 18 m, 20 s dont 10 s mesurées.
func vieMesuree(match, xuid string, start int64, medianM float64, kills int) PlacementRow {
	return PlacementRow{
		MatchID: match, XUID: xuid, StartMS: start, EndMS: start + 20000, DurationMS: 20000,
		MeasuredMS: 10000, MedianM: pf(medianM), BeyondMS: pi(2500), RadarM: pf(18),
		CarrierMS: 1000, TeamDownMS: 2000, UnplacedMS: 3000, TeammateUnplacedMS: 4100, Kills: kills,
	}
}

var compo = []domain.SessionUsageSquadPlayer{{XUID: "A", Gamertag: "Alpha"}, {XUID: "B", Gamertag: "Bravo"}}

func entree(rows ...PlacementRow) PlacementInput {
	return PlacementInput{
		Players: compo, Scope: []string{"m1", "m2", "m3"},
		Read:         PlacementRead{Rows: rows},
		CurrentRadar: map[string]float64{"m1": 18, "m2": 18, "m3": 18},
	}
}

func joueur(t *testing.T, b *domain.SquadEmprisePlacement, xuid string) domain.SquadEmprisePlacementPlayer {
	t.Helper()
	if b == nil {
		t.Fatal("bloc nil")
	}
	for _, p := range b.Players {
		if p.XUID == xuid {
			return p
		}
	}
	t.Fatalf("joueur %s absent", xuid)
	return domain.SquadEmprisePlacementPlayer{}
}

// X = médiane / portée ; part hors radar = hors radar / mesuré.
func TestPlacement_XEtPartHorsRadar(t *testing.T) {
	b, _ := Placement(entree(vieMesuree("m1", "A", 0, 27, 2)))
	v := joueur(t, b, "A").Lives[0]
	if v.RadarRatio != 1.5 || v.OutOfRadarShare != 0.25 {
		t.Errorf("X = %v, part hors radar = %v ; attendu 1,5 et 0,25", v.RadarRatio, v.OutOfRadarShare)
	}
	if v.MatchID != "m1" || v.StartMS != 0 || v.DurationMS != 20000 || v.Kills != 2 {
		t.Errorf("identifiants de la vie : %+v", v)
	}
}

// Isolé = X ≥ 1,0, rentable = au moins un frag : les deux bornes comprises.
func TestPlacement_QuartsEtBornes(t *testing.T) {
	cas := []struct {
		median float64
		kills  int
		want   string
	}{
		{18, 1, domain.EmprisePlacementIsolatedProductive},
		{18, 0, domain.EmprisePlacementIsolatedCostly},
		{17.99, 1, domain.EmprisePlacementInRangeProductive},
		{17.99, 0, domain.EmprisePlacementInRangeCostly},
	}
	for _, c := range cas {
		b, _ := Placement(entree(vieMesuree("m1", "A", 0, c.median, c.kills)))
		if got := joueur(t, b, "A").Lives[0].Quadrant; got != c.want {
			t.Errorf("médiane %v m, %d frag(s) : quart %s, attendu %s", c.median, c.kills, got, c.want)
		}
	}
}

// Une vie sans médiane est comptée, jamais tracée ni classée.
func TestPlacement_VieNonMesuree(t *testing.T) {
	nm := vieMesuree("m1", "A", 30000, 0, 3)
	nm.MedianM = nil
	b, _ := Placement(entree(vieMesuree("m1", "A", 0, 9, 1), nm))
	a := joueur(t, b, "A")
	if a.LivesTotal != 2 || a.LivesMeasured != 1 || len(a.Lives) != 1 {
		t.Errorf("vies : total %d, mesurées %d, tracées %d ; attendu 2, 1, 1", a.LivesTotal, a.LivesMeasured, len(a.Lives))
	}
	c := b.Coverage
	if c.LivesTotal != 2 || c.LivesMeasured != 1 || c.LivesUnmeasured != 1 {
		t.Errorf("couverture : %+v", c)
	}
	if a.Quadrants[0].Lives != 1 || *a.Quadrants[0].Share != 1 {
		t.Errorf("la vie non mesurée entre dans les quarts : %+v", a.Quadrants)
	}
}

// Variante sans portée courante : le match sort de l'univers et se compte.
func TestPlacement_MatchSansPortee(t *testing.T) {
	in := entree(vieMesuree("m1", "A", 0, 9, 1), vieMesuree("m2", "A", 0, 9, 1))
	delete(in.CurrentRadar, "m2")
	b, _ := Placement(in)
	c := b.Coverage
	if c.MatchesWithPlacement != 2 || c.MatchesWithoutRange != 1 || c.LivesTotal != 1 || c.StaleLives != 0 {
		t.Errorf("couverture : %+v ; attendu 2 matchs à placement, 1 sans portée, 1 vie", c)
	}
}

// Portée écrite ≠ portée courante (ou écrite NULL alors que la variante en a une) : la vie est
// écartée, comptée, et son match nommé au bilan.
func TestPlacement_PorteePerimee(t *testing.T) {
	ancienne := vieMesuree("m2", "A", 0, 9, 1)
	ancienne.RadarM = pf(24)
	sans := vieMesuree("m3", "B", 0, 9, 1)
	sans.RadarM, sans.BeyondMS = nil, nil
	b, bilan := Placement(entree(vieMesuree("m1", "A", 0, 9, 1), ancienne, sans))
	if b.Coverage.StaleLives != 2 || b.Coverage.LivesTotal != 1 || b.Coverage.MatchesWithoutRange != 0 {
		t.Errorf("couverture : %+v ; attendu 2 vies périmées, 1 retenue", b.Coverage)
	}
	if fmt.Sprint(bilan.StaleMatches) != "[m2 m3]" {
		t.Errorf("matchs périmés = %v, attendu [m2 m3]", bilan.StaleMatches)
	}
	if joueur(t, b, "B").LivesTotal != 0 {
		t.Error("une vie périmée est retenue")
	}
}

// Le reste du lobby et les matchs hors périmètre ne sont jamais retenus.
func TestPlacement_HorsCompositionEtHorsPerimetre(t *testing.T) {
	b, bilan := Placement(entree(vieMesuree("m1", "A", 0, 9, 1), vieMesuree("m1", "C", 0, 9, 1),
		vieMesuree("m9", "A", 0, 9, 1)))
	if bilan.IgnoredRows != 2 || b.Coverage.LivesTotal != 1 || len(b.Players) != 2 {
		t.Errorf("ignorées %d, retenues %d, joueurs %d ; attendu 2, 1, 2",
			bilan.IgnoredRows, b.Coverage.LivesTotal, len(b.Players))
	}
	if b.Coverage.MatchesWithPlacement != 1 || b.Coverage.MatchesTotal != 3 {
		t.Errorf("matchs : %+v", b.Coverage)
	}
}

// Ordre des fiches (un joueur sans vie garde sa place), vies dans l'ordre chronologique du
// périmètre quel que soit l'ordre de la lecture.
func TestPlacement_OrdreDesFichesEtChronologie(t *testing.T) {
	b, _ := Placement(entree(vieMesuree("m3", "A", 0, 9, 1), vieMesuree("m1", "A", 50000, 9, 1),
		vieMesuree("m1", "A", 1000, 9, 1)))
	if b.Players[0].XUID != "A" || b.Players[1].XUID != "B" || b.Players[1].Gamertag != "Bravo" {
		t.Fatalf("ordre des fiches : %+v", b.Players)
	}
	var cles []string
	for _, v := range b.Players[0].Lives {
		cles = append(cles, fmt.Sprintf("%s/%d", v.MatchID, v.StartMS))
	}
	if fmt.Sprint(cles) != "[m1/1000 m1/50000 m3/0]" {
		t.Errorf("ordre des vies = %v", cles)
	}
	bravo := b.Players[1]
	if bravo.MedianRadarRatio != nil || bravo.MedianKills != nil || bravo.Lives == nil || len(bravo.Quadrants) != 4 {
		t.Errorf("joueur sans vie : %+v", bravo)
	}
	for _, q := range bravo.Quadrants {
		if q.Share != nil {
			t.Errorf("part publiée sans vie mesurée : %+v", q)
		}
	}
}

// Médianes du gros point : nombre pair = moyenne des deux centrales.
func TestPlacement_Medianes(t *testing.T) {
	b, _ := Placement(entree(vieMesuree("m1", "A", 0, 9, 0), vieMesuree("m1", "A", 1, 18, 1),
		vieMesuree("m1", "A", 2, 27, 3), vieMesuree("m1", "A", 3, 36, 4)))
	a := joueur(t, b, "A")
	if *a.MedianRadarRatio != 1.25 || *a.MedianKills != 2 {
		t.Errorf("médianes = %v, %v ; attendu 1,25 et 2", *a.MedianRadarRatio, *a.MedianKills)
	}
}

// Quatre quarts, toujours, dans l'ordre de publication ; parts sur les vies mesurées.
func TestPlacement_PartsDesQuarts(t *testing.T) {
	b, _ := Placement(entree(vieMesuree("m1", "A", 0, 9, 1), vieMesuree("m1", "A", 1, 9, 2),
		vieMesuree("m1", "A", 2, 30, 0), vieMesuree("m1", "A", 3, 9, 0)))
	a := joueur(t, b, "A")
	want := []struct {
		q     string
		n     int
		share float64
	}{
		{domain.EmprisePlacementInRangeProductive, 2, 0.5},
		{domain.EmprisePlacementIsolatedProductive, 0, 0},
		{domain.EmprisePlacementInRangeCostly, 1, 0.25},
		{domain.EmprisePlacementIsolatedCostly, 1, 0.25},
	}
	for i, w := range want {
		got := a.Quadrants[i]
		if got.Quadrant != w.q || got.Lives != w.n || got.Share == nil || *got.Share != w.share {
			t.Errorf("quart %d = %+v, attendu %+v", i, got, w)
		}
	}
}

// Les cumuls de la couverture ne portent que sur les vies retenues.
func TestPlacement_CouvertureDesCauses(t *testing.T) {
	perimee := vieMesuree("m2", "A", 0, 9, 1)
	perimee.RadarM = pf(24)
	in := entree(vieMesuree("m1", "A", 0, 9, 1), vieMesuree("m1", "B", 0, 9, 1), perimee,
		vieMesuree("m3", "A", 0, 9, 1))
	delete(in.CurrentRadar, "m3")
	b, _ := Placement(in)
	c := b.Coverage
	if c.MeasuredMS != 20000 || c.CarrierMS != 2000 || c.TeamDownMS != 4000 || c.UnplacedMS != 6000 ||
		c.TeammateUnplacedMS != 8200 {
		t.Errorf("cumuls : %+v ; attendu ceux des deux seules vies retenues", c)
	}
}

// Aucune vie de la composition sur le périmètre : bloc absent (omission, jamais un nuage vide).
func TestPlacement_AucuneVie(t *testing.T) {
	if b, _ := Placement(entree(vieMesuree("m1", "C", 0, 9, 1))); b != nil {
		t.Errorf("bloc publié sans vie de la composition : %+v", b)
	}
	if b, _ := Placement(entree()); b != nil {
		t.Errorf("bloc publié sans lecture : %+v", b)
	}
}

// Contrat JSON : un joueur sans vie mesurée n'a NI médiane NI part (des zéros se liraient « collé au
// coéquipier » et « aucun quart »), mais garde ses quatre quarts et une liste de vies vide ; les
// seuils des quarts voyagent avec le bloc, la couverture n'en sort jamais (journal seulement).
func TestPlacement_ContratJSON(t *testing.T) {
	b, _ := Placement(entree(vieMesuree("m1", "A", 0, 27, 2)))
	brut, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		IsolatedFromRatio   *float64 `json:"isolated_from_ratio"`
		ProductiveFromKills *int     `json:"productive_from_kills"`
		Players             []map[string]json.RawMessage
	}
	if err := json.Unmarshal(brut, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.IsolatedFromRatio == nil || doc.ProductiveFromKills == nil || len(doc.Players) != 2 {
		t.Fatalf("bloc = %s", brut)
	}
	bravo := doc.Players[1]
	for _, absent := range []string{"median_radar_ratio", "median_kills"} {
		if _, ok := bravo[absent]; ok {
			t.Errorf("joueur sans vie mesurée : %s publié (%s)", absent, brut)
		}
	}
	if string(bravo["lives"]) != "[]" {
		t.Errorf("lives = %s, attendu []", bravo["lives"])
	}
	var quarts []map[string]json.RawMessage
	if err := json.Unmarshal(bravo["quadrants"], &quarts); err != nil || len(quarts) != 4 {
		t.Fatalf("quadrants = %s", bravo["quadrants"])
	}
	if _, ok := quarts[0]["share"]; ok {
		t.Errorf("part publiée sans vie mesurée : %s", bravo["quadrants"])
	}
	var racine map[string]json.RawMessage
	if err := json.Unmarshal(brut, &racine); err != nil {
		t.Fatal(err)
	}
	if _, ok := racine["coverage"]; ok {
		t.Errorf("couverture publiée (%s) : elle ne sert qu'au journal de la lecture", brut)
	}
}
