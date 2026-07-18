// Package scheduler — world_leaderboard_cron.go : cron basse fréquence qui
// capture le classement CSR mondial (scrape Halo Waypoint) et le persiste en
// append-only dans shared.world_csr_leaderboard_snapshots.
//
// Pourquoi : Halo Waypoint n'expose aucune API de classement mondial (HaloDotAPI
// mort). La seule source est la page web publique, qu'il faut re-photographier
// régulièrement car le classement évolue. Le CLI cmd/snapshot-world-leaderboard
// fait ce travail mais EXIGE d'arrêter le serveur (ouverture RW directe de la
// shared DB). Ce cron fait la même chose serveur allumé, via le SharedDBProvider
// (AcquireWriter), donc zéro intervention manuelle.
//
// Mode autonome (ADR : direction sync convergent autonome) : la saison active est
// DÉCOUVERTE sur la page (seasons[0]), jamais codée en dur — quand Halo change de
// saison, le cron suit tout seul. Seule la saison active est rafraîchie : les
// saisons passées sont figées (un snapshot one-shot via le CLI suffit).
//
// Architecture clé — fenêtre RW minimale :
//   - Le scraping (réseau, potentiellement plusieurs minutes : pagination + délais
//     polis × N playlists) se fait HORS lease writer.
//   - Le writer n'est acquis que pour les INSERT (quelques dizaines de ms), pour
//     ne pas tenir la shared DB en mode RW pendant le scrape (sinon les handlers
//     HTTP lecteurs seraient bloqués / 503 tout du long).
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// LeaderboardScraperPort abstrait halo.LeaderboardScraper pour le mocking en test.
type LeaderboardScraperPort interface {
	// FetchActiveSeason découvre la saison CSR active (seasons[0] de la page).
	FetchActiveSeason(ctx context.Context, refPlaylistID string) (string, error)
	// FetchCSRLeaderboard scrape le classement d'une playlist pour une saison.
	FetchCSRLeaderboard(ctx context.Context, seasonID, playlistID string, limit int) ([]domain.LeaderboardEntry, error)
	// FetchActivePlaylists découvre les playlists classées ACTIVES du menu déroulant
	// de la page (source directe autoritative — le manifest de build a un PlaylistLinks
	// vide). refPlaylistID = graine pour charger la page.
	FetchActivePlaylists(ctx context.Context, refPlaylistID string) ([]domain.WorldPlaylistRef, error)
	// FetchSeasons découvre la liste des saisons CSR (nom d'Operation EN + FR) du menu
	// déroulant de la page — source autoritative pour season_catalog (C2).
	FetchSeasons(ctx context.Context, refPlaylistID string) ([]domain.WorldSeasonRef, error)
	// SetSeedSeason injecte la graine de découverte (F11/LB3) : le cron y pose la
	// dernière saison persistée pour ne pas dépendre d'une graine constante figée.
	SetSeedSeason(seasonID string)
}

// WorldStatsEnricherPort agrège les stats des joueurs d'une saison (Phase C).
// Satisfait par *service.WorldStatsEnricher. Optionnel : si nil, le cron se limite
// au scrape CSR (comportement historique). Best-effort par joueur.
type WorldStatsEnricherPort interface {
	EnrichSeason(ctx context.Context, season string, players []domain.WorldPlayerRef) ([]domain.WorldPlayerSeasonStats, []error)
}

const (
	// DefaultWorldLeaderboardInterval : 1×/jour suffit — le classement mondial
	// bouge lentement et le scraping doit rester poli (page non documentée).
	DefaultWorldLeaderboardInterval = 24 * time.Hour
	// defaultWorldLeaderboardFreshness : si un snapshot de la saison active date
	// de moins que ça, on saute le cycle. Garde-fou anti-spam au boot / hot-reload
	// Air (sinon chaque redémarrage re-scraperait Halo Waypoint). < interval pour
	// tolérer une légère dérive du ticker.
	defaultWorldLeaderboardFreshness = 20 * time.Hour
	// defaultWorldLeaderboardLimit : nb max d'entrées scrapées par playlist.
	defaultWorldLeaderboardLimit = 200
	// defaultWorldLeaderboardMinEntries : plancher de cohérence. En dessous, on
	// considère que le scrape de la playlist est tronqué (page à moitié cassée,
	// markup changé, coupure réseau en pleine pagination) et on l'IGNORE plutôt
	// que de persister un snapshot partiel — les données précédentes (append-only)
	// restent affichées. Les playlists classées actives scrapées (Arène, Assassin,
	// Duo, Legacy) comptent des milliers de joueurs classés : un top-200 qui
	// renvoie < 25 est quasi-certainement un glitch, pas un classement légitime.
	defaultWorldLeaderboardMinEntries = 25
)

// WorldLeaderboardCron capture périodiquement le classement CSR mondial de la
// saison active pour toutes les playlists classées et le persiste en append-only.
type WorldLeaderboardCron struct {
	provider   sharedprovider.Provider
	scraper    LeaderboardScraperPort
	enricher   WorldStatsEnricherPort // optionnel (Phase C) ; nil → pas d'enrichissement
	playlists  func() []string        // asset IDs des playlists classées à scraper
	registry   *titlePkg.Registry     // titres actifs à itérer (PMT-7 write-path, capability-gated)
	interval   time.Duration
	freshness  time.Duration
	limit      int
	minEntries int // plancher de cohérence par playlist (sous ce seuil → skip)
}

// WithStatsEnricher branche l'enrichissement Phase C (agrégateur multi-tokens).
// Retourne le cron pour chaînage. nil-safe : un enricher nil laisse le cron en
// mode scrape-only. Le wiring (cmd/server) le fournit une fois le pool de tokens
// + le résolveur PeopleHub construits.
func (c *WorldLeaderboardCron) WithStatsEnricher(e WorldStatsEnricherPort) *WorldLeaderboardCron {
	if c != nil {
		c.enricher = e
	}
	return c
}

// NewWorldLeaderboardCron construit le cron. provider/scraper requis (nil → noop).
// Si interval <= 0, DefaultWorldLeaderboardInterval est utilisé. La liste des
// playlists est dérivée de rankedplaylists.Active() (playlists classées actives).
func NewWorldLeaderboardCron(
	provider sharedprovider.Provider,
	scraper LeaderboardScraperPort,
	interval time.Duration,
) *WorldLeaderboardCron {
	if interval <= 0 {
		interval = DefaultWorldLeaderboardInterval
	}
	return &WorldLeaderboardCron{
		provider:   provider,
		scraper:    scraper,
		playlists:  activeRankedPlaylistIDs,
		registry:   titlePkg.DefaultRegistry(),
		interval:   interval,
		freshness:  defaultWorldLeaderboardFreshness,
		limit:      defaultWorldLeaderboardLimit,
		minEntries: defaultWorldLeaderboardMinEntries,
	}
}

// activeRankedPlaylistIDs retourne les asset IDs des playlists classées actives.
func activeRankedPlaylistIDs() []string {
	pls := rankedplaylists.Active()
	out := make([]string, 0, len(pls))
	for _, pl := range pls {
		out = append(out, pl.AssetID)
	}
	return out
}

// Run lance le cron : un premier tick immédiat (peuple la prod au boot sans manip
// manuelle), puis toutes les `interval`. Bloque jusqu'à ctx.Done().
func (c *WorldLeaderboardCron) Run(ctx context.Context) {
	if c == nil || c.provider == nil || c.scraper == nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: noop (provider/scraper nil)",
			"module", logging.ModuleLeaderboard)
		return
	}
	slog.InfoContext(ctx, "world_leaderboard_cron: started",
		"module", logging.ModuleLeaderboard, "interval", c.interval)

	c.RunOnce(ctx)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "world_leaderboard_cron: stopped (ctx done)",
				"module", logging.ModuleLeaderboard)
			return
		case <-ticker.C:
			c.RunOnce(ctx)
		}
	}
}

// RunOnce exécute un cycle complet. Exporté pour un éventuel endpoint admin
// (force-refresh) et les tests. Best-effort : une playlist en échec n'interrompt
// pas le cycle. Le writer n'est acquis qu'après le scraping (fenêtre RW minimale).
//
// Title-aware (PMT-7) : itère sur les titres ACTIFS du registre et ne produit un
// snapshot QUE pour ceux qui DÉCLARENT CapWorldLeaderboard. Un titre sans la cap
// (ex. Halo 5 : pas de classement mondial scrapable) est SKIPPÉ proprement (slog
// debug, jamais d'erreur). La source de données (provider/scraper/playlists)
// reste Infinite-spécifique par construction : la brique itère mais n'invente
// aucune source pour un titre qui n'a pas la capability.
func (c *WorldLeaderboardCron) RunOnce(ctx context.Context) {
	if c == nil || c.provider == nil || c.scraper == nil {
		return
	}
	// Statut des crons (A6/DC-5, décision D1) : le cycle rapporte l'erreur AGRÉGÉE
	// réelle par titre — un cycle partiellement échoué = échec avec cause. Les
	// dégradations nominales (saison indécouvrable, snapshot frais, playlist tronquée)
	// restent des skips best-effort internes (nil), pas des échecs de cron.
	start := time.Now()
	var errs []error
	reg := c.registry
	if reg == nil {
		reg = titlePkg.DefaultRegistry()
	}
	for _, desc := range reg.Active() {
		if desc == nil {
			continue
		}
		if !desc.HasCapability(titlePkg.CapWorldLeaderboard) {
			// Dégradation gracieuse : titre actif mais sans classement mondial
			// (aucune source scrapable). On le saute, ce n'est PAS une erreur.
			slog.DebugContext(ctx, "world_leaderboard_cron: titre sans world.leaderboard — skip",
				"module", logging.ModuleLeaderboard, "titleSlug", desc.Slug)
			continue
		}
		if err := c.runOnceForTitle(ctx, desc.Slug); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", desc.Slug, err))
		}
	}
	observability.ReportCronRun("world_leaderboard", start, errors.Join(errs...), time.Since(start).Milliseconds())
}

// runOnceForTitle exécute le cycle scrape+persist+enrich pour UN titre déclarant
// CapWorldLeaderboard. Retourne l'erreur RÉELLE du cycle (décision D1) : nil pour
// les skips nominaux (playlists absentes, saison indécouvrable, snapshot frais,
// scrape vide — dégradations attendues/auto-résolutives déjà loguées WARN/INFO), la
// cause pour un échec DUR (persistance du snapshot). L'enrichissement Phase C reste
// best-effort interne (n'échoue pas le cycle : le classement est déjà persisté). Une
// erreur ici n'interrompt pas l'itération sur les autres titres (agrégée par RunOnce).
func (c *WorldLeaderboardCron) runOnceForTitle(ctx context.Context, titleSlug string) error {
	start := time.Now()

	static := c.playlists()
	if len(static) == 0 {
		slog.WarnContext(ctx, "world_leaderboard_cron: aucune playlist classée active — cycle ignoré",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug)
		return nil
	}

	// Graine de découverte = dernière saison persistée (F11/LB3), avant toute requête
	// catalogue (FetchActivePlaylists/FetchActiveSeason passent par la page-graine).
	c.applyDiscoverySeed(ctx, titleSlug)
	// Découvrir les playlists classées ACTIVES sur la page Waypoint (7 réelles) au lieu
	// de se limiter à la liste statique (historiquement 4) — sinon seules les playlists
	// scrapées ont des snapshots, donc la page classement n'en affiche qu'une poignée.
	playlists := c.discoverActivePlaylists(ctx, static)

	// 1. Découvrir la saison active (autonome) en essayant plusieurs playlists de
	//    référence. Le scraper construit l'URL de découverte avec une saison-graine
	//    FIXE (seedSeasonID) : une playlist qui n'était pas classée dans cette saison
	//    renvoie un 404 NOMINAL. L'ancien code n'essayait qu'une seule playlist et
	//    abandonnait tout le cycle sur ce 404 en loguant une ERROR quotidienne (bruit
	//    prod B3.1). On tente maintenant les candidates successives (cf.
	//    discoverActiveSeason).
	season, ok := c.discoverActiveSeason(ctx, titleSlug, static, playlists)
	if !ok {
		return nil // dégradation (WARN + compteur expvar) déjà loguée dans discoverActiveSeason
	}

	// 2. Garde-fou fraîcheur (lecture RO, sans lease writer).
	if c.snapshotIsFresh(ctx, season) {
		slog.InfoContext(ctx, "world_leaderboard_cron: snapshot récent présent — cycle ignoré",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "season", season, "freshness", c.freshness)
		return nil
	}

	// 3. Scraper toutes les playlists HORS lease writer (réseau lent).
	entries, scraped := c.scrapeAll(ctx, season, playlists)
	if len(entries) == 0 {
		slog.WarnContext(ctx, "world_leaderboard_cron: aucune entrée scrapée — rien à persister",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "season", season)
		return nil
	}

	// Découvrir les saisons (nom d'Operation + FR) HORS lease writer — best-effort,
	// persistées dans la même fenêtre writer que le snapshot CSR (C2).
	seasons := c.discoverSeasons(ctx, playlists[0])

	// 4. Acquérir le writer brièvement et insérer (fenêtre RW minimale).
	inserted, err := c.persist(ctx, titleSlug, entries, seasons)
	if err != nil {
		slog.ErrorContext(ctx, "world_leaderboard_cron: persistance échouée",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "season", season, "err", err)
		return fmt.Errorf("persist snapshot: %w", err)
	}

	slog.InfoContext(ctx, "world_leaderboard_cron: cycle terminé",
		"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "season", season,
		"playlists", len(playlists), "scraped", scraped, "inserted", inserted,
		"duration", time.Since(start))

	// 5. Enrichissement Phase C (optionnel) : agréger les stats des joueurs de la
	// saison active et les persister. Best-effort, après le snapshot CSR — un échec
	// ici ne compromet pas le classement déjà persisté.
	c.enrich(ctx, season)
	return nil
}

// applyDiscoverySeed remplace la graine de découverte du scraper par la DERNIÈRE
// saison réellement persistée dans les snapshots (F11/LB3) — garantie de rendre la
// page Waypoint. Repli silencieux sur la graine constante par défaut du scraper si
// aucun snapshot (DB vide au premier boot) ou lecture impossible : la graine figée
// n'est plus l'unique point de découverte, donc elle ne peut plus geler le classement.
func (c *WorldLeaderboardCron) applyDiscoverySeed(ctx context.Context, titleSlug string) {
	db, release, err := c.provider.Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: lecture graine saison impossible — graine par défaut conservée",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "err", err)
		return
	}
	defer release()
	season, ok, err := duckdb.WorldCSRLatestSeason(ctx, db)
	if err != nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: requête graine saison échouée — graine par défaut conservée",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "err", err)
		return
	}
	if !ok {
		return // DB vide (premier boot) → le scraper garde sa graine constante (fallback).
	}
	c.scraper.SetSeedSeason(season)
	slog.DebugContext(ctx, "world_leaderboard_cron: graine de découverte = dernière saison persistée",
		"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "season", season)
}

// discoverActivePlaylists découvre les playlists classées ACTIVES exposées par le menu
// déroulant de la page Waypoint (via une playlist de référence statique comme graine).
// Fallback sur la liste statique si la découverte échoue OU revient vide : on ne scrape
// jamais zéro playlist à cause d'un hoquet de page (résilience). C'est ce qui remplace la
// limite historique aux ~4 playlists en dur par les playlists réellement actives (7+).
func (c *WorldLeaderboardCron) discoverActivePlaylists(ctx context.Context, static []string) []string {
	refs, err := c.scraper.FetchActivePlaylists(ctx, static[0])
	if err != nil || len(refs) == 0 {
		slog.WarnContext(ctx, "world_leaderboard_cron: découverte playlists actives échouée — fallback statique",
			"module", logging.ModuleLeaderboard, "static", len(static), "err", err)
		return static
	}
	ids := make([]string, 0, len(refs))
	for _, r := range refs {
		if id := r.AssetID; id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return static
	}
	slog.InfoContext(ctx, "world_leaderboard_cron: playlists actives découvertes (Waypoint)",
		"module", logging.ModuleLeaderboard, "discovered", len(ids), "static", len(static))
	return ids
}

// discoverActiveSeason découvre la saison CSR active en essayant tour à tour
// plusieurs playlists de référence. Le scraper rend le menu de saisons via une URL
// (saison-graine FIXE × playlist) : si la playlist n'était pas classée dans la
// saison-graine, la page renvoie 404 — cas NOMINAL, pas une panne. Plutôt que
// d'abandonner tout le cycle sur la première playlist qui échoue (ancien
// comportement, source d'une ERROR quotidienne en prod, réf triage B3.1), on tente
// les candidates suivantes et on retient le premier succès.
//
// Ordre des candidats : les playlists STATIQUES (classées de longue date, donc les
// plus susceptibles d'exister dans la saison-graine) d'abord, puis les playlists
// découvertes dynamiquement. Doublons et entrées vides retirés.
//
// Dégradation (DC-B2) : si TOUTES les candidates échouent — état rare et
// auto-résolutif (page-graine Waypoint globalement indisponible) — on émet UN WARN
// agrégé + un compteur expvar `world_leaderboard_season_discovery_failed_total`,
// jamais une ERROR récurrente pour un cas attendu. Le dernier snapshot append-only
// reste servi entre-temps.
func (c *WorldLeaderboardCron) discoverActiveSeason(ctx context.Context, titleSlug string, candidateLists ...[]string) (string, bool) {
	var lastErr error
	tried := 0
	for _, pl := range dedupeNonEmpty(candidateLists...) {
		season, err := c.scraper.FetchActiveSeason(ctx, pl)
		if err == nil {
			if tried > 0 {
				slog.DebugContext(ctx, "world_leaderboard_cron: saison active découverte après repli sur une playlist de référence",
					"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
					"playlist", pl, "season", season, "skipped", tried)
			}
			return season, true
		}
		lastErr = err
		tried++
		// Chaque échec de candidate est NOMINAL tant qu'une autre peut rendre la page
		// (404 « non classée dans la saison-graine » le plus souvent) : Debug, avec
		// l'erreur pour diagnostic — la synthèse est loguée en fin de boucle si besoin.
		slog.DebugContext(ctx, "world_leaderboard_cron: playlist de référence sans page-graine — essai suivant",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "playlist", pl, "err", err)
	}
	observability.IncCounter("world_leaderboard_season_discovery_failed_total")
	slog.WarnContext(ctx, "world_leaderboard_cron: saison active indécouvrable sur toutes les playlists candidates — cycle ignoré "+
		"(page-graine Waypoint indisponible ; le dernier snapshot append-only reste servi ; auto-résolutif)",
		"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "candidates", tried, "err", lastErr)
	return "", false
}

// dedupeNonEmpty aplatit plusieurs listes en une seule, retire les entrées vides et
// les doublons, en préservant l'ordre de première apparition (la première liste est
// prioritaire).
func dedupeNonEmpty(lists ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, l := range lists {
		for _, s := range l {
			if s == "" {
				continue
			}
			if _, dup := seen[s]; dup {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// discoverSeasons récupère la liste des saisons (nom d'Operation EN + FR) du menu
// déroulant Waypoint pour alimenter season_catalog. Best-effort : une découverte en
// échec (ou vide) renvoie nil — le cycle CSR n'est jamais compromis par les saisons.
func (c *WorldLeaderboardCron) discoverSeasons(ctx context.Context, refPlaylistID string) []domain.WorldSeasonRef {
	seasons, err := c.scraper.FetchSeasons(ctx, refPlaylistID)
	if err != nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: découverte saisons échouée — season_catalog non rafraîchi",
			"module", logging.ModuleLeaderboard, "err", err)
		return nil
	}
	return seasons
}

// enrich agrège les stats des joueurs de la saison active (via l'enricher
// multi-tokens) et les persiste en append-only. No-op si l'enricher est absent.
// Lecture des gamertags hors lease writer ; writer acquis uniquement pour l'INSERT
// (même discipline de fenêtre RW minimale que le scrape CSR).
func (c *WorldLeaderboardCron) enrich(ctx context.Context, season string) {
	if c.enricher == nil {
		return
	}
	start := time.Now()

	players, err := c.seasonPlayers(ctx, season)
	if err != nil {
		slog.ErrorContext(ctx, "world_leaderboard_cron: lecture joueurs saison échouée — enrichissement ignoré",
			"module", logging.ModuleLeaderboard, "season", season, "err", err)
		return
	}
	if len(players) == 0 {
		slog.InfoContext(ctx, "world_leaderboard_cron: aucun joueur à enrichir",
			"module", logging.ModuleLeaderboard, "season", season)
		return
	}

	stats, errs := c.enricher.EnrichSeason(ctx, season, players)
	if len(errs) > 0 {
		slog.WarnContext(ctx, "world_leaderboard_cron: enrichissement partiel (erreurs par joueur)",
			"module", logging.ModuleLeaderboard, "season", season,
			"players", len(players), "failures", len(errs), "first_err", errs[0])
	}
	if len(stats) == 0 {
		slog.WarnContext(ctx, "world_leaderboard_cron: aucune stat agrégée — rien à persister",
			"module", logging.ModuleLeaderboard, "season", season)
		return
	}

	inserted, err := c.persistStats(ctx, stats)
	if err != nil {
		slog.ErrorContext(ctx, "world_leaderboard_cron: persistance stats échouée",
			"module", logging.ModuleLeaderboard, "season", season, "err", err)
		return
	}
	slog.InfoContext(ctx, "world_leaderboard_cron: enrichissement terminé",
		"module", logging.ModuleLeaderboard, "season", season,
		"players", len(players), "rows", inserted, "duration", time.Since(start))
}

// seasonPlayers lit les joueurs distincts de la saison (gamertag + xuid, reader RO).
func (c *WorldLeaderboardCron) seasonPlayers(ctx context.Context, season string) ([]domain.WorldPlayerRef, error) {
	db, release, err := c.provider.Get(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	// Top N par playlist = profondeur affichée : l'enrichissement auto ne fetch pas
	// les rangs jamais montrés (cf. duckdb.WorldLeaderboardTopN).
	return duckdb.WorldSeasonPlayers(ctx, db, season, duckdb.WorldLeaderboardTopN)
}

// persistStats acquiert le writer shared et insère les stats agrégées (append-only).
func (c *WorldLeaderboardCron) persistStats(ctx context.Context, stats []domain.WorldPlayerSeasonStats) (int, error) {
	wh, err := c.provider.AcquireWriter(ctxkeys.WithDBWriterLabel(ctx, "world_enrich_stats"))
	if err != nil {
		return 0, err
	}
	defer wh.Release()
	return duckdb.InsertPlayerSeasonStats(ctx, wh.DB(), stats)
}

// snapshotIsFresh indique si un snapshot de la saison date de moins de freshness.
// En cas d'erreur de lecture, retourne false (on préfère re-scraper que sauter).
func (c *WorldLeaderboardCron) snapshotIsFresh(ctx context.Context, season string) bool {
	db, release, err := c.provider.Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: lecture fraîcheur impossible (shared reader)",
			"module", logging.ModuleLeaderboard, "err", err)
		return false
	}
	defer release()
	age, ok, err := duckdb.WorldCSRSnapshotAge(ctx, db, season)
	if err != nil || !ok {
		return false
	}
	return age < c.freshness
}

// scrapeAll scrape chaque playlist pour la saison donnée et concatène les entrées.
// Best-effort par playlist. Retourne aussi le total scrapé (pour le log).
func (c *WorldLeaderboardCron) scrapeAll(ctx context.Context, season string, playlists []string) ([]domain.LeaderboardEntry, int) {
	all := make([]domain.LeaderboardEntry, 0, len(playlists)*c.limit)
	total := 0
	for _, pl := range playlists {
		entries, err := c.scraper.FetchCSRLeaderboard(ctx, season, pl, c.limit)
		if err != nil {
			slog.ErrorContext(ctx, "world_leaderboard_cron: scrape playlist échoué",
				"module", logging.ModuleLeaderboard, "season", season, "playlist", pl, "err", err)
			continue
		}
		// Plancher de cohérence : un scrape non vide mais suspicieusement court est
		// traité comme un glitch (page tronquée / markup) et ignoré — on ne persiste
		// pas un snapshot partiel, les données précédentes restent affichées.
		if len(entries) > 0 && len(entries) < c.minEntries {
			// Auto-réparé : on ne persiste pas, le snapshot précédent reste affiché.
			// INFO (pas WARN) — c'est un comportement nominal de robustesse, pas une
			// anomalie à investiguer.
			slog.InfoContext(ctx, "world_leaderboard_cron: snapshot playlist trop court — ignoré (glitch probable, snapshot précédent conservé)",
				"module", logging.ModuleLeaderboard, "season", season, "playlist", pl,
				"entries", len(entries), "min", c.minEntries)
			continue
		}
		total += len(entries)
		all = append(all, entries...)
	}
	return all, total
}

// persist acquiert le writer shared et insère les entrées en append-only sous le
// slug du titre courant (PMT-7 write-path : capability-gated par l'appelant
// runOnceForTitle — seuls les titres déclarant CapWorldLeaderboard arrivent ici).
// Upsert aussi season_catalog (C2) dans la MÊME fenêtre writer — best-effort : un
// échec de l'upsert saisons est loggé mais ne fait pas échouer le snapshot CSR.
func (c *WorldLeaderboardCron) persist(ctx context.Context, titleSlug string, entries []domain.LeaderboardEntry, seasons []domain.WorldSeasonRef) (int, error) {
	wh, err := c.provider.AcquireWriter(ctxkeys.WithDBWriterLabel(ctx, "world_leaderboard_snapshot"))
	if err != nil {
		return 0, err
	}
	defer wh.Release()
	inserted, err := duckdb.InsertWorldCSRSnapshot(ctx, wh.DB(), titleSlug, entries)
	if err != nil {
		return inserted, err
	}
	if len(seasons) > 0 {
		if n, serr := ops.RefreshSeasonCatalog(ctx, wh.DB(), titleSlug, seasons); serr != nil {
			slog.WarnContext(ctx, "world_leaderboard_cron: upsert season_catalog échoué (snapshot CSR conservé)",
				"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "err", serr)
		} else {
			slog.DebugContext(ctx, "world_leaderboard_cron: season_catalog rafraîchi",
				"module", logging.ModuleLeaderboard, "titleSlug", titleSlug, "seasons", n)
		}
	}
	return inserted, nil
}
