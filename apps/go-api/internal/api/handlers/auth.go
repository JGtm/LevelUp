// Package handlers — auth.go : endpoints Device Code Flow (Sprint 15).
//
// POST /auth/device-flow/start      → démarre ou récupère une tentative en cours
// GET  /auth/device-flow/{attempt_id} → retourne l'état d'une tentative
//
// Single-flight par session : si une tentative "pending" existe pour la session
// courante, elle est renvoyée sans créer une nouvelle tentative MSAL.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/session"
)

const (
	deviceFlowPollIntervalSec = 5
)

// AuthHandler gère les endpoints d'authentification Halo via Device Code Flow.
type AuthHandler struct {
	sessionStore *session.Store
	attempts     *auth_platform.AttemptStore
	demoMode     bool
	userStore    UserLinker                  // optionnel — lie gamertag→user après Device Code Flow
	provider     auth_platform.TokenProvider // abstrait le mécanisme d'acquisition de tokens
}

// UserLinker est une interface pour lier l'identité Halo à un user local.
type UserLinker interface {
	LinkIdentity(username, gamertag, xuid string) error
}

// NewAuthHandler crée un AuthHandler.
func NewAuthHandler(
	sessionStore *session.Store,
	attempts *auth_platform.AttemptStore,
	demoMode bool,
	provider auth_platform.TokenProvider,
) *AuthHandler {
	return &AuthHandler{
		sessionStore: sessionStore,
		attempts:     attempts,
		demoMode:     demoMode,
		provider:     provider,
	}
}

// WithUserStore injecte le user store pour le LinkIdentity post Device Code Flow.
func (h *AuthHandler) WithUserStore(us UserLinker) *AuthHandler {
	h.userStore = us
	return h
}

// StartDeviceFlow démarre un Device Code Flow Microsoft pour Halo Infinite.
// POST /auth/device-flow/start
func (h *AuthHandler) StartDeviceFlow(w http.ResponseWriter, r *http.Request) {
	if h.demoMode {
		writeError(w, http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
		return
	}

	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	// Single-flight : si une tentative "pending" existe, la renvoyer.
	attempt, isNew := h.attempts.GetOrCreate(sess.SessionID)
	if !isNew {
		writeJSON(w, http.StatusOK, deviceFlowStartResponse(attempt))
		return
	}

	// Nouvelle tentative : initier le Device Code Flow via le provider configuré.
	flow, err := h.provider.InitDeviceFlow(r.Context())
	if err != nil {
		h.attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "msal_init_error"
			a.ErrorDetail = err.Error()
		})
		writeError(w, http.StatusInternalServerError, "msal_init_error", "impossible de démarrer le Device Code Flow")
		return
	}

	// Remplir les champs de la tentative.
	h.attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
		a.UserCode = flow.GetUserCode()
		a.VerificationURI = flow.GetVerificationURL()
		a.ExpiresInSec = flow.GetExpiresIn()
		a.DevFlow = flow
	})

	// Lancer le polling MSAL en arrière-plan.
	go h.pollDeviceFlow(attempt.AttemptID, flow)

	snapshot := h.attempts.Snapshot(attempt.AttemptID)
	writeJSON(w, http.StatusOK, deviceFlowStartResponse(snapshot))
}

// GetDeviceFlowStatus retourne l'état d'une tentative Device Code Flow.
// GET /auth/device-flow/{attempt_id}
func (h *AuthHandler) GetDeviceFlowStatus(w http.ResponseWriter, r *http.Request) {
	attemptID := chi.URLParam(r, "attempt_id")
	sess := middleware.GetSession(r.Context())
	if sess == nil {
		writeError(w, http.StatusInternalServerError, "no_session", "session non initialisée")
		return
	}

	snapshot := h.attempts.Snapshot(attemptID)
	if snapshot == nil || snapshot.SessionID != sess.SessionID {
		writeError(w, http.StatusNotFound, "attempt_not_found", "tentative introuvable")
		return
	}

	// Si l'auth a réussi, transférer les données Halo dans la session.
	if snapshot.Status == auth_platform.AttemptStatusAuthorized || snapshot.Status == auth_platform.AttemptStatusProvisioned {
		sess.AuthReady = true
		if snapshot.Gamertag != "" {
			sess.LinkedHaloIdentity = &domain.HaloIdentity{
				Gamertag: snapshot.Gamertag,
				XUID:     snapshot.XUID,
			}
		}
		// Stocker les tokens Halo dans la session (jamais exposés au navigateur).
		if snapshot.SpartanToken != "" {
			sess.HaloTokens = &domain.HaloTokens{
				SpartanToken:   snapshot.SpartanToken,
				ClearanceToken: snapshot.ClearanceToken,
			}
		}
		// Auth locale : lier l'identité Halo au user connecté + auto-select player.
		if h.userStore != nil && sess.Username != nil && snapshot.Gamertag != "" {
			_ = h.userStore.LinkIdentity(*sess.Username, snapshot.Gamertag, snapshot.XUID)
			slug := snapshot.Gamertag
			sess.CurrentPlayerSlug = &slug
		}
		_ = h.sessionStore.Save(sess)
	}

	writeJSON(w, http.StatusOK, deviceFlowStatusResponse(snapshot))
}

// =============================================================================
// Goroutine de polling MSAL
// =============================================================================

// pollDeviceFlow attend la complétion du Device Code Flow dans une goroutine.
// Enchaîne ensuite la chaîne d'échange Halo pour obtenir les tokens.
func (h *AuthHandler) pollDeviceFlow(attemptID string, flow auth_platform.DeviceFlow) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flow.GetExpiresIn())*time.Second)
	defer cancel()

	accessToken, err := flow.AcquireToken(ctx)
	if err != nil {
		h.attempts.Update(attemptID, func(a *auth_platform.Attempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "msal_acquire_error"
			a.ErrorDetail = err.Error()
		})
		return
	}

	// Chaîne d'échange : access_token → tokens Halo + identité.
	result, err := h.provider.Exchange(ctx, accessToken)
	if err != nil {
		h.attempts.Update(attemptID, func(a *auth_platform.Attempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "halo_exchange_error"
			a.ErrorDetail = err.Error()
		})
		return
	}

	// Marquer comme autorisé et stocker tokens + identité dans l'attempt store.
	// GetDeviceFlowStatus les transférera dans la session lors du prochain poll.
	h.attempts.Update(attemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.SpartanToken = result.Tokens.SpartanToken
		a.ClearanceToken = result.Tokens.ClearanceToken
		a.Gamertag = result.Gamertag
		a.XUID = result.XUID
	})
}

// =============================================================================
// Helpers de sérialisation
// =============================================================================

func deviceFlowStartResponse(a *auth_platform.Attempt) domain.DeviceFlowStartResponse {
	return domain.DeviceFlowStartResponse{
		AttemptID:       a.AttemptID,
		UserCode:        a.UserCode,
		VerificationURI: a.VerificationURI,
		ExpiresIn:       a.ExpiresInSec,
		PollIntervalSec: deviceFlowPollIntervalSec,
	}
}

func deviceFlowStatusResponse(a *auth_platform.Attempt) domain.DeviceFlowStatusResponse {
	resp := domain.DeviceFlowStatusResponse{
		AttemptID: a.AttemptID,
		Status:    a.Status,
	}
	if a.Gamertag != "" {
		resp.Gamertag = &a.Gamertag
	}
	if a.XUID != "" {
		resp.XUID = &a.XUID
	}
	if a.ErrorCode != "" {
		resp.ErrorCode = &a.ErrorCode
	}
	if a.ErrorDetail != "" {
		resp.ErrorDetail = &a.ErrorDetail
	}
	return resp
}
