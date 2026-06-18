// Package handlers — admin_actions_test.go : contrats HTTP des actions du
// dashboard monitoring (503 si dépendance absente, 409 si job déjà en vol).
//
// MIGRÉ vers Huma : les requêtes passent par un routeur chi montant h.Mount sous
// /admin (même point de montage que server_admin_monitoring.go). Requêtes et
// assertions inchangées.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// serveAdminActions monte h sous /admin (point de montage de
// server_admin_monitoring.go) et sert la requête via le routeur chi.
func serveAdminActions(h *AdminActionsHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// TestAdminActions_DataHealth_OK : 200 + compteurs du runner.
func TestAdminActions_DataHealth_OK(t *testing.T) {
	h := NewAdminActionsHandler(func(context.Context) (*domain.MonitoringDataHealth, error) {
		return &domain.MonitoringDataHealth{WarningsTotal: 7, OrphanXUIDs: 2}, nil
	}, nil, nil, context.Background())
	rec := serveAdminActions(h, httptest.NewRequest(http.MethodPost, "/admin/actions/data-health/run", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var got domain.MonitoringDataHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.WarningsTotal != 7 || got.OrphanXUIDs != 2 {
		t.Fatalf("payload inattendu : %+v", got)
	}
}

// TestAdminActions_DataHealth_Unavailable : runner nil ou en erreur → 503.
func TestAdminActions_DataHealth_Unavailable(t *testing.T) {
	for name, h := range map[string]*AdminActionsHandler{
		"runner nil": NewAdminActionsHandler(nil, nil, nil, nil),
		"runner err": NewAdminActionsHandler(func(context.Context) (*domain.MonitoringDataHealth, error) {
			return nil, errors.New("scheduler non câblé")
		}, nil, nil, nil),
	} {
		rec := serveAdminActions(h, httptest.NewRequest(http.MethodPost, "/admin/actions/data-health/run", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503)", name, rec.Code)
		}
	}
}

// TestAdminActions_SyncCycle_UnavailableWhenNoScheduler : sched ou store nil → 503.
func TestAdminActions_SyncCycle_UnavailableWhenNoScheduler(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	for name, h := range map[string]*AdminActionsHandler{
		"sched nil": NewAdminActionsHandler(nil, nil, store, nil),
		"store nil": NewAdminActionsHandler(nil, &scheduler.AutoSyncScheduler{}, nil, nil),
	} {
		rec := serveAdminActions(h, httptest.NewRequest(http.MethodPost, "/admin/actions/auto-sync/run", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503)", name, rec.Code)
		}
	}
}

// TestAdminActions_SyncCycle_ConflictWhenAlreadyRunning : un cycle forcé déjà
// en vol → 409 en enveloppe d'erreur standard avec le job_id en details (le
// front suit celui-ci au lieu d'en créer un doublon). Le 409 est vérifié
// AVANT tout appel au scheduler.
func TestAdminActions_SyncCycle_ConflictWhenAlreadyRunning(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	existing := store.Create(domain.JobTypeForcedSyncCycle, "_all")
	store.SetStatus(existing.JobID, domain.JobStatusRunning, nil)

	h := NewAdminActionsHandler(nil, &scheduler.AutoSyncScheduler{}, store, context.Background())
	rec := serveAdminActions(h, httptest.NewRequest(http.MethodPost, "/admin/actions/auto-sync/run", nil))

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
	if got.Code != "already_running" {
		t.Fatalf("code = %q (attendu already_running)", got.Code)
	}
	if got.Details.JobID != existing.JobID {
		t.Fatalf("details.job_id = %q (attendu %s)", got.Details.JobID, existing.JobID)
	}
}
