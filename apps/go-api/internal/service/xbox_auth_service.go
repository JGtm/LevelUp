// Package service — xbox_auth_service.go : XboxSSOLinkStrategy.
//
// Implémente auth.LinkStrategy pour le mode SSO Xbox (D3, cf. SPRINT_XBOX_SSO.md §0bis).
// Quand le Device Code Flow réussit :
//  1. GetByXUID : retrouver un user existant
//  2. Sinon CreateFromXbox : créer un user à partir du gamertag/XUID
//  3. Wire la session (login automatique)
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

// WatcherDaemon est l'interface minimale dont XboxSSOLinkStrategy a besoin pour
// notifier le watcher d'un nouveau joueur après login Xbox SSO. Implémentée par
// *watcher.Daemon. Définie ici pour éviter une dépendance service → watcher.
//
// PR 2.5c : AddUserClient est préféré à AddPlayer car il crée une connexion RTA
// dédiée par user (résistant au social graph Xbox). AddPlayer reste pour les
// cas où on n'a pas de UserTokens (legacy ou daemon mono-tracker).
type WatcherDaemon interface {
	AddPlayer(ctx context.Context, p domain.PlayerSummary) error
	AddUserClient(ctx context.Context, userTokens *auth.UserTokens) error
	IsRunning() bool
}

// WatcherDaemonGetter résout le daemon au moment du login (lazy).
// Nécessaire car le daemon est démarré APRÈS la création de XboxSSOLinkStrategy
// dans main.go (ordre : router setup → strategy → daemon start). Retourne nil
// si le daemon n'est pas (encore) prêt.
type WatcherDaemonGetter func() WatcherDaemon

// XboxSSOLinkStrategy implémente auth.LinkStrategy pour le mode SSO Xbox.
// L'user est créé (ou retrouvé) à partir du XUID validé par le flow MSAL.
//
// PR 2.5a : si `tokenStore` est non-nil, persiste les tokens RTA dans
// data/auth/watcher_tokens/{xuid}.json.
// PR 2.5b : si `daemonGetter` retourne un daemon non-nil et tournant, ajoute
// le joueur au watcher pour subscribe RTA immédiat (sous réserve que le tracker
// actuel soit ami Xbox de ce joueur — sinon status=3 silencieux).
type XboxSSOLinkStrategy struct {
	users        *userstore.Store
	tokenStore   *auth.MultiUserTokenStore // optionnel
	daemonGetter WatcherDaemonGetter       // optionnel — lazy resolve
}

// NewXboxSSOLinkStrategy crée une XboxSSOLinkStrategy minimale (sans store ni daemon).
func NewXboxSSOLinkStrategy(users *userstore.Store) *XboxSSOLinkStrategy {
	return &XboxSSOLinkStrategy{users: users}
}

// WithTokenStore injecte le MultiUserTokenStore pour persister les tokens RTA
// après chaque login SSO réussi. Pattern fluent (cohérent avec les autres builders).
func (s *XboxSSOLinkStrategy) WithTokenStore(ts *auth.MultiUserTokenStore) *XboxSSOLinkStrategy {
	s.tokenStore = ts
	return s
}

// WithDaemonGetter injecte un getter qui résout le watcher daemon au moment de
// l'utilisation (PR 2.5b). Permet de capturer un *watcher.Daemon créé APRÈS
// la construction de la strategy via une closure.
func (s *XboxSSOLinkStrategy) WithDaemonGetter(getter WatcherDaemonGetter) *XboxSSOLinkStrategy {
	s.daemonGetter = getter
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

	// PR 2.5b — Notifier le watcher daemon pour subscribe RTA immédiat.
	// Le getter résout le daemon lazy (créé après cette strategy dans main.go).
	// No-op si daemon nil ou pas démarré. Best-effort : erreur loggée, login OK.
	if s.daemonGetter != nil {
		if d := s.daemonGetter(); d != nil && d.IsRunning() {
			s.notifyWatcher(ctx, attempt, user, d)
		}
	}
	return nil
}

// notifyWatcher ajoute l'user au tracking du watcher daemon après login Xbox SSO.
//
// PR 2.5c : préfère AddUserClient (1 RTA dédié par user, résistant au social
// graph Xbox) si on a des UserTokens persistés avec un XSTS valide. Sinon,
// fallback sur AddPlayer (utilise le tracker historique, dépend du social graph).
//
// Best-effort : si l'opération échoue, on loggue mais le login reste OK.
func (s *XboxSSOLinkStrategy) notifyWatcher(ctx context.Context, attempt *auth.Attempt, user *domain.User, daemon WatcherDaemon) {
	// Tente AddUserClient (RTA dédié) si on a des UserTokens utilisables.
	if s.tokenStore != nil && attempt.XSTSRTAToken != "" {
		ut, loadErr := s.tokenStore.Load(attempt.XUID)
		if loadErr == nil && ut != nil && ut.AuthHeader() != "" {
			if err := daemon.AddUserClient(ctx, ut); err != nil {
				slog.WarnContext(ctx, "xbox_sso: AddUserClient échoué, fallback AddPlayer",
					"xuid", attempt.XUID, "gamertag", attempt.Gamertag, "err", err)
			} else {
				slog.InfoContext(ctx, "xbox_sso: userClient ajouté au watcher daemon (RTA dédié)",
					"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
				return
			}
		} else if loadErr != nil {
			slog.DebugContext(ctx, "xbox_sso: tokens persistés introuvables, fallback AddPlayer",
				"xuid", attempt.XUID, "err", loadErr)
		}
	}

	// Fallback : AddPlayer (utilise le tracker historique).
	player := domain.PlayerSummary{
		PlayerSlug: attempt.Gamertag,
		Gamertag:   attempt.Gamertag,
		XUID:       attempt.XUID,
	}
	if user != nil {
		player.PlayerSlug = user.Gamertag
		player.Gamertag = user.Gamertag
	}
	if err := daemon.AddPlayer(ctx, player); err != nil {
		slog.WarnContext(ctx, "xbox_sso: AddPlayer au watcher daemon échoué (non bloquant)",
			"xuid", attempt.XUID, "gamertag", attempt.Gamertag, "err", err)
		return
	}
	slog.InfoContext(ctx, "xbox_sso: joueur ajouté au watcher daemon (tracker partagé)",
		"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
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
