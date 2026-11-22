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
		k, d, a := cadence(t, e.Player+" frags", e.KillsPerMinute), cadence(t, e.Player+" morts", e.DeathsPerMinute),
			cadence(t, e.Player+" assistances", e.AssistsPerMinute)
		if k != w.k || d != w.d || a != w.a {
			t.Errorf("%s : frags/morts/assistances par minute = %v/%v/%v, attendu %v/%v/%v",
				e.Player, k, d, a, w.k, w.d, w.a)
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
	k, d, a := cadence(t, "friend frags", f.KillsPerMinute), cadence(t, "friend morts", f.DeathsPerMinute),
		cadence(t, "friend assistances", f.AssistsPerMinute)
	if k != 1.2 || d != 0.55 || a != 0.6 {
		t.Errorf("friend : frags/morts/assistances par minute = %v/%v/%v, attendu 1.2/0.55/0.6", k, d, a)
	}
	if f.MatchCount != 2 {
		t.Errorf("friend : %d matchs, attendu 2 (le match au compte inconnu reste compté)", f.MatchCount)
	}
	if m := got["main"]; cadence(t, "main frags", m.KillsPerMinute) != 0.6 {
		t.Errorf("main : frags/min = %v, attendu 0.6 (inchangé)", *m.KillsPerMinute)
	}
}

// Un compte sans aucun match retenu est « inconnu » (nil), jamais zéro : un coéquipier dont tous les
// matchs sont sans durée, ou dont un compte manque partout, garde ses autres cadences et son MatchCount ;
// le joueur principal n'en est pas affecté.
func TestStatsParMinuteEscouade_AucunMatchRetenuEstInconnuPasZero(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	mainRows := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 6, 3, 3, ptr(600), nil, nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 6, 3, 3, ptr(600), nil, nil),
	}
	sansDuree := []canonical.PlayerMatchRow{
		ligneDeCadence("m1", t0, 12, 6, 3, nil, nil, nil),
		ligneDeCadence("m2", t0.Add(time.Hour), 8, 2, 1, ptr(0), nil, nil),
	}
	sansFrags1 := ligneDeCadence("m1", t0, 0, 6, 3, ptr(600), nil, nil)
	sansFrags2 := ligneDeCadence("m2", t0.Add(time.Hour), 0, 2, 1, ptr(600), nil, nil)
	sansFrags1.Self.Kills, sansFrags2.Self.Kills = nil, nil
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", squadLoader: &fakeSquadLoader{
		rowsByGT: map[string][]canonical.PlayerMatchRow{
			"main": mainRows, "sansduree": sansDuree, "sansfrags": {sansFrags1, sansFrags2},
		},
	}}
	squad := []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: t0, Kills: 6, Deaths: 3, Assists: 3, TimePlayedSecs: 600},
		{MatchID: "m2", StartTime: t0.Add(time.Hour), Kills: 6, Deaths: 3, Assists: 3, TimePlayedSecs: 600},
	}

	got := perMinuteByPlayer(svc.buildSquadPerMinuteStats(context.Background(), squad, "main",
		[]string{"sansduree", "sansfrags"}, nil))

	d := got["sansduree"]
	if d.KillsPerMinute != nil || d.DeathsPerMinute != nil || d.AssistsPerMinute != nil {
		t.Errorf("sansduree : aucune durée, cadences attendues inconnues (nil), obtenu %+v", d)
	}
	if d.MatchCount != 2 {
		t.Errorf("sansduree : %d matchs, attendu 2 (reste compté)", d.MatchCount)
	}
	f := got["sansfrags"]
	if f.KillsPerMinute != nil {
		t.Errorf("sansfrags : frags inconnus partout, attendu nil, obtenu %v", *f.KillsPerMinute)
	}
	if cadence(t, "sansfrags morts", f.DeathsPerMinute) != 0.4 || cadence(t, "sansfrags assistances", f.AssistsPerMinute) != 0.2 {
		t.Errorf("sansfrags : morts/assistances = %v/%v, attendu 0.4/0.2", *f.DeathsPerMinute, *f.AssistsPerMinute)
	}
	if cadence(t, "main frags", got["main"].KillsPerMinute) != 0.6 {
		t.Errorf("main : frags/min = %v, attendu 0.6", *got["main"].KillsPerMinute)
	}
}

// Quand aucun joueur n'a de cadence connue, le graphe n'a rien à dire : aucune entrée.
func TestStatsParMinuteEscouade_AucuneCadenceConnueRendNil(t *testing.T) {
	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"}
	squad := []domain.SquadMatchRow{{MatchID: "m1", StartTime: t0, Kills: 5, Deaths: 2, Assists: 1, TimePlayedSecs: 0}}
	if got := svc.buildSquadPerMinuteStats(context.Background(), squad, "main", []string{"friend"}, nil); got != nil {
		t.Errorf("aucune durée nulle part : nil attendu, obtenu %+v", got)
	}
}
