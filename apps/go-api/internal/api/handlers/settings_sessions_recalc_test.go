package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// recalcRouter monte les routes de réglages avec le faux service de recalcul des sessions.
func recalcRouter(svc port.SessionsRecalcService) *chi.Mux {
	h := handlers.NewSettingsHandler(&config.AppConfig{}, nil, nil, &mockMediaIndexer{}).WithSessionsRecalc(svc)
	r := chi.NewRouter()
	h.Mount(r)
	return r
}

// POST /settings/sessions/recalculate : 202 et le job du service, rendu tel quel.
func TestSettingsHandler_PostRecalculateSessions_Accepted(t *testing.T) {
	svc := newFakeJobService()
	svc.job = &domain.AsyncJobStatus{JobID: "job-9", JobType: string(domain.JobTypeSessionsRecalc), Status: domain.JobStatusQueued}

	w := httptest.NewRecorder()
	recalcRouter(svc).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/settings/sessions/recalculate", nil))

	if w.Code != http.StatusAccepted {
		t.Fatalf("attendu 202, obtenu %d : %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("réponse JSON invalide : %v", err)
	}
	if body["job_id"] != "job-9" || body["status"] != "queued" || body["job_type"] != "sessions_recalculate" {
		t.Errorf("corps %v : attendu le job du service", body)
	}
	if got := svc.called(); len(got) != 1 || got[0] != "recalc" {
		t.Errorf("appels %v, attendu un seul recalcul", got)
	}
}

// Des profils illisibles : 500 players_load_error, message fixe, cause jamais rendue.
func TestSettingsHandler_PostRecalculateSessions_ProfilsIllisibles(t *testing.T) {
	svc := newFakeJobService()
	svc.err = &port.SyncJobError{Kind: port.SyncErrPlayersLoad, Err: errors.New("secret interne")}

	w := httptest.NewRecorder()
	recalcRouter(svc).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/settings/sessions/recalculate", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("attendu 500, obtenu %d : %s", w.Code, w.Body.String())
	}
	if code, message := errorBody(t, w); code != "players_load_error" || message != "internal error" {
		t.Errorf("(code, message) = (%q, %q)", code, message)
	}
}
