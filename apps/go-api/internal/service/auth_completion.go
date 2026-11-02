// Package service — auth_completion.go : ce qu'une connexion Microsoft réussie écrit dans le
// magasin de jetons (ADR 0023), quel que soit le mode d'authentification ET le chemin de connexion :
// code d'appareil, ou navigateur par défaut (code d'autorisation + PKCE, ADR 0045 D-22). Les deux
// aboutissent à la même tentative autorisée, remise à la session, donc à la même stratégie post-flux
// et à ce service.
//
// POURQUOI. Le jeton de rafraîchissement est la seule credential durable d'un joueur :
// sans lui le poste n'a plus de jeton au bout d'une heure et le sync s'arrête. Seule la
// stratégie du mode "xbox" l'écrivait ; en mode "none" (poste seul) la connexion posait
// la session et laissait le magasin vide. Ce service est le point unique d'écriture
// après une connexion hors SSO : la stratégie SSO l'appelle sans condition
// (le compte existe, le jeton rendra le profil utilisable plus tard), la stratégie des
// autres modes ne l'appelle que pour un profil SUIVI (ADR 0035 D3 : aucune écriture
// disque pour une identité que rien ne suit).
//
// ORDRE RÉEL DU PARCOURS DE MISE EN PLACE. Le wizard connecte d'abord (le xuid n'est
// connu qu'à cette étape), puis déclare le profil (POST /setup/players -> Onboard). À la
// première installation, la porte "profil suivi" est donc fermée au moment de la
// connexion. Les jetons sont alors gardés en MÉMOIRE seulement (jamais sur disque,
// durée bornée) et rejoués par SeedCredentials quand l'annuaire a créé le profil, avant
// la prise en charge par le watcher. Un redémarrage du serveur entre les deux étapes
// perd ces jetons : le joueur se reconnecte, rien n'est jamais écrit pour un profil
// qui n'existe pas.
//
// Mise en commun du jeton (ADR 0038) : ce service n'écrit AUCUNE décision, sauf sous
// WithLocalPoolGrant (installation desktop à utilisateur unique, ADR 0045 D-8), qui enregistre
// l'accord du jeton qui n'en porte aucune. Sans cela, un jeton sans décision n'entre pas
// dans le pool public.
package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/tokencache"
)

const (
	// oauthExpiryMargin : expiration conservatrice du jeton d'accès Microsoft posée à
	// la connexion (expires_in ~ 1 h).
	oauthExpiryMargin = 50 * time.Minute
	// pendingCredentialsTTL borne l'attente entre la connexion et la déclaration du
	// profil dans le wizard. Passé ce délai le jeton gardé en mémoire est oublié.
	pendingCredentialsTTL = 2 * time.Hour
	// pendingCredentialsMax borne le nombre d'identités en attente (une instance de
	// poste seul en voit une ; le plafond protège un serveur exposé).
	pendingCredentialsMax = 16
)

// CredentialStore est ce que le service attend du magasin de jetons. Satisfait par
// *auth.MultiUserTokenStore.
type CredentialStore interface {
	Load(xuid string) (*auth.UserTokens, error)
	Upsert(tokens *auth.UserTokens) error
}

// pendingCredentials : jetons d'une connexion dont le profil n'existe pas encore.
type pendingCredentials struct {
	tokens *auth.UserTokens
	since  time.Time
}

// AuthCompletion écrit les jetons d'une connexion Microsoft aboutie (code d'appareil ou navigateur) dans le
// magasin, directement (Persist) ou sous la porte "profil suivi" (CompleteForTrackedProfile
// puis SeedCredentials). Sûr pour un usage concurrent.
type AuthCompletion struct {
	store CredentialStore
	// gate : porte "profil suivi" (ADR 0035 D3). nil = aucun profil connu : les jetons
	// sont gardés en attente, jamais écrits (défaut sûr).
	gate domain.ProfileGate
	now  func() time.Time
	// grantLocalPool : un jeton sans décision de mise en commun reçoit l'accord de
	// l'utilisateur local (WithLocalPoolGrant).
	grantLocalPool bool

	mu      sync.Mutex
	pending map[string]pendingCredentials
}

// NewAuthCompletion crée le service. gate peut être nil si seul Persist sert.
func NewAuthCompletion(store CredentialStore, gate domain.ProfileGate) *AuthCompletion {
	return &AuthCompletion{
		store:   store,
		gate:    gate,
		now:     time.Now,
		pending: make(map[string]pendingCredentials),
	}
}

// WithLocalPoolGrant fait enregistrer, à l'écriture d'un jeton de rafraîchissement qui ne porte
// aucune décision de mise en commun, l'accord de l'utilisateur local avec la base
// auth.PoolSharingBasisDesktopSingleUser (ADR 0045 D-8). Une décision existante, accord comme
// refus, n'est jamais écrasée. À activer par la capacité PoolGrantedToLocalUser du profil, jamais
// sur une instance à plusieurs utilisateurs.
func (c *AuthCompletion) WithLocalPoolGrant(enabled bool) *AuthCompletion {
	c.grantLocalPool = enabled
	return c
}

// Persist écrit sans condition les jetons de la tentative. Rend false, sans erreur,
// quand la tentative ne porte rien à écrire (ni jeton de rafraîchissement ni XSTS) ou
// quand le magasin est absent. Un échec d'écriture est rendu en erreur, sans journal :
// l'appelant le journalise et décide s'il est bloquant (il ne l'est jamais pour la connexion).
func (c *AuthCompletion) Persist(ctx context.Context, attempt *auth.Attempt) (bool, error) {
	tokens := c.tokensFromAttempt(attempt)
	if tokens == nil {
		slog.DebugContext(ctx, "auth_completion: ni jeton de rafraîchissement ni XSTS capturés, rien à persister",
			"xuid", attempt.XUID)
		return false, nil
	}
	if err := c.write(tokens); err != nil {
		return false, err
	}
	slog.InfoContext(ctx, "auth_completion: jetons persistés", "xuid", attempt.XUID, "gamertag", attempt.Gamertag,
		"refresh_token", tokens.OAuthRefreshToken != "", "xsts_expires_at", attempt.XSTSRTAExpiresAt)
	return true, nil
}

// CompleteForTrackedProfile persiste les jetons si l'identité a un profil suivi (titre
// par défaut, comme la stratégie SSO) ; sinon les garde en mémoire pour SeedCredentials.
// Rend true si le magasin a été écrit.
func (c *AuthCompletion) CompleteForTrackedProfile(ctx context.Context, attempt *auth.Attempt) bool {
	tokens := c.tokensFromAttempt(attempt)
	if tokens == nil {
		slog.DebugContext(ctx, "auth_completion: ni jeton de rafraîchissement ni XSTS capturés, rien à persister",
			"xuid", attempt.XUID)
		return false
	}
	if c.gate != nil && c.gate(ctx, title.DefaultSlug, attempt.XUID) {
		if err := c.write(tokens); err != nil {
			slog.ErrorContext(ctx, "auth_completion: persistance des jetons échouée — le poste perdra son jeton",
				"xuid", attempt.XUID, "err", err)
			return false
		}
		slog.InfoContext(ctx, "auth_completion: jetons persistés pour un profil suivi",
			"xuid", attempt.XUID, "gamertag", attempt.Gamertag, "refresh_token", tokens.OAuthRefreshToken != "")
		return true
	}
	c.stash(tokens)
	slog.InfoContext(ctx, "auth_completion: aucun profil suivi, jetons gardés en mémoire jusqu'à la création du profil",
		"xuid", attempt.XUID, "gamertag", attempt.Gamertag)
	return false
}

// SeedCredentials rejoue les jetons en attente d'un xuid dont le profil vient d'être
// créé (appelé par l'annuaire, entre la création du profil et le suivi live). Sans
// jeton en attente (autre administrateur, redémarrage, délai écoulé) c'est un no-op
// journalisé en debug : le joueur se connectera par le wizard.
func (c *AuthCompletion) SeedCredentials(ctx context.Context, xuid string) {
	tokens := c.takePending(xuid)
	if tokens == nil {
		slog.DebugContext(ctx, "auth_completion: aucun jeton en attente pour ce profil", "xuid", xuid)
		return
	}
	if err := c.write(tokens); err != nil {
		slog.ErrorContext(ctx, "auth_completion: persistance des jetons en attente échouée — jetons conservés en mémoire",
			"xuid", xuid, "err", err)
		c.stash(tokens)
		return
	}
	slog.InfoContext(ctx, "auth_completion: jetons en attente persistés à la création du profil",
		"xuid", xuid, "gamertag", tokens.Gamertag, "refresh_token", tokens.OAuthRefreshToken != "")
}

// tokensFromAttempt construit les UserTokens d'une tentative, ou nil quand elle ne porte
// ni jeton de rafraîchissement ni XSTS. Le jeton de rafraîchissement suffit : le XSTS
// RTA (best-effort à la connexion) ne conditionne pas la durabilité du poste.
func (c *AuthCompletion) tokensFromAttempt(attempt *auth.Attempt) *auth.UserTokens {
	if c.store == nil || attempt == nil || attempt.XUID == "" {
		return nil
	}
	if attempt.OAuthRefreshToken == "" && attempt.XSTSRTAToken == "" {
		return nil
	}
	return &auth.UserTokens{
		XUID:              attempt.XUID,
		Gamertag:          attempt.Gamertag,
		XSTSToken:         attempt.XSTSRTAToken,
		XSTSUserHash:      attempt.XSTSRTAUserHash,
		XSTSExpiresAt:     attempt.XSTSRTAExpiresAt,
		AccessToken:       attempt.MicrosoftAccessToken,
		OAuthExpiresAt:    c.now().Add(oauthExpiryMargin),
		OAuthRefreshToken: attempt.OAuthRefreshToken,
	}
}

// write fait l'Upsert. Upsert remplace le fichier ENTIER (ADR 0023 : source unique par
// xuid) : le refresh token durable déjà semé est préservé quand CE flux n'en porte pas,
// sinon une connexion sans RT effacerait le credential existant. Sous WithLocalPoolGrant,
// un jeton de rafraîchissement sans décision de mise en commun reçoit l'accord local. Après
// chaque écriture réussie (Persist, CompleteForTrackedProfile, SeedCredentials, code d'appareil
// comme navigateur), le cache process des HaloTokens du xuid est purgé.
func (c *AuthCompletion) write(tokens *auth.UserTokens) error {
	if c.store == nil {
		return errors.New("auth_completion: magasin de jetons absent")
	}
	existing, err := c.store.Load(tokens.XUID)
	// Fichier illisible : l'état de la décision est inconnu, l'accord n'est pas posé (une décision
	// de refus ne doit pas être remplacée sur un doute).
	decisionKnown := err == nil || errors.Is(err, auth.ErrUserTokensNotFound)
	if err != nil {
		existing = nil
	}
	if tokens.OAuthRefreshToken == "" && existing != nil {
		tokens.OAuthRefreshToken = existing.OAuthRefreshToken
	}
	if c.grantLocalPool && decisionKnown && tokens.OAuthRefreshToken != "" && (existing == nil || existing.PoolSharing == nil) {
		tokens.PoolSharing = &auth.PoolSharing{
			Granted: true, DecidedAt: c.now().UTC(), Basis: auth.PoolSharingBasisDesktopSingleUser,
		}
	}
	if err := c.store.Upsert(tokens); err != nil {
		return err
	}
	// Nouvelle connexion = nouvelle chaîne d'authentification : le cache process des HaloTokens
	// (TTL 50 min) servirait encore l'ancienne chaîne aux lectures live (carrière, défis, passe de
	// combat) ; même invalidation que le SSO du site après l'écriture du jeton.
	tokencache.InvalidateCachedPlayerTokens(tokens.XUID)
	return nil
}

// stash garde les jetons en mémoire (le plus récent d'une identité l'emporte), après
// avoir purgé les entrées expirées puis, au plafond, la plus ancienne.
func (c *AuthCompletion) stash(tokens *auth.UserTokens) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for xuid, p := range c.pending {
		if now.Sub(p.since) > pendingCredentialsTTL {
			delete(c.pending, xuid)
		}
	}
	if _, replacing := c.pending[tokens.XUID]; !replacing && len(c.pending) >= pendingCredentialsMax {
		var oldestXUID string
		var oldest time.Time
		for xuid, p := range c.pending {
			if oldestXUID == "" || p.since.Before(oldest) {
				oldestXUID, oldest = xuid, p.since
			}
		}
		delete(c.pending, oldestXUID)
	}
	c.pending[tokens.XUID] = pendingCredentials{tokens: tokens, since: now}
}

// takePending retire et rend les jetons en attente d'un xuid (nil si absents ou périmés).
func (c *AuthCompletion) takePending(xuid string) *auth.UserTokens {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.pending[xuid]
	if !ok {
		return nil
	}
	delete(c.pending, xuid)
	if c.now().Sub(p.since) > pendingCredentialsTTL {
		return nil
	}
	return p.tokens
}

// authLinkStrategy décore la stratégie d'un mode sans SSO (mot de passe, "none") :
// après la stratégie interne, la connexion réussie est complétée par le service.
type authLinkStrategy struct {
	inner      auth.LinkStrategy
	completion *AuthCompletion
}

// NewAuthLinkStrategy décore inner : les jetons sont persistés (profil suivi) ou
// mis en attente (profil à venir) une fois la stratégie interne passée. Un refus de la
// stratégie interne (hors ErrSessionNotAuthenticated, qui signale seulement l'absence
// de compte local à lier) n'écrit rien.
func NewAuthLinkStrategy(inner auth.LinkStrategy, completion *AuthCompletion) auth.LinkStrategy {
	return &authLinkStrategy{inner: inner, completion: completion}
}

// OnAuthSuccess implémente auth.LinkStrategy.
func (s *authLinkStrategy) OnAuthSuccess(ctx context.Context, attempt *auth.Attempt, sess *domain.SessionData) error {
	var err error
	if s.inner != nil {
		err = s.inner.OnAuthSuccess(ctx, attempt, sess)
	}
	if err != nil && !errors.Is(err, auth.ErrSessionNotAuthenticated) {
		return err
	}
	s.completion.CompleteForTrackedProfile(ctx, attempt)
	return err
}
