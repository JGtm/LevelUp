// Package auth — cli_refresh.go : helper canonique pour les CLI one-shot
// qui ont besoin de récupérer des HaloTokens (refresh-metadata,
// refresh-career-ranks, populate-career-rank-images, diag_emblem_colors).
//
// Avant ADR 0023, chaque CLI dupliquait le pipeline OAuth refresh → Exchange,
// lisait l'env var + sync_meta DuckDB, et NE PERSISTAIT PAS la rotation.
// Conséquence : un seul usage par RT, le prochain CLI échoue avec invalid_grant.
//
// Depuis ADR 0023 Phase 5 (2026-08-25), la seule source de credentials est le
// MultiUserTokenStore : plus aucune branche legacy (sync_meta, env var). Cette
// fonction centralise le pipeline et persiste systématiquement le RT rotaté.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
)

// RefreshHaloTokensViaStoreFirst obtient des HaloTokens (Spartan + Clearance)
// depuis le MultiUserTokenStore (source unique ADR 0023) : OAuth refresh du RT
// persisté puis Exchange Halo.
//
// Les rotations OAuth sont systématiquement persistées dans le store. Retourne
// (nil, nil) si le store ne couvre pas le joueur ou si aucun token utilisable
// n'en sort.
func RefreshHaloTokensViaStoreFirst(
	ctx context.Context,
	store *MultiUserTokenStore,
	provider TokenProvider,
	xuid, gamertag string,
) (*ExchangeResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("auth: provider nil")
	}
	if store == nil || xuid == "" {
		slog.WarnContext(ctx, "cli_auth: store ou xuid absent — aucune source de credentials",
			"gamertag", gamertag, "xuid", xuid)
		return nil, nil
	}

	user, err := store.Load(xuid)
	if err != nil {
		if !errors.Is(err, ErrUserTokensNotFound) {
			slog.WarnContext(ctx, "cli_auth: lecture store échouée", "xuid", xuid, "err", err)
		}
		return nil, nil
	}
	if user == nil {
		return nil, nil
	}

	result, refreshErr := RefreshFromStoreEntry(ctx, provider, store, xuid, user)
	if result != nil {
		// Refresh OK → l'éventuel flag reauth_required est obsolète.
		_ = store.ClearReauthRequired(xuid)
		return result, nil
	}
	// PR-B : un credential existait (RT) mais le refresh a échoué. On ne marque
	// reauth_required (→ bannière de reconnexion) QUE si le RT est réellement
	// RÉVOQUÉ (invalid_grant). Un échec transitoire (429/réseau/5xx) ou config ne
	// se règle pas par une reconnexion utilisateur → pas de bannière (faux
	// positif). (Aucun credential = compte jamais authentifié → pas de marquage.)
	if user.OAuthRefreshToken != "" && ClassifyAuthError(refreshErr) == AuthErrorRevoked {
		if _, merr := store.MarkReauthRequired(xuid, user.Gamertag); merr != nil {
			slog.WarnContext(ctx, "cli_auth: marquage reauth_required échoué", "xuid", xuid, "err", merr)
		} else {
			slog.WarnContext(ctx, "cli_auth: refresh_token mort — reauth_required",
				"xuid", xuid, "gamertag", user.Gamertag)
		}
	}
	return nil, nil
}

// RefreshFromStoreEntry tente un OAuth refresh (rotation persistée) depuis une
// entrée store DÉJÀ chargée. Retourne (result, nil) sur succès ; (nil, err) si
// l'OAuth refresh a échoué — err porte la classe d'échec (cf. ClassifyAuthError),
// ce qui permet au caller de ne marquer reauth_required QUE pour un RT révoqué.
// (nil, nil) si aucune source n'a produit de token sans erreur classifiable (ex.
// Exchange KO).
//
// Source UNIQUE de la cascade store (K1b) : `store` est l'interface
// `UserTokenStore` (seul `UpdateOAuthRefreshToken` y est appelé) → réutilisable par
// `ServiceRegistry.tryRefreshFromAuthStore` (api), qui applique ENSUITE sa propre
// politique reauth (clear-on-success, pas de marquage sur le chemin serveur).
func RefreshFromStoreEntry(
	ctx context.Context,
	provider TokenProvider,
	store UserTokenStore,
	xuid string,
	user *UserTokens,
) (*ExchangeResult, error) {
	if user.OAuthRefreshToken == "" {
		return nil, nil
	}
	at, rotatedRT, err := provider.TryOAuthRefreshWithRotation(ctx, user.OAuthRefreshToken)
	if err != nil {
		// Erreur classifiable (invalid_grant=revoked, 429/réseau=transient…) —
		// remontée au caller pour décider du marquage reauth.
		return nil, err
	}
	if at == "" {
		return nil, nil
	}
	if rotatedRT != "" && rotatedRT != user.OAuthRefreshToken {
		if werr := store.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
			slog.WarnContext(ctx, "cli_auth: persistance RT rotaté échouée (store)",
				"xuid", xuid, "err", werr)
		}
	}
	result, err := provider.Exchange(ctx, at)
	if err != nil || result == nil {
		return nil, nil
	}
	slog.DebugContext(ctx, "cli_auth: tokens via OAuth (store)", "xuid", xuid)
	return result, nil
}

// HaloTokensFromExchange retourne les domain.HaloTokens d'un ExchangeResult.
// Helper de commodité pour les CLI qui veulent seulement la structure tokens.
func HaloTokensFromExchange(result *ExchangeResult) *domain.HaloTokens {
	if result == nil {
		return nil
	}
	return result.Tokens
}
