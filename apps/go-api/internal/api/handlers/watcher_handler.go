// Package handlers — watcher_handler.go : endpoints de contrôle du watcher de présence Xbox.
//
// GET  /api/v1/watcher/status          → état courant (token + daemon + joueurs)
// POST /api/v1/watcher/auth/start      → démarre un Device Code Flow pour obtenir un token XSTS
// GET  /api/v1/watcher/auth/{id}       → poll état de la tentative
// PATCH /api/v1/watcher/subscriptions  → met à jour les joueurs surveillés
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /watcher (middleware RequireAuth/RequireAdmin hérités) et enregistre les 4 routes
// (status, auth/{attempt_id}, auth/start, subscriptions). Logique métier inchangée,
// seul le wrapping HTTP change. POST /auth/start déclenche une goroutine de polling
// MSAL ; son Input est vide (aucun corps lu).
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

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre via Huma les 4 routes du watcher sur le sous-routeur chi
// (préfixe /watcher + middleware RequireAuth/RequireAdmin hérités). Le body
// PATCH /subscriptions est OPTIONNEL (MarkRequestBodyOptional) — corps absent →
// défaut ["all"], comme l'inline d'origine. POST /auth/start ne lit aucun corps
// (Input *struct{}) → pas de MarkRequestBodyOptional (corps absent toléré).
func (h *WatcherHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/status", h.handleGetStatus, humacore.Op("getWatcherStatus", "Statut watcher (provider, token expiry, last_seen)", "auth"))
	huma.Get(api, "/auth/{attempt_id}", h.handleGetAuthStatus, humacore.Op("getWatcherAuthStatus", "Statut d'une tentative d'authentification watcher", "auth"))
	huma.Post(api, "/auth/start", h.handleStartAuth, humacore.Op("postWatcherAuthStart", "Démarre le flow d'auth watcher", "auth"))
	huma.Patch(api, "/subscriptions", h.handlePatchSubscriptions, humacore.Op("patchWatcherSubscriptions", "Mise à jour des abonnements watcher", "auth"))
	humacore.MarkRequestBodyOptional(api, http.MethodPatch, "/subscriptions")
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// watcherAuthStatusInput : path param {attempt_id}.
type watcherAuthStatusInput struct {
	AttemptID string `path:"attempt_id"`
}

// watcherSubscriptionsInput : corps brut décodé maison (400 invalid_body si JSON
// malformé, contrat préservé). Body OPTIONNEL via MarkRequestBodyOptional —
// corps absent → req.SubscribedPlayers nil → défaut ["all"].
type watcherSubscriptionsInput struct {
	RawBody []byte
}

type watcherStatusOutput struct{ Body watcherStatusResponse }
type watcherAuthStartOutput struct{ Body watcherAuthStartResponse }
type watcherAuthStatusOutput struct{ Body watcherAuthStatusResponse }
type watcherSubscriptionsOutput struct {
	Body struct {
		SubscribedPlayers []string `json:"subscribed_players"`
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

// handleGetStatus retourne l'état courant du watcher.
// GET /api/v1/watcher/status
func (h *WatcherHandler) handleGetStatus(ctx context.Context, _ *struct{}) (*watcherStatusOutput, error) {
	slog.Debug("watcher_handler: GetStatus appelé")
	cfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_error", "impossible de lire les settings")
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

	return &watcherStatusOutput{Body: resp}, nil
}

// handleStartAuth démarre un Device Code Flow Microsoft pour obtenir un token XSTS
// watcher. POST /api/v1/watcher/auth/start (migré Huma — aucun corps lu, Input vide).
func (h *WatcherHandler) handleStartAuth(ctx context.Context, _ *struct{}) (*watcherAuthStartOutput, error) {
	slog.Info("watcher_handler: démarrage Device Code Flow watcher")
	attempt, isNew := h.attempts.GetOrCreate()
	if !isNew {
		return &watcherAuthStartOutput{Body: watcherAuthStartResponse{
			AttemptID:       attempt.AttemptID,
			UserCode:        attempt.UserCode,
			VerificationURL: attempt.VerificationURI,
			ExpiresIn:       attempt.ExpiresInSec,
		}}, nil
	}

	flow, err := h.tokenProvider.InitDeviceFlow(ctx)
	if err != nil {
		h.attempts.Update(attempt.AttemptID, func(a *auth_platform.WatcherAttempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "device_flow_init_error"
			a.ErrorDetail = err.Error()
		})
		return nil, humacore.NewError(http.StatusInternalServerError, "device_flow_init_error", "impossible de démarrer le Device Code Flow")
	}

	h.attempts.Update(attempt.AttemptID, func(a *auth_platform.WatcherAttempt) {
		a.UserCode = flow.GetUserCode()
		a.VerificationURI = flow.GetVerificationURL()
		a.ExpiresInSec = flow.GetExpiresIn()
		a.DevFlow = flow
	})

	// Polling MSAL + acquisition XSTS en arrière-plan. pollWatcherAuth ignore son
	// parentCtx et crée son propre contexte avec timeout → passer ctx (Huma) est inerte.
	go h.pollWatcherAuth(attempt.AttemptID, flow, ctx)

	snap := h.attempts.Snapshot(attempt.AttemptID)
	return &watcherAuthStartOutput{Body: watcherAuthStartResponse{
		AttemptID:       snap.AttemptID,
		UserCode:        snap.UserCode,
		VerificationURL: snap.VerificationURI,
		ExpiresIn:       snap.ExpiresInSec,
	}}, nil
}

// handleGetAuthStatus retourne l'état d'une tentative d'auth watcher.
// GET /api/v1/watcher/auth/{attempt_id}
func (h *WatcherHandler) handleGetAuthStatus(ctx context.Context, in *watcherAuthStatusInput) (*watcherAuthStatusOutput, error) {
	slog.Debug("watcher_handler: GetAuthStatus", "attempt_id", in.AttemptID)
	snap := h.attempts.Snapshot(in.AttemptID)
	if snap == nil {
		return nil, humacore.NewError(http.StatusNotFound, "attempt_not_found", "tentative introuvable")
	}

	return &watcherAuthStatusOutput{Body: watcherAuthStatusResponse{
		Status:    snap.Status,
		ErrorCode: snap.ErrorCode,
		Gamertag:  snap.Gamertag,
		XUID:      snap.XUID,
	}}, nil
}

// handlePatchSubscriptions met à jour les joueurs surveillés.
// PATCH /api/v1/watcher/subscriptions
func (h *WatcherHandler) handlePatchSubscriptions(ctx context.Context, in *watcherSubscriptionsInput) (*watcherSubscriptionsOutput, error) {
	var req watcherSubscriptionsRequest
	// Body OPTIONNEL (MarkRequestBodyOptional) : corps absent → req zéro (défaut
	// ["all"] plus bas). Corps présent mais malformé → 400 invalid_body (parse
	// maison, contrat préservé). Le décodeur d'origine (json.NewDecoder(r.Body))
	// rejetait aussi un corps vide ; ici Huma laisse passer le corps absent.
	if len(in.RawBody) > 0 {
		if err := json.Unmarshal(in.RawBody, &req); err != nil {
			return nil, humacore.NewError(http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		}
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_error", "impossible de lire les settings")
	}

	players := req.SubscribedPlayers
	if len(players) == 0 {
		players = []string{"all"}
	}
	cfg.WatcherSubscribedPlayers = players

	if err := h.settingsStore.Save(cfg); err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "settings_save_error", "impossible de sauvegarder les settings")
	}

	// Mettre à jour le daemon si actif
	if h.daemon != nil && h.daemon.IsRunning() {
		h.daemon.UpdateSubscriptions(players)
	}

	slog.Info("watcher_handler: subscriptions mises à jour", "players", players)
	out := &watcherSubscriptionsOutput{}
	out.Body.SubscribedPlayers = players
	return out, nil
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
			a.ErrorCode = errCodeDeviceFlowAcquire
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
		slog.ErrorContext(ctx, "watcher_handler: impossible de sauvegarder les tokens XSTS", "err", err)
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
