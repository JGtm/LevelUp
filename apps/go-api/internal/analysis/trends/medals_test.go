package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// medalMatches crée n matchs identifiés (préfixe + rang) à partir de start.
func medalMatches(loc *time.Location, prefix string, n int, start time.Time) []Match {
	ms := many(loc, n, start)
	for i := range ms {
		ms[i].ID = prefix + string(rune('a'+i))
	}
	return ms
}

func medalBlock(t *testing.T, resp domain.TrendsPageResponse, days int) domain.TrendsMedalsBlock {
	t.Helper()
	for _, b := range resp.Medals {
		if b.Days == days {
			return b
		}
	}
	t.Fatalf("bloc %d j absent", days)
	return domain.TrendsMedalsBlock{}
}

func TestMedals_BlocsParHorizonSeulementSiLaSourceEstLue(t *testing.T) {
	loc := time.UTC
	ms := medalMatches(loc, "m", 3, ago(1, 0))
	if got := BuildSolo(ms, opts(loc)).Medals; len(got) != 0 {
		t.Fatalf("source absente : %d blocs", len(got))
	}
	o := opts(loc)
	o.Medals = []MedalCount{}
	got := BuildSolo(ms, o).Medals
	if len(got) != 4 || got[0].Days != 365 || got[3].Days != 7 {
		t.Fatalf("blocs = %+v", got)
	}
	for _, b := range got {
		if b.Compared || len(b.Rows) != 0 || b.Rows == nil {
			t.Fatalf("bloc vide attendu : %+v", b)
		}
	}
}

func TestMedals_NonComparePlusFrequentesEtDepartage(t *testing.T) {
	loc := time.UTC
	ms := medalMatches(loc, "m", 4, ago(1, 0))
	o := opts(loc)
	o.Medals = []MedalCount{
		{MatchID: "ma", MedalID: 30, Count: 2},
		{MatchID: "mb", MedalID: 20, Count: 1},
		{MatchID: "mc", MedalID: 10, Count: 1},
		{MatchID: "md", MedalID: 20, Count: 1},
		{MatchID: "inconnu", MedalID: 99, Count: 50},
	}
	o.MedalNames = map[int64]string{30: "Triple"}
	b := medalBlock(t, BuildSolo(ms, o), 7)
	if b.Compared {
		t.Fatal("4 matchs : non comparé attendu")
	}
	// Taux : 20 -> 2/4, 30 -> 2/4, 10 -> 1/4 ; égalité 20 / 30 départagée par identifiant.
	wantIDs := []int64{20, 30, 10}
	if len(b.Rows) != 3 {
		t.Fatalf("lignes = %+v", b.Rows)
	}
	for i, id := range wantIDs {
		if b.Rows[i].MedalID != id || b.Rows[i].PrevRate != nil {
			t.Fatalf("ligne %d = %+v, attendu id %d sans taux d'avant", i, b.Rows[i], id)
		}
	}
	if !near(b.Rows[0].Rate, 0.5) || !near(b.Rows[2].Rate, 0.25) {
		t.Fatalf("taux = %+v", b.Rows)
	}
	if b.Rows[0].Name != "20" || b.Rows[1].Name != "Triple" {
		t.Fatalf("noms = %q, %q", b.Rows[0].Name, b.Rows[1].Name)
	}
}

func TestMedals_ComparePlusFortesVariationsEtMedailleDisparue(t *testing.T) {
	loc := time.UTC
	cur := medalMatches(loc, "c", 10, ago(2, 0))
	prev := medalMatches(loc, "p", 10, ago(10, 0))
	ms := append(append([]Match{}, prev...), cur...)
	o := opts(loc)
	o.Medals = []MedalCount{
		// Médaille 1 : 10 sur la fenêtre (taux 1), 0 avant -> variation 1.
		{MatchID: "ca", MedalID: 1, Count: 10},
		// Médaille 2 : disparue de l'horizon, 5 avant (taux 0,5) -> variation 0,5.
		{MatchID: "pa", MedalID: 2, Count: 5},
		// Médaille 3 : stable (2 et 2) -> variation 0.
		{MatchID: "ca", MedalID: 3, Count: 2},
		{MatchID: "pa", MedalID: 3, Count: 2},
	}
	b := medalBlock(t, BuildSolo(ms, o), 7)
	if !b.Compared {
		t.Fatal("compared attendu")
	}
	if len(b.Rows) != 3 || b.Rows[0].MedalID != 1 || b.Rows[1].MedalID != 2 || b.Rows[2].MedalID != 3 {
		t.Fatalf("lignes = %+v", b.Rows)
	}
	if b.Rows[1].Rate != 0 || b.Rows[1].PrevRate == nil || !near(*b.Rows[1].PrevRate, 0.5) {
		t.Fatalf("médaille disparue = %+v", b.Rows[1])
	}
	if b.Rows[0].PrevRate == nil || *b.Rows[0].PrevRate != 0 {
		t.Fatalf("médaille absente d'avant = %+v", b.Rows[0])
	}
}

func TestMedals_DixLignesAuPlus(t *testing.T) {
	loc := time.UTC
	ms := medalMatches(loc, "m", 2, ago(1, 0))
	o := opts(loc)
	o.Medals = []MedalCount{}
	for id := int64(1); id <= 15; id++ {
		o.Medals = append(o.Medals, MedalCount{MatchID: "ma", MedalID: id, Count: int(id)})
	}
	b := medalBlock(t, BuildSolo(ms, o), 30)
	if len(b.Rows) != 10 || b.Rows[0].MedalID != 15 || b.Rows[9].MedalID != 6 {
		t.Fatalf("lignes = %+v", b.Rows)
	}
}

func TestMedals_PerimetreFiltre(t *testing.T) {
	loc := time.UTC
	ms := medalMatches(loc, "m", 2, ago(1, 0))
	ms[0].Chain, ms[1].Chain = "arena", "autre"
	o := opts(loc)
	o.GameType = "arena"
	o.Medals = []MedalCount{{MatchID: "ma", MedalID: 7, Count: 1}, {MatchID: "mb", MedalID: 8, Count: 5}}
	b := medalBlock(t, BuildSolo(ms, o), 7)
	if len(b.Rows) != 1 || b.Rows[0].MedalID != 7 || !near(b.Rows[0].Rate, 1) {
		t.Fatalf("lignes = %+v", b.Rows)
	}
}
