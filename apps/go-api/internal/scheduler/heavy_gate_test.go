package scheduler

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/platform/gameproc"
)

// pendingGame installe un mode jeu actif pour le test et le rend à la fin.
func pendingGame(t *testing.T) *gameproc.Mode {
	t.Helper()
	m := gameproc.NewModeRetrying(5 * time.Millisecond)
	prev := gameproc.SetDefault(m)
	t.Cleanup(func() { gameproc.SetDefault(prev) })
	return m
}

func TestRunHeavy_HorsPartieLeCycleSeLanceToutDeSuite(t *testing.T) {
	pendingGame(t) // mode créé inactif
	ran := 0
	runHeavy(context.Background(), "test", func(context.Context) { ran++ })
	if ran != 1 {
		t.Fatalf("cycle lancé %d fois, attendu 1", ran)
	}
}

func TestRunHeavy_PendantLaPartieLeCycleAttendSaFin(t *testing.T) {
	m := pendingGame(t)
	// Mode actif : on simule l'entrée en partie par le watcher, puis la sortie.
	w := &gameproc.Watcher{
		Lister: gameproc.ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(context.Background())
	if m.HeavyWorkAllowed() {
		t.Fatal("précondition : le jeu tourne")
	}

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runHeavy(context.Background(), "test", func(context.Context) { close(started) })
		close(done)
	}()
	select {
	case <-started:
		t.Fatal("le cycle ne doit pas démarrer pendant la partie")
	case <-time.After(150 * time.Millisecond):
	}

	// La partie se termine : le prochain relevé de la porte libère le cycle.
	w.Lister = gameproc.ListerFunc(func() ([]string, error) { return []string{"explorer.exe"}, nil })
	w.Poll(context.Background())
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("le cycle différé doit démarrer à la fin de la partie")
	}
	<-done
}

func TestRunHeavy_AnnulationPendantLAttenteNeLancePas(t *testing.T) {
	m := pendingGame(t)
	w := &gameproc.Watcher{
		Lister: gameproc.ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	ran := false
	done := make(chan struct{})
	go func() {
		runHeavy(ctx, "test", func(context.Context) { ran = true })
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("l'annulation doit libérer l'attente")
	}
	if ran {
		t.Fatal("un cycle différé puis annulé ne se lance pas")
	}
}

// TestGardeRail_LesBouclesLourdesPassentParRunHeavy : les quatre boucles de tâches lourdes ne
// lancent jamais leur cycle directement, elles passent par la porte du mode jeu. Une cinquième
// boucle lourde s'ajoute ici.
func TestGardeRail_LesBouclesLourdesPassentParRunHeavy(t *testing.T) {
	loops := map[string]struct{ file, runner string }{
		"asset_name_sweep":  {"asset_name_sweep_cron.go", "RunOnce"},
		"catalog_refresh":   {"catalog_refresh_cron.go", "RunOnce"},
		"world_leaderboard": {"world_leaderboard_cron.go", "RunOnce"},
		"data_health":       {"data_health_check.go", "runCycle"},
	}
	runBody := regexp.MustCompile(`(?s)\) Run\(ctx context\.Context\) \{.*?\n\}\n`)
	for task, l := range loops {
		raw, err := os.ReadFile(l.file)
		if err != nil {
			t.Fatalf("%s : %v", l.file, err)
		}
		src := strings.ReplaceAll(string(raw), "\r\n", "\n")
		body := runBody.FindString(src)
		if body == "" {
			t.Fatalf("%s : corps de Run introuvable", l.file)
		}
		if !strings.Contains(body, `runHeavy(ctx, "`+task+`"`) {
			t.Errorf("%s : Run ne passe pas par runHeavy pour %q", l.file, task)
		}
		for _, bare := range []string{"c." + l.runner + "(ctx)", "s." + l.runner + "(ctx)"} {
			if strings.Contains(body, bare) {
				t.Errorf("%s : Run lance %s sans la porte du mode jeu", l.file, bare)
			}
		}
	}
}

// TestRunHeavy_AvantLePremierReleveLeCycleAttend : un audit ou un rafraîchissement dont le premier
// cycle part dès le démarrage attend le premier relevé du sondeur du jeu : si Halo tourne déjà, le
// cycle ne part jamais pendant la partie.
func TestRunHeavy_AvantLePremierReleveLeCycleAttend(t *testing.T) {
	m := pendingGame(t)
	m.ExpectFirstPoll()
	started := make(chan struct{})
	go runHeavy(context.Background(), "test", func(context.Context) { close(started) })
	select {
	case <-started:
		t.Fatal("le cycle ne doit pas démarrer avant le premier relevé")
	case <-time.After(150 * time.Millisecond):
	}
	w := &gameproc.Watcher{
		Lister: gameproc.ListerFunc(func() ([]string, error) { return []string{"explorer.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(context.Background())
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("le premier relevé (jeu absent) doit libérer le cycle")
	}
}
