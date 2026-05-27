// Package api — registry_auth.go : factories services Halo auth-bound
// (HomeCtxWithAuth, SeasonPassCtxWithAuth) + helpers token refresh
// (buildHaloProvider, enrichWithHaloTokens, refreshTokensFromDB,
// oauthRefreshTokenForPlayer, AnyPlayerTokens, RefreshTokensForXUID).
// Découpé de registry.go (god-file split, refactor 2026-05-27).
package api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// HomeCtxWithAuth retourne un HomeService + contexte enrichi avec les HaloTokens du joueur.
// Si la session HTTP porte déjà des tokens, ils sont réutilisés.
// Sinon, tente un refresh silencieux depuis le cache MSAL stocké dans sync_meta.
// Un PersistSink est configuré pour la persistance fire-and-forget des données BP/challenges.
// Le HomeService créé est enregistré comme SessionNotifier pour ce joueur (TTL dynamique).
func (r *ServiceRegistry) HomeCtxWithAuth(ctx context.Context, slug string) (port.HomeService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)
	homeRepo := r.newHomeRepo(pdb)
	haloProvider := r.buildHaloProvider(pdb).WithTrackDefPersister(sink).WithItemDefPersister(sink)
	svc := service.NewHomeService(homeRepo).
		WithPersistSink(sink).
		WithCacheRepo(homeRepo).
		WithHaloProvider(haloProvider).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo))
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(svc))
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// SeasonPassCtxWithAuth retourne un SeasonPassService + contexte enrichi avec les HaloTokens.
// Réutilise HomeCtxWithAuth pour la résolution des tokens et le cacheRepo BP/challenges.
// Le HomeService créé est enregistré comme SessionNotifier pour ce joueur (TTL dynamique).
func (r *ServiceRegistry) SeasonPassCtxWithAuth(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, err
	}
	homeRepo := r.newHomeRepo(pdb)
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID)
	haloProvider := r.buildHaloProvider(pdb).WithTrackDefPersister(sink).WithItemDefPersister(sink)
	homeSvc := service.NewHomeService(homeRepo).
		WithPersistSink(sink).
		WithCacheRepo(homeRepo).
		WithHaloProvider(haloProvider).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo))
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(homeSvc))
	spRepo := duckdb.NewSeasonPassRepo(pdb)
	svc := service.NewSeasonPassService(spRepo, homeSvc, pdb.XUID, pdb.TitleSlug)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, nil
}

// buildHaloProvider construit un HaloProvider configuré avec le resolver unifié et le titre du joueur.
func (r *ServiceRegistry) buildHaloProvider(pdb *duckdb.PlayerDB) *halo.HaloProvider {
	titleSlug := title.DefaultSlug
	if pdb != nil && pdb.TitleSlug != "" {
		titleSlug = pdb.TitleSlug
	}
	return halo.DefaultHaloProvider.
		WithAssetResolver(r.assetResolver).
		WithTitleSlug(titleSlug)
}

// enrichWithHaloTokens injecte les HaloTokens dans le contexte si absents.
func (r *ServiceRegistry) enrichWithHaloTokens(ctx context.Context, pdb *duckdb.PlayerDB) context.Context {
	if ctxkeys.HaloTokens(ctx) != nil {
		return ctx // tokens déjà présents via session HTTP
	}
	xuid := pdb.XUID
	if cached := halo.GetCachedPlayerTokens(xuid); cached != nil {
		return ctxkeys.WithHaloAuth(ctx, cached, xuid)
	}
	result := r.refreshTokensFromDB(ctx, pdb, xuid)
	if result != nil {
		halo.SetCachedPlayerTokens(xuid, result.Tokens)
		return ctxkeys.WithHaloAuth(ctx, result.Tokens, xuid)
	}
	return ctx
}

// refreshTokensFromDB charge le cache MSAL ou le refresh_token OAuth v2 depuis sync_meta,
// puis tente un refresh silencieux pour obtenir les tokens Halo.
// Ordre :
//  1. MSAL cache (sync_meta.msal_token_cache) → TrySilentRefresh
//  2. OAuth v2 refresh_token (sync_meta.oauth_refresh_token) → TryOAuthRefreshWithRotation + persist
//  3. OAuth v2 refresh_token (env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG>) → bootstrap uniquement
//
// DuckDB est prioritaire sur l'env var. Au boot, le Pool lit l'env var, appelle Microsoft,
// et persiste le RT rotaté en DuckDB via onRotated. Si ce handler HTTP lit l'env var
// en premier (original déjà consommé par le Pool), il obtient invalid_grant.
// Avec DuckDB en premier, il utilise le RT rotaté que le Pool vient de sauvegarder.
func (r *ServiceRegistry) refreshTokensFromDB(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	// --- Chemin 1 : MSAL cache ---
	cacheJSON, err := duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	if err == nil && cacheJSON != "" {
		accessToken, err := r.provider.TrySilentRefresh(ctx, cacheJSON)
		if err == nil && accessToken != "" {
			result, err := r.provider.Exchange(ctx, accessToken)
			if err == nil && result != nil {
				slog.DebugContext(ctx, "halo_auth: tokens obtenus via MSAL cache", "xuid", xuid)
				return result
			}
			slog.WarnContext(ctx, "halo_auth: échange access_token échoué (MSAL)", "xuid", xuid, "err", err)
		} else if err != nil {
			slog.WarnContext(ctx, "halo_auth: MSAL silent refresh échoué", "xuid", xuid, "err", err)
		}
	}

	// --- Chemin 2 : refresh_token OAuth v2 ---
	// DuckDB d'abord : le Pool persiste le RT rotaté en DuckDB au boot via onRotated.
	// L'env var est un seed bootstrap (one-shot) : une fois le RT en DuckDB, l'env var
	// est dépassée et causerait invalid_grant si lue en premier.
	refreshToken, _ := duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)
	source := "duckdb"
	if refreshToken == "" {
		refreshToken = oauthRefreshTokenForPlayer(pdb.Gamertag)
		source = "env_var"
	}
	if refreshToken != "" {
		accessToken, rotatedRT, err := r.provider.TryOAuthRefreshWithRotation(ctx, refreshToken)
		if err == nil && accessToken != "" {
			if rotatedRT != "" && rotatedRT != refreshToken {
				if werr := duckdb.WriteOAuthRefreshToken(ctx, pdb.Player, rotatedRT); werr != nil {
					slog.WarnContext(ctx, "halo_auth: persistance RT rotaté échouée", "xuid", xuid, "err", werr)
				}
			}
			result, err := r.provider.Exchange(ctx, accessToken)
			if err == nil && result != nil {
				slog.DebugContext(ctx, "halo_auth: tokens obtenus via OAuth v2 refresh", "xuid", xuid, "source", source)
				return result
			}
			slog.WarnContext(ctx, "halo_auth: échange access_token échoué (OAuth v2)", "xuid", xuid, "err", err)
		} else if err != nil {
			slog.WarnContext(ctx, "halo_auth: OAuth v2 refresh échoué", "xuid", xuid, "err", err)
		}
	}

	slog.WarnContext(ctx, "halo_auth: aucun token disponible pour le joueur", "xuid", xuid, "gamertag", pdb.Gamertag)
	return nil
}

// oauthRefreshTokenForPlayer retourne le refresh_token OAuth v2 depuis l'environnement
// pour un gamertag donné.
// Convention : SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG_MAJUSCULES_SANS_ESPACES>
// Exemple : gamertag "JGtm" → SPNKR_OAUTH_REFRESH_TOKEN_JGTM
func oauthRefreshTokenForPlayer(gamertag string) string {
	if gamertag == "" {
		return ""
	}
	// Normalisation : majuscules, espaces/tirets/points → underscore.
	key := strings.ToUpper(gamertag)
	key = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '.' {
			return '_'
		}
		return r
	}, key)
	return os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_" + key)
}

// AnyPlayerTokens retourne les tokens Halo du premier joueur disponible dans le pool.
// Utilisé par les handlers d'assets qui ont besoin de tokens mais ne sont pas
// rattachés à un joueur spécifique.
func (r *ServiceRegistry) AnyPlayerTokens(ctx context.Context) (*domain.HaloTokens, error) {
	var tokens *domain.HaloTokens
	duckdb.IteratePool(func(pdb *duckdb.PlayerDB) bool {
		t, err := r.RefreshTokensForXUID(ctx, pdb.XUID)
		if err == nil && t != nil {
			tokens = t
			return false // stop
		}
		return true // continuer avec le joueur suivant
	})
	if tokens == nil {
		return nil, fmt.Errorf("aucun token Halo disponible")
	}
	return tokens, nil
}

// RefreshTokensForXUID tente un refresh silencieux pour le joueur identifié par son XUID.
// Recherche le PlayerDB dans le pool, puis tente MSAL ou OAuth v2 refresh.
// Met à jour le cache process si le refresh réussit.
// Appelé par PlayerLiveRefresher quand le cache process est expiré.
func (r *ServiceRegistry) RefreshTokensForXUID(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
	if cached := halo.GetCachedPlayerTokens(xuid); cached != nil {
		return cached, nil
	}
	var pdb *duckdb.PlayerDB
	duckdb.IteratePool(func(p *duckdb.PlayerDB) bool {
		if p.XUID == xuid {
			pdb = p
			return false // stop
		}
		return true
	})
	if pdb == nil {
		return nil, fmt.Errorf("halo_auth: joueur xuid=%s introuvable dans le pool", xuid)
	}
	result := r.refreshTokensFromDB(ctx, pdb, xuid)
	if result == nil {
		return nil, fmt.Errorf("halo_auth: refresh impossible pour xuid=%s", xuid)
	}
	halo.SetCachedPlayerTokens(xuid, result.Tokens)
	return result.Tokens, nil
}
