//go:build ignore

// Exporte les events highlight + métadonnées des 8 matchs de référence T0 en
// fixtures JSON statiques pour le golden test de la couche MatchTimeline.
//
// Usage: go run ./cmd/export_t0_fixtures/main.go <shared_db_path> <out.json>
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

var matchIDs = []string{
	"41b61fb9-3d71-40b7-bde7-45682fba6d57",
	"ac97753c-e3b3-4132-9fb3-6032ee95a202",
	"c1e8d359-5f92-4b3d-8ffd-d0dc2b954af5",
	"5d4295b8-b2c4-4b19-b239-31a74b82dd83",
	"e157a672-5ea8-4473-9bd5-f8a0c44619f0",
	"003498a3-a3c2-4a37-bf49-67e642a353b2",
	"30bdcae7-e648-4951-9149-a65a61623af3",
	"5da6fd30-f29e-4713-93a6-73d69d87626e",
}

type eventRow struct {
	MatchID   string `json:"match_id"`
	EventType string `json:"event_type"`
	TimeMS    int64  `json:"time_ms"`
	XUID      string `json:"xuid"`
}

type matchMeta struct {
	MatchID         string `json:"match_id"`
	DurationSeconds *int   `json:"duration_seconds"`
	TopKillerXUID   string `json:"top_killer_xuid"`
	// T0Ms : offset countdown pré-match en ms (Match Timeline T0, Phase 3),
	// dérivé de real_start_time. Nil si real_start_time absent.
	T0Ms *int64 `json:"t0_ms"`
}

type fixture struct {
	Doc     string      `json:"_doc"`
	Matches []matchMeta `json:"matches"`
	Events  []eventRow  `json:"events"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: export_t0_fixtures <shared_db_path> <out.json>")
		os.Exit(1)
	}
	dbPath, outPath := os.Args[1], os.Args[2]

	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	fx := fixture{
		Doc: "Fixtures golden T0 — events + metadata des 8 matchs de reference. Genere par cmd/export_t0_fixtures. Ne pas editer a la main.",
	}

	// Events: meme SELECT que HighlightEventsRepo (match_id, event_type, time_ms, xuid).
	for _, id := range matchIDs {
		rows, err := db.Query(`
			SELECT he.match_id, he.event_type, COALESCE(he.time_ms, 0) AS time_ms, COALESCE(he.xuid, '') AS xuid
			FROM highlight_events he
			WHERE he.match_id = ?
			  AND he.event_type IN ('kill','death','first_kill','first_death')
			ORDER BY he.time_ms ASC, he.event_type ASC, he.xuid ASC`, id)
		if err != nil {
			fmt.Println("events query:", err)
			os.Exit(1)
		}
		for rows.Next() {
			var e eventRow
			if err := rows.Scan(&e.MatchID, &e.EventType, &e.TimeMS, &e.XUID); err != nil {
				fmt.Println("scan event:", err)
				os.Exit(1)
			}
			fx.Events = append(fx.Events, e)
		}
		rows.Close()

		// Metadata: durée + top killer (xuid le plus actif sur kills) pour fixer
		// un joueur déterministe dans ComputeFirstEventsPerMatch.
		var dur sql.NullInt64
		_ = db.QueryRow(`SELECT duration_seconds FROM match_registry WHERE match_id = ?`, id).Scan(&dur)
		// T0 offset (countdown pré-match) en ms : même formule que la prod
		// (player_matches_repo / Q13). NULL si real_start_time absent.
		var t0 sql.NullInt64
		_ = db.QueryRow(`
			SELECT CASE WHEN real_start_time IS NOT NULL THEN
				epoch_ms(real_start_time AT TIME ZONE 'UTC')
				- epoch_ms(`+analysis.SQLStartTimeCanonical("")+`)
			END
			FROM match_registry WHERE match_id = ?`, id).Scan(&t0)
		var topKiller sql.NullString
		_ = db.QueryRow(`
			SELECT he.xuid
			FROM highlight_events he
			WHERE he.match_id = ? AND he.event_type IN ('kill','first_kill') AND he.xuid IS NOT NULL
			GROUP BY he.xuid
			ORDER BY COUNT(*) DESC, he.xuid ASC
			LIMIT 1`, id).Scan(&topKiller)

		meta := matchMeta{MatchID: id, TopKillerXUID: topKiller.String}
		if dur.Valid {
			d := int(dur.Int64)
			meta.DurationSeconds = &d
		}
		if t0.Valid {
			v := t0.Int64
			meta.T0Ms = &v
		}
		fx.Matches = append(fx.Matches, meta)
	}

	data, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		fmt.Println("marshal:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Println("write:", err)
		os.Exit(1)
	}
	fmt.Printf("exported %d events across %d matches → %s\n", len(fx.Events), len(fx.Matches), outPath)
}
