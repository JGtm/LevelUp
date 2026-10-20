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
//
// ⚠ CE T0 EST ESTIMÉ, ET IL NE PRIME PLUS. Depuis le 2026-09-02 le coup d'envoi peut
// être MESURÉ dans le film (`cmd/backfill_t0_film`, t0_quality = `film_movement`), et la
// mesure bat l'estimation : écart-type 9 752 ms contre 12 764 ms sur les 49 matchs au
// T0-API sain, aucune valeur invraisemblable côté film, là où l'API rend ~0 ms sur 10-15 %
// des matchs. Les lignes déjà marquées `film_movement` sont donc ÉCARTÉES du commit —
// sans quoi une passe de rattrapage de ce binaire ANNULERAIT silencieusement la réparation
// (décision D2 du plan T0-film).
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

	// LA GARDE : un T0 MESURÉ dans le film ne se laisse pas écraser par une estimation.
	films, err := loadFilmT0Matches(db)
	if err != nil {
		fmt.Println("load t0_quality film_movement:", err)
		os.Exit(1)
	}
	results, proteges := ecarterLesFilms(results, films)
	fmt.Printf("  protégés par le film (t0_quality=%s, non réécrits) : %d\n",
		timeline.T0QualityFilmMovement, proteges)

	if !*commit {
		fmt.Println("\n[DRY-RUN] aucune écriture. Relancer avec --commit pour persister.")
		return
	}
	written, err := commitResults(db, results)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("\n[COMMIT] %d lignes mises à jour.\n", written)
}

// result : le T0 estimé d'un match et sa qualité, prêts à écrire.
type result struct {
	matchID string
	startMS int64
	t0      int64
	quality timeline.T0Quality
}

// commitResults écrit les résultats : UPDATE séquentiels dans une transaction.
func commitResults(db *sql.DB, results []result) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	updComputed, err := tx.Prepare(`UPDATE match_registry SET real_start_time = ?, t0_quality = ? WHERE match_id = ?`)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare computed: %w", err)
	}
	updRejected, err := tx.Prepare(`UPDATE match_registry SET t0_quality = ? WHERE match_id = ?`)
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare rejected: %w", err)
	}
	written := 0
	for _, r := range results {
		var execErr error
		if r.quality.Computed() {
			gameplayStartUTC := time.UnixMilli(r.startMS + r.t0).UTC()
			_, execErr = updComputed.Exec(gameplayStartUTC, string(r.quality), r.matchID)
		} else {
			_, execErr = updRejected.Exec(string(r.quality), r.matchID)
		}
		if execErr != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("update %s: %w", r.matchID, execErr)
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return written, nil
}

// ecarterLesFilms retire les matchs dont le T0 vient DÉJÀ du film. Rend la liste gardée et
// le nombre d'écartés.
//
// L'ORDRE DES MATCHS RESTANTS EST PRÉSERVÉ : deux passages écrivent dans le même ordre.
func ecarterLesFilms(results []result, films map[string]bool) ([]result, int) {
	if len(films) == 0 {
		return results, 0
	}
	gardes := make([]result, 0, len(results))
	for _, r := range results {
		if films[r.matchID] {
			continue
		}
		gardes = append(gardes, r)
	}
	return gardes, len(results) - len(gardes)
}

// loadFilmT0Matches rend les matchs dont `t0_quality` vaut déjà `film_movement`.
//
// LA COLONNE PEUT NE PAS EXISTER sur une base antérieure à la migration
// `shared_add_t0_quality` : aucune ligne ne peut alors être marquée, et l'ensemble VIDE est
// la réponse exacte — pas une dégradation. Toute autre erreur remonte : une garde qui
// s'éteint en silence ne garde rien.
func loadFilmT0Matches(db *sql.DB) (map[string]bool, error) {
	var cols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'match_registry' AND column_name = 't0_quality'`).Scan(&cols); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	if cols == 0 {
		return out, nil
	}
	rows, err := db.Query(`SELECT match_id FROM match_registry WHERE t0_quality = ?`,
		string(timeline.T0QualityFilmMovement))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
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
