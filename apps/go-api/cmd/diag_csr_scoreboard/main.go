//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	matchID := "74a8614e-fe04-49ff-afba-be9ea5f04d67"
	if len(os.Args) > 1 {
		matchID = os.Args[1]
	}

	sharedPath := "../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
	shared, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("shared open error:", err)
		os.Exit(1)
	}
	defer shared.Close()

	// 1. Infos match
	fmt.Printf("=== Match %s ===\n", matchID[:12])
	var pairName string
	var isRanked bool
	var startTime string
	err = shared.QueryRow(
		`SELECT pair_name, is_ranked, CAST(start_time AS VARCHAR) FROM match_registry WHERE match_id = ?`,
		matchID,
	).Scan(&pairName, &isRanked, &startTime)
	if err != nil {
		fmt.Println("match_registry:", err)
	} else {
		fmt.Printf("  pair_name=%s  is_ranked=%v  start=%s\n\n", pairName, isRanked, startTime[:10])
	}

	// 2. Participants
	fmt.Println("=== match_participants (scoreboard) ===")
	rows, err := shared.Query(
		`SELECT gamertag, xuid, kills, deaths FROM match_participants WHERE match_id = ? ORDER BY team_id, kills DESC`,
		matchID,
	)
	if err != nil {
		fmt.Println("participants error:", err)
	} else {
		for rows.Next() {
			var gt, xuid string
			var k, d int
			rows.Scan(&gt, &xuid, &k, &d)
			fmt.Printf("  %-22s xuid=%-20s K=%d D=%d\n", gt, xuid, k, d)
		}
		rows.Close()
	}

	// 3. match_csrs pour ce match
	fmt.Println("\n=== shared.match_csrs_latest pour ce match ===")
	rows2, err := shared.Query(
		`SELECT xuid, rating_type, tier_label, rating_value, tier, sub_tier FROM match_csrs_latest WHERE match_id = ?`,
		matchID,
	)
	if err != nil {
		fmt.Println("match_csrs error:", err)
	} else {
		count := 0
		for rows2.Next() {
			var xuid, rtype string
			var tierLabel, tier sql.NullString
			var ratingVal sql.NullFloat64
			var subTier sql.NullInt16
			rows2.Scan(&xuid, &rtype, &tierLabel, &ratingVal, &tier, &subTier)
			fmt.Printf("  xuid=%-20s type=%-4s tier=%-12s label=%-18s val=%v\n",
				xuid, rtype, ns(tier), ns(tierLabel), nf(ratingVal))
			count++
		}
		rows2.Close()
		if count == 0 {
			fmt.Println("  (aucune ligne — match_csrs vide pour ce match)")
		} else {
			fmt.Printf("  → %d lignes\n", count)
		}
	}

	// 4. Total match_csrs dans la DB
	var total int
	shared.QueryRow("SELECT COUNT(*) FROM match_csrs").Scan(&total)
	fmt.Printf("\n=== Total match_csrs dans shared DB : %d lignes ===\n", total)
}

func ns(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return "NULL"
}
func nf(n sql.NullFloat64) string {
	if n.Valid {
		return fmt.Sprintf("%.0f", n.Float64)
	}
	return "NULL"
}
