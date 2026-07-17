// Package auth — refresh_loop.go : goroutine de refresh automatique des tokens XSTS.
//
// Vérifie toutes les checkInterval (5 min) :
//   - Si l'access_token OAuth expire dans < oauthMargin (10 min) → refresh via refresh_token
//   - Si le token XSTS expire dans < xstsMargin (15 min) → re-générer XBL + XSTS
//
// Notifie via un callback quand le XSTS est renouvelé (pour reconnecter le WebSocket RTA).
package auth

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/observability/logging"
)

const (
	// refreshCheckInterval est l'intervalle de vérification des expirations.
	refreshCheckInterval = 5 * time.Minute
	// oauthRefreshMargin : refresh l'access_token si expire dans < 10 min.
	oauthRefreshMargin = 10 * time.Minute
	// xstsRefreshMargin : refresh le XSTS si expire dans < 20 min.
	// Marge volontairement large : un XSTS Xbox Live expire après ~60 min et
	// un subscribe refusé (status=3) survient si le token expire entre deux reconnexions.
	xstsRefreshMargin = 20 * time.Minute
	// xstsDefaultTTL est utilisé comme fallback si NotAfter est absent de la réponse XSTS.
	// Valeur conservatrice : un XSTS Xbox Live expire en pratique après 60 min.
	xstsDefaultTTL = 55 * time.Minute
	// oauthDefaultTTL est la durée de vie d'un access_token OAuth (~1h).
	oauthDefaultTTL = 60 * time.Minute
)

// RefreshCallback est appelé quand le XSTS est renouvelé.
// Le XSTSResult contient le nouveau token + userhash.
type RefreshCallback func(result *XSTSResult)

// XSTSAcquireFn est une fonction qui acquiert un token XSTS à partir d'un access_token.
// Utilisé pour faciliter les tests unitaires (injection de dépendance).
type XSTSAcquireFn func(ctx context.Context, accessToken string) (*XSTSResult, error)

// RefreshLoop gère le refresh automatique des tokens.
type RefreshLoop struct {
	store         *TokenStore
	multiMirror   *MultiUserTokenStore // PR 2.5b : mirror writes vers multi-user store (nullable)
	onXSTS        RefreshCallback
	interval      time.Duration
	acquireXSTSFn XSTSAcquireFn // nil → AcquireXSTSForRTA (prod)

	// onReauthNotify est invoqué UNE fois (transition) quand le refresh_token meurt
	// (reauth requise). Nullable. Sert au ping Discord opt-in (PR-B). Le marquage
	// d'état est fait directement sur multiMirror (MarkReauthRequired).
	onReauthNotify func(xuid, gamertag string)
}

// NewRefreshLoop crée un RefreshLoop.
// onXSTSRefreshed est appelé quand un nouveau XSTS est obtenu (pour reconnecter RTA).
func NewRefreshLoop(store *TokenStore, onXSTSRefreshed RefreshCallback) *RefreshLoop {
	return &RefreshLoop{
		store:    store,
		onXSTS:   onXSTSRefreshed,
		interval: refreshCheckInterval,
	}
}

// WithMultiUserMirror branche un MultiUserTokenStore qui recevra une copie
// miroir des écritures XSTS + OAuth du tracker initial (legacy TokenStore).
// PR 2.5b — pavement pour future migration read-path : maintient la cohérence
// entre les 2 stores tant que la lecture passe par le legacy.
//
// nil → no-op (comportement legacy strict).
func (r *RefreshLoop) WithMultiUserMirror(multi *MultiUserTokenStore) *RefreshLoop {
	r.multiMirror = multi
	return r
}

// WithReauthNotify injecte le callback de notification (ping Discord opt-in)
// invoqué une seule fois lors de la transition « refresh_token mort » (PR-B).
func (r *RefreshLoop) WithReauthNotify(fn func(xuid, gamertag string)) *RefreshLoop {
	r.onReauthNotify = fn
	return r
}

// Run démarre la boucle de refresh. Bloquant — à lancer dans une goroutine.
// S'arrête quand ctx est annulé.
func (r *RefreshLoop) Run(ctx context.Context) {
	// Sprint B1 commit 17 : event_id sur la loop globale (un id pour toute
	// la vie du process). Chaque check() crée son propre sous-event_id pour
	// granularité par tick.
	ctx, loopID := logging.WithEvent(ctx, "auth.token_refresh_loop")
	slog.InfoContext(ctx, "refresh_loop: démarré",
		"check_interval", r.interval,
		"oauth_margin", oauthRefreshMargin,
		"xsts_margin", xstsRefreshMargin,
		"event", loopID,
	)

	// Vérification initiale immédiate
	r.check(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "refresh_loop: arrêté")
			return
		case <-ticker.C:
			r.check(ctx)
		}
	}
}

// check effectue une vérification et un refresh si nécessaire.
func (r *RefreshLoop) check(ctx context.Context) {
	tokens, err := r.store.Load()
	if err != nil {
		slog.WarnContext(ctx, "refresh_loop: erreur lecture token store", "err", err)
		return
	}

	if !tokens.HasRefreshToken() {
		slog.DebugContext(ctx, "refresh_loop: pas de refresh_token, skip")
		return
	}

	// Étape 1 : refresh access_token si nécessaire
	if !tokens.IsOAuthValid(oauthRefreshMargin) {
		slog.InfoContext(ctx, "refresh_loop: access_token expiré ou proche expiration, refresh...")
		if err := r.refreshOAuth(ctx, tokens); err != nil {
			slog.ErrorContext(ctx, "refresh_loop: échec refresh OAuth", "err", err)
			return
		}
		// Recharger après update
		tokens, err = r.store.Load()
		if err != nil {
			slog.WarnContext(ctx, "refresh_loop: erreur relecture après OAuth refresh", "err", err)
			return
		}
	}

	// Étape 2 : refresh XSTS si nécessaire
	if !tokens.IsXSTSValid(xstsRefreshMargin) {
		slog.InfoContext(ctx, "refresh_loop: XSTS expiré ou proche expiration, refresh...")
		r.refreshXSTS(ctx, tokens)
	}
}

// refreshOAuth utilise le refresh_token pour obtenir un nouvel access_token.
func (r *RefreshLoop) refreshOAuth(ctx context.Context, tokens *StoredTokens) error {
	// D3-03 (revue 2026-06-01) : variante AVEC rotation. Microsoft tourne le
	// refresh_token à CHAQUE usage ; l'ancienne ExchangeRefreshToken (sans rotation)
	// jetait le RT rotaté → l'ancien RT stocké finissait par devenir invalide
	// (classe incident invalid_grant, ADR 0023). On propage le RT rotaté au store.
	accessToken, rotatedRT, _, err := ExchangeRefreshTokenWithRotation(ctx, tokens.RefreshToken)
	if err != nil {
		return err
	}
	if accessToken == "" {
		slog.WarnContext(ctx, "refresh_loop: refresh_token révoqué ou expiré")
		r.signalReauthRequired(ctx, tokens)
		return nil
	}
	// rotatedRT == "" → UpdateOAuth préserve l'ancien RT (pas de rotation côté MS) ;
	// rotatedRT != "" → on stocke le nouveau RT (sinon il serait perdu).
	if err := r.store.UpdateOAuth(accessToken, rotatedRT, oauthDefaultTTL); err != nil {
		return err
	}
	slog.InfoContext(ctx, "refresh_loop: access_token renouvelé")
	// Refresh OK → l'éventuel flag reauth_required du tracker est obsolète.
	r.clearReauthRequired(ctx, tokens)
	// PR 2.5b — mirror OAuth refresh vers multi-user store si configuré.
	r.mirrorTrackerToMultiUser(ctx)
	return nil
}

// signalReauthRequired marque reauth_required (mirror multi-user) et notifie une
// seule fois (transition) — refresh_token du tracker mort. Best-effort.
func (r *RefreshLoop) signalReauthRequired(ctx context.Context, tokens *StoredTokens) {
	if r.multiMirror == nil || tokens.XSTSXUID == "" {
		return
	}
	newly, err := r.multiMirror.MarkReauthRequired(tokens.XSTSXUID, tokens.XSTSGamertag)
	if err != nil {
		slog.WarnContext(ctx, "refresh_loop: marquage reauth_required échoué", "xuid", tokens.XSTSXUID, "err", err)
		return
	}
	if newly {
		slog.WarnContext(ctx, "refresh_loop: reauth_required — reconnexion Xbox nécessaire",
			"xuid", tokens.XSTSXUID, "gamertag", tokens.XSTSGamertag)
		if r.onReauthNotify != nil {
			r.onReauthNotify(tokens.XSTSXUID, tokens.XSTSGamertag)
		}
	}
}

// clearReauthRequired efface le flag reauth_required du tracker après un refresh OK.
func (r *RefreshLoop) clearReauthRequired(ctx context.Context, tokens *StoredTokens) {
	if r.multiMirror == nil || tokens.XSTSXUID == "" {
		return
	}
	if err := r.multiMirror.ClearReauthRequired(tokens.XSTSXUID); err != nil {
		slog.WarnContext(ctx, "refresh_loop: clear reauth_required échoué", "xuid", tokens.XSTSXUID, "err", err)
	}
}

// refreshXSTS obtient un nouveau token XSTS pour RTA.
func (r *RefreshLoop) refreshXSTS(ctx context.Context, tokens *StoredTokens) {
	if tokens.AccessToken == "" {
		slog.WarnContext(ctx, "refresh_loop: pas d'access_token pour refresh XSTS")
		return
	}

	acquireFn := r.acquireXSTSFn
	if acquireFn == nil {
		acquireFn = AcquireXSTSForRTA
	}
	result, err := acquireFn(ctx, tokens.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "refresh_loop: échec acquisition XSTS", "err", err)
		return
	}

	if err := r.store.UpdateXSTS(result, xstsDefaultTTL); err != nil {
		slog.ErrorContext(ctx, "refresh_loop: échec sauvegarde XSTS", "err", err)
		return
	}

	// Utiliser la vraie date d'expiration si NotAfter est renseigné.
	expiresAt := result.NotAfter
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(xstsDefaultTTL)
	}
	slog.InfoContext(ctx, "refresh_loop: XSTS renouvelé",
		"gamertag", result.Gamertag,
		"xuid", result.XUID,
		"expires_at", expiresAt,
	)

	// PR 2.5b — mirror XSTS vers multi-user store si configuré.
	r.mirrorTrackerToMultiUser(ctx)

	if r.onXSTS != nil {
		r.onXSTS(result)
	}
}

// mirrorTrackerToMultiUser copie l'état tracker actuel (legacy TokenStore)
// vers MultiUserTokenStore[XSTSXUID]. Best-effort : log WARN sur échec mais
// pas de propagation. Skip silencieux si multiMirror nil ou tokens incomplets.
func (r *RefreshLoop) mirrorTrackerToMultiUser(ctx context.Context) {
	if r.multiMirror == nil {
		return
	}
	tokens, err := r.store.Load()
	if err != nil {
		slog.DebugContext(ctx, "refresh_loop: mirror — Load legacy échoué", "err", err)
		return
	}
	if tokens.XSTSXUID == "" {
		slog.DebugContext(ctx, "refresh_loop: mirror — xuid vide, skip")
		return
	}
	// Read-modify-write : PRÉSERVER le refresh_token + MSAL cache + flags reauth
	// déjà dans le store (ex. RT e1cb35ab frais semé par le callback SSO). Upsert
	// remplace TOUTE l'entrée ; un UserTokens neuf (sans RT) écraserait le RT du
	// store à vide → la migration boot le re-remplirait avec un RT env legacy
	// périmé → AADSTS70000 en boucle (incident 2026-06-13). Le mirror ne pousse
	// QUE les champs XSTS/access issus du tracker.
	mirror, lerr := r.multiMirror.Load(tokens.XSTSXUID)
	if lerr != nil || mirror == nil {
		mirror = &UserTokens{XUID: tokens.XSTSXUID}
	}
	if tokens.XSTSGamertag != "" {
		mirror.Gamertag = tokens.XSTSGamertag
	}
	mirror.XSTSToken = tokens.XSTSToken
	mirror.XSTSUserHash = tokens.XSTSUserHash
	mirror.XSTSExpiresAt = tokens.XSTSExpiresAt
	mirror.AccessToken = tokens.AccessToken
	mirror.OAuthExpiresAt = tokens.OAuthExpiresAt
	if err := r.multiMirror.Upsert(mirror); err != nil {
		slog.WarnContext(ctx, "refresh_loop: mirror vers multi-user store échoué (non-bloquant)",
			"xuid", tokens.XSTSXUID, "err", err)
		return
	}
	slog.DebugContext(ctx, "refresh_loop: mirror tracker -> multi-user OK",
		"xuid", tokens.XSTSXUID, "gamertag", tokens.XSTSGamertag)
}
