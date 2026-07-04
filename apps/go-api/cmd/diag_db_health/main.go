//go:build cgo

// diag_db_health — audit complet de la santé des DBs LevelUp.
//
// Inspecte shared_matches_v2.duckdb + metadata.duckdb + xbox_aliases.duckdb
// global + chaque player stats.duckdb pour détecter :
//
//   - Données stale (UUIDs bruts dans map_name, alias manquants)
//   - Bitmasks "menteurs" (MBitEvents set mais 0 highlight_events)
//   - URLs garbage post-cleanup (`/Waypoint/file/images/...` censées être vides)
//   - Sessions à 1 match (probablement non recalculées)
//   - Couverture asset_translations / map_images_registry
//
// Read-only, tolère les locks du serveur Air en cours.
//
// Usage : go run -tags cgo ./cmd/diag_db_health
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	dataRoot := "../../data"
	if len(os.Args) > 1 {
		dataRoot = os.Args[1]
	}
	titleDir := filepath.Join(dataRoot, "titles", "halo_infinite")

	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println(" AUDIT BASE DE DONNÉES LEVELUP — santé multi-DB")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()

	auditShared(filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb"))
	fmt.Println()
	auditMetadata(filepath.Join(titleDir, "warehouse", "metadata.duckdb"))
	fmt.Println()
	auditGlobalAliases(filepath.Join(dataRoot, "global", "xbox_aliases.duckdb"))
	fmt.Println()
	auditPlayers(filepath.Join(titleDir, "players"))
}

func openRO(path string) *sql.DB {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		return nil
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil
	}
	return db
}

func auditShared(path string) {
	fmt.Println("┌─ SHARED (shared_matches_v2.duckdb) ───────────────────────────")
	db := openRO(path)
	if db == nil {
		fmt.Println("│ ✗ DB lockée ou inaccessible")
		fmt.Println("└──────────────────────────────────────────────────────────────")
		return
	}
	defer db.Close()
	ctx := context.Background()

	var totalMatches int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_registry`).Scan(&totalMatches)
	fmt.Printf("│ match_registry total                    : %d matchs\n", totalMatches)

	// Maps avec UUID brut comme map_name (sync n'a pas résolu)
	var rawUUIDMaps int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_registry
		WHERE map_name ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
	`).Scan(&rawUUIDMaps)
	fmt.Printf("│ match_registry.map_name = UUID brut     : %d  %s\n", rawUUIDMaps, healthMark(rawUUIDMaps == 0))

	// Pair_name UUID brut
	var rawUUIDPairs int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM match_registry
		WHERE pair_name ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
	`).Scan(&rawUUIDPairs)
	fmt.Printf("│ match_registry.pair_name = UUID brut    : %d  %s\n", rawUUIDPairs, healthMark(rawUUIDPairs == 0))

	// Couverture map_id
	var nullMapIDs int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_registry WHERE map_id IS NULL OR map_id = ''`).Scan(&nullMapIDs)
	fmt.Printf("│ match_registry.map_id NULL ou vide      : %d  %s\n", nullMapIDs, healthMark(nullMapIDs == 0))

	fmt.Println("│")
	fmt.Println("│ ── match_participants ──")
	var totalParts, gtNull, gtIsXuid, mmrNull int
	_ = db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			SUM(CASE WHEN gamertag IS NULL OR gamertag = '' THEN 1 ELSE 0 END),
			SUM(CASE WHEN gamertag = xuid AND `+analysis.SQLIsNotBotCol("xuid")+` THEN 1 ELSE 0 END),
			SUM(CASE WHEN team_mmr IS NULL THEN 1 ELSE 0 END)
		FROM match_participants
	`).Scan(&totalParts, &gtNull, &gtIsXuid, &mmrNull)
	fmt.Printf("│ total participants                      : %d\n", totalParts)
	fmt.Printf("│ gamertag NULL ou vide                   : %d (%.1f%%)  %s\n", gtNull, pct(gtNull, totalParts), healthMark(pct(gtNull, totalParts) < 5))
	fmt.Printf("│ gamertag == xuid (alias jamais résolu)  : %d (%.1f%%)  %s\n", gtIsXuid, pct(gtIsXuid, totalParts), healthMark(pct(gtIsXuid, totalParts) < 10))
	fmt.Printf("│ team_mmr NULL                           : %d (%.1f%%)\n", mmrNull, pct(mmrNull, totalParts))

	fmt.Println("│")
	fmt.Println("│ ── highlight events / weapons / kv ──")
	var matchesWithEvents, matchesWithoutEvents, matchesWithWeapons, matchesWithoutWeapons int
	_ = db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id) THEN 1 ELSE 0 END),
			SUM(CASE WHEN NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id) THEN 1 ELSE 0 END)
		FROM match_registry r
	`).Scan(&matchesWithEvents, &matchesWithoutEvents)
	_ = db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id) THEN 1 ELSE 0 END),
			SUM(CASE WHEN NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id) THEN 1 ELSE 0 END)
		FROM match_registry r
	`).Scan(&matchesWithWeapons, &matchesWithoutWeapons)
	fmt.Printf("│ matchs avec highlight_events            : %d (%.1f%%)\n", matchesWithEvents, pct(matchesWithEvents, totalMatches))
	fmt.Printf("│ matchs SANS highlight_events            : %d (%.1f%%)\n", matchesWithoutEvents, pct(matchesWithoutEvents, totalMatches))
	fmt.Printf("│ matchs avec weapon_kills                : %d (%.1f%%)\n", matchesWithWeapons, pct(matchesWithWeapons, totalMatches))
	fmt.Printf("│ matchs SANS weapon_kills                : %d (%.1f%%)\n", matchesWithoutWeapons, pct(matchesWithoutWeapons, totalMatches))

	// Bitmasks menteurs : MBitEvents set (bit 16) mais 0 highlight_events
	const mbitEvents = 1 << 16
	const mbitWeaponKills = 1 << 21
	var liarEvents, liarWeapons int
	_ = db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
	`, mbitEvents)).Scan(&liarEvents)
	_ = db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COUNT(*) FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM weapon_kills w WHERE w.match_id = r.match_id)
	`, mbitWeaponKills)).Scan(&liarWeapons)
	fmt.Printf("│ bits MENTEURS MBitEvents (16) sans data : %d  %s\n", liarEvents, healthMark(liarEvents == 0))
	fmt.Printf("│ bits MENTEURS MBitWeaponKills (21)      : %d  %s\n", liarWeapons, healthMark(liarWeapons == 0))

	fmt.Println("│")
	fmt.Println("│ ── xuid_aliases (shared) ──")
	var aliasesTotal, aliasesBots int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM xuid_aliases`).Scan(&aliasesTotal)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM xuid_aliases WHERE `+analysis.SQLIsBotCol("xuid")+``).Scan(&aliasesBots)
	fmt.Printf("│ total alias                             : %d (dont %d bots)\n", aliasesTotal, aliasesBots)

	// Combien de xuids participants n'ont aucun alias quelque part
	var orphanXuids int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND xa.xuid IS NULL
		  AND (mp.gamertag IS NULL OR mp.gamertag = '' OR mp.gamertag = mp.xuid)
	`).Scan(&orphanXuids)
	fmt.Printf("│ xuids orphelins (alias absent partout)  : %d  %s\n", orphanXuids, healthMark(orphanXuids < 10))

	fmt.Println("└──────────────────────────────────────────────────────────────")
}

func auditMetadata(path string) {
	fmt.Println("┌─ METADATA (metadata.duckdb) ──────────────────────────────────")
	db := openRO(path)
	if db == nil {
		fmt.Println("│ ✗ DB lockée ou inaccessible")
		fmt.Println("└──────────────────────────────────────────────────────────────")
		return
	}
	defer db.Close()
	ctx := context.Background()

	// asset_translations par asset_type (langs distinctes)
	rows, err := db.QueryContext(ctx, `
		SELECT asset_type, COUNT(DISTINCT asset_id), COUNT(DISTINCT lang)
		FROM asset_translations
		GROUP BY asset_type ORDER BY asset_type
	`)
	if err == nil {
		fmt.Println("│ ── asset_translations ──")
		for rows.Next() {
			var t string
			var ids, langs int
			_ = rows.Scan(&t, &ids, &langs)
			fmt.Printf("│   %-20s : %d assets × %d langs\n", t, ids, langs)
		}
		rows.Close()
	}

	// map_images_registry
	var mapImagesTotal int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM map_images_registry`).Scan(&mapImagesTotal)
	fmt.Printf("│ map_images_registry                     : %d entrées\n", mapImagesTotal)

	// Maps en asset_translations (en-US) qui ne sont pas dans map_images_registry
	var unindexedMaps int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT at.asset_id)
		FROM asset_translations at
		LEFT JOIN map_images_registry mir
		    ON mir.title_id = 'halo_infinite' AND mir.map_id = at.asset_id
		WHERE at.asset_type = 'map' AND at.lang = 'en-US'
		  AND mir.map_id IS NULL
	`).Scan(&unindexedMaps)
	fmt.Printf("│ maps connues mais non indexées          : %d  %s\n", unindexedMaps, healthMark(unindexedMaps == 0))

	// mode_name_tr
	var modeTr int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mode_name_tr WHERE lang = 'fr'`).Scan(&modeTr)
	fmt.Printf("│ mode_name_tr (FR)                       : %d entrées\n", modeTr)

	fmt.Println("└──────────────────────────────────────────────────────────────")
}

func auditGlobalAliases(path string) {
	fmt.Println("┌─ GLOBAL (xbox_aliases.duckdb) ────────────────────────────────")
	db := openRO(path)
	if db == nil {
		fmt.Println("│ ✗ DB lockée ou inaccessible (server tournant ?)")
		fmt.Println("└──────────────────────────────────────────────────────────────")
		return
	}
	defer db.Close()
	ctx := context.Background()

	var total int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM xuid_aliases`).Scan(&total)
	fmt.Printf("│ xuid_aliases global                     : %d entrées\n", total)
	fmt.Println("└──────────────────────────────────────────────────────────────")
}

func auditPlayers(playersDir string) {
	fmt.Println("┌─ PLAYERS (data/titles/halo_infinite/players/*) ───────────────")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		fmt.Printf("│ ✗ erreur lecture : %v\n", err)
		fmt.Println("└──────────────────────────────────────────────────────────────")
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println("│")
		fmt.Printf("│ ▸ %s\n", name)
		auditPlayerDB(filepath.Join(playersDir, name, "stats.duckdb"))
	}
	fmt.Println("└──────────────────────────────────────────────────────────────")
}

func auditPlayerDB(path string) {
	db := openRO(path)
	if db == nil {
		fmt.Println("│   ✗ DB lockée ou inaccessible")
		return
	}
	defer db.Close()
	ctx := context.Background()

	// career_progression (Spartan ID URLs)
	var bannerEmpty, bannerGarbage, bannerOK int
	_ = db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN banner_image_url IS NULL OR banner_image_url = '' THEN 1 ELSE 0 END),
			SUM(CASE WHEN banner_image_url LIKE '%/Waypoint/file/images/%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN banner_image_url LIKE '%/hi/images/file/%' THEN 1 ELSE 0 END)
		FROM career_progression
	`).Scan(&bannerEmpty, &bannerGarbage, &bannerOK)
	fmt.Printf("│   career_progression banner             : empty=%d  garbage=%d  ok=%d  %s\n",
		bannerEmpty, bannerGarbage, bannerOK, healthMark(bannerGarbage == 0))

	var emblemEmpty, emblemGarbage, emblemOK int
	_ = db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN emblem_image_url IS NULL OR emblem_image_url = '' THEN 1 ELSE 0 END),
			SUM(CASE WHEN emblem_image_url LIKE '%/Waypoint/file/images/%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN emblem_image_url LIKE '%/hi/images/file/%' THEN 1 ELSE 0 END)
		FROM career_progression
	`).Scan(&emblemEmpty, &emblemGarbage, &emblemOK)
	fmt.Printf("│   career_progression emblem             : empty=%d  garbage=%d  ok=%d  %s\n",
		emblemEmpty, emblemGarbage, emblemOK, healthMark(emblemGarbage == 0))

	// player_match_enrichment sessions
	var totalEnrich, sessionAssigned, sessionsCount int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_match_enrichment`).Scan(&totalEnrich)
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM player_match_enrichment
		WHERE session_id IS NOT NULL AND session_id != 0
	`).Scan(&sessionAssigned)
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT session_id) FROM player_match_enrichment
		WHERE session_id IS NOT NULL AND session_id != 0
	`).Scan(&sessionsCount)
	fmt.Printf("│   matchs enrichis                       : %d (sessions assignées : %d, distinctes : %d)\n",
		totalEnrich, sessionAssigned, sessionsCount)

	// Sessions à 1 match seul (suspicieux : sync n'a pas regroupé)
	if sessionsCount > 0 {
		var singleSessions int
		_ = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM (
				SELECT session_id, COUNT(*) AS n FROM player_match_enrichment
				WHERE session_id IS NOT NULL AND session_id != 0
				GROUP BY session_id HAVING COUNT(*) = 1
			)
		`).Scan(&singleSessions)
		warn := ""
		if singleSessions > sessionsCount/2 {
			warn = "⚠ probablement non recalculé"
		}
		fmt.Printf("│   sessions à 1 match seul               : %d / %d %s\n", singleSessions, sessionsCount, warn)
	}
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(total)
}

func healthMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "⚠"
}

var _ = strings.TrimSpace // reserved for future use
