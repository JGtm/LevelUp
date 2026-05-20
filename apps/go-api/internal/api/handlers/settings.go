// Package handlers — settings.go : endpoints de configuration (Sprint 16).
//
// GET  /settings              → retourne la configuration courante
// PATCH /settings             → mise à jour partielle (403 demo_mode)
// POST /settings/media/reset-index → réinitialise l'index médias (job asynchrone)
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notify"
	"levelup/go-api/internal/platform/jobs"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	go_sync "levelup/go-api/internal/sync"
)

// SettingsHandler gère les endpoints de configuration.
type SettingsHandler struct {
	cfg                 *config.AppConfig
	settingsStore       *settings_platform.Store
	jobStore            *jobs.Store
	mediaIndexer        service.MediaIndexer
	friendsOrchestrator port.FriendsOrchestrator    // nil = recompute désactivé (mode legacy)
	notifierFor         NotificationsEmitterFactory // nil = pas de notifs friend_added
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

// GetSettings retourne la configuration courante.
// GET /settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DemoMode {
		// Mode démo : retourner les valeurs par défaut sans lecture disque
		defaults := settings_platform.ToResponse(settings_platform.Defaults())
		writeJSON(w, http.StatusOK, defaults)
		return
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}
	writeJSON(w, http.StatusOK, settings_platform.ToResponse(cfg))
}

// PatchSettings met à jour partiellement la configuration.
// PATCH /settings — 422 en mode démo.
func (h *SettingsHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	if h.cfg.DemoMode {
		writeError(r.Context(), w, http.StatusUnprocessableEntity, "demo_mode_unsupported",
			"La modification des settings n'est pas disponible en mode démo.")
		return
	}

	var req domain.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	// Validation des champs analyse.
	if req.SessionGapMinutes != nil && *req.SessionGapMinutes < 0 {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_session_gap",
			"session_gap_minutes doit être ≥ 0.")
		return
	}
	if req.SessionTeamChangeMode != nil {
		switch *req.SessionTeamChangeMode {
		case "ignore", "group", "friends":
			// valide
		default:
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_team_change_mode",
				"session_team_change_mode doit être \"ignore\", \"group\" ou \"friends\".")
			return
		}
	}
	if req.OutcomeBadgeSensitivity != nil {
		switch *req.OutcomeBadgeSensitivity {
		case "relaxed", "standard", "strict":
			// valide
		default:
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_badge_sensitivity",
				"outcome_badge_sensitivity doit être \"relaxed\", \"standard\" ou \"strict\".")
			return
		}
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_load_error", "Impossible de charger la configuration.")
		return
	}

	// Snapshot friend_gamertags avant Apply pour détecter le diff post-save.
	prevFriends := append([]string(nil), cfg.FriendGamertags...)

	settings_platform.Apply(cfg, &req)

	if err := h.settingsStore.Save(cfg); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_save_error", "Impossible de sauvegarder la configuration.")
		return
	}

	// §4 plan Squad/Sessions overhaul : si friend_gamertags a changé,
	// déclencher async le recompute is_with_friends sur toutes les player DBs.
	// Idempotent (la garde FALSE dans friends_recompute.go protège les retries).
	if h.friendsOrchestrator != nil && friendGamertagsChanged(prevFriends, cfg.FriendGamertags) {
		go func() {
			ctx := context.Background()
			if err := h.friendsOrchestrator.OnFriendsChanged(ctx); err != nil {
				slog.ErrorContext(ctx, "friends recompute orchestration failed", "err", err)
			}
		}()
	}

	// §6 plan Squad/Sessions overhaul : notif friend_added pour chaque nouveau
	// gamertag (set diff prev → next). Best-effort (warn log si l'émetteur
	// échoue, mais la réponse PATCH reste OK).
	if h.notifierFor != nil {
		added := newFriendsAdded(prevFriends, cfg.FriendGamertags)
		if len(added) > 0 {
			h.emitFriendsAdded(r.Context(), added)
		}
	}

	writeJSON(w, http.StatusOK, settings_platform.ToResponse(cfg))
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

// PostMediaResetIndex réinitialise l'index des médias (opération destructive).
// POST /settings/media/reset-index — retourne un AsyncJobStatus (202).
// confirm_destructive doit être true.
func (h *SettingsHandler) PostMediaResetIndex(w http.ResponseWriter, r *http.Request) {
	var req domain.MediaResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "Corps de requête JSON invalide.")
		return
	}

	if !req.ConfirmDestructive {
		writeError(r.Context(), w, http.StatusBadRequest, "confirmation_required",
			"confirm_destructive doit être true pour autoriser la réinitialisation de l'index.")
		return
	}

	job := h.jobStore.Create(domain.JobTypeReindexMedia, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	// La copie doit être lue avant tout write concurrent.
	jobSnapshot := *job

	// Charger le dossier captures configuré dans les settings.
	capturesBaseDir := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBaseDir = cfg.MediaCapturesBaseDir
		}
	}

	go func() {
		step := "Reset index médias en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		err := h.mediaIndexer.ResetAndReindex(
			context.Background(),
			h.cfg.RepoRoot,
			capturesBaseDir,
			req.ReindexAfterReset,
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

	writeJSON(w, http.StatusAccepted, &jobSnapshot)
}

// PostMediaScan lance une indexation non-destructive des médias pour tous les joueurs.
// POST /settings/media/scan — retourne un AsyncJobStatus (202).
func (h *SettingsHandler) PostMediaScan(w http.ResponseWriter, r *http.Request) {
	job := h.jobStore.Create(domain.JobTypeScanMedia, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	capturesBaseDir := ""
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			capturesBaseDir = cfg.MediaCapturesBaseDir
		}
	}

	go func() {
		step := "Scan médias en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		err := h.mediaIndexer.ScanAllMedia(
			context.Background(),
			h.cfg.RepoRoot,
			capturesBaseDir,
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

	writeJSON(w, http.StatusAccepted, &jobSnapshot)
}

// PostRecalculateSessions lance un recalcul des sessions pour tous les joueurs.
// POST /settings/sessions/recalculate — retourne un AsyncJobStatus (202).
func (h *SettingsHandler) PostRecalculateSessions(w http.ResponseWriter, r *http.Request) {
	settingsCfg := settings_platform.Defaults()
	if h.settingsStore != nil {
		if cfg, err := h.settingsStore.Load(); err == nil {
			settingsCfg = cfg
		}
	}

	players, err := h.cfg.LoadPlayers()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "players_load_error",
			"Impossible de charger les joueurs configurés.")
		return
	}

	job := h.jobStore.Create(domain.JobTypeSessionsRecalc, "")
	// Snapshot avant le go func() : la goroutine modifie in-place le job dans le store.
	jobSnapshot := *job

	gapMinutes := settingsCfg.SessionGapMinutes
	if gapMinutes <= 0 {
		gapMinutes = 120
	}
	opts := domain.SessionComputeOptions{
		GapMinutes:          gapMinutes,
		SplitOnRankedChange: settingsCfg.SessionSplitOnRankedChange,
		TeamChangeMode:      domain.TeamChangeMode(settingsCfg.SessionTeamChangeMode),
		Mode:                domain.SessionModeContext,
	}
	friendGamertags := settingsCfg.FriendGamertags
	sharedDBPath := config.SharedDBPath(h.cfg, "")

	go func() {
		step := "Recalcul des sessions en cours"
		h.jobStore.SetStatus(job.JobID, domain.JobStatusRunning, &step)

		total := 0
		for _, p := range players {
			if p.XUID == "" {
				continue
			}
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

	writeJSON(w, http.StatusAccepted, &jobSnapshot)
}
