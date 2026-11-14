//go:build integration

package sharedprovider_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// compterLignes rend le nombre de lignes du journal qui contiennent tous les fragments.
func compterLignes(logs *syncBuffer, fragments ...string) int {
	n := 0
	for _, ligne := range strings.Split(logs.String(), "\n") {
		tout := true
		for _, f := range fragments {
			if !strings.Contains(ligne, f) {
				tout = false
				break
			}
		}
		if tout {
			n++
		}
	}
	return n
}

// TestProvider_DepassementWriterUneLigneParEpisode_integration : douze prises successives du
// writer par le même détenteur, chacune au-delà du seuil, écrivent UNE ligne de dépassement et
// UNE ligne de libération (avec les lectures retenues), pas douze ; le compteur voit les douze.
func TestProvider_DepassementWriterUneLigneParEpisode_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetRWHoldWatchdogForTest(p, 30*time.Millisecond)
	sharedprovider.SetReadyTimeoutForTest(p, 5*time.Second)
	logs := captureSlogJSON(t)
	avant := sharedprovider.Snapshot().WatchdogFired

	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_episode_unique")
	const prises = 12
	for i := 0; i < prises; i++ {
		w, aerr := p.AcquireWriter(ctx)
		if aerr != nil {
			t.Fatalf("AcquireWriter %d: %v", i, aerr)
		}
		var wg sync.WaitGroup
		if i == 0 {
			// Un lecteur arrive pendant la première prise : il est retenu, donc compté.
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, release, gerr := p.Get(context.Background()); gerr == nil {
					release()
				}
			}()
		}
		time.Sleep(80 * time.Millisecond)
		w.Release()
		wg.Wait()
	}

	if got := sharedprovider.Snapshot().WatchdogFired - avant; got != prises {
		t.Errorf("compteur du watchdog : +%d, attendu +%d (le compteur voit chaque dépassement)", got, prises)
	}
	if n := compterLignes(logs, "writer RW tenu au-delà du seuil", "test_episode_unique"); n != 1 {
		t.Errorf("lignes de dépassement : %d, attendu 1\n%s", n, logs.String())
	}
	if n := compterLignes(logs, "writer RW libéré après dépassement", "test_episode_unique", `"lectures_retenues":1`); n != 1 {
		t.Errorf("lignes de libération avec 1 lecture retenue : %d, attendu 1\n%s", n, logs.String())
	}
	if n := compterLignes(logs, "writer RW libéré après dépassement", "test_episode_unique"); n != 1 {
		t.Errorf("lignes de libération : %d, attendu 1", n)
	}
}

// TestProvider_LectureLongueUneLigneParAppelant_integration : cinq lectures longues du même
// appelant écrivent une seule ligne.
func TestProvider_LectureLongueUneLigneParAppelant_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetSlowReaderThresholdForTest(p, 20*time.Millisecond)
	logs := captureSlogJSON(t)
	avant := sharedprovider.SlowReadersForTest()

	for i := 0; i < 5; i++ {
		release := lecteurQuiTient(t, p)
		time.Sleep(40 * time.Millisecond)
		release()
	}
	if got := sharedprovider.SlowReadersForTest() - avant; got != 5 {
		t.Errorf("compteur des lectures longues : +%d, attendu +5", got)
	}
	if n := compterLignes(logs, "lecture tenue longtemps"); n != 1 {
		t.Errorf("lignes de lecture longue : %d, attendu 1\n%s", n, logs.String())
	}
}
