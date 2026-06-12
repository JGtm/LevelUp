// Package handlers — admin_monitoring_test.go : contrats HTTP des endpoints
// lecture du dashboard monitoring (runners mockés — pattern admin_token_health).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

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
	h := NewAdminMonitoringHandler(okOverviewRunner(t, "halo_infinite"), nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview", nil)
	rec := httptest.NewRecorder()
	h.GetOverview(rec, req)

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
	}, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/overview?title=halo_infinite", nil)
	rec := httptest.NewRecorder()
	h.GetOverview(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d (attendu 500)", rec.Code)
	}
}

// TestAdminMonitoring_Scheduler_UnavailableWhenNil : scheduler non câblé →
// 200 avec available=false et history=[] (jamais de 500/panic).
func TestAdminMonitoring_Scheduler_UnavailableWhenNil(t *testing.T) {
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/scheduler", nil)
	rec := httptest.NewRecorder()
	h.GetScheduler(rec, req)

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
	}, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/convergence", nil)
	rec := httptest.NewRecorder()
	h.GetConvergence(rec, req)

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

	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, store)
	req := httptest.NewRequest(http.MethodGet, "/admin/monitoring/jobs?limit=9999", nil)
	rec := httptest.NewRecorder()
	h.GetJobs(rec, req)

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
	hNil := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil)
	recNil := httptest.NewRecorder()
	hNil.GetJobs(recNil, httptest.NewRequest(http.MethodGet, "/admin/monitoring/jobs", nil))
	if recNil.Code != http.StatusOK {
		t.Fatalf("status store nil = %d (attendu 200)", recNil.Code)
	}
}

// TestAdminMonitoring_Errors_OK : 200 + buckets d'erreurs agrégés.
func TestAdminMonitoring_Errors_OK(t *testing.T) {
	h := NewAdminMonitoringHandler(nil, nil, nil, func(context.Context) (domain.AdminErrorStats, error) {
		return domain.AdminErrorStats{
			GeneratedAt: "2026-06-12T12:00:00Z",
			Buckets: []domain.AdminErrorBucket{
				{Level: "ERROR", Module: "player_watcher", Message: "player_watcher: sync échoué", Count: 3},
			},
		}, nil
	}, nil, nil)
	rec := httptest.NewRecorder()
	h.GetErrors(rec, httptest.NewRequest(http.MethodGet, "/admin/monitoring/errors", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got domain.AdminErrorStats
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Buckets) != 1 || got.Buckets[0].Count != 3 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}
}
