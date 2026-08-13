// Package handlers — admin_actions_replay_build_test.go : contrats HTTP de l'action
// « construire le rejeu 2D d'un match » (503 si dépendance absente, 400 si match_id
// absent, 409 si job déjà en vol sur ce match, 202 sinon).
//
// Miroir de admin_actions_initial_sync_test.go : mount sous /admin (point de montage de
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

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// serveAdminReplayBuild monte h sous /admin et sert la requête.
func serveAdminReplayBuild(h *AdminReplayBuildActionHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func replayBuildPostReq(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/admin/actions/replay-build/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// okReplayBuildRunner : runner stub renvoyant un résultat trivial.
func okReplayBuildRunner(_ context.Context, _, matchID string) (map[string]any, error) {
	return map[string]any{"match_id": matchID, "tracks": 8}, nil
}

// TestAdminReplayBuild_Unavailable : runner ou store nil → 503.
func TestAdminReplayBuild_Unavailable(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	for name, h := range map[string]*AdminReplayBuildActionHandler{
		"runner nil": NewAdminReplayBuildActionHandler(nil, store, context.Background()),
		"store nil":  NewAdminReplayBuildActionHandler(okReplayBuildRunner, nil, context.Background()),
	} {
		rec := serveAdminReplayBuild(h, replayBuildPostReq(`{"match_id":"000d5950"}`))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s : status = %d (attendu 503) body=%s", name, rec.Code, rec.Body.String())
		}
	}
}

// TestAdminReplayBuild_InvalidInput : match_id absent ou JSON malformé → 400.
func TestAdminReplayBuild_InvalidInput(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewAdminReplayBuildActionHandler(okReplayBuildRunner, store, context.Background())
	for name, body := range map[string]string{"sans match_id": `{}`, "json malforme": `{pas du json`} {
		rec := serveAdminReplayBuild(h, replayBuildPostReq(body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s : status = %d (attendu 400) body=%s", name, rec.Code, rec.Body.String())
		}
	}
}

// TestAdminReplayBuild_Accepted : requête valide → 202 + job replay_build créé et suivi
// par le match_id.
func TestAdminReplayBuild_Accepted(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	h := NewAdminReplayBuildActionHandler(okReplayBuildRunner, store, context.Background())
	rec := serveAdminReplayBuild(h, replayBuildPostReq(`{"match_id":"000d5950"}`))
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
	if got.JobType != string(domain.JobTypeReplayBuild) {
		t.Fatalf("job_type = %q (attendu %s)", got.JobType, domain.JobTypeReplayBuild)
	}
	waitInitialSyncJobDone(t, store, got.JobID)
}

// TestAdminReplayBuild_ConflictWhenAlreadyRunning : un job replay_build déjà en vol pour
// CE match → 409 avec le job_id actif en details.
func TestAdminReplayBuild_ConflictWhenAlreadyRunning(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	existing := store.Create(domain.JobTypeReplayBuild, "000d5950")
	store.SetStatus(existing.JobID, domain.JobStatusRunning, nil)

	h := NewAdminReplayBuildActionHandler(okReplayBuildRunner, store, context.Background())
	rec := serveAdminReplayBuild(h, replayBuildPostReq(`{"match_id":"000d5950"}`))
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
