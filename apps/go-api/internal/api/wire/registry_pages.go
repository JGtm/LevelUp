// Package api — registry_pages.go : factories services Page (Filters,
// MatchView, Engagement, Sessions, SessionCompare, SessionPage, Stats,
// Timeseries) et factories XxxCtx (Citations, Explorer, Home, MatchHistory,
// Squad, SquadV2, MatchExclusion, Teammates, Compare, Leaderboard, Synthesis)
// + helpers proches (newHomeRepo, playerMatchesAdapterFor,
// friendGamertagsResolver, buildFriendsExtrasResolver). Découpé de
// registry.go (god-file split, refactor 2026-05-27).
package wire

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync/snapshot"
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
	sr := duckdb.SharedReader(snapshot.NewSnapshotPreferredSharedReader(paths, slug, pdb.SharedReadDB()))
	actual, _ := r.snapReaders.LoadOrStore(slug, sr)
	return actual.(duckdb.SharedReader)
}

// MatchView retourne un MatchViewService pour le joueur.
//
// Un match absent du substrat local (jamais synchronisé) renvoie un 404 propre
// — AUCUN fetch live vers l'API du titre depuis cette page (décision user
// 2026-07-19, BACKLOG "Retirer le fallback LIVE du Match view" : latence,
// dépendance token et échec réseau à l'affichage n'étaient pas acceptables).
// Le service ne dépend donc plus d'un DataAdapter.
func (r *ServiceRegistry) MatchView(ctx context.Context, slug string) (port.MatchViewService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	svc := service.NewMatchViewService(
		r.newMatchViewRepo(pdb), pdb.XUID)
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
		return r.newMatchViewRepo(pdb), nil
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
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug).
		WithEngagementCapability(r.engagementCapabilityFor(pdb.TitleSlug)), nil
}

// engagementCapabilityFor resout le statut de la capability fine engagement.score
// du titre (double porte F7). Title-agnostic : lit la CapabilityMap de l'adapter
// data via le resolver, jamais une comparaison de slug. Degradation gracieuse :
// resolver absent / titre inconnu → "" (traite comme supported/validated cote
// service, chemin Infinite legacy).
func (r *ServiceRegistry) engagementCapabilityFor(slug string) games.CapabilityStatus {
	if r.titleResolver == nil {
		return ""
	}
	data, err := r.titleResolver.Data(slug)
	if err != nil || data == nil {
		return ""
	}
	return data.Capabilities()[games.CapEngagement]
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
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb)).
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb)).
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb), pdb.XUID)
	if pdb.Metadata != nil {
		// Placement X/Y dans la colonne Rang : résolveur season_id → seuil CSR (5/10),
		// même source que l'Explorer/match-history. Fallback 5 si absent.
		svc = svc.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata).Get)
	}
	if r.titleHasExpectedStats(pdb.TitleSlug) && pdb.Metadata != nil {
		// Écart cumulé au FDA attendu : assists attendus (modèle personnel player DB
		// → fallback populationnel metadata). Gaté capability → aucun bruit sur H5.
		svc = svc.WithExpectedAssists(r.newMatchViewRepo(pdb), duckdb.NewMetadataRepo(pdb))
	}
	return svc, nil
}

// titleHasExpectedStats indique si le titre déclare CapExpectedStats (écart au FDA
// attendu). Gate la DI des résolveurs d'assists attendus (Timeseries/Sessions) pour
// éviter la résolution DB et le bruit de logs sur les titres sans stats attendues
// (jamais slug== — ratchet no_slug_comparison_test.go).
func (r *ServiceRegistry) titleHasExpectedStats(slug string) bool {
	d := title.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(title.CapExpectedStats)
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
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb)).
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb), pdb.XUID)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	if r.titleHasExpectedStats(pdb.TitleSlug) && pdb.Metadata != nil {
		// Écart au FDA attendu (chart Résumé) : assists attendus (modèle personnel
		// player DB → fallback populationnel metadata). Gaté capability (jamais slug==).
		svc = svc.WithExpectedAssists(r.newMatchViewRepo(pdb), duckdb.NewMetadataRepo(pdb))
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

// MedalsCtx retourne un MedalsService + identifiants joueur. Le resolver de catégorie
// du titre courant est sélectionné dans le service via le registre (baseline par
// défaut, enrichissement Halo Infinite si enregistré au boot) — jamais de gating slug.
func (r *ServiceRegistry) MedalsCtx(ctx context.Context, slug string) (port.MedalsService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewMedalsService(duckdb.NewMedalsRepo(pdb))
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
