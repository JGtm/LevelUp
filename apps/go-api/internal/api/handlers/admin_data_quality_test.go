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

// dqTranslationCalls enregistre les requêtes de traduction reçues par les runners factices.
type dqTranslationCalls struct {
	mode  []domain.ModeTranslationRequest
	asset []domain.AssetTranslationRequest
}

func newDQHandler(t *testing.T) *AdminDataQualityHandler {
	t.Helper()
	h, _ := newDQHandlerRec(t)
	return h
}

// newDQHandlerRec construit le handler et expose les requêtes de traduction que les runners reçoivent.
func newDQHandlerRec(t *testing.T) (*AdminDataQualityHandler, *dqTranslationCalls) {
	t.Helper()
	calls := &dqTranslationCalls{}
	return NewAdminDataQualityHandler(
		func(_ context.Context, titleSlug, locale string) (domain.AdminDataQualityCounts, error) {
			return domain.AdminDataQualityCounts{TitleSlug: titleSlug, Locale: locale, RawUUIDTotal: 3}, nil
		},
		func(_ context.Context, titleSlug, kind, locale string, limit, offset int) (domain.AdminDataQualityIssues, error) {
			if limit > 500 {
				t.Errorf("limit non clampé : %d", limit)
			}
			if offset < 0 {
				t.Errorf("offset négatif non normalisé : %d", offset)
			}
			return domain.AdminDataQualityIssues{TitleSlug: titleSlug, Kind: kind, Locale: locale, Total: 1,
				Items: []domain.AdminDataQualityIssue{{Kind: "untranslated_mode", ID: "Husky Raid CTF", Occurrences: 4}}}, nil
		},
		func(_ context.Context, _ string, dryRun bool) (domain.RegistryNamesBackfillResult, error) {
			if dryRun {
				return domain.RegistryNamesBackfillResult{DryRun: true, PairsScanned: 2}, nil
			}
			return domain.RegistryNamesBackfillResult{}, errBusySentinel
		},
		func(_ context.Context, _ string, req domain.ModeTranslationRequest) (domain.ResolveResult, error) {
			calls.mode = append(calls.mode, req)
			return domain.ResolveResult{Action: "created", ModeKey: req.ModeKey}, nil
		},
		func(_ context.Context, _ string, req domain.AssetTranslationRequest) (domain.ResolveResult, error) {
			calls.asset = append(calls.asset, req)
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
	), calls
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

// TestAdminDQ_LocaleParam : le paramètre ?locale= est normalisé (trim/minuscule,
// défaut « fr ») et transmis aux runners counts + issues (échoté dans le payload
// pour un libellé front honnête).
func TestAdminDQ_LocaleParam(t *testing.T) {
	h := newDQHandler(t)

	// Issues sans locale → défaut « fr ».
	rec := serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/data-quality/issues?kind=untranslated_modes", nil))
	var iss domain.AdminDataQualityIssues
	if err := json.Unmarshal(rec.Body.Bytes(), &iss); err != nil || iss.Locale != "fr" {
		t.Fatalf("issues locale défaut = %q err=%v (attendu fr)", iss.Locale, err)
	}

	// Issues ?locale=EN → normalisé « en ».
	rec = serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/data-quality/issues?kind=untranslated_modes&locale=EN", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &iss); err != nil || iss.Locale != "en" {
		t.Fatalf("issues locale = %q err=%v (attendu en)", iss.Locale, err)
	}

	// Counts ?locale= vide → défaut « fr ».
	rec = serveAdminDataQuality(h, httptest.NewRequest(http.MethodGet, "/admin/monitoring/data-quality", nil))
	var cnt domain.AdminDataQualityCounts
	if err := json.Unmarshal(rec.Body.Bytes(), &cnt); err != nil || cnt.Locale != "fr" {
		t.Fatalf("counts locale défaut = %q err=%v (attendu fr)", cnt.Locale, err)
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

// dqModeBody / dqAssetBody : corps de requête de traduction pour les tests.
func dqModeBody(key, names string) string {
	return `{"mode_key": "` + key + `", "names": ` + names + `}`
}

func dqAssetBody(kind, id, names string) string {
	return `{"asset_kind": "` + kind + `", "asset_id": "` + id + `", "names": ` + names + `}`
}

// dqErrCode lit le code de l'enveloppe d'erreur d'une réponse.
func dqErrCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode erreur : %v (%s)", err, rec.Body.String())
	}
	code, _ := body["code"].(string)
	return code
}

// TestAdminDQ_Translations_Validation : inputs vides/trop longs → 400 ;
// valides → 200 avec action.
func TestAdminDQ_Translations_Validation(t *testing.T) {
	h := newDQHandler(t)
	const mode = "/admin/actions/translations/mode"
	const asset = "/admin/actions/translations/asset"

	rec := serveAdminDataQuality(h, dqPostJSON(mode, dqModeBody("", `{"fr": "Assassin"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("mode_key vide : status=%d (attendu 400)", rec.Code)
	}

	long := strings.Repeat("x", 200)
	rec = serveAdminDataQuality(h, dqPostJSON(mode, dqModeBody("Slayer", `{"fr": "`+long+`"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("nom trop long : status=%d (attendu 400)", rec.Code)
	}

	rec = serveAdminDataQuality(h, dqPostJSON(mode, dqModeBody("Slayer", `{"fr": "Assassin"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("valide : status=%d (attendu 200)", rec.Code)
	}
	// La réponse nomme la clé technique écrite `mode_key` (clé du mode, pas un libellé) : plus de `mode_en`.
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode : %v", err)
	}
	if resp["mode_key"] != "Slayer" {
		t.Errorf("mode_key = %v, attendu Slayer : %v", resp["mode_key"], resp)
	}
	if _, ok := resp["mode_en"]; ok {
		t.Errorf("mode_en ne fait plus partie de la réponse : %v", resp)
	}

	// Asset : aucun nom fourni → 400.
	rec = serveAdminDataQuality(h, dqPostJSON(asset, `{"asset_kind": "pair", "asset_id": "p-1"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("asset sans nom : status=%d (attendu 400)", rec.Code)
	}
	rec = serveAdminDataQuality(h, dqPostJSON(asset, dqAssetBody("pair", "p-1", `{"fr": "Fiesta"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset valide : status=%d (attendu 200)", rec.Code)
	}
}

// TestAdminDQ_Translations_OnePerLanguage : une saisie fr + en, ou une seule langue, arrive au
// runner telle quelle (noms rognés, clé = code de langue).
func TestAdminDQ_Translations_OnePerLanguage(t *testing.T) {
	h, calls := newDQHandlerRec(t)

	rec := serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/mode",
		dqModeBody("Slayer", `{"fr": "  Assassin ", "en": "Slayer"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("mode fr+en : status=%d (attendu 200) %s", rec.Code, rec.Body.String())
	}
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/mode",
		dqModeBody("Strongholds", `{"en": "Strongholds"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("mode une langue : status=%d (attendu 200) %s", rec.Code, rec.Body.String())
	}
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/asset",
		dqAssetBody("map", "m-1", `{"fr": "Aquarius", "en": "Aquarius EN"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset fr+en : status=%d (attendu 200) %s", rec.Code, rec.Body.String())
	}
	// Les entrées vides sont écartées : une seule langue renseignée suffit.
	rec = serveAdminDataQuality(h, dqPostJSON("/admin/actions/translations/asset",
		dqAssetBody("map", "m-2", `{"fr": "", "en": "Streets"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset une langue : status=%d (attendu 200) %s", rec.Code, rec.Body.String())
	}

	if len(calls.mode) != 2 || len(calls.asset) != 2 {
		t.Fatalf("appels runner : %d mode, %d asset (attendu 2 et 2)", len(calls.mode), len(calls.asset))
	}
	if got := calls.mode[0].Names; len(got) != 2 || got["fr"] != "Assassin" || got["en"] != "Slayer" {
		t.Errorf("mode fr+en : names = %v", got)
	}
	if got := calls.mode[1].Names; len(got) != 1 || got["en"] != "Strongholds" {
		t.Errorf("mode une langue : names = %v", got)
	}
	if got := calls.asset[0].Names; len(got) != 2 || got["fr"] != "Aquarius" || got["en"] != "Aquarius EN" {
		t.Errorf("asset fr+en : names = %v", got)
	}
	if got := calls.asset[1]; got.AssetID != "m-2" || len(got.Names) != 1 || got.Names["en"] != "Streets" {
		t.Errorf("asset une langue : %+v", got)
	}
}

// TestAdminDQ_Translations_RejectedNames : langue inconnue (code distinct), toutes valeurs vides,
// names absent ou null → 400 et aucun appel au runner.
func TestAdminDQ_Translations_RejectedNames(t *testing.T) {
	h, calls := newDQHandlerRec(t)
	const mode = "/admin/actions/translations/mode"
	const asset = "/admin/actions/translations/asset"

	cases := []struct {
		name, path, body, code string
	}{
		{"mode langue inconnue", mode, dqModeBody("Slayer", `{"zz": "Assassin"}`), "unknown_locale"},
		{"mode langue inconnue avec une valide", mode, dqModeBody("Slayer", `{"fr": "Assassin", "zz": "x"}`), "unknown_locale"},
		{"mode etiquette complete refusee", mode, dqModeBody("Slayer", `{"en-US": "Slayer"}`), "unknown_locale"},
		{"mode valeurs vides", mode, dqModeBody("Slayer", `{"fr": "", "en": "   "}`), "invalid_input"},
		{"mode names vide", mode, dqModeBody("Slayer", `{}`), "invalid_input"},
		{"mode names null", mode, dqModeBody("Slayer", `null`), "invalid_input"},
		{"asset langue inconnue", asset, dqAssetBody("pair", "p-1", `{"zz": "Fiesta"}`), "unknown_locale"},
		{"asset valeurs vides", asset, dqAssetBody("pair", "p-1", `{"fr": "", "en": ""}`), "invalid_input"},
		{"asset names absent", asset, `{"asset_kind": "pair", "asset_id": "p-1"}`, "invalid_input"},
		{"asset id vide", asset, dqAssetBody("pair", "", `{"fr": "Fiesta"}`), "invalid_input"},
	}
	for _, c := range cases {
		rec := serveAdminDataQuality(h, dqPostJSON(c.path, c.body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s : status=%d (attendu 400) %s", c.name, rec.Code, rec.Body.String())
			continue
		}
		if got := dqErrCode(t, rec); got != c.code {
			t.Errorf("%s : code=%q (attendu %q)", c.name, got, c.code)
		}
	}
	if len(calls.mode) != 0 || len(calls.asset) != 0 {
		t.Errorf("le runner ne doit pas être appelé sur une saisie refusée : %d mode, %d asset", len(calls.mode), len(calls.asset))
	}
}
