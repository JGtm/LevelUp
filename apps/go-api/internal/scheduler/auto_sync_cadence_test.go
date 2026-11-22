// auto_sync_cadence_test.go — la cadence de la boucle de Run : intervalle réglé tant que tous les
// historiques sont complets, catchUpCycleInterval dès qu'un joueur reste en rattrapage (Halo
// Infinite par le pipeline V2, Halo 5 par le runner live), retour à l'intervalle réglé ensuite.
// Le ticker et l'horloge sont injectés : aucun test n'attend du temps réel.
package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	halo5 "levelup/go-api/internal/games/halo_5"
	settings_platform "levelup/go-api/internal/platform/settings"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// fakeTicker enregistre les recalages demandés par la boucle ; Tick() tire un tick à la main.
type fakeTicker struct {
	ch      chan time.Time
	mu      sync.Mutex
	resets  []time.Duration
	stopped bool
}

func newFakeTicker() *fakeTicker { return &fakeTicker{ch: make(chan time.Time, 1)} }

func (f *fakeTicker) C() <-chan time.Time { return f.ch }
func (f *fakeTicker) Stop()               { f.mu.Lock(); f.stopped = true; f.mu.Unlock() }
func (f *fakeTicker) Reset(d time.Duration) {
	f.mu.Lock()
	f.resets = append(f.resets, d)
	f.mu.Unlock()
}
func (f *fakeTicker) Resets() []time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Duration(nil), f.resets...)
}

// fakeHistoryReader : l'état de l'historique lu par gamertag ; absent = complet.
type fakeHistoryReader struct {
	incomplete map[string]bool
	err        error
}

func (f *fakeHistoryReader) HistoryComplete(_ context.Context, _, gamertag string) (bool, error) {
	if f.err != nil {
		return true, f.err
	}
	return !f.incomplete[gamertag], nil
}

// scriptedOrch rend l'issue que le test fixe pour chaque joueur reçu (clé PlayerSlug) ; err non nil
// fait échouer le cycle entier.
type scriptedOrch struct {
	calls   int
	outcome func(p syncv2.PlayerProfile) syncv2.PlayerOutcome
	err     error
}

func (o *scriptedOrch) Run(_ context.Context, profiles []syncv2.PlayerProfile) (syncv2.CycleResult, error) {
	o.calls++
	if o.err != nil {
		return syncv2.CycleResult{}, o.err
	}
	res := syncv2.CycleResult{PerPlayer: map[string]syncv2.PlayerOutcome{}}
	for _, p := range profiles {
		res.PerPlayer[p.PlayerSlug] = o.outcome(p)
	}
	return res, nil
}

var cadenceNow = time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

// newCadenceScheduler construit un ordonnanceur sur un profil Halo Infinite (et Halo 5 si withH5),
// auto-sync réglé à settingsJSON, horloge figée sur cadenceNow.
func newCadenceScheduler(t *testing.T, settingsJSON string, withH5 bool) *AutoSyncScheduler {
	t.Helper()
	repoRoot := t.TempDir()
	settingsPath := filepath.Join(repoRoot, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	h5 := ""
	if withH5 {
		h5 = `, "halo_5": {"H5Player": {"db_path": "h5", "xuid": "2222222222222222", "waypoint_player": "H5Player"}}`
	}
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")
	if err := os.WriteFile(profilesPath, []byte(`{"version": "3.0", "profiles": {
		"halo_infinite": {"JGtm": {"db_path": "hi", "xuid": "1111111111111111", "waypoint_player": "JGtm"}}`+h5+`}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &config.AppConfig{RepoRoot: repoRoot, AppSettingsPath: settingsPath, DBProfilesPath: profilesPath}
	s := New(cfg, settings_platform.NewStore(settingsPath), nil, &h5FakePool{has: map[string]bool{"JGtm": true, "H5Player": true}})
	s.now = func() time.Time { return cadenceNow }
	return s
}

const settings6h = `{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":6}`

func okOutcome(pending bool) func(syncv2.PlayerProfile) syncv2.PlayerOutcome {
	return func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: pending}
	}
}

func runOneTick(t *testing.T, s *AutoSyncScheduler, st *tickState, tk *fakeTicker) {
	t.Helper()
	s.runTick(context.Background(), st, tk, "")
}

// Le scénario du plan : intervalle réglé (6 h), un joueur en rattrapage ramène le cycle suivant à
// 10 minutes, la fin d'historique rangée le remet à 6 h.
func TestCadence_SixHeuresDixMinutesSixHeures(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := &scriptedOrch{outcome: okOutcome(true)}
	s.cycleOrchestrator = orch
	tk := newFakeTicker()
	st := &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}

	runOneTick(t, s, st, tk) // cycle 1 : l'historique reste incomplet
	orch.outcome = okOutcome(true)
	runOneTick(t, s, st, tk) // cycle 2 : toujours incomplet
	orch.outcome = okOutcome(false)
	runOneTick(t, s, st, tk) // cycle 3 : fin d'historique rangée

	want := []time.Duration{catchUpCycleInterval, catchUpCycleInterval, 6 * time.Hour}
	got := tk.Resets()
	if len(got) != len(want) {
		t.Fatalf("recalages = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("recalage %d = %v, attendu %v", i, got[i], want[i])
		}
	}
	if st.period != 6*time.Hour {
		t.Errorf("période après le dernier cycle = %v, attendu 6h", st.period)
	}
	if next := s.NextCycleAt(); !next.Equal(cadenceNow.Add(6 * time.Hour)) {
		t.Errorf("NextCycleAt = %v, attendu %v", next, cadenceNow.Add(6*time.Hour))
	}
}

func TestCadence_ProchainCycleDansDixMinutes(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(true)}
	tk := newFakeTicker()
	st := &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}
	runOneTick(t, s, st, tk)
	if next := s.NextCycleAt(); !next.Equal(cadenceNow.Add(catchUpCycleInterval)) {
		t.Errorf("NextCycleAt = %v, attendu dans 10 minutes", next)
	}
}

// Un intervalle réglé plus court que celui du rattrapage garde la main.
func TestCadence_IntervalleRegleCourtGardeLaMain(t *testing.T) {
	s := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_minutes":5}`, false)
	s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(true)}
	tk := newFakeTicker()
	st := &tickState{configured: 5 * time.Minute, period: 5 * time.Minute}
	runOneTick(t, s, st, tk)
	if got := tk.Resets(); len(got) != 1 || got[0] != 5*time.Minute {
		t.Errorf("recalages = %v, attendu [5m]", got)
	}
}

// Un joueur dont la passe a échoué avant de pouvoir le dire compte comme en rattrapage s'il n'a pas
// la clé de fin ; avec la clé, ou sans lecteur câblé, il n'y compte pas.
func TestCadence_JoueurEnEchecSelonLaCleDeFin(t *testing.T) {
	cases := []struct {
		name   string
		reader *fakeHistoryReader
		want   time.Duration
	}{
		{"sans la clé de fin", &fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}}, catchUpCycleInterval},
		{"avec la clé de fin", &fakeHistoryReader{}, 6 * time.Hour},
		{"lecture en échec", &fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}, err: errors.New("base verrouillée")}, 6 * time.Hour},
		{"sans lecteur", nil, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run("issue en échec : "+c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, false)
			if c.reader != nil {
				s.WithHistoryReader(c.reader)
			}
			s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "failed", FirstError: "découverte en échec"}
			}}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.want {
				t.Errorf("recalages = %v, attendu [%v]", got, c.want)
			}
		})
		t.Run("erreur de cycle : "+c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, false)
			if c.reader != nil {
				s.WithHistoryReader(c.reader)
			}
			s.cycleOrchestrator = &scriptedOrch{err: errors.New("orchestrateur indisponible")}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.want {
				t.Errorf("recalages = %v, attendu [%v]", got, c.want)
			}
		})
	}
}

// Une passe réussie fait foi : HistoryPending faux = complet, même si la base dirait le contraire
// (la fin vient d'être rangée par la passe).
func TestCadence_PasseReussieFaitFoi(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	s.WithHistoryReader(&fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}})
	s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
	tk := newFakeTicker()
	runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
	if got := tk.Resets(); len(got) != 1 || got[0] != 6*time.Hour {
		t.Errorf("recalages = %v, attendu [6h]", got)
	}
}

// Halo 5 passe par le runner live (syncPlayer), pas par l'orchestrateur V2 : son HistoryPending
// ramène le cycle à 10 minutes, et un échec du runner suit la clé de fin.
func TestCadence_JoueurHalo5(t *testing.T) {
	cases := []struct {
		name    string
		result  domain.SyncResult
		runErr  error
		reader  *fakeHistoryReader
		wantDur time.Duration
	}{
		{"passe réussie, historique en attente", domain.SyncResult{HistoryPending: true}, nil, nil, catchUpCycleInterval},
		{"passe réussie, historique complet", domain.SyncResult{}, nil, nil, 6 * time.Hour},
		{"runner en échec, sans clé de fin", domain.SyncResult{}, errors.New("page perdue"),
			&fakeHistoryReader{incomplete: map[string]bool{"H5Player": true}}, catchUpCycleInterval},
		{"runner en échec, clé de fin rangée", domain.SyncResult{}, errors.New("page perdue"), &fakeHistoryReader{}, 6 * time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, settings6h, true)
			if c.reader != nil {
				s.WithHistoryReader(c.reader)
			}
			s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
			s.liveRunner = func(ctx context.Context, slug, _, _ string) (DeltaRunner, context.Context, func(), error) {
				if slug != halo5.TitleSlug {
					t.Fatalf("runner live demandé pour %q", slug)
				}
				return stubResultRunner{res: c.result, err: c.runErr}, ctx, func() {}, nil
			}
			tk := newFakeTicker()
			runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
			if got := tk.Resets(); len(got) != 1 || got[0] != c.wantDur {
				t.Errorf("recalages = %v, attendu [%v]", got, c.wantDur)
			}
		})
	}
}

type stubResultRunner struct {
	res domain.SyncResult
	err error
}

func (r stubResultRunner) RunDelta(context.Context, domain.SyncOptions) (domain.SyncResult, error) {
	return r.res, r.err
}

// Auto-sync désactivée : aucun cycle, aucun recalage ; l'intervalle réglé reste en vigueur.
func TestCadence_AutoSyncDesactiveeAucunCycle(t *testing.T) {
	s := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, false)
	orch := &scriptedOrch{outcome: okOutcome(true)}
	s.cycleOrchestrator = orch
	tk := newFakeTicker()
	runOneTick(t, s, &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}, tk)
	if orch.calls != 0 {
		t.Errorf("l'orchestrateur a tourné %d fois alors que l'auto-sync est désactivée", orch.calls)
	}
	if got := tk.Resets(); len(got) != 0 {
		t.Errorf("recalages = %v, attendu aucun", got)
	}
}

// L'intervalle réglé change entre deux ticks : le cycle suivant le suit.
func TestCadence_IntervalleRegleChange(t *testing.T) {
	s := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_hours":2}`, false)
	s.cycleOrchestrator = &scriptedOrch{outcome: okOutcome(false)}
	tk := newFakeTicker()
	st := &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}
	runOneTick(t, s, st, tk)
	if got := tk.Resets(); len(got) != 1 || got[0] != 2*time.Hour {
		t.Errorf("recalages = %v, attendu [2h]", got)
	}
}

// La boucle de Run elle-même : ticker créé à l'intervalle réglé, prochain cycle daté, un tick tire un
// cycle puis recale sur le rattrapage, l'annulation arrête la boucle et le ticker.
func TestRun_TickerInjecte(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	orch := &scriptedOrch{outcome: okOutcome(true)}
	s.cycleOrchestrator = orch
	tk := newFakeTicker()
	var created []time.Duration
	s.newTicker = func(d time.Duration) cycleTicker { created = append(created, d); return tk }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	tk.ch <- cadenceNow
	deadline := time.Now().Add(5 * time.Second)
	for len(tk.Resets()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("aucun recalage après le tick")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done

	if len(created) != 1 || created[0] != 6*time.Hour {
		t.Errorf("ticker créé à %v, attendu [6h]", created)
	}
	if got := tk.Resets(); len(got) != 1 || got[0] != catchUpCycleInterval {
		t.Errorf("recalages = %v, attendu [10m]", got)
	}
	if orch.calls != 1 {
		t.Errorf("cycles = %d, attendu 1", orch.calls)
	}
	tk.mu.Lock()
	stopped := tk.stopped
	tk.mu.Unlock()
	if !stopped {
		t.Error("le ticker n'a pas été arrêté à l'annulation")
	}
}

// Avant le démarrage de la boucle, aucun prochain cycle n'est connu ; au démarrage il est daté.
func TestNextCycleAt_DateAuDemarrage(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	if !s.NextCycleAt().IsZero() {
		t.Error("NextCycleAt non nul avant le démarrage de la boucle")
	}
	tk := newFakeTicker()
	s.newTicker = func(time.Duration) cycleTicker { return tk }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for s.NextCycleAt().IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("NextCycleAt jamais daté")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done
	if want := cadenceNow.Add(6 * time.Hour); !s.NextCycleAt().Equal(want) {
		t.Errorf("NextCycleAt = %v, attendu %v", s.NextCycleAt(), want)
	}
}
