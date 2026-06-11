// Package handlers — admin_data_quality.go : endpoints qualité données du
// dashboard monitoring admin (compteurs + listes d'inconnus, et actions de
// résolution synchrones : backfill registry names, traductions metadata).
//
// Routes (RequireAuth+RequireAdmin) :
//   - GET  /admin/monitoring/data-quality              : compteurs
//   - GET  /admin/monitoring/data-quality/issues       : listes (kind, limit)
//   - POST /admin/actions/registry-names/backfill      : {dry_run}
//   - POST /admin/actions/translations/mode            : {mode_en, name_fr}
//   - POST /admin/actions/translations/asset           : {asset_kind, asset_id, name_en?, name_fr?}
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"levelup/go-api/internal/domain"
)

// Runners injectés (implémentés par ServiceRegistry).
type (
	DataQualityCountsRunner func(ctx context.Context, titleSlug string) (domain.AdminDataQualityCounts, error)
	DataQualityIssuesRunner func(ctx context.Context, titleSlug, kind string, limit int) (domain.AdminDataQualityIssues, error)
	RegistryNamesRunner     func(ctx context.Context, titleSlug string, dryRun bool) (domain.RegistryNamesBackfillResult, error)
	ModeTranslationRunner   func(ctx context.Context, titleSlug, modeEN, nameFR string) (domain.ResolveResult, error)
	AssetTranslationRunner  func(ctx context.Context, titleSlug string, req domain.AssetTranslationRequest) (domain.ResolveResult, error)
	CatalogRefreshRunner    func(ctx context.Context, titleSlug string) (domain.CatalogRefreshResult, error)
)

// ErrActionBusyMessage : message du 409 single-flight (aligné registry).
const actionBusyCode = "already_running"

// AdminDataQualityHandler sert les endpoints qualité données.
type AdminDataQualityHandler struct {
	counts           DataQualityCountsRunner
	issues           DataQualityIssuesRunner
	registryNames    RegistryNamesRunner
	modeTranslation  ModeTranslationRunner
	assetTranslation AssetTranslationRunner
	catalogRefresh   CatalogRefreshRunner
	// busyErr : sentinelle ErrActionBusy du package api (injectée pour éviter
	// le cycle d'import handlers → api).
	busyErr error
}

// NewAdminDataQualityHandler construit le handler.
func NewAdminDataQualityHandler(
	counts DataQualityCountsRunner,
	issues DataQualityIssuesRunner,
	registryNames RegistryNamesRunner,
	modeTranslation ModeTranslationRunner,
	assetTranslation AssetTranslationRunner,
	catalogRefresh CatalogRefreshRunner,
	busyErr error,
) *AdminDataQualityHandler {
	return &AdminDataQualityHandler{
		counts: counts, issues: issues, registryNames: registryNames,
		modeTranslation: modeTranslation, assetTranslation: assetTranslation,
		catalogRefresh: catalogRefresh,
		busyErr:        busyErr,
	}
}

// RunCatalogRefresh seed les tables catalog depuis match_registry (zéro
// réseau — le drain DiscoveryUGC reste CLI-only).
// POST /admin/actions/catalog/refresh.
func (h *AdminDataQualityHandler) RunCatalogRefresh(w http.ResponseWriter, r *http.Request) {
	titleSlug := titleOrDefault(r)
	resp, err := h.catalogRefresh(r.Context(), titleSlug)
	if err != nil {
		if h.busyErr != nil && errors.Is(err, h.busyErr) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"code": actionBusyCode, "message": "Refresh catalogue déjà en cours.", "retryable": true,
			})
			return
		}
		slog.ErrorContext(r.Context(), "admin_data_quality: catalog refresh failed",
			"title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "catalog_refresh_unavailable",
			"Refresh catalogue indisponible (metadata ou shared inaccessibles).")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// validIssueKinds : kinds acceptés par GET .../issues.
var validIssueKinds = map[string]struct{}{
	"raw_uuids": {}, "untranslated_modes": {}, "orphan_playlists": {}, "orphan_xuids": {},
}

// GetCounts retourne les compteurs d'inconnus.
// GET /admin/monitoring/data-quality?title={slug}.
func (h *AdminDataQualityHandler) GetCounts(w http.ResponseWriter, r *http.Request) {
	titleSlug := titleOrDefault(r)
	resp, err := h.counts(r.Context(), titleSlug)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_data_quality: counts failed", "title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "data_quality_error",
			"Impossible de calculer les compteurs qualité données.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetIssues retourne la liste détaillée d'un kind d'inconnu.
// GET /admin/monitoring/data-quality/issues?title=&kind=&limit= (limit max 500).
func (h *AdminDataQualityHandler) GetIssues(w http.ResponseWriter, r *http.Request) {
	titleSlug := titleOrDefault(r)
	kind := r.URL.Query().Get("kind")
	if _, ok := validIssueKinds[kind]; !ok {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_kind",
			"kind doit être raw_uuids | untranslated_modes | orphan_playlists | orphan_xuids.")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	resp, err := h.issues(r.Context(), titleSlug, kind, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_data_quality: issues failed",
			"title", titleSlug, "kind", kind, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "data_quality_error",
			"Impossible de lister les inconnus.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// RunRegistryNamesBackfill exécute le backfill (ou le scan dry-run).
// POST /admin/actions/registry-names/backfill {dry_run}.
func (h *AdminDataQualityHandler) RunRegistryNamesBackfill(w http.ResponseWriter, r *http.Request) {
	var req domain.RegistryNamesBackfillRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req) // corps vide → run réel
	}
	titleSlug := titleOrDefault(r)
	resp, err := h.registryNames(r.Context(), titleSlug, req.DryRun)
	if err != nil {
		if h.busyErr != nil && errors.Is(err, h.busyErr) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"code": actionBusyCode, "message": "Backfill registry names déjà en cours.", "retryable": true,
			})
			return
		}
		slog.ErrorContext(r.Context(), "admin_data_quality: registry names backfill failed",
			"title", titleSlug, "dry_run", req.DryRun, "err", err)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "registry_names_unavailable",
			"Backfill indisponible (writer shared occupé ou metadata absente).")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ResolveModeTranslation écrit une traduction FR de mode.
// POST /admin/actions/translations/mode {mode_en, name_fr}.
func (h *AdminDataQualityHandler) ResolveModeTranslation(w http.ResponseWriter, r *http.Request) {
	var req domain.ModeTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps JSON invalide.")
		return
	}
	if !validResolveInput(req.ModeEN) || !validResolveInput(req.NameFR) {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_input",
			"mode_en et name_fr sont requis (1-128 caractères).")
		return
	}
	titleSlug := titleOrDefault(r)
	resp, err := h.modeTranslation(r.Context(), titleSlug, req.ModeEN, req.NameFR)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_data_quality: mode translation failed",
			"title", titleSlug, "mode_en", req.ModeEN, "err", err)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "translation_failed",
			"Écriture de la traduction impossible.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ResolveAssetTranslation écrit une traduction d'asset (en-US et/ou fr-FR).
// POST /admin/actions/translations/asset {asset_kind, asset_id, name_en?, name_fr?}.
func (h *AdminDataQualityHandler) ResolveAssetTranslation(w http.ResponseWriter, r *http.Request) {
	var req domain.AssetTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps JSON invalide.")
		return
	}
	if !validResolveInput(req.AssetID) || (req.NameEN == "" && req.NameFR == "") ||
		!optionalResolveInput(req.NameEN) || !optionalResolveInput(req.NameFR) {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_input",
			"asset_id requis + au moins un nom (1-128 caractères).")
		return
	}
	titleSlug := titleOrDefault(r)
	resp, err := h.assetTranslation(r.Context(), titleSlug, req)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_data_quality: asset translation failed",
			"title", titleSlug, "asset_kind", req.AssetKind, "asset_id", req.AssetID, "err", err)
		writeError(r.Context(), w, http.StatusBadRequest, "translation_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// validResolveInput : champ requis, borné (anti-payload abusif).
func validResolveInput(s string) bool { return len(s) >= 1 && len(s) <= 128 }

// optionalResolveInput : champ optionnel mais borné si fourni.
func optionalResolveInput(s string) bool { return len(s) <= 128 }
