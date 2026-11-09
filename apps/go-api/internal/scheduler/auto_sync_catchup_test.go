// auto_sync_catchup_test.go — le rattrapage d'un cycle hors de l'horloge périodique (ADR 0045 D-10) :
// l'âge de la dernière synchronisation connue (relue du disque au boot) se compare à l'intervalle
// avec une horloge injectée, et une demande de rattrapage tire un tick sans attendre l'intervalle.
package scheduler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/platform/adminstate"
	"levelup/go-api/internal/scheduler"
)

// newSchedulerWithSnapshot rend un ordonnanceur (intervalle 1 h, auto-sync actif) dont le
// instantané post-sync vit dans snapshotPath.
func newSchedulerWithSnapshot(t *testing.T, root, snapshotPath string) *scheduler.AutoSyncScheduler {
	t.Helper()
	return newSchedulerForTest(t, root, nil).WithSnapshotStore(adminstate.NewFileStore(snapshotPath))
}

// TestCatchUpDue_DepuisLeDernierCycleRelueDuDisque : un premier ordonnanceur tourne un cycle et
// l'écrit sur disque ; un second, démarré après, relit cette date et compare son âge à l'intervalle
// avec l'horloge qu'on lui donne.
func TestCatchUpDue_DepuisLeDernierCycleRelueDuDisque(t *testing.T) {
	root := t.TempDir()
	snapshot := filepath.Join(root, "post_sync.json")

	first := newSchedulerWithSnapshot(t, root, snapshot)
	first.RunOnce(context.Background())
	last := first.Snapshot().LastCycleAt
	if last.IsZero() {
		t.Fatal("le premier ordonnanceur n'a pas daté son cycle")
	}

	second := newSchedulerWithSnapshot(t, root, snapshot)
	second.RehydrateFromDisk(context.Background())

	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"cycle d'il y a 10 minutes : à l'heure prévue", last.Round(0).Add(10 * time.Minute), false},
		{"pile l'intervalle : pas encore", last.Round(0).Add(time.Hour), false},
		{"cycle d'il y a 8 heures (poste en veille) : rattrapage", last.Round(0).Add(8 * time.Hour), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			due, got, interval := second.CatchUpDue(c.now)
			if due != c.want {
				t.Errorf("CatchUpDue = %v, attendu %v", due, c.want)
			}
			if interval != time.Hour {
				t.Errorf("intervalle = %v, attendu 1h", interval)
			}
			if !got.Round(0).Equal(last.Round(0)) {
				t.Errorf("dernier cycle = %v, attendu %v (relu du disque)", got, last)
			}
		})
	}
}

// TestCatchUpDue_AucunCycleConnu : une instance neuve (ou un instantané perdu) est en retard.
func TestCatchUpDue_AucunCycleConnu(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil)
	due, last, _ := s.CatchUpDue(time.Now())
	if !due || !last.IsZero() {
		t.Errorf("CatchUpDue = (%v, %v), attendu (true, zéro)", due, last)
	}
}

// TestCatchUpDue_AutoSyncDesactive : un auto-sync coupé dans les réglages n'a rien à rattraper.
func TestCatchUpDue_AutoSyncDesactive(t *testing.T) {
	root := t.TempDir()
	s := newSchedulerForTest(t, root, nil)
	settingsPath := filepath.Join(root, "app_settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"spnkr_auto_sync_enabled": false, "spnkr_auto_sync_interval_hours": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if due, _, _ := s.CatchUpDue(time.Now()); due {
		t.Error("rattrapage dû alors que l'auto-sync est désactivé")
	}
}

// TestRequestCatchUp_TireUnTickImmediat : avec un intervalle d'une heure, une demande de rattrapage
// fait tourner un cycle tout de suite, dans la boucle de Run ; sans demande, rien ne tourne.
func TestRequestCatchUp_TireUnTickImmediat(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	time.Sleep(100 * time.Millisecond)
	if !s.Snapshot().LastCycleAt.IsZero() {
		t.Fatal("un cycle a tourné sans demande, avant l'intervalle")
	}
	if !s.RequestCatchUp("test") {
		t.Fatal("la demande n'a pas été acceptée")
	}
	deadline := time.Now().Add(5 * time.Second)
	for s.Snapshot().LastCycleAt.IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("aucun cycle après la demande de rattrapage")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestRequestCatchUp_NeBloquePasEtSeConfond : deux demandes avant que la boucle les lise se
// confondent, et la demande ne bloque jamais l'appelant.
func TestRequestCatchUp_NeBloquePasEtSeConfond(t *testing.T) {
	s := newSchedulerForTest(t, t.TempDir(), nil) // aucune boucle de Run : rien ne lit
	if !s.RequestCatchUp("a") {
		t.Fatal("première demande refusée")
	}
	if s.RequestCatchUp("b") {
		t.Error("une seconde demande en attente doit se confondre avec la première")
	}
}
