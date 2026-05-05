//go:build cgo

// diag_intensity_timeline — vérifie l'orientation de highlight_events.time_ms
// (0 = début du match, ou compte à rebours ?).
//
// Compare pour quelques matchs :
//   - duration_seconds × 1000 (durée canonique du match en ms)
//   - max(time_ms) sur highlight_events
//   - min(time_ms) sur highlight_events
//
// Si max(time_ms) ≈ duration_ms → time_ms est elapsed (start=0).
// Si max(time_ms) ≪ duration_ms → time_ms est tronqué (suspect).
// Si min(time_ms) > 0 et max(time_ms) ≈ duration_ms → on perd la 1re seconde.
//
// Usage : go run -tags cgo ./cmd/diag_intensity_timeline
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

func main() {
	const dbPath = "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

	connector, err := duckdb.NewConnector(dbPath+"?access_mode=READ_ONLY", nil)
	if err != nil {
		log.Fatalf("connector: %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	ctx := context.Background()

	q := `
SELECT
    he.match_id,
    r.duration_seconds * 1000 AS dur_ms,
    MIN(he.time_ms)            AS min_t,
    MAX(he.time_ms)            AS max_t,
    COUNT(*)                   AS n_events
FROM highlight_events he
JOIN match_registry r ON r.match_id = he.match_id
WHERE r.duration_seconds > 60
GROUP BY he.match_id, r.duration_seconds
HAVING COUNT(*) > 5
ORDER BY r.duration_seconds DESC
LIMIT 10`

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	fmt.Printf("%-40s %10s %10s %10s %8s %s\n", "match_id", "dur_ms", "min_t", "max_t", "n", "verdict")
	for rows.Next() {
		var mid string
		var durMs, minT, maxT, n int64
		if err := rows.Scan(&mid, &durMs, &minT, &maxT, &n); err != nil {
			log.Fatalf("scan: %v", err)
		}
		verdict := "?"
		switch {
		case maxT >= durMs-2000 && maxT <= durMs+2000:
			verdict = "OK (elapsed: max≈duration)"
		case maxT < durMs/2:
			verdict = "SUSPECT (max<<duration — countdown?)"
		case minT > durMs/2:
			verdict = "SUSPECT (min>>0 — countdown?)"
		default:
			verdict = fmt.Sprintf("partiel (max=%d%% de duration)", 100*maxT/durMs)
		}
		fmt.Printf("%-40s %10d %10d %10d %8d %s\n", mid, durMs, minT, maxT, n, verdict)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}

	// Distribution par bucket pour les 3 premiers matchs : confirme que
	// l'orientation est bien start→end (bucket 0 = début).
	fmt.Println("\n--- Distribution des kills par bucket (10%) sur 3 matchs ---")
	q2 := `
WITH match_dur AS (
    SELECT he.match_id,
           r.duration_seconds * 1000 AS dur_ms
    FROM highlight_events he
    JOIN match_registry r ON r.match_id = he.match_id
    WHERE r.duration_seconds > 60 AND he.event_type = 'kill'
    GROUP BY he.match_id, r.duration_seconds
    HAVING COUNT(*) > 30
    ORDER BY r.duration_seconds DESC
    LIMIT 3
)
SELECT he.match_id,
       LEAST(9, CAST(he.time_ms * 10 / md.dur_ms AS INTEGER)) AS bucket,
       COUNT(*) AS n_kills
FROM highlight_events he
JOIN match_dur md USING (match_id)
WHERE he.event_type = 'kill'
GROUP BY he.match_id, bucket
ORDER BY he.match_id, bucket`
	rows2, err := db.QueryContext(ctx, q2)
	if err != nil {
		log.Fatalf("query2: %v", err)
	}
	defer rows2.Close()
	curMID := ""
	for rows2.Next() {
		var mid string
		var bucket, n int
		if err := rows2.Scan(&mid, &bucket, &n); err != nil {
			log.Fatalf("scan2: %v", err)
		}
		if mid != curMID {
			fmt.Printf("\n%s\n  ", mid)
			curMID = mid
		}
		fmt.Printf("[%d-%d%%]=%d  ", bucket*10, (bucket+1)*10, n)
	}
	fmt.Println()
	_ = driver.Value(0)
}
