// Package handlers — admin_actions_catalog_drain_test.go : contrat HTTP de
// l'action drain DiscoveryUGC (job async : 503 si deps absentes, 409 si déjà
// en vol, 202 + job au lancement).
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

func okDrainRunner() CatalogDrainRunner {
	return func(context.Context, string) (domain.CatalogUGCDrainResult, error) {
		return domain.CatalogUGCDrainResult{}, nil
	}
}

// TestAdminCatalogDrain_Unavailable : run ou store nil → 503.
func TestAdminCatalogDrain_Unavailable(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	for name, h := range map[string]*AdminCatalogDrainHandler{
		"run nil":   NewAdminCatalogDrainHandler(nil, store, nil),
		"store nil": NewAdminCatalogDrainHandler(okDrainRunner(), nil, nil),
	} {
		rec := httptest.NewRecorder()
		h.Run(rec, httptest.NewRequest(http.MethodPost, "/admin/actions/catalog/ugc-drain", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503)", name, rec.Code)
		}
	}
}

// TestAdminCatalogDrain_Conflict : un drain déjà en vol pour ce titre → 409 avec
// le job_id existant en details (vérifié AVANT tout travail).
func TestAdminCatalogDrain_Conflict(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	existing := store.Create(domain.JobTypeCatalogUGCDrain, "halo_infinite")
	store.SetStatus(existing.JobID, domain.JobStatusRunning, nil)

	h := NewAdminCatalogDrainHandler(okDrainRunner(), store, context.Background())
	rec := httptest.NewRecorder()
	h.Run(rec, httptest.NewRequest(http.MethodPost, "/admin/actions/catalog/ugc-drain?title=halo_infinite", nil))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d (attendu 409)", rec.Code)
	}
	var got struct {
		Code    string `json:"code"`
		Details struct {
			JobID string `json:"job_id"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.Code != "already_running" || got.Details.JobID != existing.JobID {
		t.Fatalf("409 inattendu : %+v", got)
	}
}

// TestAdminCatalogDrain_Accepted : lancement → 202 + job ; le résultat est posé
// par la goroutine (attente du statut terminal pour éviter une race au cleanup).
func TestAdminCatalogDrain_Accepted(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewAdminCatalogDrainHandler(func(context.Context, string) (domain.CatalogUGCDrainResult, error) {
		return domain.CatalogUGCDrainResult{Seeded: 3, Playlists: 1}, nil
	}, store, context.Background())

	rec := httptest.NewRecorder()
	h.Run(rec, httptest.NewRequest(http.MethodPost, "/admin/actions/catalog/ugc-drain?title=halo_infinite", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d (attendu 202) body=%s", rec.Code, rec.Body.String())
	}
	var job domain.AsyncJobStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil || job.JobID == "" {
		t.Fatalf("payload job inattendu : %+v err=%v", job, err)
	}

	// Attendre que la goroutine du job atteigne un statut terminal.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if j := store.Get(job.JobID); j != nil && j.Status == domain.JobStatusSucceeded {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("le job n'a pas atteint le statut succeeded dans le délai")
}
