package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

type stubDiagnoser struct {
	rep *domain.TitleDiagnostic
}

func (s stubDiagnoser) Diagnose(_ context.Context, _ string) (*domain.TitleDiagnostic, error) {
	return s.rep, nil
}

func TestAdminTitleDiagnostic_Get(t *testing.T) {
	h := NewAdminTitleDiagnosticHandler(stubDiagnoser{rep: &domain.TitleDiagnostic{
		TitleSlug:   "halo_infinite",
		ConfigFiles: []domain.ConfigFileStatus{{Name: "fields.toml", Present: true, Required: true}},
		Databases: []domain.DatabaseStatus{{
			Name:   "metadata.duckdb",
			Exists: true,
			Tables: []domain.TableStatus{{Name: "season_calendars", Exists: true, Rows: 5}},
		}},
	}}, nil)
	r := chi.NewRouter()
	r.Get("/admin/titles/{slug}/diagnostic", h.Get)

	req := httptest.NewRequest("GET", "/admin/titles/halo_infinite/diagnostic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var rep domain.TitleDiagnostic
	if err := json.Unmarshal(w.Body.Bytes(), &rep); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rep.TitleSlug != "halo_infinite" || len(rep.Databases) != 1 {
		t.Errorf("rep = %+v", rep)
	}
}
