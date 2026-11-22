// auto_sync_cadence_paths_test.go — la cadence reçoit l'historique incomplet par CHAQUE chemin de cycle, par
// la boucle de Run elle-même (ticker injecté) : PlayerOutcome.HistoryPending rendu par l'orchestrateur V2
// (Halo Infinite) et domain.SyncResult.HistoryPending rendu par le runner de Halo 5 (syncPlayer). Un cycle qui
// rend « en attente » ramène le suivant à catchUpCycleInterval ; un cycle qui rend « complet » le remet à
// l'intervalle réglé.
package scheduler

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	syncv2 "levelup/go-api/internal/sync/v2"
)

func TestRun_CadenceSuitLHistoriqueParChaqueChemin(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		pending bool
		want    time.Duration
	}{
		{"V2 : PlayerOutcome.HistoryPending vrai", "v2", true, catchUpCycleInterval},
		{"V2 : PlayerOutcome.HistoryPending faux", "v2", false, 6 * time.Hour},
		{"Halo 5 : SyncResult.HistoryPending vrai", "h5", true, catchUpCycleInterval},
		{"Halo 5 : SyncResult.HistoryPending faux", "h5", false, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, c.path == "h5")
			// Le chemin qui n'est pas sous test rend « complet » : seul l'autre peut ramener la cadence à 10 min.
			s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: c.path == "v2" && c.pending}
			}}
			s.liveRunner = func(ctx context.Context, _, _, _ string) (DeltaRunner, context.Context, func(), error) {
				return stubResultRunner{res: domain.SyncResult{HistoryPending: c.path == "h5" && c.pending}}, ctx, func() {}, nil
			}
			tk := newFakeTicker()
			s.newTicker = func(time.Duration) cycleTicker { return tk }

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { s.Run(ctx); close(done) }()
			t.Cleanup(func() { cancel(); <-done })

			tk.ch <- cadenceNow
			waitFor(t, "recalage de la cadence après le tick", func() bool { return len(tk.Resets()) >= 1 })
			if got := tk.Resets(); got[0] != c.want {
				t.Errorf("recalage = %v, attendu %v", got[0], c.want)
			}
		})
	}
}
