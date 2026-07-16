// Package duckdb — leaderboard_world_repo.go : lecture du classement CSR mondial
// (snapshots scrapés depuis Halo Waypoint) et des classements de stats
// communautaires (agrégation de shared.match_participants).
//
// Écriture des snapshots : InsertWorldCSRSnapshot (INSERT pur, règle ART).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// logModuleLeaderboard route les logs de lecture du classement vers
// logs/leaderboard.log (cf. observability/logging.ModuleLeaderboard).
const logModuleLeaderboard = "leaderboard"

// statLeaderboardMinMatches : nombre minimal de matchs pour figurer dans un
// classement de stats (évite les flukes sur 1-2 parties).
const statLeaderboardMinMatches = 10

// GetCSRWorldLeaderboard lit le dernier snapshot du classement CSR mondial pour
// une saison + playlist depuis world_csr_leaderboard_latest (shared).
// Le tier/sous-palier sont re-dérivés du CSR (source unique domain.DeriveCSRTier).
// is_local = true si le xuid correspond au joueur courant.
func (r *LeaderboardRepo) GetCSRWorldLeaderboard(
	ctx context.Context, titleSlug, season, playlist string, limit int,
) ([]domain.LeaderboardEntry, error) {
	if strings.TrimSpace(titleSlug) == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	if strings.TrimSpace(season) == "" || strings.TrimSpace(playlist) == "" {
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: season et playlist requis")
	}
	if limit <= 0 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: shared reader: %w", err)
	}
	defer release()

	// Le snapshot Halo Waypoint EXPOSE le xuid de chaque joueur (parsé du bloc
	// __NEXT_DATA__ par le scraper, persisté depuis B1) : on le sélectionne pour
	// activer is_local (mise en évidence du joueur courant sur ce classement mondial).
	// Les lignes scrapées AVANT l'ajout de la colonne xuid restent à NULL → COALESCE ''
	// → is_local false, jusqu'au prochain scrape qui les remplit.
	// Masquage des joueurs privés/sans données (historique inaccessible → aucune stat) :
	// on les exclut EN SQL (avant le LIMIT) pour servir un top complet de joueurs
	// exploitables. Best-effort : table absente → set vide → aucun filtre (dégradation).
	//
	// GARDE : on ne masque QUE si la saison a des joueurs enrichis. Une vieille saison
	// entièrement expirée (historique API perdu → 0 enrichi, TOUS marqués privés)
	// laisserait sinon un classement VIDE ; on montre alors le CSR brut.
	noData, ndErr := WorldSeasonNoDataGamertags(ctx, sharedDB, titleSlug, season)
	if ndErr != nil {
		slog.WarnContext(ctx, "GetCSRWorldLeaderboard: lecture world_player_no_data échouée — pas de masquage",
			"module", logModuleLeaderboard, "season", season, "err", ndErr)
		noData = nil
	}
	if len(noData) > 0 {
		hasEnriched, hErr := WorldSeasonHasEnriched(ctx, sharedDB, season)
		if hErr != nil || !hasEnriched {
			// Saison sans aucune stat (expirée) ou lecture en échec → pas de masquage.
			noData = nil
		}
	}
	q := `
		SELECT rank, COALESCE(gamertag, ''), COALESCE(xuid, '') AS xuid, csr_value
		FROM world_csr_leaderboard_latest
		WHERE title_slug = ? AND season_id = ? AND playlist_id = ?`
	args := []any{titleSlug, season, playlist}
	if len(noData) > 0 {
		ph := make([]string, 0, len(noData))
		for gt := range noData {
			ph = append(ph, "?")
			args = append(args, gt)
		}
		q += ` AND gamertag NOT IN (` + strings.Join(ph, ",") + `)`
	}
	q += ` ORDER BY rank ASC LIMIT ?`
	args = append(args, limit)
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "lecture classement CSR mondial échouée", "module", logModuleLeaderboard,
			"season", season, "playlist", playlist, "err", err)
		return nil, fmt.Errorf("GetCSRWorldLeaderboard: query: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LeaderboardEntry, 0, limit)
	for rows.Next() {
		var rank, csr int
		var gamertag, xuid string
		if err := rows.Scan(&rank, &gamertag, &xuid, &csr); err != nil {
			return nil, fmt.Errorf("GetCSRWorldLeaderboard: scan: %w", err)
		}
		tier, subTier := domain.DeriveCSRTier(csr)
		out = append(out, domain.LeaderboardEntry{
			Rank:     rank,
			Gamertag: gamertag,
			XUID:     xuid,
			CSR:      csr,
			CSRValue: csr,
			Tier:     tier,
			SubTier:  subTier,
			Season:   season,
			Playlist: playlist,
			Category: string(domain.LeaderboardCSRWorld),
			Value:    float64(csr),
			IsLocal:  r.isLocalXUID(xuid),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Enrichissement best-effort (Phase B) : stats joueur + delta rang, sur la même
	// connexion shared. Données absentes (avant backfill Phase D) → entrées inchangées.
	r.enrichWorldEntries(ctx, sharedDB, out, season, playlist)
	slog.DebugContext(ctx, "classement CSR mondial lu", "module", logModuleLeaderboard,
		"season", season, "playlist", playlist, "entries", len(out))
	return out, nil
}

// enrichWorldEntries fusionne les stats par joueur (queryWorldPlayerStats) + le
// delta de rang inter-saison dans les entrées du classement. Best-effort : toute
// erreur d'enrichissement est loguée et n'invalide pas le classement de base.
func (r *LeaderboardRepo) enrichWorldEntries(ctx context.Context, db *sql.DB, entries []domain.LeaderboardEntry, season, playlist string) {
	if len(entries) == 0 {
		return
	}
	enriched, err := queryWorldPlayerStats(ctx, db, season, playlist)
	if err != nil {
		slog.WarnContext(ctx, "enrichissement classement mondial échoué (non bloquant)",
			"module", logModuleLeaderboard, "season", season, "playlist", playlist, "err", err)
		return
	}
	byGT := make(map[string]domain.WorldPlayerSeasonStats, len(enriched))
	for _, s := range enriched {
		byGT[strings.ToLower(s.Gamertag)] = s
	}
	prevRanks, err := loadPrevSeasonRanks(ctx, db, playlist, season)
	if err != nil {
		slog.WarnContext(ctx, "delta rang classement mondial échoué (non bloquant)",
			"module", logModuleLeaderboard, "season", season, "playlist", playlist, "err", err)
		prevRanks = nil
	}
	cumul, err := loadCumulativeMatchCounts(ctx, db, season, playlist)
	if err != nil {
		slog.WarnContext(ctx, "matchs cumulés classement mondial échoués (non bloquant)",
			"module", logModuleLeaderboard, "season", season, "playlist", playlist, "err", err)
		cumul = nil
	}
	for i := range entries {
		key := strings.ToLower(entries[i].Gamertag)
		if s, ok := byGT[key]; ok {
			applyWorldEnrichment(&entries[i], s)
		}
		if pr, ok := prevRanks[key]; ok {
			d := pr - entries[i].Rank // positif = a grimpé (rang plus petit = meilleur)
			entries[i].RankDelta = &d
		}
		if c, ok := cumul[key]; ok {
			cc := c
			entries[i].CumulativeMatchCount = &cc
		}
	}
}

// loadCumulativeMatchCounts somme par gamertag les matchs (match_count) sur toutes les
// saisons de rang NUMÉRIQUE <= la saison affichée (un `season_id < ?` SQL serait
// LEXICOGRAPHIQUE → csrseason6-1 < csrseason13-2 faux). Filtré sur la playlist si
// fournie (vide = toutes). Sert la colonne "Matchs" cumulés. Best-effort.
func loadCumulativeMatchCounts(ctx context.Context, db *sql.DB, season, playlist string) (map[string]int, error) {
	all, err := scanIDColumn(ctx, db,
		`SELECT DISTINCT season_id FROM world_player_season_stats_latest WHERE season_id <> ''`)
	if err != nil {
		return nil, err
	}
	cur := worldSeasonRank(season)
	inc := make([]string, 0, len(all))
	for _, s := range all {
		if worldSeasonRank(s) <= cur {
			inc = append(inc, s)
		}
	}
	if len(inc) == 0 {
		return map[string]int{}, nil
	}
	args := make([]any, 0, len(inc)+2)
	args = append(args, defaultLeaderboardTitleSlug)
	ph := make([]string, len(inc))
	for i, s := range inc {
		ph[i] = "?"
		args = append(args, s)
	}
	q := `SELECT gamertag, SUM(match_count) FROM world_player_season_stats_latest
		WHERE title_slug = ? AND season_id IN (` + strings.Join(ph, ",") + `)`
	if playlist != "" {
		q += ` AND playlist_id = ?`
		args = append(args, playlist)
	}
	q += ` GROUP BY gamertag`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("loadCumulativeMatchCounts: query: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var gt string
		var sum int64
		if err := rows.Scan(&gt, &sum); err != nil {
			return nil, fmt.Errorf("loadCumulativeMatchCounts: scan: %w", err)
		}
		out[strings.ToLower(gt)] = int(sum)
	}
	return out, rows.Err()
}

// applyWorldEnrichment recopie compteurs bruts + ratios dérivés + indicateur
// inter-saison d'un WorldPlayerSeasonStats dans une LeaderboardEntry (pointeurs
// frais → chaque entrée a ses propres pointeurs).
func applyWorldEnrichment(e *domain.LeaderboardEntry, s domain.WorldPlayerSeasonStats) {
	mc, wc, lc, tc, dc := s.MatchCount, s.WinCount, s.LossCount, s.TieCount, s.DnfCount
	k, d, a, pt, md := s.Kills, s.Deaths, s.Assists, s.PlaytimeSec, s.MedalCount
	e.MatchCount, e.WinCount, e.LossCount, e.TieCount, e.DnfCount = &mc, &wc, &lc, &tc, &dc
	e.Kills, e.Deaths, e.Assists, e.PlaytimeSec, e.MedalCount = &k, &d, &a, &pt, &md
	// Valeurs natives brutes (sommées) — pointeurs frais.
	kda, acc, dd, dt := s.KDA, s.Accuracy, s.DamageDealt, s.DamageTaken
	e.KDA, e.Accuracy, e.DamageDealt, e.DamageTaken = &kda, &acc, &dd, &dt
	e.WinRate, e.KillsPerMin = s.WinRate, s.KillsPerMin
	e.PrevSeasonID, e.PrevWinRate, e.PrevKDA = s.PrevSeasonID, s.PrevWinRate, s.PrevKDA
	e.KDATrend, e.WinRateTrend = s.KDATrend, s.WinRateTrend
}

// GetWorldLeaderboardCatalog liste les saisons et playlists du classement CSR mondial
// scrappé (world_csr_leaderboard_latest) : on EXPOSE toutes les saisons dont on a un
// classement (rangs + gamertags), y compris les saisons archivées. Chaque saison porte
// un flag Enriched : true si des stats détaillées (world_player_season_stats) existent,
// false si seule la donnée de classement est disponible — historique de matchs au-delà
// de l'horizon de l'API Halo, ex. csrseason3-1 / 4-1 → le front l'affiche en « classement
// seul » avec un badge. Tri du plus récent au plus ancien par NUMÉRO de saison (un ORDER
// BY season_id SQL serait LEXICOGRAPHIQUE : csrseason6-1 > csrseason13-2 car '6' > '1' →
// faux). Les playlists reçoivent un libellé via rankedplaylists (FR si dispo, sinon EN, sinon id brut).
func (r *LeaderboardRepo) GetWorldLeaderboardCatalog(ctx context.Context, titleSlug string) (domain.LeaderboardCatalog, error) {
	if strings.TrimSpace(titleSlug) == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	// Le shared DB est isolé par titre (ADR 0008 : data/titles/<slug>/…), donc les
	// world_*_latest n'y contiennent que les lignes du titre courant — pas de WHERE
	// title_slug redondant ici. Le slug est porté pour la traçabilité (PMT-7).
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: shared reader: %w", err)
	}
	defer release()

	seasons, err := scanCatalogColumn(ctx, sharedDB,
		`SELECT DISTINCT season_id FROM world_csr_leaderboard_latest
		 WHERE season_id <> ''`, nil)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: seasons: %w", err)
	}
	sortSeasonsRecentFirst(seasons)
	// Marque les saisons réellement enrichies (stats détaillées présentes). Celles sans
	// stats restent affichées (classement seul) mais Enriched=false → badge côté front.
	enrichedIDs, err := scanIDColumn(ctx, sharedDB,
		`SELECT DISTINCT season_id FROM world_player_season_stats_latest WHERE season_id <> ''`)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: enriched seasons: %w", err)
	}
	enrichedSet := make(map[string]bool, len(enrichedIDs))
	for _, id := range enrichedIDs {
		enrichedSet[id] = true
	}
	for i := range seasons {
		seasons[i].Enriched = enrichedSet[seasons[i].ID]
	}
	plIDs, err := scanIDColumn(ctx, sharedDB,
		`SELECT DISTINCT playlist_id FROM world_csr_leaderboard_latest
		 WHERE playlist_id <> '' ORDER BY playlist_id`)
	if err != nil {
		return domain.LeaderboardCatalog{}, fmt.Errorf("GetWorldLeaderboardCatalog: playlists: %w", err)
	}
	// Phase F : noms via le catalogue metadata mutualisé, locale-aware (header
	// X-LevelUp-Locale → ctxkeys.Locale). Cascade EN : asset_translations[en] >
	// rankedplaylists EN > name_canonical (EN) > FR > id ; cascade FR :
	// asset_translations[fr] > rankedplaylists FR > name_canonical (EN) > EN > id.
	locale := ctxkeys.Locale(ctx)
	// C2b : libellé autoritatif "Saison N · Nom" depuis season_catalog (scrape
	// Waypoint). Best-effort : catalogue vide (pas encore scrapé) → fallback
	// "Saison N" dérivé du numéro.
	seasonNames, err := LoadSeasonCatalogNames(ctx, sharedDB, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "GetWorldLeaderboardCatalog: season_catalog illisible — libellés dérivés",
			"module", logModuleLeaderboard, "err", err)
		seasonNames = map[string]SeasonName{}
	}
	for i := range seasons {
		seasons[i].DisplayName = SeasonSelectorLabel(locale, seasons[i].ID, seasonNames,
			fallbackSeasonLabel(locale, seasons[i].ID))
	}
	frMap, enMap, canonMap := r.resolvePlaylistNamesFromCatalog(ctx, plIDs)
	playlists := make([]domain.LeaderboardCatalogRef, 0, len(plIDs))
	for _, id := range plIDs {
		playlists = append(playlists, domain.LeaderboardCatalogRef{
			ID: id, DisplayName: playlistName(id, locale, frMap[id], enMap[id], canonMap[id]),
		})
	}
	slog.DebugContext(ctx, "catalogue classement mondial lu", "module", logModuleLeaderboard,
		"titleSlug", titleSlug, "locale", locale,
		"seasons", len(seasons), "playlists", len(playlists),
		"noms_fr_catalogue", len(frMap), "noms_en_catalogue", len(enMap))
	return domain.LeaderboardCatalog{Seasons: seasons, Playlists: playlists}, nil
}

// worldSeasonRank extrait un rang triable d'un id "csrseason{major}-{minor}"
// (major*100 + minor) ; format inconnu → 0. Sert au tri récent-d'abord du catalogue
// (un tri lexicographique mettrait csrseason6-1 avant csrseason13-2).
func worldSeasonRank(id string) int {
	s := strings.TrimPrefix(id, "csrseason")
	major, minor := s, "0"
	if i := strings.IndexByte(s, '-'); i >= 0 {
		major, minor = s[:i], s[i+1:]
	}
	mj, _ := strconv.Atoi(major)
	mn, _ := strconv.Atoi(minor)
	return mj*100 + mn
}

// fallbackSeasonLabel dérive "Saison N" / "Season N" du season_id quand season_catalog
// ne connaît pas la saison (pas encore scrapée). Format inconnu → id brut.
func fallbackSeasonLabel(locale, id string) string {
	major := worldSeasonRank(id) / 100
	if major <= 0 {
		return id
	}
	if locale == "en" {
		return fmt.Sprintf("Season %d", major)
	}
	return fmt.Sprintf("Saison %d", major)
}

// sortSeasonsRecentFirst trie les saisons du plus récent au plus ancien (numérique).
func sortSeasonsRecentFirst(seasons []domain.LeaderboardCatalogRef) {
	sort.SliceStable(seasons, func(i, j int) bool {
		return worldSeasonRank(seasons[i].ID) > worldSeasonRank(seasons[j].ID)
	})
}

// playlistName applique la cascade de résolution d'un nom de playlist, selon la
// locale UI ("en"/"fr"). EN : asset_translations[en] > rankedplaylists EN >
// name_canonical (EN) > FR (officiel/curé) > id. FR (défaut) : asset_translations[fr]
// > rankedplaylists FR > name_canonical (EN) > EN (officiel/curé) > id.
func playlistName(id, locale, frOfficial, enOfficial, canonical string) string {
	pl, hasPL := rankedplaylists.Lookup(id)
	if locale == "en" {
		if enOfficial != "" {
			return enOfficial
		}
		if hasPL && pl.NameEN != "" {
			return pl.NameEN
		}
		if canonical != "" {
			return canonical
		}
		if frOfficial != "" {
			return frOfficial
		}
		if hasPL && pl.NameFR != "" {
			return pl.NameFR
		}
		return id
	}
	if frOfficial != "" {
		return frOfficial
	}
	if hasPL && pl.NameFR != "" {
		return pl.NameFR
	}
	if canonical != "" {
		return canonical
	}
	if enOfficial != "" {
		return enOfficial
	}
	return playlistDisplayName(id) // rankedplaylists EN > id brut
}

// scanIDColumn lit une colonne d'IDs (une seule colonne string) en slice ordonné.
func scanIDColumn(ctx context.Context, db *sql.DB, q string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// resolvePlaylistNamesFromCatalog lit le catalogue metadata MUTUALISÉ et retourne
// trois maps : frMap (asset_translations[fr]) + enMap (asset_translations[en]),
// noms officiels localisés, et canonMap (playlists_catalog.name_canonical, EN brut).
// Vides si metadata indisponible (le caller retombe sur rankedplaylists). Ne touche
// jamais la shared DB.
func (r *LeaderboardRepo) resolvePlaylistNamesFromCatalog(ctx context.Context, ids []string) (frMap, enMap, canonMap map[string]string) {
	frMap, enMap, canonMap = map[string]string{}, map[string]string{}, map[string]string{}
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return frMap, enMap, canonMap
	}
	meta := r.pdb.Metadata
	scan := func(q string, args []any, dst map[string]string) {
		rows, err := meta.QueryRecovered(ctx, q, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) == nil && strings.TrimSpace(name) != "" {
				dst[id] = name
			}
		}
	}
	scan(fmt.Sprintf(
		`SELECT playlist_asset_id, COALESCE(name_canonical, '') FROM playlists_catalog
		 WHERE title_slug = ? AND playlist_asset_id IN (%s)`, Placeholders(len(ids))),
		append([]any{defaultLeaderboardTitleSlug}, ToAnySlice(ids)...), canonMap)
	// asset_translations fr+en en une requête : on route par la colonne lang.
	scanByLang := func(q string, args []any) {
		rows, err := meta.QueryRecovered(ctx, q, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, name, lang string
			if rows.Scan(&id, &name, &lang) == nil && strings.TrimSpace(name) != "" {
				switch lang {
				case "fr":
					frMap[id] = name
				case "en":
					enMap[id] = name
				}
			}
		}
	}
	scanByLang(fmt.Sprintf(
		`SELECT asset_id, name, lang FROM asset_translations
		 WHERE asset_type = 'playlist' AND lang IN ('fr','en') AND asset_id IN (%s)
		   AND name IS NOT NULL AND TRIM(name) <> ''`, Placeholders(len(ids))),
		ToAnySlice(ids))
	return frMap, enMap, canonMap
}

// scanCatalogColumn lit une colonne d'IDs et construit des refs. displayFn
// dérive le libellé depuis l'ID (nil → libellé = ID).
func scanCatalogColumn(ctx context.Context, db *sql.DB, q string, displayFn func(string) string) ([]domain.LeaderboardCatalogRef, error) {
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LeaderboardCatalogRef
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		display := id
		if displayFn != nil {
			display = displayFn(id)
		}
		out = append(out, domain.LeaderboardCatalogRef{ID: id, DisplayName: display})
	}
	return out, rows.Err()
}

// playlistDisplayName résout un asset_id de playlist en libellé (FR > EN > id).
func playlistDisplayName(assetID string) string {
	if pl, ok := rankedplaylists.Lookup(assetID); ok {
		if pl.NameFR != "" {
			return pl.NameFR
		}
		if pl.NameEN != "" {
			return pl.NameEN
		}
	}
	return assetID
}

// statMetric décrit l'expression SQL d'agrégation et l'unité d'une catégorie.
type statMetric struct {
	expr string
	unit string
}

// statMetrics mappe chaque catégorie de stat à son agrégat (pas de magic string).
// GREATEST(...,1) / NULLIF évitent les divisions par zéro.
var statMetrics = map[domain.LeaderboardCategory]statMetric{
	domain.LeaderboardKills:         {"SUM(mp.kills)", ""},
	domain.LeaderboardDeaths:        {"SUM(mp.deaths)", ""},
	domain.LeaderboardAssists:       {"SUM(mp.assists)", ""},
	domain.LeaderboardKillsPerGame:  {"SUM(mp.kills) * 1.0 / COUNT(DISTINCT mp.match_id)", ""},
	domain.LeaderboardKDR:           {"SUM(mp.kills) * 1.0 / GREATEST(SUM(mp.deaths), 1)", ""},
	domain.LeaderboardKDA:           {"((SUM(mp.kills) + SUM(mp.assists) / 3.0) - SUM(mp.deaths)) / GREATEST(COUNT(DISTINCT mp.match_id), 1)", ""},
	domain.LeaderboardAccuracy:      {"SUM(mp.shots_hit) * 100.0 / NULLIF(SUM(mp.shots_fired), 0)", "%"},
	domain.LeaderboardDamage:        {"SUM(mp.damage_dealt)", ""},
	domain.LeaderboardDamagePerGame: {"SUM(mp.damage_dealt) * 1.0 / COUNT(DISTINCT mp.match_id)", ""},
}

// GetStatLeaderboard agrège shared.match_participants par xuid pour une catégorie
// de stat (joueurs réellement croisés). Filtres optionnels :
//   - playlist : ILIKE sur match_registry.playlist_name.
//   - season   : égalité exacte sur match_registry.season_id (format interne
//     "CsrSeasonN", PAS le format Waypoint du classement mondial — les deux
//     domaines de saison sont distincts).
//
// Bots exclus, seuil min de matchs appliqué. Le JOIN match_registry n'est ajouté
// que si au moins un des deux filtres est actif (évite un JOIN inutile sinon).
func (r *LeaderboardRepo) GetStatLeaderboard(
	ctx context.Context, titleSlug string, category domain.LeaderboardCategory, playlist, season string, limit int,
) ([]domain.LeaderboardEntry, error) {
	if strings.TrimSpace(titleSlug) == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	// Agrégation de match_participants : le shared DB est isolé par titre (ADR 0008)
	// → pas de colonne/WHERE title_slug. Le slug est porté pour la traçabilité (PMT-7).
	metric, ok := statMetrics[category]
	if !ok {
		return nil, fmt.Errorf("GetStatLeaderboard: catégorie inconnue %q", category)
	}
	if limit <= 0 {
		limit = 100
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetStatLeaderboard: shared reader: %w", err)
	}
	defer release()

	args := []any{}
	registryWhere := strings.Builder{}
	playlist, season = strings.TrimSpace(playlist), strings.TrimSpace(season)
	if playlist != "" {
		registryWhere.WriteString("AND lower(COALESCE(r.playlist_name, '')) LIKE '%' || lower(?) || '%'\n")
		args = append(args, playlist)
	}
	if season != "" {
		registryWhere.WriteString("AND r.season_id = ?\n")
		args = append(args, season)
	}
	registryJoin := ""
	if registryWhere.Len() > 0 {
		registryJoin = "JOIN match_registry r ON r.match_id = mp.match_id"
	}
	// #nosec G201 -- metric.expr provient d'une allowlist interne (statMetrics), pas d'entrée utilisateur.
	q := fmt.Sprintf(`
		SELECT mp.xuid,
		       COALESCE(vg.gamertag, 'Joueur ' || RIGHT(mp.xuid, 4)) AS gamertag,
		       COUNT(DISTINCT mp.match_id) AS matches,
		       %s AS value
		FROM match_participants mp
		LEFT JOIN v_gamertag_lookup vg ON vg.xuid = mp.xuid
		%s
		WHERE mp.xuid NOT LIKE 'bid(%%'
		%s
		GROUP BY mp.xuid, vg.gamertag
		HAVING COUNT(DISTINCT mp.match_id) >= ? AND value IS NOT NULL
		ORDER BY value DESC
		LIMIT ?`, metric.expr, registryJoin,
		// Masquage Campagne via sous-requête (self-contained : n'ajoute pas la
		// jointure registre conditionnelle ci-dessus). No-op pour Infinite.
		registryWhere.String()+excludeCampaignByMatchID(titleSlug, "mp.match_id"))
	args = append(args, statLeaderboardMinMatches, limit)

	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "lecture classement de stats échouée", "module", logModuleLeaderboard,
			"category", string(category), "playlist", playlist, "season", season, "err", err)
		return nil, fmt.Errorf("GetStatLeaderboard(%s): query: %w", category, err)
	}
	defer rows.Close()

	out := make([]domain.LeaderboardEntry, 0, limit)
	rank := 0
	for rows.Next() {
		var xuid, gamertag string
		var matches int
		var value float64
		if err := rows.Scan(&xuid, &gamertag, &matches, &value); err != nil {
			return nil, fmt.Errorf("GetStatLeaderboard(%s): scan: %w", category, err)
		}
		rank++
		out = append(out, domain.LeaderboardEntry{
			Rank:          rank,
			XUID:          xuid,
			Gamertag:      gamertag,
			Category:      string(category),
			Value:         value,
			Unit:          metric.unit,
			MatchesPlayed: matches,
			IsLocal:       r.isLocalXUID(xuid),
		})
	}
	return out, rows.Err()
}

// isLocalXUID indique si le xuid est celui du joueur courant (mise en évidence).
func (r *LeaderboardRepo) isLocalXUID(xuid string) bool {
	return xuid != "" && xuid == r.pdb.XUID
}

// WorldCSRSnapshotAge retourne l'âge du snapshot le plus récent pour une saison
// (toutes playlists confondues). ok=false si aucun snapshot. Utilisé par le cron
// comme garde-fou de fraîcheur : évite de re-scraper Halo Waypoint à chaque boot /
// hot-reload Air si un snapshot récent existe déjà.
//
// L'âge est calculé ENTIÈREMENT en SQL (CURRENT_TIMESTAMP - max(written_at)) :
// les deux timestamps partagent l'horloge/zone de la DB, ce qui évite le piège
// TZ d'un timestamp DuckDB naïf relu comme UTC puis comparé à time.Now() local
// (cf. reference_timezone_canonical_pattern). `db` peut être RO ou RW.
func WorldCSRSnapshotAge(ctx context.Context, db *sql.DB, seasonID string) (time.Duration, bool, error) {
	const q = `
		SELECT date_part('epoch', CURRENT_TIMESTAMP - max(written_at))
		FROM world_csr_leaderboard_snapshots
		WHERE season_id = ?`
	var ageSeconds sql.NullFloat64
	if err := db.QueryRowContext(ctx, q, seasonID).Scan(&ageSeconds); err != nil {
		return 0, false, fmt.Errorf("WorldCSRSnapshotAge: %w", err)
	}
	if !ageSeconds.Valid {
		return 0, false, nil
	}
	return time.Duration(ageSeconds.Float64 * float64(time.Second)), true, nil
}

// InsertWorldCSRSnapshot persiste un lot d'entrées du classement CSR mondial en
// INSERT pur (règle ART — jamais d'UPDATE) dans world_csr_leaderboard_snapshots.
// `db` est une connexion shared en écriture (cron ou job CLI). Retourne le nombre
// de lignes insérées.
//
// ATOMIQUE : tout le lot est inséré dans une seule transaction → en cas d'échec
// en cours de route, rien n'est commité (pas de demi-snapshot). Garantit aussi un
// `fetched_at` cohérent sur tout le lot (déjà fixé en amont par le scraper), ce
// qui permet à la vue _latest de grouper par batch de scrape.
func InsertWorldCSRSnapshot(ctx context.Context, db *sql.DB, titleSlug string, entries []domain.LeaderboardEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("InsertWorldCSRSnapshot: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op si Commit a réussi ; rollback sinon

	const ins = `
		INSERT INTO world_csr_leaderboard_snapshots
			(title_slug, season_id, playlist_id, rank, gamertag, csr_value, tier_derived, fetched_at, xuid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, e := range entries {
		if _, err := tx.ExecContext(ctx, ins,
			titleSlug, e.Season, e.Playlist, e.Rank, e.Gamertag, e.CSRValue, e.Tier, e.FetchedAt, e.XUID,
		); err != nil {
			return 0, fmt.Errorf("InsertWorldCSRSnapshot (rank %d): %w", e.Rank, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("InsertWorldCSRSnapshot: commit: %w", err)
	}
	return len(entries), nil
}
