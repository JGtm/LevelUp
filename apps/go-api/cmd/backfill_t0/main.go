// Command backfill_t0 calcule et persiste le T0 (countdown pré-match) de chaque
// match dans match_registry (real_start_time = début gameplay UTC + t0_quality).
//
// T0 = MIN(first_joined_time des joueurs present_at_beginning, hors bots) −
// start_time_utc, avec filet multi-joueurs (cf. analysis/timeline.ComputeT0).
//
// Usage :
//
//	go run ./cmd/backfill_t0 --db <shared.duckdb>            # dry-run (lecture seule)
//	go run ./cmd/backfill_t0 --db <shared.duckdb> --commit   # écrit en DB
//
// Sûr : en --commit, les UPDATE sont séquentiels single-connection (aucune
// pression concurrente → pas de risque ART). Les rejets (negative,
// suspicious_high, no_data) n'écrivent que t0_quality, real_start_time reste NULL.
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := flag.String("db", "", "chemin shared_matches_v2.duckdb")
	commit := flag.Bool("commit", false, "écrit en DB (défaut: dry-run lecture seule)")
	flag.Parse()

	if *dbPath == "" {
		fmt.Println("usage: backfill_t0 --db <shared.duckdb> [--commit]")
		os.Exit(2)
	}

	mode := "read_only"
	if *commit {
		mode = "read_write"
	}
	db, err := sql.Open("duckdb", *dbPath+"?access_mode="+mode)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	if *commit {
		// Migration idempotente : la colonne t0_quality doit exister avant l'UPDATE.
		if _, err := db.Exec(`ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS t0_quality VARCHAR`); err != nil {
			fmt.Println("migration t0_quality:", err)
			os.Exit(1)
		}
	}

	starts, err := loadMatchStarts(db)
	if err != nil {
		fmt.Println("load matches:", err)
		os.Exit(1)
	}
	partsByMatch, err := loadParticipations(db)
	if err != nil {
		fmt.Println("load participations:", err)
		os.Exit(1)
	}

	type result struct {
		matchID string
		startMS int64
		t0      int64
		quality timeline.T0Quality
	}
	results := make([]result, 0, len(starts))
	dist := map[timeline.T0Quality]int{}
	var computedT0s []int64

	for matchID, start := range starts {
		t0, q := timeline.ComputeT0(partsByMatch[matchID], start)
		dist[q]++
		results = append(results, result{matchID, start.UnixMilli(), t0, q})
		if q.Computed() {
			computedT0s = append(computedT0s, t0)
		}
	}

	printDistribution(len(starts), dist, computedT0s)

	if !*commit {
		fmt.Println("\n[DRY-RUN] aucune écriture. Relancer avec --commit pour persister.")
		return
	}

	// --- COMMIT : UPDATE séquentiels dans une transaction ---
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("begin tx:", err)
		os.Exit(1)
	}
	updComputed, err := tx.Prepare(`UPDATE match_registry SET real_start_time = ?, t0_quality = ? WHERE match_id = ?`)
	if err != nil {
		fmt.Println("prepare computed:", err)
		os.Exit(1)
	}
	updRejected, err := tx.Prepare(`UPDATE match_registry SET t0_quality = ? WHERE match_id = ?`)
	if err != nil {
		fmt.Println("prepare rejected:", err)
		os.Exit(1)
	}

	written := 0
	for _, r := range results {
		if r.quality.Computed() {
			gameplayStartUTC := time.UnixMilli(r.startMS + r.t0).UTC()
			if _, err := updComputed.Exec(gameplayStartUTC, string(r.quality), r.matchID); err != nil {
				_ = tx.Rollback()
				fmt.Printf("update computed %s: %v\n", r.matchID, err)
				os.Exit(1)
			}
		} else {
			if _, err := updRejected.Exec(string(r.quality), r.matchID); err != nil {
				_ = tx.Rollback()
				fmt.Printf("update rejected %s: %v\n", r.matchID, err)
				os.Exit(1)
			}
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("commit tx:", err)
		os.Exit(1)
	}
	fmt.Printf("\n[COMMIT] %d lignes mises à jour.\n", written)
}

// loadMatchStarts charge le start UTC canonique de chaque match.
func loadMatchStarts(db *sql.DB) (map[string]time.Time, error) {
	rows, err := db.Query(`
		SELECT match_id, ` + analysis.SQLStartTimeCanonical("") + ` AS start_utc
		FROM match_registry`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]time.Time)
	for rows.Next() {
		var id string
		var t time.Time
		if err := rows.Scan(&id, &t); err != nil {
			return nil, err
		}
		out[id] = t.UTC()
	}
	return out, rows.Err()
}

// loadParticipations charge les inputs T0 par match (joueurs présents au début).
//
// Détection bot : le `gamertag` est NULL dans match_participants pour la quasi-
// totalité des vrais joueurs (résolu via xuid_aliases en aval), donc inutilisable
// ici. Les bots Halo se reconnaissent à un xuid NON numérique (format "bid(N.M)"),
// les vrais joueurs ayant un xuid entier 16 chiffres.
func loadParticipations(db *sql.DB) (map[string][]timeline.ParticipationT0Input, error) {
	rows, err := db.Query(`
		SELECT match_id, first_joined_time, COALESCE(present_at_beginning, false), COALESCE(xuid, '')
		FROM match_participants
		WHERE first_joined_time IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]timeline.ParticipationT0Input)
	for rows.Next() {
		var id, xuid string
		var fjt time.Time
		var present bool
		if err := rows.Scan(&id, &fjt, &present, &xuid); err != nil {
			return nil, err
		}
		out[id] = append(out[id], timeline.ParticipationT0Input{
			FirstJoinedTime:    fjt.UTC(),
			PresentAtBeginning: present,
			IsBot:              isBotXUID(xuid),
		})
	}
	return out, rows.Err()
}

// isBotXUID retourne true si l'xuid n'est pas un identifiant joueur numérique
// (bots Halo : "bid(19.0)"; vrais joueurs : "2533274823110022").
func isBotXUID(xuid string) bool {
	if xuid == "" {
		return true
	}
	for _, c := range xuid {
		if c < '0' || c > '9' {
			return true
		}
	}
	return false
}

func printDistribution(total int, dist map[timeline.T0Quality]int, computedT0s []int64) {
	fmt.Printf("=== Backfill T0 — %d matchs ===\n\n", total)
	order := []timeline.T0Quality{
		timeline.T0QualityOK, timeline.T0QualitySingleSource, timeline.T0QualitySpreadHigh,
		timeline.T0QualityNoData, timeline.T0QualityNegative, timeline.T0QualitySuspiciousHigh,
	}
	computed := 0
	for _, q := range order {
		n := dist[q]
		tag := "rejet "
		if q.Computed() {
			tag = "stocké"
			computed += n
		}
		fmt.Printf("  %-16s %5d  [%s]\n", q, n, tag)
	}
	fmt.Printf("\n  T0 stockés : %d / %d (%.1f%%)\n", computed, total, pct(computed, total))

	if len(computedT0s) > 0 {
		sort.Slice(computedT0s, func(i, j int) bool { return computedT0s[i] < computedT0s[j] })
		fmt.Printf("  T0 min/médiane/max : %ds / %ds / %ds\n",
			computedT0s[0]/1000,
			computedT0s[len(computedT0s)/2]/1000,
			computedT0s[len(computedT0s)-1]/1000)
	}
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
