package teammates

import (
	"context"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// sansDegatsSubis : resolver de test qui déclare un titre sans damage_taken (Halo 5).
type sansDegatsSubis struct{}

func (sansDegatsSubis) HostFor(string, games.EndpointKey) (string, bool) { return "", false }

func (sansDegatsSubis) DamageModelFor(string) (mappings.DamageModelConstants, bool) {
	return mappings.DamageModelConstants{NoDamageTaken: true}, true
}

func withDamage(row canonical.PlayerMatchRow, dealt, taken *int) canonical.PlayerMatchRow {
	row.Self.DamageDealt = dealt
	row.Self.DamageTaken = taken
	return row
}

func intp(v int) *int { return &v }

func perMinuteByPlayer(entries []domain.SquadPerMinuteEntry) map[string]domain.SquadPerMinuteEntry {
	out := make(map[string]domain.SquadPerMinuteEntry, len(entries))
	for _, e := range entries {
		out[e.Player] = e
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Les dégâts par minute sont la somme des dégâts rapportée à la somme des minutes (jamais la
// moyenne des rapports) ; un match sans dégâts connus sort du numérateur ET du dénominateur ; un
// joueur sans aucun dégât connu a des cadences absentes (nil), pas nulles.
func TestBuildSquadPerMinuteStats_DamageSumOverSumAndAbsent(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{
		"main": {
			// 600 s à 3000/1200 puis 180 s à 900/300 : (3900) / 13 min = 300 ; (1500) / 13 = 115,38.
			withDamage(rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80), intp(3000), intp(1200)),
			withDamage(rowWithStats("m2", t0, canonical.OutcomeWin, 3, 1, 1, 180, 50, 80), intp(900), intp(300)),
			// Dégâts inconnus : le match n'entre pas dans les dégâts (mais dans les frags).
			withDamage(rowWithStats("m3", t0, canonical.OutcomeWin, 4, 4, 0, 600, 50, 80), nil, nil),
			// Hors périmètre : ignoré.
			withDamage(rowWithStats("m9", t0, canonical.OutcomeWin, 9, 9, 9, 600, 50, 80), intp(99999), intp(99999)),
		},
		"friend1": {
			withDamage(rowWithStats("m1", t0, canonical.OutcomeWin, 4, 8, 6, 600, 40, 50), nil, nil),
		},
	}}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
		{MatchID: "m2", StartTime: t0, Kills: 3, Deaths: 1, Assists: 1, TimePlayedSecs: 180},
		{MatchID: "m3", StartTime: t0, Kills: 4, Deaths: 4, TimePlayedSecs: 600},
	}
	got := perMinuteByPlayer(svc.buildSquadPerMinuteStats(context.Background(), allSquadRows, "main", []string{"friend1"}, nil))

	m := got["main"]
	if m.DamageDealtPerMinute == nil || m.DamageTakenPerMinute == nil {
		t.Fatalf("main : dégâts attendus, got %+v", m)
	}
	if want := math.Round(3900.0/13*100) / 100; !approx(*m.DamageDealtPerMinute, want) {
		t.Errorf("infligés/min : want %.2f got %.2f", want, *m.DamageDealtPerMinute)
	}
	if want := math.Round(1500.0/13*100) / 100; !approx(*m.DamageTakenPerMinute, want) {
		t.Errorf("subis/min : want %.2f got %.2f", want, *m.DamageTakenPerMinute)
	}
	// m3 compte pour les frags : 17 frags / 23 min.
	if want := math.Round(17.0/23*100) / 100; !approx(m.KillsPerMinute, want) {
		t.Errorf("frags/min : want %.2f got %.2f", want, m.KillsPerMinute)
	}
	f := got["friend1"]
	if f.DamageDealtPerMinute != nil || f.DamageTakenPerMinute != nil {
		t.Errorf("friend1 sans dégâts connus : cadences absentes attendues, got %+v", f)
	}
}

// Un match sans seconde jouée n'entre dans aucune cadence de dégâts.
func TestBuildSquadPerMinuteStats_DamageIgnoresMatchWithoutSeconds(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{
		"main": {
			withDamage(rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80), intp(1000), intp(500)),
			withDamage(rowWithStats("m2", t0, canonical.OutcomeWin, 1, 1, 1, 0, 50, 80), intp(777777), intp(777777)),
		},
	}}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
		{MatchID: "m2", StartTime: t0, Kills: 1, Deaths: 1, Assists: 1, TimePlayedSecs: 0},
	}
	got := svc.buildSquadPerMinuteStats(context.Background(), allSquadRows, "main", nil, nil)
	if len(got) != 1 || got[0].DamageDealtPerMinute == nil || !approx(*got[0].DamageDealtPerMinute, 100) {
		t.Fatalf("infligés/min : 1000 / 10 min = 100 attendu, got %+v", got)
	}
}

// Sans lecteur, le main garde ses cadences de frags et n'a aucun dégât ; sans aucun signal, nil.
func TestBuildSquadPerMinuteStats_NoLoaderMainWithoutDamage(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
	}
	got := svc.buildSquadPerMinuteStats(context.Background(), allSquadRows, "main", nil, nil)
	if len(got) != 1 || got[0].DamageDealtPerMinute != nil || got[0].KillsPerMinute != 1 {
		t.Fatalf("main sans lecteur : frags seuls attendus, got %+v", got)
	}
}

// Titre sans dégâts subis (Halo 5) : les subis sont absents même si une valeur est rangée, les
// infligés restent servis.
func TestBuildSquadPerMinuteStats_TitleWithoutDamageTaken(t *testing.T) {
	precedent := games.DefaultEndpointResolver()
	games.SetDefaultEndpointResolver(sansDegatsSubis{})
	t.Cleanup(func() { games.SetDefaultEndpointResolver(precedent) })
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	loader := &fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{
		"main": {withDamage(rowWithStats("m1", t0, canonical.OutcomeWin, 10, 5, 2, 600, 50, 80), intp(1000), intp(0))},
	}}
	svc := &TeammatesService{titleSlug: "halo_5", gamertag: "main", squadLoader: loader}
	allSquadRows := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
	}
	got := svc.buildSquadPerMinuteStats(context.Background(), allSquadRows, "main", nil, nil)
	if len(got) != 1 || got[0].DamageDealtPerMinute == nil || got[0].DamageTakenPerMinute != nil {
		t.Fatalf("Halo 5 : infligés présents, subis absents attendus, got %+v", got)
	}
}
