// Package handlers — admin_data_quality_test.go : contrats HTTP des endpoints
// qualité données (validation kind/inputs, 409 single-flight, payloads).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

var errBusySentinel = errors.New("busy")

func newDQHandler(t *testing.T) *AdminDataQualityHandler {
	t.Helper()
	return NewAdminDataQualityHandler(
		func(_ context.Context, titleSlug string) (domain.AdminDataQualityCounts, error) {
			return domain.AdminDataQualityCounts{TitleSlug: titleSlug, RawUUIDTotal: 3}, nil
		},
		func(_ context.Context, titleSlug, kind string, limit int) (domain.AdminDataQualityIssues, error) {
			if limit > 500 {
				t.Errorf("limit non clampé : %d", limit)
			}
			return domain.AdminDataQualityIssues{TitleSlug: titleSlug, Kind: kind,
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
	rec := httptest.NewRecorder()
	h.RunCatalogRefresh(rec, httptest.NewRequest(http.MethodPost, "/x", nil))
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

	rec := httptest.NewRecorder()
	h.RunLyingBitsReset(rec, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"dry_run":true}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("dry-run : status=%d (attendu 200)", rec.Code)
	}
	var got domain.LyingBitsResetResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !got.DryRun || got.EventsBitsCleared != 7 {
		t.Fatalf("payload dry-run inattendu : %+v err=%v", got, err)
	}

	rec = httptest.NewRecorder()
	h.RunLyingBitsReset(rec, httptest.NewRequest(http.MethodPost, "/x", nil)) // corps vide → run réel → busy
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

	rec := httptest.NewRecorder()
	h.GetIssues(rec, httptest.NewRequest(http.MethodGet, "/x/issues?kind=nimporte", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("kind invalide : status=%d (attendu 400)", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.GetIssues(rec, httptest.NewRequest(http.MethodGet, "/x/issues?kind=untranslated_modes&limit=9999", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("kind valide : status=%d (attendu 200)", rec.Code)
	}
	var got domain.AdminDataQualityIssues
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || len(got.Items) != 1 {
		t.Fatalf("payload inattendu : %+v err=%v", got, err)
	}
}

// TestAdminDQ_RegistryNames_DryRunAndBusy : dry-run → 200 compteurs ; runner
// busy (sentinelle) → 409 enveloppe already_running.
func TestAdminDQ_RegistryNames_DryRunAndBusy(t *testing.T) {
	h := newDQHandler(t)

	rec := httptest.NewRecorder()
	h.RunRegistryNamesBackfill(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"dry_run": true}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("dry-run : status=%d (attendu 200) body=%s", rec.Code, rec.Body.String())
	}
	var res domain.RegistryNamesBackfillResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || !res.DryRun || res.PairsScanned != 2 {
		t.Fatalf("payload dry-run inattendu : %+v err=%v", res, err)
	}

	rec = httptest.NewRecorder()
	h.RunRegistryNamesBackfill(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"dry_run": false}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("busy : status=%d (attendu 409)", rec.Code)
	}
}

// TestAdminDQ_Translations_Validation : inputs vides/trop longs → 400 ;
// valides → 200 avec action.
func TestAdminDQ_Translations_Validation(t *testing.T) {
	h := newDQHandler(t)

	rec := httptest.NewRecorder()
	h.ResolveModeTranslation(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"mode_en": "", "name_fr": "Assassin"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mode_en vide : status=%d (attendu 400)", rec.Code)
	}

	long := strings.Repeat("x", 200)
	rec = httptest.NewRecorder()
	h.ResolveModeTranslation(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"mode_en": "Slayer", "name_fr": "`+long+`"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("name_fr trop long : status=%d (attendu 400)", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ResolveModeTranslation(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"mode_en": "Slayer", "name_fr": "Assassin"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("valide : status=%d (attendu 200)", rec.Code)
	}

	// Asset : aucun nom fourni → 400.
	rec = httptest.NewRecorder()
	h.ResolveAssetTranslation(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"asset_kind": "pair", "asset_id": "p-1"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("asset sans nom : status=%d (attendu 400)", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ResolveAssetTranslation(rec, httptest.NewRequest(http.MethodPost, "/x",
		strings.NewReader(`{"asset_kind": "pair", "asset_id": "p-1", "name_fr": "Fiesta"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset valide : status=%d (attendu 200)", rec.Code)
	}
}
