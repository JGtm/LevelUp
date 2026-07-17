// Package api — registry_auth.go : factories services Halo auth-bound
// (HomeCtxWithAuth, SeasonPassCtxWithAuth) + helpers token refresh
// (buildHaloProvider, enrichWithHaloTokens, refreshTokensFromDB,
// oauthRefreshTokenForPlayer, AnyPlayerTokens, RefreshTokensForXUID).
// Découpé de registry.go (god-file split, refactor 2026-05-27).
package wire

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
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
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID, pdb.TitleSlug)
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
		WithSquadSessionTeammates(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo)).
		WithSkillBadgeResolver(skillBadgeResolverFor(pdb.TitleSlug)).
		WithDemoMode(r.cfg.DemoMode)
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(svc))
	enriched := forcePageIdentityXUID(r.enrichWithHaloTokens(ctx, pdb), pdb.XUID)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// forcePageIdentityXUID garantit que le contexte porte le xuid du joueur de la
// PAGE (pdb.XUID) — jamais celui du compte connecté. HomeService.GetSpartanIdentity
// résout l'identité Spartan via ctxkeys.HaloXUID ; deux cas imposent ce forçage :
//
//   - Démo / non authentifié : enrichWithHaloTokens ne pose aucun xuid → les
//     lectures xuid-filtrées ciblent "" (bannière/emblème introuvables).
//   - Consultation d'un AUTRE joueur (admin ou membre de groupe, ADR 0029) :
//     depuis que SISU complète l'identité de session (train 2026-07-15), la
//     session porte le xuid du COMPTE CONNECTÉ, qu'enrichWithHaloTokens conserve
//     tel quel (token de session frais → réutilisé). Sans ce forçage, TOUTES les
//     pages home affichaient alors la MÊME identité Spartan — celle du compte
//     appelant (régression 2026-07-16, tous titres confondus).
//
// Les tokens (de session ou re-mintés) restent utilisables tels quels : les
// endpoints careerranks/customization ciblent xuid(<pdb>) dans l'URL et ne
// dérivent jamais l'identité du token porteur. Le contrôle d'accès (403 slug
// étranger) est appliqué en amont, indépendamment de ce forçage.
//
// Consommateurs OWNERSHIP-SCOPED (SeasonPassCtxWithAuth : BP/défis) : le forçage
// aligne le SUJET (xuid dans l'URL players/xuid(<sujet>)/{decks,rewardtracks}) sur
// le xuid persisté par le sink → jamais d'écriture des données d'un porteur étranger
// sous le xuid de la page ; porteur≠page → 403 upstream → fallback cache DB du sujet.
func forcePageIdentityXUID(ctx context.Context, pageXUID string) context.Context {
	if pageXUID == "" || ctxkeys.HaloXUID(ctx) == pageXUID {
		return ctx
	}
	return ctxkeys.WithHaloXUID(ctx, pageXUID)
}

// SeasonPassCtxWithAuth retourne un SeasonPassService + contexte enrichi avec les HaloTokens.
// Réutilise HomeCtxWithAuth pour la résolution des tokens et le cacheRepo BP/challenges.
// Le HomeService créé est enregistré comme SessionNotifier pour ce joueur (TTL dynamique).
//
// Le contexte porte le xuid de la PAGE (forcePageIdentityXUID) — même invariant que
// HomeCtxWithAuth. Les fetches BP/défis (GetBattlePass/GetChallenges) ciblent
// players/xuid(<sujet>)/rewardtracks|decks où <sujet> = ctxkeys.HaloXUID, et les
// snapshots sont persistés sous pdb.XUID (sink). Ces endpoints economy sont
// OWNERSHIP-SCOPED (fetchables uniquement pour le porteur du token) : sans forçage, un
// compte connecté consultant la page d'un AUTRE joueur fetchait SES défis puis les
// persistait sous le xuid de la page (pollution, 4e occurrence du bug PR #63). Avec le
// forçage, le sujet == page : porteur=page → fetch/persist corrects ; porteur≠page →
// fetch xuid(page) avec un token étranger → 403 → fallback cache DB du sujet, aucune
// écriture croisée.
func (r *ServiceRegistry) SeasonPassCtxWithAuth(ctx context.Context, slug string) (port.SeasonPassService, context.Context, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, err
	}
	homeRepo := r.newHomeRepo(pdb)
	sink := duckdb.NewPersistSink(pdb.Metadata.Path(), pdb.Player.Path(), pdb.XUID, pdb.TitleSlug)
	haloProvider := r.buildHaloProvider(pdb).WithTrackDefPersister(sink).WithItemDefPersister(sink)
	homeSvc := service.NewHomeService(homeRepo).
		WithPersistSink(sink).
		WithCacheRepo(homeRepo).
		WithHaloProvider(haloProvider).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo)).
		WithDemoMode(r.cfg.DemoMode)
	r.notifiers.Store(pdb.XUID, port.SessionNotifier(homeSvc))
	spRepo := duckdb.NewSeasonPassRepo(pdb)
	svc := service.NewSeasonPassService(spRepo, homeSvc, pdb.XUID, pdb.TitleSlug)
	enriched := forcePageIdentityXUID(r.enrichWithHaloTokens(ctx, pdb), pdb.XUID)
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

// enrichWithHaloTokens injecte des HaloTokens FRAIS et DÉTERMINISTES dans le contexte.
//
// Garantie expiry-aware STRICTE : on ne réutilise un token de session que si on peut PROUVER
// qu'il est encore valide (expiry CONNUE et fraîche, `TokensFreshStrict`). Un token de session
// d'expiry inconnue (session créée avant la capture de SpartanExpiresAt) est parfois réellement
// périmé → il provoquait un 401 INTERMITTENT sur les chemins sans filet (rang carrière). On le
// re-résout donc systématiquement via halo.ResolveFreshPlayerTokens (cache expiry-aware + re-mint
// singleflight, expiry réelle). Sur échec de re-mint (ex. SSO-only hors pool) → on garde le token
// de session existant (mieux que rien) : dégradation, pas de régression.
func (r *ServiceRegistry) enrichWithHaloTokens(ctx context.Context, pdb *duckdb.PlayerDB) context.Context {
	if sess := ctxkeys.HaloTokens(ctx); halo.TokensFreshStrict(sess) {
		return ctx // token de session présent, expiry CONNUE et encore valide → on le garde
	}
	xuid := pdb.XUID
	tokens, err := halo.ResolveFreshPlayerTokens(ctx, xuid)
	if err != nil || tokens == nil {
		if err != nil {
			slog.DebugContext(ctx, "halo_auth: résolution token live impossible — fallback token de session", "xuid", xuid, "err", err)
		}
		return ctx
	}
	return ctxkeys.WithHaloAuth(ctx, tokens, xuid)
}

// WireGlobalTokenRefresher branche le hook global de re-dérivation des tokens
// (halo.ResolveFreshPlayerTokens) sur le minting registry. À appeler une fois au boot,
// après construction du registry. Idempotent.
func (r *ServiceRegistry) WireGlobalTokenRefresher() {
	halo.SetPlayerTokenRefresher(r.mintFreshTokensForXUID)
}

// mintFreshTokensForXUID MINTE des tokens Halo frais pour un joueur (résolution pool +
// refresh MSAL/OAuth), SANS lire ni écrire le cache process — c'est le hook branché sur
// halo.SetPlayerTokenRefresher (ResolveFreshPlayerTokens gère cache + singleflight).
func (r *ServiceRegistry) mintFreshTokensForXUID(ctx context.Context, xuid string) (*domain.HaloTokens, error) {
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
	return result.Tokens, nil
}

// refreshTokensFromDB obtient des tokens Halo en exécutant un refresh OAuth/MSAL.
// Source des credentials, par ordre de priorité :
//  1. MultiUserTokenStore (ADR 0023 — source unique post-migration) :
//     a. MSALCacheJSON → TrySilentRefresh
//     b. OAuthRefreshToken → TryOAuthRefreshWithRotation (+ écriture rotation au store)
//  2. Fallback legacy (DEPRECATED — log warn à chaque hit, à supprimer Phase 5) :
//     a. sync_meta.msal_token_cache → TrySilentRefresh
//     b. sync_meta.oauth_refresh_token → TryOAuthRefreshWithRotation
//     c. env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> → TryOAuthRefreshWithRotation
//
// La Phase 2 migration (boot-time) garantit que le store contient les valeurs des
// sources legacy si elles existaient. Le chemin legacy ne devrait être atteint
// que sur des installs non encore migrées ou en cas de corruption du store.
func (r *ServiceRegistry) refreshTokensFromDB(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	// --- Source 1 : MultiUserTokenStore (canonique) ---
	if r.authStore != nil && xuid != "" {
		if result := r.tryRefreshFromAuthStore(ctx, pdb, xuid); result != nil {
			return result
		}
	}

	// --- Source 2 : sync_meta DuckDB + env var (DEPRECATED) ---
	if result := r.tryRefreshFromLegacy(ctx, pdb, xuid); result != nil {
		return result
	}

	slog.WarnContext(ctx, "halo_auth: aucun token disponible pour le joueur", "xuid", xuid, "gamertag", pdb.Gamertag)
	return nil
}

// tryRefreshFromAuthStore tente un refresh via le MultiUserTokenStore.
// MSAL cache prioritaire (silent refresh), puis OAuth RT avec persistance rotation.
// Au succès, efface l'éventuel flag reauth_required (auto-guérison de la bannière
// de reconnexion : un refresh par-joueur réussi prouve que le RT est vivant).
func (r *ServiceRegistry) tryRefreshFromAuthStore(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	user, err := r.authStore.Load(xuid)
	if err != nil || user == nil {
		return nil
	}
	// Cascade store MSAL→OAuth (rotation persistée) déléguée à la source unique
	// auth.RefreshFromStoreEntry (K1b dédup). L'erreur classifiée est LOGUÉE (jamais
	// avalée) mais N'entraîne PAS de marquage reauth_required : le chemin serveur
	// haute-fréquence conserve sa politique historique (clear-on-success uniquement ;
	// c'est le chemin CLI de cli_refresh.go qui marque un RT révoqué).
	result, refreshErr := auth.RefreshFromStoreEntry(ctx, r.provider, r.authStore, xuid, user)
	if refreshErr != nil {
		slog.WarnContext(ctx, "halo_auth: refresh store échoué", "xuid", xuid, "err", refreshErr)
	}
	if result != nil {
		// Refresh OK → l'éventuel flag reauth_required est obsolète : on l'efface
		// (auto-guérison de la bannière, symétrique du clear côté CLI). Idempotent,
		// best-effort, non bloquant. Un xuid au RT réellement mort n'atteint jamais
		// ce point → son flag reste posé (pas de faux clear).
		if cerr := r.authStore.ClearReauthRequired(xuid); cerr != nil {
			slog.WarnContext(ctx, "halo_auth: clear reauth_required échoué (non-bloquant)", "xuid", xuid, "err", cerr)
		}
	}
	return result
}

// tryRefreshFromLegacy tente un refresh via sync_meta DuckDB + env var.
// DEPRECATED : sera supprimé Phase 5. Logue chaque hit pour identifier les
// installs pas encore migrées vers le store.
func (r *ServiceRegistry) tryRefreshFromLegacy(ctx context.Context, pdb *duckdb.PlayerDB, xuid string) *auth.ExchangeResult {
	cacheJSON, err := duckdb.ReadMSALCacheJSON(ctx, pdb.Player)
	if err == nil && cacheJSON != "" {
		slog.WarnContext(ctx, "halo_auth: legacy source MSAL utilisée (sync_meta DuckDB) — à migrer",
			"xuid", xuid, "gamertag", pdb.Gamertag, "deprecated_since", "ADR-0023")
		observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBMSAL)
		if accessToken, err := r.provider.TrySilentRefresh(ctx, cacheJSON); err == nil && accessToken != "" {
			if result, err := r.provider.Exchange(ctx, accessToken); err == nil && result != nil {
				slog.DebugContext(ctx, "halo_auth: tokens obtenus via MSAL cache (legacy)", "xuid", xuid)
				return result
			}
		}
	}

	refreshToken, _ := duckdb.ReadOAuthRefreshToken(ctx, pdb.Player)
	source := "legacy_duckdb"
	if refreshToken == "" {
		refreshToken = oauthRefreshTokenForPlayer(pdb.Gamertag)
		source = "legacy_env_var"
	}
	if refreshToken == "" {
		return nil
	}

	slog.WarnContext(ctx, "halo_auth: legacy source RT utilisée — à migrer",
		"xuid", xuid, "gamertag", pdb.Gamertag, "source", source, "deprecated_since", "ADR-0023")
	if source == "legacy_env_var" {
		observability.RecordLegacySourceUsed(observability.LegacySourceEnvOAuth)
	} else {
		observability.RecordLegacySourceUsed(observability.LegacySourceDuckDBOAuth)
	}

	accessToken, rotatedRT, err := r.provider.TryOAuthRefreshWithRotation(ctx, refreshToken)
	if err != nil || accessToken == "" {
		if err != nil {
			slog.WarnContext(ctx, "halo_auth: OAuth refresh échoué (legacy)", "xuid", xuid, "err", err)
		}
		return nil
	}

	// Persistance rotation : store si disponible, sinon DuckDB (legacy).
	if rotatedRT != "" && rotatedRT != refreshToken {
		if r.authStore != nil && xuid != "" {
			if werr := r.authStore.UpdateOAuthRefreshToken(xuid, rotatedRT); werr != nil {
				slog.WarnContext(ctx, "halo_auth: persistance RT rotaté store échouée (legacy refresh)", "xuid", xuid, "err", werr)
			}
		} else if werr := duckdb.WriteOAuthRefreshToken(ctx, pdb.Player, rotatedRT); werr != nil {
			slog.WarnContext(ctx, "halo_auth: persistance RT rotaté DuckDB échouée", "xuid", xuid, "err", werr)
		}
	}

	result, err := r.provider.Exchange(ctx, accessToken)
	if err != nil || result == nil {
		if err != nil {
			slog.WarnContext(ctx, "halo_auth: Exchange échoué (legacy)", "xuid", xuid, "err", err)
		}
		return nil
	}
	slog.DebugContext(ctx, "halo_auth: tokens obtenus via OAuth refresh (legacy)", "xuid", xuid, "source", source)
	return result
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
	tokens, err := r.mintFreshTokensForXUID(ctx, xuid)
	if err != nil {
		return nil, err
	}
	halo.SetCachedPlayerTokens(xuid, tokens)
	return tokens, nil
}
