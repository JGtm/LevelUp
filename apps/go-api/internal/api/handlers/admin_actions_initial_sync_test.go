// Package handlers — admin_actions_initial_sync_test.go : contrats HTTP de
// l'action « synchronisation initiale d'un joueur » (503 si dépendance absente,
// 400 si player_slug absent, 409 si job déjà en vol, 202 sinon).
//
// Miroir de admin_actions_test.go : mount sous /admin (point de montage de
// server_admin_monitoring.go), requêtes servies via un routeur chi.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// serveAdminInitialSync monte h sous /admin et sert la requête.
func serveAdminInitialSync(h *AdminInitialSyncActionHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func initialSyncPostReq(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/admin/actions/initial-sync/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// okInitialSyncRunner : runner stub renvoyant un résultat trivial.
func okInitialSyncRunner(_ context.Context, _, _ string, _ int) (map[string]any, error) {
	return map[string]any{"gamertag": "Tester", "matches_inserted": 0}, nil
}

// TestAdminInitialSync_Unavailable : runner ou store nil → 503.
func TestAdminInitialSync_Unavailable(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	for name, h := range map[string]*AdminInitialSyncActionHandler{
		"runner nil": NewAdminInitialSyncActionHandler(nil, store, context.Background(), nil),
		"store nil":  NewAdminInitialSyncActionHandler(okInitialSyncRunner, nil, context.Background(), nil),
	} {
		rec := serveAdminInitialSync(h, initialSyncPostReq(`{"player_slug":"tester"}`))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503) body=%s", name, rec.Code, rec.Body.String())
		}
	}
}

// TestAdminInitialSync_InvalidInput : player_slug absent → 400.
func TestAdminInitialSync_InvalidInput(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewAdminInitialSyncActionHandler(okInitialSyncRunner, store, context.Background(), nil)
	rec := serveAdminInitialSync(h, initialSyncPostReq(`{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d (attendu 400) body=%s", rec.Code, rec.Body.String())
	}
}

// TestAdminInitialSync_Accepted : requête valide → 202 + job admin_initial_sync
// créé et suivi par le PlayerSlug.
func TestAdminInitialSync_Accepted(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewAdminInitialSyncActionHandler(okInitialSyncRunner, store, context.Background(), nil)
	rec := serveAdminInitialSync(h, initialSyncPostReq(`{"player_slug":"tester","max_matches":50}`))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d (attendu 202) body=%s", rec.Code, rec.Body.String())
	}
	var got domain.AsyncJobStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.JobID == "" {
		t.Fatalf("job_id vide dans la réponse 202")
	}
	if got.JobType != string(domain.JobTypeAdminInitialSync) {
		t.Fatalf("job_type = %q (attendu %s)", got.JobType, domain.JobTypeAdminInitialSync)
	}
	waitInitialSyncJobDone(t, store, got.JobID)
}

// waitInitialSyncJobDone attend l'état terminal du job lancé en goroutine par le
// handler : sans cette attente, l'écriture asynchrone de jobs.json court-circuite
// le nettoyage du TempDir (« RemoveAll cleanup: directory not empty » — échec
// observé 2×sur le job CI Coverage Linux, 2026-07-25).
func waitInitialSyncJobDone(t *testing.T, store *jobs.Store, jobID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st := store.Get(jobID)
		if st != nil && st.Status != domain.JobStatusQueued && st.Status != domain.JobStatusRunning {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s toujours non terminal après 5 s", jobID)
}

// TestAdminInitialSync_ConflictWhenAlreadyRunning : un job initial-sync déjà en
// vol pour ce joueur → 409 en enveloppe d'erreur standard (job_id en details).
func TestAdminInitialSync_ConflictWhenAlreadyRunning(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	existing := store.Create(domain.JobTypeAdminInitialSync, "tester")
	store.SetStatus(existing.JobID, domain.JobStatusRunning, nil)

	h := NewAdminInitialSyncActionHandler(okInitialSyncRunner, store, context.Background(), nil)
	rec := serveAdminInitialSync(h, initialSyncPostReq(`{"player_slug":"tester"}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d (attendu 409) body=%s", rec.Code, rec.Body.String())
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
