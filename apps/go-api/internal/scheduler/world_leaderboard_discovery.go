// Package scheduler — world_leaderboard_discovery.go : phase de DÉCOUVERTE du cron
// classement mondial (world_leaderboard_cron.go), extraite pour tenir les seuils de
// taille de fichier. Aucune logique de scrape ni de persistance ici.
//
// Ce que la découverte doit résoudre avant qu'un cycle puisse scraper quoi que ce
// soit : QUELLE saison est active, et QUELLES playlists classées existent — deux
// informations que seule la page publique Halo Waypoint expose.
//
// Trois chemins, du plus riche au plus robuste :
//   - la page-graine (saison DÉJÀ persistée × playlist de référence) rend le menu
//     déroulant complet : saisons + playlists. C'est la source nominale ;
//   - à défaut, les playlists retombent sur la liste statique (jamais zéro playlist) ;
//   - à défaut de toute page-graine, la saison active se lit dans le header Location
//     de la racine des classements (FetchActiveSeasonByRedirect) : aucun paramètre en
//     entrée, donc insensible au retrait d'une saison du site.
//
// Ce dernier repli existe parce que la découverte par graine est un POINT FIXE
// MORTEL quand la saison-graine disparaît de Waypoint : le cron a tourné 267 cycles
// à vide (2026-07-13 → 2026-09-03) sans jamais pouvoir guérir seul. D'où aussi
// l'escalade WARN → ERROR sur les échecs CONSÉCUTIFS (noteSeasonDiscoveryFailure) :
// une panne durable de la découverte doit se voir, pas se noyer dans le bruit.
package scheduler

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	// maxSilentSeasonDiscoveryFails : nombre de cycles CONSÉCUTIFS sans saison
	// découvrable tolérés en WARN. Au-delà (strictement plus), le log passe en
	// ERROR : un échec isolé est un hoquet Waypoint, mais une série signifie que
	// le classement mondial ne se met plus à jour du tout — état vécu 267 cycles
	// (2026-07-13 → 2026-09-03) sans qu'aucune ERROR ne le signale.
	maxSilentSeasonDiscoveryFails = 3
	// seasonDiscoveryStreakMetric : jauge expvar (/debug/vars → levelup.<nom>) du
	// nombre de cycles consécutifs sans saison découvrable. 0 = dernier cycle OK.
	seasonDiscoveryStreakMetric = "world_leaderboard_season_discovery_fail_streak"
)

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
// Repli SANS page-graine (LB-1.2) : quand TOUTES les candidates échouent, la cause
// n'est pas forcément un hoquet — la saison-graine elle-même peut avoir été RETIRÉE
// du site (csrseason13-2 en 2026-09 : 404 sur toutes ses playlists). La graine étant
// la dernière saison persistée, l'état est alors un POINT FIXE MORT : aucune
// découverte ne peut plus aboutir, et le cron a tourné 267 cycles à vide. On tente
// donc `FetchActiveSeasonByRedirect`, qui lit la saison active dans le header
// Location de la racine des classements — aucune saison ni playlist en entrée, donc
// insensible au retrait d'une saison. Un repli réussi alimente le cycle exactement
// comme une découverte nominale.
//
// Dégradation (DC-B2) : si le repli échoue AUSSI, on émet un WARN agrégé + le
// compteur expvar `world_leaderboard_season_discovery_failed_total` ; la série
// d'échecs consécutifs est escaladée en ERROR (cf. noteSeasonDiscoveryFailure). Le
// dernier snapshot append-only reste servi entre-temps.
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
			c.noteSeasonDiscoverySuccess()
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

	season, err := c.scraper.FetchActiveSeasonByRedirect(ctx)
	if err == nil {
		slog.InfoContext(ctx, "world_leaderboard_cron: saison active obtenue par le repli sans page-graine (redirection Waypoint) "+
			"— la saison-graine ne rend plus aucune page",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
			"season", season, "candidates", tried, "seed_err", lastErr)
		c.noteSeasonDiscoverySuccess()
		return season, true
	}

	c.noteSeasonDiscoveryFailure(ctx, titleSlug, tried, lastErr, err)
	return "", false
}

// noteSeasonDiscoverySuccess remet à zéro la série d'échecs de découverte (champ +
// jauge expvar), pour que l'escalade ERROR ne se déclenche que sur une panne DURABLE.
func (c *WorldLeaderboardCron) noteSeasonDiscoverySuccess() {
	c.seasonDiscoveryFails.Store(0)
	observability.SetInt(seasonDiscoveryStreakMetric, 0)
}

// noteSeasonDiscoveryFailure comptabilise un cycle sans saison découvrable et logue
// la dégradation. Sous le seuil : WARN (hoquet Waypoint, le dernier snapshot reste
// servi). Au-delà de maxSilentSeasonDiscoveryFails cycles CONSÉCUTIFS : ERROR — le
// classement mondial ne se met plus à jour et aucun repli n'y peut plus rien, donc
// une intervention est requise (le WARN quotidien, lui, se noie dans le bruit).
func (c *WorldLeaderboardCron) noteSeasonDiscoveryFailure(ctx context.Context, titleSlug string, tried int, seedErr, redirectErr error) {
	observability.IncCounter("world_leaderboard_season_discovery_failed_total")
	streak := c.seasonDiscoveryFails.Add(1)
	observability.SetInt(seasonDiscoveryStreakMetric, streak)

	attrs := []any{
		"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
		"candidates", tried, "consecutive_failures", streak,
		"seed_err", seedErr, "redirect_err", redirectErr,
	}
	if streak > maxSilentSeasonDiscoveryFails {
		slog.ErrorContext(ctx, "world_leaderboard_cron: saison active indécouvrable depuis plusieurs cycles consécutifs — "+
			"le classement mondial N'EST PLUS mis à jour (page-graine ET repli par redirection en échec ; "+
			"vérifier que www.halowaypoint.com/halo-infinite/leaderboards redirige toujours vers la saison active)", attrs...)
		return
	}
	slog.WarnContext(ctx, "world_leaderboard_cron: saison active indécouvrable (page-graine et repli par redirection en échec) — "+
		"cycle ignoré ; le dernier snapshot append-only reste servi, nouvelle tentative au prochain cycle", attrs...)
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
