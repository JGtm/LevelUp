//go:build cgo

// diag_orphan_xuids — analyse temporelle des XUIDs orphelins (sans alias en DB).
//
// Objectif : déterminer si les 84 orphelins viennent des matchs récents
// (régression sync post-bascule Go) ou s'ils sont distribués dans tout
// l'historique. Si concentration récente → re-parser les highlight events
// de ces matchs avec le parser corrigé devrait les résoudre.
//
// Usage : go run -tags cgo ./cmd/diag_orphan_xuids
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"path/filepath"

	"levelup/go-api/internal/analysis"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"

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

	fmt.Println("══ Distribution temporelle des XUIDs orphelins ══")
	fmt.Println()

	// Pour chaque xuid orphelin, son 1er + dernier match observé + nombre total
	rows, err := db.QueryContext(ctx, `
		WITH orphans AS (
			SELECT DISTINCT mp.xuid
			FROM match_participants mp
			LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
			WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
			  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
		)
		SELECT
			o.xuid,
			COUNT(DISTINCT mp.match_id) AS match_count,
			MIN(`+analysis.SQLStartTimeCanonical("r")+`)::VARCHAR AS first_seen,
			MAX(`+analysis.SQLStartTimeCanonical("r")+`)::VARCHAR AS last_seen
		FROM orphans o
		JOIN match_participants mp ON mp.xuid = o.xuid
		LEFT JOIN match_registry r ON r.match_id = mp.match_id
		GROUP BY o.xuid
		ORDER BY last_seen DESC NULLS LAST
	`)
	if err != nil {
		log.Fatalf("orphan query: %v", err)
	}
	defer rows.Close()

	type orphan struct {
		xuid       string
		matchCount int
		firstSeen  string
		lastSeen   string
	}
	var orphans []orphan
	for rows.Next() {
		var o orphan
		var first, last sql.NullString
		_ = rows.Scan(&o.xuid, &o.matchCount, &first, &last)
		if first.Valid {
			o.firstSeen = first.String[:10] // YYYY-MM-DD
		}
		if last.Valid {
			o.lastSeen = last.String[:10]
		}
		orphans = append(orphans, o)
	}

	// Bucketize par mois de last_seen
	buckets := map[string]int{}
	for _, o := range orphans {
		if o.lastSeen != "" {
			bucket := o.lastSeen[:7] // YYYY-MM
			buckets[bucket]++
		}
	}

	fmt.Printf("Total orphelins : %d\n\n", len(orphans))
	fmt.Println("Distribution par mois (last_seen) :")
	// Tri par mois desc
	keys := []string{}
	for k := range buckets {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] > keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	for _, k := range keys {
		bar := ""
		for i := 0; i < buckets[k]; i++ {
			bar += "█"
		}
		fmt.Printf("  %s : %3d  %s\n", k, buckets[k], bar)
	}

	// Calcul : combien apparaissent UNIQUEMENT en matchs post-régression
	// (régression bascule Go : on prend ~26 avril 2026 comme pivot conservateur)
	const pivotDate = "2026-04-23"
	postReg, preReg, mixed := 0, 0, 0
	for _, o := range orphans {
		if o.firstSeen >= pivotDate {
			postReg++
		} else if o.lastSeen < pivotDate {
			preReg++
		} else {
			mixed++
		}
	}
	fmt.Println()
	fmt.Printf("Pivot régression bascule Go : %s\n", pivotDate)
	fmt.Printf("  ▸ orphelins post-régression UNIQUEMENT  : %3d (%5.1f%%)\n", postReg, pct(postReg, len(orphans)))
	fmt.Printf("  ▸ orphelins pré-régression UNIQUEMENT   : %3d (%5.1f%%)\n", preReg, pct(preReg, len(orphans)))
	fmt.Printf("  ▸ orphelins à cheval                   : %3d (%5.1f%%)\n", mixed, pct(mixed, len(orphans)))

	// Top 10 plus récents
	fmt.Println()
	fmt.Println("Top 10 orphelins les plus récents :")
	fmt.Printf("  %-20s %-12s %-12s %s\n", "xuid", "first_seen", "last_seen", "matches")
	for i, o := range orphans {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-20s %-12s %-12s %d\n", o.xuid, o.firstSeen, o.lastSeen, o.matchCount)
	}

	// Combien apparaissent dans des matchs avec highlight_events parsés (donc
	// candidats à la résolution via re-parse / extraction event.Gamertag)
	var inEventsMatches int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT mp.xuid)
		FROM match_participants mp
		LEFT JOIN xuid_aliases xa ON xa.xuid = mp.xuid
		WHERE `+analysis.SQLIsNotBotCol("mp.xuid")+`
		  AND (xa.xuid IS NULL OR xa.gamertag IS NULL OR xa.gamertag = '')
		  AND EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = mp.match_id)
	`).Scan(&inEventsMatches)
	fmt.Println()
	fmt.Printf("Orphelins dont au moins 1 match a des highlight_events parsés : %d / %d\n",
		inEventsMatches, len(orphans))
	fmt.Println("→ candidats potentiels à un re-parse (le parser corrigé peut")
	fmt.Println("  extraire event.Gamertag absent de match_participants).")
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(total)
}
