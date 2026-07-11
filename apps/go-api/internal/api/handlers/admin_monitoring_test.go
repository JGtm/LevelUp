// Package handlers — admin_monitoring_test.go : contrats HTTP des endpoints
// lecture du dashboard monitoring (runners mockés — pattern admin_token_health).
//
// MIGRÉ vers Huma : les requêtes passent par un routeur chi montant h.Mount sous
// /admin (même point de montage que server_admin_monitoring.go). Requêtes et
// assertions inchangées.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
)

// serveAdminMonitoring monte h sous /admin (point de montage de
// server_admin_monitoring.go) et sert la requête via le routeur chi.
func serveAdminMonitoring(h *AdminMonitoringHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func okOverviewRunner(t *testing.T, wantTitle string) MonitoringOverviewRunner {
	t.Helper()
	return func(_ context.Context, titleSlug string) (domain.AdminMonitoringOverview, error) {
		if titleSlug != wantTitle {
			t.Errorf("titleSlug = %q (attendu %q)", titleSlug, wantTitle)
		}
		return domain.AdminMonitoringOverview{
			TitleSlug: titleSlug,
			Scheduler: domain.MonitoringSchedulerSummary{Available: true, LastSynced: 2},
		}, nil
	}
}

// TestAdminMonitoring_Overview_OKAndTitleDefault : 200 + titre par défaut
// quand ?title= absent.
func TestAdminMonitoring_Overview_OKAndTitleDefault(t *testing.T) {
	h := NewAdminMonitoringHandler(okOverviewRunner(t, "halo_infinite"), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	rec := serveAdminMonitoring(h, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var got domain.AdminMonitoringOverview
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if !got.Scheduler.Available || got.Scheduler.LastSynced != 2 {
		t.Fatalf("payload inattendu : %+v", got.Scheduler)
	}
}

// TestAdminMonitoring_Overview_RunnerError : erreur runner → 500 enveloppé.
func TestAdminMonitoring_Overview_RunnerError(t *testing.T) {
	h := NewAdminMonitoringHandler(func(context.Context, string) (domain.AdminMonitoringOverview, error) {
		return domain.AdminMonitoringOverview{}, errors.New("boom")
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview?title=halo_infinite", nil)
	rec := serveAdminMonitoring(h, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d (attendu 500)", rec.Code)
	}
}

// TestAdminMonitoring_Scheduler_UnavailableWhenNil : scheduler non câblé →
// 200 avec available=false et history=[] (jamais de 500/panic).
func TestAdminMonitoring_Scheduler_UnavailableWhenNil(t *testing.T) {
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/scheduler", nil)
	rec := serveAdminMonitoring(h, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got AdminSchedulerStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.Available || got.Snapshot != nil {
		t.Fatalf("scheduler nil doit donner available=false sans snapshot : %+v", got)
	}
	if got.History == nil {
		t.Fatal("history doit être un tableau vide, pas null")
	}
	if got.ZeroInsertWarnThreshold <= 0 {
		t.Fatal("le seuil zero-insert doit être exposé au front")
	}
}

// TestAdminMonitoring_Convergence_OK : 200 + payload du runner.
func TestAdminMonitoring_Convergence_OK(t *testing.T) {
	h := NewAdminMonitoringHandler(nil, func(_ context.Context, titleSlug string) (domain.AdminConvergenceReport, error) {
		return domain.AdminConvergenceReport{
			TitleSlug: titleSlug,
			Horizon:   50,
			Players: []domain.PlayerConvergenceReport{
				{Gamertag: "JGtm", MissingEvents: 3},
			},
		}, nil
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/convergence", nil)
	rec := serveAdminMonitoring(h, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got domain.AdminConvergenceReport
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if got.Horizon != 50 || len(got.Players) != 1 || got.Players[0].MissingEvents != 3 {
		t.Fatalf("payload inattendu : %+v", got)
	}
}

// TestAdminMonitoring_Jobs_ListAndClamp : jobs listés depuis le store, limit
// clampé à 50, store nil → tableau vide.
func TestAdminMonitoring_Jobs_ListAndClamp(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	store.Create(domain.JobTypeBackfill, "p1")

	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, store)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/jobs?limit=9999", nil)
	rec := serveAdminMonitoring(h, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got AdminJobsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if len(got.Jobs) != 1 {
		t.Fatalf("len(jobs) = %d (attendu 1)", len(got.Jobs))
	}

	// Store nil : dégradation sans panic.
	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	recNil := serveAdminMonitoring(hNil, httptest.NewRequest(http.MethodGet, "/admin/monitoring/jobs", nil))
	if recNil.Code != http.StatusOK {
		t.Fatalf("status store nil = %d (attendu 200)", recNil.Code)
	}
}

// TestAdminMonitoring_Errors_OK : 200 + buckets agrégés + propagation du slug
// (MT-05 : ?title= propagé au runner, défaut sinon).
func TestAdminMonitoring_Errors_OK(t *testing.T) {
	var gotSlug string
	runner := func(_ context.Context, titleSlug string) (domain.AdminErrorStats, error) {
		gotSlug = titleSlug
		return domain.AdminErrorStats{
			GeneratedAt: "2026-06-12T12:00:00Z",
			Buckets: []domain.AdminErrorBucket{
				{Level: "ERROR", Module: "player_watcher", Message: "player_watcher: sync échoué", Count: 3},
			},
		}, nil
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, runner, nil, nil, nil, nil, nil, nil, nil)

	rec := serveAdminMonitoring(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/errors", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	if gotSlug != titlePkg.DefaultSlug {
		t.Errorf("sans ?title= : slug propagé = %q, want %q", gotSlug, titlePkg.DefaultSlug)
	}
	var got domain.AdminErrorStats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Buckets) != 1 || got.Buckets[0].Count != 3 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}

	serveAdminMonitoring(h,
		httptest.NewRequest(http.MethodGet, "/admin/monitoring/errors?title=synthetic_title_b", nil))
	if gotSlug != "synthetic_title_b" {
		t.Errorf("?title= : slug propagé = %q, want synthetic_title_b", gotSlug)
	}
}

// TestAdminMonitoring_Detections_ListWithFilters : 200 + filtres propagés au
// runner ; store nil (runner nil) → liste vide sans panic.
func TestAdminMonitoring_Detections_ListWithFilters(t *testing.T) {
	var gotStatus, gotLevel string
	var gotLimit int
	runner := func(_ context.Context, status, level, _ /*module*/, _ /*title*/ string, limit int) (domain.AdminDetectionsResponse, error) {
		gotStatus, gotLevel, gotLimit = status, level, limit
		return domain.AdminDetectionsResponse{
			GeneratedAt: "2026-07-10T12:00:00Z",
			OpenCount:   1,
			Detections: []domain.MonitoringDetection{
				{Fingerprint: "abc", Level: "WARN", Message: "m", Count: 4, Status: "open"},
			},
		}, nil
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, runner, nil, nil, nil, nil, nil, nil)

	rec := serveAdminMonitoring(h,
		httptest.NewRequest(http.MethodGet, "/admin/monitoring/detections?status=open&level=WARN&limit=25", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	if gotStatus != "open" || gotLevel != "WARN" || gotLimit != 25 {
		t.Errorf("filtres propagés : status=%q level=%q limit=%d", gotStatus, gotLevel, gotLimit)
	}
	var got domain.AdminDetectionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Detections) != 1 || got.OpenCount != 1 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}

	// Runner nil → dégradation en liste vide.
	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	recNil := serveAdminMonitoring(hNil, httptest.NewRequest(http.MethodGet, "/admin/monitoring/detections", nil))
	if recNil.Code != http.StatusOK {
		t.Fatalf("runner nil : status = %d (attendu 200)", recNil.Code)
	}
	var gotNil domain.AdminDetectionsResponse
	if err := json.Unmarshal(recNil.Body.Bytes(), &gotNil); err != nil || gotNil.Detections == nil {
		t.Fatalf("runner nil doit donner detections=[] : %+v err=%v", gotNil, err)
	}
}

// TestAdminMonitoring_Detections_Patch : PATCH statue une détection ; statut
// invalide → 400 ; runner nil → 503.
func TestAdminMonitoring_Detections_Patch(t *testing.T) {
	var gotFp, gotStatus, gotNote string
	setter := func(_ context.Context, fingerprint, status, note string) error {
		gotFp, gotStatus, gotNote = fingerprint, status, note
		return nil
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, setter, nil, nil, nil, nil, nil)

	body := bytes.NewReader([]byte(`{"status":"acked","note":"vu"}`))
	req := httptest.NewRequest(http.MethodPatch, "/admin/monitoring/detections/abc123", body)
	req.Header.Set("Content-Type", "application/json")
	rec := serveAdminMonitoring(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	if gotFp != "abc123" || gotStatus != "acked" || gotNote != "vu" {
		t.Errorf("PATCH propagé : fp=%q status=%q note=%q", gotFp, gotStatus, gotNote)
	}

	// Statut invalide → 400.
	badReq := httptest.NewRequest(http.MethodPatch, "/admin/monitoring/detections/abc123",
		bytes.NewReader([]byte(`{"status":"bogus"}`)))
	badReq.Header.Set("Content-Type", "application/json")
	if rec := serveAdminMonitoring(h, badReq); rec.Code != http.StatusBadRequest {
		t.Fatalf("statut invalide : status = %d (attendu 400)", rec.Code)
	}

	// Setter nil → 503.
	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	nilReq := httptest.NewRequest(http.MethodPatch, "/admin/monitoring/detections/abc123",
		bytes.NewReader([]byte(`{"status":"acked"}`)))
	nilReq.Header.Set("Content-Type", "application/json")
	if rec := serveAdminMonitoring(hNil, nilReq); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("setter nil : status = %d (attendu 503)", rec.Code)
	}
}

// TestAdminMonitoring_Freshness_OK : 200 + filtre titre propagé SANS fallback
// défaut (vide = tous les titres actifs) ; runner nil → réponse vide propre.
func TestAdminMonitoring_Freshness_OK(t *testing.T) {
	var gotFilter string
	runner := func(_ context.Context, titleFilter string) (domain.AdminFreshnessResponse, error) {
		gotFilter = titleFilter
		return domain.AdminFreshnessResponse{
			GeneratedAt:   "2026-07-10T12:00:00Z",
			CriticalTotal: 1,
			Titles: []domain.TitleFreshnessReport{
				{TitleSlug: "halo_infinite", CriticalCount: 1, Players: []domain.PlayerFreshness{
					{Gamertag: "JGtm", Status: "critical"},
				}},
			},
		}, nil
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, runner, nil, nil, nil, nil)

	rec := serveAdminMonitoring(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/freshness", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	if gotFilter != "" {
		t.Errorf("sans ?title= : filtre = %q, attendu vide (tous titres)", gotFilter)
	}
	var got domain.AdminFreshnessResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.CriticalTotal != 1 || len(got.Titles) != 1 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}

	serveAdminMonitoring(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/freshness?title=halo_5", nil))
	if gotFilter != "halo_5" {
		t.Errorf("?title= : filtre = %q, attendu halo_5", gotFilter)
	}

	// Runner nil → réponse vide sans panic.
	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	recNil := serveAdminMonitoring(hNil, httptest.NewRequest(http.MethodGet, "/admin/monitoring/freshness", nil))
	if recNil.Code != http.StatusOK {
		t.Fatalf("runner nil : status = %d (attendu 200)", recNil.Code)
	}
}

// TestAdminMonitoring_Crons_OK : 200 + payload du runner ; runner nil → listes
// vides sans panic (A6.4).
func TestAdminMonitoring_Crons_OK(t *testing.T) {
	runner := func(_ context.Context) (domain.AdminCronsResponse, error) {
		return domain.AdminCronsResponse{
			GeneratedAt: "2026-07-10T12:00:00Z",
			Crons: []domain.CronStatusEntry{
				{Name: "auto_sync", Status: "ok", Runs: 4, SinceBoot: true},
				{Name: "backup", Status: "critical", ConsecutiveFailures: 3, SinceBoot: true},
			},
			Features: []domain.FeatureHeartbeat{
				{Feature: "prestige_hook", Status: "never"},
				{Feature: "watcher_rta", Status: "ok", AgeSeconds: 12},
			},
		}, nil
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, runner, nil, nil)

	rec := serveAdminMonitoring(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/crons", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var got domain.AdminCronsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Crons) != 2 || len(got.Features) != 2 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}
	if got.Crons[1].Status != "critical" || got.Features[0].Status != "never" {
		t.Fatalf("statuts non propagés : %+v", got)
	}

	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	recNil := serveAdminMonitoring(hNil, httptest.NewRequest(http.MethodGet, "/admin/monitoring/crons", nil))
	if recNil.Code != http.StatusOK {
		t.Fatalf("runner nil : status = %d (attendu 200)", recNil.Code)
	}
}
