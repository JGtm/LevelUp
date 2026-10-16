package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// captureSlog redirige le logger par défaut vers un tampon le temps du test.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// stuck rend une phase qui ne se termine jamais d'elle-même (jusqu'à la fin du test).
func stuck(t *testing.T, name string, limit time.Duration) shutdownPhase {
	t.Helper()
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	return shutdownPhase{name, limit, blocking(func() { <-release })}
}

// fakeClock rend une horloge d'arrêt dont l'instant se pilote à la main.
func fakeClock(total, reserve time.Duration) (*shutdownClock, func(time.Duration)) {
	var mu sync.Mutex
	base := time.Unix(1_000_000, 0)
	offset := time.Duration(0)
	c := &shutdownClock{start: base, total: total, reserve: reserve, now: func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return base.Add(offset)
	}}
	return c, func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		offset += d
	}
}

// Une phase reçoit la plus petite de sa borne et du budget restant ; la réserve de
// fermeture n'est jamais entamée ; la durée accordée ne devient jamais négative.
func TestShutdownClock_AllotNeverExceedsRemainingNorTouchesReserve(t *testing.T) {
	c, advance := fakeClock(40*time.Second, 5*time.Second)
	if got := c.allot(15 * time.Second); got != 15*time.Second {
		t.Fatalf("départ : allot(15 s) = %s, attendu la borne de la phase", got)
	}
	advance(30 * time.Second)
	if got := c.allot(15 * time.Second); got != 5*time.Second {
		t.Fatalf("à 30 s : allot(15 s) = %s, attendu 5 s (35 s de budget d'attente - 30 s écoulées)", got)
	}
	advance(10 * time.Second) // 40 s écoulées : au-delà du budget d'attente
	if got := c.allot(15 * time.Second); got != 0 {
		t.Fatalf("budget épuisé : allot = %s, attendu 0", got)
	}
}

// Une phase qui dépasse est nommée dans un Warn (phase, durée accordée) et la phase
// suivante s'exécute quand même.
func TestRunPhase_OverrunIsNamedAndNextPhaseRuns(t *testing.T) {
	logs := captureSlog(t)
	c := newShutdownClock(2*time.Second, 500*time.Millisecond)
	ran := false
	next := shutdownPhase{"suivante", time.Second, func(context.Context) bool { ran = true; return true }}

	begin := time.Now()
	c.runAll(context.Background(), []shutdownPhase{stuck(t, "bloquee", 150*time.Millisecond), next})

	if elapsed := time.Since(begin); elapsed > time.Second {
		t.Errorf("la phase bloquée a retenu la séquence %s, sa borne est 150 ms", elapsed)
	}
	if !ran {
		t.Error("la phase suivante ne s'est pas exécutée")
	}
	out := logs.String()
	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, "phase=bloquee") || !strings.Contains(out, "allotted=150ms") {
		t.Errorf("Warn nommant la phase en dépassement absent :\n%s", out)
	}
	if strings.Contains(out, "phase=suivante") {
		t.Errorf("la phase terminée dans son délai ne doit pas être signalée :\n%s", out)
	}
}

// Toutes les phases bloquées : la séquence rend la main dans le budget d'attente (total
// moins réserve), avec la réserve intacte pour la fermeture des bases.
func TestRunAll_AllPhasesStuckStaysWithinBudgetAndLeavesReserve(t *testing.T) {
	captureSlog(t)
	const total, reserve = 1200 * time.Millisecond, 400 * time.Millisecond
	c := newShutdownClock(total, reserve)
	phases := []shutdownPhase{
		stuck(t, "a", time.Hour), stuck(t, "b", time.Hour), stuck(t, "c", time.Hour),
	}

	begin := time.Now()
	c.runAll(context.Background(), phases)
	elapsed := time.Since(begin)

	if elapsed > total-reserve+150*time.Millisecond {
		t.Errorf("séquence d'attente : %s, attendu au plus %s (total - réserve)", elapsed, total-reserve)
	}
	if left := total - elapsed; left < reserve-150*time.Millisecond {
		t.Errorf("il reste %s avant la fin du budget, la réserve de fermeture est de %s", left, reserve)
	}
}

// Phase qui se termine dans son délai : rien n'est signalé.
func TestRunPhase_FinishedInTimeIsSilent(t *testing.T) {
	logs := captureSlog(t)
	c := newShutdownClock(time.Second, 0)
	ok := c.runPhase(context.Background(), shutdownPhase{"rapide", time.Second, blocking(func() {})})
	if !ok {
		t.Fatal("phase terminée dans son délai rendue en dépassement")
	}
	if logs.Len() != 0 {
		t.Errorf("sortie inattendue :\n%s", logs.String())
	}
}

// composeFile est la part de docker-compose.yml que le garde-rail lit.
type composeFile struct {
	Services map[string]struct {
		StopGracePeriod string `yaml:"stop_grace_period"`
	} `yaml:"services"`
}

// shutdownGraceMargin est la marge minimale entre le budget d'arrêt et le délai que Docker
// accorde avant SIGKILL : l'écart couvre le démarrage de l'arrêt et la sortie du process.
const shutdownGraceMargin = 5 * time.Second

// Le budget d'arrêt tient sous le délai d'arrêt de Docker pour chaque service qui lance le
// serveur. La clé qui disparaît (retour au défaut de 10 s) fait échouer le test.
func TestShutdownBudget_FitsUnderComposeStopGracePeriod(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("lecture de docker-compose.yml : %v — le garde-rail ne se tait pas quand il ne sait pas", err)
	}
	var compose composeFile
	if err := yaml.Unmarshal(raw, &compose); err != nil {
		t.Fatalf("docker-compose.yml illisible : %v", err)
	}
	for _, name := range []string{"levelup", "levelup-demo"} {
		svc, ok := compose.Services[name]
		if !ok {
			t.Errorf("service %s absent de docker-compose.yml", name)
			continue
		}
		if svc.StopGracePeriod == "" {
			t.Errorf("service %s sans stop_grace_period : Docker tue le process à 10 s, avant la fin de la séquence d'arrêt (budget %s)", name, shutdownBudget)
			continue
		}
		grace, err := time.ParseDuration(svc.StopGracePeriod)
		if err != nil {
			t.Errorf("service %s : stop_grace_period %q illisible : %v", name, svc.StopGracePeriod, err)
			continue
		}
		if grace < shutdownBudget+shutdownGraceMargin {
			t.Errorf("service %s : stop_grace_period %s < budget d'arrêt %s + marge %s", name, grace, shutdownBudget, shutdownGraceMargin)
		}
	}
}

// Les bornes de phase sont cohérentes avec le budget : la réserve de fermeture est une
// part du budget, et aucune borne de phase ne dépasse le budget d'attente.
func TestShutdownBudget_PhaseCapsFitInWaitBudget(t *testing.T) {
	if shutdownCloseReserve <= 0 || shutdownCloseReserve >= shutdownBudget {
		t.Fatalf("réserve de fermeture %s hors de ]0, %s[", shutdownCloseReserve, shutdownBudget)
	}
	waitBudget := shutdownBudget - shutdownCloseReserve
	for name, limit := range map[string]time.Duration{
		"watcher": phaseWatcherCap, "http": phaseHTTPCap, "scheduler": phaseSchedulerCap, "gate": phaseGateCap,
		"persist.drain": phaseBatchDrainCap, "persist.close": phaseBatchCloseCap, "persist.worker": phaseWorkerCap,
		"checkpoint": phaseCheckpointCap,
	} {
		if limit <= 0 || limit > waitBudget {
			t.Errorf("borne de la phase %s = %s hors de ]0, %s]", name, limit, waitBudget)
		}
	}
}
