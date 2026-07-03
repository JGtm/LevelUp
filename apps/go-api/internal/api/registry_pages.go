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
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/platform/halo"
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

// matchViewSharedReader retourne le SharedReader des lectures shared de MatchView :
// un reader snapshot-préféré (lecture découplée du B-swap) avec fallback live, SINGLETON
// par titre (cache de queriers :memory: partagé entre requêtes). Dégrade sur
// pdb.SharedReadDB() si cfg indisponible (tests). SCOPED a MatchView : le snapshot
// reconstruit l'integralite des relations shared MATCH-IMMUTABLES que MatchView lit
// (fidelite testee) ; les relations shared non-match-immutables (world leaderboard) lues
// par d'autres repos restent sur le live (pas wrappees).
func (r *ServiceRegistry) matchViewSharedReader(pdb *duckdb.PlayerDB) duckdb.SharedReader {
	if pdb == nil {
		return nil
	}
	if r.cfg == nil {
		return pdb.SharedReadDB()
	}
	slug := pdb.TitleSlug
	if v, ok := r.snapReaders.Load(slug); ok {
		return v.(duckdb.SharedReader)
	}
	paths := title.NewPathResolver(r.cfg.RepoRoot)
	sr := duckdb.SharedReader(sync_pkg.NewSnapshotPreferredSharedReader(paths, slug, pdb.SharedReadDB()))
	actual, _ := r.snapReaders.LoadOrStore(slug, sr)
	return actual.(duckdb.SharedReader)
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
	svc := service.NewMatchViewService(
		duckdb.NewMatchViewRepo(pdb, pdb.XUID).WithSharedReader(r.matchViewSharedReader(pdb)), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		// Voie canonique (repo-first / adapter-fallback) : le viewer gamertag est
		// requis par LoadMatchDetail des titres GAMERTAG-keyés (Halo 5, Player.Xuid
		// null). Posé sur le service → injecté dans le ctx avant l'appel adapter.
		// No-op pour un titre xuid-keyé (HINF n'emprunte pas la voie canonique).
		svc = svc.WithDataAdapter(a).WithViewerGamertag(pdb.Gamertag)
	}
	svc = svc.WithCitationsRepo(duckdb.NewCitationsRepo(pdb)).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithAssetURL(r.assetURLFor(pdb.TitleSlug)).
		WithTitleSlug(pdb.TitleSlug).
		WithMetadataRepo(duckdb.NewMetadataRepo(pdb)).
		// Loader unifié des highlight_events (MV4.A) : sans lui, d.canonicalEvents
		// reste nil et la correction T0 (vrai début de match) est du code mort sur
		// cette page → cadence + 8 rôles bucketés sur l'horloge du film (countdown
		// inclus). Câblé ici comme Timeseries (parité), le pipeline route les events
		// par timeline.CorrectEvents avant les builders narrative.
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb))
	if loader := r.buildFriendsExtrasResolver(pdb); loader != nil {
		svc = svc.WithFriendsExtras(loader)
	}
	return svc, nil
}

// MatchEvents retourne un MatchEventsService pour le joueur : timeline canonique
// d'events (kill-feed / timeline) servie par l'adapter du titre, avec résolution
// des gamertags via le chokepoint canonique (v_gamertag_lookup).
//
// Multi-titre : l'adapter vient du builder enregistré pour le titre du joueur
// (dataAdapterForPDB). Titre sans builder → adapter nil → le service renvoie
// ErrCapabilityNotSupported (handler 503). Le resolver gamertag est branché sur
// le SharedReader du joueur (no-op pour un titre déjà gamertag-keyé comme Halo 5).
func (r *ServiceRegistry) MatchEvents(ctx context.Context, slug string) (port.MatchEventsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	adapter := r.dataAdapterForPDB(pdb)
	resolver := duckdb.NewGamertagRepo(pdb.SharedReadDB())
	return service.NewMatchEventsService(adapter, resolver), nil
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
		return duckdb.NewMatchViewRepo(pdb, pdb.XUID).WithSharedReader(r.matchViewSharedReader(pdb)), nil
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

// SessionPage retourne un SessionPageService pour le joueur.
func (r *ServiceRegistry) SessionPage(ctx context.Context, slug string) (port.SessionPageService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewSessionPageService(duckdb.NewStatsRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	if pdb.Metadata != nil {
		// Placement X/Y dans la colonne Rang : résolveur season_id → seuil CSR (5/10),
		// même source que l'Explorer/match-history. Fallback 5 si absent.
		svc = svc.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata).Get)
	}
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

// CommendationTotalsCtx retourne un CommendationTotalsService + identifiants joueur.
// L'adapter du titre est type-asserté à la surface LoadCommendationTotals : seuls les
// titres l'implémentant (Halo 5) renvoient des totaux natifs ; les autres → loader nil
// → réponse vide (dégradation gracieuse, jamais de gating par slug).
func (r *ServiceRegistry) CommendationTotalsCtx(ctx context.Context, slug string) (port.CommendationTotalsService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	var loader service.CommendationTotalsLoader
	if a := r.dataAdapterForPDB(pdb); a != nil {
		if l, ok := a.(service.CommendationTotalsLoader); ok {
			loader = l
		}
	}
	return service.NewCommendationTotalsService(loader), pdb.XUID, pdb.Gamertag, nil
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
	// Encart "Profil joueur cible" :
	//   - LocalIdentity : identité depuis la DB locale de la cible (si suivie)
	//   - LiveIdentity  : career rank/emblem/backdrop live pour un xuid arbitraire
	//   - RemoteStats   : service record (stats + temps de jeu + médailles), caché
	//   - MedalDefs     : labels/descriptions médailles (top médailles)
	//   - CSR           : classements CSR saison courante (live, tout xuid)
	homeRepo := r.newHomeRepo(pdb)
	csrSeasonID := ""
	if r.cfg != nil {
		csrSeasonID = r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil)
	}
	var seasons []service.SeasonCatalogEntry
	if r.seasonsCatalog != nil {
		seasons = r.seasonsCatalog.Load(ctx, pdb.TitleSlug)
	}
	// Catalogue des rangs de carrière (même source que HomeService) pour
	// localiser RankTitle/NextRankTitle du DTO d'identité cible. nil-safe.
	var ranks *mappings.RankCatalog
	if sem := r.semanticFor(pdb.TitleSlug); sem != nil {
		ranks = sem.Ranks()
	}
	// CSR live (Explorer) : providers spécifiques Infinite → n'injecter que si le
	// titre expose match.skill.snapshot, sinon nil (le service dégrade : encart CSR
	// vide, pas de fuite de playlists/CSR Infinite sous un autre titre — F2).
	var csrProvider service.ExplorerTargetCSRProvider
	var seasonCSRProvider service.ExplorerSeasonCSRProvider
	if r.titleSupportsLiveCSR(pdb) {
		csrProvider = r.newExplorerCSRProvider()
		seasonCSRProvider = r.newExplorerSeasonCSRProvider()
	}
	svc = svc.WithTargetProfileProviders(service.ExplorerTargetProfileDeps{
		LocalIdentity:   r.newExplorerLocalIdentityResolver(pdb.TitleSlug),
		LiveIdentity:    r.newCareerLiveService(pdb, homeRepo),
		RemoteStats:     r.remoteStats,
		MedalDefs:       duckdb.NewMedalDefinitionsRepo(pdb),
		CSR:             csrProvider,
		CurrentSeasonID: csrSeasonID,
		Seasons:         seasons,
		SeasonSR:        r.remoteStats, // *CachedStatsProvider implémente port.SeasonStatsProvider
		SeasonCSR:       seasonCSRProvider,
		Ranks:           ranks,
		RecentMatches:   wrapRecentMatchesAuthRetry(r.recentMatches),
		LocalBannerPool: r.newExplorerLocalBannerPool(pdb.TitleSlug),
		TitleSlug:       pdb.TitleSlug,
	})
	// Fallback live gamertag→xuid (joueur jamais croisé) — nil-safe (no-op en démo).
	svc = svc.WithLiveGamertagResolver(r.liveGamertagResolver)
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// titleSupportsLiveCSR indique si le titre du joueur expose la capability
// match.skill.snapshot (Infinite = degraded → oui ; H5 = not_exposed → non).
// Les providers CSR Explorer/Compare sont spécifiques au live Infinite
// (rankedplaylists.Active() + endpoints CSR HINF) : ne les injecter que pour un
// titre déclarant cette capability évite de servir des playlists/CSR Infinite
// sous un autre titre (F2). Gate par CAPABILITY, jamais par slug (ratchet ADR 0025).
func (r *ServiceRegistry) titleSupportsLiveCSR(pdb *duckdb.PlayerDB) bool {
	a := r.dataAdapterForPDB(pdb)
	return a != nil && a.Capabilities().Has(games.CapMatchSkillSnapshot)
}

// newExplorerCSRProvider construit le provider CSR de l'encart cible : instancie
// un client Halo depuis les tokens du contexte (Spartan + clearance) et appelle
// l'endpoint skill CSR (service token, fonctionne pour tout xuid), puis mappe
// vers le type domain. nil tokens → nil (pas d'erreur).
func (r *ServiceRegistry) newExplorerCSRProvider() service.ExplorerTargetCSRProvider {
	return service.CSRProviderFunc(func(ctx context.Context, xuid, seasonID string) ([]domain.CareerPlaylistCSR, error) {
		// Filet auth (defense-in-depth) : un 401/403 sur GetPlayerCSRs (token owner
		// révoqué en cours de requête) → re-mint + retry unique. Péremption normale déjà
		// couverte par le cache token expiry-aware en amont.
		return halo.RetryWithFreshTokens(ctx, sync_pkg.IsAuthError, func(c context.Context) ([]domain.CareerPlaylistCSR, error) {
			tokens := ctxkeys.HaloTokens(c)
			if tokens == nil || tokens.SpartanToken == "" {
				return nil, nil
			}
			client := sync_pkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 10)
			// 1. Playlists ranked ENGAGÉES de la saison (endpoint player-level).
			raw, err := client.GetPlayerCSRs(c, xuid, seasonID)
			if err != nil {
				return nil, err
			}
			// 2. Compléter avec les playlists ranked ACTIVES manquantes (endpoint
			//    par-playlist) — parité avec la page Carrière. Même mécanisme que
			//    sync.augmentWithActiveRankedCSRs.
			seen := make(map[string]struct{}, len(raw))
			for i := range raw {
				seen[strings.ToLower(strings.TrimSpace(raw[i].PlaylistID))] = struct{}{}
			}
			for _, pl := range rankedplaylists.Active() {
				if _, ok := seen[strings.ToLower(pl.AssetID)]; ok {
					continue
				}
				res, perr := client.GetPlaylistCsr(c, pl.AssetID, xuid, seasonID)
				if perr != nil {
					slog.WarnContext(c, "explorer_target_csr_augment_failed", "playlist", pl.AssetID, "err", perr)
					continue
				}
				if res == nil {
					continue
				}
				res.PlaylistName = pl.NameFR
				res.Queue = pl.Queue
				res.Input = pl.Input
				raw = append(raw, *res)
			}
			return mapSyncCSRsToDomain(raw), nil
		})
	})
}

// newExplorerSeasonCSRProvider construit le provider de PIC CSR PAR SAISON PASSÉE
// (badge au-dessus des barres "matchs par saison").
//
// IMPORTANT : le endpoint player-level /hi/players/.../csrs?Season= renvoie 404
// (vérifié empiriquement, y compris pour la saison courante). Le CSR par saison
// — y compris PASSÉE — n'est servi de façon fiable que par GetPlaylistCsr
// (/hi/playlist/{id}/csrs?players=...&season=, HTTP 200 + SeasonMax = pic de la
// saison). On interroge donc chaque playlist ranked active et on retient le plus
// haut tier. (nil, nil) sans tokens / si aucune donnée CSR pour la saison.
func (r *ServiceRegistry) newExplorerSeasonCSRProvider() service.ExplorerSeasonCSRProvider {
	return service.SeasonCSRPeakFunc(func(ctx context.Context, xuid, csrSeasonID string, engagedPlaylistIDs []string) (*service.SeasonCSRPeak, error) {
		tokens := ctxkeys.HaloTokens(ctx)
		if tokens == nil || tokens.SpartanToken == "" || len(engagedPlaylistIDs) == 0 {
			return nil, nil
		}
		// Optim : ne requêter que les playlists ranked actives RÉELLEMENT engagées
		// par le joueur (intersection avec Subqueries.PlaylistAssetIds). Un joueur
		// social → engaged ∩ ranked = ∅ → 0 appel CSR.
		engaged := make(map[string]struct{}, len(engagedPlaylistIDs))
		for _, id := range engagedPlaylistIDs {
			engaged[strings.ToLower(strings.TrimSpace(id))] = struct{}{}
		}
		client := sync_pkg.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, 10)

		var best *sync_pkg.CSRRankSnapshot
		for _, pl := range rankedplaylists.Active() {
			if _, ok := engaged[strings.ToLower(strings.TrimSpace(pl.AssetID))]; !ok {
				continue // playlist ranked jamais jouée par ce joueur → skip
			}
			res, err := client.GetPlaylistCsr(ctx, pl.AssetID, xuid, csrSeasonID)
			if err != nil || res == nil {
				continue // playlist absente cette saison-là / erreur → on ignore
			}
			s := res.Season // SeasonMax = pic de rang de la saison demandée
			if strings.TrimSpace(s.Tier) == "" {
				continue
			}
			if best == nil || s.Value > best.Value {
				snap := s
				best = &snap
			}
		}
		if best == nil {
			return nil, nil
		}
		peak := &service.SeasonCSRPeak{Tier: best.Tier, SubTier: best.SubTier}
		if url := csrBadgeURL(halo_games.NewAssetURLAdapter(), best.Tier, best.SubTier); url != "" {
			peak.BadgeURL = &url
		}
		return peak, nil
	})
}

// wrapRecentMatchesAuthRetry décore un RecentMatchesProvider avec le filet auth
// (defense-in-depth) : un 401/403 du token owner en cours de requête → re-mint + retry
// unique. nil → nil (pas de provider). Le re-mint cible le xuid OWNER (ctx), pas la cible.
func wrapRecentMatchesAuthRetry(inner port.RecentMatchesProvider) port.RecentMatchesProvider {
	if inner == nil {
		return nil
	}
	return authRetryRecentMatches{inner: inner}
}

type authRetryRecentMatches struct{ inner port.RecentMatchesProvider }

func (a authRetryRecentMatches) FetchRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error) {
	return halo.RetryWithFreshTokens(ctx, sync_pkg.IsAuthError, func(c context.Context) ([]domain.ExplorerTargetRecentMatch, error) {
		return a.inner.FetchRecentMatches(c, xuid, limit)
	})
}

// mapSyncCSRsToDomain projette les CSR du client sync vers le type domain
// (CareerPlaylistCSR), avec résolution du badge image via l'AssetURLAdapter.
func mapSyncCSRsToDomain(in []sync_pkg.PlayerPlaylistCSR) []domain.CareerPlaylistCSR {
	adapter := halo_games.NewAssetURLAdapter()
	out := make([]domain.CareerPlaylistCSR, 0, len(in))
	for i := range in {
		c := &in[i]
		out = append(out, domain.CareerPlaylistCSR{
			PlaylistID:   c.PlaylistID,
			PlaylistName: c.PlaylistName,
			Queue:        c.Queue,
			Input:        c.Input,
			Current:      mapSyncCSRSnapshot(adapter, c.Current),
			Season:       mapSyncCSRSnapshot(adapter, c.Season),
			AllTime:      mapSyncCSRSnapshot(adapter, c.AllTime),
		})
	}
	return out
}

func mapSyncCSRSnapshot(adapter *halo_games.AssetURLAdapter, s sync_pkg.CSRRankSnapshot) domain.CareerCSRRank {
	out := domain.CareerCSRRank{
		Value:                       s.Value,
		Tier:                        s.Tier,
		SubTier:                     s.SubTier,
		MeasurementMatchesRemaining: s.MeasurementMatchesRemaining,
	}
	if url := csrBadgeURL(adapter, s.Tier, s.SubTier); url != "" {
		out.BadgeImageURL = &url
	}
	return out
}

// csrBadgeURL résout l'URL du badge CSR (/static/ranks/...) via l'AssetURLAdapter
// halo_infinite. "" si le tier est vide (non classé) ou hors plage.
func csrBadgeURL(adapter *halo_games.AssetURLAdapter, tier string, subTier int) string {
	if strings.TrimSpace(tier) == "" {
		return ""
	}
	if strings.EqualFold(tier, "Onyx") {
		return adapter.CSRRankImageURLOnyx()
	}
	if subTier < 1 || subTier > 6 {
		return ""
	}
	return adapter.CSRRankImageURL(tier, subTier)
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

// newExplorerLocalBannerPool construit le résolveur PARESSEUX du pool de
// bannières locales (Phase 3.6) : la liste dédupliquée et à ordre stable des
// banner_image_url des joueurs suivis (db_profiles). Appelé uniquement quand une
// cible non-locale n'a ni bannière ni backdrop → on lui attribue une nameplate
// de repli déterministe par xuid. resolveByGT est pool-cached ;
// LoadSpartanIdentity lit la player DB (même chemin que l'identité locale).
func (r *ServiceRegistry) newExplorerLocalBannerPool(titleSlug string) func(ctx context.Context) []string {
	return func(ctx context.Context) []string {
		if r.cfg == nil || r.resolveByGT == nil {
			return nil
		}
		players, err := r.cfg.LoadPlayers(titleSlug)
		if err != nil {
			slog.WarnContext(ctx, "explorer_banner_pool_load_players_failed", "err", err)
			return nil
		}
		seen := make(map[string]struct{}, len(players))
		out := make([]string, 0, len(players))
		for _, p := range players {
			if p.Gamertag == "" {
				continue
			}
			tpdb, gerr := r.resolveByGT(ctx, titleSlug, p.Gamertag)
			if gerr != nil || tpdb == nil {
				continue
			}
			row, lerr := r.newHomeRepo(tpdb).LoadSpartanIdentity(ctx)
			if lerr != nil || row == nil || row.BannerImageURL == nil || *row.BannerImageURL == "" {
				continue
			}
			if _, ok := seen[*row.BannerImageURL]; ok {
				continue
			}
			seen[*row.BannerImageURL] = struct{}{}
			out = append(out, *row.BannerImageURL)
		}
		slog.DebugContext(ctx, "explorer_banner_pool_built", "title_slug", titleSlug, "size", len(out))
		return out
	}
}

// skillBadgeResolverFor construit le résolveur d'URL de badge CSR title-aware
// pour un slug de titre donné. C'est le pont entre le package analysis (pur,
// title-agnostic) et la résolution title-aware de duckdb.TitleSkillBadgeURL
// (csr_designations pour Halo 5, sinon static HINF). subTier : 0 pour Onyx, 1..6 sinon.
func skillBadgeResolverFor(slug string) func(tierEN string, subTier int) string {
	return func(tierEN string, subTier int) string {
		return duckdb.TitleSkillBadgeURL(slug, tierEN, subTier)
	}
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
		WithSquadSessionTeammates(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo)).
		WithSkillBadgeResolver(skillBadgeResolverFor(pdb.TitleSlug)).
		WithDemoMode(r.cfg.DemoMode)
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
			// newHomeRepo n'a pas de ctx en paramètre ; le titre vient du pdb.
			csrSeasonID = r.cfg.CSRSeasonIDForTitle(context.Background(), pdb.TitleSlug, nil)
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
	// r.remoteStats (CachedStatsProvider) plutôt que haloProvider direct : Compare
	// bénéficie du même cache TTL 5 min + singleflight que l'Explorer (dédup +
	// latence réduite sur les cibles consultées en parallèle).
	svc := service.NewCompareService(
		duckdb.NewCompareRepo(pdb),
		r.remoteStats,
		pdb.XUID,
		pdb.TitleSlug,
	).WithLiveIdentity(r.newCareerLiveService(pdb, r.newHomeRepo(pdb))).
		WithLiveGamertagResolver(r.liveGamertagResolver) // nil-safe (no-op en démo)
	csrSeasonID := ""
	if r.cfg != nil {
		csrSeasonID = r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil)
	}
	// CSR live (Compare) : provider spécifique Infinite → gate capability (F2).
	if r.titleSupportsLiveCSR(pdb) {
		svc = svc.WithCSR(r.newExplorerCSRProvider(), csrSeasonID)
	}
	// Catalogue de rangs carrière (même source que le profil de combat) pour
	// afficher le rang en titre ("Général Platine VI") plutôt qu'en numéro.
	if sem := r.semanticFor(pdb.TitleSlug); sem != nil {
		svc = svc.WithRanks(sem.Ranks())
	}
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
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb)).
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb))
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}
