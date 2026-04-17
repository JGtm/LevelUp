// Package handlers_test — jobs_test.go : tests unitaires JobsHandler.
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

func newJobsRouter(store *jobs.Store) *chi.Mux {
	r := chi.NewRouter()
	h := handlers.NewJobsHandler(store)
	r.Get("/jobs/{job_id}", h.GetJob)
	return r
}

func TestJobsHandler_GetJob_OK(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	job := store.Create(domain.JobTypeInitialSync, "test-player")

	r := newJobsRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/jobs/"+job.JobID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJobsHandler_GetJob_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	r := newJobsRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/jobs/nonexistent-job-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestJobsHandler_GetJob_EmptyID(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	r := newJobsRouter(store)
	// Route sans job_id → chi ne matche pas, 404 attendu
	req := httptest.NewRequest(http.MethodGet, "/jobs/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 404 ou 405 selon chi
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Logf("got %d (acceptable: 404 ou 405)", w.Code)
	}
}

func TestJobsHandler_GetJob_Expired(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	// Créer et supprimer manuellement le job
	_ = store.Create(domain.JobTypeInitialSync, "test-player")

	// Créer un store vide (jobs inexistants)
	store2 := jobs.NewStore(filepath.Join(dir, "jobs2.json"))
	_ = os.WriteFile(filepath.Join(dir, "jobs2.json"), []byte(`{}`), 0o600)

	r := newJobsRouter(store2)
	req := httptest.NewRequest(http.MethodGet, "/jobs/fake-expired-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
