// Outil de diagnostic temporaire — investigate_matches.
// Usage : go run ./cmd/investigate_matches/
package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const sharedDB = `../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`

var matchIDs = []string{
	"20fcbfe4-5a35-4992-a4b8-4bb7d92b62b6",
	"bc0bdda3-4116-4d08-913d-628285633197",
}

// Q13 exact (copié de queries_match.go) — 17 colonnes, types Go identiques au service.
const q13Full = `
SELECT
    r.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    r.duration_seconds,
    r.map_name,
    r.pair_name,
    r.playlist_name,
    COALESCE(r.is_firefight, FALSE) AS is_firefight,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE
        ELSE FALSE
    END AS is_ranked,
    r.playable_duration_seconds,
    r.map_id,
    r.game_variant_name,
    r.playlist_id,
    r.team_0_score,
    r.team_1_score,
    r.pair_name_fr,
    r.pair_id,
    r.game_variant_id
FROM match_registry r
WHERE r.match_id = ?`

func main() {
	db, err := sql.Open("duckdb", sharedDB+"?access_mode=read_only")
	if err != nil {
		log.Fatal("open:", err)
	}
	defer db.Close()

	for _, mid := range matchIDs {
		fmt.Printf("\n=== %s ===\n", mid)
		investigateMatch(db, mid)
	}
}

func investigateMatch(db *sql.DB, matchID string) {
	// Test Q13 complet avec les MÊMES types Go que le service de production.
	var (
		matchID2           string
		startTime          *time.Time
		durationSeconds    *float64
		mapName            *string
		pairName           *string
		playlistName       *string
		isFirefight        bool
		isRanked           bool
		playableDurSec     *int64
		mapAssetID         *string
		gameVariantName    *string
		playlistAssetID    *string
		team0Score         *int16
		team1Score         *int16
		pairNameFR         *string
		pairAssetID        *string
		gameVariantAssetID *string
	)

	err := db.QueryRow(q13Full, matchID).Scan(
		&matchID2,
		&startTime,
		&durationSeconds,
		&mapName,
		&pairName,
		&playlistName,
		&isFirefight,
		&isRanked,
		&playableDurSec,
		&mapAssetID,
		&gameVariantName,
		&playlistAssetID,
		&team0Score,
		&team1Score,
		&pairNameFR,
		&pairAssetID,
		&gameVariantAssetID,
	)
	if err != nil {
		fmt.Printf("  [Q13 FULL SCAN] ERREUR: %v\n", err)
		return
	}

	ptrStr := func(s *string) string {
		if s == nil {
			return "<nil>"
		}
		return *s
	}
	ptrI16 := func(v *int16) string {
		if v == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d", *v)
	}
	ptrI64 := func(v *int64) string {
		if v == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%d", *v)
	}
	ptrF64 := func(v *float64) string {
		if v == nil {
			return "<nil>"
		}
		return fmt.Sprintf("%.1f", *v)
	}
	ptrTime := func(v *time.Time) string {
		if v == nil {
			return "<nil>"
		}
		return v.String()
	}

	fmt.Printf("  match_id             = %s\n", matchID2)
	fmt.Printf("  start_time           = %s\n", ptrTime(startTime))
	fmt.Printf("  duration_seconds     = %s\n", ptrF64(durationSeconds))
	fmt.Printf("  map_name             = %s\n", ptrStr(mapName))
	fmt.Printf("  pair_name            = %s\n", ptrStr(pairName))
	fmt.Printf("  playlist_name        = %s\n", ptrStr(playlistName))
	fmt.Printf("  is_firefight         = %v\n", isFirefight)
	fmt.Printf("  is_ranked            = %v\n", isRanked)
	fmt.Printf("  playable_dur_sec     = %s\n", ptrI64(playableDurSec))
	fmt.Printf("  map_id               = %s\n", ptrStr(mapAssetID))
	fmt.Printf("  game_variant_name    = %s\n", ptrStr(gameVariantName))
	fmt.Printf("  playlist_id          = %s\n", ptrStr(playlistAssetID))
	fmt.Printf("  team_0_score         = %s\n", ptrI16(team0Score))
	fmt.Printf("  team_1_score         = %s\n", ptrI16(team1Score))
	fmt.Printf("  pair_name_fr         = %s\n", ptrStr(pairNameFR))
	fmt.Printf("  pair_id              = %s\n", ptrStr(pairAssetID))
	fmt.Printf("  game_variant_id      = %s\n", ptrStr(gameVariantAssetID))
	fmt.Printf("  [Q13 FULL SCAN] OK\n")

	// Count dans match_participants
	var count int
	if e := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, matchID).Scan(&count); e != nil {
		fmt.Printf("  [match_participants] ERREUR: %v\n", e)
	} else {
		fmt.Printf("  [match_participants] %d participant(s)\n", count)
	}
}
