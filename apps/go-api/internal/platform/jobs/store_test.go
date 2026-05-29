// Package jobs — store_test.go : tests unitaires du Store de jobs asynchrones.
package jobs_test

import (
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

func newTestStore(t *testing.T) *jobs.Store {
	t.Helper()
	dir := t.TempDir()
	return jobs.NewStore(filepath.Join(dir, "jobs.json"))
}

func TestStore_Create(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "player-1")

	if job == nil {
		t.Fatal("expected job, got nil")
	}
	if job.JobID == "" {
		t.Fatal("job ID should not be empty")
	}
	if job.Status != domain.JobStatusQueued {
		t.Errorf("expected status 'queued', got %s", job.Status)
	}
	if job.PlayerSlug != "player-1" {
		t.Errorf("expected player_slug 'player-1', got %s", job.PlayerSlug)
	}
}

// TestStore_Create_ReturnsCopy verrouille le fix data-race : Create doit
// renvoyer une COPIE, pas le pointeur vivant. Une mutation ultérieure via le
// store (SetStatus) ne doit donc PAS être visible sur le job retourné — sinon
// un caller qui lit job.Status hors lock court une data race avec la goroutine
// de fond qui le mute (cf. -race openspartan_import).
func TestStore_Create_ReturnsCopy(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "player-1")

	step := "running"
	store.SetStatus(job.JobID, domain.JobStatusRunning, &step)

	if job.Status != domain.JobStatusQueued {
		t.Errorf("le job retourné par Create doit rester une copie figée à 'queued', got %s", job.Status)
	}
	// L'état réel (vivant) est bien muté, lisible via Get.
	if updated := store.Get(job.JobID); updated == nil || updated.Status != domain.JobStatusRunning {
		t.Fatalf("l'état vivant via Get doit refléter 'running', got %+v", updated)
	}
}

func TestStore_Get_Found(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "player-1")

	found := store.Get(job.JobID)
	if found == nil {
		t.Fatal("expected job, got nil")
	}
	if found.JobID != job.JobID {
		t.Errorf("job ID mismatch: %q vs %q", found.JobID, job.JobID)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	store := newTestStore(t)
	found := store.Get("nonexistent-job-id")
	if found != nil {
		t.Fatal("expected nil for unknown job")
	}
}

func TestStore_SetStatus_Running(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "player-1")
	step := "importing matches"

	ok := store.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	if !ok {
		t.Fatal("SetStatus should return true for existing job")
	}

	updated := store.Get(job.JobID)
	if updated.Status != domain.JobStatusRunning {
		t.Errorf("expected 'running', got %s", updated.Status)
	}
}

func TestStore_SetStatus_Succeeded(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "player-1")
	summary := "inserted=42 duration=3.5s"

	store.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)

	updated := store.Get(job.JobID)
	if updated.Status != domain.JobStatusSucceeded {
		t.Errorf("expected 'succeeded', got %s", updated.Status)
	}
	if updated.FinishedAt == nil {
		t.Fatal("FinishedAt should be set on terminal status")
	}
}

func TestStore_FindActiveInitialSync_Found(t *testing.T) {
	store := newTestStore(t)
	store.Create(domain.JobTypeInitialSync, "active-player")

	active := store.FindActiveInitialSync("active-player")
	if active == nil {
		t.Fatal("expected active job for 'active-player'")
	}
}

func TestStore_FindActiveInitialSync_NotFound(t *testing.T) {
	store := newTestStore(t)
	active := store.FindActiveInitialSync("no-job-player")
	if active != nil {
		t.Fatalf("expected nil for player with no job, got %+v", active)
	}
}

func TestStore_FindActiveInitialSync_AfterSuccess(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "completed-player")
	summary := "done"
	store.SetStatus(job.JobID, domain.JobStatusSucceeded, &summary)

	// Job terminé → ne doit plus être "actif"
	active := store.FindActiveInitialSync("completed-player")
	if active != nil {
		t.Fatalf("expected nil for completed job, got %+v", active)
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jobs.json")

	// Créer un job dans le store 1
	store1 := jobs.NewStore(path)
	job := store1.Create(domain.JobTypeInitialSync, "persist-player")

	// Charger le store 2 depuis le même fichier
	store2 := jobs.NewStore(path)
	found := store2.Get(job.JobID)
	if found == nil {
		t.Fatal("job should persist across store instances")
	}
}

func TestStore_TTL_ExpiredJobReturnsNil(t *testing.T) {
	store := newTestStore(t)
	job := store.Create(domain.JobTypeInitialSync, "ttl-player")

	// Marquer comme terminé avec FinishedAt dans le passé
	store.Update(job.JobID, func(j *domain.AsyncJobStatus) {
		j.Status = domain.JobStatusSucceeded
		past := time.Now().Add(-2 * time.Hour)
		j.FinishedAt = &past
	})

	// Job expiré (> 1h après finishedAt) → nil
	found := store.Get(job.JobID)
	if found != nil {
		t.Fatal("expired job should return nil")
	}
}
