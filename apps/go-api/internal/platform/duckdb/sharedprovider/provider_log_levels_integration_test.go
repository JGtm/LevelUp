//go:build integration

package sharedprovider_test

// Tests d'intégration du NIVEAU de journalisation du cycle B-swap (lot C
// dé-bruitage de provider.log, 2026-08-26).
//
// Contrat vérifié : un cycle nominal ne produit AUCUN log INFO pour la base
// concernée (il reste intégralement en DEBUG), tandis qu'un cycle qui dépasse
// le seuil de lenteur remonte en INFO avec sa durée. Les WARN/ERROR ne passent
// pas par ce chemin et restent inconditionnels.

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// syncBuffer sérialise les écritures du handler slog : les logs du cycle
// peuvent partir d'autres goroutines que celle du test (timers, notify).
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// logRecord est la projection minimale d'une ligne de log JSON.
type logRecord struct {
	Level       string `json:"level"`
	Msg         string `json:"msg"`
	Path        string `json:"path"`
	DrainMs     *int64 `json:"drain_ms"`
	TotalMs     *int64 `json:"total_ms"`
	ThresholdMs *int64 `json:"threshold_ms"`
}

// captureSlogJSON redirige le logger par défaut vers un buffer JSON de niveau
// DEBUG pour la durée du test. Niveau DEBUG volontaire : le test doit pouvoir
// constater que le nominal est bien ÉMIS (en debug) et pas simplement supprimé.
func captureSlogJSON(t *testing.T) *syncBuffer {
	t.Helper()
	buf := &syncBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// providerRecordsFor ne garde que les logs émis pour la base `path` : les autres
// tests du package et leurs goroutines résiduelles partagent le logger par défaut.
func providerRecordsFor(t *testing.T, buf *syncBuffer, path string) []logRecord {
	t.Helper()
	var out []logRecord
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var rec logRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log non-JSON: %q (%v)", line, err)
		}
		if rec.Path != path {
			continue
		}
		out = append(out, rec)
	}
	return out
}

// levelOf retourne le niveau du premier record portant ce message ("" si absent).
func levelOf(recs []logRecord, msg string) string {
	for _, r := range recs {
		if r.Msg == msg {
			return r.Level
		}
	}
	return ""
}

// phaseMessages : les 3 phases chronométrées du cycle, celles qui basculent
// DEBUG↔INFO selon la durée.
var phaseMessages = []string{
	"provider: readers drainés",
	"provider: swap RO→RW terminé",
	"provider: swap RW→RO terminé",
}

// acquireMessage : le début d'acquisition, sans durée à qualifier — toujours DEBUG.
const acquireMessage = "provider: AcquireWriter démarré"

// TestProvider_NominalCycleLogsNoInfo_integration : un cycle acquire/release
// nominal (rapide, seuil par défaut à 2s) ne laisse AUCUN INFO dans provider.log.
// C'est le cas qui représentait 99,8 % du volume mesuré en prod le 2026-08-25.
func TestProvider_NominalCycleLogsNoInfo_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	buf := captureSlogJSON(t)

	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_log_nominal")
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	recs := providerRecordsFor(t, buf, path)
	if len(recs) == 0 {
		t.Fatal("aucun log capturé pour cette base : le test n'observe pas le cycle")
	}
	for _, r := range recs {
		if r.Level == slog.LevelInfo.String() {
			t.Errorf("cycle nominal : log INFO inattendu %q (le nominal doit rester en DEBUG)", r.Msg)
		}
	}

	// Contre-épreuve : le cycle a bien été journalisé, en DEBUG. Sans ça, une
	// suppression pure et simple des 4 logs passerait aussi l'assertion ci-dessus.
	for _, msg := range append([]string{acquireMessage}, phaseMessages...) {
		if got := levelOf(recs, msg); got != slog.LevelDebug.String() {
			t.Errorf("%q : niveau %q, attendu %q", msg, got, slog.LevelDebug.String())
		}
	}
}

// TestProvider_SlowCycleLogsInfoWithDuration_integration : au-delà du seuil, les
// 3 phases chronométrées repassent en INFO avec leur durée et threshold_ms —
// l'anomalie reste donc visible sans réactiver le niveau debug.
func TestProvider_SlowCycleLogsInfoWithDuration_integration(t *testing.T) {
	path := setupSharedDB(t)
	p, err := sharedprovider.New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = p.Close() }()

	// Seuil ras-du-sol AVANT l'acquisition : tout cycle réel est « lent »,
	// sans avoir à en fabriquer un de plusieurs secondes.
	sharedprovider.SetSlowSwapThresholdForTest(p, time.Nanosecond)

	buf := captureSlogJSON(t)

	// Un reader en vol relâché après 20ms : sans lui, le drain d'un provider au
	// repos est un no-op dont la durée mesurée est 0 (résolution du timer
	// Windows) — il ne franchirait aucun seuil, même à 1 ns. Les deux autres
	// phases contiennent des I/O fichier (open RW, close RW + reopen RO) et sont
	// mesurables sans artifice.
	_, release, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		release()
	}()

	ctx := ctxkeys.WithDBWriterLabel(context.Background(), "test_log_lent")
	w, err := p.AcquireWriter(ctx)
	if err != nil {
		t.Fatalf("AcquireWriter: %v", err)
	}
	w.Release()

	recs := providerRecordsFor(t, buf, path)
	for _, msg := range phaseMessages {
		if got := levelOf(recs, msg); got != slog.LevelInfo.String() {
			t.Errorf("cycle lent : %q en %q, attendu %q", msg, got, slog.LevelInfo.String())
		}
	}
	// Le début d'acquisition n'a pas de durée : il reste DEBUG même sur un cycle lent.
	if got := levelOf(recs, acquireMessage); got != slog.LevelDebug.String() {
		t.Errorf("%q : niveau %q, attendu %q (pas de durée à qualifier)",
			acquireMessage, got, slog.LevelDebug.String())
	}

	// La durée et le seuil doivent accompagner chaque INFO : un INFO sans
	// chiffre ne dit pas au lecteur de provider.log à quel point c'est lent.
	for _, r := range recs {
		if r.Level != slog.LevelInfo.String() {
			continue
		}
		if r.ThresholdMs == nil {
			t.Errorf("%q : INFO sans threshold_ms", r.Msg)
		}
		if r.DrainMs == nil && r.TotalMs == nil {
			t.Errorf("%q : INFO sans durée (ni drain_ms ni total_ms)", r.Msg)
		}
	}
}
