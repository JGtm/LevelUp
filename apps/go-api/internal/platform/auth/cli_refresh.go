// Package auth — cli_refresh.go : helper canonique pour les CLI one-shot
// qui ont besoin de récupérer des HaloTokens (refresh-metadata,
// refresh-career-ranks, populate-career-rank-images, diag_emblem_colors).
//
// Avant ADR 0023, chaque CLI dupliquait le pipeline MSAL silent refresh →
// OAuth refresh → Exchange, lisait l'env var + sync_meta DuckDB, et NE
// PERSISTAIT PAS la rotation. Conséquence : un seul usage par RT, le
// prochain CLI échoue avec invalid_grant.
//
// Cette fonction centralise le pipeline et :
//  1. Lit MultiUserTokenStore en premier (canonique).
//  2. Tombe sur les sources legacy fournies par le caller (MSAL/OAuth depuis
//     DuckDB ou env var) si le store n'a rien.
//  3. Persiste systématiquement le RT rotaté dans le store.
//
// Le caller fournit les valeurs legacy déjà lues (le package auth ne dépend
// pas de DuckDB).
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// LegacyAuthInputs regroupe les sources legacy déjà lues par le caller.
// Tous champs optionnels — fournir uniquement ceux disponibles.
type LegacyAuthInputs struct {
	OAuthRT   string // sync_meta.oauth_refresh_token OU env var
	MSALCache string // sync_meta.msal_token_cache
	Source    string // label pour les logs (ex: "duckdb", "env_var")
}

// RefreshHaloTokensViaStoreFirst tente d'obtenir des HaloTokens (Spartan +
// Clearance) en suivant la priorité ADR 0023 :
//
//  1. MultiUserTokenStore (canonique) — MSAL silent puis OAuth refresh
//  2. Legacy fourni par le caller — MSAL silent puis OAuth refresh
//
// Les rotations OAuth sont systématiquement persistées dans le store (et le
// cache process invalidé pour le xuid). Retourne nil si aucune source ne
// donne de tokens utilisables.
//
// Si store == nil ou xuid == "", saute directement au chemin legacy.
func RefreshHaloTokensViaStoreFirst(
	ctx context.Context,
	store *MultiUserTokenStore,
	provider TokenProvider,
	xuid, gamertag string,
	legacy LegacyAuthInputs,
) (*ExchangeResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("auth: provider nil")
	}

	// --- Source 1 : MultiUserTokenStore ---
	if store != nil && xuid != "" {
		if user, err := store.Load(xuid); err == nil && user != nil {
			result, refreshErr := tryRefreshFromUserEntry(ctx, provider, store, xuid, user)
			if result != nil {
				// Refresh OK → l'éventuel flag reauth_required est obsolète.
				_ = store.ClearReauthRequired(xuid)
				return result, nil
			}
			// PR-B : des credentials existaient (MSAL cache OU RT) mais le refresh a
			// échoué. On ne marque reauth_required (→ bannière de reconnexion) QUE si
			// le RT est réellement RÉVOQUÉ (invalid_grant). Un échec transitoire
			// (429/réseau/5xx) ou config ne se règle pas par une reconnexion
			// utilisateur → pas de bannière (faux positif). (Aucun credential = compte
			// jamais authentifié → pas de marquage non plus.)
			if (user.MSALCacheJSON != "" || user.OAuthRefreshToken != "") &&
				ClassifyAuthError(refreshErr) == AuthErrorRevoked {
				if _, merr := store.MarkReauthRequired(xuid, user.Gamertag); merr != nil {
					slog.WarnContext(ctx, "cli_auth: marquage reauth_required échoué", "xuid", xuid, "err", merr)
				} else {
					slog.WarnContext(ctx, "cli_auth: refresh_token mort — reauth_required",
						"xuid", xuid, "gamertag", user.Gamertag)
				}
			}
		} else if err != nil && !errors.Is(err, ErrUserTokensNotFound) {
			slog.WarnContext(ctx, "cli_auth: lecture store échouée", "xuid", xuid, "err", err)
		}
	}

	// --- Source 2 : legacy fourni par le caller ---
	return tryRefreshFromLegacyInputs(ctx, provider, store, xuid, gamertag, legacy), nil
}

// tryRefreshFromUserEntry tente MSAL silent puis OAuth refresh depuis l'entrée
// store. Retourne (result, nil) sur succès ; (nil, err) si l'OAuth refresh a
// échoué — err porte la classe d'échec (cf. ClassifyAuthError), ce qui permet au
// caller de ne marquer reauth_required QUE pour un RT révoqué. (nil, nil) si
// aucune source n'a produit de token sans erreur classifiable (ex. Exchange KO).
func tryRefreshFromUserEntry(
	ctx context.Context,
	provider TokenProvider,
	store *MultiUserTokenStore,
	xuid string,
	user *UserTokens,
) (*ExchangeResult, error) {
	if user.MSALCacheJSON != "" {
		if at, err := provider.TrySilentRefresh(ctx, user.MSALCacheJSON); err == nil && at != "" {
			if result, err := provider.Exchange(ctx, at); err == nil && result != nil {
				slog.DebugContext(ctx, "cli_auth: tokens via MSAL (store)", "xuid", xuid)
				return result, nil
			}
		}
	}
	if user.OAuthRefreshToken != "" {
		at, rotatedRT, err := provider.TryOAuthRefreshWithRotation(ctx, user.OAuthRefreshToken)
		if err != nil {
			// Erreur classifiable (invalid_grant=revoked, 429/réseau=transient…) —
			// remontée au caller pour décider du marquage reauth.
			return nil, err
		}
		if at != "" {
			if rotatedRT != "" && rotatedRT != user.OAuthRefreshToken {
				if werr := store.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
					slog.WarnContext(ctx, "cli_auth: persistance RT rotaté échouée (store)",
						"xuid", xuid, "err", werr)
				}
			}
			if result, err := provider.Exchange(ctx, at); err == nil && result != nil {
				slog.DebugContext(ctx, "cli_auth: tokens via OAuth (store)", "xuid", xuid)
				return result, nil
			}
		}
	}
	return nil, nil
}

func tryRefreshFromLegacyInputs(
	ctx context.Context,
	provider TokenProvider,
	store *MultiUserTokenStore,
	xuid, gamertag string,
	legacy LegacyAuthInputs,
) *ExchangeResult {
	if legacy.MSALCache != "" {
		slog.WarnContext(ctx, "cli_auth: legacy MSAL utilisé — à migrer",
			"gamertag", gamertag, "source", legacy.Source, "deprecated_since", "ADR-0023")
		observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBMSAL)
		if at, err := provider.TrySilentRefresh(ctx, legacy.MSALCache); err == nil && at != "" {
			if result, err := provider.Exchange(ctx, at); err == nil && result != nil {
				return result
			}
		}
	}
	if legacy.OAuthRT == "" {
		return nil
	}
	slog.WarnContext(ctx, "cli_auth: legacy RT utilisé — à migrer",
		"gamertag", gamertag, "source", legacy.Source, "deprecated_since", "ADR-0023")
	observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBOAuth)

	at, rotatedRT, err := provider.TryOAuthRefreshWithRotation(ctx, legacy.OAuthRT)
	if err != nil || at == "" {
		if err != nil {
			slog.WarnContext(ctx, "cli_auth: OAuth refresh échoué (legacy)", "err", err)
		}
		return nil
	}
	if store != nil && xuid != "" && rotatedRT != "" && rotatedRT != legacy.OAuthRT {
		if werr := store.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
			slog.WarnContext(ctx, "cli_auth: persistance RT rotaté (legacy refresh) échouée",
				"xuid", xuid, "err", werr)
		}
	}
	result, err := provider.Exchange(ctx, at)
	if err != nil || result == nil {
		return nil
	}
	return result
}

// HaloTokensFromExchange retourne les domain.HaloTokens d'un ExchangeResult.
// Helper de commodité pour les CLI qui veulent seulement la structure tokens.
func HaloTokensFromExchange(result *ExchangeResult) *domain.HaloTokens {
	if result == nil {
		return nil
	}
	return result.Tokens
}
