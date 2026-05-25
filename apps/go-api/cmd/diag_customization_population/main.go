//go:build cgo

// diag_customization_population — audit uniforme de la population
// banner/emblem/backdrop/spartan_id dans career_progression pour TOUS les
// joueurs.
//
// Pour chaque player DB sous data/titles/{titleSlug}/players/*/stats.duckdb :
//   - total de rows dans career_progression
//   - rows avec banner_image_url non vide
//   - rows avec emblem_image_url non vide
//   - rows avec backdrop_image_url non vide
//   - rows avec spartan_id non vide
//   - dernier recorded_at global + dernier recorded_at avec banner non vide
//
// Read-only, aucun INSERT.
//
// Usage : go run -tags cgo ./cmd/diag_customization_population [-title halo_infinite]
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

type stats struct {
	gamertag           string
	total              int
	withBanner         int
	withEmblem         int
	withBackdrop       int
	withSpartanID      int
	lastRecordedAt     sql.NullString
	lastBannerAt       sql.NullString
	lastBannerValue    sql.NullString
	lastEmblemValue    sql.NullString
	lastSpartanIDValue sql.NullString
}

func main() {
	titleSlug := flag.String("title", "halo_infinite", "title slug")
	flag.Parse()

	playersRoot := filepath.Join("data", "titles", *titleSlug, "players")
	entries, err := os.ReadDir(playersRoot)
	if err != nil {
		log.Fatalf("ReadDir %s: %v", playersRoot, err)
	}

	rows := make([]stats, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbPath := filepath.Join(playersRoot, e.Name(), "stats.duckdb")
		if _, err := os.Stat(dbPath); err != nil {
			continue
		}
		s, err := inspect(dbPath, e.Name())
		if err != nil {
			fmt.Printf("ERR %s: %v\n", e.Name(), err)
			continue
		}
		rows = append(rows, s)
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].gamertag < rows[j].gamertag })

	fmt.Printf("\n%-22s %5s %8s %8s %10s %10s %s\n",
		"GAMERTAG", "TOTAL", "BANNER", "EMBLEM", "BACKDROP", "SPARTAN_ID", "DERNIÈRE BANNIÈRE")
	fmt.Println(stringRepeat("-", 110))
	for _, r := range rows {
		fmt.Printf("%-22s %5d %8d %8d %10d %10d  %s\n",
			r.gamertag, r.total,
			r.withBanner, r.withEmblem, r.withBackdrop, r.withSpartanID,
			nullStr(r.lastBannerAt))
	}

	fmt.Println()
	for _, r := range rows {
		fmt.Printf("[%s] dernier snapshot : %s\n", r.gamertag, nullStr(r.lastRecordedAt))
		if r.lastBannerValue.Valid {
			fmt.Printf("  banner: %s\n", truncate(r.lastBannerValue.String, 100))
		}
		if r.lastEmblemValue.Valid {
			fmt.Printf("  emblem: %s\n", truncate(r.lastEmblemValue.String, 100))
		}
		if r.lastSpartanIDValue.Valid {
			fmt.Printf("  spartan_id: %s\n", r.lastSpartanIDValue.String)
		}
	}
}

func inspect(dbPath, gamertag string) (stats, error) {
	s := stats{gamertag: gamertag}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=read_only")
	if err != nil {
		return s, fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	q := `
SELECT
    COUNT(*)                                                              AS total,
    COUNT(NULLIF(TRIM(banner_image_url),   '')) AS with_banner,
    COUNT(NULLIF(TRIM(emblem_image_url),   '')) AS with_emblem,
    COUNT(NULLIF(TRIM(backdrop_image_url), '')) AS with_backdrop,
    COUNT(NULLIF(TRIM(spartan_id),         '')) AS with_spartan_id,
    MAX(recorded_at)                                                      AS last_recorded_at,
    MAX(CASE WHEN NULLIF(TRIM(banner_image_url), '') IS NOT NULL THEN recorded_at END) AS last_banner_at
FROM career_progression`
	err = db.QueryRowContext(ctx, q).Scan(
		&s.total, &s.withBanner, &s.withEmblem, &s.withBackdrop, &s.withSpartanID,
		&s.lastRecordedAt, &s.lastBannerAt,
	)
	if err != nil {
		return s, fmt.Errorf("scan agg: %w", err)
	}

	// Last non-empty values per field via ARG_MAX FILTER (même pattern que prod)
	q2 := `
SELECT
    ARG_MAX(banner_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url), '') IS NOT NULL) AS banner,
    ARG_MAX(emblem_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url), '') IS NOT NULL) AS emblem,
    ARG_MAX(spartan_id,       recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),       '') IS NOT NULL) AS spartan_id
FROM career_progression`
	_ = db.QueryRowContext(ctx, q2).Scan(&s.lastBannerValue, &s.lastEmblemValue, &s.lastSpartanIDValue)
	return s, nil
}

func nullStr(s sql.NullString) string {
	if !s.Valid {
		return "—"
	}
	return s.String
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func stringRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
