//go:build cgo

// lusr_v2_tier_analysis — analyse la distribution de μ dans
// player_skill_state_v2_latest pour calibrer les seuils tier Phase 3e.
//
// Sortie : stats par playlist_group (count, percentiles), + valeurs des
// joueurs trackés pour cross-référence avec leurs cibles connues.
//
// Read-only. Usage : go run -tags cgo ./cmd/lusr_v2_tier_analysis
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"

	_ "github.com/duckdb/duckdb-go/v2"
)

const defaultDBPath = "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"

var trackedPlayers = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func main() {
	dbPath := flag.String("db", defaultDBPath, "path to shared_matches_v2.duckdb")
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	db, err := sql.Open("duckdb", *dbPath+"?access_mode=read_only")
	if err != nil {
		slog.Error("open db", "err", err, "path", *dbPath)
		os.Exit(1)
	}
	defer db.Close()

	// Resolve tracked XUIDs.
	xuidByGT := resolveXUIDs(db, trackedPlayers)

	groups := []string{"arena_slayer", "arena_objectif", "btb", "chaos"}

	fmt.Println("# LUSR v2 — Phase 3e : analyse distribution μ")
	fmt.Println()
	fmt.Println("## Distribution par playlist_group")
	fmt.Println()
	fmt.Println("| Groupe | N | μ min | p10 | p25 | p50 | p75 | p90 | p95 | p99 | μ max |")
	fmt.Println("|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|")

	for _, g := range groups {
		mus := loadMus(db, g)
		if len(mus) == 0 {
			fmt.Printf("| %s | 0 | — | — | — | — | — | — | — | — | — |\n", g)
			continue
		}
		sort.Float64s(mus)
		fmt.Printf("| %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f |\n",
			g, len(mus),
			mus[0],
			percentile(mus, 0.10),
			percentile(mus, 0.25),
			percentile(mus, 0.50),
			percentile(mus, 0.75),
			percentile(mus, 0.90),
			percentile(mus, 0.95),
			percentile(mus, 0.99),
			mus[len(mus)-1],
		)
	}

	fmt.Println()
	fmt.Println("## Joueurs trackés (cross-référence avec cibles)")
	fmt.Println()
	fmt.Println("| Joueur | Cible | Groupe | μ | σ | exp |")
	fmt.Println("|---|---|---|---:|---:|---:|")
	targets := map[string]string{
		"Madina97294":     "fin Platine / début Diamant",
		"Chocoboflor":     "milieu/bas Or",
		"JGtm":            "milieu/bas Or",
		"XxDaemonGamerxX": "Bronze (faible)",
	}
	for _, gt := range trackedPlayers {
		xuid := xuidByGT[gt]
		if xuid == "" {
			fmt.Printf("| %s | %s | — | — | — | — |\n", gt, targets[gt])
			continue
		}
		for _, g := range groups {
			mu, sigma, exp := loadPlayerState(db, xuid, g)
			if exp == 0 {
				continue
			}
			fmt.Printf("| %s | %s | %s | %.2f | %.2f | %d |\n", gt, targets[gt], g, mu, sigma, exp)
		}
	}

	fmt.Println()
	fmt.Println("## Proposition seuils Phase 3e")
	fmt.Println()
	fmt.Println("Critères : (a) couvrir la distribution observée par percentiles ; (b) caler les 4 joueurs trackés sur leurs cibles connues.")
}

func resolveXUIDs(db *sql.DB, gamertags []string) map[string]string {
	out := map[string]string{}
	for _, gt := range gamertags {
		var xuid sql.NullString
		_ = db.QueryRow(`SELECT xuid FROM xuid_aliases WHERE lower(gamertag) = lower(?) ORDER BY last_seen DESC NULLS LAST LIMIT 1`, gt).Scan(&xuid)
		if xuid.Valid {
			out[gt] = xuid.String
		}
	}
	return out
}

func loadMus(db *sql.DB, group string) []float64 {
	rows, err := db.Query(`SELECT mu FROM player_skill_state_v2_latest WHERE playlist_group = ?`, group)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var m float64
		if rows.Scan(&m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func loadPlayerState(db *sql.DB, xuid, group string) (mu, sigma float64, exp int) {
	_ = db.QueryRow(`SELECT mu, sigma, experience FROM player_skill_state_v2_latest WHERE xuid = ? AND playlist_group = ?`,
		xuid, group).Scan(&mu, &sigma, &exp)
	return
}

// percentile : interpolation linéaire sur slice triée ascendante.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	low := int(pos)
	high := low + 1
	if high >= len(sorted) {
		return sorted[low]
	}
	frac := pos - float64(low)
	return sorted[low]*(1-frac) + sorted[high]*frac
}
