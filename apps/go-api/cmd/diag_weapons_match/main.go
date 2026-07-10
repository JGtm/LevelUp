//go:build cgo

// diag_weapons_match — diagnostic ciblé pour un match donné en argument.
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: diag_weapons_match <match_id>")
	}
	matchID := os.Args[1]

	sharedPath := "../../../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	if env := os.Getenv("DIAG_SHARED_DB"); env != "" {
		sharedPath = env
	}
	fmt.Printf("[shared] %s\n", sharedPath)
	shared, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		log.Fatal("shared open: ", err)
	}
	defer shared.Close()

	meta, err2 := sql.Open("duckdb", "../../../../data/titles/halo_infinite/warehouse/metadata.duckdb?access_mode=read_only")
	if err2 != nil {
		log.Printf("[WARN] meta open failed (server lock?): %v — labels will show as NOT FOUND", err2)
		meta = nil
	} else {
		defer meta.Close()
	}

	fmt.Printf("=== Diag weapons for match %s ===\n\n", matchID)

	// List all tables in the shared DB to detect schema differences
	fmt.Println("[SCHEMA] Tables présentes dans shared_matches_v2 :")
	tRows, _ := shared.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = 'main' ORDER BY table_name`)
	for tRows.Next() {
		var tn string
		_ = tRows.Scan(&tn)
		fmt.Printf("  - %s\n", tn)
	}
	tRows.Close()
	fmt.Println()

	var regCount int
	if err := shared.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, matchID).Scan(&regCount); err != nil {
		fmt.Printf("[ERR] match_registry lookup: %v\n", err)
	}
	fmt.Printf("match_registry rows: %d\n", regCount)

	// Bitmask backfill_completed
	var bits sql.NullInt64
	_ = shared.QueryRow(`SELECT backfill_completed FROM match_registry WHERE match_id = ?`, matchID).Scan(&bits)
	if bits.Valid {
		hasWK := (bits.Int64 & (1 << 21)) != 0
		hasNoFilm := (bits.Int64 & (1 << 22)) != 0
		fmt.Printf("backfill_completed = %d (bit21 weapon_kills=%v, bit22 no_film=%v)\n", bits.Int64, hasWK, hasNoFilm)
	} else {
		fmt.Println("backfill_completed = NULL")
	}

	var startTime sql.NullTime
	_ = shared.QueryRow(`SELECT `+analysis.SQLStartTimeCanonical("")+` FROM match_registry WHERE match_id = ?`, matchID).Scan(&startTime)
	if startTime.Valid {
		ageDays := time.Since(startTime.Time).Hours() / 24.0
		fmt.Printf("start_time = %s (âge: %.1f jours)\n", startTime.Time.Format("2006-01-02 15:04 MST"), ageDays)
	}

	var partCount int
	if err := shared.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, matchID).Scan(&partCount); err != nil {
		fmt.Printf("[ERR] match_participants lookup: %v\n", err)
	}
	fmt.Printf("match_participants rows: %d\n", partCount)

	var wkTotal int
	if err := shared.QueryRow(`SELECT COUNT(*) FROM weapon_kills WHERE match_id = ?`, matchID).Scan(&wkTotal); err != nil {
		fmt.Printf("[ERR] weapon_kills lookup: %v\n", err)
	}
	fmt.Printf("weapon_kills rows (total): %d\n", wkTotal)

	// LIKE % match_id pour vérifier qu'il n'y a pas un encoding problème
	var wkLike int
	_ = shared.QueryRow(`SELECT COUNT(*) FROM weapon_kills WHERE match_id LIKE ?`, "%"+matchID[:8]+"%").Scan(&wkLike)
	fmt.Printf("weapon_kills LIKE '%%%s%%': %d\n", matchID[:8], wkLike)

	// Tous les match_id qui commencent par les 8 premiers caractères
	rowsLike, _ := shared.Query(`SELECT DISTINCT match_id, COUNT(*) FROM weapon_kills WHERE match_id LIKE ? GROUP BY match_id`, matchID[:8]+"%")
	if rowsLike != nil {
		fmt.Println("[DEBUG] match_id qui commencent par", matchID[:8], ":")
		for rowsLike.Next() {
			var mid string
			var c int
			_ = rowsLike.Scan(&mid, &c)
			fmt.Printf("  '%s' (len=%d) wk=%d\n", mid, len(mid), c)
		}
		rowsLike.Close()
	}

	// Inspect via la VIEW v_weapon_kills (que Python ET le donut utilisent)
	var vwkTotal int
	_ = shared.QueryRow(`SELECT COUNT(*) FROM v_weapon_kills WHERE match_id = ?`, matchID).Scan(&vwkTotal)
	fmt.Printf("v_weapon_kills rows (total via VIEW): %d\n", vwkTotal)

	// Top weapons via la requête EXACTE du scoreboard
	rowsTop, _ := shared.Query(`
		SELECT xuid, effective_weapon_id, COUNT(*) AS k
		FROM v_weapon_kills
		WHERE match_id = ? AND effective_weapon_id NOT IN (0, 1, 2)
		GROUP BY xuid, effective_weapon_id
		ORDER BY k DESC LIMIT 20`, matchID)
	if rowsTop != nil {
		fmt.Println("[DEBUG] v_weapon_kills sample (match) :")
		for rowsTop.Next() {
			var xu string
			var wid uint64
			var k int
			_ = rowsTop.Scan(&xu, &wid, &k)
			fmt.Printf("  xuid=%-22s effective_weapon_id=%-22d kills=%d\n", xu, wid, k)
		}
		rowsTop.Close()
	}

	// highlight_events — la source des kills pour le pipeline weapons
	var heTotal, heKill int
	_ = shared.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, matchID).Scan(&heTotal)
	_ = shared.QueryRow(`SELECT COUNT(*) FROM highlight_events
		WHERE match_id = ? AND LOWER(COALESCE(event_type,'')) LIKE '%kill%'`, matchID).Scan(&heKill)
	fmt.Printf("highlight_events rows (total): %d (kill events: %d)\n", heTotal, heKill)
	if heTotal > 0 {
		hRows, _ := shared.Query(`
			SELECT event_type, COUNT(*) FROM highlight_events
			WHERE match_id = ? GROUP BY event_type ORDER BY COUNT(*) DESC LIMIT 8`, matchID)
		for hRows.Next() {
			var et string
			var c int
			_ = hRows.Scan(&et, &c)
			fmt.Printf("  event_type=%-30s count=%d\n", et, c)
		}
		hRows.Close()
	}

	var wkFiltered int
	if err := shared.QueryRow(`
		SELECT COUNT(*) FROM weapon_kills
		WHERE match_id = ?
		  AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
	`, matchID).Scan(&wkFiltered); err != nil {
		fmt.Printf("[ERR] weapon_kills filtered lookup: %v\n", err)
	}
	fmt.Printf("weapon_kills rows (NOT IN 0/1/2): %d\n\n", wkFiltered)

	if wkTotal == 0 {
		fmt.Println("[CONCLUSION] weapon_kills VIDE pour ce match — pas de sync/backfill effectué")
		fmt.Println("    => l'expander et la colonne 'Outil de destr.' seront vides par construction")

		// Coverage globale
		var totalMatches, matchesWithWeapons int
		_ = shared.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&totalMatches)
		_ = shared.QueryRow(`SELECT COUNT(DISTINCT match_id) FROM weapon_kills`).Scan(&matchesWithWeapons)
		fmt.Printf("\n[GLOBAL] Coverage weapon_kills : %d / %d matchs (%.1f%%)\n",
			matchesWithWeapons, totalMatches, 100.0*float64(matchesWithWeapons)/float64(totalMatches))

		// FAUX-POSITIFS : bit21 = set mais 0 lignes weapon_kills
		var ghostFlagged, bitSetTotal, bitNoFilmTotal, bitNeitherTotal int
		_ = shared.QueryRow(`
			SELECT COUNT(*) FROM match_registry mr
			WHERE (COALESCE(mr.backfill_completed,0) & ?) != 0
			  AND NOT EXISTS (SELECT 1 FROM weapon_kills wk WHERE wk.match_id = mr.match_id)
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE
		`, 1<<21).Scan(&ghostFlagged)
		_ = shared.QueryRow(`
			SELECT COUNT(*) FROM match_registry mr
			WHERE (COALESCE(mr.backfill_completed,0) & ?) != 0
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE`, 1<<21).Scan(&bitSetTotal)
		_ = shared.QueryRow(`
			SELECT COUNT(*) FROM match_registry mr
			WHERE (COALESCE(mr.backfill_completed,0) & ?) != 0
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE`, 1<<22).Scan(&bitNoFilmTotal)
		_ = shared.QueryRow(`
			SELECT COUNT(*) FROM match_registry mr
			WHERE (COALESCE(mr.backfill_completed,0) & ?) = 0
			  AND (COALESCE(mr.backfill_completed,0) & ?) = 0
			  AND COALESCE(mr.is_firefight, FALSE) = FALSE`, 1<<21, 1<<22).Scan(&bitNeitherTotal)
		fmt.Printf("[BITMASK] bit21 set (weapon_kills done): %d  | bit22 set (no_film): %d  | aucun: %d\n",
			bitSetTotal, bitNoFilmTotal, bitNeitherTotal)
		fmt.Printf("[FAUX-POSITIFS] bit21 set MAIS weapon_kills vide : %d matchs (sur %d bit21 set)\n",
			ghostFlagged, bitSetTotal)

		// Quelques matchs voisins pour voir si c'est un trou ou systémique
		fmt.Println("\n[CONTEXTE] 10 matchs proches en date — état weapon_kills :")
		rows, err := shared.Query(`
			WITH target AS (
				SELECT `+analysis.SQLStartTimeCanonical("")+` AS t
				FROM match_registry WHERE match_id = ?
			)
			SELECT mr.match_id,
			       `+analysis.SQLStartTimeCanonical("mr")+` AS st,
			       COALESCE(mr.pair_name, ''),
			       (SELECT COUNT(*) FROM weapon_kills wk WHERE wk.match_id = mr.match_id) AS wk_count
			FROM match_registry mr, target
			WHERE ABS(EXTRACT(EPOCH FROM (
			    `+analysis.SQLStartTimeCanonical("mr")+` - target.t
			))) < 86400 * 7
			ORDER BY st DESC
			LIMIT 10`, matchID)
		if err != nil || rows == nil {
			fmt.Printf("  (contexte indisponible : %v)\n", err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var mid, mode string
			var st sql.NullTime
			var wk int
			_ = rows.Scan(&mid, &st, &mode, &wk)
			marker := " "
			if mid == matchID {
				marker = ">"
			}
			tlabel := ""
			if st.Valid {
				tlabel = st.Time.Format("2006-01-02 15:04")
			}
			fmt.Printf("  %s match_id=%s  %s  %-30s wk=%d\n", marker, mid[:8], tlabel, mode, wk)
		}
		return
	}

	fmt.Println("--- Distribution par xuid (avant filtre 0/1/2) ---")
	rows, _ := shared.Query(`
		SELECT xuid, COUNT(*) AS n,
		       SUM(CASE WHEN COALESCE(reconciled_as, weapon_id) IN (0,1,2) THEN 1 ELSE 0 END) AS junk,
		       SUM(CASE WHEN COALESCE(reconciled_as, weapon_id) NOT IN (0,1,2) THEN 1 ELSE 0 END) AS valid
		FROM weapon_kills
		WHERE match_id = ?
		GROUP BY xuid
		ORDER BY n DESC`, matchID)
	defer rows.Close()
	for rows.Next() {
		var xuid string
		var n, junk, valid int
		_ = rows.Scan(&xuid, &n, &junk, &valid)
		fmt.Printf("  xuid=%-22s total=%-5d junk(0/1/2)=%-3d valid=%d\n", xuid, n, junk, valid)
	}

	fmt.Println("\n--- Échantillon weapon_id distincts (filtrés) ---")
	rows2, _ := shared.Query(`
		SELECT DISTINCT COALESCE(reconciled_as, weapon_id) AS wid
		FROM weapon_kills
		WHERE match_id = ?
		  AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
		LIMIT 30`, matchID)
	defer rows2.Close()
	var allWIDs []uint64
	for rows2.Next() {
		var wid uint64
		_ = rows2.Scan(&wid)
		allWIDs = append(allWIDs, wid)
	}
	fmt.Printf("Distinct weapon_ids: %d\n", len(allWIDs))
	for _, wid := range allWIDs {
		lbl := "NOT FOUND (meta unavailable)"
		if meta != nil {
			var label sql.NullString
			_ = meta.QueryRow(`SELECT COALESCE(name_fr, name_en) FROM weapon_labels WHERE weapon_id = ?`, wid).Scan(&label)
			lbl = "NOT FOUND"
			if label.Valid {
				lbl = label.String
			}
		}
		fmt.Printf("  weapon_id=%-22d label=%s\n", wid, lbl)
	}

	fmt.Println("\n--- Q12 top_weapons CTE simulation ---")
	rows3, _ := shared.Query(`
		SELECT xuid, wid AS top_weapon_id
		FROM (
			SELECT xuid, COALESCE(reconciled_as, weapon_id) AS wid, COUNT(*) AS wk,
			       ROW_NUMBER() OVER (PARTITION BY xuid ORDER BY COUNT(*) DESC) AS rn
			FROM weapon_kills
			WHERE match_id = ? AND COALESCE(reconciled_as, weapon_id) NOT IN (0, 1, 2)
			GROUP BY xuid, COALESCE(reconciled_as, weapon_id)
		) t WHERE rn = 1
		ORDER BY xuid`, matchID)
	defer rows3.Close()
	for rows3.Next() {
		var xuid string
		var wid uint64
		_ = rows3.Scan(&xuid, &wid)
		lbl := "NOT FOUND (meta unavailable)"
		if meta != nil {
			var label sql.NullString
			_ = meta.QueryRow(`SELECT COALESCE(name_fr, name_en) FROM weapon_labels WHERE weapon_id = ?`, wid).Scan(&label)
			lbl = "NOT FOUND"
			if label.Valid {
				lbl = label.String
			}
		}
		fmt.Printf("  xuid=%-22s top_weapon_id=%-22d label=%s\n", xuid, wid, lbl)
	}
}
