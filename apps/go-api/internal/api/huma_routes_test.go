//go:build cgo

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// stubGamertagSearch : mock port.GamertagSearchService pour le golden Huma.
type stubGamertagSearch struct {
	results []domain.GamertagSearchResult
	err     error
}

func (s *stubGamertagSearch) Search(_ context.Context, _ string) ([]domain.GamertagSearchResult, error) {
	return s.results, s.err
}

func writeTmpChangelog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "CHANGELOG.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestRegisterChangelogHuma_ContractPreserved (Phase 3b, 1re route migrée) : la
// route GET /changelog migrée vers Huma préserve STRICTEMENT le contrat —
// 200 {content} SANS champ $schema, et 404 au format writeError
// {code:CHANGELOG_NOT_FOUND, message, retryable:false}. C'est le golden qui
// valide le pattern de migration (error-model + $schema off) bout-en-bout.
func TestRegisterChangelogHuma_ContractPreserved(t *testing.T) {
	// 200 — contenu présent.
	r := chi.NewRouter()
	registerChangelogHuma(newHumaAPI(r), handlers.NewChangelogHandler(writeTmpChangelog(t, "# CL\n- x")))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/changelog", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("200 attendu, got %d: %s", rec.Code, rec.Body.String())
	}
	var ok map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if ok["content"] != "# CL\n- x" {
		t.Errorf("content = %v", ok["content"])
	}
	if _, has := ok["$schema"]; has {
		t.Error("$schema ne doit PAS être injecté (contrat front identique à writeJSON)")
	}

	// 404 — CHANGELOG.md absent → format d'erreur writeError.
	r2 := chi.NewRouter()
	registerChangelogHuma(newHumaAPI(r2), handlers.NewChangelogHandler(t.TempDir()))
	rec2 := httptest.NewRecorder()
	r2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/changelog", nil))
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("404 attendu, got %d", rec2.Code)
	}
	var er map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &er); err != nil {
		t.Fatalf("JSON erreur invalide: %v", err)
	}
	if er["code"] != "CHANGELOG_NOT_FOUND" {
		t.Errorf("code = %v, want CHANGELOG_NOT_FOUND", er["code"])
	}
	if er["retryable"] != false {
		t.Errorf("retryable = %v, want false", er["retryable"])
	}
}

// TestRegisterJobsHuma_ContractPreserved (Phase 3b, shape path-param) : GET
// /jobs/{job_id} migré vers Huma préserve le contrat — 200 AsyncJobStatus
// (job_id lié depuis le path, SANS $schema) ; 404 {code:job_not_found,
// retryable:false} au format writeError pour un id inconnu.
func TestRegisterJobsHuma_ContractPreserved(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	job := store.Create(domain.JobTypeInitialSync, "test-player")

	// 200 — job présent, path param correctement lié.
	r := chi.NewRouter()
	registerJobsHuma(newHumaAPI(r), handlers.NewJobsHandler(store))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/jobs/"+job.JobID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("200 attendu, got %d: %s", rec.Code, rec.Body.String())
	}
	var ok map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if ok["job_id"] != job.JobID {
		t.Errorf("job_id = %v, want %v", ok["job_id"], job.JobID)
	}
	if _, has := ok["$schema"]; has {
		t.Error("$schema ne doit PAS être injecté (contrat front identique à writeJSON)")
	}

	// 404 — id inconnu → format writeError.
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/jobs/unknown-id", nil))
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("404 attendu, got %d", rec2.Code)
	}
	var er map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &er); err != nil {
		t.Fatalf("JSON erreur invalide: %v", err)
	}
	if er["code"] != "job_not_found" {
		t.Errorf("code = %v, want job_not_found", er["code"])
	}
	if er["retryable"] != false {
		t.Errorf("retryable = %v, want false", er["retryable"])
	}
}

// TestRegisterGamertagHuma_ContractPreserved (Phase 3b, shape query-param) : GET
// /directory/gamertags/search?q= migré vers Huma préserve le contrat —
// 200 {query, items} (query param lié), 503 si service absent, 500 sur erreur
// service (message générique « internal error » + retryable:true, comme writeError).
func TestRegisterGamertagHuma_ContractPreserved(t *testing.T) {
	newRouter := func(svc *stubGamertagSearch) *chi.Mux {
		r := chi.NewRouter()
		registerGamertagHuma(newHumaAPI(r), handlers.NewGamertagHandler(svc))
		return r
	}
	do := func(r *chi.Mux, url string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
		return rec
	}

	// 200 — query param lié, items renvoyés, SANS $schema.
	r := newRouter(&stubGamertagSearch{results: []domain.GamertagSearchResult{{Gamertag: "Chocoboflor", XUID: "1"}}})
	rec := do(r, "/directory/gamertags/search?q=cho")
	if rec.Code != http.StatusOK {
		t.Fatalf("200 attendu, got %d: %s", rec.Code, rec.Body.String())
	}
	var ok struct {
		Query string `json:"query"`
		Items []struct {
			Gamertag string `json:"gamertag"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &ok); err != nil {
		t.Fatalf("JSON invalide: %v", err)
	}
	if ok.Query != "cho" {
		t.Errorf("query = %q, want cho (query param non lié)", ok.Query)
	}
	if len(ok.Items) != 1 || ok.Items[0].Gamertag != "Chocoboflor" {
		t.Errorf("items = %+v", ok.Items)
	}
	if strings.Contains(rec.Body.String(), "$schema") {
		t.Error("$schema ne doit PAS être injecté")
	}

	// 200 — query vide → items [] sans appel service.
	recEmpty := do(newRouter(&stubGamertagSearch{err: errors.New("ne doit pas être appelé")}), "/directory/gamertags/search?q=")
	if recEmpty.Code != http.StatusOK {
		t.Fatalf("200 attendu (query vide), got %d", recEmpty.Code)
	}

	// 503 — service absent.
	r503 := chi.NewRouter()
	registerGamertagHuma(newHumaAPI(r503), handlers.NewGamertagHandler(nil))
	rec503 := do(r503, "/directory/gamertags/search?q=abc")
	if rec503.Code != http.StatusServiceUnavailable {
		t.Fatalf("503 attendu, got %d", rec503.Code)
	}
	var er503 map[string]any
	if err := json.Unmarshal(rec503.Body.Bytes(), &er503); err != nil {
		t.Fatalf("JSON erreur invalide: %v", err)
	}
	if er503["code"] != "shared_db_unavailable" {
		t.Errorf("code = %v, want shared_db_unavailable", er503["code"])
	}

	// 500 — erreur service (message générique, retryable:true).
	rec500 := do(newRouter(&stubGamertagSearch{err: errors.New("db boom")}), "/directory/gamertags/search?q=abc")
	if rec500.Code != http.StatusInternalServerError {
		t.Fatalf("500 attendu, got %d", rec500.Code)
	}
	var er500 map[string]any
	if err := json.Unmarshal(rec500.Body.Bytes(), &er500); err != nil {
		t.Fatalf("JSON erreur invalide: %v", err)
	}
	if er500["code"] != "gamertag_search_error" {
		t.Errorf("code = %v, want gamertag_search_error", er500["code"])
	}
	if er500["message"] != "internal error" {
		t.Errorf("message = %v, want 'internal error' (pas de fuite err.Error() sur 5xx)", er500["message"])
	}
	if er500["retryable"] != true {
		t.Errorf("retryable = %v, want true", er500["retryable"])
	}
}
