// auto_sync_notify_catchup_test.go — la notification de fin de cycle ne compte que les NOUVEAUX matchs, pas
// ceux que le rattrapage de l'historique a rangés.
package scheduler

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/notify"
	syncv2 "levelup/go-api/internal/sync/v2"
)

func TestNewMatchesOf(t *testing.T) {
	cases := []struct{ inserted, catchUp, want int }{
		{200, 200, 0}, {205, 200, 5}, {3, 0, 3}, {0, 0, 0}, {10, 25, 0},
	}
	for _, c := range cases {
		if got := newMatchesOf(c.inserted, c.catchUp); got != c.want {
			t.Errorf("newMatchesOf(%d, %d) = %d, attendu %d", c.inserted, c.catchUp, got, c.want)
		}
	}
}

// Pipeline V2 : une passe de rattrapage pure n'annonce rien ; une passe qui range aussi des nouveaux matchs
// n'annonce que ceux-là.
func TestRunOnceV2_NotificationSansLesMatchsRattrapes(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", MatchesInserted: 205, CatchUpInserted: 200}
	}}
	players, err := s.cfg.LoadPlayers()
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.runOnceV2(context.Background(), players)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.newMatchPlayers) != 1 || res.newMatchPlayers[0].MatchesSynced != 5 {
		t.Errorf("annoncés = %+v, attendu 5 nouveaux matchs", res.newMatchPlayers)
	}
	if res.inserted["JGtm"] != 205 {
		t.Errorf("insérés = %v, attendu 205 (la garde de progrès les compte tous)", res.inserted)
	}

	s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", MatchesInserted: 200, CatchUpInserted: 200}
	}}
	res, err = s.runOnceV2(context.Background(), players)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.newMatchPlayers) != 0 {
		t.Errorf("annoncés = %+v, attendu rien pour une passe de rattrapage pure", res.newMatchPlayers)
	}
}

// Joueurs live-only (Halo 5) et filet : la fusion lit le détail du joueur, rattrapage retiré.
func TestMergeCycleNewMatchPlayers_SansLeRattrapage(t *testing.T) {
	start := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	snapshot := []PlayerOutcomeDetail{
		{Gamertag: "H5Pur", Outcome: "ok", AttemptedAt: start, MatchesInserted: 25, CatchUpInserted: 25},
		{Gamertag: "H5Mixte", Outcome: "ok", AttemptedAt: start, MatchesInserted: 27, CatchUpInserted: 25},
		{Gamertag: "H5Delta", Outcome: "ok", AttemptedAt: start, MatchesInserted: 3},
	}
	got := mergeCycleNewMatchPlayers(start, []notify.PlayerSyncResult{{Gamertag: "JGtm", MatchesSynced: 2}}, snapshot)
	want := map[string]int{"JGtm": 2, "H5Mixte": 2, "H5Delta": 3}
	if len(got) != len(want) {
		t.Fatalf("annoncés = %+v, attendu %v", got, want)
	}
	for _, p := range got {
		if want[p.Gamertag] != p.MatchesSynced {
			t.Errorf("%s : %d, attendu %d", p.Gamertag, p.MatchesSynced, want[p.Gamertag])
		}
	}
}
