//go:build cgo

// diag_audit_crash — diag pour audit post-crash 2026-05-22.
// Cible 4 questions :
//   - Q4 : weapon_kills/match_registry.backfill_completed pour
//     de3cec8b-edf1-4edc-ad87-830369e0a358 et 0941d737-1fb4-4a11-8a9a-169624911729
//   - Q6 : count matchs avec scoreboard 0 sous la query Q12 originale
//     (toutes les rows match_participants ayant kills=0 AND deaths=0 AND assists=0
//     AND personal_score=0)
//
// Read-only. Sortie texte console.
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"path/filepath"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

var probeMatches = []string{
	"de3cec8b-edf1-4edc-ad87-830369e0a358",
	"0941d737-1fb4-4a11-8a9a-169624911729",
}

func main() {
	dataRoot := "../../data"
	sharedPath := filepath.Join(dataRoot, "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")

	connector, err := duckdb.NewConnector(sharedPath+"?access_mode=READ_ONLY", func(execer driver.ExecerContext) error {
		_, e := execer.ExecContext(context.Background(), "SET TimeZone='"+tz+"'", nil)
		return e
	})
	if err != nil {
		log.Fatalf("open shared: %v", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	ctx := context.Background()

	// === Section 1 : backfill_completed + counts pour les matchs probe ===
	fmt.Println("══ Section 1 : matchs probes ══")
	for _, mid := range probeMatches {
		fmt.Printf("\n— match %s\n", mid)
		var bc *int64
		var startTime *string
		var modeName *string
		err := db.QueryRowContext(ctx, `
			SELECT backfill_completed, COALESCE(start_time_utc::VARCHAR, start_time::VARCHAR), pair_name
			FROM match_registry WHERE match_id = ?`, mid).Scan(&bc, &startTime, &modeName)
		if err != nil {
			fmt.Printf("  registry: %v\n", err)
		} else {
			b := int64(0)
			if bc != nil {
				b = *bc
			}
			fmt.Printf("  match_registry.backfill_completed = %d (0x%X)\n", b, b)
			fmt.Printf("    bit Skill(4)         : %v\n", (b&4) != 0)
			fmt.Printf("    bit Events(1<<16)    : %v\n", (b&(1<<16)) != 0)
			fmt.Printf("    bit Medals(1<<17)    : %v\n", (b&(1<<17)) != 0)
			fmt.Printf("    bit Weapons(1<<19)   : %v\n", (b&(1<<19)) != 0)
			fmt.Printf("    bit Participants(512): %v\n", (b&512) != 0)
			if startTime != nil {
				fmt.Printf("  start_time_utc = %s\n", *startTime)
			}
			if modeName != nil {
				fmt.Printf("  pair_name = %s\n", *modeName)
			}
		}

		// Counts
		var cParts, cWeapon, cMedals, cEvents, cKV int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_participants WHERE match_id = ?", mid).Scan(&cParts)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM weapon_kills WHERE match_id = ?", mid).Scan(&cWeapon)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM medals_earned WHERE match_id = ?", mid).Scan(&cMedals)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM highlight_events WHERE match_id = ?", mid).Scan(&cEvents)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM killer_victim_pairs WHERE match_id = ?", mid).Scan(&cKV)
		fmt.Printf("  count rows : participants=%d  weapon_kills=%d  medals_earned=%d  highlight_events=%d  killer_victim_pairs=%d\n",
			cParts, cWeapon, cMedals, cEvents, cKV)

		// Bug ART : compare indexed vs scan
		var idxParts, scanParts int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_participants WHERE match_id = ?", mid).Scan(&idxParts)
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?", mid).Scan(&scanParts)
		if idxParts != scanParts {
			fmt.Printf("  !! ART INDEX BUG : indexed=%d scan=%d (diff=%d)\n", idxParts, scanParts, scanParts-idxParts)
		}
	}

	// === Section 2 : impact Q12 ORIGINAL avec WHERE filtrant ===
	fmt.Println("\n══ Section 2 : matchs où ANCIENNE Q12 (WHERE kills+deaths+assists+score=0) renvoie 0 row scoreboard ══")
	// Compter les match_id ayant >=1 participant, mais où TOUS les participants
	// ont kills=0 AND deaths=0 AND assists=0 AND personal_score=0
	// (équivalent à l'ancien filtre kills > 0 OR deaths > 0 OR ...)
	var totalMatches int
	_ = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT match_id) FROM match_participants").Scan(&totalMatches)
	fmt.Printf("Total match_id distincts avec ≥1 participant : %d\n", totalMatches)

	rows, err := db.QueryContext(ctx, `
		WITH per_match AS (
			SELECT match_id,
			       SUM(COALESCE(kills,0))           AS sum_k,
			       SUM(COALESCE(deaths,0))          AS sum_d,
			       SUM(COALESCE(assists,0))         AS sum_a,
			       SUM(COALESCE(personal_score,0))  AS sum_ps,
			       COUNT(*) AS n
			FROM match_participants
			GROUP BY match_id
		)
		SELECT match_id, n, sum_k, sum_d, sum_a, sum_ps
		FROM per_match
		WHERE sum_k=0 AND sum_d=0 AND sum_a=0 AND sum_ps=0
		ORDER BY match_id
		LIMIT 50`)
	if err != nil {
		log.Fatalf("query Q12 impact: %v", err)
	}
	defer rows.Close()
	count := 0
	var samples []string
	for rows.Next() {
		var mid string
		var n int
		var sk, sd, sa, sps float64
		if err := rows.Scan(&mid, &n, &sk, &sd, &sa, &sps); err != nil {
			log.Fatal(err)
		}
		count++
		if len(samples) < 10 {
			samples = append(samples, fmt.Sprintf("  %s n=%d k=%.0f d=%.0f a=%.0f ps=%.0f", mid, n, sk, sd, sa, sps))
		}
	}
	// Re-run COUNT seul
	var totalImpact int
	_ = db.QueryRowContext(ctx, `
		WITH per_match AS (
			SELECT match_id,
			       SUM(COALESCE(kills,0))+SUM(COALESCE(deaths,0))+SUM(COALESCE(assists,0))+SUM(COALESCE(personal_score,0)) AS total
			FROM match_participants GROUP BY match_id)
		SELECT COUNT(*) FROM per_match WHERE total = 0`).Scan(&totalImpact)
	fmt.Printf("Matchs impactés par ANCIEN WHERE (kills>0 OR deaths>0 OR ... ) : %d / %d (%.1f%%)\n",
		totalImpact, totalMatches, float64(totalImpact)*100/float64(totalMatches))
	fmt.Println("Premiers 10 exemples :")
	for _, s := range samples {
		fmt.Println(s)
	}

	// === Section 3 : ART probe global sur match_participants ===
	fmt.Println("\n══ Section 3 : ART probe global match_participants (50 samples) ══")
	probeRows, err := db.QueryContext(ctx, `
		SELECT match_id FROM match_participants
		WHERE match_id IS NOT NULL
		ORDER BY random() LIMIT 50`)
	if err != nil {
		log.Fatal(err)
	}
	var samplesMID []string
	for probeRows.Next() {
		var m string
		probeRows.Scan(&m)
		samplesMID = append(samplesMID, m)
	}
	probeRows.Close()
	divergences := 0
	for _, m := range samplesMID {
		var ci, cs int
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_participants WHERE match_id = ?", m).Scan(&ci)
		db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_participants WHERE match_id || '' = ?", m).Scan(&cs)
		if ci != cs {
			divergences++
			if divergences <= 5 {
				fmt.Printf("  divergent %s : indexed=%d scan=%d (missing %d)\n", m, ci, cs, cs-ci)
			}
		}
	}
	fmt.Printf("Divergences détectées sur 50 samples : %d\n", divergences)
}
