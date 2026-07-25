// Package handlers — admin_data_quality.go : endpoints qualité données du
// dashboard monitoring admin (compteurs + listes d'inconnus, et actions de
// résolution synchrones : backfill registry names, traductions metadata).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin hérités) et enregistre les 7 routes
// via huma.*. Logique métier inchangée (runners injectés), seul le wrapping HTTP
// change. Les deux GET portent Cache-Control:no-store via un champ header de sortie
// (préserve le contrat du middleware NoStore sans dépendre du point de montage).
//
// Routes (RequireAuth+RequireAdmin) :
//   - GET  /admin/monitoring/data-quality              : compteurs (NoStore)
//   - GET  /admin/monitoring/data-quality/issues       : listes (kind, limit) (NoStore)
//   - POST /admin/actions/registry-names/backfill      : {dry_run} (body optionnel)
//   - POST /admin/actions/translations/mode            : {mode_en, name_fr}
//   - POST /admin/actions/translations/asset           : {asset_kind, asset_id, name_en?, name_fr?}
//   - POST /admin/actions/catalog/refresh              : (sans corps)
//   - POST /admin/actions/lying-bits/reset             : {dry_run} (body optionnel)
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// Runners injectés (implémentés par ServiceRegistry).
type (
	DataQualityCountsRunner func(ctx context.Context, titleSlug, locale string) (domain.AdminDataQualityCounts, error)
	DataQualityIssuesRunner func(ctx context.Context, titleSlug, kind, locale string, limit, offset int) (domain.AdminDataQualityIssues, error)
	RegistryNamesRunner     func(ctx context.Context, titleSlug string, dryRun bool) (domain.RegistryNamesBackfillResult, error)
	ModeTranslationRunner   func(ctx context.Context, titleSlug, modeEN, nameFR string) (domain.ResolveResult, error)
	AssetTranslationRunner  func(ctx context.Context, titleSlug string, req domain.AssetTranslationRequest) (domain.ResolveResult, error)
	CatalogRefreshRunner    func(ctx context.Context, titleSlug string) (domain.CatalogRefreshResult, error)
	LyingBitsResetRunner    func(ctx context.Context, titleSlug string, dryRun bool) (domain.LyingBitsResetResult, error)
)

// ErrActionBusyMessage : message du 409 single-flight (aligné registry).
const actionBusyCode = "already_running"

// noStoreCacheControl : valeur posée sur les GET (équivalent middleware.NoStore).
const noStoreCacheControl = "no-store"

// AdminDataQualityHandler sert les endpoints qualité données.
type AdminDataQualityHandler struct {
	counts           DataQualityCountsRunner
	issues           DataQualityIssuesRunner
	registryNames    RegistryNamesRunner
	modeTranslation  ModeTranslationRunner
	assetTranslation AssetTranslationRunner
	catalogRefresh   CatalogRefreshRunner
	lyingBitsReset   LyingBitsResetRunner
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
	lyingBitsReset LyingBitsResetRunner,
	busyErr error,
) *AdminDataQualityHandler {
	return &AdminDataQualityHandler{
		counts: counts, issues: issues, registryNames: registryNames,
		modeTranslation: modeTranslation, assetTranslation: assetTranslation,
		catalogRefresh: catalogRefresh, lyingBitsReset: lyingBitsReset,
		busyErr: busyErr,
	}
}

// Mount enregistre les 7 routes via Huma sur le sous-routeur chi (préfixe /admin
// + middleware RequireAuth/RequireAdmin hérités). Les corps {dry_run} des actions
// backfill/reset sont OPTIONNELS (MarkRequestBodyOptional) — corps absent → run réel.
func (h *AdminDataQualityHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/monitoring/data-quality", h.handleGetCounts, humacore.Op(
		"getAdminMonitoringDataQuality",
		"Dashboard monitoring — compteurs d'inconnus data (assets UUID bruts, modes non traduits, playlists hors catalogue, xuids orphelins, lying bits) "+
			"(auth admin requis)",
		"admin"))
	huma.Get(api, "/monitoring/data-quality/issues", h.handleGetIssues, humacore.Op(
		"getAdminMonitoringDataQualityIssues",
		"Dashboard monitoring — listes détaillées des inconnus d'un kind (alimente les formulaires de résolution) (auth admin requis)",
		"admin"))
	huma.Post(api, "/actions/registry-names/backfill", h.handleRegistryNamesBackfill, humacore.Op(
		"postAdminActionRegistryNamesBackfill",
		"Action admin — résout les assets UUID bruts de match_registry via metadata.asset_translations (dry_run supporté, writer shared sérialisé "+
			"dblease) (auth admin requis)",
		"admin"))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/actions/registry-names/backfill")
	huma.Post(api, "/actions/translations/mode", h.handleResolveModeTranslation, humacore.Op(
		"postAdminActionModeTranslation",
		"Action admin — upsert mode_name_tr[fr] pour un mode normalisé (résout un mode non traduit, effet immédiat) (auth admin requis)",
		"admin"))
	huma.Post(api, "/actions/translations/asset", h.handleResolveAssetTranslation, humacore.Op(
		"postAdminActionAssetTranslation",
		"Action admin — upsert asset_translations (en-US et/ou fr-FR) pour un asset playlist/map/pair/game_variant (résolution effective des UUID inconnus) "+
			"(auth admin requis)",
		"admin"))
	huma.Post(api, "/actions/catalog/refresh", h.handleRunCatalogRefresh, humacore.Op(
		"postAdminActionCatalogRefresh",
		"Action admin — seed les tables catalog metadata (playlists/maps/pairs/variants) depuis match_registry, zéro réseau (le drain DiscoveryUGC reste "+
			"CLI-only) (auth admin requis)",
		"admin"))
	huma.Post(api, "/actions/lying-bits/reset", h.handleRunLyingBitsReset, humacore.Op(
		"postAdminActionLyingBitsReset",
		"Action admin — clear les bits backfill_completed menteurs de match_registry (events/weapons posés mais tables vides) + events_loaded menteur, "+
			"débloque le heal au prochain sync (dry_run supporté, writer shared sérialisé dblease) (auth admin requis)",
		"admin"))
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/actions/lying-bits/reset")
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// dqTitleInput : ?title= optionnel (fallback titre par défaut via titleOrDefaultSlug).
type dqTitleInput struct {
	Title string `query:"title"`
}

// dqCountsInput : ?title= + ?locale= (défaut « fr ») — la locale cible le compteur
// untranslated_modes (échotée pour un libellé front honnête).
type dqCountsInput struct {
	Title  string `query:"title"`
	Locale string `query:"locale"`
}

// dqIssuesInput : ?title=&kind=&locale=&limit=&offset= — limit/offset pris en
// STRING pour reproduire le contrat d'origine (valeur non numérique ou <=0
// ignorée : limit défaut 50 / clamp 500, offset défaut 0), PAS le 422 de
// validation Huma qu'un `int` produirait. offset rétrocompatible : absent → 0.
// locale défaut « fr » (paramètre ; on ne construit pas d'autre locale aujourd'hui).
type dqIssuesInput struct {
	Title  string `query:"title"`
	Kind   string `query:"kind"`
	Locale string `query:"locale"`
	Limit  string `query:"limit"`
	Offset string `query:"offset"`
}

// dqDryRunInput : ?title= + corps OPTIONNEL {dry_run} décodé maison (corps absent
// → run réel).
type dqDryRunInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// dqModeTranslationInput : ?title= + corps {mode_en, name_fr} (décodage maison →
// 400 invalid_body si JSON malformé).
type dqModeTranslationInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// dqAssetTranslationInput : ?title= + corps {asset_kind, asset_id, name_en?, name_fr?}.
type dqAssetTranslationInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// dqCountsOutput / dqIssuesOutput : corps domain + Cache-Control:no-store (GET).
type dqCountsOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         domain.AdminDataQualityCounts
}
type dqIssuesOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         domain.AdminDataQualityIssues
}

type dqRegistryNamesOutput struct {
	Body domain.RegistryNamesBackfillResult
}
type dqResolveOutput struct{ Body domain.ResolveResult }
type dqCatalogRefreshOutput struct{ Body domain.CatalogRefreshResult }
type dqLyingBitsOutput struct{ Body domain.LyingBitsResetResult }

// dqBusyError reproduit EXACTEMENT l'enveloppe 409 single-flight d'origine
// ({code, message, retryable:true}) — distincte de humacore.NewError qui force
// retryable=false sur un 409 (retryable dérivé de status>=500). Implémente
// huma.StatusError → Huma sérialise le struct tel quel via le format byte-identique.
type dqBusyError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *dqBusyError) Error() string  { return e.Message }
func (e *dqBusyError) GetStatus() int { return http.StatusConflict }

// busyResponse construit l'erreur 409 already_running (retryable:true).
func busyResponse(message string) huma.StatusError {
	return &dqBusyError{Code: actionBusyCode, Message: message, Retryable: true}
}

// noStoreError attache Cache-Control:no-store aux réponses d'erreur des GET
// (l'ancien middleware.NoStore le posait sur TOUTES les réponses, succès ET erreur).
func noStoreError(err huma.StatusError) error {
	return huma.ErrorWithHeaders(err, http.Header{"Cache-Control": []string{noStoreCacheControl}})
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// defaultDQLocale : locale de traduction par défaut (untranslated_modes).
const defaultDQLocale = "fr"

// normalizeDQLocale : trim + minuscule, défaut « fr » si vide. Le back n'accepte
// aujourd'hui que le paramétrage (pas de construction d'une autre locale).
func normalizeDQLocale(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return defaultDQLocale
	}
	return s
}

// handleGetCounts retourne les compteurs d'inconnus.
// GET /admin/monitoring/data-quality?title={slug}&locale={fr}.
func (h *AdminDataQualityHandler) handleGetCounts(ctx context.Context, in *dqCountsInput) (*dqCountsOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.counts(ctx, titleSlug, normalizeDQLocale(in.Locale))
	if err != nil {
		slog.ErrorContext(ctx, "admin_data_quality: counts failed", "title", titleSlug, "err", err)
		return nil, noStoreError(humacore.NewError(http.StatusInternalServerError, "data_quality_error",
			"Impossible de calculer les compteurs qualité données."))
	}
	return &dqCountsOutput{CacheControl: noStoreCacheControl, Body: resp}, nil
}

// validIssueKinds : kinds acceptés par GET .../issues.
var validIssueKinds = map[string]struct{}{
	"raw_uuids": {}, "untranslated_modes": {}, "orphan_playlists": {}, "orphan_xuids": {},
}

// handleGetIssues retourne la liste détaillée d'un kind d'inconnu.
// GET /admin/monitoring/data-quality/issues?title=&kind=&limit= (limit max 500).
func (h *AdminDataQualityHandler) handleGetIssues(ctx context.Context, in *dqIssuesInput) (*dqIssuesOutput, error) {
	if _, ok := validIssueKinds[in.Kind]; !ok {
		return nil, noStoreError(humacore.NewError(http.StatusBadRequest, "invalid_kind",
			"kind doit être raw_uuids | untranslated_modes | orphan_playlists | orphan_xuids."))
	}
	limit := 50
	if in.Limit != "" {
		if n, err := strconv.Atoi(in.Limit); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	offset := 0
	if in.Offset != "" {
		if o, err := strconv.Atoi(in.Offset); err == nil && o > 0 {
			offset = o
		}
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.issues(ctx, titleSlug, in.Kind, normalizeDQLocale(in.Locale), limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "admin_data_quality: issues failed",
			"title", titleSlug, "kind", in.Kind, "err", err)
		return nil, noStoreError(humacore.NewError(http.StatusInternalServerError, "data_quality_error",
			"Impossible de lister les inconnus."))
	}
	return &dqIssuesOutput{CacheControl: noStoreCacheControl, Body: resp}, nil
}

// handleRegistryNamesBackfill exécute le backfill (ou le scan dry-run).
// POST /admin/actions/registry-names/backfill {dry_run} (corps optionnel → run réel).
func (h *AdminDataQualityHandler) handleRegistryNamesBackfill(ctx context.Context, in *dqDryRunInput) (*dqRegistryNamesOutput, error) {
	var req domain.RegistryNamesBackfillRequest
	if len(in.RawBody) > 0 {
		_ = json.Unmarshal(in.RawBody, &req) // corps vide → run réel
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.registryNames(ctx, titleSlug, req.DryRun)
	if err != nil {
		if h.busyErr != nil && errors.Is(err, h.busyErr) {
			return nil, busyResponse("Backfill registry names déjà en cours.")
		}
		slog.ErrorContext(ctx, "admin_data_quality: registry names backfill failed",
			"title", titleSlug, "dry_run", req.DryRun, "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "registry_names_unavailable",
			"Backfill indisponible (writer shared occupé ou metadata absente).")
	}
	return &dqRegistryNamesOutput{Body: resp}, nil
}

// handleRunLyingBitsReset clear les bits backfill_completed menteurs de
// match_registry (events/weapons posés mais tables vides) + events_loaded
// menteur. POST /admin/actions/lying-bits/reset {dry_run} (corps optionnel → run réel).
func (h *AdminDataQualityHandler) handleRunLyingBitsReset(ctx context.Context, in *dqDryRunInput) (*dqLyingBitsOutput, error) {
	var req domain.LyingBitsResetRequest
	if len(in.RawBody) > 0 {
		_ = json.Unmarshal(in.RawBody, &req) // corps vide → run réel
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.lyingBitsReset(ctx, titleSlug, req.DryRun)
	if err != nil {
		if h.busyErr != nil && errors.Is(err, h.busyErr) {
			return nil, busyResponse("Reset des bits menteurs déjà en cours.")
		}
		slog.ErrorContext(ctx, "admin_data_quality: lying bits reset failed",
			"title", titleSlug, "dry_run", req.DryRun, "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "lying_bits_reset_unavailable",
			"Reset indisponible (writer shared occupé ou shared absente).")
	}
	return &dqLyingBitsOutput{Body: resp}, nil
}

// handleRunCatalogRefresh seed les tables catalog depuis match_registry (zéro
// réseau — le drain DiscoveryUGC reste CLI-only).
// POST /admin/actions/catalog/refresh.
func (h *AdminDataQualityHandler) handleRunCatalogRefresh(ctx context.Context, in *dqTitleInput) (*dqCatalogRefreshOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.catalogRefresh(ctx, titleSlug)
	if err != nil {
		if h.busyErr != nil && errors.Is(err, h.busyErr) {
			return nil, busyResponse("Refresh catalogue déjà en cours.")
		}
		slog.ErrorContext(ctx, "admin_data_quality: catalog refresh failed",
			"title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "catalog_refresh_unavailable",
			"Refresh catalogue indisponible (metadata ou shared inaccessibles).")
	}
	return &dqCatalogRefreshOutput{Body: resp}, nil
}

// handleResolveModeTranslation écrit une traduction FR de mode.
// POST /admin/actions/translations/mode {mode_en, name_fr}.
func (h *AdminDataQualityHandler) handleResolveModeTranslation(ctx context.Context, in *dqModeTranslationInput) (*dqResolveOutput, error) {
	var req domain.ModeTranslationRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps JSON invalide.")
	}
	if !validResolveInput(req.ModeEN) || !validResolveInput(req.NameFR) {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input",
			"mode_en et name_fr sont requis (1-128 caractères).")
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.modeTranslation(ctx, titleSlug, req.ModeEN, req.NameFR)
	if err != nil {
		slog.ErrorContext(ctx, "admin_data_quality: mode translation failed",
			"title", titleSlug, "mode_en", req.ModeEN, "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "translation_failed",
			"Écriture de la traduction impossible.")
	}
	return &dqResolveOutput{Body: resp}, nil
}

// handleResolveAssetTranslation écrit une traduction d'asset (en-US et/ou fr-FR).
// POST /admin/actions/translations/asset {asset_kind, asset_id, name_en?, name_fr?}.
func (h *AdminDataQualityHandler) handleResolveAssetTranslation(ctx context.Context, in *dqAssetTranslationInput) (*dqResolveOutput, error) {
	var req domain.AssetTranslationRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps JSON invalide.")
	}
	if !validResolveInput(req.AssetID) || (req.NameEN == "" && req.NameFR == "") ||
		!optionalResolveInput(req.NameEN) || !optionalResolveInput(req.NameFR) {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input",
			"asset_id requis + au moins un nom (1-128 caractères).")
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.assetTranslation(ctx, titleSlug, req)
	if err != nil {
		slog.ErrorContext(ctx, "admin_data_quality: asset translation failed",
			"title", titleSlug, "asset_kind", req.AssetKind, "asset_id", req.AssetID, "err", err)
		return nil, humacore.NewError(http.StatusBadRequest, "translation_failed", err.Error())
	}
	return &dqResolveOutput{Body: resp}, nil
}

// validResolveInput : champ requis, borné (anti-payload abusif).
func validResolveInput(s string) bool { return len(s) >= 1 && len(s) <= 128 }

// optionalResolveInput : champ optionnel mais borné si fourni.
func optionalResolveInput(s string) bool { return len(s) <= 128 }
