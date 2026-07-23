// Package handlers — admin_data_quality_test.go : contrats HTTP des endpoints
// qualité données (validation kind/inputs, 409 single-flight, payloads).
//
// MIGRÉ vers Huma : les requêtes passent par un routeur chi montant h.Mount sous
// /admin (même point de montage que server_admin_monitoring.go). Bodies et
// assertions inchangés.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

var errBusySentinel = errors.New("busy")

// serveAdminDataQuality monte h sous /admin (point de montage de
// server_admin_monitoring.go) et sert la requête via le routeur chi.
func serveAdminDataQuality(h *AdminDataQualityHandler, req *http.Request) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Route("/admin", func(r chi.Router) {
		h.Mount(r)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// dqPostJSON construit un POST JSON (Content-Type requis par Huma pour peupler
// RawBody depuis un corps présent).
func dqPostJSON(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newDQHandler(t *testing.T) *AdminDataQualityHandler {
	t.Helper()
	return NewAdminDataQualityHandler(
		func(_ context.Context, titleSlug string) (domain.AdminDataQualityCounts, error) {
			return domain.AdminDataQualityCounts{TitleSlug: titleSlug, RawUUIDTotal: 3}, nil
		},
		func(_ context.Context, titleSlug, kind string, limit, offset int) (domain.AdminDataQualityIssues, error) {
			if limit > 500 {
				t.Errorf("limit non clampé : %d", limit)
			}
			if offset < 0 {
				t.Errorf("offset négatif non normalisé : %d", offset)
			}
			return domain.AdminDataQualityIssues{TitleSlug: titleSlug, Kind: kind, Total: 1,
				Items: []domain.AdminDataQualityIssue{{Kind: "untranslated_mode", ID: "Husky Raid CTF", Occurrences: 4}}}, nil
		},
		func(_ context.Context, _ string, dryRun bool) (domain.RegistryNamesBackfillResult, error) {
			if dryRun {
				return domain.RegistryNamesBackfillResult{DryRun: true, PairsScanned: 2}, nil
			}
			return domain.RegistryNamesBackfillResult{}, errBusySentinel
		},
		func(_ context.Context, _, modeEN, nameFR string) (domain.ResolveResult, error) {
			return domain.ResolveResult{Action: "created", ModeEN: modeEN}, nil
		},
		func(_ context.Context, _ string, req domain.AssetTranslationRequest) (domain.ResolveResult, error) {
			return domain.ResolveResult{Action: "created", Langs: []string{"fr-FR"}}, nil
		},
		func(_ context.Context, _ string) (domain.CatalogRefreshResult, error) {
			return domain.CatalogRefreshResult{Playlists: 5}, nil
		},
		func(_ context.Context, _ string, dryRun bool) (domain.LyingBitsResetResult, error) {
			if dryRun {
				return domain.LyingBitsResetResult{DryRun: true, EventsBitsCleared: 7, Total: 7}, nil
			}
			return domain.LyingBitsResetResult{}, errBusySentinel
		},
		errBusySentinel,
	)
}

// TestAdminDQ_CatalogRefresh : 200 + compteurs.
func TestAdminDQ_CatalogRefresh(t *testing.T) {
	h := newDQHandler(t)
	rec := serveAdminDataQuality(h, httptest.NewRequest(http.MethodPost, "/admin/actions/catalog/refresh", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (attendu 200)", rec.Code)
	}
	var got domain.CatalogRefreshResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Playlists != 5 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}
}

// TestAdminDQ_LyingBitsReset_DryRunAndBusy : dry-run → 200 compteurs ; run réel
// busy (sentinelle) → 409 enveloppe already_running.
func TestAdminDQ_LyingBitsReset_DryRunAndBusy(t *testing.T) {
	h := newDQHandler(t)

	rec := serveAdminDataQuality(h, dqPostJSON("/admin/actions/lying-bits/reset", `{"dry_run":true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("dry-run : status=%d (attendu 200)", rec.Code)
	}
	var got domain.LyingBitsResetResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !got.DryRun || got.EventsBitsCleared != 7 {
		t.Fatalf("payload dry-run inattendu : %+v err=%v", got, err)
	}

	rec = serveAdminDataQuality(h, httptest.NewRequest(http.MethodPost, "/admin/actions/lying-bits/reset", nil)) // corps vide → run réel → busy
	if rec.Code != http.StatusConflict {
		t.Fatalf("busy : status=%d (attendu 409)", rec.Code)
	}
	var conflict map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &conflict); err != nil || conflict["code"] != actionBusyCode {
		t.Fatalf("enveloppe 409 inattendue : %+v err=%v", conflict, err)
	}
}

// TestAdminDQ_GetIssues_KindValidation : kind absent/inconnu → 400 ; kind
// valide → 200 avec items.
func TestAdminDQ_GetIssues_KindValidation(t *testing.T) {
	h := newDQHandler(t)

	rec := serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/data-quality/issues?kind=nimporte", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("kind invalide : status=%d (attendu 400)", rec.Code)
	}

	rec = serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/data-quality/issues?kind=untranslated_modes&limit=9999&offset=25", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("kind valide : status=%d (attendu 200)", rec.Code)
	}
	var got domain.AdminDataQualityIssues
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Items) != 1 || got.Total != 1 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}
}

// TestAdminDQ_RegistryNames_DryRunAndBusy : dry-run → 200 compteurs ; runner
// busy (sentinelle) → 409 enveloppe already_running.
func TestAdminDQ_RegistryNames_DryRunAndBusy(t *testing.T) {
	h := newDQHandler(t)

	rec := serveAdminDataQuality(h, dqPostJSON("/admin/actions/registry-names/backfill", `{"dry_run": true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("dry-run : status=%d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var res domain.RegistryNamesBackfillResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || !res.DryRun || res.PairsScanned != 2 {
		t.Fatalf("payload dry-run inattendu : %+v err=%v", res, err)
	}

	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/registry-names/backfill", `{"dry_run": false}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("busy : status=%d (attendu 409)", rec.Code)
	}
}

// TestAdminDQ_Translations_Validation : inputs vides/trop longs → 400 ;
// valides → 200 avec action.
func TestAdminDQ_Translations_Validation(t *testing.T) {
	h := newDQHandler(t)

	rec := serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/mode",
		`{"mode_en": "", "name_fr": "Assassin"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mode_en vide : status=%d (attendu 400)", rec.Code)
	}

	long := strings.Repeat("x", 200)
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/mode",
		`{"mode_en": "Slayer", "name_fr": "`+long+`"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("name_fr trop long : status=%d (attendu 400)", rec.Code)
	}

	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/mode",
		`{"mode_en": "Slayer", "name_fr": "Assassin"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("valide : status=%d (attendu 200)", rec.Code)
	}

	// Asset : aucun nom fourni → 400.
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/asset",
		`{"asset_kind": "pair", "asset_id": "p-1"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("asset sans nom : status=%d (attendu 400)", rec.Code)
	}
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/asset",
		`{"asset_kind": "pair", "asset_id": "p-1", "name_fr": "Fiesta"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset valide : status=%d (attendu 200)", rec.Code)
	}
}
