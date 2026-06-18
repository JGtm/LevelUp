// Package handlers_test — settings_media_test.go : test du remplacement du stub media reset.
//
// Sprint 53 A7 : vérifie que PostMediaResetIndex passe le job en JobStatusSucceeded
// avec ProgressPct=100 et que "Terminé (stub)" n'apparaît plus dans le step.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/service"
)

// mockMediaIndexer est un MediaIndexer instantané qui marque le job 100% succeed.
type mockMediaIndexer struct {
	// simulateError fait échouer le ResetAndReindex si true.
	simulateError bool
}

func (m *mockMediaIndexer) ResetAndReindex(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ bool,
	jobStore *jobs.Store,
	jobID string,
) error {
	if m.simulateError {
		return &mediaIndexError{msg: "erreur simulée"}
	}
	pct := 100
	step := "Index médias réinitialisé (mock)"
	jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.CurrentStep = &step
	})
	return nil
}

func (m *mockMediaIndexer) ScanAllMedia(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	jobStore *jobs.Store,
	jobID string,
) error {
	if m.simulateError {
		return &mediaIndexError{msg: "erreur simulée"}
	}
	pct := 100
	step := "Scan médias terminé (mock)"
	jobStore.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.ProgressPct = &pct
		j.CurrentStep = &step
	})
	return nil
}

// Vérification compile-time que mockMediaIndexer implémente service.MediaIndexer.
var _ service.MediaIndexer = (*mockMediaIndexer)(nil)

// mediaIndexError est une erreur de test.
type mediaIndexError struct{ msg string }

func (e *mediaIndexError) Error() string { return e.msg }

// ── Helpers ──────────────────────────────────────────────────────────────────

func newSettingsRouterWithIndexer(t *testing.T, indexer service.MediaIndexer) (*chi.Mux, *jobs.Store) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.AppConfig{DemoMode: false, RepoRoot: dir}
	settingsStore := settings_platform.NewStore(filepath.Join(dir, "app_settings.json"))
	jobStore := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	h := handlers.NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, indexer)

	r := chi.NewRouter()
	h.Mount(r)
	return r, jobStore
}

// pollJobSucceeded attend que le job atteigne un état terminal.
// Timeout rapide pour les tests (200ms).
func pollJobSucceeded(t *testing.T, store *jobs.Store, jobID string) *domain.AsyncJobStatus {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		j := store.Get(jobID)
		if j == nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		if j.Status == domain.JobStatusSucceeded || j.Status == domain.JobStatusFailed {
			return j
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout : le job n'a pas atteint un état terminal en 200ms")
	return nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestPostMediaResetIndex_Stub_Replaced vérifie que le stub "Terminé (stub)" est remplacé
// par la vraie implémentation : le job doit atteindre JobStatusSucceeded avec ProgressPct=100.
func TestPostMediaResetIndex_Stub_Replaced(t *testing.T) {
	r, jobStore := newSettingsRouterWithIndexer(t, &mockMediaIndexer{})

	body := `{"confirm_destructive": true, "reindex_after_reset": false}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d: %s", w.Code, w.Body.String())
	}

	// Extraire le jobID depuis la réponse.
	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}
	if jobResp.JobID == "" {
		t.Fatal("jobID absent dans la réponse")
	}

	// Attendre que le job se termine.
	finalJob := pollJobSucceeded(t, jobStore, jobResp.JobID)

	if finalJob.Status != domain.JobStatusSucceeded {
		t.Errorf("Status = %q, attendu %q", finalJob.Status, domain.JobStatusSucceeded)
	}
	if finalJob.ProgressPct == nil || *finalJob.ProgressPct != 100 {
		pct := 0
		if finalJob.ProgressPct != nil {
			pct = *finalJob.ProgressPct
		}
		t.Errorf("ProgressPct = %d, attendu 100", pct)
	}
	// Vérifier que le step ne contient plus "stub"
	if finalJob.CurrentStep != nil && containsCI(*finalJob.CurrentStep, "stub") {
		t.Errorf("CurrentStep contient encore 'stub' : %q", *finalJob.CurrentStep)
	}
}

// TestPostMediaResetIndex_IndexerError vérifie qu'une erreur du MediaIndexer marque le job Failed.
func TestPostMediaResetIndex_IndexerError(t *testing.T) {
	r, jobStore := newSettingsRouterWithIndexer(t, &mockMediaIndexer{simulateError: true})

	body := `{"confirm_destructive": true}`
	req := httptest.NewRequest(http.MethodPost, "/settings/media/reset-index", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d", w.Code)
	}

	var jobResp domain.AsyncJobStatus
	if err := json.NewDecoder(w.Body).Decode(&jobResp); err != nil {
		t.Fatalf("décodage: %v", err)
	}

	finalJob := pollJobSucceeded(t, jobStore, jobResp.JobID)
	if finalJob.Status != domain.JobStatusFailed {
		t.Errorf("Status = %q, attendu %q", finalJob.Status, domain.JobStatusFailed)
	}
	if finalJob.Error == nil {
		t.Error("Error attendu non nil pour un job échoué")
	}
}

// containsCI est un contains insensible à la casse (simple).
func containsCI(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sl, subl := []rune(s), []rune(sub)
	for i := 0; i <= len(sl)-len(subl); i++ {
		match := true
		for j, r := range subl {
			c := sl[i+j]
			if c != r && toLower(c) != toLower(r) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
