// Package service — xbox_auth_service.go : XboxSSOLinkStrategy.
//
// Implémente auth.LinkStrategy pour le mode SSO Xbox (D3, cf. SPRINT_XBOX_SSO.md §0bis).
// Quand le Device Code Flow réussit :
//  1. GetByXUID : retrouver un user existant
//  2. Sinon CreateFromXbox : créer un user à partir du gamertag/XUID
//  3. Wire la session (login automatique)
//
// Ce que la stratégie NE fait PAS (ADR 0035 D3, 2026-09-15) : créer un profil de
// suivi, ni mettre le joueur sous surveillance du watcher tant qu'aucun profil
// n'existe. Le compte et les credentials sont bien créés — le refresh token est
// ce qui rendra le profil utilisable plus tard — mais un compte sans profil est
// un état VALIDE et INERTE : aucun poller, aucun sync, aucune écriture disque.
// Sa seule sortie est le wizard de mise en place (ou une invitation, cf. le plan
// frère). C'est le maillon manquant de l'incident du 2026-07-23.
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
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/userstore"
)

// WatcherDaemon est l'interface minimale dont XboxSSOLinkStrategy a besoin pour
// notifier le watcher d'un nouveau joueur après login Xbox SSO. Implémentée par
// *watcher.Daemon. Définie ici pour éviter une dépendance service → watcher.
//
// Cleanup 2026-05-26 : AddUserClient (multi-user RTA dédié) supprimé en même
// temps que la dépendance RTA WebSocket. Le watcher utilise désormais le REST
// poller partagé via le client tracker — AddPlayer suffit.
type WatcherDaemon interface {
	AddPlayer(ctx context.Context, p domain.PlayerSummary) error
	IsRunning() bool
}

// WatcherDaemonGetter résout le daemon au moment du login (lazy).
// Nécessaire car le daemon est démarré APRÈS la création de XboxSSOLinkStrategy
// dans main.go (ordre : router setup → strategy → daemon start). Retourne nil
// si le daemon n'est pas (encore) prêt.
type WatcherDaemonGetter func() WatcherDaemon

// InviteResolver réserve un code d'invitation (vérification et marquage atomiques,
// émetteur encore existant) et le rend si la création du compte échoue. Satisfait
// par *userstore.InviteStore.
type InviteResolver interface {
	Consume(code, usedBy string) (*domain.InviteCode, error)
	Release(code, usedBy string) error
}

// GroupJoiner ajoute un membre à un groupe. Satisfait par *groupstore.GroupStore.
type GroupJoiner interface {
	AddMember(groupID, xuid, gamertag string) error
}

// XboxSSOLinkStrategy implémente auth.LinkStrategy pour le mode SSO Xbox.
// L'user est créé (ou retrouvé) à partir du XUID validé par le flow MSAL.
//
// PR 2.5a : si `tokenStore` est non-nil, persiste les tokens RTA dans
// data/auth/watcher_tokens/{xuid}.json.
// PR 2.5b : si `daemonGetter` retourne un daemon non-nil et tournant, ET que
// `profileGate` s'ouvre pour ce xuid (ADR 0035 D3, 2026-09-15), ajoute le joueur
// au watcher pour subscribe RTA immédiat (sous réserve que le tracker actuel
// soit ami Xbox de ce joueur — sinon status=3 silencieux). Sans profil suivi, le
// login aboutit et les tokens sont persistés, mais rien n'est mis en marche.
type XboxSSOLinkStrategy struct {
	users        *userstore.Store
	tokenStore   *auth.MultiUserTokenStore // optionnel
	daemonGetter WatcherDaemonGetter       // optionnel — lazy resolve
	// instanceLocked résout le verrou « instance fermée ». Quand true, un XUID
	// INCONNU est refusé (pas de CreateFromXbox) ; un XUID connu se connecte
	// normalement. nil → jamais verrouillé. Cf. WithInstanceLock.
	instanceLocked func() bool
	// profileGate : porte « profil suivi » (ADR 0035 D3). Le watcher n'est notifié
	// que si elle s'ouvre pour (titre par défaut, xuid). nil = porte ouverte —
	// seam de test ; le serveur la pose toujours.
	profileGate domain.ProfileGate
	// invites + groups : flow d'invitation. Si la session porte un
	// PendingInviteCode valide, le login lève le verrou d'instance, consomme le
	// code, et — SI l'invitation porte un groupe — y ajoute le joueur.
	// nil → flow désactivé.
	invites InviteResolver
	groups  GroupJoiner
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

// WithInstanceLock injecte le résolveur du verrou « instance fermée ». Quand il
// retourne true, un XUID inconnu ne peut pas créer de compte (login SSO refusé) ;
// les utilisateurs existants se connectent normalement.
func (s *XboxSSOLinkStrategy) WithInstanceLock(fn func() bool) *XboxSSOLinkStrategy {
	s.instanceLocked = fn
	return s
}

// WithProfileGate pose la porte « profil suivi » : sans profil déclaré pour le
// xuid, le login SSO réussit et les tokens sont persistés, mais le watcher n'est
// PAS notifié (ADR 0035 D3).
func (s *XboxSSOLinkStrategy) WithProfileGate(g domain.ProfileGate) *XboxSSOLinkStrategy {
	s.profileGate = g
	return s
}

// WithInviteStore injecte le résolveur d'invitations (flow d'invitation).
func (s *XboxSSOLinkStrategy) WithInviteStore(inv InviteResolver) *XboxSSOLinkStrategy {
	s.invites = inv
	return s
}

// WithGroupStore injecte le store de groupes (ajout du joueur au groupe après login).
func (s *XboxSSOLinkStrategy) WithGroupStore(g GroupJoiner) *XboxSSOLinkStrategy {
	s.groups = g
	return s
}

// Vérification compile-time.
var _ auth.LinkStrategy = (*XboxSSOLinkStrategy)(nil)

// OnAuthSuccess login l'user via XUID après un Device Code Flow Xbox réussi.
// Crée le user s'il n'existe pas (CreateFromXbox). Modifie sess en place
// (Username, Role, Locale, CurrentPlayerSlug, LinkedHaloIdentity).
//
// Une invitation portée par la session est RÉSERVÉE (consommée) avant toute création
// de compte : de deux connexions qui présentent le même code, une seule la reçoit ;
// si la création échoue ensuite, le code est rendu.
func (s *XboxSSOLinkStrategy) OnAuthSuccess(ctx context.Context, attempt *auth.Attempt, sess *domain.SessionData) error {
	if attempt.XUID == "" || attempt.Gamertag == "" {
		return fmt.Errorf("xbox_sso: xuid ou gamertag manquant après Exchange (xuid=%q, gamertag=%q)",
			attempt.XUID, attempt.Gamertag)
	}

	invite := s.claimPendingInvite(ctx, sess, attempt)
	user, created, err := s.resolveAccount(ctx, attempt, invite != nil)
	if err != nil {
		s.releaseInvite(ctx, invite, attempt)
		return refusalCause(err, sess, invite)
	}

	// Wire la session — login automatique.
	username := user.Username
	role := string(user.Role)
	sess.Username = &username
	sess.Role = &role
	if user.Locale != "" {
		sess.Locale = user.Locale
	}

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

	// Flow d'invitation : ajout au groupe s'il y en a un, droit de provisioning si le
	// compte vient d'être créé.
	if invite != nil {
		s.redeemInvite(ctx, invite, attempt, sess, user, created)
	}

	// PR 2.5a — Persistance tokens RTA (best-effort, non bloquant).
	// Si tokenStore est nil (test ou config minimale), on saute simplement.
	if s.tokenStore != nil {
		s.persistRTATokens(ctx, attempt)
	}

	// PR 2.5b — Notifier le watcher daemon pour subscribe RTA immédiat.
	// Le getter résout le daemon lazy (créé après cette strategy dans main.go).
	// No-op si daemon nil ou pas démarré. Best-effort : erreur loggée, login OK.
	// ADR 0035 D3 : uniquement si le xuid a un profil SUIVI — sinon on s'arrête
	// ici, compte et tokens en place, rien d'actif.
	if s.daemonGetter != nil && s.watcherAllowedFor(ctx, attempt) {
		if d := s.daemonGetter(); d != nil && d.IsRunning() {
			s.notifyWatcher(ctx, attempt, user, d)
		}
	}
	return nil
}

// resolveAccount retrouve le compte du xuid, ou le crée. created dit si CE login l'a
// créé (seul ce cas donne droit au provisioning de profil, D3). Instance fermée : un
// xuid inconnu sans invitation réservée est refusé (auth.ErrInstanceLocked).
func (s *XboxSSOLinkStrategy) resolveAccount(ctx context.Context, attempt *auth.Attempt, invited bool) (*domain.User, bool, error) {
	user, err := s.users.GetByXUID(attempt.XUID)
	switch {
	case errors.Is(err, userstore.ErrUserNotFound):
		if s.instanceLocked != nil && s.instanceLocked() && !invited {
			slog.WarnContext(ctx, "xbox_sso: nouvelle identité refusée — instance verrouillée",
				"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
			return nil, false, auth.ErrInstanceLocked
		}
		user, err = s.users.CreateFromXbox(attempt.Gamertag, attempt.XUID)
		if err != nil {
			return nil, false, fmt.Errorf("xbox_sso: CreateFromXbox: %w", err)
		}
		slog.InfoContext(ctx, "xbox_sso: user créé depuis SSO",
			"username", user.Username, "xuid", attempt.XUID)
		return user, true, nil
	case err != nil:
		return nil, false, fmt.Errorf("xbox_sso: GetByXUID: %w", err)
	}
	// User existant — touche LastLoginAt.
	if _, authErr := s.users.AuthenticateByXUID(attempt.XUID); authErr != nil {
		slog.WarnContext(ctx, "xbox_sso: AuthenticateByXUID échec (non bloquant)",
			"username", user.Username, "err", authErr)
	}
	slog.InfoContext(ctx, "xbox_sso: user existant authentifié",
		"username", user.Username, "xuid", attempt.XUID)
	return user, false, nil
}

// watcherAllowedFor dit si le watcher peut prendre en charge ce compte : il lui
// faut un profil suivi sur le titre par défaut (ADR 0035 D3). Le refus est
// journalisé — un compte qui se connecte sans profil est une information, pas un
// silence.
func (s *XboxSSOLinkStrategy) watcherAllowedFor(ctx context.Context, attempt *auth.Attempt) bool {
	if s.profileGate == nil {
		return true
	}
	if s.profileGate(ctx, title.DefaultSlug, attempt.XUID) {
		return true
	}
	slog.InfoContext(ctx, "xbox_sso: compte sans profil suivi — watcher non notifié, provisioning requis",
		"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
	return false
}

// notifyWatcher ajoute l'user au tracking du watcher daemon après login Xbox SSO.
//
// Cleanup 2026-05-26 : avec la suppression du RTA legacy + multi-user, il ne
// reste qu'AddPlayer (le watcher spawn un REST poller via le client tracker
// partagé pour ce joueur). Best-effort : un échec ne bloque pas le login.
func (s *XboxSSOLinkStrategy) notifyWatcher(ctx context.Context, attempt *auth.Attempt, user *domain.User, daemon WatcherDaemon) {
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
	slog.InfoContext(ctx, "xbox_sso: joueur ajouté au watcher daemon",
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
		XUID:              attempt.XUID,
		Gamertag:          attempt.Gamertag,
		XSTSToken:         attempt.XSTSRTAToken,
		XSTSUserHash:      attempt.XSTSRTAUserHash,
		XSTSExpiresAt:     attempt.XSTSRTAExpiresAt,
		AccessToken:       attempt.MicrosoftAccessToken,
		OAuthExpiresAt:    time.Now().Add(50 * time.Minute), // conservateur (Microsoft expires_in ~1h)
		OAuthRefreshToken: attempt.OAuthRefreshToken,
	}
	// Upsert remplace le fichier ENTIER (ADR 0023 : source unique par xuid).
	// Préserver le refresh_token durable si CE flow ne l'a pas porté — sans ce
	// merge, un login sans RT effacerait le credential déjà semé.
	if existing, err := s.tokenStore.Load(attempt.XUID); err == nil && existing != nil {
		if tokens.OAuthRefreshToken == "" {
			tokens.OAuthRefreshToken = existing.OAuthRefreshToken
		}
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
