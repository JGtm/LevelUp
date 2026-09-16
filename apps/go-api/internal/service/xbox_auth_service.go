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

// InviteResolver lit et consomme un code d'invitation. Satisfait par
// *userstore.InviteStore.
type InviteResolver interface {
	Get(code string) (*domain.InviteCode, error)
	Consume(code, usedBy string) error
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
// PR 2.5b : si `daemonGetter` retourne un daemon non-nil et tournant, ajoute
// le joueur au watcher pour subscribe RTA immédiat (sous réserve que le tracker
// actuel soit ami Xbox de ce joueur — sinon status=3 silencieux).
type XboxSSOLinkStrategy struct {
	users        *userstore.Store
	tokenStore   *auth.MultiUserTokenStore // optionnel
	daemonGetter WatcherDaemonGetter       // optionnel — lazy resolve
	// instanceLocked résout le verrou « instance fermée ». Quand true, un XUID
	// INCONNU est refusé (pas de CreateFromXbox) ; un XUID connu se connecte
	// normalement. nil → jamais verrouillé. Cf. WithInstanceLock.
	instanceLocked func() bool
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

// ErrInstanceLocked est retournée par OnAuthSuccess quand un XUID inconnu tente
// de se connecter sur une instance fermée.
var ErrInstanceLocked = errors.New("xbox_sso: instance fermée (nouvelle identité refusée)")

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

	// Flow d'invitation : une invitation valide en session (avec ou sans groupe)
	// lève le verrou d'instance et est consommée après login.
	pendingInvite := s.resolvePendingInvite(ctx, sess)

	// created : le compte vient d'être créé par CE login. Seul ce cas donne droit
	// au provisioning de profil (D3).
	created := false
	user, err := s.users.GetByXUID(attempt.XUID)
	switch {
	case errors.Is(err, userstore.ErrUserNotFound):
		// Instance fermée : un XUID inconnu ne peut pas créer de compte — sauf s'il
		// présente une invitation valide.
		if s.instanceLocked != nil && s.instanceLocked() && pendingInvite == nil {
			slog.WarnContext(ctx, "xbox_sso: nouvelle identité refusée — instance verrouillée",
				"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
			return ErrInstanceLocked
		}
		user, err = s.users.CreateFromXbox(attempt.Gamertag, attempt.XUID)
		if err != nil {
			return fmt.Errorf("xbox_sso: CreateFromXbox: %w", err)
		}
		created = true
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

	// Flow d'invitation : consommation du code, ajout au groupe s'il y en a un,
	// droit de provisioning si le compte vient d'être créé.
	if pendingInvite != nil {
		s.redeemInvite(ctx, pendingInvite, attempt, sess, user, created)
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

// resolvePendingInvite retourne l'invitation valide portée par la session, ou
// nil. Une invitation expirée/consommée/introuvable, ou sans store câblé, est
// traitée comme absente (login normal, soumis au verrou).
//
// UNE INVITATION SANS GROUPE EST VALIDE (2026-09-15, D3). Auparavant elle était
// rejetée ici comme « legacy password », ce qui rendait `POST /admin/invites`
// inopérant en SSO Xbox : l'admin générait un lien qui ne créait aucun compte
// sur instance verrouillée.
func (s *XboxSSOLinkStrategy) resolvePendingInvite(ctx context.Context, sess *domain.SessionData) *domain.InviteCode {
	if s.invites == nil || sess == nil || sess.PendingInviteCode == "" {
		return nil
	}
	inv, err := s.invites.Get(sess.PendingInviteCode)
	if err != nil || inv == nil {
		slog.WarnContext(ctx, "xbox_sso: invitation en session introuvable", "code", sess.PendingInviteCode, "err", err)
		return nil
	}
	if !inv.IsValid() {
		slog.WarnContext(ctx, "xbox_sso: invitation en session invalide (expirée/consommée)", "code", inv.Code)
		return nil
	}
	return inv
}

// redeemInvite consomme TOUJOURS le code, ajoute le joueur au groupe SI
// l'invitation en porte un, et pose le droit de provisioning sur le compte
// UNIQUEMENT s'il vient d'être créé (`created`) — un compte existant qui
// présente une invitation ne gagne aucun droit de créer un profil.
//
// Best-effort : un échec est loggé mais ne bloque pas le login (la session est
// déjà câblée). Vide PendingInviteCode pour éviter une re-consommation.
func (s *XboxSSOLinkStrategy) redeemInvite(
	ctx context.Context, inv *domain.InviteCode, attempt *auth.Attempt,
	sess *domain.SessionData, user *domain.User, created bool,
) {
	if inv.GroupID != "" {
		if s.groups != nil {
			if err := s.groups.AddMember(inv.GroupID, attempt.XUID, attempt.Gamertag); err != nil {
				slog.ErrorContext(ctx, "xbox_sso: ajout au groupe échoué (non bloquant)",
					"group_id", inv.GroupID, "xuid", attempt.XUID, "err", err)
			} else {
				slog.InfoContext(ctx, "xbox_sso: joueur ajouté au groupe via invitation",
					"group_id", inv.GroupID, "gamertag", attempt.Gamertag)
			}
		}
	} else {
		slog.InfoContext(ctx, "xbox_sso: invitation sans groupe consommée",
			"code", inv.Code, "gamertag", attempt.Gamertag)
	}
	if err := s.invites.Consume(inv.Code, attempt.Gamertag); err != nil {
		slog.WarnContext(ctx, "xbox_sso: consommation invitation échouée (non bloquant)",
			"code", inv.Code, "err", err)
	}
	if created && user != nil {
		if err := s.users.SetProvisionGrant(user.Username, inv.Code); err != nil {
			slog.ErrorContext(ctx, "xbox_sso: pose du droit de provisioning échouée (non bloquant)",
				"username", user.Username, "err", err)
		} else {
			slog.InfoContext(ctx, "xbox_sso: droit de provisioning posé pour l'invité",
				"username", user.Username)
		}
	}
	sess.PendingInviteCode = ""
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
