// Package api — registry_pages.go : factories services Page (Filters,
// MatchView, Engagement, Sessions, SessionCompare, SessionPage, Stats,
// Timeseries) et factories XxxCtx (Citations, Explorer, Home, MatchHistory,
// Squad, SquadV2, MatchExclusion, Teammates, Compare, Leaderboard, Synthesis)
// + helpers proches (newHomeRepo, playerMatchesAdapterFor,
// friendGamertagsResolver, buildFriendsExtrasResolver). Découpé de
// registry.go (god-file split, refactor 2026-05-27).
package api

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	sync_pkg "levelup/go-api/internal/sync"
)

// Filters retourne un FiltersService pour le joueur.
//
// Injecte le catalog unifié des saisons (TOML + DB live + lazy fetch) pour
// alimenter les SeasonCounts du folding SaisonPill. Si le catalog n'est
// pas câblé OU si le titre n'a aucune saison résolue → aucun SeasonCount
// renvoyé (dégradation gracieuse, le frontend affiche les saisons sans
// folding).
func (r *ServiceRegistry) Filters(ctx context.Context, slug string) (port.FiltersService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewFiltersService(duckdb.NewFiltersRepo(pdb))
	if r.seasonsCatalog != nil {
		// Bug fix 2026-05-08 : passer pdb.TitleSlug (le titre du joueur, ex
		// "halo_infinite") et NON le `slug` paramètre qui est le **player
		// slug** URL (ex "JGtm"). Avant, seasons_catalog recevait "JGtm"
		// comme titleID → fetch live échouait toujours sur fallback TOML
		// (logs : `seasons_catalog: fetch live échec titleSlug=JGtm`).
		svc = svc.WithSeasonsCatalog(pdb.TitleSlug, r.seasonsCatalog)
	}
	return svc, nil
}

// MatchView retourne un MatchViewService pour le joueur.
//
// Phase C+ multi-titres : injecte le DataAdapter HI pour permettre une
// future bascule LoadMatchDetail (le service utilise le hook WithDataAdapter
// pour préparer la migration sans la déclencher tant que canonical.MatchDetail
// ne couvre pas la totalité du payload Match View).
func (r *ServiceRegistry) MatchView(ctx context.Context, slug string) (port.MatchViewService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewMatchViewService(duckdb.NewMatchViewRepo(pdb, pdb.XUID), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	svc = svc.WithCitationsRepo(duckdb.NewCitationsRepo(pdb)).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithAssetURL(r.assetURLFor(pdb.TitleSlug)).
		WithTitleSlug(pdb.TitleSlug).
		WithMetadataRepo(duckdb.NewMetadataRepo(pdb))
	if loader := r.buildFriendsExtrasResolver(pdb); loader != nil {
		svc = svc.WithFriendsExtras(loader)
	}
	return svc, nil
}

// buildFriendsExtrasResolver construit un loader d'extras per-friend pour le
// panneau d'expander scoreboard (Match View). Lookup xuid → (titleSlug,
// gamertag) depuis cfg.LoadPlayers, puis ouverture lazy de la player DB de
// l'ami via le pool DuckDB (cached). Retourne nil si cfg indisponible (tests).
func (r *ServiceRegistry) buildFriendsExtrasResolver(mainPDB *duckdb.PlayerDB) port.FriendsExtrasResolver {
	if r.cfg == nil {
		return nil
	}
	players, err := r.cfg.LoadPlayers(mainPDB.TitleSlug)
	if err != nil {
		slog.Warn("friends_extras: load players failed (resolver disabled)",
			"titleSlug", mainPDB.TitleSlug, "err", err)
		return nil
	}
	if len(players) == 0 {
		return nil
	}
	friendsByXUID := make(map[string]service.FriendProfile, len(players))
	for _, p := range players {
		if p.XUID == "" || p.XUID == mainPDB.XUID {
			continue
		}
		friendsByXUID[p.XUID] = service.FriendProfile{
			XUID:      p.XUID,
			Gamertag:  p.Gamertag,
			TitleSlug: mainPDB.TitleSlug,
		}
	}
	if len(friendsByXUID) == 0 {
		return nil
	}
	cfg := r.cfg
	opener := func(ctx context.Context, titleSlug, gamertag string) (service.FriendMatchExtrasRepo, error) {
		// Trouve le slug du joueur depuis son gamertag (cfg.LoadPlayers).
		ps, lerr := cfg.LoadPlayers(titleSlug)
		if lerr != nil {
			return nil, lerr
		}
		var slug string
		for i := range ps {
			if ps[i].Gamertag == gamertag {
				slug = ps[i].PlayerSlug
				break
			}
		}
		if slug == "" {
			return nil, fmt.Errorf("friends_extras: player not found gamertag=%q title=%q", gamertag, titleSlug)
		}
		pdb, perr := config.ResolvePlayer(ctx, cfg, slug, titleSlug)
		if perr != nil {
			return nil, perr
		}
		return duckdb.NewMatchViewRepo(pdb, pdb.XUID), nil
	}
	return service.NewFriendsExtrasResolver(friendsByXUID, opener, r.assetURLFor(mainPDB.TitleSlug))
}

// Engagement retourne un PlayerEngagementService pour le joueur.
// (Phase 4 plan engagement — endpoint /matches/{id}/engagement et /engagement_profile).
//
// Le PlayerMatchesRepo est injecte pour que GetTimeseries (POST
// /engagement/timeseries) puisse honorer le FilterContextInput de la page
// Timeseries Mock 11, comme POST /pages/timeseries.
func (r *ServiceRegistry) Engagement(ctx context.Context, slug string) (*service.PlayerEngagementService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewEngagementScoreRepo(pdb)
	return service.NewPlayerEngagementService(repo, pdb.XUID, pdb.Gamertag).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug), nil
}

// Sessions retourne un SessionsService pour le joueur.
func (r *ServiceRegistry) Sessions(ctx context.Context, slug string) (port.SessionsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewSessionsService(duckdb.NewSessionsRepo(pdb)), nil
}

// playerMatchesAdapterFor construit un adapter PlayerMatchesRepository pour le
// joueur (pdb). P4.3 finale : permet aux services match-rows de consommer
// canonical exclusivement (legacy fallback path supprimé).
func (r *ServiceRegistry) playerMatchesAdapterFor(pdb *duckdb.PlayerDB) port.PlayerMatchesRepository {
	pmRepo := duckdb.NewPlayerMatchesRepo(pdb)
	return duckdb.NewPlayerMatchesAdapter(pmRepo, pdb.TitleSlug, pdb.Gamertag)
}

// SessionCompare retourne un SessionCompareService pour le joueur.
func (r *ServiceRegistry) SessionCompare(ctx context.Context, slug string) (port.SessionCompareService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	sessionsRepo := duckdb.NewSessionsRepo(pdb)
	statsRepo := duckdb.NewStatsRepo(pdb)
	svc := service.NewSessionCompareService(sessionsRepo, statsRepo).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, nil
}

// SessionPage retourne un SessionPageService pour le joueur.
func (r *ServiceRegistry) SessionPage(ctx context.Context, slug string) (port.SessionPageService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewSessionPageService(duckdb.NewStatsRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, nil
}

// Stats retourne un StatsService pour le joueur.
func (r *ServiceRegistry) Stats(ctx context.Context, slug string) (port.StatsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return service.NewStatsService(duckdb.NewStatsRepo(pdb)).
		WithTitleSlug(pdb.TitleSlug).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.Gamertag), nil
}

// Timeseries retourne un TimeseriesService pour le joueur.
//
// Phase C+ multi-titres : injecte le DataAdapter HI pour permettre une
// future bascule LoadTimeseries.
func (r *ServiceRegistry) Timeseries(ctx context.Context, slug string) (port.TimeseriesService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewTimeseriesService(duckdb.NewStatsRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb)).
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, nil
}

// ---------------------------------------------------------------------------
// Factories avec contexte joueur (service + XUID + Gamertag)
// Pour les handlers qui ont besoin des identifiants joueur dans les appels.
// Signature : func(ctx, slug) → (service, xuid, gamertag, error)
// ---------------------------------------------------------------------------

// CitationsCtx retourne un CitationsService + identifiants joueur.
func (r *ServiceRegistry) CitationsCtx(ctx context.Context, slug string) (port.CitationsService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewCitationsService(duckdb.NewCitationsRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ExplorerCtxWithAuth retourne un ExplorerService + contexte enrichi avec les
// HaloTokens du propriétaire de la page (résolus depuis le store, ADR 0023).
//
// L'encart "Profil joueur cible" est local-first : identité lue depuis la DB
// locale de la cible (si joueur suivi), sample stats calculés localement. Seule
// la carrière agrégée (servicerecord) est un fetch live → nécessite des tokens
// dans le contexte (d'où enrichWithHaloTokens, même pattern que HomeCtxWithAuth
// et Compare). Aucune privacy n'est fetchée (bruit sans valeur).
func (r *ServiceRegistry) ExplorerCtxWithAuth(ctx context.Context, slug string) (port.ExplorerService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	svc := service.NewExplorerService(duckdb.NewExplorerRepo(pdb, pdb.XUID), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	// localIdentity : identité résolue 100% local (DB de la cible si suivie).
	// r.remoteStats : stats carrière remote (servicerecord) décorées d'un cache
	// TTL 5 min + singleflight → réouvrir la même cible est instantané.
	svc = svc.WithTargetProfileProviders(
		r.newExplorerLocalIdentityResolver(pdb.TitleSlug),
		r.remoteStats,
		pdb.TitleSlug,
	)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// newExplorerLocalIdentityResolver construit le résolveur d'identité local de
// l'encart Explorer : si le gamertag cible correspond à un joueur suivi
// (db_profiles), on ouvre SA player DB et on lit son identité Spartan
// (rang/emblem/skill peaks). Sinon nil (un adversaire n'a pas d'identité
// publiée — aucun fetch live). resolveByGT est pool-cached.
func (r *ServiceRegistry) newExplorerLocalIdentityResolver(titleSlug string) service.ExplorerLocalIdentityResolver {
	return service.LocalIdentityResolverFunc(func(ctx context.Context, targetGamertag string) *domain.HomeSpartanIdentityRow {
		if r.resolveByGT == nil || targetGamertag == "" {
			return nil
		}
		tpdb, err := r.resolveByGT(ctx, titleSlug, targetGamertag)
		if err != nil || tpdb == nil {
			return nil // cible non suivie localement → pas d'identité
		}
		row, lerr := r.newHomeRepo(tpdb).LoadSpartanIdentity(ctx)
		if lerr != nil {
			slog.WarnContext(ctx, "explorer_local_identity_failed",
				"gamertag", targetGamertag, "err", lerr)
			return nil
		}
		return row
	})
}

// HomeCtx retourne un HomeService + identifiants joueur.
func (r *ServiceRegistry) HomeCtx(ctx context.Context, slug string) (port.HomeService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	homeRepo := r.newHomeRepo(pdb)
	svc := service.NewHomeService(homeRepo).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// newHomeRepo construit un HomeRepo avec l'AssetURLAdapter du titre câblé
// quand disponible (fallback static FS pour les images map non encore
// peuplées dans map_images_registry — évite la dépendance à
// cmd/migrate-static-maps à chaque ajout de fichier static). Dégradation
// gracieuse : sans titleResolver ou sans adapter, le HomeRepo reste nu et
// ne fait que le lookup registry.
func (r *ServiceRegistry) newHomeRepo(pdb *duckdb.PlayerDB) *duckdb.HomeRepo {
	repo := duckdb.NewHomeRepo(pdb)
	// Phase 6 du plan CSR : injection du repo thresholds + saison courante.
	// Sans cette injection, le seuil par défaut (5) est utilisé partout, ce qui
	// est juste pour les saisons S3+ mais incorrect pour les historiques S1-S2.
	if pdb.Metadata != nil {
		csrSeasonID := ""
		if r.cfg != nil {
			csrSeasonID = r.cfg.CurrentCSRSeasonID
		}
		repo = repo.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata), csrSeasonID)
	}
	if r.titleResolver == nil {
		return repo
	}
	adapter, err := r.titleResolver.AssetURL(pdb.TitleSlug)
	if err != nil || adapter == nil {
		return repo
	}
	return repo.WithAssetURL(adapter)
}

// MatchHistoryCtx retourne un MatchHistoryService + identifiants joueur.
func (r *ServiceRegistry) MatchHistoryCtx(ctx context.Context, slug string) (port.MatchHistoryService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewMatchHistoryService(duckdb.NewMatchHistoryRepo(pdb), pdb.Gamertag).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	if pdb.Metadata != nil {
		// PR2 placement X/Y : résolveur season_id → threshold (5 ou 10 selon
		// saison CSR). Fallback à 5 si absent. Cf. csr_thresholds_repo.go.
		svc = svc.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata).Get)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadCtx retourne un SquadService + identifiants joueur.
func (r *ServiceRegistry) SquadCtx(ctx context.Context, slug string) (port.SquadService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSquadService(duckdb.NewSquadRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadV2Ctx retourne un SquadV2Service + identifiants joueur. Le service
// utilise un loader production qui résout (titleSlug, gamertag) → PlayerDB
// via le pool global, ce qui permet de charger les matchs des coéquipiers
// sans avoir leur player_slug — ils sont sélectionnés par gamertag dans l'UI.
func (r *ServiceRegistry) SquadV2Ctx(ctx context.Context, slug string) (port.SquadV2Service, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	loader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	// Le loader resout les DBs `shared` (events / weapons / medals) via le main
	// player ; on lui propage le gamertag de la session courante (chunk S11).
	loader.SetDefaultGamertag(pdb.Gamertag)
	svc := service.NewSquadServiceV2(loader)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// MatchExclusion retourne un MatchExclusionService pour le joueur, câblé sur
// un MatchRecomputer (recompute global perf_score + LUSR après chaque
// (dé)exclusion). metadataDBPath peut être vide → le recompute fonctionne
// sans bonus médailles.
func (r *ServiceRegistry) MatchExclusion(ctx context.Context, slug string) (port.MatchExclusionService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	metadataPath := ""
	if pdb.Metadata != nil {
		metadataPath = pdb.Metadata.Path()
	}
	// Sprint B1 commit 13b : pdb.SharedDBPath() survit en mode B-swap où
	// pdb.Shared peut être nil. Provider passé pour coordonner avec readers.
	var provider sharedprovider.Provider
	if r.cfg != nil {
		provider = r.cfg.SharedProvider
	}
	recomputer := sync_pkg.NewMatchRecomputer(
		pdb.Player.Path(),
		pdb.SharedDBPath(),
		metadataPath,
		pdb.XUID,
		pdb.Gamertag,
		provider,
	)
	return service.NewMatchExclusionService(
		duckdb.NewMatchExclusionRepo(pdb),
		recomputer,
	), nil
}

// TeammatesCtx retourne un TeammatesService + identifiants joueur.
//
// Le resolver friend_gamertags est branché sur r.settingsStore quand le
// store est attaché (cf. WithSettingsStore). Sans store → comportement
// legacy : top dropdown brut sans filtre amis.
func (r *ServiceRegistry) TeammatesCtx(ctx context.Context, slug string) (port.TeammatesService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	// SessionBriefing mode squad : besoin d'un loader per-gamertag (le
	// playerMatchesAdapterFor est bound au main, ne sait pas charger les
	// canonical rows d'un coequipier different). On reutilise le SquadV2Loader.
	briefingLoader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	briefingLoader.SetDefaultGamertag(pdb.Gamertag)
	svc := service.NewTeammatesService(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithSquadLoader(briefingLoader).
		WithMedalDefs(duckdb.NewMedalDefinitionsRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// friendGamertagsResolver construit un resolver lisant app_settings.friend_gamertags
// à chaque appel. Retourne nil si aucun settings store n'est attaché — le
// service tourne alors en mode legacy.
func (r *ServiceRegistry) friendGamertagsResolver() service.FriendGamertagsResolver {
	if r.settingsStore == nil {
		return nil
	}
	return func(ctx context.Context) []string {
		s, err := r.settingsStore.Load()
		if err != nil {
			slog.WarnContext(ctx, "friend_gamertags_load_failed", "err", err)
			return nil
		}
		return s.FriendGamertags
	}
}

// ─── Sprint 54 : Compare + Leaderboard ───────────────────────────────────────

// Compare retourne un CompareService pour le joueur (slug = joueur A).
// Le PlayerStatsProvider (Waypoint) est injecté via DefaultHaloProvider.
// Le contexte est enrichi avec les HaloTokens pour que FetchRemoteStats puisse
// s'authentifier auprès de Waypoint même si la session HTTP ne porte pas de tokens.
func (r *ServiceRegistry) Compare(ctx context.Context, slug string) (port.CompareService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	svc := service.NewCompareService(
		duckdb.NewCompareRepo(pdb),
		haloProvider,
		pdb.XUID,
		pdb.TitleSlug,
	)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// Leaderboard retourne un LeaderboardService pour le joueur.
func (r *ServiceRegistry) Leaderboard(ctx context.Context, slug string) (port.LeaderboardService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewLeaderboardService(duckdb.NewLeaderboardRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ─── Sprint 55 : Synthesis (extrait de Squad) ────────────────────────────────

// SynthesisCtx retourne un SynthesisService + identifiants joueur.
// Sprint 55 D1 : séparé de SquadCtx pour refléter la frontière produit.
func (r *ServiceRegistry) SynthesisCtx(ctx context.Context, slug string) (port.SynthesisService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSynthesisService(duckdb.NewSynthesisRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithPersonalScoreAwardsRepo(duckdb.NewPersonalScoreAwardsRepo(pdb), pdb.XUID).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb))
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}
