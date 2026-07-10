//go:build cgo

// diag_recent_match_sync — bilan des ECRITURES SECONDAIRES pour un (ou plusieurs)
// match_id donné. Conçu pour valider/réfuter l'hypothèse "la sync parallèle
// insère match_registry mais skip silencieusement les writes secondaires".
//
// Pour chaque match : compte les lignes dans participants, highlight_events,
// weapon_kills, killer_victim_pairs, medals_earned, et inspecte xuid_aliases
// (fraction de gamertags résolus vs xuid bruts). Affiche aussi backfill_completed
// décodé en bits et les colonnes meta de match_registry pertinentes.
//
// Usage :
//
//	go run -tags cgo ./cmd/diag_recent_match_sync <match_id> [<match_id>...]
//	go run -tags cgo ./cmd/diag_recent_match_sync --recent N    # N derniers matchs
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	recent := flag.Int("recent", 0, "Inspecter les N derniers matchs au lieu de match_id explicites")
	summary := flag.Int("summary", 0, "Tableau résumé sur les N derniers matchs (1 ligne / match)")
	flag.Parse()

	dataRoot := "../../data/titles/halo_infinite"
	sharedPath := filepath.Join(dataRoot, "warehouse", "shared_matches_v2.duckdb")
	globalPath := "../../data/global/xbox_aliases.duckdb"

	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}

	shared := openRO(sharedPath)
	defer shared.Close()

	var globalDB *sql.DB
	if _, err := os.Stat(globalPath); err == nil {
		gdb, err := openROOptional(globalPath)
		if err != nil {
			fmt.Printf("[INFO] global DB lockée (server tourne ?) — checks globaux skippés : %v\n\n", err)
		} else {
			globalDB = gdb
			defer globalDB.Close()
		}
	} else {
		fmt.Printf("[WARN] global DB ABSENTE : %s — c'est précisément le RootCause #1\n\n", globalPath)
	}

	if *summary > 0 {
		runSummary(shared, *summary)
		return
	}

	// Mode "asset <type> <uuid>" : interroge metadata.duckdb pour voir l'état
	// d'un asset (lookups asset_translations + catalogs).
	if len(flag.Args()) == 3 && flag.Arg(0) == "asset" {
		runAssetInspect(flag.Arg(1), flag.Arg(2))
		return
	}

	var matchIDs []string
	if *recent > 0 {
		matchIDs = loadRecentMatches(shared, *recent)
	} else {
		matchIDs = flag.Args()
	}
	if len(matchIDs) == 0 {
		log.Fatalf("usage: diag_recent_match_sync [--recent N | --summary N | asset <type> <uuid> | <match_id>...]")
	}

	for _, mid := range matchIDs {
		inspectMatch(shared, globalDB, mid)
		fmt.Println()
	}
}

func runSummary(shared *sql.DB, n int) {
	rows, err := shared.Query(`
		SELECT r.match_id,
		       `+analysis.SQLStartTimeCanonical("r")+`::VARCHAR,
		       COALESCE(r.map_name, ''),
		       COALESCE(r.pair_name, ''),
		       (SELECT COUNT(*) FROM highlight_events h WHERE h.match_id = r.match_id),
		       (SELECT COUNT(*) FROM weapon_kills w WHERE w.match_id = r.match_id),
		       (SELECT COUNT(*) FROM killer_victim_pairs k WHERE k.match_id = r.match_id),
		       COALESCE(r.backfill_completed, 0)
		FROM match_registry r
		ORDER BY `+analysis.SQLStartTimeCanonical("r")+` DESC NULLS LAST
		LIMIT ?`, n)
	if err != nil {
		log.Fatalf("summary: %v", err)
	}
	defer rows.Close()
	fmt.Printf("%-19s %-6s %-6s %-6s %-22s %-22s %s\n", "start_time", "he", "wk", "kv", "map_name", "pair_name", "bf")
	fmt.Println(strings.Repeat("-", 100))
	for rows.Next() {
		var mid, st, mn, pn string
		var he, wk, kv int
		var bf int64
		_ = rows.Scan(&mid, &st, &mn, &pn, &he, &wk, &kv, &bf)
		mark := " "
		if he == 0 || wk == 0 || kv == 0 {
			mark = "✗"
		}
		stShort := st
		if len(stShort) > 19 {
			stShort = stShort[:19]
		}
		fmt.Printf("%s %s %-6d %-6d %-6d %-22s %-22s %d\n",
			mark, stShort, he, wk, kv, trunc(mn, 22), trunc(pn, 22), bf)
	}
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func runAssetInspect(assetType, assetID string) {
	metaPath := "../../data/titles/halo_infinite/warehouse/metadata.duckdb"
	meta, err := openROOptional(metaPath)
	if err != nil {
		fmt.Printf("[ERR] metadata DB lockée ou inaccessible : %v\n", err)
		fmt.Printf("Astuce : arrêter le serveur Air avant de relancer.\n")
		return
	}
	defer meta.Close()

	fmt.Printf("=== asset_type=%s asset_id=%s ===\n\n", assetType, assetID)

	// 1. asset_translations — toutes les langs présentes pour cet asset_id
	fmt.Println("--- asset_translations ---")
	rows, err := meta.Query(`
		SELECT lang, name, COALESCE(description, ''), fetched_at
		FROM asset_translations
		WHERE asset_id = ? AND asset_type = ?
		ORDER BY lang`, assetID, assetType)
	if err != nil {
		fmt.Printf("  ERR: %v\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var lang, name, desc string
			var fetched sql.NullString
			_ = rows.Scan(&lang, &name, &desc, &fetched)
			fmt.Printf("  lang=%-8s name=%-30s fetched_at=%s\n", lang, name, nstr(fetched))
			count++
		}
		if count == 0 {
			fmt.Println("  (aucune entrée)")
		}
	}

	// 2. Catalog correspondant
	fmt.Println()
	catalogTable := assetTypeToCatalogTable(assetType)
	if catalogTable == "" {
		fmt.Printf("--- catalog : (asset_type %s sans table catalog connue) ---\n", assetType)
		return
	}
	fmt.Printf("--- %s ---\n", catalogTable)
	idCol := assetTypeToCatalogIDColumn(assetType)
	q := fmt.Sprintf(`SELECT * FROM %s WHERE %s = ?`, catalogTable, idCol)
	rows2, err := meta.Query(q, assetID)
	if err != nil {
		fmt.Printf("  ERR: %v\n", err)
		return
	}
	defer rows2.Close()
	cols, _ := rows2.Columns()
	if rows2.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		_ = rows2.Scan(ptrs...)
		for i, c := range cols {
			fmt.Printf("  %-25s: %v\n", c, vals[i])
		}
	} else {
		fmt.Println("  (asset absent du catalog)")
	}
}

func assetTypeToCatalogTable(t string) string {
	switch t {
	case "map":
		return "maps_catalog"
	case "playlist":
		return "playlists_catalog"
	case "pair":
		return "map_mode_pair_definitions"
	case "game_variant":
		return "game_variants_catalog"
	}
	return ""
}

func assetTypeToCatalogIDColumn(t string) string {
	switch t {
	case "map":
		return "map_id"
	case "playlist":
		return "playlist_id"
	case "pair":
		return "pair_id"
	case "game_variant":
		return "variant_id"
	}
	return "id"
}

func openRO(path string) *sql.DB {
	db, err := openROOptional(path)
	if err != nil {
		log.Fatalf("openRO(%s): %v", path, err)
	}
	return db
}

func openROOptional(path string) (*sql.DB, error) {
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func loadRecentMatches(shared *sql.DB, n int) []string {
	rows, err := shared.Query(`
		SELECT match_id
		FROM match_registry
		ORDER BY `+analysis.SQLStartTimeCanonical("")+` DESC NULLS LAST
		LIMIT ?`, n)
	if err != nil {
		log.Fatalf("loadRecentMatches: %v", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func inspectMatch(shared, globalDB *sql.DB, mid string) {
	fmt.Printf("=== match_id = %s ===\n", mid)

	// 1. match_registry meta
	var (
		startTime                sql.NullString
		mapID, mapName, pairName sql.NullString
		pairNameFR, mapNameFR    sql.NullString
		playlistID, playlistName sql.NullString
		isFirefight, isRanked    sql.NullBool
		backfillCompleted        sql.NullInt64
		eventsLoaded             sql.NullBool
		matchIntensity           sql.NullFloat64
	)
	err := shared.QueryRow(`
		SELECT
			`+analysis.SQLStartTimeCanonical("")+`::VARCHAR,
			map_id, map_name, pair_name,
			pair_name_fr, map_name_fr,
			playlist_id, playlist_name,
			is_firefight, is_ranked,
			backfill_completed,
			events_loaded,
			match_intensity
		FROM match_registry WHERE match_id = ?`, mid).Scan(
		&startTime, &mapID, &mapName, &pairName,
		&pairNameFR, &mapNameFR,
		&playlistID, &playlistName,
		&isFirefight, &isRanked,
		&backfillCompleted, &eventsLoaded, &matchIntensity,
	)
	if err != nil {
		fmt.Printf("  [match_registry] ABSENT : %v\n", err)
		return
	}
	fmt.Printf("  start_time     : %s\n", nstr(startTime))
	fmt.Printf("  map_id         : %s\n", flag2(mapID))
	fmt.Printf("  map_name       : %s\n", flag2(mapName))
	fmt.Printf("  map_name_fr    : %s\n", flag2(mapNameFR))
	fmt.Printf("  pair_name      : %s\n", flag2(pairName))
	fmt.Printf("  pair_name_fr   : %s\n", flag2(pairNameFR))
	fmt.Printf("  playlist_id    : %s\n", flag2(playlistID))
	fmt.Printf("  is_firefight   : %v   is_ranked: %v\n", isFirefight.Bool, isRanked.Bool)
	fmt.Printf("  events_loaded  : %v\n", eventsLoaded.Bool)
	fmt.Printf("  match_intensity: %v\n", nfloat(matchIntensity))
	fmt.Printf("  backfill_compl : %d  bits=%s\n", backfillCompleted.Int64, decodeBackfillBits(backfillCompleted.Int64))

	// 2. Participants
	var (
		participantsCount int
		gtResolvedCount   int
		gtRawXuidCount    int
		mmrSetCount       int
		expectedSetCount  int
	)
	_ = shared.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, mid).Scan(&participantsCount)
	_ = shared.QueryRow(`SELECT
		SUM(CASE WHEN gamertag IS NOT NULL AND gamertag != '' AND gamertag NOT LIKE 'bid(%' AND gamertag != xuid THEN 1 ELSE 0 END),
		SUM(CASE WHEN (gamertag IS NULL OR gamertag = '' OR gamertag = xuid) AND `+analysis.SQLIsNotBotCol("xuid")+` THEN 1 ELSE 0 END),
		SUM(CASE WHEN team_mmr IS NOT NULL THEN 1 ELSE 0 END),
		SUM(CASE WHEN kills_expected IS NOT NULL THEN 1 ELSE 0 END)
		FROM match_participants WHERE match_id = ?`, mid).Scan(
		&gtResolvedCount, &gtRawXuidCount, &mmrSetCount, &expectedSetCount)
	fmt.Printf("  match_participants : %d lignes  | gt_résolus=%d  xuid_brut=%d  mmr_set=%d  expected_set=%d\n",
		participantsCount, gtResolvedCount, gtRawXuidCount, mmrSetCount, expectedSetCount)

	// 3. Secondaires
	for _, t := range []struct {
		label, table string
	}{
		{"highlight_events  ", "highlight_events"},
		{"weapon_kills      ", "weapon_kills"},
		{"killer_victim_pairs", "killer_victim_pairs"},
		{"medals_earned     ", "medals_earned"},
	} {
		var n int
		_ = shared.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE match_id = ?", t.table), mid).Scan(&n)
		marker := " "
		if n == 0 {
			marker = "✗"
		}
		fmt.Printf("  %s %s : %d\n", marker, t.label, n)
	}

	// 4. xuid_aliases — combien de joueurs de ce match ont un alias ?
	var aliasCovered, aliasMissing int
	_ = shared.QueryRow(`
		WITH players AS (
			SELECT DISTINCT xuid FROM match_participants
			WHERE match_id = ? AND `+analysis.SQLIsNotBotCol("xuid")+`
		)
		SELECT
			SUM(CASE WHEN xa.gamertag IS NOT NULL AND xa.gamertag != '' THEN 1 ELSE 0 END),
			SUM(CASE WHEN xa.gamertag IS NULL OR xa.gamertag = '' THEN 1 ELSE 0 END)
		FROM players p
		LEFT JOIN xuid_aliases xa ON p.xuid = xa.xuid`, mid).Scan(&aliasCovered, &aliasMissing)
	fmt.Printf("  xuid_aliases (shared) : couverts=%d  manquants=%d\n", aliasCovered, aliasMissing)

	if globalDB != nil {
		var globalCovered, globalMissing int
		// shared a aussi xuid_aliases ; on copie via ATTACH virtuel — ici on
		// requête directement le global avec la liste de xuids.
		xuids, err := shared.Query(`SELECT DISTINCT xuid FROM match_participants WHERE match_id = ? AND `+analysis.SQLIsNotBotCol("xuid")+``, mid)
		if err == nil {
			defer xuids.Close()
			var ids []string
			for xuids.Next() {
				var x string
				_ = xuids.Scan(&x)
				ids = append(ids, x)
			}
			if len(ids) > 0 {
				placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
				args := make([]any, len(ids))
				for i, v := range ids {
					args[i] = v
				}
				covered := 0
				rows, err := globalDB.Query(fmt.Sprintf(`SELECT COUNT(*) FROM xuid_aliases WHERE xuid IN (%s) AND gamertag IS NOT NULL AND gamertag != ''`, placeholders), args...)
				if err == nil {
					defer rows.Close()
					if rows.Next() {
						_ = rows.Scan(&covered)
					}
				}
				globalCovered = covered
				globalMissing = len(ids) - covered
			}
		}
		fmt.Printf("  xuid_aliases (global) : couverts=%d  manquants=%d\n", globalCovered, globalMissing)
	}

	// 5. map_images_registry lookup
	if mapID.Valid && mapID.String != "" {
		// Il faudrait ATTACHer metadata.duckdb mais on prend un raccourci : log juste si map_id valide
		fmt.Printf("  → map_images_registry : à vérifier sur metadata.duckdb (map_id présent)\n")
	} else {
		fmt.Printf("  ✗ map_id NULL/vide — map_images_registry impossible à joindre\n")
	}
}

func decodeBackfillBits(v int64) string {
	// Bits référencés dans le code sync :
	// 1=registry, 2=participants, 4=medals, 8=weapons, 16=highlights, 32=killer_victim,
	// 64=skill, 128=expected, 65536=events_attempted (MBitEvents)
	// Le mapping exact peut varier — on dump les bits pour analyse.
	if v == 0 {
		return "(0)"
	}
	parts := []string{}
	for i := uint(0); i < 32; i++ {
		if v&(1<<i) != 0 {
			parts = append(parts, fmt.Sprintf("bit%d", i))
		}
	}
	return strings.Join(parts, "|")
}

func nstr(s sql.NullString) string {
	if !s.Valid {
		return "<NULL>"
	}
	return s.String
}

func flag2(s sql.NullString) string {
	if !s.Valid || s.String == "" {
		return "<NULL/empty>"
	}
	return s.String
}

func nfloat(f sql.NullFloat64) string {
	if !f.Valid {
		return "<NULL>"
	}
	return fmt.Sprintf("%.3f", f.Float64)
}
