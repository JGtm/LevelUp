// Package handlers — auth.go : endpoints Device Code Flow (Sprint 15).
//
// POST /auth/device-flow/start      → démarre ou récupère une tentative en cours
// GET  /auth/device-flow/{attempt_id} → retourne l'état d'une tentative
//
// MIGRÉ vers Huma (Phase 3b) : MountDeviceFlow crée humacore.NewAPI(r) sur le
// routeur /api/v1 et enregistre les 2 routes via huma.Post/Get. Aucun middleware
// d'authz (bootstrap auth public, seule la session est requise). Logique métier
// inchangée, seul le wrapping HTTP change.
//
// Single-flight par session : si une tentative "pending" existe pour la session
// courante, elle est renvoyée sans créer une nouvelle tentative MSAL.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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
	linkStrategy auth_platform.LinkStrategy  // logique post-flow (password ou xbox SSO)
	provider     auth_platform.TokenProvider // abstrait le mécanisme d'acquisition de tokens
}

// UserLinker est l'interface dont a besoin PasswordLinkStrategy.
// Conservée comme alias type pour limiter le diff sur les call sites — la définition
// canonique vit dans platform/auth (cf. link_strategy.go).
type UserLinker = auth_platform.UserLinker

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

// WithUserStore est conservé pour la backward compat des call sites existants.
// Auto-wrap le UserLinker dans une PasswordLinkStrategy (comportement historique).
// Pour le mode SSO Xbox, préférer WithLinkStrategy() avec un XboxSSOLinkStrategy.
func (h *AuthHandler) WithUserStore(us UserLinker) *AuthHandler {
	h.linkStrategy = auth_platform.NewPasswordLinkStrategy(us)
	return h
}

// WithLinkStrategy injecte explicitement la LinkStrategy à appliquer après le flow.
// Override tout WithUserStore antérieur.
func (h *AuthHandler) WithLinkStrategy(s auth_platform.LinkStrategy) *AuthHandler {
	h.linkStrategy = s
	return h
}

// MountDeviceFlow enregistre les 2 routes Device Code Flow via Huma sur le routeur
// fourni (/api/v1, sans authz : bootstrap auth public). Le path param {attempt_id}
// est local à la route GET.
func (h *AuthHandler) MountDeviceFlow(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/auth/device-flow/start", h.handleStartDeviceFlow)
	huma.Get(api, "/auth/device-flow/{attempt_id}", h.handleGetDeviceFlowStatus)
}

// deviceFlowStatusInput : path param {attempt_id}.
type deviceFlowStatusInput struct {
	AttemptID string `path:"attempt_id"`
}
type deviceFlowStartOutput struct {
	Body domain.DeviceFlowStartResponse
}
type deviceFlowStatusOutput struct {
	Body domain.DeviceFlowStatusResponse
}

// handleStartDeviceFlow démarre un Device Code Flow Microsoft pour Halo Infinite.
// POST /auth/device-flow/start (migré Huma — aucun corps lu, Input vide).
func (h *AuthHandler) handleStartDeviceFlow(ctx context.Context, _ *struct{}) (*deviceFlowStartOutput, error) {
	if h.demoMode {
		return nil, humacore.NewError(http.StatusUnprocessableEntity, "demo_mode", "authentification indisponible en mode démo")
	}

	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session non initialisée")
	}

	// Single-flight : si une tentative "pending" existe, la renvoyer.
	attempt, isNew := h.attempts.GetOrCreate(sess.SessionID)
	// CRUCIAL : lier la session à la tentative → IsMeaningful → la session anonyme
	// est PERSISTÉE (sinon le SessionID n'est jamais sauvé et le poll de statut,
	// clé = SessionID, retourne 404 indéfiniment / le single-flight se casse).
	sess.DeviceFlowAttemptID = &attempt.AttemptID
	if !isNew {
		return &deviceFlowStartOutput{Body: deviceFlowStartResponse(attempt)}, nil
	}

	// Nouvelle tentative : initier le Device Code Flow via le provider configuré.
	flow, err := h.provider.InitDeviceFlow(ctx)
	if err != nil {
		h.attempts.Update(attempt.AttemptID, func(a *auth_platform.Attempt) {
			a.Status = auth_platform.AttemptStatusFailed
			a.ErrorCode = "msal_init_error"
			a.ErrorDetail = err.Error()
		})
		return nil, humacore.NewError(http.StatusInternalServerError, "msal_init_error", "impossible de démarrer le Device Code Flow")
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
	return &deviceFlowStartOutput{Body: deviceFlowStartResponse(snapshot)}, nil
}

// handleGetDeviceFlowStatus retourne l'état d'une tentative Device Code Flow.
// GET /auth/device-flow/{attempt_id} (migré Huma).
func (h *AuthHandler) handleGetDeviceFlowStatus(ctx context.Context, in *deviceFlowStatusInput) (*deviceFlowStatusOutput, error) {
	sess := middleware.GetSession(ctx)
	if sess == nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "no_session", "session non initialisée")
	}

	snapshot := h.attempts.Snapshot(in.AttemptID)
	if snapshot == nil || snapshot.SessionID != sess.SessionID {
		return nil, humacore.NewError(http.StatusNotFound, "attempt_not_found", "tentative introuvable")
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
		// LinkStrategy gère le post-flow (password : LinkIdentity ; xbox SSO : login direct).
		if h.linkStrategy != nil && snapshot.Gamertag != "" {
			if err := h.linkStrategy.OnAuthSuccess(ctx, snapshot, sess); err != nil {
				slog.ErrorContext(ctx, "auth: LinkStrategy.OnAuthSuccess échec",
					"attempt_id", in.AttemptID, "err", err)
			}
		}
		_ = h.sessionStore.Save(sess)
	}

	return &deviceFlowStatusOutput{Body: deviceFlowStatusResponse(snapshot)}, nil
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
			a.ErrorCode = errCodeMSALAcquire
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

	// PR 2.5a : capture des éléments nécessaires pour persistance RTA via SSO Xbox.
	// Best-effort : tout échec ici est non bloquant (l'user peut quand même se connecter).
	var (
		msalCacheJSON string
		xstsRTA       *auth_platform.XSTSResult
	)
	// Cache MSAL (contient le refresh_token Microsoft pour rafraîchir l'access_token plus tard).
	if msalFlow, ok := flow.(interface{ MSALCacheJSON() string }); ok {
		msalCacheJSON = msalFlow.MSALCacheJSON()
	}
	// XSTS audience xboxlive.com (RTA) — distinct du XSTS Halo retourné par provider.Exchange.
	// L'access_token Microsoft est encore valide ici (~1h), on l'utilise pour l'acquisition.
	xstsRTAResult, xstsErr := auth_platform.AcquireXSTSForRTA(ctx, accessToken)
	if xstsErr != nil {
		slog.WarnContext(ctx, "auth: AcquireXSTSForRTA échec (non bloquant)",
			"attempt_id", attemptID, "err", xstsErr)
	} else {
		xstsRTA = xstsRTAResult
	}

	// Marquer comme autorisé et stocker tokens + identité dans l'attempt store.
	// GetDeviceFlowStatus les transférera dans la session lors du prochain poll.
	h.attempts.Update(attemptID, func(a *auth_platform.Attempt) {
		a.Status = auth_platform.AttemptStatusAuthorized
		a.SpartanToken = result.Tokens.SpartanToken
		a.ClearanceToken = result.Tokens.ClearanceToken
		a.Gamertag = result.Gamertag
		a.XUID = result.XUID

		// PR 2.5a : transport vers OnAuthSuccess (jamais exposé via HTTP).
		a.MicrosoftAccessToken = accessToken
		a.MSALCacheJSON = msalCacheJSON
		if xstsRTA != nil {
			a.XSTSRTAToken = xstsRTA.Token
			a.XSTSRTAUserHash = xstsRTA.UserHash
			a.XSTSRTAExpiresAt = xstsRTA.NotAfter
		}
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
