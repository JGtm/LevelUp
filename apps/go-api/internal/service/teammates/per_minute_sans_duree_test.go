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
