// Command backfill_time_played recalcule time_played_seconds par (match, joueur)
// depuis les timestamps de participation (first_joined_time / last_leave_time),
// en gérant quitters et latecomers, dans la fenêtre de gameplay [real_start_time,
// start_time_utc + duration].
//
// Cf. analysis/timeline.ComputeTimePlayed + PLAN_MATCH_TIMELINE_T0 §7.
//
// Usage :
//
//	go run ./cmd/backfill_time_played --db <shared.duckdb>            # dry-run (lecture seule)
//	go run ./cmd/backfill_time_played --db <shared.duckdb> --commit   # écrit en DB
//
// Le dry-run affiche : distribution qualité, comparaison recomputed vs valeur
// API existante, et le garde-rail §7.3 (médiane time_played des full-match ≈
// gameplay_duration). N'écrase QUE les lignes de qualité "ok" (first_joined
// présent) ; les lignes no_data conservent leur valeur API.
//
// Sûr : en --commit, les UPDATE sont séquentiels single-connection (aucune
// pression concurrente → pas de risque ART).
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"

	_ "github.com/duckdb/duckdb-go/v2"
)

type partRow struct {
	matchID            string
	xuid               string
	firstJoined        time.Time
	lastLeave          *time.Time
	presentAtBeginning bool
	presentAtComplete  bool
	apiTimePlayed      sql.NullInt64
}

type computedRow struct {
	matchID  string
	xuid     string
	secs     int64
	quality  timeline.TimePlayedQuality
	apiValue sql.NullInt64
	fullGame bool // present_at_beginning && present_at_completion
	winSec   int64
}

func main() {
	dbPath := flag.String("db", "", "chemin shared_matches_v2.duckdb")
	commit := flag.Bool("commit", false, "écrit en DB (défaut: dry-run lecture seule)")
	flag.Parse()

	if *dbPath == "" {
		fmt.Println("usage: backfill_time_played --db <shared.duckdb> [--commit]")
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

	windows, err := loadMatchWindows(db)
	if err != nil {
		fmt.Println("load matches:", err)
		os.Exit(1)
	}
	parts, err := loadParticipants(db)
	if err != nil {
		fmt.Println("load participants:", err)
		os.Exit(1)
	}

	results := make([]computedRow, 0, len(parts))
	dist := map[timeline.TimePlayedQuality]int{}
	for _, p := range parts {
		tl, ok := windows[p.matchID]
		if !ok {
			continue
		}
		secs, q := timeline.ComputeTimePlayedFor(
			timeline.TimePlayedInput{FirstJoinedTime: p.firstJoined, LastLeaveTime: p.lastLeave},
			tl,
		)
		dist[q]++
		results = append(results, computedRow{
			matchID: p.matchID, xuid: p.xuid, secs: secs, quality: q,
			apiValue: p.apiTimePlayed,
			fullGame: p.presentAtBeginning && p.presentAtComplete,
			winSec:   tl.GameplayDurationSeconds(),
		})
	}

	printReport(len(parts), dist, results)

	if !*commit {
		fmt.Println("\n[DRY-RUN] aucune écriture. Relancer avec --commit pour persister.")
		return
	}

	written := commitUpdates(db, results)
	fmt.Printf("\n[COMMIT] %d lignes time_played_seconds mises à jour (qualité ok).\n", written)
}

// loadMatchWindows charge l'horloge de gameplay de chaque match sous forme de
// domain.MatchTimeline (StartUTC = start_utc, T0 = real_start_time − start_utc).
// GameplayStartUTC/EndUTC/DurationSeconds en dérivent (source unique).
func loadMatchWindows(db *sql.DB) (map[string]domain.MatchTimeline, error) {
	rows, err := db.Query(`
		SELECT
			match_id,
			` + analysis.SQLStartTimeCanonical("") + `               AS start_utc,
			real_start_time,
			COALESCE(duration_seconds, 0)                                         AS dur
		FROM match_registry`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]domain.MatchTimeline)
	for rows.Next() {
		var id string
		var startUTC time.Time
		var realStart sql.NullTime
		var dur int64
		if err := rows.Scan(&id, &startUTC, &realStart, &dur); err != nil {
			return nil, err
		}
		startUTC = startUTC.UTC()
		var t0ms int64
		if realStart.Valid {
			t0ms = realStart.Time.UTC().Sub(startUTC).Milliseconds()
		}
		out[id] = domain.NewMatchTimelineAt(startUTC, dur*1000, t0ms)
	}
	return out, rows.Err()
}

// loadParticipants charge la participation de chaque (match, joueur) ayant un
// first_joined_time. Les lignes sans first_joined sont exclues (NoData → on
// conserve leur valeur API).
func loadParticipants(db *sql.DB) ([]partRow, error) {
	rows, err := db.Query(`
		SELECT
			match_id, COALESCE(xuid, ''), first_joined_time, last_leave_time,
			COALESCE(present_at_beginning, false), COALESCE(present_at_completion, false),
			time_played_seconds
		FROM match_participants
		WHERE first_joined_time IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []partRow
	for rows.Next() {
		var p partRow
		var fjt time.Time
		var llt sql.NullTime
		if err := rows.Scan(&p.matchID, &p.xuid, &fjt, &llt,
			&p.presentAtBeginning, &p.presentAtComplete, &p.apiTimePlayed); err != nil {
			return nil, err
		}
		p.firstJoined = fjt.UTC()
		if llt.Valid {
			t := llt.Time.UTC()
			p.lastLeave = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// commitUpdates applique les UPDATE séquentiels (qualité ok uniquement) dans
// une transaction. Les lignes no_data / clamped_zero ne sont pas écrites.
func commitUpdates(db *sql.DB, results []computedRow) int {
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("begin tx:", err)
		os.Exit(1)
	}
	upd, err := tx.Prepare(`UPDATE match_participants SET time_played_seconds = ? WHERE match_id = ? AND xuid = ?`)
	if err != nil {
		fmt.Println("prepare:", err)
		os.Exit(1)
	}
	written := 0
	for _, r := range results {
		if r.quality != timeline.TimePlayedOK {
			continue
		}
		if _, err := upd.Exec(r.secs, r.matchID, r.xuid); err != nil {
			_ = tx.Rollback()
			fmt.Printf("update %s/%s: %v\n", r.matchID, r.xuid, err)
			os.Exit(1)
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		fmt.Println("commit tx:", err)
		os.Exit(1)
	}
	return written
}

// printReport affiche distribution qualité + comparaison vs API + garde-rail.
func printReport(total int, dist map[timeline.TimePlayedQuality]int, results []computedRow) {
	fmt.Printf("=== Backfill time_played — %d lignes (match,joueur) avec first_joined ===\n\n", total)
	order := []timeline.TimePlayedQuality{
		timeline.TimePlayedOK, timeline.TimePlayedClampedZero, timeline.TimePlayedNoData,
	}
	for _, q := range order {
		tag := "skip  "
		if q == timeline.TimePlayedOK {
			tag = "écrit "
		}
		fmt.Printf("  %-14s %6d  [%s]\n", q, dist[q], tag)
	}

	printAPIComparison(results)
	printGuardrail(results)
}

// printAPIComparison compare le time_played recomputed à la valeur API existante
// sur les lignes ok (où les deux existent). Valide que le recompute n'introduit
// pas de dérive massive avant d'écraser la donnée.
func printAPIComparison(results []computedRow) {
	var diffs []float64
	within5, within10, n := 0, 0, 0
	for _, r := range results {
		if r.quality != timeline.TimePlayedOK || !r.apiValue.Valid || r.apiValue.Int64 <= 0 {
			continue
		}
		n++
		d := math.Abs(float64(r.secs - r.apiValue.Int64))
		diffs = append(diffs, d)
		rel := d / float64(r.apiValue.Int64)
		if rel <= 0.05 {
			within5++
		}
		if rel <= 0.10 {
			within10++
		}
	}
	fmt.Printf("\n--- Comparaison recomputed vs API (n=%d lignes ok avec valeur API) ---\n", n)
	if n == 0 {
		fmt.Println("  (aucune valeur API à comparer)")
		return
	}
	sort.Float64s(diffs)
	fmt.Printf("  |diff| médiane=%.0fs  p90=%.0fs  max=%.0fs\n",
		diffs[len(diffs)/2], diffs[int(float64(len(diffs))*0.9)], diffs[len(diffs)-1])
	fmt.Printf("  à ±5%%  : %d (%.1f%%)   à ±10%% : %d (%.1f%%)\n",
		within5, pct(within5, n), within10, pct(within10, n))
}

// printGuardrail vérifie §7.3 : la médiane des time_played des joueurs full-match
// (présents début+fin) doit ≈ gameplay_duration de leur match.
func printGuardrail(results []computedRow) {
	var ratios []float64
	pass5, n := 0, 0
	for _, r := range results {
		if r.quality != timeline.TimePlayedOK || !r.fullGame || r.winSec <= 0 {
			continue
		}
		n++
		ratio := float64(r.secs) / float64(r.winSec)
		ratios = append(ratios, ratio)
		if math.Abs(ratio-1.0) <= 0.05 {
			pass5++
		}
	}
	fmt.Printf("\n--- Garde-rail §7.3 (full-match: time_played ≈ gameplay_duration, n=%d) ---\n", n)
	if n == 0 {
		fmt.Println("  (aucun joueur full-match exploitable)")
		return
	}
	sort.Float64s(ratios)
	fmt.Printf("  ratio médian=%.3f  p10=%.3f  p90=%.3f\n",
		ratios[len(ratios)/2], ratios[int(float64(len(ratios))*0.1)], ratios[int(float64(len(ratios))*0.9)])
	fmt.Printf("  full-match à ±5%% de gameplay_duration : %d (%.1f%%)  [cible >95%%]\n", pass5, pct(pass5, n))
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
