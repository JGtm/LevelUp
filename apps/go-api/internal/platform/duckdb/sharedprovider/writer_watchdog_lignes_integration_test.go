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

// Une prise du writer qui n'est pas libérée, sous un épisode de dépassements déjà annoncé, écrit sa
// ligne au second seuil : le journal des épisodes tait le premier dépassement, jamais celle-là. Sa
// libération est journalisée aussi.
func TestProvider_PriseNonLibereeSousEpisodeAnnonceEcritSaLigne_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetRWHoldWatchdogForTest(p, 20*time.Millisecond)
	sharedprovider.SetHoldAlarmForTest(p, 150*time.Millisecond)
	logs := captureSlogJSON(t)
	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_alarme")

	w, err := p.AcquireWriter(ctx) // ouvre l'épisode : annoncé
	if err != nil {
		t.Fatalf("AcquireWriter 1: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	w.Release()

	w, err = p.AcquireWriter(ctx) // même détenteur, épisode en cours : le watchdog est tu
	if err != nil {
		t.Fatalf("AcquireWriter 2: %v", err)
	}
	time.Sleep(400 * time.Millisecond) // jamais libérée pendant ce temps
	if n := compterLignes(logs, "writer RW tenu au-delà du seuil", "test_alarme"); n != 1 {
		t.Errorf("lignes de dépassement = %d, attendu 1 (la seconde prise est tue par l'épisode)", n)
	}
	if n := compterLignes(logs, "writer RW tenu anormalement longtemps", "test_alarme"); n != 1 {
		t.Errorf("lignes d'alarme pendant la tenue = %d, attendu 1 (ligne garantie)\n%s", n, logs.String())
	}
	w.Release()
	if n := compterLignes(logs, "writer RW libéré après dépassement", "test_alarme"); n != 2 {
		t.Errorf("lignes de libération = %d, attendu 2 (la seconde prise a été annoncée par l'alarme)", n)
	}
}

// Une lecture au-delà du second seuil est journalisée même sous un épisode déjà annoncé.
func TestProvider_LectureTresLongueEcriteSousEpisode_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	sharedprovider.SetSlowReaderThresholdForTest(p, 20*time.Millisecond)
	sharedprovider.SetHoldAlarmForTest(p, 150*time.Millisecond)
	logs := captureSlogJSON(t)

	for _, tenue := range []time.Duration{40 * time.Millisecond, 40 * time.Millisecond, 250 * time.Millisecond} {
		release := lecteurQuiTient(t, p)
		time.Sleep(tenue)
		release()
	}
	if n := compterLignes(logs, "lecture tenue longtemps"); n != 2 {
		t.Errorf("lignes de lecture longue = %d, attendu 2 (la première, et la très longue malgré l'épisode)\n%s", n, logs.String())
	}
}

// À la fermeture, le décompte des dépassements tus de l'épisode en cours est écrit.
func TestProvider_CloseRendLesDepassementsTus_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sharedprovider.SetRWHoldWatchdogForTest(p, 20*time.Millisecond)
	logs := captureSlogJSON(t)
	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_close")
	for i := 0; i < 4; i++ {
		w, aerr := p.AcquireWriter(ctx)
		if aerr != nil {
			t.Fatalf("AcquireWriter %d: %v", i, aerr)
		}
		time.Sleep(60 * time.Millisecond)
		w.Release()
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if n := compterLignes(logs, "dépassements tus à la fermeture", "test_close", `"depassements_tus":3`); n != 1 {
		t.Errorf("lignes de décompte à la fermeture = %d, attendu 1 avec 3 dépassements tus\n%s", n, logs.String())
	}
}
