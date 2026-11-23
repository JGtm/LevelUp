// auto_sync_progress_test.go — la garde de progrès de la cadence de rattrapage : un joueur en échec
// permanent ne tient pas la cadence courte pour toujours (10 min tant que les cycles progressent, au plus
// catchUpStallLimit cycles sans progrès), et un démarrage avec une dette avance le premier cycle.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	syncv2 "levelup/go-api/internal/sync/v2"
)

// fakeMarkerReader : la position de l'historique lue par gamertag ; advance la fait avancer comme une
// passe de rattrapage qui range un marque-page.
type fakeMarkerReader struct {
	mu  sync.Mutex
	m   map[string]port.HistoryMarker
	err error
}

func (f *fakeMarkerReader) ReadHistoryMarker(_ context.Context, _, gamertag string) (port.HistoryMarker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.m[gamertag], f.err
}

func (f *fakeMarkerReader) advance(gamertag string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.m == nil {
		f.m = map[string]port.HistoryMarker{}
	}
	cur := f.m[gamertag]
	cur.Offset += 200
	cur.LastMatchID = fmt.Sprintf("match-%d", cur.Offset)
	f.m[gamertag] = cur
}

// stalledScheduler : un joueur Halo Infinite dont l'historique reste incomplet (clé de fin absente) et dont
// la passe ne range rien ; le test règle ce que chaque cycle fait par `outcome`.
func stalledScheduler(t *testing.T) (*AutoSyncScheduler, *scriptedOrch, *fakeTicker, *tickState) {
	t.Helper()
	s := newCadenceScheduler(t, settings6h, false)
	s.WithHistoryReader(&fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}})
	orch := &scriptedOrch{outcome: okOutcome(true)}
	s.cycleOrchestrator = orch
	return s, orch, newFakeTicker(), &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}
}

func wantResets(t *testing.T, tk *fakeTicker, want ...time.Duration) {
	t.Helper()
	got := tk.Resets()
	if len(got) != len(want) {
		t.Fatalf("recalages = %v, attendu %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("recalage %d = %v, attendu %v (tous : %v)", i, got[i], want[i], got)
		}
	}
}

// Un joueur en échec permanent (aucun marque-page qui bouge, aucun film, aucun match rangé) : 10 min, 10 min,
// 10 min, puis l'intervalle réglé, et il y reste tant que rien ne progresse.
func TestProgres_EchecPermanentRetourneALIntervalleRegle(t *testing.T) {
	s, orch, tk, st := stalledScheduler(t)
	s.WithHistoryMarkerReader(&fakeMarkerReader{})
	orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "failed", FirstError: "jeton mort"}
	}
	for range 6 {
		runOneTick(t, s, st, tk)
	}
	wantResets(t, tk, catchUpCycleInterval, catchUpCycleInterval, catchUpCycleInterval, 6*time.Hour, 6*time.Hour, 6*time.Hour)
	if st.stalled != 6 {
		t.Errorf("cycles sans progrès = %d, attendu 6", st.stalled)
	}
}

// Le progrès retrouvé (le marque-page avance) remet la cadence à 10 minutes et le compteur à zéro ; la fin
// d'historique rangée solde la dette.
func TestProgres_ProgresRetrouveRemetLaCadenceCourte(t *testing.T) {
	s, orch, tk, st := stalledScheduler(t)
	markers := &fakeMarkerReader{}
	s.WithHistoryMarkerReader(markers)
	for range 4 { // trois cycles à 10 min, le quatrième retombe à l'intervalle réglé
		runOneTick(t, s, st, tk)
	}
	orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		markers.advance(p.Gamertag) // la passe range un marque-page
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: true}
	}
	runOneTick(t, s, st, tk)
	if st.stalled != 0 {
		t.Errorf("compteur après un progrès = %d, attendu 0", st.stalled)
	}
	orch.outcome = okOutcome(true) // plus rien ne bouge : un nouveau compte part de zéro
	runOneTick(t, s, st, tk)
	orch.outcome = okOutcome(false) // fin d'historique rangée : dette soldée
	runOneTick(t, s, st, tk)
	wantResets(t, tk,
		catchUpCycleInterval, catchUpCycleInterval, catchUpCycleInterval, 6*time.Hour, // sans progrès
		catchUpCycleInterval, // progrès retrouvé
		catchUpCycleInterval, // premier cycle sans progrès après lui
		6*time.Hour)          // dette soldée
	if st.stalled != 0 {
		t.Errorf("compteur après la dette soldée = %d, attendu 0", st.stalled)
	}
}

// Tant que le marque-page avance à chaque cycle, la cadence courte tient sans limite.
func TestProgres_MarquePageQuiAvanceTientLaCadenceCourte(t *testing.T) {
	s, orch, tk, st := stalledScheduler(t)
	markers := &fakeMarkerReader{}
	s.WithHistoryMarkerReader(markers)
	orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		markers.advance(p.Gamertag)
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: true}
	}
	for range 8 {
		runOneTick(t, s, st, tk)
	}
	for i, d := range tk.Resets() {
		if d != catchUpCycleInterval {
			t.Errorf("recalage %d = %v, attendu %v", i, d, catchUpCycleInterval)
		}
	}
}

// Les deux autres signaux suffisent seuls : un film écrit, ou un match rangé par un joueur resté en
// rattrapage (position illisible ou lecteur non câblé).
func TestProgres_FilmEcritOuMatchRangeSuffisent(t *testing.T) {
	cases := []struct {
		name  string
		setup func(s *AutoSyncScheduler, orch *scriptedOrch)
	}{
		{"film écrit", func(s *AutoSyncScheduler, orch *scriptedOrch) {
			films := fakeFilmBacklog{}
			s.WithFilmBacklogReader(films)
			orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				b := films["halo_infinite"]
				b.Ecrits += 2
				films["halo_infinite"] = b
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", HistoryPending: true}
			}
		}},
		{"match rangé, sans lecteur de position", func(_ *AutoSyncScheduler, orch *scriptedOrch) {
			orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", MatchesInserted: 5, HistoryPending: true}
			}
		}},
		{"match rangé, position illisible", func(s *AutoSyncScheduler, orch *scriptedOrch) {
			s.WithHistoryMarkerReader(&fakeMarkerReader{err: errors.New("base verrouillée")})
			orch.outcome = func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
				return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "ok", MatchesInserted: 5, HistoryPending: true}
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, orch, tk, st := stalledScheduler(t)
			c.setup(s, orch)
			for range 5 {
				runOneTick(t, s, st, tk)
			}
			for i, d := range tk.Resets() {
				if d != catchUpCycleInterval {
					t.Errorf("recalage %d = %v, attendu %v", i, d, catchUpCycleInterval)
				}
			}
		})
	}
}

// Des nouveaux matchs d'un joueur dont l'historique est complet ne sont pas un progrès du rattrapage : ils
// ne tiennent pas la cadence courte d'un autre joueur en échec permanent.
func TestProgres_NouveauxMatchsDUnJoueurCompletNeCompentPas(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, true) // JGtm (Infinite) et H5Player (Halo 5)
	s.WithHistoryReader(&fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}})
	s.cycleOrchestrator = &scriptedOrch{outcome: func(p syncv2.PlayerProfile) syncv2.PlayerOutcome {
		return syncv2.PlayerOutcome{Gamertag: p.Gamertag, Status: "failed", HistoryPending: true}
	}}
	s.liveRunner = func(ctx context.Context, _, _, _ string) (DeltaRunner, context.Context, func(), error) {
		return stubResultRunner{res: syncResultInserted(4)}, ctx, func() {}, nil // joueur H5 complet, actif
	}
	tk, st := newFakeTicker(), &tickState{configured: 6 * time.Hour, period: 6 * time.Hour}
	for range 5 {
		runOneTick(t, s, st, tk)
	}
	wantResets(t, tk, catchUpCycleInterval, catchUpCycleInterval, catchUpCycleInterval, 6*time.Hour, 6*time.Hour)
}

// Un cycle demandé à la main remet le compteur à zéro et la cadence courte repart de la dette.
func TestProgres_CycleManuelRemetLeCompteurAZero(t *testing.T) {
	s, _, tk, st := stalledScheduler(t)
	st.stalled = 5
	st.period = 6 * time.Hour
	s.replanFromCycle(context.Background(), st, tk, cycleDebt{history: []string{"JGtm"}})
	wantResets(t, tk, catchUpCycleInterval)
	if st.stalled != 0 {
		t.Errorf("compteur après un cycle manuel = %d, attendu 0", st.stalled)
	}
}

// Redémarrage avec un joueur en rattrapage : le premier cycle vient dans les 10 minutes, tout de suite si le
// dernier cycle connu est plus ancien, jamais plus tard que l'intervalle réglé.
func TestDemarrage_PremierCycleAvanceQuandUneDetteReste(t *testing.T) {
	incomplete := &fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}}
	cases := []struct {
		name     string
		settings string
		reader   *fakeHistoryReader
		lastAgo  time.Duration // 0 : aucun cycle connu
		films    port.FilmBacklogReader
		want     time.Duration
	}{
		{"historique incomplet, aucun cycle connu", settings6h, incomplete, 0, nil, catchUpCycleInterval},
		{"historique incomplet, dernier cycle il y a 3 min", settings6h, incomplete, 3 * time.Minute, nil, 7 * time.Minute},
		{"historique incomplet, dernier cycle il y a 1 h", settings6h, incomplete, time.Hour, nil, startupCatchUpMinDelay},
		{"lecture de l'état en échec = incomplet", settings6h, &fakeHistoryReader{err: errors.New("illisible")}, time.Hour, nil, startupCatchUpMinDelay},
		{"arriéré de films seul", settings6h, &fakeHistoryReader{}, 0, fakeFilmBacklog{"halo_infinite": {Frags: 3}}, catchUpCycleInterval},
		{"historique complet, aucun film", settings6h, &fakeHistoryReader{}, time.Hour, nil, 6 * time.Hour},
		{"sans lecteur d'historique", settings6h, nil, time.Hour, nil, 6 * time.Hour},
		{"auto-sync désactivée", `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, incomplete, time.Hour, nil, 6 * time.Hour},
		{"intervalle réglé court", `{"spnkr_auto_sync_enabled":true,"spnkr_auto_sync_interval_minutes":5}`, incomplete, time.Hour, nil, 5 * time.Minute},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCadenceScheduler(t, c.settings, false)
			if c.reader != nil {
				s.WithHistoryReader(c.reader)
			}
			if c.films != nil {
				s.WithFilmBacklogReader(c.films)
			}
			if c.lastAgo > 0 {
				s.lastCycleAt = cadenceNow.Add(-c.lastAgo)
			}
			if got := s.firstCycleDelay(context.Background(), s.CurrentInterval()); got != c.want {
				t.Errorf("délai du premier cycle = %v, attendu %v", got, c.want)
			}
		})
	}
}

// La boucle de Run arme son ticker sur ce délai et date le prochain cycle en conséquence.
func TestRun_RedemarrageAvecJoueurEnRattrapage(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	s.WithHistoryReader(&fakeHistoryReader{incomplete: map[string]bool{"JGtm": true}})
	s.lastCycleAt = cadenceNow.Add(-4 * time.Minute)
	tk := newFakeTicker()
	var created []time.Duration
	s.newTicker = func(d time.Duration) cycleTicker { created = append(created, d); return tk }

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
	if len(created) != 1 || created[0] != 6*time.Minute {
		t.Errorf("ticker créé à %v, attendu [6m] (10 min moins les 4 min du dernier cycle)", created)
	}
	if want := cadenceNow.Add(6 * time.Minute); !s.NextCycleAt().Equal(want) {
		t.Errorf("NextCycleAt = %v, attendu %v", s.NextCycleAt(), want)
	}
}

// Auto-sync désactivée après un démarrage avec dette : le ticker ne garde pas la période courte.
func TestRunTick_DesactiveeReprendLIntervalleRegle(t *testing.T) {
	s := newCadenceScheduler(t, `{"spnkr_auto_sync_enabled":false,"spnkr_auto_sync_interval_hours":6}`, false)
	tk := newFakeTicker()
	st := &tickState{configured: 6 * time.Hour, period: startupCatchUpMinDelay}
	runOneTick(t, s, st, tk)
	wantResets(t, tk, 6*time.Hour)
	if st.period != 6*time.Hour {
		t.Errorf("période = %v, attendu 6h", st.period)
	}
}

// L'arriéré de films non suivi (lecteur non branché) ne se journalise pas comme « 0 film en retard ».
func TestFilmBacklogAttrs_NonSuivi(t *testing.T) {
	s := newCadenceScheduler(t, settings6h, false)
	if got := s.filmBacklogAttrs(port.FilmBacklog{}); len(got) != 2 || got[0] != "films" || got[1] != "non_suivi" {
		t.Errorf("sans lecteur : %v, attendu [films non_suivi]", got)
	}
	s.WithFilmBacklogReader(fakeFilmBacklog{})
	got := s.filmBacklogAttrs(port.FilmBacklog{Frags: 2, Rejeux: 1})
	if len(got) != 6 || got[0] != "films_en_retard" || got[1] != 3 {
		t.Errorf("avec lecteur : %v, attendu films_en_retard=3 et le détail", got)
	}
}

func syncResultInserted(n int) domain.SyncResult { return domain.SyncResult{MatchesInserted: n} }
