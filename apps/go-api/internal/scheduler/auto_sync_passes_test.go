// auto_sync_passes_test.go — la dernière passe de chaque joueur, par titre, telle que l'écran « état de
// l'historique » la lit : alimentée par l'issue du cycle V2 (Halo Infinite) et par syncPlayer (Halo 5).
package scheduler

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/port"
	syncv2 "levelup/go-api/internal/sync/v2"
)

func TestLastPass_IssueDuCycleV2(t *testing.T) {
	cases := []struct {
		name   string
		status string
		want   port.SyncPassStatus
	}{
		{"réussie", "ok", port.SyncPassOK},
		{"partielle", "partial", port.SyncPassPartial},
		{"en échec", "failed", port.SyncPassFailed},
		{"différée", "skipped", port.SyncPassSkipped},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, false)
			s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: c.status, MatchesInserted: 200}
			}}
			if _, ok := s.LastPass("halo_infinite", "JGtm"); ok {
				t.Fatal("une passe est connue avant tout cycle")
			}
			s.RunOnceTrigger(context.Background(), "manual")
			got, ok := s.LastPass("halo_infinite", "JGtm")
			if !ok {
				t.Fatal("aucune passe rangée après le cycle")
			}
			if got.Status != c.want || got.MatchesInserted != 200 || !got.At.Equal(cadenceNow) {
				t.Errorf("passe = %+v, attendu statut %v, 200 matchs, à %v", got, c.want, cadenceNow)
			}
		})
	}
}

// Un cycle V2 en erreur rend une passe en échec pour chaque joueur du moteur.
func TestLastPass_CycleV2EnErreur(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	s.cycleOrchestrator = &scriptedOrch{err: errors.New("orchestrateur indisponible")}
	s.RunOnceTrigger(context.Background(), "manual")
	got, ok := s.LastPass("halo_infinite", "JGtm")
	if !ok || got.Status != port.SyncPassFailed {
		t.Errorf("passe = %+v (connue : %v), attendu une passe en échec", got, ok)
	}
}

// Halo 5 passe par syncPlayer : la passe est rangée sous le titre du joueur, et un même gamertag suivi
// sur deux titres garde une passe par titre.
func TestLastPass_Halo5ParSyncPlayer(t *testing.T) {
	cases := []struct {
		name   string
		result domain.SyncResult
		err    error
		want   port.SyncPassStatus
		added  int
	}{
		{"réussie", domain.SyncResult{MatchesInserted: 3}, nil, port.SyncPassOK, 3},
		{"erreurs partielles", domain.SyncResult{MatchesInserted: 1, Errors: []string{"page 4 perdue"}}, nil, port.SyncPassPartial, 1},
		{"runner en échec", domain.SyncResult{}, errors.New("page perdue"), port.SyncPassFailed, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, true)
			s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
			s.liveRunner = func(ctx context.Context, _, _, _ string) (DeltaRunner, context.Context, func(), error) {
				return stubResultRunner{res: c.result, err: c.err}, ctx, func() {}, nil
			}
			s.RunOnceTrigger(context.Background(), "manual")
			got, ok := s.LastPass(halo5.TitleSlug, "H5Player")
			if !ok || got.Status != c.want || got.MatchesInserted != c.added {
				t.Errorf("passe Halo 5 = %+v (connue : %v), attendu %v avec %d matchs", got, ok, c.want, c.added)
			}
			if _, ok := s.LastPass("halo_infinite", "H5Player"); ok {
				t.Error("la passe Halo 5 ne doit pas apparaître sous Halo Infinite")
			}
			if _, ok := s.LastPass("halo_infinite", "JGtm"); !ok {
				t.Error("la passe Halo Infinite du même cycle est absente")
			}
		})
	}
}

func TestAutoSyncEnabled_SuitLesReglages(t *testing.T) {
	on := newCadenceScheduler(t, settings6h, false)
	if !on.AutoSyncEnabled() {
		t.Error("auto-sync activée dans les réglages : AutoSyncEnabled faux")
	}
	off := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, false)
	if off.AutoSyncEnabled() {
		t.Error("auto-sync désactivée dans les réglages : AutoSyncEnabled vrai")
	}
}
