package teammates

import (
	"context"
	"strconv"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Témoin de caractérisation des stats par minute de l'escouade : les cadences rendues sur un jeu de
// matchs aux durées inégales (937 s, non multiple de 60), à dégâts connus sur une partie seulement, avec
// un doublon de ligne, un match sans seconde jouée et un match hors session. Les valeurs sont celles que
// rendait le code avant le passage par analysis.PooledPerMinute (chaîne 'g', -1), sauf les frags, morts et
// assistances du joueur principal : le match sans seconde jouée (m3) n'y entre plus au numérateur.

func ptr(v int) *int { return &v }

func ligneDeCadence(id string, ts time.Time, kills, deaths, assists int, secs *int, dealt, taken *int) canonical.PlayerMatchRow {
	row := rowWithStatsXUID("xuid", id, ts, canonical.OutcomeWin, kills, deaths, assists, 0, 40, 50)
	row.Self.Kills, row.Self.Deaths, row.Self.Assists = ptr(kills), ptr(deaths), ptr(assists)
	row.Self.TimePlayed = secs
	row.Self.DamageDealt, row.Self.DamageTaken = dealt, taken
	return row
}

func temoinCadence(t *testing.T, label string, v *float64, want string) {
	t.Helper()
	if v == nil {
		t.Errorf("%s : cadence absente, attendu %s", label, want)
		return
	}
	if got := strconv.FormatFloat(*v, 'g', -1, 64); got != want {
		t.Errorf("%s = %s, attendu %s", label, got, want)
	}
}

func TestTemoinCadence_StatsParMinuteDeLEscouade(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 10, 5, 2, ptr(600), ptr(3000), ptr(2000)),
		ligneDeCadence("m2", t0.Add(time.Hour), 7, 3, 1, ptr(937), nil, ptr(1500)),
		ligneDeCadence("m3", t0.Add(2*time.Hour), 4, 2, 2, nil, ptr(500), ptr(400)),
		ligneDeCadence("m4", t0.Add(3*time.Hour), 99, 99, 99, ptr(600), ptr(9999), ptr(9999)),
	}
	friendRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 4, 8, 6, ptr(600), ptr(1000), nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 2, 1, 3, ptr(937), ptr(250), ptr(300)),
		ligneDeCadence("m4", t0.Add(3*time.Hour), 50, 50, 50, ptr(600), ptr(5000), ptr(5000)),
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{"main": mainRows, "friend": friendRows},
	}}
	squad := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600},
		{MatchID: "m1", StartTime: t0, Kills: 10, Deaths: 5, Assists: 2, TimePlayedSecs: 600}, // doublon
		{MatchID: "m2", StartTime: t0.Add(time.Hour), Kills: 7, Deaths: 3, Assists: 1, TimePlayedSecs: 937},
		{MatchID: "m3", StartTime: t0.Add(2 * time.Hour), Kills: 4, Deaths: 2, Assists: 2, TimePlayedSecs: 0},
		{MatchID: "m4", StartTime: t0.Add(3 * time.Hour), Kills: 99, Deaths: 99, Assists: 99, TimePlayedSecs: 600},
	}
	session := map[string]bool{"m1": true, "m2": true, "m3": true} // m4 hors session
	got := svc.buildSquadPerMinuteStats(context.Background(), squad, "main", []string{"friend"}, session)
	if len(got) != 2 {
		t.Fatalf("2 entrées attendues, obtenu %d", len(got))
	}
	want := map[string]struct {
		matches               int
		k, d, a, dealt, taken string
	}{
		"main":   {3, "0.66", "0.31", "0.12", "300", "136.63"},
		"friend": {2, "0.23", "0.35", "0.35", "48.8", "19.21"},
	}
	for _, e := range got {
		w := want[e.Player]
		if e.MatchCount != w.matches {
			t.Errorf("%s : %d matchs, attendu %d", e.Player, e.MatchCount, w.matches)
		}
		temoinCadence(t, e.Player+" frags", e.KillsPerMinute, w.k)
		temoinCadence(t, e.Player+" morts", e.DeathsPerMinute, w.d)
		temoinCadence(t, e.Player+" assistances", e.AssistsPerMinute, w.a)
		temoinCadence(t, e.Player+" dégâts infligés", e.DamageDealtPerMinute, w.dealt)
		temoinCadence(t, e.Player+" dégâts subis", e.DamageTakenPerMinute, w.taken)
	}
}
