//go:build cgo

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Analyse la convention de stockage de match_registry.start_time :
// - Si naïf=UTC   : delta(capture_utc, start_time) doit être ≈ 0..+durée match
// - Si naïf=Paris : delta doit être ≈ +60min (CET hiver) ou +120min (CEST été)
//
// Usage : go run ./cmd/analyze_media_tz/main.go
func main() {
	socialPath := "../../data/titles/halo_infinite/warehouse/shared_social.duckdb"
	matchesPath := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

	if _, err := os.Stat(socialPath); err != nil {
		log.Fatalf("DB non trouvée : %s", socialPath)
	}

	db, err := sql.Open("duckdb", socialPath+"?access_mode=read_only")
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(fmt.Sprintf(`ATTACH '%s' AS sm (READ_ONLY)`, matchesPath)); err != nil {
		log.Fatalf("attach: %v", err)
	}

	// Pas de SET TimeZone : on travaille en UTC natif pour voir les valeurs brutes.
	rows, err := db.Query(`
		SELECT
			STRFTIME(mr.start_time, '%Y-%m')                                                AS month,
			COUNT(*)                                                                         AS n,
			AVG(EPOCH(mf.capture_start_utc) - EPOCH(mr.start_time))           / 60.0       AS avg_delta_naive_as_utc_min,
			AVG(EPOCH(mf.capture_start_utc) - EPOCH(mr.start_time + INTERVAL 1 HOUR)) / 60.0 AS avg_delta_naive_as_paris_cet_min,
			AVG(mr.duration_seconds) / 60.0                                                 AS avg_match_dur_min
		FROM media_match_associations mma
		JOIN media_files        mf  ON mf.id       = mma.media_file_id
		JOIN sm.match_registry  mr  ON mr.match_id = mma.match_id
		WHERE mma.is_manual = FALSE
		GROUP BY STRFTIME(mr.start_time, '%Y-%m')
		ORDER BY month
	`)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	fmt.Printf("%-10s  %5s  %20s  %20s  %10s\n",
		"month", "n", "naïf=UTC (min)", "naïf=Paris (min)", "dur match")
	fmt.Println("----------------------------------------------------------------------")
	for rows.Next() {
		var month string
		var n int
		var deltaUTC, deltaParis, dur float64
		if err := rows.Scan(&month, &n, &deltaUTC, &deltaParis, &dur); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		fmt.Printf("%-10s  %5d  %+20.1f  %+20.1f  %10.1f\n",
			month, n, deltaUTC, deltaParis, dur)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	// Distribution Paris vs UTC par mois de first_sync_at via real_start_time
	fmt.Println("\n=== CONVENTION start_time : Paris vs UTC (via real_start_time) ===")
	fmt.Println("delta = real_start_time - start_time (naïf)")
	fmt.Println("  ~0min  → start_time stocké en UTC (sync post-fix DuckDB)")
	fmt.Println("  ~60min → start_time stocké en Paris/CET (hiver)")
	fmt.Println("  ~120min→ start_time stocké en Paris/CEST (été)")
	r2, err := db.Query(`
		WITH conv AS (
			SELECT
				STRFTIME(mr.first_sync_at, '%Y-%m')               AS sync_month,
				EPOCH(mr.real_start_time) - EPOCH(mr.start_time)  AS delta_s
			FROM sm.match_registry mr
			WHERE mr.real_start_time IS NOT NULL
			  AND mr.start_time     IS NOT NULL
		)
		SELECT
			sync_month,
			COUNT(*)                                           AS n,
			AVG(delta_s) / 60.0                                AS avg_delta_min,
			MIN(delta_s) / 60.0                                AS min_delta_min,
			MAX(delta_s) / 60.0                                AS max_delta_min,
			SUM(CASE WHEN ABS(delta_s) < 1800 THEN 1 ELSE 0 END) AS n_utc,
			SUM(CASE WHEN delta_s BETWEEN 1800 AND 9000 THEN 1 ELSE 0 END) AS n_paris
		FROM conv
		GROUP BY sync_month
		ORDER BY sync_month
	`)
	if err != nil {
		log.Fatalf("query2: %v", err)
	}
	defer r2.Close()
	fmt.Printf("%-10s  %6s  %12s  %12s  %12s  %6s  %6s\n",
		"sync_month", "n", "avg_delta(m)", "min_delta(m)", "max_delta(m)", "n_utc", "n_paris")
	fmt.Println("--------------------------------------------------------------------------------------")
	for r2.Next() {
		var month string
		var n, nUtc, nParis int
		var avg, min2, max2 float64
		if err := r2.Scan(&month, &n, &avg, &min2, &max2, &nUtc, &nParis); err != nil {
			log.Printf("scan2: %v", err)
			continue
		}
		fmt.Printf("%-10s  %6d  %+12.1f  %+12.1f  %+12.1f  %6d  %6d\n",
			month, n, avg, min2, max2, nUtc, nParis)
	}
	if err := r2.Err(); err != nil {
		log.Fatalf("rows2: %v", err)
	}

	// Validation start_time_utc vs real_start_time (doit être ≈0 pour tous)
	fmt.Println("\n=== VALIDATION start_time_utc vs real_start_time ===")
	fmt.Println("delta = real_start_time - start_time_utc (doit être ≈ 0 pour tous)")
	rVal, err := db.Query(`
		SELECT
			STRFTIME(mr.first_sync_at, '%Y-%m')                               AS sync_month,
			COUNT(*)                                                            AS n,
			AVG(ABS(EPOCH(mr.real_start_time) - EPOCH(mr.start_time_utc)))    AS avg_abs_delta_s,
			MAX(ABS(EPOCH(mr.real_start_time) - EPOCH(mr.start_time_utc)))    AS max_abs_delta_s
		FROM sm.match_registry mr
		WHERE mr.real_start_time IS NOT NULL AND mr.start_time_utc IS NOT NULL
		GROUP BY STRFTIME(mr.first_sync_at, '%Y-%m')
		ORDER BY sync_month
	`)
	if err != nil {
		log.Fatalf("queryVal: %v", err)
	}
	defer rVal.Close()
	fmt.Printf("%-10s  %6s  %16s  %16s\n", "sync_month", "n", "avg_abs_delta_s", "max_abs_delta_s")
	fmt.Println("----------------------------------------------------------")
	for rVal.Next() {
		var month string
		var n int
		var avgAbs, maxAbs float64
		if err := rVal.Scan(&month, &n, &avgAbs, &maxAbs); err != nil {
			log.Printf("scanVal: %v", err)
			continue
		}
		status := "OK"
		if maxAbs > 60 {
			status = "WARN"
		}
		fmt.Printf("%-10s  %6d  %16.1f  %16.1f  %s\n", month, n, avgAbs, maxAbs, status)
	}
	if err := rVal.Err(); err != nil {
		log.Fatalf("rowsVal: %v", err)
	}

	// Debug : real_start_time pour les matchs auto-assoc jan 2026
	fmt.Println("\n=== DEBUG real_start_time pour matchs auto-assoc jan 2026 ===")
	rDbg, err := db.Query(`
		SELECT
			mma.media_file_id,
			mr.match_id,
			mr.start_time,
			mr.real_start_time,
			mr.start_time_utc,
			mr.first_sync_at,
			ABS(EPOCH(mr.real_start_time) - EPOCH(mr.start_time)) / 60.0 AS delta_real_start_min
		FROM media_match_associations mma
		JOIN sm.match_registry mr ON mr.match_id = mma.match_id
		WHERE mma.is_manual = FALSE
		  AND STRFTIME(mr.start_time, '%Y-%m') = '2026-01'
		ORDER BY mr.start_time
	`)
	if err != nil {
		log.Fatalf("queryDbg: %v", err)
	}
	defer rDbg.Close()
	fmt.Printf("%-4s  %-20s  %-20s  %-20s  %-20s  %12s\n",
		"mfid", "start_naive", "real_start_naive", "start_time_utc", "first_sync_at", "delta_real(m)")
	for rDbg.Next() {
		var mfid int
		var matchID string
		var start, realStart, startUTC, firstSync sql.NullTime
		var deltaMin sql.NullFloat64
		rDbg.Scan(&mfid, &matchID, &start, &realStart, &startUTC, &firstSync, &deltaMin)
		stStr := "-"
		if start.Valid {
			stStr = start.Time.Format("01-02 15:04:05")
		}
		rsTStr := "-"
		if realStart.Valid {
			rsTStr = realStart.Time.Format("01-02 15:04:05")
		}
		sutcStr := "-"
		if startUTC.Valid {
			sutcStr = startUTC.Time.UTC().Format("01-02 15:04:05Z")
		}
		fsStr := "-"
		if firstSync.Valid {
			fsStr = firstSync.Time.Format("2006-01-02")
		}
		dm := 0.0
		if deltaMin.Valid {
			dm = deltaMin.Float64
		}
		fmt.Printf("%-4d  %-20s  %-20s  %-20s  %-20s  %+12.1f\n",
			mfid, stStr, rsTStr, sutcStr, fsStr, dm)
	}
	if err := rDbg.Err(); err != nil {
		log.Fatalf("rowsDbg: %v", err)
	}

	// Vérification fonctionnelle : captures auto-assoc de jan 2026 dans la fenêtre start_time_utc
	fmt.Println("\n=== CHECK fonctionnel : captures dans [start_time_utc, end_time_utc] ===")
	rCheck, err := db.Query(`
		SELECT
			mf.id,
			mf.capture_start_utc,
			sm.match_registry.start_time_utc,
			sm.match_registry.end_time_utc,
			DATEDIFF('second', sm.match_registry.start_time_utc, mf.capture_start_utc) AS d_start_s,
			DATEDIFF('second', mf.capture_start_utc, sm.match_registry.end_time_utc)   AS d_end_s,
			CASE WHEN mf.capture_start_utc BETWEEN sm.match_registry.start_time_utc AND sm.match_registry.end_time_utc
			     THEN 'IN_WINDOW' ELSE 'OUT' END AS status
		FROM media_match_associations mma
		JOIN media_files       mf ON mf.id       = mma.media_file_id
		JOIN sm.match_registry    ON match_registry.match_id = mma.match_id
		WHERE mma.is_manual = FALSE
		  AND STRFTIME(match_registry.start_time_utc, '%Y-%m') = '2026-01'
		ORDER BY match_registry.start_time_utc
	`)
	if err != nil {
		log.Fatalf("queryCheck: %v", err)
	}
	defer rCheck.Close()
	fmt.Printf("%-4s  %-20s  %-20s  %-20s  %8s  %8s  %-10s\n",
		"id", "capture_utc", "start_utc", "end_utc", "d_start_s", "d_end_s", "status")
	for rCheck.Next() {
		var id int
		var cap, startUTC, endUTC sql.NullTime
		var dStart, dEnd sql.NullInt64
		var status string
		rCheck.Scan(&id, &cap, &startUTC, &endUTC, &dStart, &dEnd, &status)
		capStr := "-"
		if cap.Valid {
			capStr = cap.Time.Format("01-02 15:04:05Z")
		}
		stStr := "-"
		if startUTC.Valid {
			stStr = startUTC.Time.UTC().Format("01-02 15:04:05Z")
		}
		etStr := "-"
		if endUTC.Valid {
			etStr = endUTC.Time.UTC().Format("01-02 15:04:05Z")
		}
		dsv := int64(0)
		if dStart.Valid {
			dsv = dStart.Int64
		}
		dev := int64(0)
		if dEnd.Valid {
			dev = dEnd.Int64
		}
		fmt.Printf("%-4d  %-20s  %-20s  %-20s  %+8d  %+8d  %-10s\n",
			id, capStr, stStr, etStr, dsv, dev, status)
	}
	if err := rCheck.Err(); err != nil {
		log.Fatalf("rowsCheck: %v", err)
	}

	// Toutes les auto-assos de jan 2026 avec capture et start bruts
	fmt.Println("\n=== TOUTES les auto-assos jan 2026 ===")
	r3, err := db.Query(`
		SELECT
			mf.id,
			mf.file_name,
			mf.capture_start_utc,
			mr.start_time,
			mr.end_time,
			mr.duration_seconds,
			DATEDIFF('second', mf.capture_start_utc, mr.start_time) AS dStart_s,
			DATEDIFF('second', mf.capture_start_utc, mr.end_time)   AS dEnd_s
		FROM media_match_associations mma
		JOIN media_files       mf ON mf.id       = mma.media_file_id
		JOIN sm.match_registry mr ON mr.match_id = mma.match_id
		WHERE mma.is_manual = FALSE
		  AND STRFTIME(mr.start_time, '%Y-%m') = '2026-01'
		ORDER BY mr.start_time
	`)
	if err != nil {
		log.Fatalf("query3: %v", err)
	}
	defer r3.Close()
	fmt.Printf("%-4s  %-26s  %-20s  %-20s  %8s  %8s  %5s\n",
		"id", "file_name(trunc)", "capture_utc", "start_naive", "dStart_s", "dEnd_s", "dur")
	for r3.Next() {
		var id, dur int
		var fname string
		var cap, st, et sql.NullTime
		var ds, de sql.NullInt64
		r3.Scan(&id, &fname, &cap, &st, &et, &dur, &ds, &de)
		capStr := "-"
		if cap.Valid {
			capStr = cap.Time.Format("01-02 15:04:05Z")
		}
		stStr := "-"
		if st.Valid {
			stStr = st.Time.Format("01-02 15:04:05")
		}
		if len(fname) > 26 {
			fname = fname[:26]
		}
		dsv := int64(0)
		if ds.Valid {
			dsv = ds.Int64
		}
		dev := int64(0)
		if de.Valid {
			dev = de.Int64
		}
		fmt.Printf("%-4d  %-26s  %-20s  %-20s  %+8d  %+8d  %5d\n",
			id, fname, capStr, stStr, dsv, dev, dur)
	}
}
