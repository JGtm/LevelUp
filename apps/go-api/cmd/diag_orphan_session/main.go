//go:build cgo

// diag_orphan_session — diagnostic one-shot pour identifier les matchs qui
// apparaissent dans le bucket "(no session)" du chart teammates.04
// (Performance d'escouade par session). Cause habituelle : match importé
// sans backfill session_recalc → player_match_enrichment.session_label
// reste NULL.
//
// Usage : go run -tags cgo ./cmd/diag_orphan_session [smallhalla]
//
// L'argument optionnel filtre les matchs par map_name ILIKE %arg%.
// Sans argument, liste tous les matchs sans session_label sur tous les
// joueurs.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

func main() {
	mapFilter := ""
	probeMatchID := ""
	if len(os.Args) > 1 {
		mapFilter = os.Args[1]
	}
	if len(os.Args) > 2 {
		probeMatchID = os.Args[2]
	}

	sharedPath := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	playersDir := "../../data/titles/halo_infinite/players"

	if _, err := os.Stat(sharedPath); err != nil {
		log.Fatalf("shared DB introuvable : %s", sharedPath)
	}

	db := openDuckDB(sharedPath)
	defer db.Close()

	// 1. Matchs candidats dans shared.match_registry
	candidates := loadCandidateMatches(db, mapFilter)
	if len(candidates) == 0 {
		if mapFilter != "" {
			fmt.Printf("Aucun match avec map_name ILIKE %%%s%%\n", mapFilter)
		} else {
			fmt.Println("Aucun match dans match_registry")
		}
		return
	}

	fmt.Printf("=== %d match(s) candidat(s) dans match_registry ===\n", len(candidates))
	for _, m := range candidates {
		fmt.Printf("  match_id=%s  start=%s  map=%s  playlist=%s  pair=%s\n",
			m.matchID, m.startTime, m.mapName, m.playlistName, m.pairName)
	}

	// 2. Pour chaque match, lister les participants (xuid+gamertag)
	fmt.Println()
	fmt.Println("=== Participants par match ===")
	for _, m := range candidates {
		ps := loadParticipants(db, m.matchID)
		fmt.Printf("  %s : %d participants\n", m.matchID, len(ps))
		for _, p := range ps {
			fmt.Printf("    - xuid=%s  gamertag=%s  team=%d  outcome=%d\n",
				p.xuid, p.gamertag, p.teamID, p.outcome)
		}
	}

	// 3. Pour chaque player DB, vérifier session_label sur ces matchs
	fmt.Println()
	fmt.Println("=== Enrichment session_label par joueur (player_match_enrichment) ===")
	entries, err := os.ReadDir(playersDir)
	if err != nil {
		log.Fatalf("lecture playersDir: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		statsPath := filepath.Join(playersDir, gamertag, "stats.duckdb")
		if _, err := os.Stat(statsPath); err != nil {
			continue
		}
		pdb := openDuckDB(statsPath)
		fmt.Printf("  %s :\n", gamertag)
		for _, m := range candidates {
			lbl, hasRow := loadEnrichmentLabel(pdb, m.matchID)
			switch {
			case !hasRow:
				fmt.Printf("    %s  → AUCUNE LIGNE dans player_match_enrichment ⚠\n", m.matchID)
			case lbl == "":
				fmt.Printf("    %s  → session_label NULL/vide ⚠\n", m.matchID)
			default:
				fmt.Printf("    %s  → session_label=%q ✓\n", m.matchID, lbl)
			}
		}
		pdb.Close()
	}

	// 4. Scan global : tous les matchs orphelins par joueur (participant en
	// shared mais pme absent/NULL session_label) — toutes maps confondues.
	fmt.Println()
	fmt.Println("=== Scan global : matchs orphelins par joueur (toutes maps) ===")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		statsPath := filepath.Join(playersDir, gamertag, "stats.duckdb")
		if _, err := os.Stat(statsPath); err != nil {
			continue
		}
		// Récupérer xuid depuis xuid_aliases shared
		var xuid string
		err := db.QueryRow(
			"SELECT xuid FROM xuid_aliases WHERE gamertag = ? ORDER BY last_seen DESC NULLS LAST LIMIT 1",
			gamertag,
		).Scan(&xuid)
		if err == sql.ErrNoRows {
			fmt.Printf("  %s : pas d'entrée xuid_aliases — skip\n", gamertag)
			continue
		}
		if err != nil {
			log.Fatalf("lookup xuid(%s): %v", gamertag, err)
		}

		// Match-IDs où ce xuid participe (shared)
		rows, err := db.Query(
			`SELECT mp.match_id,
			        COALESCE(CAST(r.start_time_utc AS VARCHAR), CAST(r.start_time AS VARCHAR), ''),
			        COALESCE(r.map_name, '')
			 FROM match_participants mp
			 JOIN match_registry r ON r.match_id = mp.match_id
			 WHERE mp.xuid = ?
			 ORDER BY r.start_time DESC`,
			xuid,
		)
		if err != nil {
			log.Fatalf("scan participants(%s): %v", gamertag, err)
		}
		var sharedMatches []matchRow
		for rows.Next() {
			var m matchRow
			if err := rows.Scan(&m.matchID, &m.startTime, &m.mapName); err != nil {
				log.Fatalf("scan: %v", err)
			}
			sharedMatches = append(sharedMatches, m)
		}
		rows.Close()

		// Pour chacun, vérifier pme
		pdb := openDuckDB(statsPath)
		var orphans []matchRow
		for _, m := range sharedMatches {
			lbl, hasRow := loadEnrichmentLabel(pdb, m.matchID)
			if !hasRow || lbl == "" {
				orphans = append(orphans, m)
			}
		}
		pdb.Close()

		if len(orphans) == 0 {
			fmt.Printf("  %s : %d matchs participants, 0 orphelin ✓\n", gamertag, len(sharedMatches))
			continue
		}
		fmt.Printf("  %s : %d matchs participants, %d ORPHELIN(s) ⚠\n", gamertag, len(sharedMatches), len(orphans))
		for _, m := range orphans {
			fmt.Printf("    - match_id=%s  start=%s  map=%s\n", m.matchID, m.startTime, m.mapName)
		}
	}

	// 5. Probe ciblé sur un match_id donné (deuxième arg) — détail complet
	//    pour préparer un plan de suppression.
	if probeMatchID != "" {
		fmt.Println()
		fmt.Printf("=== Probe ciblé : %s ===\n", probeMatchID)
		probeMatch(db, playersDir, probeMatchID)
	}
}

func probeMatch(db *sql.DB, playersDir, matchID string) {
	// Détails registry
	rows, err := db.Query(`
		SELECT match_id, start_time, start_time_utc, end_time, duration_seconds,
		       map_id, map_name, playlist_id, playlist_name, pair_name,
		       team_0_score, team_1_score
		FROM match_registry WHERE match_id = ?`, matchID)
	if err != nil {
		log.Fatalf("registry probe: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id, st, mapID, mapName, plID, plName, pair sql.NullString
			stUtc, et                                  sql.NullString
			dur, t0, t1                                sql.NullInt64
		)
		_ = rows.Scan(&id, &st, &stUtc, &et, &dur, &mapID, &mapName, &plID, &plName, &pair, &t0, &t1)
		fmt.Printf("  registry: id=%s  start=%v  start_utc=%v  end=%v  dur=%vs\n",
			id.String, st.String, stUtc.String, et.String, dur.Int64)
		fmt.Printf("            map_id=%v  map_name=%q  playlist_id=%v  playlist_name=%q  pair=%q\n",
			mapID.String, mapName.String, plID.String, plName.String, pair.String)
		fmt.Printf("            scores: team_0=%v  team_1=%v\n", t0.Int64, t1.Int64)
	}

	// Comptes des autres tables shared
	for _, tbl := range []string{"match_participants", "medals_earned", "highlight_events", "killer_victim_pairs", "weapon_kills"} {
		var n int
		_ = db.QueryRow("SELECT COUNT(*) FROM "+tbl+" WHERE match_id = ?", matchID).Scan(&n)
		fmt.Printf("  shared.%s : %d ligne(s)\n", tbl, n)
	}

	// Per player DB
	entries, _ := os.ReadDir(playersDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gt := e.Name()
		statsPath := filepath.Join(playersDir, gt, "stats.duckdb")
		if _, err := os.Stat(statsPath); err != nil {
			continue
		}
		pdb := openDuckDB(statsPath)
		fmt.Printf("  player[%s]:\n", gt)
		for _, tbl := range []string{"player_match_enrichment", "personal_score_awards", "match_citations", "media_match_associations", "match_skill_rank"} {
			var n int
			err := pdb.QueryRow("SELECT COUNT(*) FROM "+tbl+" WHERE match_id = ?", matchID).Scan(&n)
			if err != nil {
				fmt.Printf("    %s : ERROR %v\n", tbl, err)
				continue
			}
			fmt.Printf("    %s : %d\n", tbl, n)
		}
		pdb.Close()
	}
}

type matchRow struct {
	matchID, startTime, mapName, playlistName, pairName string
}

type participant struct {
	xuid, gamertag string
	teamID         int
	outcome        int
}

func openDuckDB(path string) *sql.DB {
	// READ_ONLY pour ne pas concurrencer le serveur si tourne en parallèle.
	connector, err := duckdb.NewConnector(path+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("connector(%s): %v", path, err)
	}
	return sql.OpenDB(connector)
}

func loadCandidateMatches(db *sql.DB, mapFilter string) []matchRow {
	q := `
		SELECT match_id,
		       COALESCE(CAST(start_time_utc AS VARCHAR), CAST(start_time AS VARCHAR), ''),
		       COALESCE(map_name, ''),
		       COALESCE(playlist_name, ''),
		       COALESCE(pair_name, '')
		FROM match_registry`
	args := []any{}
	if mapFilter != "" {
		q += " WHERE map_name ILIKE ? OR pair_name ILIKE ?"
		pattern := "%" + mapFilter + "%"
		args = append(args, pattern, pattern)
	}
	q += " ORDER BY start_time DESC LIMIT 50"

	rows, err := db.Query(q, args...)
	if err != nil {
		log.Fatalf("loadCandidateMatches: %v", err)
	}
	defer rows.Close()

	var out []matchRow
	for rows.Next() {
		var m matchRow
		if err := rows.Scan(&m.matchID, &m.startTime, &m.mapName, &m.playlistName, &m.pairName); err != nil {
			log.Fatalf("scan: %v", err)
		}
		out = append(out, m)
	}
	return out
}

func loadParticipants(db *sql.DB, matchID string) []participant {
	rows, err := db.Query(`
		SELECT mp.xuid,
		       COALESCE(va.gamertag, mp.xuid),
		       COALESCE(mp.team_id, 0),
		       COALESCE(mp.outcome, 0)
		FROM match_participants mp
		LEFT JOIN xuid_aliases va ON va.xuid = mp.xuid
		WHERE mp.match_id = ?
		ORDER BY mp.team_id, va.gamertag`, matchID)
	if err != nil {
		log.Fatalf("loadParticipants(%s): %v", matchID, err)
	}
	defer rows.Close()
	var out []participant
	for rows.Next() {
		var p participant
		if err := rows.Scan(&p.xuid, &p.gamertag, &p.teamID, &p.outcome); err != nil {
			log.Fatalf("scan participant: %v", err)
		}
		out = append(out, p)
	}
	return out
}

func loadEnrichmentLabel(pdb *sql.DB, matchID string) (string, bool) {
	var lbl sql.NullString
	err := pdb.QueryRow(
		"SELECT session_label FROM player_match_enrichment WHERE match_id = ?",
		matchID,
	).Scan(&lbl)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		log.Fatalf("loadEnrichmentLabel(%s): %v", matchID, err)
	}
	if !lbl.Valid {
		return "", true
	}
	return lbl.String, true
}
