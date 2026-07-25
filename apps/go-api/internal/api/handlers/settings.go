// Package handlers — settings.go : endpoints de configuration (Sprint 16).
//
// GET  /settings              → retourne la configuration courante
// PATCH /settings             → mise à jour partielle (403 demo_mode)
// POST /settings/media/reset-index → réinitialise l'index médias (job asynchrone)
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au point de montage
// racine (server.go monte ces routes en absolu, ex. r.Get("/settings", ...)) et
// enregistre les 7 routes via huma.*. Logique métier inchangée (settingsStore +
// jobStore + scheduler backup), seul le wrapping HTTP change. PATCH /settings et
// les POST media/reset-index ont un body décodé maison (RawBody + MarkRequestBody
// Optional) pour préserver le contrat 400 {invalid_body} sur JSON malformé / corps
// absent (un Body typé renverrait le 422 de validation Huma).
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/authz"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/pkg/duckdbbackup"
)

// SettingsHandler gère les endpoints de configuration.
type SettingsHandler struct {
	cfg                 *config.AppConfig
	settingsStore       *settings_platform.Store
	jobStore            *jobs.Store
	mediaIndexer        service.MediaIndexer
	friendsOrchestrator port.FriendsOrchestrator    // nil = recompute désactivé (mode legacy)
	notifierFor         NotificationsEmitterFactory // nil = pas de notifs friend_added
	backupSched         *duckdbbackup.Scheduler     // nil = backup désactivé
}

// NewSettingsHandler crée un SettingsHandler avec le DirMediaIndexer par défaut.
func NewSettingsHandler(cfg *config.AppConfig, settingsStore *settings_platform.Store, jobStore *jobs.Store) *SettingsHandler {
	return NewSettingsHandlerWithIndexer(cfg, settingsStore, jobStore, service.NewDirMediaIndexer())
}

// NewSettingsHandlerWithIndexer crée un SettingsHandler avec un MediaIndexer explicite.
// Utilisé en production et dans les tests (permet l'injection d'un mock).
func NewSettingsHandlerWithIndexer(
	cfg *config.AppConfig,
	settingsStore *settings_platform.Store,
	jobStore *jobs.Store,
	indexer service.MediaIndexer,
) *SettingsHandler {
	return &SettingsHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		jobStore:      jobStore,
		mediaIndexer:  indexer,
	}
}

// WithFriendsOrchestrator branche un FriendsOrchestrator déclenché sur diff
// friend_gamertags lors d'un PATCH /settings. §4 plan Squad/Sessions overhaul.
func (h *SettingsHandler) WithFriendsOrchestrator(o port.FriendsOrchestrator) *SettingsHandler {
	h.friendsOrchestrator = o
	return h
}

// WithNotificationsEmitter branche la factory d'émetteurs pour les notifications
// friend_added (§6 plan Squad/Sessions overhaul). Sans wiring, le PATCH
// fonctionne mais aucune notif n'est émise.
func (h *SettingsHandler) WithNotificationsEmitter(f NotificationsEmitterFactory) *SettingsHandler {
	h.notifierFor = f
	return h
}

// WithBackupScheduler branche le scheduler de backup DuckDB.
// Sans wiring, GetBackupStatus retourne {enabled:false} et PostBackupRun renvoie 503.
func (h *SettingsHandler) WithBackupScheduler(s *duckdbbackup.Scheduler) *SettingsHandler {
	h.backupSched = s
	return h
}

// Mount enregistre les 7 routes via Huma au point de montage racine (server.go
// monte ces chemins en absolu). Les POST media/reset-index et le PATCH /settings
// ont un body OPTIONNEL/décodé maison (RawBody + MarkRequestBodyOptional) :
// préserve le 400 {invalid_body} sur corps absent ou JSON malformé.
func (h *SettingsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/settings", h.handleGetSettings)
	huma.Patch(api, "/settings", h.handlePatchSettings)
	humacore.MarkRequestBodyOptional(api, http.MethodPatch, "/settings")
	huma.Post(api, "/settings/media/reset-index", h.handlePostMediaResetIndex)
	humacore.MarkRequestBodyOptional(api, http.MethodPost, "/settings/media/reset-index")
	huma.Post(api, "/settings/media/scan", h.handlePostMediaScan)
	huma.Post(api, "/settings/sessions/recalculate", h.handlePostRecalculateSessions)
	huma.Get(api, "/settings/backup/status", h.handleGetBackupStatus)
	huma.Post(api, "/settings/backup/run", h.handlePostBackupRun)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// settingsBodyInput : corps brut décodé maison (400 invalid_body sur JSON
// malformé ou corps absent — body marqué optionnel pour que le décodage maison
// reproduise le contrat EOF→400 de l'ancien json.NewDecoder).
type settingsBodyInput struct {
	RawBody []byte
}

// settingsJSONOutput : 200 avec un corps JSON quelconque (settings response,
// backup status/run). Reproduit writeJSON(w, 200, ...).
type settingsJSONOutput struct {
	Body any
}

// asyncJobOutput : 202 Accepted avec un AsyncJobStatus.
type asyncJobOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetSettings retourne la configuration courante.
// GET /settings
func (h *SettingsHandler) handleGetSettings(ctx context.Context, _ *struct{}) (*settingsJSONOutput, error) {
	if h.cfg.DemoMode {
		// Mode démo : renvoyer les settings RÉELS de la démo (app_settings.json
		// avec lang=fr) et non Defaults() — qui renverrait lang="en" et ferait
		// afficher la démo en anglais. Fallback Defaults+fr si lecture échoue.
		if h.settingsStore != nil {
			if cfg, err := h.settingsStore.Load(); err == nil {
				return &settingsJSONOutput{Body: h.settingsResponse(cfg)}, nil
			}
		}
		d := settings_platform.Defaults()
		d.Lang = "fr"
		return &settingsJSONOutput{Body: h.settingsResponse(d)}, nil
	}

	// PMT-4 PR-3c : résout l'overlay per-titre (ShowProgression / OutcomeExclude*).
	// Titre par défaut sans overlay ⇒ == Load() (byte-identique).
	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	cfg, err := h.settingsStore.ResolveForTitle(pr.TitleSettingsPath(ctxkeys.TitleSlug(ctx)))
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}
	return &settingsJSONOutput{Body: h.settingsResponse(cfg)}, nil
}

// settingsResponse construit la réponse GET/PATCH à partir d'un AppSettings, en
// écrasant media_delete_source_after_transcode par sa valeur EFFECTIVE résolue
// (config.ResolveMediaDeleteSource : env > store > isProd). Le *bool brut du store
// (nil = auto) n'est jamais exposé tel quel — le toggle UI reflète la réalité du
// runtime même quand le champ n'est pas défini.
func (h *SettingsHandler) settingsResponse(cfg *settings_platform.AppSettings) *domain.SettingsResponse {
	resp := settings_platform.ToResponse(cfg)
	resp.MediaDeleteSourceAfterTranscode = config.ResolveMediaDeleteSource(
		os.Getenv(config.EnvMediaDeleteSource),
		cfg.MediaDeleteSourceAfterTranscode,
		h.cfg.IsProduction(),
	)
	// Présence webhook Discord : résolue dans settings_platform.ToResponse (env > store,
	// source unique alignée sur l'envoi réel). Ne pas la ré-évaluer ici.
	return resp
}

// perTitleOverlayKeys — champs « valeur » per-titre (PMT-4 PR-3c) : un PATCH sur un
// titre NON défaut les écrit dans son overlay (data/titles/<slug>/settings.json),
// pas dans le global. Les autres champs (amis, verrou d'instance, langue, Discord,
// sync…) restent globaux. Liste explicite et bornée (cf. ResolveForTitle).
//
// extractPerTitleOverlay sort ces champs du req (les met à nil pour le Save global)
// et retourne leur projection JSON sparse pour l'overlay. Mutation in-place du req.
func extractPerTitleOverlay(req *domain.UpdateSettingsRequest) map[string]json.RawMessage {
	overlay := map[string]json.RawMessage{}
	if req.ShowProgression != nil {
		overlay["show_progression"], _ = json.Marshal(*req.ShowProgression)
		req.ShowProgression = nil
	}
	if req.OutcomeExcludeBotMatchesFromBadges != nil {
		overlay["outcome_exclude_bot_matches_from_badges"], _ = json.Marshal(*req.OutcomeExcludeBotMatchesFromBadges)
		req.OutcomeExcludeBotMatchesFromBadges = nil
	}
	if req.OutcomeExcludeBotMatchesFromRecords != nil {
		overlay["outcome_exclude_bot_matches_from_records"], _ = json.Marshal(*req.OutcomeExcludeBotMatchesFromRecords)
		req.OutcomeExcludeBotMatchesFromRecords = nil
	}
	return overlay
}

// handlePatchSettings met à jour partiellement la configuration.
// PATCH /settings — 422 en mode démo.
func (h *SettingsHandler) handlePatchSettings(ctx context.Context, in *settingsBodyInput) (*settingsJSONOutput, error) {
	if h.cfg.DemoMode {
		return nil, humacore.NewError(http.StatusUnprocessableEntity, "demo_mode_unsupported",
			"La modification des settings n'est pas disponible en mode démo.")
	}

	var req domain.UpdateSettingsRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}

	// Le verrou d'instance ne peut être basculé que par un admin : sinon un
	// utilisateur standard pourrait rouvrir une instance volontairement fermée.
	// No-op quand l'auth n'est pas appliquée (mode none/demo, single-user dev) :
	// l'opérateur passe alors par env / app_settings.json directement.
	if req.InstanceLocked != nil && authz.Enforced(h.cfg.DemoMode, h.cfg.AuthMode) {
		sess := middleware.GetSession(ctx)
		if sess == nil || sess.Role == nil || *sess.Role != "admin" {
			slog.WarnContext(ctx, "settings: toggle instance_locked refusé (admin requis)")
			return nil, humacore.NewError(http.StatusForbidden, "admin_required",
				"Seul un administrateur peut modifier le verrou d'instance.")
		}
		slog.InfoContext(ctx, "settings: verrou d'instance modifié", "locked", *req.InstanceLocked)
	}

	// Validation des champs analyse.
	if req.SessionGapMinutes != nil && *req.SessionGapMinutes < 0 {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_session_gap",
			"session_gap_minutes doit être ≥ 0.")
	}
	if req.SessionTeamChangeMode != nil {
		switch *req.SessionTeamChangeMode {
		case "ignore", "group", "friends":
			// valide
		default:
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_team_change_mode",
				"session_team_change_mode doit être \"ignore\", \"group\" ou \"friends\".")
		}
	}
	if req.OutcomeBadgeSensitivity != nil {
		switch *req.OutcomeBadgeSensitivity {
		case "relaxed", "standard", "strict":
			// valide
		default:
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_badge_sensitivity",
				"outcome_badge_sensitivity doit être \"relaxed\", \"standard\" ou \"strict\".")
		}
	}

	// PMT-4 PR-3c : pour un titre NON défaut, les champs « valeur » per-titre
	// (ShowProgression / OutcomeExclude*) sont retirés du req global et écrits dans
	// l'overlay du titre. Titre par défaut ⇒ perTitleOverlay vide, req inchangé,
	// chemin global byte-identique. Détection via le flag IsDefault (jamais une
	// comparaison de slug littérale — archlint no_slug_comparison).
	titleSlug := ctxkeys.TitleSlug(ctx)
	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	var perTitleOverlay map[string]json.RawMessage
	if desc := titlePkg.DefaultRegistry().Get(titleSlug); desc != nil && !desc.IsDefault {
		perTitleOverlay = extractPerTitleOverlay(&req)
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
	}

	// Snapshot friend_gamertags avant Apply pour détecter le diff post-save.
	prevFriends := append([]string(nil), cfg.FriendGamertags...)

	settings_platform.Apply(cfg, &req)

	if err := h.settingsStore.Save(cfg); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_save_error", "Impossible de sauvegarder la configuration.")
	}

	// Le rendement combat (OffensiveConversion) est un réglage global process :
	// propager immédiatement pour que les recomputes au query-time reflètent le
	// toggle sans redémarrage. Idempotent.
	if req.RendementExcludeAssists != nil {
		slog.InfoContext(ctx, "settings: rendement_exclude_assists modifié",
			"value", cfg.RendementExcludeAssists)
	}
	analysis.SetExcludeAssistsFromYield(cfg.RendementExcludeAssists)

	// §4 plan Squad/Sessions overhaul : si friend_gamertags a changé,
	// déclencher async le recompute is_with_friends sur toutes les player DBs.
	// Idempotent (la garde FALSE dans friends_recompute.go protège les retries).
	if h.friendsOrchestrator != nil && friendGamertagsChanged(prevFriends, cfg.FriendGamertags) {
		go func() {
			bgCtx := context.Background()
			if err := h.friendsOrchestrator.OnFriendsChanged(bgCtx); err != nil {
				slog.ErrorContext(bgCtx, "friends recompute orchestration failed", "err", err)
			}
		}()
	}

	// §6 plan Squad/Sessions overhaul : notif friend_added pour chaque nouveau
	// gamertag (set diff prev → next). Best-effort (warn log si l'émetteur
	// échoue, mais la réponse PATCH reste OK).
	if h.notifierFor != nil {
		added := newFriendsAdded(prevFriends, cfg.FriendGamertags)
		if len(added) > 0 {
			h.emitFriendsAdded(ctx, added)
		}
	}

	// PMT-4 PR-3c : persiste l'overlay per-titre (après le Save global réussi).
	if len(perTitleOverlay) > 0 {
		if err := h.settingsStore.SaveTitleOverlay(pr.TitleSettingsPath(titleSlug), perTitleOverlay); err != nil {
			return nil, humacore.NewError(http.StatusInternalServerError, "settings_overlay_save_error",
				"Impossible de sauvegarder l'overlay du titre.")
		}
	}

	// Réponse : settings RÉSOLUS pour le titre (global + overlay). Titre par défaut
	// sans overlay ⇒ == cfg global (byte-identique).
	resolved, rerr := h.settingsStore.ResolveForTitle(pr.TitleSettingsPath(titleSlug))
	if rerr != nil {
		resolved = cfg
	}
	return &settingsJSONOutput{Body: h.settingsResponse(resolved)}, nil
}

// newFriendsAdded retourne les gamertags présents dans next mais pas dans prev
// (set diff case-insensitive + trim, cohérent avec friendGamertagsChanged).
// Retourne les gamertags **dans la casse next** (préserve la saisie utilisateur).
func newFriendsAdded(prev, next []string) []string {
	prevSet := make(map[string]struct{}, len(prev))
	for _, gt := range prev {
		prevSet[normalizeGamertag(gt)] = struct{}{}
	}
	var added []string
	for _, gt := range next {
		if _, ok := prevSet[normalizeGamertag(gt)]; !ok {
			added = append(added, gt)
		}
	}
	return added
}

// emitFriendsAdded émet 1 notification friend_added (in-app) par nouveau
// gamertag, et déclenche le webhook Discord si activé. §6.A + §6.B.
// Best-effort sur les 2 canaux : warn log + continue, jamais bloquant.
func (h *SettingsHandler) emitFriendsAdded(ctx context.Context, added []string) {
	if h.notifierFor == nil {
		return
	}
	em, err := h.notifierFor(ctx, "")
	if err != nil || em == nil {
		slog.WarnContext(ctx, "notifications: friend_added emitter factory failed", "err", err)
		return
	}
	// Charger NotifyConfig une fois pour la batch (évite N reads disk).
	notifyCfg := notify.LoadNotifyConfig(h.cfg.AppSettingsPath)
	// PMT-11 : libellés Discord du titre courant (outcomes + footer). Failsafe Halo.
	notifyCfg.Labels = notify.LabelsForSlug(ctxkeys.TitleSlug(ctx))
	for _, gt := range added {
		if err := em.Emit(ctx, notifications.EmitInput{
			Category: notifications.CategoryFriendAdded,
			Severity: notifications.SeverityInfo,
			TitleKey: "notif.friend_added.title",
			BodyKey:  "notif.friend_added.body",
			Params:   map[string]any{"gamertag": gt},
			Source:   "settings_handler",
		}); err != nil {
			slog.WarnContext(ctx, "notifications: friend_added emit", "gamertag", gt, "err", err)
		}
		// §6.B Discord : webhook failsafe (no-op si webhook vide / NotifyFriends off).
		go notify.NotifyFriendAdded(notifyCfg, gt)
	}
}

// friendGamertagsChanged compare deux listes (set-equality, case-insensitive).
func friendGamertagsChanged(prev, next []string) bool {
	if len(prev) != len(next) {
		return true
	}
	set := make(map[string]struct{}, len(prev))
	for _, gt := range prev {
		set[normalizeGamertag(gt)] = struct{}{}
	}
	for _, gt := range next {
		if _, ok := set[normalizeGamertag(gt)]; !ok {
			return true
		}
	}
	return false
}

func normalizeGamertag(gt string) string {
	return strings.ToLower(strings.TrimSpace(gt))
}

// handlePostMediaResetIndex réinitialise l'index des médias (opération destructive).
// POST /settings/media/reset-index — retourne un AsyncJobStatus (202).
// confirm_destructive doit être true.
func (h *SettingsHandler) handlePostMediaResetIndex(ctx context.Context, in *settingsBodyInput) (*asyncJobOutput, error) {
	var req domain.MediaResetRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
	}

	if !req.ConfirmDestructive {
		return nil, humacore.NewError(http.StatusBadRequest, "confirmation_required",
			"confirm_destructive doit être true pour autoriser la réinitialisation de l'index.")
	}

	job := h.jobStore.Create(domain.JobTypeReindexMedia, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	// La copie doit être lue avant tout write concurrent.
	jobSnapshot := *job

	// Charger le dossier captures et la timezone configurés dans les settings.
	// timezone est REQUISE pour que parseCaptureTimeFromFilename extraie la
	// datetime des noms OBS/Xbox ; sans elle, capture_start_utc=NULL → 0 assoc.
	capturesBaseDir := ""
	timezone := ""
	var deleteSourceVal *bool
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBaseDir = cfg.MediaCapturesBaseDir
			timezone = cfg.UserTimezone
			deleteSourceVal = cfg.MediaDeleteSourceAfterTranscode
		}
	}
	// Politique de rétention du source résolue LIVE (env > store > isProd) : un
	// toggle UI prend effet sans redémarrage (le store est relu à chaque déclenchement).
	deleteSource := config.ResolveMediaDeleteSource(
		os.Getenv(config.EnvMediaDeleteSource), deleteSourceVal, h.cfg.IsProduction())

	// Le titre courant (X-LevelUp-Title / session) vit dans le ctx de la REQUÊTE,
	// mais le job tourne en async sur context.Background() → on capture le slug ICI
	// et on le ré-injecte dans le ctx du job. Sinon l'indexation retombe sur
	// halo_infinite (défaut ctxkeys.TitleSlug) et n'indexe JAMAIS les médias des
	// autres titres (ex. Halo 5). Cf. media_index_service (chemins résolus par titre).
	titleSlug := ctxkeys.TitleSlug(ctx)

	go func() {
		step := "Reset index médias en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		err := h.mediaIndexer.ResetAndReindex(
			ctxkeys.WithTitleSlug(context.Background(), titleSlug),
			h.cfg.RepoRoot,
			capturesBaseDir,
			timezone,
			req.ReindexAfterReset,
			deleteSource,
			h.jobStore,
			job.JobID,
		)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.Status = domain.JobStatusFailed
				j.Error = &domain.JobErrorDetail{
					Code:    "media_reset_error",
					Message: errMsg,
				}
			})
			return
		}

		done := "Index médias réinitialisé"
		pct := 100
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct
			j.CurrentStep = &done
		})
	}()

	return &asyncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}

// handlePostMediaScan lance une indexation non-destructive des médias pour tous les joueurs.
// POST /settings/media/scan — retourne un AsyncJobStatus (202). 422 en mode démo.
func (h *SettingsHandler) handlePostMediaScan(ctx context.Context, _ *struct{}) (*asyncJobOutput, error) {
	if h.cfg.DemoMode {
		return nil, humacore.NewError(http.StatusUnprocessableEntity, "demo_mode_unsupported",
			"Le scan des médias n'est pas disponible en mode démo.")
	}
	job := h.jobStore.Create(domain.JobTypeScanMedia, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	// Cf. PostMediaResetIndex : timezone REQUISE pour la regex filename.
	capturesBaseDir := ""
	timezone := ""
	var deleteSourceVal *bool
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBaseDir = cfg.MediaCapturesBaseDir
			timezone = cfg.UserTimezone
			deleteSourceVal = cfg.MediaDeleteSourceAfterTranscode
		}
	}
	// Politique de rétention résolue LIVE (env > store > isProd), cf. handlePostMediaResetIndex.
	deleteSource := config.ResolveMediaDeleteSource(
		os.Getenv(config.EnvMediaDeleteSource), deleteSourceVal, h.cfg.IsProduction())

	// Cf. handlePostMediaResetIndex : ré-injecter le titre courant dans le job async
	// (sinon halo_infinite par défaut → médias des autres titres jamais scannés).
	titleSlug := ctxkeys.TitleSlug(ctx)

	go func() {
		step := "Scan médias en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		err := h.mediaIndexer.ScanAllMedia(
			ctxkeys.WithTitleSlug(context.Background(), titleSlug),
			h.cfg.RepoRoot,
			capturesBaseDir,
			timezone,
			deleteSource,
			h.jobStore,
			job.JobID,
		)
		if err != nil {
			errMsg := err.Error()
			h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
				j.Status = domain.JobStatusFailed
				j.Error = &domain.JobErrorDetail{
					Code:    "media_scan_error",
					Message: errMsg,
				}
			})
			return
		}

		done := "Scan médias terminé"
		pct := 100
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct
			j.CurrentStep = &done
		})
	}()

	return &asyncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}

// sessionComputeOptionsFor résout les options de calcul de sessions d'un titre
// via l'overlay PMT-4 (SessionGapMinutes / split / team-change sont per-titre :
// le rythme de session dépend du titre). Overlay absent ⇒ valeurs globales
// byte-identiques ; store nil ou erreur ⇒ Defaults(). Extrait pour rester
// testable hors de la goroutine de PostRecalculateSessions.
func sessionComputeOptionsFor(store *settings_platform.Store, pr *titlePkg.PathResolver, titleSlug string) domain.SessionComputeOptions {
	cfg := settings_platform.Defaults()
	if store != nil {
		if resolved, err := store.ResolveForTitle(pr.TitleSettingsPath(titleSlug)); err == nil && resolved != nil {
			cfg = resolved
		}
	}
	gap := cfg.SessionGapMinutes
	if gap <= 0 {
		gap = 120
	}
	return domain.SessionComputeOptions{
		GapMinutes:          gap,
		SplitOnRankedChange: cfg.SessionSplitOnRankedChange,
		TeamChangeMode:      domain.TeamChangeMode(cfg.SessionTeamChangeMode),
		Mode:                domain.SessionModeContext,
	}
}

// handlePostRecalculateSessions lance un recalcul des sessions pour tous les joueurs.
// POST /settings/sessions/recalculate — retourne un AsyncJobStatus (202).
func (h *SettingsHandler) handlePostRecalculateSessions(ctx context.Context, _ *struct{}) (*asyncJobOutput, error) {
	// FriendGamertags reste GLOBAL (cross-titre, PMT-4) : les amis sont des
	// personnes transverses au titre + le grant d'accès famille ne doit pas
	// varier par jeu. Résolu une seule fois via Load(), jamais par overlay.
	globalCfg := settings_platform.Defaults()
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			globalCfg = cfg
		}
	}
	friendGamertags := globalCfg.FriendGamertags

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "players_load_error",
			"Impossible de charger les joueurs configurés.")
	}

	job := h.jobStore.Create(domain.JobTypeSessionsRecalc, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	pr := titlePkg.NewPathResolver(h.cfg.RepoRoot)
	sharedDBPath := config.SharedDBPath(h.cfg, "")

	go func() {
		step := "Recalcul des sessions en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		total := 0
		for _, p := range players {
			if p.XUID == "" {
				continue
			}
			// SessionGapMinutes / split / team-change sont per-titre (overlay
			// PMT-4 : le rythme de session dépend du titre). Overlay absent ⇒
			// valeurs globales byte-identiques.
			opts := sessionComputeOptionsFor(h.settingsStore, pr, p.TitleSlug)
			playerDBPath := config.PlayerDBPath(h.cfg, "", p.Gamertag)
			n, err := go_sync.RecalculatePlayerSessions(
				context.Background(), h.cfg.SharedProvider, playerDBPath, sharedDBPath, p.XUID,
				opts, friendGamertags,
			)
			if err != nil {
				slog.Warn("sessions recalculate: erreur joueur",
					"gamertag", p.Gamertag, "err", err)
				continue
			}
			total += n
		}

		done := fmt.Sprintf("Sessions recalculées (%d matchs mis à jour)", total)
		pct := 100
		h.jobStore.Update(job.JobID, func(j *domain.AsyncJobStatus) {
			j.Status = domain.JobStatusSucceeded
			j.ProgressPct = &pct
			j.CurrentStep = &done
		})
	}()

	return &asyncJobOutput{Status: http.StatusAccepted, Body: &jobSnapshot}, nil
}
