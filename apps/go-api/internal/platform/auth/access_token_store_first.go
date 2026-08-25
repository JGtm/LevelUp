// Package auth — access_token_store_first.go : résolution canonique d'un
// access_token Microsoft BRUT depuis le MultiUserTokenStore (ADR 0023).
//
// Différence avec RefreshHaloTokensViaStoreFirst (cli_refresh.go) : ce helper
// s'arrête à l'access_token Microsoft (PAS d'Exchange Halo). Il sert les chemins
// qui ont besoin de l'access_token pour un usage NON-Halo — typiquement
// AcquireXSTSForRTA (Xbox Live : achievements, PeopleHub) — ou un Exchange
// ultérieur choisi par le caller (world-enrich).
//
// Source UNIQUE de la résolution « access_token » : avant ce helper, sync
// (post-sync achievements) et worldenrich dupliquaient cet ordre — et la copie
// sync ignorait purement le store, servant toujours un RT legacy sync_meta
// (incident prod 2026-07-12, 4 joueurs, engine_postsync_csr.go). Toute nouvelle
// résolution d'access_token DOIT passer par ici (garde-rail :
// internal/sync/no_legacy_source_used_test.go).
//
// ADR 0023 Phase 5 (2026-08-25) : les branches legacy (sync_meta DuckDB, env var
// variable d'environnement) et la télémétrie legacy_source_used associée sont
// supprimées — le store est la seule source.
package auth

import (
	"context"
	"fmt"
	"log/slog"
)

// ResolveMSAccessTokenStoreFirst résout un access_token Microsoft frais depuis le
// MultiUserTokenStore : OAuth refresh du RT persisté, rotation réécrite au store.
//
// Retourne ("", nil) si aucune source n'a produit de token SANS erreur (skip
// légitime). Retourne ("", err) en enveloppant la dernière erreur OAuth
// sous-jacente (ex. invalid_grant = RT révoqué / minté par un autre client) pour
// permettre au caller de diagnostiquer plutôt qu'un skip opaque.
//
// Si store == nil ou xuid == "", retourne ("", nil) : plus aucune source de repli.
func ResolveMSAccessTokenStoreFirst(
	ctx context.Context,
	provider TokenProvider,
	store *MultiUserTokenStore,
	xuid, gamertag string,
) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("auth: provider nil")
	}
	if store == nil || xuid == "" {
		slog.DebugContext(ctx, "auth: pas de store/xuid — aucun access_token résolvable",
			"gamertag", gamertag, "xuid", xuid)
		return "", nil
	}

	user, err := store.Load(xuid)
	if err != nil {
		// AU3 (revue 2026-07) : ne jamais AVALER l'échec de lecture du store — un
		// store illisible/corrompu doit être visible, sinon le skip est opaque.
		slog.ErrorContext(ctx, "auth: échec lecture store canonique — aucun access_token",
			"xuid", xuid, "gamertag", gamertag, "err", err)
		return "", nil
	}
	if user == nil || user.OAuthRefreshToken == "" {
		return "", nil
	}

	at, rotatedRT, rerr := provider.TryOAuthRefreshWithRotation(ctx, user.OAuthRefreshToken)
	if rerr != nil {
		return "", fmt.Errorf("auth: aucun access_token frais pour xuid(%s): %w", xuid, rerr)
	}
	if at == "" {
		return "", nil
	}
	if rotatedRT != "" && rotatedRT != user.OAuthRefreshToken {
		persistRotatedRT(ctx, store, xuid, rotatedRT)
	}
	return at, nil
}

// persistRotatedRT écrit le RT rotaté dans le store (retry 1x). Un échec
// persistant est LOGUÉ (jamais avalé) : sans le RT roté persisté, le prochain
// refresh relit un RT mort (invalid_grant) → chaîne auth du joueur cassée.
func persistRotatedRT(ctx context.Context, store *MultiUserTokenStore, xuid, rotatedRT string) {
	if err := store.UpdateOAuthRefreshToken(xuid, rotatedRT); err != nil {
		if err = store.UpdateOAuthRefreshToken(xuid, rotatedRT); err != nil {
			slog.ErrorContext(ctx, "auth: persistance du refresh token roté échouée — chaîne auth à risque au prochain refresh",
				"xuid", xuid, "err", err)
		}
	}
}
