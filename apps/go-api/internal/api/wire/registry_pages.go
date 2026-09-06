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
	// WithViewer : l'onglet Médias sert le cœur DU VIEWER (session), pas du joueur
	// dont on consulte la page — les deux diffèrent dès qu'on ouvre le match d'un
	// coéquipier. Résolu ici, au point de composition, parce que platform/duckdb
	// n'a pas (et ne doit pas avoir) accès à la session HTTP.
	svc := service.NewMatchViewService(
		r.newMatchViewRepo(pdb).WithViewer(viewerSlugFor(ctx)), pdb.XUID)
	svc = svc.WithCitationsRepo(duckdb.NewCitationsRepo(pdb)).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithAssetURL(r.assetURLFor(pdb.TitleSlug)).
		WithTitleSlug(pdb.TitleSlug).
		// ModeCategory du header (garde Fiesta du rejeu 2D) : même taxonomie que
		// MediaRepo, cf. haloInfiniteModeTaxonomy (registry_media.go).
		WithModeTaxonomy(haloInfiniteModeTaxonomy()).
		// Flag « Prolongation » : table réglementaire du titre (regulation.toml).
		// Titre sans table → nil → jamais de flag.
		WithRegulation(r.regulationFor(pdb)).
		// Score en MANCHES : même fichier de config, autre table. Titre qui n'en déclare
		// aucune → nil → l'en-tête garde le score de l'API.
		WithRoundsDecide(r.roundsDecideFor(pdb)).
		// Lecture du bloc « Score dans le temps » : même fichier de config, table
		// [score_timeline]. Titre sans table → nil → le client garde la courbe.
		WithScoreTimelineKind(r.scoreTimelineKindFor(pdb)).
		WithMetadataRepo(duckdb.NewMetadataRepo(pdb)).
		// Loader unifié des highlight_events (MV4.A) : sans lui, d.canonicalEvents
		// reste nil et la correction T0 (vrai début de match) est du code mort sur
		// cette page → cadence + 8 rôles bucketés sur l'horloge du film (countdown
		// inclus). Câblé ici comme Timeseries (parité), le pipeline route les events
		// par timeline.CorrectEvents avant les builders narrative.
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb)).
		// Rejeu 2D : MÊME service que l'endpoint /replay (une seule résolution de
		// chemin dans le dépôt). Seule IsAvailable est appelée par la Match View,
		// pour publier `replay_available` sans lire l'artefact.
		WithReplay(r.replayServiceFor(pdb))
	// Timeline objectif v3 + positions joueurs keyframe v3 : les DEUX PROJECTIONS DE
	// L'ARTEFACT DE REJEU servies à la Match View. Câblées SOUS CONDITION : un titre qui
	// ne déclare pas `film.replay_artifact` n'obtient AUCUN des deux loaders, et ses deux
	// endpoints (/objective-events, /positions) rendent alors un 503
	// capability_not_supported — non plus un 200 [] indistinguable d'un match sans données.
	// Le pourquoi et la chaîne complète : registry_pages_film.go.
	svc = r.filmArtifactReposFor(svc, pdb)
	if repo := r.killDistanceRepoFor(pdb); repo != nil {
		svc = svc.WithKillDistanceRepo(repo)
	}
	if loader := r.buildFriendsExtrasResolver(pdb); loader != nil {
		svc = svc.WithFriendsExtras(loader)
	}
	return svc, nil
}

// Replay retourne un ReplayService pour le joueur : sert l'artefact de rejeu 2D
// pré-construit (data/cache/replays/{title}/{matchId}.json) résolu par PathResolver. Le
// titre vient du joueur résolu (pdb.TitleSlug, jamais un slug en dur — multi-titre).
func (r *ServiceRegistry) Replay(ctx context.Context, slug string) (port.ReplayService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return r.replayServiceFor(pdb), nil
}

// replayServiceFor construit le service de rejeu d'un joueur — UN SEUL endroit, partagé par
// l'endpoint /replay et la Match View (qui n'en appelle qu'IsAvailable). Deux constructions
// divergentes, ce serait une Match View qui annonce un rejeu que l'endpoint ne sert pas.
//
// La résolution de carte (fond de carte) lit le registre partagé et les traductions d'assets ;
// elle est passée au service, jamais reconstruite ailleurs.
func (r *ServiceRegistry) replayServiceFor(pdb *duckdb.PlayerDB) port.ReplayService {
	maps := duckdb.NewReplayMapRepo(pdb.SharedReadDB(), pdb.Metadata)
	return service.NewReplayService(pdb.TitleSlug, r.cfg.RepoRoot, maps)
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
		WithWeaponKillsRepo(r.weaponKillsRepoFor(pdb)).
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb)).
		WithHighlightEventsRepo(duckdb.NewHighlightEventsRepo(pdb), pdb.XUID)
	// Axe « Objectifs » par opportunité (profil de participation Session) : gated par
	// la capability match.objective.stats (Infinite ; absente pour Halo 5 → axe
	// retiré). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats) {
		svc = svc.WithObjectiveIndexRepo(duckdb.NewObjectiveStatsRepo(pdb), pdb.XUID)
	}
	// Bloc « usages d'équipement, socles et objectifs » de la session (chantier
	// session-usage S2) : gated par film.usage_summary (Infinite ; absente pour
	// Halo 5 → bloc Available=false avec raison machine). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapFilmUsageSummary) {
		svc = svc.WithSessionUsage(duckdb.NewSessionUsageRepo(pdb), pdb.XUID, r.friendGamertagsResolver())
	}
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
		WithWeaponKillsRepo(r.weaponKillsRepoFor(pdb)).
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
	// KPI objectifs (CTF/Zones/Oddball) : gated par la capability match.objective.stats
	// (Infinite ; absente pour Halo 5 → bloc objective_stats omis). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats) {
		svc = svc.WithObjectiveStatsRepo(duckdb.NewObjectiveStatsRepo(pdb))
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

// weaponKillsRepoFor choisit LE lecteur de l'arme d'un kill pour ce titre.
//
// DEUX IMPLEMENTATIONS, UN SEUL PORT, et le choix est title-agnostic — aucune comparaison
// de slug :
//
//  1. le titre DECLARE la capability DATA-LEVEL `film.kill_source` (capabilities.toml, via
//     la CapabilityMap de son TitleDataAdapter) ET son adapter d'assets sait TRADUIRE une
//     source de degat en cle de registre (`port.KillSourceClassifier`, interface
//     OPTIONNELLE decouverte par assertion) -> le lecteur adosse a la SOURCE DE DEGAT
//     (`match_kill_events_latest.source_tag`), qui voit l'epee, le marteau et le faisceau ;
//  2. sinon -> le lecteur historique sur `weapon_kills`, ou l'arme est NATIVE de l'API du
//     titre (Halo 5 : timeline API, 550 926 lignes — donnee autoritaire, sans rapport avec
//     la correlation defaillante de Halo Infinite).
//
// Un titre qui declare la capability sans fournir de classificateur retombe sur le second :
// degradation gracieuse, jamais de panique (decision A1.7 du plan).
func (r *ServiceRegistry) weaponKillsRepoFor(pdb *duckdb.PlayerDB) port.WeaponKillsRepository {
	if pdb == nil {
		return nil
	}
	if classifier := r.killSourceClassifierFor(pdb); classifier != nil {
		return duckdb.NewKillSourceWeaponKillsRepo(pdb, classifier)
	}
	return duckdb.NewWeaponKillsRepo(pdb)
}

// killSourceClassifierFor rend le traducteur « source de degat -> cle de registre » du
// titre, ou nil s'il n'en a pas.
//
// ATTENTION AU PIEGE GO : le type de retour est l'INTERFACE. Rendre un pointeur concret nil
// produirait une interface NON nil cote appelant (interface non-vide portant un pointeur
// nil), le garde `if classifier != nil` passerait, et le premier appel dereferencerait un
// receveur nil.
func (r *ServiceRegistry) killSourceClassifierFor(pdb *duckdb.PlayerDB) port.KillSourceClassifier {
	if pdb == nil {
		return nil
	}
	return r.killSourceClassifierForSlug(pdb.TitleSlug)
}

// killSourceClassifierForSlug : meme resolution, a partir du seul slug — pour les runners
// d administration, qui travaillent sur un titre sans ouvrir de base de joueur.
func (r *ServiceRegistry) killSourceClassifierForSlug(slug string) port.KillSourceClassifier {
	if r.titleResolver == nil {
		return nil
	}
	data, err := r.titleResolver.Data(slug)
	if err != nil || data == nil || !data.Capabilities().Has(games.CapFilmKillSource) {
		return nil
	}
	classifier, ok := r.assetURLFor(slug).(port.KillSourceClassifier)
	if !ok {
		return nil
	}
	return classifier
}

// killDistanceRepoFor construit le loader « distance par arme, par joueur »
// (POC LOT G.3, plan retours-utilisateur §3bis DEC-8), ou nil si ce titre n'a
// rien à en dire.
//
// MÊME GATE que weaponKillsRepoFor, et c'est un choix délibéré, PAS
// `film.kill_positions` (qui gouverne la CAPTURE des positions, pas la
// lecture — cf. games/adapter.go, doc de CapFilmKillPositions) ni
// `match.events.spatial` (qui gouverne la timeline CANONIQUE cross-titre,
// un pipeline distinct — cf. games/halo_infinite/events.go:
// infiniteEventLimitations). Ce lecteur a besoin de exactement la même chose
// que KillSourceClassRepo : `match_kill_events_latest.source_tag` (via
// `film.kill_source`) ET le classificateur qui le traduit en weapon_key. Une
// table `kill_positions_latest` vide (titre pas encore backfillé) dégrade
// proprement via LoadMatch (zéro ligne, zéro erreur) — inutile de la gater
// une deuxième fois ici.
func (r *ServiceRegistry) killDistanceRepoFor(pdb *duckdb.PlayerDB) port.KillDistanceRepository {
	if pdb == nil || r.titleResolver == nil {
		return nil
	}
	data, err := r.titleResolver.Data(pdb.TitleSlug)
	if err != nil || data == nil || !data.Capabilities().Has(games.CapFilmKillSource) {
		return nil
	}
	classifier, ok := r.assetURLFor(pdb.TitleSlug).(port.KillSourceClassifier)
	if !ok {
		return nil
	}
	return duckdb.NewKillDistanceRepo(pdb, classifier)
}
