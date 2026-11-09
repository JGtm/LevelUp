// boot_failure_test.go — un démarrage refusé reste visible : état d'échec servi en 503 non
// réessayable, journal Error, port gardé jusqu'au signal d'arrêt, code de sortie non nul.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

func TestClassifyBootFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bootFailureState
	}{
		{"schéma plus récent", fmt.Errorf("metadata: %w", migration.ErrSchemaNewerThanBinary), bootFailureSchemaNewer},
		{"sentinelle de base tenue", fmt.Errorf("open metadata rw: %w", duckdb.ErrBaseTenueEnEcriture), bootFailureLocked},
		{"message de verrou DuckDB", errors.New("IO Error: Could not set lock on file x: Conflicting lock is held"), bootFailureLocked},
		{"étape en échec", errors.New("migration add_x schema: boom"), bootFailureMigrationFailed},
		{"espace insuffisant", fmt.Errorf("copie: %w", migration.ErrInsufficientBackupSpace), bootFailureMigrationFailed},
		{"échec nu", errors.New("boot: classifier non posé"), bootFailureMigrationFailed},
	}
	for _, c := range cases {
		if got := classifyBootFailure(c.err); got != c.want {
			t.Errorf("%s : %q, attendu %q", c.name, got, c.want)
		}
	}
}

// assertFailed vérifie la réponse 503 server_start_failed non réessayable portant l'état want.
func assertFailed(t *testing.T, g *bootGate, want bootFailureState) {
	t.Helper()
	for _, path := range []string{"/health", "/api/v1/bootstrap", "/"} {
		w := gateGet(g, path)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET %s : statut %d, attendu 503", path, w.Code)
		}
		if ra := w.Header().Get("Retry-After"); ra != "" {
			t.Errorf("GET %s : Retry-After = %q, une réponse non réessayable n'en porte pas", path, ra)
		}
		var body struct {
			Code      string `json:"code"`
			Retryable bool   `json:"retryable"`
			Details   struct {
				Step  string `json:"step"`
				State string `json:"state"`
			} `json:"details"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("corps illisible %q : %v", w.Body.String(), err)
		}
		if body.Code != serverStartFailedCode || body.Retryable ||
			body.Details.State != string(want) || body.Details.Step != string(bootStepMigrations) {
			t.Errorf("GET %s : %+v, attendu code %q, non réessayable, étape migrations, état %q",
				path, body, serverStartFailedCode, want)
		}
	}
}

// waitFailed attend que refuseStart ait posé l'état d'échec sur la porte (donc initDone).
func waitFailed(t *testing.T, g *bootGate) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, failed := g.failureState(); failed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("état d'échec jamais posé sur la porte")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestBootGate_SansEchec_PasDEtatEtReessayable(t *testing.T) {
	g := newBootGate(&config.AppConfig{})
	if _, failed := g.failureState(); failed {
		t.Error("échec posé sans appel à fail")
	}
	w := gateGet(g, "/api/v1/bootstrap")
	if strings.Contains(w.Body.String(), `"state"`) || !strings.Contains(w.Body.String(), `"retryable":true`) {
		t.Errorf("réponse de démarrage normale = %s, sans état et réessayable attendu", w.Body.String())
	}
}

func TestRefuseStart_JournalErrorPorteGardeeCodeNonNul(t *testing.T) {
	logs := captureSlog(t)
	b, sigCh, _ := startSignalWatch(t)

	cause := errors.New("migration add_x schema: boom")
	done := make(chan int, 1)
	go func() { done <- b.refuseStart(t.Context(), bootFailureMigrationFailed, cause) }()

	// Tant qu'aucun signal n'est reçu, le processus reste debout et sert l'état d'échec.
	waitFailed(t, b.gate)
	assertFailed(t, b.gate, bootFailureMigrationFailed)
	select {
	case code := <-done:
		t.Fatalf("refuseStart rendu (%d) avant le signal d'arrêt", code)
	case <-time.After(100 * time.Millisecond):
	}

	sigCh <- os.Interrupt
	select {
	case code := <-done:
		if code == 0 {
			t.Error("code de sortie nul pour un démarrage refusé")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("refuseStart ne rend pas la main au signal d'arrêt")
	}

	out := logs.String()
	for _, want := range []string{"level=ERROR", "démarrage refusé", "state=migration_failed", "boom"} {
		if !strings.Contains(out, want) {
			t.Errorf("journal sans %q :\n%s", want, out)
		}
	}
}
