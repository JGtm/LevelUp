package handlers

// pagination_ceiling_test.go — plafond des paramètres de taille déclarés en chaîne
// (limit, n, count) : une valeur démesurée ne dépasse jamais le plafond du handler.
// Chaque test est cité par paginationStringParams (internal/api/pagination_guard_test.go) ;
// le renommer exige de mettre cette liste à jour.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/prestige"
)

const hugeCount = "1000000000"

func TestAdminDQ_GetIssues_LimitCeiling(t *testing.T) {
	var gotLimit int
	h := NewAdminDataQualityHandler(
		func(_ context.Context, titleSlug, locale string) (domain.AdminDataQualityCounts, error) {
			return domain.AdminDataQualityCounts{TitleSlug: titleSlug, Locale: locale}, nil
		},
		func(_ context.Context, titleSlug, kind, locale string, limit, _ int) (domain.AdminDataQualityIssues, error) {
			gotLimit = limit
			return domain.AdminDataQualityIssues{TitleSlug: titleSlug, Kind: kind, Locale: locale}, nil
		},
		nil, nil, nil, nil, nil, errBusySentinel,
	)
	rec := serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet,
		"/admin/monitoring/data-quality/issues?kind=untranslated_modes&limit="+hugeCount, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	if gotLimit != 500 {
		t.Errorf("limite transmise = %d, attendu le plafond 500", gotLimit)
	}
}

func TestAdminMonitoring_Jobs_LimitCeiling(t *testing.T) {
	store := jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
	for i := 0; i < 60; i++ {
		store.Create(domain.JobTypeBackfill, fmt.Sprintf("p%d", i))
	}
	h := NewAdminMonitoringHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, store)
	rec := serveAdminMonitoring(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/jobs?limit="+hugeCount, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got AdminJobsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if len(got.Jobs) != 50 {
		t.Errorf("jobs servis = %d, attendu le plafond 50", len(got.Jobs))
	}
}

func TestAdminLogsTail_NCeiling(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	base := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 1100; i++ {
		fmt.Fprintf(&sb, `{"time":"%s","level":"INFO","msg":"ligne-%04d"}`+"\n",
			base.Add(time.Duration(i)*time.Second).Format(time.RFC3339Nano), i)
	}
	if err := os.WriteFile(filepath.Join(dir, "sync.log"), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("écriture du journal : %v", err)
	}
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) { NewAdminLogsHandler(dir).Mount(r) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/monitoring/logs/tail?module=sync&n="+hugeCount, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200) : %s", rec.Code, rec.Body.String())
	}
	var got domain.AdminLogTail
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if len(got.Entries) != 1000 {
		t.Errorf("entrées servies = %d, attendu le plafond 1000", len(got.Entries))
	}
}

// nCapturingPatternsRepo retient le n demandé au chargement des lignes.
type nCapturingPatternsRepo struct {
	mockPatternsRepo
	gotN int
}

func (m *nCapturingPatternsRepo) LoadRows(_ context.Context, n int) ([]patterns.MatchRow, error) {
	m.gotN = n
	return nil, nil
}

func TestPatterns_NCeiling(t *testing.T) {
	repo := &nCapturingPatternsRepo{}
	h := NewPatternsHandler(func(_ context.Context, _ string) (port.PatternsRepository, error) {
		return repo, nil
	}, "halo_infinite")
	if _, err := h.GetPatterns(context.Background(), &patternsInput{PlayerSlug: "p", N: hugeCount}); err != nil {
		t.Fatalf("GetPatterns : %v", err)
	}
	if repo.gotN > patternMaxN {
		t.Errorf("n transmis au chargement = %d, au-dessus du plafond %d", repo.gotN, patternMaxN)
	}
}

// countCapturingPrestige retient le count demandé à SuggestTemplates.
type countCapturingPrestige struct {
	mockPrestigeService
	gotCount int
}

func (m *countCapturingPrestige) SuggestTemplates(_ context.Context, _, _ string, count int) ([]prestige.Template, error) {
	m.gotCount = count
	return nil, nil
}

func TestPrestige_SuggestTemplates_CountCeiling(t *testing.T) {
	svc := &countCapturingPrestige{}
	rec := httptest.NewRecorder()
	newRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/templates/suggest?user_id=u1&title_slug=halo_infinite&count="+hugeCount, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	if svc.gotCount < 1 || svc.gotCount > 10 {
		t.Errorf("count transmis au service = %d, attendu dans [1, 10]", svc.gotCount)
	}
}
