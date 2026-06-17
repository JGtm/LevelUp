// Package handlers_test — jobs_test.go : tests unitaires JobsHandler.Lookup.
// Le routage HTTP (200/404, format d'erreur) est testé côté api via le golden
// Huma TestRegisterJobsHuma_ContractPreserved (Phase 3b).
package handlers_test

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

func TestJobsHandler_Lookup_OK(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	job := store.Create(domain.JobTypeInitialSync, "test-player")

	got := handlers.NewJobsHandler(store).Lookup(job.JobID)
	if got == nil {
		t.Fatalf("Lookup(%q) = nil, want job", job.JobID)
	}
	if got.JobID != job.JobID {
		t.Errorf("JobID = %q, want %q", got.JobID, job.JobID)
	}
}

func TestJobsHandler_Lookup_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))

	if got := handlers.NewJobsHandler(store).Lookup("nonexistent-job-id"); got != nil {
		t.Fatalf("Lookup(unknown) = %+v, want nil", got)
	}
}

func TestJobsHandler_Lookup_Expired(t *testing.T) {
	dir := t.TempDir()
	// Store vide (jobs inexistants / expirés).
	store := jobs.NewStore(filepath.Join(dir, "jobs.json"))
	_ = os.WriteFile(filepath.Join(dir, "jobs.json"), []byte(`{}`), 0o600)

	if got := handlers.NewJobsHandler(store).Lookup("fake-expired-id"); got != nil {
		t.Fatalf("Lookup(expired) = %+v, want nil", got)
	}
}
