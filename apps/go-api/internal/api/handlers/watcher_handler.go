// Package handlers — watcher_handler.go : endpoints de contrôle du watcher de présence Xbox.
//
// GET  /api/v1/watcher/status          → état courant (token + daemon + joueurs)
// POST /api/v1/watcher/auth/start      → démarre un Device Code Flow pour obtenir un token XSTS
// GET  /api/v1/watcher/auth/{id}       → poll état de la tentative
// PATCH /api/v1/watcher/subscriptions  → met à jour les joueurs surveillés
//
// Tous les endpoints nécessitent RequireAuth + RequireAdmin.
// Le daemon peut être nil (watcher désactivé) — géré proprement sans panic.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/watcher"
)

// WatcherHandler gère les endpoints de contrôle du watcher de présence.
type WatcherHandler struct {
	cfg           *config.AppConfig
	settingsStore *settings_platform.Store
	daemon        watcher.DaemonController // peut être nil
	tokenProvider auth_platform.TokenProvider
	attempts      *auth_platform.WatcherAttemptStore
	tokenStore    *auth_platform.TokenStore
}

// NewWatcherHandler crée un WatcherHandler.
// daemon peut être nil si le watcher n'est pas actif au démarrage.
func NewWatcherHandler(
	cfg *config.AppConfig,
	settingsStore *settings_platform.Store,
	daemon watcher.DaemonController,
	tokenProvider auth_platform.TokenProvider,
	attempts *auth_platform.WatcherAttemptStore,
) *WatcherHandler {
	tokenStorePath := title.NewPathResolver(cfg.RepoRoot).WatcherTokensPath()
	return &WatcherHandler{
		cfg:           cfg,
		settingsStore: settingsStore,
		daemon:        daemon,
		tokenProvider: tokenProvider,
		attempts:      attempts,
		tokenStore:    auth_platform.NewTokenStore(tokenStorePath),
	}
}

// watcherStatusResponse est la réponse de GET /watcher/status.
type watcherStatusResponse struct {
	DaemonRunning     bool                           `json:"daemon_running"`
	RTAConnected      bool                           `json:"rta_connected"`
	TokenValid        bool                           `json:"token_valid"`
	TokenExpiresAt    string                         `json:"token_expires_at,omitempty"`
	TokenGamertag     string                         `json:"token_gamertag,omitempty"`
	SubscribedPlayers []string                       `json:"subscribed_players"`
	Players           []watcher.PlayerPresenceStatus `json:"players"`
	LastEventAt       string                         `json:"last_event_at,omitempty"` // RFC3339 UTC du dernier event reçu (tous joueurs), vivacité du daemon
}

// watcherAuthStartResponse est la réponse de POST /watcher/auth/start.
type watcherAuthStartResponse struct {
	AttemptID       string `json:"attempt_id"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
}

// watcherAuthStatusResponse est la réponse de GET /watcher/auth/{id}.
type watcherAuthStatusResponse struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code,omitempty"`
	Gamertag  string `json:"gamertag,omitempty"`
	XUID      string `json:"xuid,omitempty"`
}

// watcherSubscriptionsRequest est le corps de PATCH /watcher/subscriptions.
type watcherSubscriptionsRequest struct {
	SubscribedPlayers []string `json:"subscribed_players"`
}

// GetStatus retourne l'état courant du watcher.
// GET /api/v1/watcher/status
func (h *WatcherHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	slog.Debug("watcher_handler: GetStatus appelé")
	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_error", "impossible de lire les settings")
		return
	}

	resp := watcherStatusResponse{
		SubscribedPlayers: cfg.WatcherSubscribedPlayers,
		Players:           []watcher.PlayerPresenceStatus{},
	}
	if len(resp.SubscribedPlayers) == 0 {
		resp.SubscribedPlayers = []string{"all"}
	}

	// Lire les tokens pour la validité XSTS
	tokens, err := h.tokenStore.Load()
	if err == nil {
		resp.TokenValid = tokens.IsXSTSValid(5 * time.Minute)
		if tokens.XSTSExpiresAt != (time.Time{}) {
			resp.TokenExpiresAt = tokens.XSTSExpiresAt.UTC().Format(time.RFC3339)
		}
		resp.TokenGamertag = tokens.XSTSGamertag
	}

	// État du daemon
	if h.daemon != nil {
		status := h.daemon.GetStatus()
		resp.DaemonRunning = status.Running
		resp.RTAConnected = status.RTAConnected
		resp.LastEventAt = status.LastEventAt
		if status.Players != nil {
			resp.Players = status.Players
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// StartAuth démarre un Device Code Flow Microsoft pour obtenir un token XSTS watcher.
// POST /api/v1/watcher/auth/start
func (h *WatcherHandler) StartAuth(w http.ResponseWriter, r *http.Request) {
	slog.Info("watcher_handler: démarrage Device Code Flow watcher")
	attempt, isNew := h.attempts.GetOrCreate()
	if !isNew {
		writeJSON(w, http.StatusOK, watcherAuthStartResponse{
			AttemptID:       attempt.AttemptID,
			UserCode:        attempt.UserCode,
			VerificationURL: attempt.VerificationURI,
			ExpiresIn:       attempt.ExpiresInSec,
		})
		return
	}

	flow, err := h.tokenProvider.InitDeviceFlow(r.Context())
	if err != nil {
		h.attempts.Update(attempt.AttemptID, func(a *auth_platform.WatcherAttempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "msal_init_error"
			a.ErrorDetail = err.Error()
		})
		writeError(r.Context(), w, http.StatusInternalServerError, "msal_init_error", "impossible de démarrer le Device Code Flow")
		return
	}

	h.attempts.Update(attempt.AttemptID, func(a *auth_platform.WatcherAttempt) {
		a.UserCode = flow.GetUserCode()
		a.VerificationURI = flow.GetVerificationURL()
		a.ExpiresInSec = flow.GetExpiresIn()
		a.DevFlow = flow
	})

	// Polling MSAL + acquisition XSTS en arrière-plan
	go h.pollWatcherAuth(attempt.AttemptID, flow, r.Context())

	snap := h.attempts.Snapshot(attempt.AttemptID)
	writeJSON(w, http.StatusOK, watcherAuthStartResponse{
		AttemptID:       snap.AttemptID,
		UserCode:        snap.UserCode,
		VerificationURL: snap.VerificationURI,
		ExpiresIn:       snap.ExpiresInSec,
	})
}

// GetAuthStatus retourne l'état d'une tentative d'auth watcher.
// GET /api/v1/watcher/auth/{attempt_id}
func (h *WatcherHandler) GetAuthStatus(w http.ResponseWriter, r *http.Request) {
	attemptID := chi.URLParam(r, "attempt_id")
	slog.Debug("watcher_handler: GetAuthStatus", "attempt_id", attemptID)
	snap := h.attempts.Snapshot(attemptID)
	if snap == nil {
		writeError(r.Context(), w, http.StatusNotFound, "attempt_not_found", "tentative introuvable")
		return
	}

	writeJSON(w, http.StatusOK, watcherAuthStatusResponse{
		Status:    snap.Status,
		ErrorCode: snap.ErrorCode,
		Gamertag:  snap.Gamertag,
		XUID:      snap.XUID,
	})
}

// PatchSubscriptions met à jour les joueurs surveillés.
// PATCH /api/v1/watcher/subscriptions
func (h *WatcherHandler) PatchSubscriptions(w http.ResponseWriter, r *http.Request) {
	var req watcherSubscriptionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_error", "impossible de lire les settings")
		return
	}

	players := req.SubscribedPlayers
	if len(players) == 0 {
		players = []string{"all"}
	}
	cfg.WatcherSubscribedPlayers = players

	if err := h.settingsStore.Save(cfg); err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "settings_save_error", "impossible de sauvegarder les settings")
		return
	}

	// Mettre à jour le daemon si actif
	if h.daemon != nil && h.daemon.IsRunning() {
		h.daemon.UpdateSubscriptions(players)
	}

	slog.Info("watcher_handler: subscriptions mises à jour", "players", players)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"subscribed_players": players,
	})
}

// pollWatcherAuth attend la validation du Device Code Flow puis acquiert le token XSTS.
func (h *WatcherHandler) pollWatcherAuth(attemptID string, flow auth_platform.DeviceFlow, parentCtx context.Context) {
	// Utiliser un contexte indépendant pour que l'auth survive à la requête HTTP
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flow.GetExpiresIn())*time.Second)
	defer cancel()

	_ = parentCtx // ignoré intentionnellement

	slog.Info("watcher_handler: début poll Device Code Flow", "attempt_id", attemptID)

	result, err := flow.AcquireToken(ctx)
	if err != nil {
		h.attempts.Update(attemptID, func(a *auth_platform.WatcherAttempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = errCodeMSALAcquire
			a.ErrorDetail = err.Error()
		})
		slog.Warn("watcher_handler: Device Code Flow échoué", "attempt_id", attemptID, "err", err)
		return
	}

	slog.Info("watcher_handler: Device Code Flow validé, acquisition XSTS…", "attempt_id", attemptID)

	// Acquisition XSTS Xbox Live
	xsts, err := auth_platform.AcquireXSTSForRTA(ctx, result)
	if err != nil {
		h.attempts.Update(attemptID, func(a *auth_platform.WatcherAttempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "xsts_error"
			a.ErrorDetail = err.Error()
		})
		slog.Warn("watcher_handler: XSTS acquisition échouée", "attempt_id", attemptID, "err", err)
		return
	}

	// Sauvegarder dans le token store
	tokens, _ := h.tokenStore.Load()
	if tokens == nil {
		tokens = &auth_platform.StoredTokens{}
	}
	tokens.AccessToken = result
	tokens.XSTSToken = xsts.Token
	tokens.XSTSUserHash = xsts.UserHash
	tokens.XSTSGamertag = xsts.Gamertag
	tokens.XSTSXUID = xsts.XUID
	tokens.XSTSExpiresAt = time.Now().Add(24 * time.Hour) // durée standard XSTS Xbox Live

	if err := h.tokenStore.Save(tokens); err != nil {
		slog.Error("watcher_handler: impossible de sauvegarder les tokens XSTS", "err", err)
	}

	// Mettre à jour le daemon si actif
	if h.daemon != nil {
		h.daemon.UpdateAuth(xsts.AuthHeader())
	}

	h.attempts.Update(attemptID, func(a *auth_platform.WatcherAttempt) {
		a.Status = "authorized"
		a.Gamertag = xsts.Gamertag
		a.XUID = xsts.XUID
		a.XSTSToken = xsts.Token
		a.XSTSUserHash = xsts.UserHash
	})

	slog.Info("watcher_handler: auth watcher complète",
		"attempt_id", attemptID,
		"gamertag", xsts.Gamertag,
	)
}
