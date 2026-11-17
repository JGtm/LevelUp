package teammates

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// Un match sans durée n'entre ni au numérateur ni au dénominateur des frags, morts et assistances par
// minute (analysis.PooledPerMinute), pour le joueur principal comme pour un coéquipier. Ses comptes
// bruts ne gonflent donc pas la cadence ; il reste compté dans MatchCount.
func TestStatsParMinuteEscouade_MatchSansDureeHorsDuTauxEtDuCompte(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 6, 3, 3, ptr(600), nil, nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 50, 40, 30, nil, nil, nil),
	}
	friendRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 12, 6, 9, ptr(600), nil, nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 70, 60, 50, ptr(0), nil, nil),
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{"main": mainRows, "friend": friendRows},
	}}
	squad := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 6, Deaths: 3, Assists: 3, TimePlayedSecs: 600},
		{MatchID: "m2", StartTime: t0.Add(time.Hour), Kills: 50, Deaths: 40, Assists: 30, TimePlayedSecs: 0},
	}

	got := svc.buildSquadPerMinuteStats(context.Background(), squad, "main", []string{"friend"}, nil)
	if len(got) != 2 {
		t.Fatalf("2 entrées attendues, obtenu %d", len(got))
	}
	want := map[string]struct{ k, d, a float64 }{
		"main":   {0.6, 0.3, 0.3},
		"friend": {1.2, 0.6, 0.9},
	}
	for _, e := range got {
		w := want[e.Player]
		if e.KillsPerMinute != w.k || e.DeathsPerMinute != w.d || e.AssistsPerMinute != w.a {
			t.Errorf("%s : frags/morts/assistances par minute = %v/%v/%v, attendu %v/%v/%v",
				e.Player, e.KillsPerMinute, e.DeathsPerMinute, e.AssistsPerMinute, w.k, w.d, w.a)
		}
		if e.MatchCount != 2 {
			t.Errorf("%s : %d matchs, attendu 2 (le match sans durée reste compté)", e.Player, e.MatchCount)
		}
	}
}

// Un compte inconnu (nil) d'un coéquipier sort le match de CE taux, numérateur et dénominateur : il n'est
// pas un zéro qui tire la cadence vers le bas. Les deux autres comptes du même match, connus, le gardent.
func TestStatsParMinuteEscouade_CompteInconnuDUnCoequipierHorsDuTaux(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 6, 3, 3, ptr(600), nil, nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 6, 3, 3, ptr(600), nil, nil),
	}
	friendM2 := ligneDeCadence("m2", t0.Add(time.Hour), 0, 5, 9, ptr(600), nil, nil)
	friendM2.Self.Kills = nil
	friendRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 12, 6, 3, ptr(600), nil, nil),
		friendM2,
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{"main": mainRows, "friend": friendRows},
	}}
	squad := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 6, Deaths: 3, Assists: 3, TimePlayedSecs: 600},
		{MatchID: "m2", StartTime: t0.Add(time.Hour), Kills: 6, Deaths: 3, Assists: 3, TimePlayedSecs: 600},
	}

	got := perMinuteByPlayer(svc.buildSquadPerMinuteStats(context.Background(), squad, "main", []string{"friend"}, nil))
	f := got["friend"]
	// Frags : m2 inconnu hors du taux, 12 / 10 min = 1,2 (et non 12 / 20 min = 0,6).
	// Morts : 11 / 20 min = 0,55. Assistances : 12 / 20 min = 0,6.
	if f.KillsPerMinute != 1.2 || f.DeathsPerMinute != 0.55 || f.AssistsPerMinute != 0.6 {
		t.Errorf("friend : frags/morts/assistances par minute = %v/%v/%v, attendu 1.2/0.55/0.6",
			f.KillsPerMinute, f.DeathsPerMinute, f.AssistsPerMinute)
	}
	if f.MatchCount != 2 {
		t.Errorf("friend : %d matchs, attendu 2 (le match au compte inconnu reste compté)", f.MatchCount)
	}
	if m := got["main"]; m.KillsPerMinute != 0.6 {
		t.Errorf("main : frags/min = %v, attendu 0.6 (inchangé)", m.KillsPerMinute)
	}
}
