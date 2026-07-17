// Package auth — access_token_store_first.go : résolution canonique d'un
// access_token Microsoft BRUT selon la priorité ADR 0023 (store-first).
//
// Différence avec RefreshHaloTokensViaStoreFirst (cli_refresh.go) : ce helper
// s'arrête à l'access_token Microsoft (PAS d'Exchange Halo). Il sert les chemins
// qui ont besoin de l'access_token pour un usage NON-Halo — typiquement
// AcquireXSTSForRTA (Xbox Live : achievements, PeopleHub) — ou un Exchange
// ultérieur choisi par le caller (world-enrich).
//
// Source UNIQUE de l'ordre de résolution « access_token store→legacy » : avant ce
// helper, sync (post-sync achievements) et worldenrich dupliquaient cet ordre —
// et la copie sync ignorait purement le store, servant toujours un RT legacy
// sync_meta et faussant la télémétrie legacy_source_used (incident prod
// 2026-07-12, 4 joueurs, engine_postsync_csr.go). Toute nouvelle résolution
// d'access_token DOIT passer par ici (garde-rail : pas de RecordLegacySourceUsed
// dans internal/sync — cf. engine_postsync_no_legacy_source_test.go).
//
// Invariant D1a (ADR 0023, prérequis D2) : la télémétrie legacy_source_used
// n'est émise QUE lorsque le store canonique n'a pas produit de token ET qu'une
// source legacy est réellement atteinte. En régime normal (store couvrant le
// joueur), le compteur reste à 0.
package auth

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// ResolveMSAccessTokenStoreFirst résout un access_token Microsoft frais :
//  1. MultiUserTokenStore (canonique) — MSAL silent puis OAuth refresh (rotation
//     persistée dans le store).
//  2. Sources legacy fournies par le caller (sync_meta DuckDB ou env var) — MSAL
//     silent puis OAuth refresh (rotation migrée vers le store si disponible).
//
// Retourne ("", nil) si aucune source n'a produit de token SANS erreur (skip
// légitime). Retourne ("", err) en enveloppant la dernière erreur OAuth
// sous-jacente (ex. invalid_grant = RT révoqué / minté par un autre client) pour
// permettre au caller de diagnostiquer plutôt qu'un skip opaque.
//
// Si store == nil ou xuid == "", saute directement au chemin legacy.
func ResolveMSAccessTokenStoreFirst(
	ctx context.Context,
	provider TokenProvider,
	store *MultiUserTokenStore,
	xuid, gamertag string,
	legacy LegacyAuthInputs,
) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("auth: provider nil")
	}
	r := &accessTokenResolver{provider: provider, xuid: xuid}

	// --- Source 1 : store canonique ---
	if store != nil && xuid != "" {
		user, err := store.Load(xuid)
		if err != nil {
			// AU3 (revue 2026-07) : ne plus AVALER l'échec de lecture du store. Sans ce
			// log, une bascule legacy causée par un store illisible/corrompu était
			// invisible → la télémétrie legacy_source_used (gate D2) devenait trompeuse
			// (une adoption legacy attribuée à une "vraie" absence de token store).
			slog.ErrorContext(ctx, "auth: échec lecture store canonique — bascule legacy (télémétrie legacy_source_used potentiellement trompeuse)",
				"xuid", xuid, "err", err)
		} else if user != nil {
			if at := r.attempt(ctx, user.MSALCacheJSON, user.OAuthRefreshToken, store); at != "" {
				return at, nil
			}
		}
	}

	// --- Source 2 : legacy (le store canonique n'a pas résolu) ---
	if at := r.attemptLegacy(ctx, store, gamertag, legacy); at != "" {
		return at, nil
	}

	if r.lastErr != nil {
		return "", fmt.Errorf("auth: aucun access_token frais pour xuid(%s): %w", xuid, r.lastErr)
	}
	return "", nil
}

// accessTokenResolver porte l'état d'une résolution store→legacy : la dernière
// erreur OAuth sous-jacente est mémorisée à travers les tentatives pour être
// surfacée au caller (diagnostic invalid_grant vs skip opaque).
type accessTokenResolver struct {
	provider TokenProvider
	xuid     string
	lastErr  error
}

// attempt : MSAL silent (si msal) puis OAuth refresh (si rt). persistTo non-nil →
// le RT rotaté est écrit dans le store pour le xuid (rotation ADR 0023).
func (r *accessTokenResolver) attempt(ctx context.Context, msal, rt string, persistTo *MultiUserTokenStore) string {
	if msal != "" {
		if at, err := r.provider.TrySilentRefresh(ctx, msal); err == nil && at != "" {
			return at
		} else if err != nil {
			r.lastErr = err
		}
	}
	if rt == "" {
		return ""
	}
	at, rotatedRT, err := r.provider.TryOAuthRefreshWithRotation(ctx, rt)
	if err != nil {
		r.lastErr = err
		return ""
	}
	if at == "" {
		return ""
	}
	if persistTo != nil && r.xuid != "" && rotatedRT != "" && rotatedRT != rt {
		persistRotatedRT(ctx, persistTo, r.xuid, rotatedRT)
	}
	return at
}

// attemptLegacy tente les résidus legacy (MSAL puis OAuth). La télémétrie de
// dépréciation n'est émise QU'ICI : atteindre ce point prouve que le store
// canonique n'a pas produit de token → dépendance legacy réelle (signal machine
// du gate D2). Le RT rotaté legacy est migré vers le store si disponible.
func (r *accessTokenResolver) attemptLegacy(ctx context.Context, store *MultiUserTokenStore, gamertag string, legacy LegacyAuthInputs) string {
	if legacy.MSALCache != "" {
		r.recordLegacy(ctx, observability.LegacySourceDuckDBMSAL, gamertag)
		if at := r.attempt(ctx, legacy.MSALCache, "", nil); at != "" {
			return at
		}
	}
	if legacy.OAuthRT == "" {
		return ""
	}
	src := observability.LegacySourceDuckDBOAuth
	if legacy.OAuthRTFromEnv {
		src = observability.LegacySourceEnvOAuth
	}
	r.recordLegacy(ctx, src, gamertag)
	// Persiste la rotation legacy dans le store → migre le RT vers le canonique (au
	// prochain refresh, le store résout et la télémétrie ne se déclenche plus).
	return r.attempt(ctx, "", legacy.OAuthRT, store)
}

// recordLegacy émet le WARN slog + incrémente le compteur expvar de dépréciation.
func (r *accessTokenResolver) recordLegacy(ctx context.Context, source, gamertag string) {
	observability.RecordLegacySourceUsed(source)
	slog.WarnContext(ctx, "legacy_source_used", "source", source,
		"gamertag", gamertag, "xuid", r.xuid, "deprecated_since", "ADR-0023")
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
