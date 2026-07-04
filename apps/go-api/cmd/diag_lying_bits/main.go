//go:build cgo

// diag_lying_bits — distribution temporelle des matchs avec
// MBitEvents=TRUE mais 0 row dans highlight_events (bits "menteurs").
//
// Objectif : déterminer si les 76 bits menteurs sont concentrés sur
// une fenêtre récente (= régression actuelle à corriger) ou répartis
// sur l'historique (= films Halo expirés côté CDN, limite irréductible).
//
// Read-only sur shared_matches_v2.duckdb — compatible serveur tournant.
//
// Usage : go run -tags cgo ./cmd/diag_lying_bits
package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"

	duckdb "github.com/duckdb/duckdb-go/v2"
)

const tz = "Europe/Paris"
const mbitEvents = 1 << 16

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

	fmt.Println("══ Distribution temporelle des bits MENTEURS MBitEvents ══")
	fmt.Println()
	fmt.Printf("Critère : (backfill_completed & %d) != 0 ET aucun row dans highlight_events.\n\n", mbitEvents)

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			r.match_id,
			`+analysis.SQLStartTimeCanonical("r")+`::VARCHAR AS played_at,
			COALESCE(r.pair_name, '')                                            AS pair_name,
			COALESCE(r.map_name, '')                                             AS map_name
		FROM match_registry r
		WHERE (COALESCE(r.backfill_completed, 0) & %d) != 0
		  AND NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)
		ORDER BY played_at DESC NULLS LAST
	`, mbitEvents))
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()

	type m struct {
		matchID, playedAt, pairName, mapName string
	}
	var matches []m
	for rows.Next() {
		var x m
		var played sql.NullString
		_ = rows.Scan(&x.matchID, &played, &x.pairName, &x.mapName)
		if played.Valid {
			x.playedAt = played.String[:10] // YYYY-MM-DD
		}
		matches = append(matches, x)
	}

	// Bucket par mois YYYY-MM
	buckets := map[string]int{}
	for _, x := range matches {
		if x.playedAt != "" {
			buckets[x.playedAt[:7]]++
		}
	}

	fmt.Printf("Total bits menteurs : %d\n\n", len(matches))
	fmt.Println("Distribution par mois (start_time) :")
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	for _, k := range keys {
		bar := ""
		for i := 0; i < buckets[k]; i++ {
			bar += "█"
		}
		fmt.Printf("  %s : %3d  %s\n", k, buckets[k], bar)
	}

	// Comparaison vs pivot bascule Go (23 avril 2026)
	const pivot = "2026-04-23"
	post, pre := 0, 0
	for _, x := range matches {
		if x.playedAt == "" {
			continue
		}
		if x.playedAt >= pivot {
			post++
		} else {
			pre++
		}
	}
	fmt.Println()
	fmt.Printf("Pivot bascule Go : %s\n", pivot)
	fmt.Printf("  ▸ matchs post-pivot : %3d (%5.1f%%)\n", post, pct(post, len(matches)))
	fmt.Printf("  ▸ matchs pré-pivot  : %3d (%5.1f%%)\n", pre, pct(pre, len(matches)))

	fmt.Println()
	fmt.Println("Top 10 matchs cassés les plus récents :")
	fmt.Printf("  %-22s %-12s %-30s %s\n", "match_id", "played_at", "pair_name", "map_name")
	for i, x := range matches {
		if i >= 10 {
			break
		}
		fmt.Printf("  %-22s %-12s %-30s %s\n", x.matchID[:22], x.playedAt, truncate(x.pairName, 30), truncate(x.mapName, 30))
	}
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(total)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
