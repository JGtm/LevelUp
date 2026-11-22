// auto_sync_manual_test.go — le cycle demandé à la main (« Synchroniser maintenant ») : même cycle que le
// tick, jamais deux cycles à la fois, lancé même auto-sync désactivée, cadence recalée à sa fin.
package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/port"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// blockingOrch tient le cycle ouvert jusqu'à release ; started dit qu'il a démarré.
type blockingOrch struct {
	started chan struct{}
	release chan struct{}
	pending bool
}

func newBlockingOrch(pending bool) *blockingOrch {
	return &blockingOrch{started: make(chan struct{}, 4), release: make(chan struct{}), pending: pending}
}

func (o *blockingOrch) Run(_ context.Context, profiles []syncv2.PlayerProfile) (syncv2.CycleResult, error) {
	o.started <- struct{}{}
	<-o.release
	res := syncv2.CycleResult{PerPlayer: map[string]syncv2.PlayerOutcome{}}
	for _, p := range profiles {
		res.PerPlayer[p.PlayerSlug] = syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: o.pending}
	}
	return res, nil
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("délai dépassé : %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func bindManual(t *testing.T, s *AutoSyncScheduler) (*sync.WaitGroup, context.CancelFunc) {
	t.Helper()
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	s.BindManualCycles(ctx, &wg)
	t.Cleanup(func() { cancel(); wg.Wait() })
	return &wg, cancel
}

// Le clic lance le même cycle que le tick (RunOnceTrigger, déclencheur manuel), en arrière-plan.
func TestStartSyncNow_LanceLeCycleEnArrierePlan(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := newBlockingOrch(false)
	s.cycleOrchestrator = orch
	wg, _ := bindManual(t, s)

	if err := s.StartSyncNow(); err != nil {
		t.Fatalf("StartSyncNow = %v, attendu nil", err)
	}
	<-orch.started // le cycle tourne pendant que l'appelant a déjà la main
	if !s.cycleRunning.Load() {
		t.Error("cycleRunning faux pendant le cycle")
	}
	close(orch.release)
	wg.Wait()

	hist := s.History()
	if len(hist) != 1 || hist[0].Trigger != "manual" {
		t.Fatalf("historique des cycles = %+v, attendu un cycle déclenché par manual", hist)
	}
	if s.cycleRunning.Load() {
		t.Error("cycleRunning vrai après le cycle")
	}
}

// Un clic pendant un cycle est refusé ; un tick pendant un cycle ne lance rien non plus ; après le cycle,
// un nouveau clic passe.
func TestStartSyncNow_JamaisDeuxCyclesALaFois(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := newBlockingOrch(false)
	s.cycleOrchestrator = orch
	wg, _ := bindManual(t, s)

	if err := s.StartSyncNow(); err != nil {
		t.Fatalf("premier clic = %v", err)
	}
	<-orch.started
	if err := s.StartSyncNow(); !errors.Is(err, port.ErrSyncCycleBusy) {
		t.Errorf("second clic = %v, attendu ErrSyncCycleBusy", err)
	}
	if res := s.RunOnceTrigger(context.Background(), "tick"); !res.busy {
		t.Error("un tick pendant un cycle doit rendre un résultat busy")
	}
	close(orch.release)
	wg.Wait()
	if len(s.History()) != 1 {
		t.Errorf("cycles enregistrés = %d, attendu 1 (ni le second clic ni le tick n'ont tourné)", len(s.History()))
	}

	orch2 := &scriptedOrch{outcome: okOutcome(false)}
	s.cycleOrchestrator = orch2
	if err := s.StartSyncNow(); err != nil {
		t.Errorf("clic après le cycle = %v, attendu nil", err)
	}
	wg.Wait()
	if orch2.calls != 1 {
		t.Errorf("cycles du second orchestrateur = %d, attendu 1", orch2.calls)
	}
}

// Un clic explicite lance le cycle même quand la synchronisation automatique est désactivée.
func TestStartSyncNow_LanceMemeAutoSyncDesactivee(t *testing.T) {
	s := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, false)
	orch := &scriptedOrch{outcome: okOutcome(false)}
	s.cycleOrchestrator = orch
	wg, _ := bindManual(t, s)
	if err := s.StartSyncNow(); err != nil {
		t.Fatalf("StartSyncNow = %v", err)
	}
	wg.Wait()
	if orch.calls != 1 {
		t.Errorf("cycles = %d, attendu 1 malgré l'auto-sync désactivée", orch.calls)
	}
}

// Sans contexte lié, ou serveur en arrêt : indisponible, aucun cycle.
func TestStartSyncNow_Indisponible(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := &scriptedOrch{outcome: okOutcome(false)}
	s.cycleOrchestrator = orch
	if err := s.StartSyncNow(); !errors.Is(err, port.ErrSyncCycleUnavailable) {
		t.Errorf("sans BindManualCycles : %v, attendu ErrSyncCycleUnavailable", err)
	}
	_, cancel := bindManual(t, s)
	cancel()
	if err := s.StartSyncNow(); !errors.Is(err, port.ErrSyncCycleUnavailable) {
		t.Errorf("serveur en arrêt : %v, attendu ErrSyncCycleUnavailable", err)
	}
	if orch.calls != 0 {
		t.Errorf("cycles = %d, attendu 0", orch.calls)
	}
}

// La boucle de Run recale sa cadence à la fin d'un cycle demandé : un historique encore incomplet
// ramène le cycle suivant à 10 minutes ; auto-sync désactivée, rien ne change.
func TestStartSyncNow_RecaleLaCadenceDeLaBoucle(t *testing.T) {
	cases := []struct {
		name     string
		settings string
		want     []time.Duration
	}{
		{"auto-sync activée", settings6h, []time.Duration{catchUpCycleInterval}},
		{"auto-sync désactivée", `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, c.settings, false)
			s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(true)}
			tk := newFakeTicker()
			s.newTicker = func(time.Duration) cycleTicker { return tk }
			wg, _ := bindManual(t, s)

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() { s.Run(ctx); close(done) }()
			t.Cleanup(func() { cancel(); <-done })

			if err := s.StartSyncNow(); err != nil {
				t.Fatalf("StartSyncNow = %v", err)
			}
			wg.Wait()
			if c.want != nil {
				waitFor(t, "recalage de la cadence", func() bool { return len(tk.Resets()) >= 1 })
			} else {
				time.Sleep(100 * time.Millisecond) // le signal, s'il partait, serait lu en quelques ms
			}
			got := tk.Resets()
			if len(got) != len(c.want) || (len(got) == 1 && got[0] != c.want[0]) {
				t.Errorf("recalages = %v, attendu %v", got, c.want)
			}
		})
	}
}

// Un tick tombant pendant un cycle demandé ne lance rien et ne recale rien : le cycle demandé le fera.
func TestRunTick_PendantUnCycleDemande(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := newBlockingOrch(false)
	s.cycleOrchestrator = orch
	wg, _ := bindManual(t, s)
	if err := s.StartSyncNow(); err != nil {
		t.Fatal(err)
	}
	<-orch.started
	tk := newFakeTicker()
	runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
	if got := tk.Resets(); len(got) != 0 {
		t.Errorf("recalages = %v, attendu aucun pendant un cycle demandé", got)
	}
	close(orch.release)
	wg.Wait()
}
