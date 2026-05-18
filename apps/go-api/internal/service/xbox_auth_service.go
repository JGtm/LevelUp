// Package service — xbox_auth_service.go : XboxSSOLinkStrategy.
//
// Implémente auth.LinkStrategy pour le mode SSO Xbox (D3, cf. SPRINT_XBOX_SSO.md §0bis).
// Quand le Device Code Flow réussit :
//   1. GetByXUID : retrouver un user existant
//   2. Sinon CreateFromXbox : créer un user à partir du gamertag/XUID
//   3. Wire la session (login automatique)
//
// La récupération de BDD orpheline (§11 du plan) est différée à une future PR
// (nécessite pool.Invalidate + scan filesystem multi-titre).
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
)

// XboxSSOLinkStrategy implémente auth.LinkStrategy pour le mode SSO Xbox.
// L'user est créé (ou retrouvé) à partir du XUID validé par le flow MSAL.
//
// PR 2.5a : si `tokenStore` est non-nil, persiste les tokens RTA (XSTS + cache
// MSAL + access_token) dans data/auth/watcher_tokens/{xuid}.json pour usage
// ultérieur par le watcher daemon multi-user (PR 2.5b).
type XboxSSOLinkStrategy struct {
	users      *userstore.Store
	tokenStore *auth.MultiUserTokenStore // optionnel — nil = pas de persistance RTA
}

// NewXboxSSOLinkStrategy crée une XboxSSOLinkStrategy sans persistance RTA.
func NewXboxSSOLinkStrategy(users *userstore.Store) *XboxSSOLinkStrategy {
	return &XboxSSOLinkStrategy{users: users}
}

// WithTokenStore injecte le MultiUserTokenStore pour persister les tokens RTA
// après chaque login SSO réussi. Pattern fluent (cohérent avec les autres builders).
func (s *XboxSSOLinkStrategy) WithTokenStore(ts *auth.MultiUserTokenStore) *XboxSSOLinkStrategy {
	s.tokenStore = ts
	return s
}

// Vérification compile-time.
var _ auth.LinkStrategy = (*XboxSSOLinkStrategy)(nil)

// OnAuthSuccess login l'user via XUID après un Device Code Flow Xbox réussi.
// Crée le user s'il n'existe pas (CreateFromXbox). Modifie sess en place
// (Username, Role, CurrentPlayerSlug, LinkedHaloIdentity).
func (s *XboxSSOLinkStrategy) OnAuthSuccess(ctx context.Context, attempt *auth.Attempt, sess *domain.SessionData) error {
	if attempt.XUID == "" || attempt.Gamertag == "" {
		return fmt.Errorf("xbox_sso: xuid ou gamertag manquant après Exchange (xuid=%q, gamertag=%q)",
			attempt.XUID, attempt.Gamertag)
	}

	user, err := s.users.GetByXUID(attempt.XUID)
	switch {
	case errors.Is(err, userstore.ErrUserNotFound):
		user, err = s.users.CreateFromXbox(attempt.Gamertag, attempt.XUID)
		if err != nil {
			return fmt.Errorf("xbox_sso: CreateFromXbox: %w", err)
		}
		slog.InfoContext(ctx, "xbox_sso: user créé depuis SSO",
			"username", user.Username, "xuid", attempt.XUID)
	case err != nil:
		return fmt.Errorf("xbox_sso: GetByXUID: %w", err)
	default:
		// User existant — touche LastLoginAt.
		if _, authErr := s.users.AuthenticateByXUID(attempt.XUID); authErr != nil {
			slog.WarnContext(ctx, "xbox_sso: AuthenticateByXUID échec (non bloquant)",
				"username", user.Username, "err", authErr)
		}
		slog.InfoContext(ctx, "xbox_sso: user existant authentifié",
			"username", user.Username, "xuid", attempt.XUID)
	}

	// Wire la session — login automatique.
	username := user.Username
	role := string(user.Role)
	sess.Username = &username
	sess.Role = &role

	// CurrentPlayerSlug = gamertag (clé d'isolation FS dans data/players/{gamertag}/).
	slug := user.Gamertag
	if slug == "" {
		slug = attempt.Gamertag
	}
	sess.CurrentPlayerSlug = &slug

	sess.LinkedHaloIdentity = &domain.HaloIdentity{
		Gamertag: attempt.Gamertag,
		XUID:     attempt.XUID,
	}

	// PR 2.5a — Persistance tokens RTA (best-effort, non bloquant).
	// Si tokenStore est nil (test ou config minimale), on saute simplement.
	if s.tokenStore != nil {
		s.persistRTATokens(ctx, attempt)
	}
	return nil
}

// persistRTATokens écrit les UserTokens dans le MultiUserTokenStore.
// Best-effort : un échec est loggé mais n'interrompt pas le login.
// Le watcher daemon utilisera ces tokens (PR 2.5b) pour subscribe RTA.
func (s *XboxSSOLinkStrategy) persistRTATokens(ctx context.Context, attempt *auth.Attempt) {
	if attempt.XSTSRTAToken == "" {
		slog.DebugContext(ctx, "xbox_sso: pas de XSTS RTA capturé (AcquireXSTSForRTA a échoué), skip persistance")
		return
	}

	tokens := &auth.UserTokens{
		XUID:           attempt.XUID,
		Gamertag:       attempt.Gamertag,
		XSTSToken:      attempt.XSTSRTAToken,
		XSTSUserHash:   attempt.XSTSRTAUserHash,
		XSTSExpiresAt:  attempt.XSTSRTAExpiresAt,
		AccessToken:    attempt.MicrosoftAccessToken,
		OAuthExpiresAt: time.Now().Add(50 * time.Minute), // conservateur (Microsoft expires_in ~1h)
		MSALCacheJSON:  attempt.MSALCacheJSON,
	}
	if err := s.tokenStore.Upsert(tokens); err != nil {
		slog.WarnContext(ctx, "xbox_sso: persistance tokens RTA échouée (non bloquant)",
			"xuid", attempt.XUID, "err", err)
		return
	}
	slog.InfoContext(ctx, "xbox_sso: tokens RTA persistés",
		"xuid", attempt.XUID, "gamertag", attempt.Gamertag,
		"xsts_expires_at", attempt.XSTSRTAExpiresAt)
}
