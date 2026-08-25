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

// ErrHaloExchangeFailed marque un échec de l'échange Halo Waypoint : le refresh
// OAuth a bien rendu un access_token Microsoft, mais Waypoint n'a pas rendu de
// Spartan/Clearance. À NE PAS confondre avec un refresh token mort — ce cas ne
// doit jamais déclencher de marquage reauth_required ni de re-capture (ADR 0023).
var ErrHaloExchangeFailed = errors.New("auth: exchange Halo échoué")

// RefreshHaloTokensViaStoreFirst obtient des HaloTokens (Spartan + Clearance)
// depuis le MultiUserTokenStore (source unique ADR 0023) : OAuth refresh du RT
// persisté puis Exchange Halo.
//
// Les rotations OAuth sont systématiquement persistées dans le store.
//
// Retourne (nil, nil) UNIQUEMENT quand il n'y a rien à tenter : store/xuid
// absent, entrée introuvable, ou entrée sans refresh token. Tout échec RÉEL
// (refresh OAuth KO, exchange Halo KO) remonte une erreur — sans quoi les
// appelants affichent « aucun refresh token » sur un store pourtant sain
// (constat de la revue adversariale r1).
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
		if errors.Is(err, ErrUserTokensNotFound) {
			// Joueur jamais authentifié : rien à tenter, pas une anomalie.
			slog.InfoContext(ctx, "cli_auth: aucune entrée store pour ce joueur",
				"xuid", xuid, "gamertag", gamertag,
				"hint", "SSO Xbox ou `go run ./cmd/token-capture/ <GT>`")
			return nil, nil
		}
		// Store illisible/corrompu : anomalie franche, jamais avalée.
		slog.ErrorContext(ctx, "cli_auth: lecture store échouée",
			"xuid", xuid, "gamertag", gamertag, "err", err)
		return nil, fmt.Errorf("auth: lecture du store pour xuid(%s): %w", xuid, err)
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
	// Remonter la cause : sans elle, l'appelant conclut « aucun refresh token »
	// alors que le store en contient un (revue adversariale r1). Le seul (nil, nil)
	// légitime restant est l'entrée sans RT — rien à tenter, rien à diagnostiquer.
	if refreshErr != nil {
		return nil, refreshErr
	}
	return nil, nil
}

// RefreshFromStoreEntry tente un OAuth refresh (rotation persistée) depuis une
// entrée store DÉJÀ chargée.
//
//   - (result, nil) : succès.
//   - (nil, err) : le refresh OAuth a échoué — err porte la classe d'échec
//     (cf. ClassifyAuthError), ce qui permet au caller de ne marquer
//     reauth_required QUE pour un RT révoqué ; OU l'exchange Halo a échoué —
//     err enveloppe alors ErrHaloExchangeFailed, qui ne doit JAMAIS déclencher
//     de marquage reauth (le RT est vivant, c'est Waypoint qui a refusé).
//   - (nil, nil) : l'entrée n'a pas de refresh token, ou le refresh a rendu un
//     access_token vide sans erreur. Rien à diagnostiquer.
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
	if err != nil {
		// Revue adversariale r1 : ne JAMAIS avaler cet échec. Le refresh OAuth a
		// RÉUSSI (access_token Microsoft obtenu) — c'est Waypoint qui n'a pas rendu
		// de Spartan. Un (nil, nil) faisait afficher « aucun refresh token » aux
		// appelants : diagnostic faux qui pousse vers une re-capture, interdite
		// par l'ADR 0023.
		slog.ErrorContext(ctx, "cli_auth: exchange Halo échoué (refresh OAuth pourtant OK)",
			"xuid", xuid, "gamertag", user.Gamertag, "err", err)
		return nil, fmt.Errorf("%w pour xuid(%s): %v", ErrHaloExchangeFailed, xuid, err)
	}
	if result == nil {
		slog.ErrorContext(ctx, "cli_auth: exchange Halo sans erreur mais sans tokens",
			"xuid", xuid, "gamertag", user.Gamertag)
		return nil, fmt.Errorf("%w pour xuid(%s): réponse vide", ErrHaloExchangeFailed, xuid)
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
