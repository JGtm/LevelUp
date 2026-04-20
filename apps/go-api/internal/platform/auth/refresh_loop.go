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
)

const (
	// refreshCheckInterval est l'intervalle de vérification des expirations.
	refreshCheckInterval = 5 * time.Minute
	// oauthRefreshMargin : refresh l'access_token si expire dans < 10 min.
	oauthRefreshMargin = 10 * time.Minute
	// xstsRefreshMargin : refresh le XSTS si expire dans < 15 min.
	xstsRefreshMargin = 15 * time.Minute
	// xstsDefaultTTL est la durée de vie estimée d'un token XSTS (~90 min).
	xstsDefaultTTL = 90 * time.Minute
	// oauthDefaultTTL est la durée de vie d'un access_token OAuth (~1h).
	oauthDefaultTTL = 60 * time.Minute
)

// RefreshCallback est appelé quand le XSTS est renouvelé.
// Le XSTSResult contient le nouveau token + userhash.
type RefreshCallback func(result *XSTSResult)

// RefreshLoop gère le refresh automatique des tokens.
type RefreshLoop struct {
	store    *TokenStore
	onXSTS   RefreshCallback
	interval time.Duration
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

// Run démarre la boucle de refresh. Bloquant — à lancer dans une goroutine.
// S'arrête quand ctx est annulé.
func (r *RefreshLoop) Run(ctx context.Context) {
	slog.InfoContext(ctx, "refresh_loop: démarré",
		"check_interval", r.interval,
		"oauth_margin", oauthRefreshMargin,
		"xsts_margin", xstsRefreshMargin,
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
	accessToken, err := ExchangeRefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		return err
	}
	if accessToken == "" {
		slog.WarnContext(ctx, "refresh_loop: refresh_token révoqué ou expiré")
		return nil
	}
	if err := r.store.UpdateOAuth(accessToken, "", oauthDefaultTTL); err != nil {
		return err
	}
	slog.InfoContext(ctx, "refresh_loop: access_token renouvelé")
	return nil
}

// refreshXSTS obtient un nouveau token XSTS pour RTA.
func (r *RefreshLoop) refreshXSTS(ctx context.Context, tokens *StoredTokens) {
	if tokens.AccessToken == "" {
		slog.WarnContext(ctx, "refresh_loop: pas d'access_token pour refresh XSTS")
		return
	}

	result, err := AcquireXSTSForRTA(ctx, tokens.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "refresh_loop: échec acquisition XSTS", "err", err)
		return
	}

	if err := r.store.UpdateXSTS(result, xstsDefaultTTL); err != nil {
		slog.ErrorContext(ctx, "refresh_loop: échec sauvegarde XSTS", "err", err)
		return
	}

	slog.InfoContext(ctx, "refresh_loop: XSTS renouvelé",
		"gamertag", result.Gamertag,
		"xuid", result.XUID,
		"expires_in", xstsDefaultTTL,
	)

	if r.onXSTS != nil {
		r.onXSTS(result)
	}
}
